package xrayconf

import (
	"bytes"
	"strings"
	"testing"

	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/infra/conf/serial"
	_ "github.com/xtls/xray-core/main/distro/all"

	"github.com/freeb5d/kite/pkg/profile"
)

func TestVLESSEncryption(t *testing.T) {
	enc := "mlkem768x25519plus.native.0rtt.o6NZ0a1EJo29bvlnOqDUQO2rjfiSFw91q2dBqrOqtxw"
	s, err := profile.ParseLink("vless://id@h.example:443?encryption=" + enc + "&security=reality&pbk=K&sid=ab&sni=www.example.com&fp=chrome&type=tcp#x")
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := Build(s, Options{SOCKSPort: 1080})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfg), `"encryption":"`+enc+`"`) {
		t.Fatalf("encryption not passed through: %s", cfg)
	}

	plain, _ := profile.ParseLink("vless://id@h.example:443?security=tls&type=ws#y")
	cfg, _ = Build(plain, Options{SOCKSPort: 1080})
	if !strings.Contains(string(cfg), `"encryption":"none"`) {
		t.Fatalf("expected encryption none by default: %s", cfg)
	}
}

func TestHysteria2Outbound(t *testing.T) {
	s := profile.Server{Protocol: "hysteria2", Address: "example.com", Port: 443, Password: "pw",
		Extra: map[string]string{"obfs": "salamander", "obfs-password": "x", "mport": "20000-30000"}}
	cfg, err := Build(s, Options{SOCKSPort: 1080})
	if err != nil {
		t.Fatal(err)
	}
	config, err := serial.LoadJSONConfig(bytes.NewReader(cfg))
	if err != nil {
		t.Fatalf("xray rejected config: %v\n%s", err, cfg)
	}
	if _, err := core.New(config); err != nil {
		t.Fatalf("core.New: %v\n%s", err, cfg)
	}
}

func TestTUNDNSConfig(t *testing.T) {
	s := profile.Server{Protocol: "trojan", Address: "example.com", Port: 443, Password: "pw", Extra: map[string]string{"security": "tls"}}
	cfg, err := Build(s, Options{SOCKSPort: 1080, TUN: true, DirectPrivate: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(cfg), "dns-query") || !strings.Contains(string(cfg), "10.0.0.0/8") || !strings.Contains(string(cfg), "destOverride") || !strings.Contains(string(cfg), "blackhole") {
		t.Fatalf("missing DNS / private rules: %s", cfg)
	}
	if _, err := serial.LoadJSONConfig(bytes.NewReader(cfg)); err != nil {
		t.Fatalf("xray rejected config: %v\n%s", err, cfg)
	}
}

func TestSSHConfig(t *testing.T) {
	s := profile.Server{Protocol: "ssh", Address: "1.2.3.4", Port: 22, Password: "pw"}
	cfg, err := Build(s, Options{SOCKSPort: 1080, TUN: false, SSHBridgePort: 5555})
	if err != nil {
		t.Fatal(err)
	}
	config, err := serial.LoadJSONConfig(bytes.NewReader(cfg))
	if err != nil {
		t.Fatalf("xray rejected config: %v\n%s", err, cfg)
	}
	if _, err := core.New(config); err != nil {
		t.Fatalf("core.New: %v\n%s", err, cfg)
	}
}

func TestTransports(t *testing.T) {
	for _, extra := range []map[string]string{
		{"type": "httpupgrade", "path": "/up", "host": "h.example"},
		{"type": "xhttp", "path": "/x", "mode": "packet-up", "extra": `{"xPaddingBytes":"100-1000"}`},
		{"type": "h2", "path": "/h2", "host": "h.example", "security": "tls"},
		{"type": "kcp", "seed": "s", "headerType": "wechat-video"},
		{"type": "kcp"},
		{"type": "grpc", "serviceName": "svc", "mode": "multi", "security": "tls"},
		{"type": "ws", "security": "tls", "sni": "a.example", "ech": "cloudflare-ech.com+https://1.1.1.1/dns-query"},
		{"type": "tcp", "security": "tls", "mux": "1", "muxConcurrency": "4", "fragment": "1"},
		{"type": "ws", "fragment": "1", "fragmentPackets": "1-3", "fragmentLength": "10-20", "fragmentInterval": "5-10"},
	} {
		s := profile.Server{Protocol: "vless", Address: "example.com", Port: 443, UUID: "b831381d-6324-4d53-ad4f-8cda48b30811", Extra: extra}
		cfg, err := Build(s, Options{SOCKSPort: 1080})
		if err != nil {
			t.Fatal(err)
		}
		config, err := serial.LoadJSONConfig(bytes.NewReader(cfg))
		if err != nil {
			t.Fatalf("%v: xray rejected config: %v\n%s", extra, err, cfg)
		}
		if _, err := core.New(config); err != nil {
			t.Fatalf("%v: core.New: %v\n%s", extra, err, cfg)
		}
	}
}
