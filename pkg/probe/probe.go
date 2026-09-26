// Package probe measures server latency three ways, shared by the desktop
// and Android apps:
//
//   - TCP: time to open a TCP connection to the server. Fast, but only
//     proves the port is reachable.
//   - HTTP: time until the server answers a plain HTTP request on its port.
//   - Real: start a temporary xray-core instance using the server and time
//     a real request through it. Slowest, but the only mode that proves the
//     server actually works (right keys, transport, encryption...).
package probe

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/infra/conf/serial"
	_ "github.com/xtls/xray-core/main/distro/all"

	"github.com/freeb5d/kite/pkg/profile"
	"github.com/freeb5d/kite/pkg/xrayconf"
)

const (
	ModeTCP  = "tcp"
	ModeHTTP = "http"
	ModeReal = "real"

	// TestURL returns an empty 204 quickly from everywhere; it's what most
	// V2Ray clients use for "real delay".
	TestURL = "https://www.gstatic.com/generate_204"
	timeout = 8 * time.Second
)

// Ping returns the delay to server in milliseconds using mode.
func Ping(server profile.Server, mode string) (int, error) {
	switch mode {
	case ModeHTTP:
		return httpPing(server)
	case ModeReal:
		return realDelay(server)
	default:
		return tcpPing(server)
	}
}

func address(s profile.Server) string {
	return net.JoinHostPort(s.Address, strconv.Itoa(s.Port))
}

func tcpPing(s profile.Server) (int, error) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", address(s), timeout)
	if err != nil {
		return 0, err
	}
	conn.Close()
	return ms(start), nil
}

// httpPing sends a minimal HTTP request straight to the server's port and
// times the first byte of any reply. Proxy servers usually answer with an
// error page or close the connection; either way they answered. Servers
// that only speak TLS answer the plain request with a TLS alert, which also
// counts.
func httpPing(s profile.Server) (int, error) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", address(s), timeout)
	if err != nil {
		return 0, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))
	host := firstNonEmpty(s.Extra["host"], s.Extra["sni"], s.Address)
	if _, err := fmt.Fprintf(conn, "HEAD / HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", host); err != nil {
		return 0, err
	}
	if _, err := bufio.NewReader(conn).ReadByte(); err != nil {
		return 0, fmt.Errorf("no HTTP response: %w", err)
	}
	return ms(start), nil
}

// realDelay starts a throwaway xray-core instance with only a local SOCKS
// inbound and the server as outbound, then times a request through it.
func realDelay(s profile.Server) (int, error) {
	port, err := freePort()
	if err != nil {
		return 0, err
	}
	cfg, err := xrayconf.Build(s, xrayconf.Options{SOCKSPort: port, LogLevel: "none"})
	if err != nil {
		return 0, err
	}
	config, err := serial.LoadJSONConfig(bytes.NewReader(cfg))
	if err != nil {
		return 0, err
	}
	instance, err := core.New(config)
	if err != nil {
		return 0, err
	}
	if err := instance.Start(); err != nil {
		_ = instance.Close()
		return 0, err
	}
	defer instance.Close()

	proxyURL, _ := url.Parse("socks5://127.0.0.1:" + strconv.Itoa(port))
	client := &http.Client{
		Timeout:   timeout,
		Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL), DisableKeepAlives: true},
	}
	start := time.Now()
	resp, err := client.Get(TestURL)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	if resp.StatusCode >= 500 {
		return 0, errors.New(resp.Status)
	}
	return ms(start), nil
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func ms(start time.Time) int {
	d := int(time.Since(start).Milliseconds())
	if d < 1 {
		d = 1
	}
	return d
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
