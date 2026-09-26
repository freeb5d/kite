package profile

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strconv"
)

// internalKeys are bookkeeping fields Kite adds to Extra; they're never part
// of a share link.
var internalKeys = map[string]bool{"subGroup": true, "subGroupName": true, "subURL": true, "subUsage": true, "subNotes": true, "subUpdatedAt": true, "subUpdateHours": true}

// ShareLink turns a saved server back into a standard share link, so it can
// be copied into another client (or another Kite) and parsed by ParseLink.
func ShareLink(s Server) (string, error) {
	hostport := net.JoinHostPort(s.Address, strconv.Itoa(s.Port))
	switch s.Protocol {
	case "vless", "trojan":
		user := s.UUID
		if s.Protocol == "trojan" {
			user = s.Password
		}
		u := url.URL{Scheme: s.Protocol, User: url.User(user), Host: hostport, RawQuery: extraQuery(s.Extra), Fragment: s.Name}
		return u.String(), nil
	case "shadowsocks":
		creds := base64.RawURLEncoding.EncodeToString([]byte(s.Method + ":" + s.Password))
		return "ss://" + creds + "@" + hostport + "#" + url.PathEscape(s.Name), nil
	case "vmess":
		e := s.Extra
		payload := map[string]string{
			"v": "2", "ps": s.Name, "add": s.Address, "port": strconv.Itoa(s.Port), "id": s.UUID, "aid": "0",
			"net":  firstNonEmpty(e["network"], e["type"], "tcp"),
			"type": firstNonEmpty(e["headerType"], "none"),
			"tls":  firstNonEmpty(e["tls"], e["security"]),
			"host": e["host"], "path": e["path"], "sni": e["sni"], "alpn": e["alpn"], "fp": e["fp"], "scy": e["scy"],
		}
		b, err := json.Marshal(payload)
		if err != nil {
			return "", err
		}
		return "vmess://" + base64.StdEncoding.EncodeToString(b), nil
	}
	return "", fmt.Errorf("can't build a share link for protocol %q", s.Protocol)
}

func extraQuery(extra map[string]string) string {
	keys := make([]string, 0, len(extra))
	for k, v := range extra {
		if !internalKeys[k] && v != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	q := url.Values{}
	for _, k := range keys {
		q.Set(k, extra[k])
	}
	return q.Encode()
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
