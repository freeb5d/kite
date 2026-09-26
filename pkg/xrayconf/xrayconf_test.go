package xrayconf

import (
	"strings"
	"testing"

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
