package main

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"net"
	"strings"

	"golang.org/x/crypto/chacha20poly1305"

	"github.com/Yawning/chacha20"
	"github.com/riobard/go-shadowsocks2/core"
	"github.com/riobard/go-shadowsocks2/shadowaead"
	"github.com/riobard/go-shadowsocks2/shadowstream"
)

// ErrKeySize means the supplied key size does not meet the requirement of cipher choosed.
var ErrKeySize = errors.New("key size error")

func pickCipher(name string, key []byte) (core.StreamConnCipher, core.PacketConnCipher, error) {

	switch strings.ToLower(name) {
	case "aes-128-gcm", "aes-192-gcm", "aes-256-gcm":
		aead, err := aesGCM(key, 0) // 0 for standard 12-byte nonce
		return aeadStream(aead), aeadPacket(aead), err

	case "aes-128-gcm-16", "aes-192-gcm-16", "aes-256-gcm-16":
		aead, err := aesGCM(key, 16) // 16-byte nonce for better collision avoidance
		return aeadStream(aead), aeadPacket(aead), err

	case "chacha20-ietf-poly1305":
		aead, err := chacha20poly1305.New(key)
		return aeadStream(aead), aeadPacket(aead), err

	case "aes-128-ctr", "aes-192-ctr", "aes-256-ctr":
		ciph, err := aesCTR(key)
		return streamStream(ciph), streamPacket(ciph), err

	case "aes-128-cfb", "aes-192-cfb", "aes-256-cfb":
		ciph, err := aesCFB(key)
		return streamStream(ciph), streamPacket(ciph), err

	case "chacha20-ietf":
		if len(key) != chacha20.KeySize {
			return nil, nil, ErrKeySize
		}
		k := chacha20ietfkey(key)
		return streamStream(k), streamPacket(k), nil

	case "dummy": // only for benchmarking and debugging
		return dummyStream(), dummyPacket(), nil

	default:
		err := fmt.Errorf("cipher not supported: %s", name)
		return nil, nil, err
	}
}

func dummyStream() core.StreamConnCipher {
	return func(c net.Conn) net.Conn { return c }
}
func dummyPacket() core.PacketConnCipher {
	return func(c net.PacketConn) net.PacketConn { return c }
}

func aeadStream(aead cipher.AEAD) core.StreamConnCipher {
	return func(c net.Conn) net.Conn { return shadowaead.NewConn(c, aead) }
}
func aeadPacket(aead cipher.AEAD) core.PacketConnCipher {
	return func(c net.PacketConn) net.PacketConn { return shadowaead.NewPacketConn(c, aead) }
}

func aesGCM(key []byte, nonceSize int) (cipher.AEAD, error) {
	blk, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if nonceSize > 0 {
		return cipher.NewGCMWithNonceSize(blk, nonceSize)
	}
	return cipher.NewGCM(blk) // standard 12-byte nonce
}

func streamStream(ciph shadowstream.Cipher) core.StreamConnCipher {
	return func(c net.Conn) net.Conn { return shadowstream.NewConn(c, ciph) }
}

func streamPacket(ciph shadowstream.Cipher) core.PacketConnCipher {
	return func(c net.PacketConn) net.PacketConn { return shadowstream.NewPacketConn(c, ciph) }
}

type ctrStream struct{ cipher.Block }

func (b *ctrStream) IVSize() int                       { return b.BlockSize() }
func (b *ctrStream) Encrypter(iv []byte) cipher.Stream { return cipher.NewCTR(b, iv) }
func (b *ctrStream) Decrypter(iv []byte) cipher.Stream { return b.Encrypter(iv) }

func aesCTR(key []byte) (shadowstream.Cipher, error) {
	blk, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return &ctrStream{blk}, nil
}

type cfbStream struct{ cipher.Block }

func (b *cfbStream) IVSize() int                       { return b.BlockSize() }
func (b *cfbStream) Encrypter(iv []byte) cipher.Stream { return cipher.NewCFBEncrypter(b, iv) }
func (b *cfbStream) Decrypter(iv []byte) cipher.Stream { return cipher.NewCFBDecrypter(b, iv) }

func aesCFB(key []byte) (shadowstream.Cipher, error) {
	blk, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return &ctrStream{blk}, nil
}

type chacha20ietfkey []byte

func (k chacha20ietfkey) IVSize() int { return chacha20.INonceSize }
func (k chacha20ietfkey) Encrypter(iv []byte) cipher.Stream {
	ciph, err := chacha20.NewCipher(k, iv)
	if err != nil {
		panic(err)
	}
	return ciph
}
func (k chacha20ietfkey) Decrypter(iv []byte) cipher.Stream { return k.Encrypter(iv) }
