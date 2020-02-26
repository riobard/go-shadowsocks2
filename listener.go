package main

import (
	"errors"
	"net"

	"github.com/riobard/go-shadowsocks2/socks"
)

var listeners map[string]func(network, address string) (net.Listener, error)

func init() {
	listeners = make(map[string]func(network, address string) (net.Listener, error))
}

func listen(kind, network, address string) (net.Listener, error) {
	f, ok := listeners[kind]
	if ok {
		return f(network, address)
	}
	return nil, errors.New("unsupported listener " + kind)
}

type strAddr string

func (a strAddr) Network() string { return "tcp" }
func (a strAddr) String() string  { return string(a) }

type tunConn struct {
	net.Conn
	target net.Addr
}

func (c *tunConn) LocalAddr() net.Addr { return c.target }

type tunListener struct {
	net.Listener
	target net.Addr
}

func tunListen(network, address, target string) (net.Listener, error) {
	l, err := net.Listen(network, address)
	if err != nil {
		return nil, err
	}
	return &tunListener{l, strAddr(target)}, err
}

func (l *tunListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return &tunConn{c, l.target}, nil
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
