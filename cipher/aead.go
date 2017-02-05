package cipher

import (
	"crypto/aes"
	"crypto/cipher"
	"net"

	"github.com/riobard/go-shadowsocks2/core"
	"github.com/riobard/go-shadowsocks2/shadowaead"
)

// AEAD ciphers

func aeadStream(aead cipher.AEAD) core.StreamConnCipher {
	return func(c net.Conn) net.Conn { return shadowaead.NewConn(c, aead) }
}

func aeadPacket(aead cipher.AEAD) core.PacketConnCipher {
	return func(c net.PacketConn) net.PacketConn { return shadowaead.NewPacketConn(c, aead) }
}

// AES-GCM with standard 12-byte nonce
func aesGCM(key []byte) (cipher.AEAD, error) {
	blk, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(blk)
}

// AES-GCM with 16-byte nonce for better collision avoidance
func aesGCM16(key []byte) (cipher.AEAD, error) {
	blk, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCMWithNonceSize(blk, 16)
}
