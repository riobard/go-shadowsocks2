package main

import (
	"errors"
	"net"
	"syscall"
	"unsafe"

	"github.com/Dreamacro/go-shadowsocks2/socks"
)

const (
	SO_ORIGINAL_DST      = 80 // from linux/include/uapi/linux/netfilter_ipv4.h
	IP6T_SO_ORIGINAL_DST = 80 // from linux/include/uapi/linux/netfilter_ipv6/ip6_tables.h
)

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

// Get the original destination of a TCP connection.
func getOrigDst(conn net.Conn, ipv6 bool) (socks.Addr, error) {
	c, ok := conn.(*net.TCPConn)
	if !ok {
		return nil, errors.New("only work with TCP connection")
	}

	rc, err := c.SyscallConn()
	if err != nil {
		return nil, err
	}

	var addr socks.Addr

	rc.Control(func(fd uintptr) {
		if ipv6 {
			addr, err = ipv6_getorigdst(fd)
		} else {
			addr, err = getorigdst(fd)
		}
	})

	return addr, err
}

// Call getorigdst() from linux/net/ipv4/netfilter/nf_conntrack_l3proto_ipv4.c
func getorigdst(fd uintptr) (socks.Addr, error) {
	raw := syscall.RawSockaddrInet4{}
	siz := unsafe.Sizeof(raw)
	if err := socketcall(GETSOCKOPT, fd, syscall.IPPROTO_IP, SO_ORIGINAL_DST, uintptr(unsafe.Pointer(&raw)), uintptr(unsafe.Pointer(&siz)), 0); err != nil {
		return nil, err
	}

	addr := make([]byte, 1+net.IPv4len+2)
	addr[0] = socks.AtypIPv4
	copy(addr[1:1+net.IPv4len], raw.Addr[:])
	port := (*[2]byte)(unsafe.Pointer(&raw.Port)) // big-endian
	addr[1+net.IPv4len], addr[1+net.IPv4len+1] = port[0], port[1]
	return addr, nil
}

// Call ipv6_getorigdst() from linux/net/ipv6/netfilter/nf_conntrack_l3proto_ipv6.c
// NOTE: I haven't tried yet but it should work since Linux 3.8.
func ipv6_getorigdst(fd uintptr) (socks.Addr, error) {
	raw := syscall.RawSockaddrInet6{}
	siz := unsafe.Sizeof(raw)
	if err := socketcall(GETSOCKOPT, fd, syscall.IPPROTO_IPV6, IP6T_SO_ORIGINAL_DST, uintptr(unsafe.Pointer(&raw)), uintptr(unsafe.Pointer(&siz)), 0); err != nil {
		return nil, err
	}

	addr := make([]byte, 1+net.IPv6len+2)
	addr[0] = socks.AtypIPv6
	copy(addr[1:1+net.IPv6len], raw.Addr[:])
	port := (*[2]byte)(unsafe.Pointer(&raw.Port)) // big-endian
	addr[1+net.IPv6len], addr[1+net.IPv6len+1] = port[0], port[1]
	return addr, nil
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
