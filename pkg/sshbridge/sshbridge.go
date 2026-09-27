// Package sshbridge lets xray-core use an SSH server as its proxy.
//
// xray-core has no SSH outbound, so Kite runs a tiny SOCKS5 server on
// 127.0.0.1 that forwards each CONNECT through one SSH session
// (direct-tcpip channels, like `ssh -D`). The generated xray config then
// points its proxy outbound at that SOCKS port. SSH carries TCP only: UDP
// ASSOCIATE is refused, and DNS is resolved over TCP instead (see xrayconf).
//
// SSH servers use protocol "ssh" with Password and these extra keys:
// user, pk (private key, PEM/OpenSSH), pp (key passphrase) and hk (host key:
// "SHA256:..." fingerprint or an authorized_keys line; empty = don't verify).
package sshbridge

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/freeb5d/kite/pkg/profile"
)

const dialTimeout = 10 * time.Second

// Bridge is a running local SOCKS5 -> SSH forwarder.
type Bridge struct {
	Port int

	cfg  *ssh.ClientConfig
	addr string
	ln   net.Listener

	mu     sync.Mutex
	client *ssh.Client
	closed bool
	done   chan struct{}
}

// Start connects to the SSH server (failing fast on bad credentials) and
// starts the local SOCKS5 listener.
func Start(s profile.Server) (*Bridge, error) {
	cfg, err := clientConfig(s)
	if err != nil {
		return nil, err
	}
	b := &Bridge{
		cfg:  cfg,
		addr: net.JoinHostPort(s.Address, strconv.Itoa(s.Port)),
		done: make(chan struct{}),
	}
	if _, err := b.sshClient(); err != nil {
		return nil, err
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Close()
		return nil, err
	}
	b.ln = ln
	b.Port = ln.Addr().(*net.TCPAddr).Port
	go b.serve()
	go b.keepalive()
	return b, nil
}

// Close stops the listener and the SSH session. Safe on nil.
func (b *Bridge) Close() {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return
	}
	b.closed = true
	close(b.done)
	if b.ln != nil {
		_ = b.ln.Close()
	}
	if b.client != nil {
		_ = b.client.Close()
		b.client = nil
	}
}

func clientConfig(s profile.Server) (*ssh.ClientConfig, error) {
	e := s.Extra
	user := e["user"]
	if user == "" {
		user = "root"
	}
	var auths []ssh.AuthMethod
	if pk := strings.TrimSpace(e["pk"]); pk != "" {
		var signer ssh.Signer
		var err error
		if e["pp"] != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(pk), []byte(e["pp"]))
		} else {
			signer, err = ssh.ParsePrivateKey([]byte(pk))
		}
		if err != nil {
			return nil, fmt.Errorf("ssh private key: %w", err)
		}
		auths = append(auths, ssh.PublicKeys(signer))
	}
	if s.Password != "" {
		pw := s.Password
		auths = append(auths, ssh.Password(pw), ssh.KeyboardInteractive(
			func(_, _ string, questions []string, _ []bool) ([]string, error) {
				answers := make([]string, len(questions))
				for i := range answers {
					answers[i] = pw
				}
				return answers, nil
			}))
	}
	if len(auths) == 0 {
		return nil, errors.New("ssh: set a password or a private key")
	}
	return &ssh.ClientConfig{
		User:            user,
		Auth:            auths,
		HostKeyCallback: hostKeyCallback(e["hk"]),
		Timeout:         dialTimeout,
	}, nil
}

func hostKeyCallback(want string) ssh.HostKeyCallback {
	want = strings.TrimSpace(want)
	if want == "" {
		return ssh.InsecureIgnoreHostKey()
	}
	return func(_ string, _ net.Addr, key ssh.PublicKey) error {
		if ssh.FingerprintSHA256(key) == want || strings.TrimPrefix(ssh.FingerprintSHA256(key), "SHA256:") == want {
			return nil
		}
		if pub, _, _, _, err := ssh.ParseAuthorizedKey([]byte(want)); err == nil && bytes.Equal(pub.Marshal(), key.Marshal()) {
			return nil
		}
		return fmt.Errorf("ssh: host key mismatch (server has %s)", ssh.FingerprintSHA256(key))
	}
}

// sshClient returns the live SSH session, reconnecting if it dropped.
func (b *Bridge) sshClient() (*ssh.Client, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.closed {
		return nil, net.ErrClosed
	}
	if b.client != nil {
		return b.client, nil
	}
	c, err := ssh.Dial("tcp", b.addr, b.cfg)
	if err != nil {
		return nil, fmt.Errorf("ssh: %w", err)
	}
	b.client = c
	go func() {
		_ = c.Wait()
		b.mu.Lock()
		if b.client == c {
			b.client = nil
		}
		b.mu.Unlock()
	}()
	return c, nil
}

func (b *Bridge) keepalive() {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-b.done:
			return
		case <-t.C:
			b.mu.Lock()
			c := b.client
			b.mu.Unlock()
			if c != nil {
				if _, _, err := c.SendRequest("keepalive@openssh.com", true, nil); err != nil {
					_ = c.Close()
				}
			}
		}
	}
}

func (b *Bridge) dial(target string) (net.Conn, error) {
	c, err := b.sshClient()
	if err != nil {
		return nil, err
	}
	conn, err := c.Dial("tcp", target)
	if err == nil {
		return conn, nil
	}
	// The session may have died silently; retry once on a fresh one.
	b.mu.Lock()
	if b.client == c {
		_ = c.Close()
		b.client = nil
	}
	b.mu.Unlock()
	if c, err = b.sshClient(); err != nil {
		return nil, err
	}
	return c.Dial("tcp", target)
}

func (b *Bridge) serve() {
	for {
		conn, err := b.ln.Accept()
		if err != nil {
			return
		}
		go b.handle(conn)
	}
}

// handle speaks just enough SOCKS5 for xray-core's socks outbound:
// no authentication, CONNECT only.
func (b *Bridge) handle(conn net.Conn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(dialTimeout * 2))
	buf := make([]byte, 262)
	if _, err := io.ReadFull(conn, buf[:2]); err != nil || buf[0] != 5 {
		return
	}
	if _, err := io.ReadFull(conn, buf[:buf[1]]); err != nil {
		return
	}
	if _, err := conn.Write([]byte{5, 0}); err != nil {
		return
	}
	if _, err := io.ReadFull(conn, buf[:4]); err != nil {
		return
	}
	cmd, atyp := buf[1], buf[3]
	var host string
	switch atyp {
	case 1:
		if _, err := io.ReadFull(conn, buf[:4]); err != nil {
			return
		}
		host = net.IP(buf[:4]).String()
	case 4:
		if _, err := io.ReadFull(conn, buf[:16]); err != nil {
			return
		}
		host = net.IP(buf[:16]).String()
	case 3:
		if _, err := io.ReadFull(conn, buf[:1]); err != nil {
			return
		}
		n := int(buf[0])
		if _, err := io.ReadFull(conn, buf[:n]); err != nil {
			return
		}
		host = string(buf[:n])
	default:
		reply(conn, 8)
		return
	}
	if _, err := io.ReadFull(conn, buf[:2]); err != nil {
		return
	}
	port := binary.BigEndian.Uint16(buf[:2])
	if cmd != 1 {
		reply(conn, 7) // command not supported (UDP can't go over SSH)
		return
	}
	remote, err := b.dial(net.JoinHostPort(host, strconv.Itoa(int(port))))
	if err != nil {
		reply(conn, 5)
		return
	}
	defer remote.Close()
	reply(conn, 0)
	_ = conn.SetDeadline(time.Time{})

	go func() {
		_, _ = io.Copy(remote, conn)
		_ = remote.Close()
	}()
	_, _ = io.Copy(conn, remote)
}

func reply(conn net.Conn, code byte) {
	_, _ = conn.Write([]byte{5, code, 0, 1, 0, 0, 0, 0, 0, 0})
}
