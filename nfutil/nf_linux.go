package nfutil

import (
	"net"
	"syscall"
	"unsafe"
)

// Get the original destination of a TCP connection redirected by Netfilter.
func GetOrigDst(c *net.TCPConn, ipv6 bool) (*net.TCPAddr, error) {
	rc, err := c.SyscallConn()
	if err != nil {
		return nil, err
	}
	var addr *net.TCPAddr
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
func getorigdst(fd uintptr) (*net.TCPAddr, error) {
	const _SO_ORIGINAL_DST = 80 // from linux/include/uapi/linux/netfilter_ipv4.h
	var raw syscall.RawSockaddrInet4
	siz := unsafe.Sizeof(raw)
	if err := socketcall(GETSOCKOPT, fd, syscall.IPPROTO_IP, _SO_ORIGINAL_DST, uintptr(unsafe.Pointer(&raw)), uintptr(unsafe.Pointer(&siz)), 0); err != nil {
		return nil, err
	}
	var addr net.TCPAddr
	addr.IP = raw.Addr[:]
	port := (*[2]byte)(unsafe.Pointer(&raw.Port)) // raw.Port is big-endian
	addr.Port = int(port[0])<<8 | int(port[1])
	return &addr, nil
}

// Call ipv6_getorigdst() from linux/net/ipv6/netfilter/nf_conntrack_l3proto_ipv6.c
// NOTE: I haven't tried yet but it should work since Linux 3.8.
func ipv6_getorigdst(fd uintptr) (*net.TCPAddr, error) {
	const _IP6T_SO_ORIGINAL_DST = 80 // from linux/include/uapi/linux/netfilter_ipv6/ip6_tables.h
	var raw syscall.RawSockaddrInet6
	siz := unsafe.Sizeof(raw)
	if err := socketcall(GETSOCKOPT, fd, syscall.IPPROTO_IPV6, _IP6T_SO_ORIGINAL_DST, uintptr(unsafe.Pointer(&raw)), uintptr(unsafe.Pointer(&siz)), 0); err != nil {
		return nil, err
	}
	var addr net.TCPAddr
	addr.IP = raw.Addr[:]
	port := (*[2]byte)(unsafe.Pointer(&raw.Port)) // raw.Port is big-endian
	addr.Port = int(port[0])<<8 | int(port[1])
	return &addr, nil
}
