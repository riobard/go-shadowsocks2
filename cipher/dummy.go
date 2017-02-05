package cipher

import (
	"net"

	"github.com/riobard/go-shadowsocks2/core"
)

// Dummy ciphers (no encryption)

func dummyStream() core.StreamConnCipher {
	return func(c net.Conn) net.Conn { return c }
}
func dummyPacket() core.PacketConnCipher {
	return func(c net.PacketConn) net.PacketConn { return c }
}
