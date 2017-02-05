package main

import (
	"crypto/cipher"
	"crypto/md5"
	"fmt"
	"net"
	"sort"
	"strings"

	sscipher "github.com/riobard/go-shadowsocks2/cipher"
	"github.com/riobard/go-shadowsocks2/core"
	"github.com/riobard/go-shadowsocks2/shadowaead"
	"github.com/riobard/go-shadowsocks2/shadowstream"
)

// List of AEAD ciphers: key size in bytes and constructor
var aeadList = map[string]struct {
	KeySize int
	New     func(key []byte) (cipher.AEAD, error)
}{
	"aes-128-gcm":            {16, sscipher.AESGCM},
	"aes-192-gcm":            {24, sscipher.AESGCM},
	"aes-256-gcm":            {32, sscipher.AESGCM},
	"aes-128-gcm-16":         {16, sscipher.AESGCM16},
	"aes-192-gcm-16":         {24, sscipher.AESGCM16},
	"aes-256-gcm-16":         {32, sscipher.AESGCM16},
	"chacha20-ietf-poly1305": {32, sscipher.Chacha20IETFPoly1305},
}

// List of stream ciphers: key size in bytes and constructor
var streamList = map[string]struct {
	KeySize int
	New     func(key []byte) (shadowstream.Cipher, error)
}{
	"aes-128-ctr":   {16, sscipher.AESCTR},
	"aes-192-ctr":   {24, sscipher.AESCTR},
	"aes-256-ctr":   {32, sscipher.AESCTR},
	"aes-128-cfb":   {16, sscipher.AESCFB},
	"aes-192-cfb":   {24, sscipher.AESCFB},
	"aes-256-cfb":   {32, sscipher.AESCFB},
	"chacha20-ietf": {32, sscipher.Chacha20IETF},
}

// listCipher returns a list of available cipher names sorted alphabetically.
func listCipher() []string {
	var l []string
	for k := range aeadList {
		l = append(l, k)
	}
	for k := range streamList {
		l = append(l, k)
	}
	sort.Strings(l)
	return l
}

// derive key from password if given key is empty
func pickCipher(name string, key []byte, password string) (core.StreamConnCipher, core.PacketConnCipher, error) {
	name = strings.ToLower(name)

	if name == "dummy" {
		return dummyStream(), dummyPacket(), nil
	}

	if choice, ok := aeadList[name]; ok {
		if len(key) == 0 {
			key = kdf(password, choice.KeySize)
		}
		if len(key) != choice.KeySize {
			return nil, nil, fmt.Errorf("key size error: need %d-byte key", choice.KeySize)
		}
		aead, err := choice.New(key)
		return aeadStream(aead), aeadPacket(aead), err
	}

	if choice, ok := streamList[name]; ok {
		if len(key) == 0 {
			key = kdf(password, choice.KeySize)
		}
		if len(key) != choice.KeySize {
			return nil, nil, fmt.Errorf("key size error: need %d-byte key", choice.KeySize)
		}
		ciph, err := choice.New(key)
		return streamStream(ciph), streamPacket(ciph), err
	}

	return nil, nil, fmt.Errorf("cipher %q not supported", name)
}

func aeadStream(aead cipher.AEAD) core.StreamConnCipher {
	return func(c net.Conn) net.Conn { return shadowaead.NewConn(c, aead) }
}
func aeadPacket(aead cipher.AEAD) core.PacketConnCipher {
	return func(c net.PacketConn) net.PacketConn { return shadowaead.NewPacketConn(c, aead) }
}

func streamStream(ciph shadowstream.Cipher) core.StreamConnCipher {
	return func(c net.Conn) net.Conn { return shadowstream.NewConn(c, ciph) }
}
func streamPacket(ciph shadowstream.Cipher) core.PacketConnCipher {
	return func(c net.PacketConn) net.PacketConn { return shadowstream.NewPacketConn(c, ciph) }
}

// dummy cipher does not encrypt
func dummyStream() core.StreamConnCipher { return func(c net.Conn) net.Conn { return c } }
func dummyPacket() core.PacketConnCipher { return func(c net.PacketConn) net.PacketConn { return c } }

// key-derivation function from original Shadowsocks
func kdf(password string, keyLen int) []byte {
	var b, prev []byte
	h := md5.New()
	for len(b) < keyLen {
		h.Write(prev)
		h.Write([]byte(password))
		b = h.Sum(b)
		prev = b[len(b)-h.Size():]
		h.Reset()
	}
	return b[:keyLen]
}
