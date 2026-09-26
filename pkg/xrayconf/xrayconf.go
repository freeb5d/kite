// Package xrayconf builds xray-core JSON configs from saved server
// profiles. It's shared by the desktop app and the Android app
// (github.com/freeb5d/kite-android), so both speak every link shape the
// same way.
package xrayconf

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/freeb5d/kite/pkg/profile"
)

// Options selects what the generated config listens on.
type Options struct {
	HTTPPort  int    // local HTTP proxy port; 0 = no HTTP inbound
	SOCKSPort int    // local SOCKS5 proxy port; 0 = no SOCKS inbound
	TUN       bool   // add xray's TUN inbound
	TUNName   string // TUN adapter name (desktop); ignored on Android
	LogPath   string // xray error/debug log file; empty = stderr
	LogLevel  string // xray loglevel; default "debug"
}

// Build returns an xray-core JSON config for server. The proxy outbound is
// tagged "proxy" and its traffic counters are registered as
// "outbound>>>proxy>>>traffic>>>uplink"/"downlink" in xray's stats.Manager.
func Build(server profile.Server, o Options) ([]byte, error) {
	outbound, err := Outbound(server)
	if err != nil {
		return nil, err
	}

	inbounds := []map[string]interface{}{}
	if o.HTTPPort > 0 {
		inbounds = append(inbounds, map[string]interface{}{
			"tag": "http-in", "listen": "127.0.0.1", "port": o.HTTPPort, "protocol": "http",
		})
	}
	if o.SOCKSPort > 0 {
		inbounds = append(inbounds, map[string]interface{}{
			"tag": "socks-in", "listen": "127.0.0.1", "port": o.SOCKSPort, "protocol": "socks",
			"settings": map[string]interface{}{"udp": true},
		})
	}
	if o.TUN {
		name := o.TUNName
		if name == "" {
			name = "xray0"
		}
		inbounds = append(inbounds, map[string]interface{}{
			"tag": "tun-in", "protocol": "tun", "port": 0,
			"settings": map[string]interface{}{"name": name, "MTU": 1500},
		})
	}

	logLevel := o.LogLevel
	if logLevel == "" {
		logLevel = "debug"
	}
	logCfg := map[string]interface{}{"loglevel": logLevel}
	if o.LogPath != "" {
		logCfg["error"] = o.LogPath
	}

	config := map[string]interface{}{
		"log":      logCfg,
		"inbounds": inbounds,
		"outbounds": []map[string]interface{}{
			outbound,
			{"tag": "direct", "protocol": "freedom"},
		},
		"policy": map[string]interface{}{
			"system": map[string]interface{}{
				"statsOutboundUplink":   true,
				"statsOutboundDownlink": true,
			},
		},
		"stats": map[string]interface{}{},
	}
	return json.Marshal(config)
}

// Outbound builds the "proxy" outbound for server.
func Outbound(server profile.Server) (map[string]interface{}, error) {
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
					"security": FirstNonEmpty(server.Extra["scy"], "auto"),
				}},
			}},
		}
	case "vless":
		user := map[string]interface{}{
			"id":         server.UUID,
			// Usually "none", but servers using xray's post-quantum VLESS
			// encryption (mlkem768x25519plus...) reject anything else.
			"encryption": FirstNonEmpty(server.Extra["encryption"], "none"),
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
	network := FirstNonEmpty(server.Extra["network"], server.Extra["type"], "tcp")
	stream := map[string]interface{}{"network": network}

	switch network {
	case "ws", "websocket":
		stream["network"] = "ws"
		ws := map[string]interface{}{"path": FirstNonEmpty(server.Extra["path"], "/")}
		if host := server.Extra["host"]; host != "" {
			ws["headers"] = map[string]interface{}{"Host": host}
		}
		stream["wsSettings"] = ws
	case "grpc":
		grpc := map[string]interface{}{}
		if svc := FirstNonEmpty(server.Extra["serviceName"], server.Extra["path"]); svc != "" {
			grpc["serviceName"] = svc
		}
		stream["grpcSettings"] = grpc
	default:
		stream["network"] = "tcp"
		// headerType=http disguises a raw TCP connection as a plaintext
		// HTTP request/response, for servers behind an HTTP-sniffing front end.
		if server.Extra["headerType"] == "http" {
			host := FirstNonEmpty(server.Extra["host"], server.Address)
			stream["tcpSettings"] = map[string]interface{}{
				"header": map[string]interface{}{
					"type": "http",
					"request": map[string]interface{}{
						"path":    []string{FirstNonEmpty(server.Extra["path"], "/")},
						"headers": map[string]interface{}{"Host": []string{host}},
					},
				},
			}
		}
	}

	security := FirstNonEmpty(server.Extra["security"], server.Extra["tls"])
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
	if sni := FirstNonEmpty(server.Extra["sni"], server.Extra["host"], server.Address); sni != "" {
		tls["serverName"] = sni
	}
	// WebSocket is an HTTP/1.1 Upgrade; offering h2 lets the server pick it
	// and every dial then fails with `websocket: protocol "h2" was given
	// but is not supported`.
	network := FirstNonEmpty(server.Extra["network"], server.Extra["type"])
	if network == "ws" || network == "websocket" {
		tls["alpn"] = []string{"http/1.1"}
	} else if alpn := server.Extra["alpn"]; alpn != "" {
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
	if sni := FirstNonEmpty(server.Extra["sni"], server.Extra["host"], server.Address); sni != "" {
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

// FirstNonEmpty returns the first non-empty value.
func FirstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
