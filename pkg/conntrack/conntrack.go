// Package conntrack closes xray-core's live outbound connections on demand.
//
// Closing an xray-core instance stops its listeners, but connections that
// are already open keep flowing through the old server until they go idle
// (minutes). After switching servers, a browser's kept-alive connections
// would still exit through the previous location. Install wraps xray's
// system dialer so every outbound socket is recorded, and CloseAll drops
// them all when a connection stops.
package conntrack

import (
	"context"
	gonet "net"
	"sync"

	"github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/transport/internet"
)

var (
	once  sync.Once
	mu    sync.Mutex
	conns = map[*conn]struct{}{}
)

// Install routes xray-core's outbound dialing through the tracker. Safe to
// call more than once.
func Install() {
	once.Do(func() {
		internet.UseAlternativeSystemDialer(&dialer{inner: &internet.DefaultSystemDialer{}})
	})
}

// CloseAll closes every outbound connection opened so far.
func CloseAll() {
	mu.Lock()
	list := make([]*conn, 0, len(conns))
	for c := range conns {
		list = append(list, c)
	}
	conns = map[*conn]struct{}{}
	mu.Unlock()
	for _, c := range list {
		_ = c.Conn.Close()
	}
}

// Count reports how many tracked connections are open (for tests).
func Count() int {
	mu.Lock()
	defer mu.Unlock()
	return len(conns)
}

type dialer struct{ inner internet.SystemDialer }

func (d *dialer) Dial(ctx context.Context, src net.Address, dest net.Destination, sockopt *internet.SocketConfig) (net.Conn, error) {
	c, err := d.inner.Dial(ctx, src, dest, sockopt)
	if err != nil {
		return nil, err
	}
	t := &conn{Conn: c}
	mu.Lock()
	conns[t] = struct{}{}
	mu.Unlock()
	if pc, ok := c.(gonet.PacketConn); ok {
		return &packetConn{conn: t, pc: pc}, nil
	}
	return t, nil
}

func (d *dialer) DestIpAddress() net.IP { return d.inner.DestIpAddress() }

type conn struct {
	gonet.Conn
	closeOnce sync.Once
}

// untrack forgets c without closing it. Called as soon as the connection
// fails or ends: xray doesn't always Close the exact wrapper it was handed
// (it may drop it once the peer is gone), and a forgotten entry would keep
// the connection's memory alive for good -- slowly growing until Android
// kills the app.
func (c *conn) untrack() {
	mu.Lock()
	delete(conns, c)
	mu.Unlock()
}

func (c *conn) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)
	if err != nil {
		c.untrack()
	}
	return n, err
}

func (c *conn) Write(b []byte) (int, error) {
	n, err := c.Conn.Write(b)
	if err != nil {
		c.untrack()
	}
	return n, err
}

func (c *conn) Close() error {
	var err error
	c.closeOnce.Do(func() {
		c.untrack()
		err = c.Conn.Close()
	})
	return err
}

// packetConn keeps the PacketConn methods of UDP sockets (QUIC needs them).
type packetConn struct {
	*conn
	pc gonet.PacketConn
}

func (p *packetConn) ReadFrom(b []byte) (int, gonet.Addr, error) {
	n, a, err := p.pc.ReadFrom(b)
	if err != nil {
		p.untrack()
	}
	return n, a, err
}

func (p *packetConn) WriteTo(b []byte, a gonet.Addr) (int, error) {
	n, err := p.pc.WriteTo(b, a)
	if err != nil {
		p.untrack()
	}
	return n, err
}
