package main

import (
	"net"
	"syscall"
	"time"

	"github.com/shadowsocks/go-shadowsocks2/nfutil"
	"github.com/shadowsocks/go-shadowsocks2/socks"
)

func getOrigDst(c net.Conn, ipv6 bool) (socks.Addr, error) {
	if tc, ok := c.(*net.TCPConn); ok {
		addr, err := nfutil.GetOrigDst(tc, ipv6)
		return socks.ParseAddr(addr.String()), err
	}
	panic("not a TCP connection")
}

// Listen on addr for netfilter redirected TCP connections
func redirLocal(addr, server string, shadow func(net.Conn) net.Conn) {
	logf("TCP redirect %s <-> %s", addr, server)
	tcpLocal(addr, server, shadow, func(c net.Conn) (socks.Addr, error) { return getOrigDst(c, false) })
}

// Listen on addr for netfilter redirected TCP IPv6 connections.
func redir6Local(addr, server string, shadow func(net.Conn) net.Conn) {
	logf("TCP6 redirect %s <-> %s", addr, server)
	tcpLocal(addr, server, shadow, func(c net.Conn) (socks.Addr, error) { return getOrigDst(c, true) })
}

func timedCork(c *net.TCPConn, d time.Duration) error {
	rc, err := c.SyscallConn()
	if err != nil {
		return err
	}
	rc.Control(func(fd uintptr) { err = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_TCP, syscall.TCP_CORK, 1) })
	if err != nil {
		return err
	}
	time.AfterFunc(d, func() {
		rc.Control(func(fd uintptr) { syscall.SetsockoptInt(int(fd), syscall.IPPROTO_TCP, syscall.TCP_CORK, 0) })
	})
	return nil
}
