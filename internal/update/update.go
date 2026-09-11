// Package update checks GitHub Releases for a newer Kite build and can
// swap the running binary for the downloaded one. Kite ships as a portable
// single-file executable (no installer), so "update" means: download the
// new exe, rename the current one aside, move the new one into place, and
// relaunch -- the same rename+relaunch pattern Go self-updaters use, which
// works without admin rights on both Windows and Linux.
package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const releaseAPI = "https://api.github.com/repos/freeb5d/kite/releases/latest"

// Info describes the result of a Check. downloadURL is deliberately
// unexported: it's only ever needed internally by Apply, and keeping it
// out of the struct's JSON encoding means it never crosses the Wails
// bridge to the frontend.
type Info struct {
	Available bool   `json:"available"`
	Current   string `json:"current"`
	Latest    string `json:"latest"`
	Notes     string `json:"notes"`

	downloadURL string
}

type ghRelease struct {
	TagName string `json:"tag_name"`
	Body    string `json:"body"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// Check compares currentVersion (e.g. "v0.3.0", or "dev" for a local
// build) against the latest GitHub release tag for this platform.
func Check(currentVersion string) (Info, error) {
	req, err := http.NewRequest(http.MethodGet, releaseAPI, nil)
	if err != nil {
		return Info{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return Info{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Info{}, fmt.Errorf("github returned %d", resp.StatusCode)
	}

	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return Info{}, err
	}

	assetName := assetNameForPlatform()
	var downloadURL string
	for _, a := range rel.Assets {
		if a.Name == assetName {
			downloadURL = a.BrowserDownloadURL
			break
		}
	}

	current := strings.TrimPrefix(currentVersion, "v")
	latest := strings.TrimPrefix(rel.TagName, "v")

	return Info{
		Available:   downloadURL != "" && isNewer(latest, current),
		Current:     current,
		Latest:      latest,
		Notes:       rel.Body,
		downloadURL: downloadURL,
	}, nil
}

func assetNameForPlatform() string {
	switch runtime.GOOS {
	case "windows":
		return "kite-windows-amd64.exe"
	case "linux":
		return "kite-linux-amd64"
	case "darwin":
		return "kite-macos-arm64"
	default:
		return ""
	}
}

// isNewer does a plain numeric x.y.z comparison. A non-numeric current
// version (e.g. "dev", "dev-abc1234" from an unreleased build) is always
// treated as older, so it always offers the latest real release.
func isNewer(latest, current string) bool {
	lp, latestOK := semverParts(latest)
	cp, currentOK := semverParts(current)
	if !latestOK {
		return false
	}
	if !currentOK {
		return true
	}
	for i := 0; i < 3; i++ {
		if lp[i] != cp[i] {
			return lp[i] > cp[i]
		}
	}
	return false
}

func semverParts(v string) ([3]int, bool) {
	var out [3]int
	segs := strings.SplitN(v, ".", 3)
	if len(segs) == 0 {
		return out, false
	}
	for i := 0; i < len(segs) && i < 3; i++ {
		n, err := strconv.Atoi(strings.TrimSpace(segs[i]))
		if err != nil {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

// Progress reports download progress: Total is 0 if the server didn't send
// a Content-Length (progress is then indeterminate; report a spinner).
type Progress struct {
	Downloaded int64
	Total      int64
}

// Apply downloads the platform asset referenced by a prior Check() result
// and swaps it in for the running executable, then relaunches it. Only
// call this after Check() reported Available -- it returns an error on any
// failure *before* relaunching; once the new process is successfully
// started, it does not return at all (the caller is expected to exit).
// onProgress may be nil; when set, it's called periodically during the
// download (never concurrently).
func Apply(info Info, onProgress func(Progress)) error {
	if info.downloadURL == "" {
		return fmt.Errorf("no downloadable build for this platform")
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate running executable: %w", err)
	}
	if resolved, err := filepath.EvalSymlinks(exePath); err == nil {
		exePath = resolved
	}

	newPath := exePath + ".new"
	if err := download(info.downloadURL, newPath, onProgress); err != nil {
		return fmt.Errorf("download update: %w", err)
	}
	if err := os.Chmod(newPath, 0o755); err != nil {
		_ = os.Remove(newPath)
		return fmt.Errorf("set executable permission: %w", err)
	}

	oldPath := exePath + ".old"
	_ = os.Remove(oldPath)
	if err := os.Rename(exePath, oldPath); err != nil {
		_ = os.Remove(newPath)
		return fmt.Errorf("move current executable aside: %w", err)
	}
	if err := os.Rename(newPath, exePath); err != nil {
		_ = os.Rename(oldPath, exePath) // best-effort rollback
		return fmt.Errorf("install new executable: %w", err)
	}

	cmd := exec.Command(exePath)
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	if err := cmd.Start(); err != nil {
		// Roll back so the user isn't left with nothing runnable.
		_ = os.Rename(exePath, exePath+".broken")
		_ = os.Rename(oldPath, exePath)
		return fmt.Errorf("launch updated executable: %w", err)
	}

	_ = os.Remove(oldPath)
	return nil
}

func download(url, dest string, onProgress func(Progress)) error {
	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	var reader io.Reader = resp.Body
	if onProgress != nil {
		reader = &progressReader{r: resp.Body, total: resp.ContentLength, onProgress: onProgress}
	}

	_, err = io.Copy(out, reader)
	return err
}

// progressReader calls onProgress after each underlying Read, throttled so
// a fast local network doesn't flood the frontend with events.
type progressReader struct {
	r          io.Reader
	total      int64
	downloaded int64
	onProgress func(Progress)
	lastReport time.Time
}

func (p *progressReader) Read(buf []byte) (int, error) {
	n, err := p.r.Read(buf)
	p.downloaded += int64(n)

	if time.Since(p.lastReport) > 100*time.Millisecond || err != nil {
		p.lastReport = time.Now()
		p.onProgress(Progress{Downloaded: p.downloaded, Total: p.total})
	}
	return n, err
}
