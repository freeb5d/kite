package profile

import (
	"encoding/base64"
	"testing"
)

func TestParseVMessNumericPort(t *testing.T) {
	payload := `{"v":"2","ps":"test","add":"example.com","port":443,"id":"abc","aid":0,"net":"ws","tls":"tls","alpn":"h2,http/1.1"}`
	s, err := ParseLink("vmess://" + base64.StdEncoding.EncodeToString([]byte(payload)))
	if err != nil {
		t.Fatal(err)
	}
	if s.Port != 443 || s.Address != "example.com" || s.Extra["network"] != "ws" || s.Extra["alpn"] != "h2,http/1.1" {
		t.Fatalf("unexpected server: %+v", s)
	}
}

func TestParseShadowsocksLegacy(t *testing.T) {
	body := base64.StdEncoding.EncodeToString([]byte("aes-256-gcm:secret@1.2.3.4:8388"))
	s, err := ParseLink("ss://" + body + "#My%20Server")
	if err != nil {
		t.Fatal(err)
	}
	if s.Method != "aes-256-gcm" || s.Password != "secret" || s.Address != "1.2.3.4" || s.Port != 8388 || s.Name != "My Server" {
		t.Fatalf("unexpected server: %+v", s)
	}
}

func TestParseShadowsocksSIP002(t *testing.T) {
	creds := base64.RawURLEncoding.EncodeToString([]byte("chacha20-ietf-poly1305:pw"))
	s, err := ParseLink("ss://" + creds + "@host.example:443#n")
	if err != nil {
		t.Fatal(err)
	}
	if s.Method != "chacha20-ietf-poly1305" || s.Password != "pw" || s.Port != 443 {
		t.Fatalf("unexpected server: %+v", s)
	}
}

func TestParseVLESSQuery(t *testing.T) {
	s, err := ParseLink("vless://uuid@piruz.example:8443?type=ws&path=%2F&host=&security=tls&alpn=http%2F1.1%2Ch2#Server%201")
	if err != nil {
		t.Fatal(err)
	}
	if s.Name != "Server 1" || s.Extra["type"] != "ws" || s.Extra["security"] != "tls" || s.Port != 8443 {
		t.Fatalf("unexpected server: %+v", s)
	}
}
