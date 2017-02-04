package shadowaead

import (
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
	"net"
)

// ErrShortPacket means that the packet is too short for a valid encrypted packet.
var ErrShortPacket = errors.New("shadow: short packet")

// Pack encrypts plaintext using aead with a randomly generated nonce and
// returns a slice of dst containing the encrypted packet and any error occurred.
// Ensure len(dst) >= aead.NonceSize() + len(plaintext) + aead.Overhead().
func Pack(dst, plaintext []byte, aead cipher.AEAD) ([]byte, error) {
	nsiz := aead.NonceSize()
	if len(dst) < nsiz+len(plaintext)+aead.Overhead() {
		return nil, io.ErrShortBuffer
	}

	nonce := dst[:nsiz]
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	b := aead.Seal(dst[nsiz:nsiz], nonce, plaintext, nil)
	return dst[:nsiz+len(b)], nil
}

// Unpack decrypts pkt using aead and returns a slice of dst containing the decrypted payload and any error occurred.
// Ensure len(dst) >= len(pkt) - aead.NonceSize() - aead.Overhead().
func Unpack(dst, pkt []byte, aead cipher.AEAD) ([]byte, error) {
	nsiz := aead.NonceSize()

	if len(pkt) < nsiz+aead.Overhead() {
		return nil, ErrShortPacket
	}

	if len(dst) < len(pkt)-nsiz-aead.Overhead() {
		return nil, io.ErrShortBuffer
	}

	b, err := aead.Open(dst[:0], pkt[:nsiz], pkt[nsiz:], nil)
	return b, err
}

// packetConn encrypts net.packetConn with cipher.AEAD
type packetConn struct {
	net.PacketConn
	cipher.AEAD
}

// NewPacketConn wraps a net.PacketConn with AEAD protection.
func NewPacketConn(c net.PacketConn, aead cipher.AEAD) net.PacketConn {
	return &packetConn{PacketConn: c, AEAD: aead}
}

// WriteTo encrypts b and write to addr using the embedded PacketConn.
func (c *packetConn) WriteTo(b []byte, addr net.Addr) (int, error) {
	buf := make([]byte, c.AEAD.NonceSize()+len(b)+c.AEAD.Overhead())
	buf, err := Pack(buf, b, c.AEAD)
	if err != nil {
		return 0, err
	}
	_, err = c.PacketConn.WriteTo(buf, addr)
	return len(b), err
}

// ReadFrom reads from the embedded PacketConn and decrypts into b.
func (c *packetConn) ReadFrom(b []byte) (int, net.Addr, error) {
	n, addr, err := c.PacketConn.ReadFrom(b)
	if err != nil {
		return n, addr, err
	}
	b, err = Unpack(b, b[:n], c.AEAD)
	return len(b), addr, err
}
