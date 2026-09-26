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

func TestParseClashYAML(t *testing.T) {
	body := `proxies:
  - name: "DE ws"
    type: vless
    server: de.example.com
    port: 443
    uuid: abc
    network: ws
    tls: true
    servername: cdn.example.com
    ws-opts:
      path: /ray
      headers:
        Host: cdn.example.com
  - name: SS
    type: ss
    server: 1.2.3.4
    port: 8388
    cipher: aes-128-gcm
    password: pw
  - name: hy
    type: hysteria2
    server: x
    port: 1
`
	servers, _, errs := ParseSubscription(body)
	if len(servers) != 2 || len(errs) != 1 {
		t.Fatalf("got %d servers, %d errs: %+v %v", len(servers), len(errs), servers, errs)
	}
	v := servers[0]
	if v.Protocol != "vless" || v.Port != 443 || v.Extra["type"] != "ws" || v.Extra["security"] != "tls" || v.Extra["path"] != "/ray" || v.Extra["host"] != "cdn.example.com" {
		t.Fatalf("bad vless: %+v", v)
	}
	if servers[1].Method != "aes-128-gcm" || servers[1].Protocol != "shadowsocks" {
		t.Fatalf("bad ss: %+v", servers[1])
	}
}

func TestParseSingBoxJSON(t *testing.T) {
	body := `{"outbounds":[
	  {"type":"vless","tag":"R","server":"r.example","server_port":443,"uuid":"u","flow":"xtls-rprx-vision",
	   "tls":{"enabled":true,"server_name":"www.microsoft.com","utls":{"enabled":true,"fingerprint":"chrome"},
	          "reality":{"enabled":true,"public_key":"PBK","short_id":"ab"}}},
	  {"type":"direct","tag":"direct"}]}`
	servers, _, errs := ParseSubscription(body)
	if len(servers) != 1 || len(errs) != 0 {
		t.Fatalf("got %+v %v", servers, errs)
	}
	s := servers[0]
	if s.Extra["security"] != "reality" || s.Extra["pbk"] != "PBK" || s.Extra["sni"] != "www.microsoft.com" || s.Extra["flow"] != "xtls-rprx-vision" {
		t.Fatalf("bad: %+v", s)
	}
}

func TestShareLinkRoundTrip(t *testing.T) {
	links := []string{
		"vless://uuid@piruz.example:8443?path=%2F&security=tls&type=ws#Server%201",
		"trojan://pw@t.example:443?sni=t.example#T",
		"ss://" + base64.RawURLEncoding.EncodeToString([]byte("aes-256-gcm:secret")) + "@1.2.3.4:8388#S",
	}
	vmess := `{"ps":"V","add":"v.example","port":"443","id":"id1","net":"ws","tls":"tls","path":"/p","host":"h"}`
	links = append(links, "vmess://"+base64.StdEncoding.EncodeToString([]byte(vmess)))

	for _, link := range links {
		a, err := ParseLink(link)
		if err != nil {
			t.Fatal(err)
		}
		a.Extra = withInternal(a.Extra)
		shared, err := ShareLink(a)
		if err != nil {
			t.Fatal(err)
		}
		b, err := ParseLink(shared)
		if err != nil {
			t.Fatalf("%s: %v", shared, err)
		}
		if a.Name != b.Name || a.Address != b.Address || a.Port != b.Port || a.UUID != b.UUID || a.Password != b.Password || a.Method != b.Method {
			t.Fatalf("round trip mismatch:\n%+v\n%+v\n%s", a, b, shared)
		}
		if b.Extra["subURL"] != "" {
			t.Fatalf("internal key leaked into share link: %s", shared)
		}
	}
}

func withInternal(e map[string]string) map[string]string {
	if e == nil {
		e = map[string]string{}
	}
	e["subURL"] = "https://secret.example/sub"
	return e
}
