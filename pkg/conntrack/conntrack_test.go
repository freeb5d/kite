package conntrack

import (
	"context"
	"io"
	gonet "net"
	"testing"

	"github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/transport/internet"
)

func TestCloseAllDropsLiveConnections(t *testing.T) {
	Install()
	ln, err := gonet.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go io.Copy(io.Discard, c)
		}
	}()
	port := ln.Addr().(*gonet.TCPAddr).Port
	c, err := internet.DialSystem(context.Background(), net.TCPDestination(net.LocalHostIP, net.Port(port)), nil)
	if err != nil {
		t.Fatal(err)
	}
	if Count() != 1 {
		t.Fatalf("tracked %d, want 1", Count())
	}
	CloseAll()
	if Count() != 0 {
		t.Fatalf("tracked %d after CloseAll", Count())
	}
	if _, err := c.Write([]byte("x")); err == nil {
		t.Fatal("write on a closed connection succeeded")
	}
}
