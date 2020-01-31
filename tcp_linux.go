package main

import (
	"net"
	"syscall"

	"github.com/riobard/go-shadowsocks2/nfutil"
	"github.com/riobard/go-shadowsocks2/socks"
)

func getOrigDst(c net.Conn, ipv6 bool) (socks.Addr, error) {
	if tc, ok := c.(*net.TCPConn); ok {
		addr, err := nfutil.GetOrigDst(tc, ipv6)
		return socks.ParseAddr(addr.String()), err
	}
	panic("not a TCP connection")
}

// Listen on addr for netfilter redirected TCP connections
func redirLocal(addr string, d Dialer) {
	// logf("TCP redirect %s <-> %s", addr, server)
	tcpLocal(addr, d, func(c net.Conn) (socks.Addr, error) { return getOrigDst(c, false) })
}

// Listen on addr for netfilter redirected TCP IPv6 connections.
func redir6Local(addr string, d Dialer) {
	// logf("TCP6 redirect %s <-> %s", addr, server)
	tcpLocal(addr, d, func(c net.Conn) (socks.Addr, error) { return getOrigDst(c, true) })
}

func tproxyTCP(addr string, d Dialer) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	rc, err := l.(*net.TCPListener).SyscallConn()
	if err != nil {
		return err
	}
	rc.Control(func(fd uintptr) { err = syscall.SetsockoptInt(int(fd), syscall.SOL_IP, syscall.IP_TRANSPARENT, 1) })
	if err != nil {
		return err
	}
	logf("TPROXY on tcp://%v", addr)
	for {
		c, err := l.Accept()
		if err != nil {
			return err
		}
		go func() {
			defer c.Close()
			tcpKeepAlive(c)
			rc, err := d.Dial("tcp", c.LocalAddr().String())
			if err != nil {
				logf("failed to connect: %v", err)
				return
			}
			defer rc.Close()
			tcpKeepAlive(rc)
			logf("TPROXY TCP %s <--[%s]--> %s", c.RemoteAddr(), rc.RemoteAddr(), c.LocalAddr())
			if err = relay(rc, c); err != nil {
				if err, ok := err.(net.Error); ok && err.Timeout() {
					return // ignore i/o timeout
				}
				logf("relay error: %v", err)
			}
		}()
	}
}
