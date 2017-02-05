package main

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"fmt"
	"io"
	"net"
	"sort"
	"strings"

	"github.com/Yawning/chacha20"
	"github.com/riobard/go-shadowsocks2/core"
	"github.com/riobard/go-shadowsocks2/shadowaead"
	"github.com/riobard/go-shadowsocks2/shadowstream"
	"golang.org/x/crypto/chacha20poly1305"
)

// ErrKeySize means the key size does not meet the requirement of cipher.
var ErrKeySize = errors.New("key size error")

// ErrCipherNotSupported means the cipher has not been implemented.
var ErrCipherNotSupported = errors.New("cipher not supported")

// List of AEAD ciphers: key size in bytes and constructor
var aeadList = map[string]struct {
	KeySize int
	New     func(key []byte) (cipher.AEAD, error)
}{
	"aes-128-gcm":            {16, aesGCM},
	"aes-192-gcm":            {24, aesGCM},
	"aes-256-gcm":            {32, aesGCM},
	"aes-128-gcm-16":         {16, aesGCM16},
	"aes-192-gcm-16":         {24, aesGCM16},
	"aes-256-gcm-16":         {32, aesGCM16},
	"chacha20-ietf-poly1305": {32, chacha20poly1305.New},
}

// List of stream ciphers: key size in bytes and constructor
var streamList = map[string]struct {
	KeySize int
	New     func(key []byte) (shadowstream.Cipher, error)
}{
	"aes-128-ctr":   {16, aesCTR},
	"aes-192-ctr":   {24, aesCTR},
	"aes-256-ctr":   {32, aesCTR},
	"aes-128-cfb":   {16, aesCFB},
	"aes-192-cfb":   {24, aesCFB},
	"aes-256-cfb":   {32, aesCFB},
	"chacha20-ietf": {32, newChacha20ietf},
}

// Return two lists of sorted cipher names.
func availableCiphers() (aead []string, stream []string) {
	for k := range aeadList {
		aead = append(aead, k)
	}

	for k := range streamList {
		stream = append(stream, k)
	}

	sort.Strings(aead)
	sort.Strings(stream)
	return
}

// Print available ciphers to w
func printCiphers(w io.Writer) {
	fmt.Fprintf(w, "## Available AEAD ciphers (recommended)\n\n")

	aead, stream := availableCiphers()
	for _, name := range aead {
		fmt.Fprintf(w, "%s\n", name)
	}

	fmt.Fprintf(w, "\n## Available stream ciphers\n\n")
	for _, name := range stream {
		fmt.Fprintf(w, "%s\n", name)
	}
}

func pickCipher(name string, key []byte) (core.StreamConnCipher, core.PacketConnCipher, error) {
	name = strings.ToLower(name)

	if choice, ok := aeadList[name]; ok {
		if len(key) != choice.KeySize {
			return nil, nil, ErrKeySize
		}
		aead, err := choice.New(key)
		return aeadStream(aead), aeadPacket(aead), err
	}

	if choice, ok := streamList[name]; ok {
		if len(key) != choice.KeySize {
			return nil, nil, ErrKeySize
		}
		ciph, err := choice.New(key)
		return streamStream(ciph), streamPacket(ciph), err
	}

	if name == "dummy" {
		return dummyStream(), dummyPacket(), nil
	}
	return nil, nil, ErrCipherNotSupported
}

// Dummy ciphers (no encryption)

func dummyStream() core.StreamConnCipher {
	return func(c net.Conn) net.Conn { return c }
}
func dummyPacket() core.PacketConnCipher {
	return func(c net.PacketConn) net.PacketConn { return c }
}

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
