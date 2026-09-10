package xray

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/freeb5d/kite/internal/profile"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/infra/conf"
)

const (
	HTTPInboundPort  = 2080
	SOCKSInboundPort = 2081
)

// LogFilePath returns where xray-core's own error log is written, so it can
// be surfaced in the UI when a connection silently fails to actually route
// traffic (started fine, but the outbound handshake/dial is failing).
func LogFilePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	logDir := filepath.Join(dir, "kite", "logs")
	_ = os.MkdirAll(logDir, 0o700)
	return filepath.Join(logDir, "xray.log")
}

// BuildConfig turns a saved server profile into a *core.Config by building
// the standard Xray JSON config shape and running it through xray-core's
// own conf.Config loader — the same path xray-core uses when loading a
// config file from disk, so every protocol/transport it understands is
// supported without us re-implementing its internal proto types.
func BuildConfig(server profile.Server) (*core.Config, error) {
	raw, err := buildJSON(server)
	if err != nil {
		return nil, err
	}

	var jsonConfig conf.Config
	if err := json.Unmarshal(raw, &jsonConfig); err != nil {
		return nil, fmt.Errorf("invalid generated xray config: %w", err)
	}

	pbConfig, err := jsonConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build xray config: %w", err)
	}
	return pbConfig, nil
}

func buildJSON(server profile.Server) ([]byte, error) {
	outbound, err := outboundSettings(server)
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
				"protocol": "http",
				"listen":   "127.0.0.1",
				"port":     HTTPInboundPort,
			},
			{
				"tag":      "socks-in",
				"protocol": "socks",
				"listen":   "127.0.0.1",
				"port":     SOCKSInboundPort,
				"settings": map[string]interface{}{"udp": true},
			},
		},
		"outbounds": []map[string]interface{}{
			{
				"tag":            "proxy",
				"protocol":       server.Protocol,
				"settings":       outbound,
				"streamSettings": streamSettings(server),
			},
			{
				"tag":      "direct",
				"protocol": "freedom",
			},
		},
	}

	return json.Marshal(config)
}

// outboundSettings builds the protocol-specific "settings" object using the
// classic nested "vnext"/"servers" array shape (rather than the newer flat
// address/port/id shorthand some conf.*OutboundConfig types also accept),
// since the nested shape is understood by every xray-core version we might
// end up building against.
func outboundSettings(server profile.Server) (map[string]interface{}, error) {
	switch server.Protocol {
	case "vmess":
		return map[string]interface{}{
			"vnext": []map[string]interface{}{
				{
					"address": server.Address,
					"port":    server.Port,
					"users": []map[string]interface{}{
						{
							"id":       server.UUID,
							"security": firstNonEmpty(server.Extra["security"], "auto"),
						},
					},
				},
			},
		}, nil
	case "vless":
		return map[string]interface{}{
			"vnext": []map[string]interface{}{
				{
					"address": server.Address,
					"port":    server.Port,
					"users": []map[string]interface{}{
						{
							"id":         server.UUID,
							"encryption": firstNonEmpty(server.Extra["encryption"], "none"),
							"flow":       server.Extra["flow"],
						},
					},
				},
			},
		}, nil
	case "trojan":
		return map[string]interface{}{
			"servers": []map[string]interface{}{
				{
					"address":  server.Address,
					"port":     server.Port,
					"password": server.Password,
				},
			},
		}, nil
	case "shadowsocks":
		return map[string]interface{}{
			"servers": []map[string]interface{}{
				{
					"address":  server.Address,
					"port":     server.Port,
					"method":   server.Method,
					"password": server.Password,
				},
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported protocol %q", server.Protocol)
	}
}

// streamSettings builds the "streamSettings" object (network + optional TLS)
// from the extra fields captured while parsing the share link. vmess://
// links are base64 JSON and use "net"/"tls" (mapped by parser.go to
// Extra["network"]/Extra["tls"]); vless://, trojan://, ss:// links are
// plain query strings that use "type"/"security" instead -- both are
// checked here since Extra just holds whatever the link actually used.
func streamSettings(server profile.Server) map[string]interface{} {
	network := firstNonEmpty(server.Extra["network"], server.Extra["type"], "tcp")
	settings := map[string]interface{}{"network": network}

	security := firstNonEmpty(server.Extra["security"], server.Extra["tls"])
	if security == "tls" || security == "reality" {
		settings["security"] = "tls"
		tlsSettings := map[string]interface{}{"allowInsecure": false}
		if sni := firstNonEmpty(server.Extra["sni"], server.Extra["host"]); sni != "" {
			tlsSettings["serverName"] = sni
		}
		// alpn is skipped for ws/websocket: the WS transport shares its
		// tls.Config with net/http, and an alpn list that includes "h2"
		// makes Go negotiate HTTP/2 instead of the plain WS upgrade,
		// failing every dial with `websocket: protocol "h2" was given
		// but is not supported`. ws requires http/1.1 semantics anyway,
		// so there's nothing useful alpn would add here.
		if alpn := server.Extra["alpn"]; alpn != "" && network != "ws" && network != "websocket" {
			tlsSettings["alpn"] = alpn
		}
		if fp := server.Extra["fp"]; fp != "" {
			tlsSettings["fingerprint"] = fp
		}
		settings["tlsSettings"] = tlsSettings
	}

	switch network {
	case "ws", "websocket":
		ws := map[string]interface{}{"path": firstNonEmpty(server.Extra["path"], "/")}
		if host := server.Extra["host"]; host != "" {
			ws["headers"] = map[string]interface{}{"Host": host}
		}
		settings["wsSettings"] = ws
	}

	return settings
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
