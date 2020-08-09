package main

import (
	"net"
	"syscall"
	"time"

	"github.com/shadowsocks/go-shadowsocks2/pfutil"
	"github.com/shadowsocks/go-shadowsocks2/socks"
)

func redirLocal(addr, server string, shadow func(net.Conn) net.Conn) {
	tcpLocal(addr, server, shadow, natLookup)
}

func redir6Local(addr, server string, shadow func(net.Conn) net.Conn) {
	panic("TCP6 redirect not supported")
}

func natLookup(c net.Conn) (socks.Addr, error) {
	if tc, ok := c.(*net.TCPConn); ok {
		addr, err := pfutil.NatLookup(tc)
		return socks.ParseAddr(addr.String()), err
	}
	panic("not TCP connection")
}

func timedCork(c *net.TCPConn, d time.Duration) error {
	rc, err := c.SyscallConn()
	if err != nil {
		return err
	}
	rc.Control(func(fd uintptr) { err = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_TCP, syscall.TCP_NOPUSH, 1) })
	if err != nil {
		return err
	}
	time.AfterFunc(d, func() {
		rc.Control(func(fd uintptr) { syscall.SetsockoptInt(int(fd), syscall.IPPROTO_TCP, syscall.TCP_NOPUSH, 0) })
	})
	return nil
}
