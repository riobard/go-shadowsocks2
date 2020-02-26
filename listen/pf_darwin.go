package listen

import (
	"net"

	"github.com/riobard/go-shadowsocks2/pfutil"
)

func init() {
	listeners["redir"] = pfListen
}

type pfConn struct{ net.Conn }

func (c *pfConn) LocalAddr() net.Addr {
	tc, ok := c.Conn.(*net.TCPConn)
	if !ok {
		return nil
	}
	addr, err := pfutil.NatLookup(tc)
	if err != nil {
		return nil
	}
	return addr
}

type pfListener struct{ net.Listener }

func (l *pfListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return &pfConn{c}, nil
}

func pfListen(network, address string) (net.Listener, error) {
	l, err := net.Listen(network, address)
	if err != nil {
		return nil, err
	}
	return &pfListener{l}, nil
}
