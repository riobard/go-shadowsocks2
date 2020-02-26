package listen

import (
	"errors"
	"net"
)

var listeners = make(map[string]func(network, address string) (net.Listener, error))

var ErrUnsupported = errors.New("unsupported")

func Listen(kind, network, address string) (net.Listener, error) {
	f, ok := listeners[kind]
	if ok {
		return f(network, address)
	}
	return nil, ErrUnsupported
}

type strAddr string

func (a strAddr) Network() string { return "tcp" }
func (a strAddr) String() string  { return string(a) }

type targetConn struct {
	net.Conn
	target net.Addr
}

func (c *targetConn) LocalAddr() net.Addr { return c.target }

type targetListener struct {
	net.Listener
	target net.Addr
}

func ListenTo(network, address, target string) (net.Listener, error) {
	l, err := net.Listen(network, address)
	if err != nil {
		return nil, err
	}
	return &targetListener{l, strAddr(target)}, err
}

func (l *targetListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return &targetConn{c, l.target}, nil
}
