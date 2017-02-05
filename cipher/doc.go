// Package cipher provides ciphers for Shadowsocks
package cipher

import (
	"crypto/cipher"
	"errors"
	"sort"
	"strings"

	"github.com/riobard/go-shadowsocks2/core"
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

// ListCiphers returns a list of available cipher names sorted alphabetically.
func ListCiphers() []string {
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

// MakeCipher returns a pair of ciphers for the given key.
func MakeCipher(name string, key []byte) (core.StreamConnCipher, core.PacketConnCipher, error) {
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
