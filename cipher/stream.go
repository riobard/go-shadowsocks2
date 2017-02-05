package cipher

import (
	"crypto/aes"
	"crypto/cipher"
	"net"

	"github.com/Yawning/chacha20"
	"github.com/riobard/go-shadowsocks2/core"
	"github.com/riobard/go-shadowsocks2/shadowstream"
)

// Stream ciphers

func streamStream(ciph shadowstream.Cipher) core.StreamConnCipher {
	return func(c net.Conn) net.Conn { return shadowstream.NewConn(c, ciph) }
}

func streamPacket(ciph shadowstream.Cipher) core.PacketConnCipher {
	return func(c net.PacketConn) net.PacketConn { return shadowstream.NewPacketConn(c, ciph) }
}

// CTR mode
type ctrStream struct{ cipher.Block }

func (b *ctrStream) IVSize() int                       { return b.BlockSize() }
func (b *ctrStream) Decrypter(iv []byte) cipher.Stream { return b.Encrypter(iv) }
func (b *ctrStream) Encrypter(iv []byte) cipher.Stream { return cipher.NewCTR(b, iv) }

func aesCTR(key []byte) (shadowstream.Cipher, error) {
	blk, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return &ctrStream{blk}, nil
}

// CFB mode
type cfbStream struct{ cipher.Block }

func (b *cfbStream) IVSize() int                       { return b.BlockSize() }
func (b *cfbStream) Decrypter(iv []byte) cipher.Stream { return cipher.NewCFBDecrypter(b, iv) }
func (b *cfbStream) Encrypter(iv []byte) cipher.Stream { return cipher.NewCFBEncrypter(b, iv) }

func aesCFB(key []byte) (shadowstream.Cipher, error) {
	blk, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return &ctrStream{blk}, nil
}

// IETF-variant of chacha20
type chacha20ietfkey []byte

func (k chacha20ietfkey) IVSize() int                       { return chacha20.INonceSize }
func (k chacha20ietfkey) Decrypter(iv []byte) cipher.Stream { return k.Encrypter(iv) }
func (k chacha20ietfkey) Encrypter(iv []byte) cipher.Stream {
	ciph, err := chacha20.NewCipher(k, iv)
	if err != nil {
		panic(err) // should never happen
	}
	return ciph
}

func newChacha20ietf(key []byte) (shadowstream.Cipher, error) {
	if len(key) != chacha20.KeySize {
		return nil, ErrKeySize
	}
	return chacha20ietfkey(key), nil
}
