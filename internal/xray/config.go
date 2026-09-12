package xray

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

	// TUNGateway is the address (with netmask) sing-box assigns to the
	// TUN adapter itself when it's brought up.
	TUNGateway = "172.19.0.1/30"
)

// Mode selects what the engine listens on: a local HTTP/SOCKS proxy the OS
// or individual apps are pointed at, or a TUN network adapter that
// captures all IP traffic routed to it.
type Mode string

const (
	ModeProxy Mode = "proxy"
	ModeTUN   Mode = "tun"
)

// CoreVersion returns the embedded sing-box version (e.g. "1.11.0"), for
// display in the About panel. sing-box's own constant.Version is only set
// via a build-time ldflag we don't pass, so this reads the resolved
// module version straight from the Go module graph instead -- works on
// any build without extra ldflags.
func CoreVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range info.Deps {
			if dep.Path == "github.com/sagernet/sing-box" {
				return strings.TrimPrefix(dep.Version, "v")
			}
		}
	}
	return "unknown"
}

// LogFilePath returns where sing-box's own error log is written, so it can
// be surfaced in the UI when a connection silently fails to actually route
// traffic (started fine, but the outbound handshake/dial is failing).
func LogFilePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	logDir := filepath.Join(dir, "kite", "logs")
	_ = os.MkdirAll(logDir, 0o700)
	return filepath.Join(logDir, "sing-box.log")
}

// buildJSON turns a saved server profile into a sing-box JSON config --
// the same shape sing-box's own config file uses, so it's parsed through
// sing-box's own decoder (see manager.go) rather than us re-implementing
// its internal option types.
//
// NOTE: sing-box has no equivalent of xray-core's tcp "headerType=http"
// disguise (a raw TCP connection dressed up as a plaintext HTTP request
// so an HTTP-sniffing front end doesn't reject it) -- its transports are
// ws/http/grpc/httpupgrade/quic only, so a link relying on that specific
// obfuscation won't work after this migration. Every other transport/
// security combination Kite supported under xray-core (tcp, ws, tls,
// REALITY) is covered here.
func buildJSON(server profile.Server, mode Mode) ([]byte, error) {
	outbound, err := outboundJSON(server)
	if err != nil {
		return nil, err
	}

	inbounds := []map[string]interface{}{
		{
			"type":        "http",
			"tag":         "http-in",
			"listen":      "127.0.0.1",
			"listen_port": HTTPInboundPort,
		},
		{
			"type":        "mixed",
			"tag":         "socks-in",
			"listen":      "127.0.0.1",
			"listen_port": SOCKSInboundPort,
		},
	}

	if mode == ModeTUN {
		// sing-box brings up the TUN adapter and its own default route
		// itself (auto_route) the same way xray-core's TUN inbound did;
		// the exception route for the VPN server's own IP -- required so
		// the engine's own outbound connection doesn't loop back through
		// the adapter it's feeding -- is added separately by the caller
		// (internal/system.AddExceptionRoute) *before* this inbound comes
		// up, same as before.
		inbounds = append(inbounds, map[string]interface{}{
			"type":           "tun",
			"tag":            "tun-in",
			"interface_name": "kite-tun",
			"address":        []string{TUNGateway},
			"mtu":            1500,
			"auto_route":     true,
			// "gvisor" needs the with_gvisor Go build tag (an extra
			// netstack dependency) that release.yml's build step doesn't
			// pass, so it always fails at runtime with "gVisor is not
			// included in this build". "system" uses the OS's own TUN
			// handling instead and needs nothing extra.
			"stack": "system",
		})
	}

	config := map[string]interface{}{
		"log": map[string]interface{}{
			"level":  "debug",
			"output": LogFilePath(),
		},
		"inbounds": inbounds,
		"outbounds": []map[string]interface{}{
			outbound,
			{"type": "direct", "tag": "direct"},
		},
	}

	return json.Marshal(config)
}

// outboundJSON builds the "proxy" outbound entry (protocol fields +
// transport + tls) from the extra fields captured while parsing the
// share link.
func outboundJSON(server profile.Server) (map[string]interface{}, error) {
	base := map[string]interface{}{
		"tag":         "proxy",
		"server":      server.Address,
		"server_port": server.Port,
	}

	switch server.Protocol {
	case "vmess":
		base["type"] = "vmess"
		base["uuid"] = server.UUID
		base["security"] = "auto"
	case "vless":
		base["type"] = "vless"
		base["uuid"] = server.UUID
		if flow := server.Extra["flow"]; flow != "" {
			base["flow"] = flow
		}
	case "trojan":
		base["type"] = "trojan"
		base["password"] = server.Password
	case "shadowsocks":
		base["type"] = "shadowsocks"
		base["method"] = server.Method
		base["password"] = server.Password
	default:
		return nil, fmt.Errorf("unsupported protocol %q", server.Protocol)
	}

	if tls := tlsJSON(server); tls != nil {
		base["tls"] = tls
	}
	if transport := transportJSON(server); transport != nil {
		base["transport"] = transport
	}
	return base, nil
}

// tlsJSON builds the outbound "tls" object, including REALITY/uTLS, from
// the extra fields captured while parsing the share link. vmess:// links
// are base64 JSON and use "tls" (mapped by parser.go to Extra["tls"]);
// vless://, trojan://, ss:// links are plain query strings that use
// "security" instead -- both are checked here since Extra just holds
// whatever the link actually used.
func tlsJSON(server profile.Server) map[string]interface{} {
	security := firstNonEmpty(server.Extra["security"], server.Extra["tls"])
	if security != "tls" && security != "reality" {
		return nil
	}

	tls := map[string]interface{}{"enabled": true}
	// A link can specify security=tls with no sni/host at all (or host set
	// to an empty string, as some generators do) -- xray-core silently
	// defaulted TLS's SNI to the server address in that case, but sing-box
	// sends no SNI extension at all unless server_name is set, which many
	// TLS termination points (CDNs especially) reject outright. Match
	// xray-core's behavior so these links keep working.
	if sni := firstNonEmpty(server.Extra["sni"], server.Extra["host"], server.Address); sni != "" {
		tls["server_name"] = sni
	}
	network := firstNonEmpty(server.Extra["network"], server.Extra["type"], "tcp")
	if alpn := server.Extra["alpn"]; alpn != "" {
		if network == "ws" || network == "websocket" {
			// WebSocket transport is a plain HTTP/1.1 Upgrade -- if h2 (or
			// h3, which is QUIC-only and meaningless here anyway) is in the
			// offered ALPN list, a TLS-terminating edge that prefers h2
			// (Cloudflare, notably) will negotiate it, turning the
			// connection into an HTTP/2 stream that sing-box's WS client
			// can't upgrade over. It handshakes fine and then gets closed
			// right after, which otherwise looks identical to a working
			// connection until that point. Force http/1.1 for ws.
			tls["alpn"] = []string{"http/1.1"}
		} else {
			tls["alpn"] = strings.Split(alpn, ",")
		}
	}
	fp := server.Extra["fp"]
	if fp == "" && security == "reality" {
		// REALITY only works because the client mimics a real browser's TLS
		// fingerprint (uTLS) instead of Go's own -- without it the reality
		// server can't distinguish us from a probe and falls back to
		// serving its camouflage site in plaintext, which is what causes
		// the "unknown version" TLS-parse error on our end. Most share
		// links set fp explicitly, but default to chrome when one's
		// missing since REALITY is unusable without some fingerprint.
		fp = "chrome"
	}
	if fp != "" {
		tls["utls"] = map[string]interface{}{"enabled": true, "fingerprint": fp}
	}
	if security == "reality" {
		tls["reality"] = map[string]interface{}{
			"enabled":    true,
			"public_key": server.Extra["pbk"],
			"short_id":   server.Extra["sid"],
		}
	}
	return tls
}

// transportJSON builds the outbound "transport" object for ws/grpc.
// Plain tcp (the default when no transport is specified) needs nothing
// here.
func transportJSON(server profile.Server) map[string]interface{} {
	network := firstNonEmpty(server.Extra["network"], server.Extra["type"], "tcp")
	switch network {
	case "ws", "websocket":
		transport := map[string]interface{}{
			"type": "ws",
			"path": firstNonEmpty(server.Extra["path"], "/"),
		}
		if host := server.Extra["host"]; host != "" {
			transport["headers"] = map[string]interface{}{"Host": host}
		}
		return transport
	case "grpc":
		transport := map[string]interface{}{"type": "grpc"}
		if svc := firstNonEmpty(server.Extra["serviceName"], server.Extra["path"]); svc != "" {
			transport["service_name"] = svc
		}
		return transport
	default:
		return nil
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
