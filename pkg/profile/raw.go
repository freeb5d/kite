package profile

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Transports Kite can't model in its own server fields; outbounds using them
// are kept verbatim (Extra["raw"]) and handed to xray-core unchanged.
var rawNetworks = map[string]bool{"xdrive": true}

// keepRaw stores outbound o verbatim on s when its transport needs it.
func keepRaw(o map[string]any, s Server, err error) (Server, error) {
	if err != nil || !rawNetworks[str(sub(o, "streamSettings"), "network")] {
		return s, err
	}
	normalizeXDrive(o)
	b, jerr := json.Marshal(o)
	if jerr != nil {
		return s, jerr
	}
	if s.Extra == nil {
		s.Extra = map[string]string{}
	}
	s.Extra["raw"] = string(b)
	s.Extra["type"] = "xdrive"
	return s, nil
}

// normalizeXDrive accepts the panel layout (tuning values grouped under
// "tuning", a "front" object inside xdriveSettings) and rewrites it to
// xray-core's: tuning fields flat in xdriveSettings, the front as the
// stream-level address/port.
func normalizeXDrive(o map[string]any) {
	stream := sub(o, "streamSettings")
	xs := sub(stream, "xdriveSettings")
	if len(xs) == 0 {
		return
	}
	if tuning := sub(xs, "tuning"); len(tuning) > 0 {
		for k, v := range tuning {
			if _, set := xs[k]; !set {
				xs[k] = v
			}
		}
		delete(xs, "tuning")
	}
	if front := sub(xs, "front"); len(front) > 0 {
		if addr := cleanHost(str(front, "address")); addr != "" {
			stream["address"] = addr
		}
		if port, ok := front["port"]; ok {
			stream["port"] = port
		}
		delete(xs, "front")
	}
}

// cleanHost undoes a host pasted as a Markdown link: "[host](https://host)".
func cleanHost(h string) string {
	h = strings.TrimSpace(h)
	if strings.HasPrefix(h, "[") {
		if i := strings.Index(h, "]"); i > 0 {
			h = h[1:i]
		}
	}
	h = strings.TrimPrefix(strings.TrimPrefix(h, "https://"), "http://")
	return strings.TrimSuffix(h, "/")
}

// parseConfigJSON reads a pasted Xray/sing-box JSON config and returns its
// first proxy outbound.
func parseConfigJSON(body string) (Server, error) {
	servers, errs, _ := parseStructured(body)
	if len(servers) > 0 {
		return servers[0], nil
	}
	if len(errs) > 0 {
		return Server{}, errs[0]
	}
	return Server{}, fmt.Errorf("no proxy outbound in this JSON")
}
