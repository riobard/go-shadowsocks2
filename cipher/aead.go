package cipher

import (
	"crypto/aes"
	"crypto/cipher"

	"golang.org/x/crypto/chacha20poly1305"
)

// AEAD ciphers

func aesGCM(key []byte, nonceSize int) (cipher.AEAD, error) {
	blk, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	if nonceSize > 0 {
		return cipher.NewGCMWithNonceSize(blk, nonceSize)
	}
	return cipher.NewGCM(blk)
}

// AES-GCM with standard 12-byte nonce
func AESGCM(key []byte) (cipher.AEAD, error) { return aesGCM(key, 0) }

// AES-GCM with 16-byte nonce for better collision avoidance.
func AESGCM16(key []byte) (cipher.AEAD, error) { return aesGCM(key, 16) }

func Chacha20IETFPoly1305(key []byte) (cipher.AEAD, error) { return chacha20poly1305.New(key) }
