package listen

import (
	"net"

	"github.com/riobard/go-shadowsocks2/socks"
)

func init() {
	listeners["socks"] = socksListen
}

type socksConn struct{ net.Conn }

func (sc socksConn) LocalAddr() net.Addr {
	addr, err := socks.Handshake(sc.Conn)
	if err != nil {
		return nil
	}
	return strAddr(addr.String())
}

type socksListener struct{ net.Listener }

func (l *socksListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return socksConn{c}, nil
}

func socksListen(network, addr string) (net.Listener, error) {
	l, err := net.Listen(network, addr)
	if err != nil {
		return nil, err
	}
	return &socksListener{l}, nil
}
