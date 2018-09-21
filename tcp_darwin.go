package main

import (
	"net"
	"syscall"
	"unsafe"

	"github.com/Dreamacro/go-shadowsocks2/socks"
)

func redirLocal(addr string, d Dialer)  { tcpLocal(addr, d, pfNatLookup) }
func redir6Local(addr string, d Dialer) { panic("TCP6 redirect not supported") }
func tproxyTCP(addr string, d Dialer)   { panic("TPROXY TCP not supported") }

func pfNatLookup(c net.Conn) (socks.Addr, error) {
	const (
		PF_INOUT     = 0
		PF_IN        = 1
		PF_OUT       = 2
		IOC_OUT      = 0x40000000
		IOC_IN       = 0x80000000
		IOC_INOUT    = IOC_IN | IOC_OUT
		IOCPARM_MASK = 0x1FFF
		LEN          = 4*16 + 4*4 + 4*1
		// #define	_IOC(inout,group,num,len) (inout | ((len & IOCPARM_MASK) << 16) | ((group) << 8) | (num))
		// #define	_IOWR(g,n,t)	_IOC(IOC_INOUT,	(g), (n), sizeof(t))
		// #define DIOCNATLOOK		_IOWR('D', 23, struct pfioc_natlook)
		DIOCNATLOOK = IOC_INOUT | ((LEN & IOCPARM_MASK) << 16) | ('D' << 8) | 23
	)

	fd, err := syscall.Open("/dev/pf", 0, syscall.O_RDONLY)
	if err != nil {
		return nil, err
	}
	defer syscall.Close(fd)

	nl := struct { // struct pfioc_natlook
		saddr, daddr, rsaddr, rdaddr       [16]byte
		sxport, dxport, rsxport, rdxport   [4]byte
		af, proto, protoVariant, direction uint8
	}{
		af:        syscall.AF_INET,
		proto:     syscall.IPPROTO_TCP,
		direction: PF_OUT,
	}
	saddr := c.RemoteAddr().(*net.TCPAddr)
	daddr := c.LocalAddr().(*net.TCPAddr)
	copy(nl.saddr[:], saddr.IP)
	copy(nl.daddr[:], daddr.IP)
	nl.sxport[0], nl.sxport[1] = byte(saddr.Port>>8), byte(saddr.Port)
	nl.dxport[0], nl.dxport[1] = byte(daddr.Port>>8), byte(daddr.Port)

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), DIOCNATLOOK, uintptr(unsafe.Pointer(&nl))); errno != 0 {
		return nil, errno
	}

	addr := make([]byte, 1+net.IPv4len+2)
	addr[0] = socks.AtypIPv4
	copy(addr[1:1+net.IPv4len], nl.rdaddr[:4])
	copy(addr[1+net.IPv4len:], nl.rdxport[:2])
	return addr, nil
}
