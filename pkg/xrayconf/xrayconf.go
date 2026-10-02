// Package xrayconf builds xray-core JSON configs from saved server
// profiles. It's shared by the desktop app and the Android app
// (github.com/freeb5d/kite-android), so both speak every link shape the
// same way.
package xrayconf

import (
	"encoding/json"
	"fmt"
	"strconv"
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

	// SSHBridgePort is the local SOCKS port of a running sshbridge.Bridge;
	// required for "ssh" servers, which xray-core can't speak itself.
	SSHBridgePort int
}

// Build returns an xray-core JSON config for server. The proxy outbound is
// tagged "proxy" and its traffic counters are registered as
// "outbound>>>proxy>>>traffic>>>uplink"/"downlink" in xray's stats.Manager.
func Build(server profile.Server, o Options) ([]byte, error) {
	var outbound map[string]interface{}
	var err error
	if server.Protocol == "ssh" {
		if o.SSHBridgePort == 0 {
			return nil, fmt.Errorf("ssh server needs a running SSH bridge")
		}
		outbound = map[string]interface{}{
			"tag": "proxy", "protocol": "socks",
			"settings": map[string]interface{}{
				"servers": []map[string]interface{}{{"address": "127.0.0.1", "port": o.SSHBridgePort}},
			},
		}
	} else if outbound, err = Outbound(server); err != nil {
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
	if server.Protocol == "ssh" {
		// SSH carries TCP only, so plain UDP DNS (e.g. from the TUN) is
		// answered by xray's DNS module, which asks 1.1.1.1 over TCP
		// through the tunnel.
		config["outbounds"] = append(config["outbounds"].([]map[string]interface{}),
			map[string]interface{}{"tag": "dns-out", "protocol": "dns"})
		config["dns"] = map[string]interface{}{"servers": []string{"tcp://1.1.1.1", "tcp://8.8.8.8"}}
		config["routing"] = map[string]interface{}{
			"rules": []map[string]interface{}{
				{"type": "field", "network": "udp", "port": "53", "outboundTag": "dns-out"},
			},
		}
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
			"id": server.UUID,
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
	case "hysteria2":
		outbound["protocol"] = "hysteria"
		outbound["settings"] = map[string]interface{}{
			"version": 2,
			"address": server.Address,
			"port":    server.Port,
		}
		outbound["streamSettings"] = hysteriaStreamJSON(server)
		return outbound, nil
	default:
		return nil, fmt.Errorf("unsupported protocol %q", server.Protocol)
	}

	outbound["streamSettings"] = streamSettingsJSON(server)
	if server.Extra["mux"] == "1" {
		concurrency, _ := strconv.Atoi(server.Extra["muxConcurrency"])
		if concurrency <= 0 {
			concurrency = 8
		}
		outbound["mux"] = map[string]interface{}{"enabled": true, "concurrency": concurrency}
	}
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
		if server.Extra["mode"] == "multi" {
			grpc["multiMode"] = true
		}
		if a := server.Extra["authority"]; a != "" {
			grpc["authority"] = a
		}
		stream["grpcSettings"] = grpc
	case "httpupgrade":
		hu := map[string]interface{}{"path": FirstNonEmpty(server.Extra["path"], "/")}
		if host := server.Extra["host"]; host != "" {
			hu["host"] = host
		}
		stream["httpupgradeSettings"] = hu
	case "xhttp", "splithttp", "h2", "http":
		// xray-core dropped the plain HTTP/2 transport; XHTTP in stream-one
		// mode is its replacement, so h2 servers are carried over XHTTP.
		stream["network"] = "xhttp"
		xh := map[string]interface{}{
			"path": FirstNonEmpty(server.Extra["path"], "/"),
			"mode": FirstNonEmpty(server.Extra["mode"], "auto"),
		}
		if network == "h2" || network == "http" {
			xh["mode"] = "stream-one"
		}
		if host := server.Extra["host"]; host != "" {
			xh["host"] = host
		}
		// Share links carry advanced XHTTP options as a JSON "extra" param.
		if raw := server.Extra["extra"]; raw != "" {
			var extra map[string]interface{}
			if json.Unmarshal([]byte(raw), &extra) == nil {
				xh["extra"] = extra
			}
		}
		stream["xhttpSettings"] = xh
	case "kcp", "mkcp":
		stream["network"] = "kcp"
		stream["kcpSettings"] = map[string]interface{}{}
		// xray-core moved mKCP's seed and header disguise into finalmask:
		// the seed becomes mkcp-aes128gcm (no seed = mkcp-original) and
		// headerType becomes a header-* UDP mask.
		masks := []map[string]interface{}{{"type": "mkcp-original"}}
		if seed := server.Extra["seed"]; seed != "" {
			masks[0] = map[string]interface{}{"type": "mkcp-aes128gcm", "settings": map[string]interface{}{"password": seed}}
		}
		if h := kcpHeaderMask(server.Extra["headerType"]); h != "" {
			masks = append(masks, map[string]interface{}{"type": h})
		}
		stream["finalmask"] = map[string]interface{}{"udp": masks}
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

	// Fragment splits the first packets (by default the TLS ClientHello)
	// into small, delayed pieces to slip past SNI-based filtering.
	if server.Extra["fragment"] == "1" && stream["network"] != "kcp" {
		stream["finalmask"] = map[string]interface{}{
			"tcp": []map[string]interface{}{{
				"type": "fragment",
				"settings": map[string]interface{}{
					"packets": FirstNonEmpty(server.Extra["fragmentPackets"], "tlshello"),
					"length":  FirstNonEmpty(server.Extra["fragmentLength"], "100-200"),
					"delay":   FirstNonEmpty(server.Extra["fragmentInterval"], "10-20"),
				},
			}},
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
	addECH(tls, server)
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

func kcpHeaderMask(h string) string {
	switch h {
	case "srtp", "utp", "dtls", "wireguard", "dns":
		return "header-" + h
	case "wechat-video", "wechat":
		return "header-wechat"
	}
	return ""
}

// hysteriaStreamJSON is Hysteria2 over QUIC: always TLS (ALPN h3), with
// optional Salamander obfuscation and UDP port hopping, which xray-core
// configures through finalmask.
func hysteriaStreamJSON(server profile.Server) map[string]interface{} {
	e := server.Extra
	tls := map[string]interface{}{
		"serverName": FirstNonEmpty(e["sni"], e["peer"], server.Address),
		"alpn":       []string{"h3"},
	}
	if alpn := e["alpn"]; alpn != "" {
		tls["alpn"] = strings.Split(alpn, ",")
	}
	// xray-core no longer supports allowInsecure; a self-signed server
	// is trusted by pinning its certificate instead.
	if pin := e["pinSHA256"]; pin != "" {
		tls["pinnedPeerCertSha256"] = pin
	}
	addECH(tls, server)
	stream := map[string]interface{}{
		"network":     "hysteria",
		"security":    "tls",
		"tlsSettings": tls,
		"hysteriaSettings": map[string]interface{}{
			"version": 2,
			"auth":    server.Password,
		},
	}
	mask := map[string]interface{}{}
	if e["obfs"] == "salamander" {
		mask["udp"] = []map[string]interface{}{{
			"type":     "salamander",
			"settings": map[string]interface{}{"password": e["obfs-password"]},
		}}
	}
	if ports := e["mport"]; ports != "" {
		mask["quicParams"] = map[string]interface{}{
			"udpHop": map[string]interface{}{"ports": ports},
		}
	}
	if len(mask) > 0 {
		stream["finalmask"] = mask
	}
	return stream
}

// addECH enables Encrypted Client Hello, which hides the real SNI. The
// "ech" value is either a base64 ECHConfigList or a domain plus DNS server
// to fetch it from (e.g. "cloudflare-ech.com+https://1.1.1.1/dns-query").
func addECH(tls map[string]interface{}, server profile.Server) {
	if ech := server.Extra["ech"]; ech != "" {
		tls["echConfigList"] = ech
		if force := server.Extra["echForceQuery"]; force != "" {
			tls["echForceQuery"] = force
		}
	}
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
