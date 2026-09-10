package xray

import (
	"fmt"

	"github.com/freeb5d/kite/internal/profile"
)

// Config is a minimal stand-in for the real xray-core JSON config
// (inbounds/outbounds/routing). It will be replaced with the actual
// xray-core conf types once the dependency is wired in with `go mod tidy`.
type Config struct {
	Inbounds  []Inbound  `json:"inbounds"`
	Outbounds []Outbound `json:"outbounds"`
}

type Inbound struct {
	Tag      string `json:"tag"`
	Protocol string `json:"protocol"` // "http" or "socks"
	Port     int    `json:"port"`
}

type Outbound struct {
	Tag      string      `json:"tag"`
	Protocol string      `json:"protocol"` // vmess, vless, trojan, shadowsocks
	Settings interface{} `json:"settings"`
}

// BuildConfig turns a saved server profile into an xray-core config with a
// local HTTP/SOCKS inbound (phase 1: system proxy only, no TUN).
func BuildConfig(server profile.Server) (*Config, error) {
	if server.Protocol == "" {
		return nil, fmt.Errorf("server %q has no protocol set", server.Name)
	}

	return &Config{
		Inbounds: []Inbound{
			{Tag: "http-in", Protocol: "http", Port: 2080},
			{Tag: "socks-in", Protocol: "socks", Port: 2081},
		},
		Outbounds: []Outbound{
			{Tag: "proxy", Protocol: server.Protocol, Settings: server},
		},
	}, nil
}
