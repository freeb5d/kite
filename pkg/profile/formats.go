package profile

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/ghodss/yaml"
)

// parseStructured handles subscription bodies that aren't a list of share
// links: Clash/Mihomo YAML ("proxies:"), sing-box JSON ("outbounds"),
// Xray/V2Ray JSON ("outbounds" with "protocol"), and Shadowsocks SIP008
// JSON ("servers"). ok is false when the body is none of these.
func parseStructured(body string) (servers []Server, errs []error, ok bool) {
	trimmed := strings.TrimSpace(body)
	add := collector(&servers, &errs)

	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		var doc struct {
			Outbounds []map[string]any `json:"outbounds"`
			Servers   []map[string]any `json:"servers"`
		}
		if err := json.Unmarshal([]byte(trimmed), &doc); err != nil {
			return nil, []error{fmt.Errorf("invalid JSON subscription: %w", err)}, true
		}
		for _, o := range doc.Outbounds {
			if _, isXray := o["protocol"]; isXray {
				add(fromXray(o))
			} else {
				add(fromSingBox(o))
			}
		}
		for _, s := range doc.Servers {
			add(fromSIP008(s))
		}
		return servers, errs, true
	}

	if strings.Contains(body, "proxies:") {
		var doc struct {
			Proxies []map[string]any `json:"proxies"`
		}
		if err := yaml.Unmarshal([]byte(body), &doc); err != nil {
			return nil, []error{fmt.Errorf("invalid Clash YAML subscription: %w", err)}, true
		}
		for _, p := range doc.Proxies {
			add(fromClash(p))
		}
		return servers, errs, true
	}

	return nil, nil, false
}

// errSkip marks entries that aren't proxies (direct/block/selector groups).
var errSkip = fmt.Errorf("skip")

func collector(servers *[]Server, errs *[]error) func(Server, error) {
	return func(s Server, err error) {
		switch {
		case err == errSkip:
		case err != nil:
			*errs = append(*errs, err)
		default:
			*servers = append(*servers, s)
		}
	}
}

// --- loose field access over decoded JSON/YAML ---

func str(m map[string]any, keys ...string) string {
	for _, k := range keys {
		switch v := m[k].(type) {
		case string:
			if v != "" {
				return v
			}
		case float64:
			return strconv.FormatFloat(v, 'f', -1, 64)
		case bool:
			return strconv.FormatBool(v)
		}
	}
	return ""
}

func sub(m map[string]any, key string) map[string]any {
	if v, ok := m[key].(map[string]any); ok {
		return v
	}
	return map[string]any{}
}

func list(m map[string]any, key string) string {
	switch v := m[key].(type) {
	case []any:
		parts := make([]string, 0, len(v))
		for _, x := range v {
			if s, ok := x.(string); ok {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, ",")
	case string:
		return v
	}
	return ""
}

func truthy(m map[string]any, key string) bool {
	switch v := m[key].(type) {
	case bool:
		return v
	case string:
		return v == "true" || v == "tls"
	}
	return false
}

func newServer(name, protocol, address, port string) Server {
	return Server{Name: nameOrDefault(name, address), Protocol: protocol, Address: address, Port: atoiSafe(port), Extra: map[string]string{}}
}

func setIf(extra map[string]string, key, value string) {
	if value != "" {
		extra[key] = value
	}
}

// --- Clash / Mihomo ---

func fromClash(p map[string]any) (Server, error) {
	typ := str(p, "type")
	switch typ {
	case "vmess", "vless", "trojan", "ss":
	case "hysteria2":
		s := newServer(str(p, "name"), "hysteria2", str(p, "server"), str(p, "port"))
		s.Password = str(p, "password")
		e := s.Extra
		setIf(e, "sni", str(p, "sni"))
		setIf(e, "alpn", list(p, "alpn"))
		setIf(e, "obfs", str(p, "obfs"))
		setIf(e, "obfs-password", str(p, "obfs-password"))
		setIf(e, "mport", str(p, "ports"))
		if truthy(p, "skip-cert-verify") {
			e["insecure"] = "1"
		}
		return s, nil
	default:
		return Server{}, fmt.Errorf("%s: unsupported Clash proxy type %q", str(p, "name"), typ)
	}
	protocol := typ
	if typ == "ss" {
		protocol = "shadowsocks"
	}
	s := newServer(str(p, "name"), protocol, str(p, "server"), str(p, "port"))
	s.UUID = str(p, "uuid")
	s.Password = str(p, "password")
	s.Method = str(p, "cipher")
	if protocol == "vmess" {
		s.Method = ""
		setIf(s.Extra, "scy", str(p, "cipher"))
	}
	e := s.Extra
	setIf(e, "flow", str(p, "flow"))
	network := str(p, "network")
	setIf(e, "type", network)
	setIf(e, "sni", str(p, "servername", "sni"))
	setIf(e, "fp", str(p, "client-fingerprint"))
	setIf(e, "alpn", list(p, "alpn"))
	if reality := sub(p, "reality-opts"); len(reality) > 0 {
		e["security"] = "reality"
		setIf(e, "pbk", str(reality, "public-key"))
		setIf(e, "sid", str(reality, "short-id"))
	} else if truthy(p, "tls") || typ == "trojan" {
		e["security"] = "tls"
	}
	switch network {
	case "ws":
		ws := sub(p, "ws-opts")
		setIf(e, "path", str(ws, "path"))
		setIf(e, "host", str(sub(ws, "headers"), "Host", "host"))
	case "grpc":
		setIf(e, "serviceName", str(sub(p, "grpc-opts"), "grpc-service-name"))
	case "h2":
		h2 := sub(p, "h2-opts")
		setIf(e, "path", str(h2, "path"))
		setIf(e, "host", list(h2, "host"))
	case "http":
		e["type"] = "tcp"
		e["headerType"] = "http"
		h := sub(p, "http-opts")
		setIf(e, "path", list(h, "path"))
		setIf(e, "host", list(sub(h, "headers"), "Host"))
	}
	return s, nil
}

// --- sing-box ---

func fromSingBox(o map[string]any) (Server, error) {
	typ := str(o, "type")
	switch typ {
	case "vmess", "vless", "trojan", "shadowsocks":
	case "hysteria2":
		s := newServer(str(o, "tag"), "hysteria2", str(o, "server"), str(o, "server_port"))
		s.Password = str(o, "password")
		e := s.Extra
		tls := sub(o, "tls")
		setIf(e, "sni", str(tls, "server_name"))
		setIf(e, "alpn", list(tls, "alpn"))
		if truthy(tls, "insecure") {
			e["insecure"] = "1"
		}
		obfs := sub(o, "obfs")
		setIf(e, "obfs", str(obfs, "type"))
		setIf(e, "obfs-password", str(obfs, "password"))
		setIf(e, "mport", strings.ReplaceAll(list(o, "server_ports"), ":", "-"))
		return s, nil
	case "direct", "block", "dns", "selector", "urltest":
		return Server{}, errSkip
	default:
		return Server{}, fmt.Errorf("%s: unsupported sing-box outbound type %q", str(o, "tag"), typ)
	}
	s := newServer(str(o, "tag"), typ, str(o, "server"), str(o, "server_port"))
	s.UUID = str(o, "uuid")
	s.Password = str(o, "password")
	s.Method = str(o, "method")
	e := s.Extra
	setIf(e, "flow", str(o, "flow"))
	if typ == "vmess" {
		setIf(e, "scy", str(o, "security"))
	}
	if tls := sub(o, "tls"); truthy(tls, "enabled") {
		e["security"] = "tls"
		setIf(e, "sni", str(tls, "server_name"))
		setIf(e, "alpn", list(tls, "alpn"))
		setIf(e, "fp", str(sub(tls, "utls"), "fingerprint"))
		if reality := sub(tls, "reality"); truthy(reality, "enabled") {
			e["security"] = "reality"
			setIf(e, "pbk", str(reality, "public_key"))
			setIf(e, "sid", str(reality, "short_id"))
		}
	}
	if tr := sub(o, "transport"); len(tr) > 0 {
		setIf(e, "type", str(tr, "type"))
		setIf(e, "path", str(tr, "path"))
		setIf(e, "host", str(sub(tr, "headers"), "Host", "host"))
		setIf(e, "serviceName", str(tr, "service_name"))
		setIf(e, "host", list(tr, "host"))
	}
	return s, nil
}

// --- Xray / V2Ray JSON ---

func fromXray(o map[string]any) (Server, error) {
	protocol := str(o, "protocol")
	settings := sub(o, "settings")
	var target, user map[string]any
	switch protocol {
	case "vmess", "vless":
		target = firstOf(settings, "vnext")
		user = firstOf(target, "users")
	case "trojan", "shadowsocks":
		target = firstOf(settings, "servers")
		user = target
	case "hysteria":
		stream := sub(o, "streamSettings")
		s := newServer(str(o, "tag"), "hysteria2", str(settings, "address"), str(settings, "port"))
		s.Password = str(sub(stream, "hysteriaSettings"), "auth")
		e := s.Extra
		tls := sub(stream, "tlsSettings")
		setIf(e, "sni", str(tls, "serverName"))
		setIf(e, "alpn", list(tls, "alpn"))
		setIf(e, "pinSHA256", str(tls, "pinnedPeerCertSha256"))
		if truthy(tls, "allowInsecure") {
			e["insecure"] = "1"
		}
		fm := sub(stream, "finalmask")
		if udp, ok := fm["udp"].([]any); ok {
			for _, m := range udp {
				if mm, ok := m.(map[string]any); ok && str(mm, "type") == "salamander" {
					e["obfs"] = "salamander"
					setIf(e, "obfs-password", str(sub(mm, "settings"), "password"))
				}
			}
		}
		setIf(e, "mport", str(sub(sub(fm, "quicParams"), "udpHop"), "ports"))
		return s, nil
	case "freedom", "blackhole", "dns":
		return Server{}, errSkip
	default:
		return Server{}, fmt.Errorf("%s: unsupported Xray outbound protocol %q", str(o, "tag"), protocol)
	}
	s := newServer(str(o, "tag"), protocol, str(target, "address"), str(target, "port"))
	s.UUID = str(user, "id")
	s.Password = str(user, "password")
	s.Method = str(user, "method")
	e := s.Extra
	setIf(e, "flow", str(user, "flow"))
	if protocol == "vmess" {
		setIf(e, "scy", str(user, "security"))
	}

	stream := sub(o, "streamSettings")
	setIf(e, "type", str(stream, "network"))
	switch str(stream, "security") {
	case "tls":
		e["security"] = "tls"
		tls := sub(stream, "tlsSettings")
		setIf(e, "sni", str(tls, "serverName"))
		setIf(e, "alpn", list(tls, "alpn"))
		setIf(e, "fp", str(tls, "fingerprint"))
		setIf(e, "ech", str(tls, "echConfigList"))
	case "reality":
		e["security"] = "reality"
		r := sub(stream, "realitySettings")
		setIf(e, "sni", str(r, "serverName"))
		setIf(e, "fp", str(r, "fingerprint"))
		setIf(e, "pbk", str(r, "publicKey"))
		setIf(e, "sid", str(r, "shortId"))
		setIf(e, "spx", str(r, "spiderX"))
	}
	ws := sub(stream, "wsSettings")
	setIf(e, "path", str(ws, "path"))
	setIf(e, "host", str(sub(ws, "headers"), "Host", "host"))
	grpc := sub(stream, "grpcSettings")
	setIf(e, "serviceName", str(grpc, "serviceName"))
	if truthy(grpc, "multiMode") {
		e["mode"] = "multi"
	}
	for _, key := range []string{"httpupgradeSettings", "xhttpSettings", "splithttpSettings"} {
		t := sub(stream, key)
		setIf(e, "path", str(t, "path"))
		setIf(e, "host", str(t, "host"))
		setIf(e, "mode", str(t, "mode"))
		if x := sub(t, "extra"); len(x) > 0 {
			if b, err := json.Marshal(x); err == nil {
				e["extra"] = string(b)
			}
		}
	}
	kcp := sub(stream, "kcpSettings")
	setIf(e, "seed", str(kcp, "seed"))
	if h := str(sub(kcp, "header"), "type"); h != "" && h != "none" {
		e["headerType"] = h
	}
	if udp, ok := sub(stream, "finalmask")["udp"].([]any); ok {
		for _, m := range udp {
			mm, _ := m.(map[string]any)
			switch typ := str(mm, "type"); {
			case typ == "mkcp-aes128gcm":
				setIf(e, "seed", str(sub(mm, "settings"), "password"))
			case strings.HasPrefix(typ, "header-"):
				e["headerType"] = strings.TrimPrefix(typ, "header-")
			}
		}
	}
	if header := sub(sub(stream, "tcpSettings"), "header"); str(header, "type") == "http" {
		e["headerType"] = "http"
	}
	return s, nil
}

func firstOf(m map[string]any, key string) map[string]any {
	if arr, ok := m[key].([]any); ok && len(arr) > 0 {
		if first, ok := arr[0].(map[string]any); ok {
			return first
		}
	}
	return map[string]any{}
}

// --- Shadowsocks SIP008 ---

func fromSIP008(s map[string]any) (Server, error) {
	srv := newServer(str(s, "remarks"), "shadowsocks", str(s, "server"), str(s, "server_port"))
	srv.Method = str(s, "method")
	srv.Password = str(s, "password")
	return srv, nil
}
