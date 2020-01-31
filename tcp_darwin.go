package main

import (
	"net"

	"github.com/riobard/go-shadowsocks2/pfutil"
	"github.com/riobard/go-shadowsocks2/socks"
)

func redirLocal(addr string, d Dialer)  { tcpLocal(addr, d, natLookup) }
func redir6Local(addr string, d Dialer) { panic("TCP6 redirect not supported") }
func tproxyTCP(addr string, d Dialer)   { panic("TPROXY TCP not supported") }

func natLookup(c net.Conn) (socks.Addr, error) {
	if tc, ok := c.(*net.TCPConn); ok {
		addr, err := pfutil.NatLookup(tc)
		return socks.ParseAddr(addr.String()), err
	}
	panic("not TCP connection")
}
