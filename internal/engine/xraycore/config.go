// Package xraycore wraps xray-core (github.com/xtls/xray-core) as an
// in-process Go library, the engine Kite originally shipped with before
// the v0.9.0 sing-box migration. It's kept alongside sing-box (see
// internal/engine/singbox) because xray-core supports link shapes
// sing-box has no equivalent for -- notably tcp+headerType=http -- at
// the cost of no TUN support here (see manager.go).
package xraycore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/freeb5d/kite/internal/profile"
)

const (
	HTTPInboundPort  = 2080
	SOCKSInboundPort = 2081
)

// Mode mirrors internal/xray's engine.Mode as a plain string so this
// package has no dependency on the facade.
type Mode = string

const (
	ModeProxy Mode = "proxy"
	ModeTUN   Mode = "tun"
)

// CoreVersion returns the embedded xray-core version, for the About panel.
// xray-core doesn't expose a runtime version constant, so this reads the
// resolved module version straight from the Go module graph instead.
func CoreVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range info.Deps {
			if dep.Path == "github.com/xtls/xray-core" {
				return strings.TrimPrefix(dep.Version, "v")
			}
		}
	}
	return "unknown"
}

// LogFilePath returns where xray-core's own error log is written.
func LogFilePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	logDir := filepath.Join(dir, "kite", "logs")
	_ = os.MkdirAll(logDir, 0o700)
	return filepath.Join(logDir, "xray.log")
}

// buildJSON turns a saved server profile into an xray-core JSON config
// (the same V2Ray-compatible shape xray-core's own config file uses),
// parsed through core.StartInstance("json", ...) -- see manager.go.
func buildJSON(server profile.Server) ([]byte, error) {
	outbound, err := outboundJSON(server)
	if err != nil {
		return nil, err
	}

	config := map[string]interface{}{
		"log": map[string]interface{}{
			"loglevel": "debug",
			"error":    LogFilePath(),
		},
		"inbounds": []map[string]interface{}{
			{
				"tag":      "http-in",
				"listen":   "127.0.0.1",
				"port":     HTTPInboundPort,
				"protocol": "http",
			},
			{
				"tag":      "socks-in",
				"listen":   "127.0.0.1",
				"port":     SOCKSInboundPort,
				"protocol": "socks",
				"settings": map[string]interface{}{"udp": true},
			},
		},
		"outbounds": []map[string]interface{}{
			outbound,
			{"tag": "direct", "protocol": "freedom"},
		},
	}

	return json.Marshal(config)
}

func outboundJSON(server profile.Server) (map[string]interface{}, error) {
	outbound := map[string]interface{}{"tag": "proxy"}

	switch server.Protocol {
	case "vmess":
		outbound["protocol"] = "vmess"
		outbound["settings"] = map[string]interface{}{
			"vnext": []map[string]interface{}{{
				"address": server.Address,
				"port":    server.Port,
				"users": []map[string]interface{}{{
					"id":       server.UUID,
					"security": "auto",
				}},
			}},
		}
	case "vless":
		user := map[string]interface{}{
			"id":         server.UUID,
			"encryption": "none",
		}
		if flow := server.Extra["flow"]; flow != "" {
			user["flow"] = flow
		}
		outbound["protocol"] = "vless"
		outbound["settings"] = map[string]interface{}{
			"vnext": []map[string]interface{}{{
				"address": server.Address,
				"port":    server.Port,
				"users":   []map[string]interface{}{user},
			}},
		}
	case "trojan":
		outbound["protocol"] = "trojan"
		outbound["settings"] = map[string]interface{}{
			"servers": []map[string]interface{}{{
				"address":  server.Address,
				"port":     server.Port,
				"password": server.Password,
			}},
		}
	case "shadowsocks":
		outbound["protocol"] = "shadowsocks"
		outbound["settings"] = map[string]interface{}{
			"servers": []map[string]interface{}{{
				"address":  server.Address,
				"port":     server.Port,
				"method":   server.Method,
				"password": server.Password,
			}},
		}
	default:
		return nil, fmt.Errorf("unsupported protocol %q", server.Protocol)
	}

	outbound["streamSettings"] = streamSettingsJSON(server)
	return outbound, nil
}

func streamSettingsJSON(server profile.Server) map[string]interface{} {
	network := firstNonEmpty(server.Extra["network"], server.Extra["type"], "tcp")
	stream := map[string]interface{}{"network": network}

	switch network {
	case "ws", "websocket":
		stream["network"] = "ws"
		ws := map[string]interface{}{"path": firstNonEmpty(server.Extra["path"], "/")}
		if host := server.Extra["host"]; host != "" {
			ws["headers"] = map[string]interface{}{"Host": host}
		}
		stream["wsSettings"] = ws
	case "grpc":
		grpc := map[string]interface{}{}
		if svc := firstNonEmpty(server.Extra["serviceName"], server.Extra["path"]); svc != "" {
			grpc["serviceName"] = svc
		}
		stream["grpcSettings"] = grpc
	default:
		stream["network"] = "tcp"
		// headerType=http disguises a raw TCP connection as a plaintext
		// HTTP request/response -- this is the one transport sing-box has
		// no equivalent for, and the reason this engine still exists.
		if server.Extra["headerType"] == "http" {
			host := firstNonEmpty(server.Extra["host"], server.Address)
			stream["tcpSettings"] = map[string]interface{}{
				"header": map[string]interface{}{
					"type": "http",
					"request": map[string]interface{}{
						"path":    []string{firstNonEmpty(server.Extra["path"], "/")},
						"headers": map[string]interface{}{"Host": []string{host}},
					},
				},
			}
		}
	}

	security := firstNonEmpty(server.Extra["security"], server.Extra["tls"])
	switch security {
	case "tls":
		stream["security"] = "tls"
		stream["tlsSettings"] = tlsSettingsJSON(server)
	case "reality":
		stream["security"] = "reality"
		stream["realitySettings"] = realitySettingsJSON(server)
	}

	return stream
}

func tlsSettingsJSON(server profile.Server) map[string]interface{} {
	tls := map[string]interface{}{}
	if sni := firstNonEmpty(server.Extra["sni"], server.Extra["host"], server.Address); sni != "" {
		tls["serverName"] = sni
	}
	if alpn := server.Extra["alpn"]; alpn != "" {
		tls["alpn"] = strings.Split(alpn, ",")
	}
	if fp := server.Extra["fp"]; fp != "" {
		tls["fingerprint"] = fp
	}
	return tls
}

func realitySettingsJSON(server profile.Server) map[string]interface{} {
	reality := map[string]interface{}{
		"publicKey": server.Extra["pbk"],
		"shortId":   server.Extra["sid"],
	}
	if sni := firstNonEmpty(server.Extra["sni"], server.Extra["host"], server.Address); sni != "" {
		reality["serverName"] = sni
	}
	fp := server.Extra["fp"]
	if fp == "" {
		fp = "chrome"
	}
	reality["fingerprint"] = fp
	if spx := server.Extra["spx"]; spx != "" {
		reality["spiderX"] = spx
	}
	return reality
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
