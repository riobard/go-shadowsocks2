package listen

import (
	"context"
	"net"
	"syscall"

	"github.com/riobard/go-shadowsocks2/nfutil"
)

func init() {
	listeners["redir"] = nfListen
	listeners["tproxy"] = tproxyListen
}

type nfConn struct{ net.Conn }

func (c *nfConn) LocalAddr() net.Addr {
	tc, ok := c.Conn.(*net.TCPConn)
	if !ok {
		return nil
	}
	addr, err := nfutil.GetOrigDst(tc, false) // TODO: detect if c is ipv6
	if err != nil {
		return nil
	}
	return addr
}

type nfListener struct{ net.Listener }

func (l *nfListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return &nfConn{c}, nil
}

func nfListen(network, address string) (net.Listener, error) {
	l, err := net.Listen(network, address)
	if err != nil {
		return nil, err
	}
	return &nfListener{l}, nil
}

func tproxyListen(network, address string) (net.Listener, error) {
	lcfg := net.ListenConfig{Control: func(network, address string, rc syscall.RawConn) error {
		var err1, err2 error
		err2 = rc.Control(func(fd uintptr) { err1 = syscall.SetsockoptInt(int(fd), syscall.SOL_IP, syscall.IP_TRANSPARENT, 1) })
		if err1 != nil {
			return err1
		}
		return err2
	}}
	return lcfg.Listen(context.Background(), network, address)
}
