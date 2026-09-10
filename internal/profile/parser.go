// Package profile parses vmess://, vless://, trojan://, ss:// share links
// into Server profiles, and persists the saved server list as JSON.
package profile

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// Server is one saved proxy profile, storage- and UI-agnostic.
type Server struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Protocol string `json:"protocol"` // vmess, vless, trojan, shadowsocks
	Address  string `json:"address"`
	Port     int    `json:"port"`
	UUID     string `json:"uuid,omitempty"`     // vmess/vless
	Password string `json:"password,omitempty"` // trojan/ss
	Method   string `json:"method,omitempty"`   // ss cipher
	Extra    map[string]string `json:"extra,omitempty"` // network, tls, sni, path, etc.
}

// ParseLink dispatches to the right parser based on the URI scheme.
func ParseLink(link string) (Server, error) {
	link = strings.TrimSpace(link)
	switch {
	case strings.HasPrefix(link, "vmess://"):
		return parseVMess(link)
	case strings.HasPrefix(link, "vless://"):
		return parseVLESS(link)
	case strings.HasPrefix(link, "trojan://"):
		return parseTrojan(link)
	case strings.HasPrefix(link, "ss://"):
		return parseShadowsocks(link)
	default:
		return Server{}, fmt.Errorf("unsupported or unrecognized link format")
	}
}

// vmess:// carries a base64-encoded JSON payload, not a standard URI.
func parseVMess(link string) (Server, error) {
	raw := strings.TrimPrefix(link, "vmess://")
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		decoded, err = base64.RawStdEncoding.DecodeString(raw)
		if err != nil {
			return Server{}, fmt.Errorf("invalid vmess link: %w", err)
		}
	}

	var payload struct {
		Ps   string `json:"ps"`
		Add  string `json:"add"`
		Port string `json:"port"`
		ID   string `json:"id"`
		Net  string `json:"net"`
		TLS  string `json:"tls"`
		Path string `json:"path"`
		Host string `json:"host"`
		SNI  string `json:"sni"`
	}
	if err := json.Unmarshal(decoded, &payload); err != nil {
		return Server{}, fmt.Errorf("invalid vmess payload: %w", err)
	}

	return Server{
		Name:     payload.Ps,
		Protocol: "vmess",
		Address:  payload.Add,
		Port:     atoiSafe(payload.Port),
		UUID:     payload.ID,
		Extra: map[string]string{
			"network": payload.Net,
			"tls":     payload.TLS,
			"path":    payload.Path,
			"host":    payload.Host,
			"sni":     payload.SNI,
		},
	}, nil
}

// vless:// and trojan:// are standard URIs: scheme://user@host:port?query#name
func parseVLESS(link string) (Server, error) {
	u, err := url.Parse(link)
	if err != nil {
		return Server{}, fmt.Errorf("invalid vless link: %w", err)
	}
	return Server{
		Name:     nameOrDefault(u.Fragment, u.Host),
		Protocol: "vless",
		Address:  u.Hostname(),
		Port:     atoiSafe(u.Port()),
		UUID:     u.User.Username(),
		Extra:    queryToExtra(u.Query()),
	}, nil
}

func parseTrojan(link string) (Server, error) {
	u, err := url.Parse(link)
	if err != nil {
		return Server{}, fmt.Errorf("invalid trojan link: %w", err)
	}
	return Server{
		Name:     nameOrDefault(u.Fragment, u.Host),
		Protocol: "trojan",
		Address:  u.Hostname(),
		Port:     atoiSafe(u.Port()),
		Password: u.User.Username(),
		Extra:    queryToExtra(u.Query()),
	}, nil
}

// ss:// commonly appears as ss://base64(method:password)@host:port#name
func parseShadowsocks(link string) (Server, error) {
	u, err := url.Parse(link)
	if err != nil {
		return Server{}, fmt.Errorf("invalid shadowsocks link: %w", err)
	}

	method, password := u.User.Username(), ""
	if pw, ok := u.User.Password(); ok {
		password = pw
	} else {
		// userinfo is base64(method:password) when no ':' was present in the URL.
		decoded, err := base64.RawURLEncoding.DecodeString(u.User.Username())
		if err != nil {
			decoded, err = base64.StdEncoding.DecodeString(u.User.Username())
		}
		if err == nil {
			parts := strings.SplitN(string(decoded), ":", 2)
			if len(parts) == 2 {
				method, password = parts[0], parts[1]
			}
		}
	}

	return Server{
		Name:     nameOrDefault(u.Fragment, u.Host),
		Protocol: "shadowsocks",
		Address:  u.Hostname(),
		Port:     atoiSafe(u.Port()),
		Method:   method,
		Password: password,
	}, nil
}

func queryToExtra(q url.Values) map[string]string {
	extra := make(map[string]string, len(q))
	for k := range q {
		extra[k] = q.Get(k)
	}
	return extra
}

func nameOrDefault(name, fallback string) string {
	if name == "" {
		return fallback
	}
	return name
}

func atoiSafe(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return n
		}
		n = n*10 + int(c-'0')
	}
	return n
}
