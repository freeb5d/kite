package sshbridge

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"io"
	"net"
	"strconv"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/freeb5d/kite/pkg/profile"
)

// testSSHServer accepts password "pw" and serves direct-tcpip channels.
func testSSHServer(t *testing.T) (addr string, fingerprint string) {
	_, key, _ := ed25519.GenerateKey(rand.Reader)
	signer, err := ssh.NewSignerFromKey(key)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &ssh.ServerConfig{
		PasswordCallback: func(_ ssh.ConnMetadata, pw []byte) (*ssh.Permissions, error) {
			if string(pw) == "pw" {
				return nil, nil
			}
			return nil, io.EOF
		},
	}
	cfg.AddHostKey(signer)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() {
				_, chans, reqs, err := ssh.NewServerConn(c, cfg)
				if err != nil {
					return
				}
				go ssh.DiscardRequests(reqs)
				for nc := range chans {
					var p struct {
						Host     string
						Port     uint32
						OrigHost string
						OrigPort uint32
					}
					if nc.ChannelType() != "direct-tcpip" || ssh.Unmarshal(nc.ExtraData(), &p) != nil {
						nc.Reject(ssh.UnknownChannelType, "no")
						continue
					}
					target, err := net.Dial("tcp", net.JoinHostPort(p.Host, strconv.Itoa(int(p.Port))))
					if err != nil {
						nc.Reject(ssh.ConnectionFailed, err.Error())
						continue
					}
					ch, r, _ := nc.Accept()
					go ssh.DiscardRequests(r)
					go func() { io.Copy(ch, target); ch.Close() }()
					go func() { io.Copy(target, ch); target.Close() }()
				}
			}()
		}
	}()
	return ln.Addr().String(), ssh.FingerprintSHA256(signer.PublicKey())
}

func echoServer(t *testing.T) *net.TCPAddr {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() { io.Copy(c, c); c.Close() }()
		}
	}()
	return ln.Addr().(*net.TCPAddr)
}

func server(addr, pw, hk string) profile.Server {
	host, port, _ := net.SplitHostPort(addr)
	p, _ := strconv.Atoi(port)
	return profile.Server{Protocol: "ssh", Address: host, Port: p, Password: pw, Extra: map[string]string{"user": "u", "hk": hk}}
}

func TestBridgeForwardsThroughSSH(t *testing.T) {
	addr, fp := testSSHServer(t)
	echo := echoServer(t)
	b, err := Start(server(addr, "pw", fp))
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()

	c, err := net.Dial("tcp", "127.0.0.1:"+strconv.Itoa(b.Port))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.Write([]byte{5, 1, 0})
	resp := make([]byte, 10)
	if _, err := io.ReadFull(c, resp[:2]); err != nil || resp[1] != 0 {
		t.Fatalf("greeting %v %v", resp[:2], err)
	}
	req := []byte{5, 1, 0, 1}
	req = append(req, echo.IP.To4()...)
	req = binary.BigEndian.AppendUint16(req, uint16(echo.Port))
	c.Write(req)
	if _, err := io.ReadFull(c, resp); err != nil || resp[1] != 0 {
		t.Fatalf("connect reply %v %v", resp, err)
	}
	c.Write([]byte("hello"))
	got := make([]byte, 5)
	if _, err := io.ReadFull(c, got); err != nil || string(got) != "hello" {
		t.Fatalf("echo %q %v", got, err)
	}
}

func TestBridgeRejectsBadCredentialsAndHostKey(t *testing.T) {
	addr, fp := testSSHServer(t)
	if _, err := Start(server(addr, "wrong", "")); err == nil {
		t.Fatal("expected auth failure")
	}
	if _, err := Start(server(addr, "pw", "SHA256:not-it")); err == nil {
		t.Fatal("expected host key mismatch")
	}
	b, err := Start(server(addr, "pw", fp))
	if err != nil {
		t.Fatal(err)
	}
	b.Close()
}
