package core

import "net"

type PacketConnCipher func(net.PacketConn) net.PacketConn

func ListenPacket(network, address string, ciph PacketConnCipher) (net.PacketConn, error) {
	c, err := net.ListenPacket(network, address)
	return ciph(c), err
}
