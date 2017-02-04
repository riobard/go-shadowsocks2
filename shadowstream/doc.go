/*
Package shadowstream implements the original Shadowsocks protocol protected by stream cipher.
*/
package shadowstream

import "crypto/cipher"

// Cipher generates a pair of stream ciphers for encryption and decryption.
type Cipher interface {
	IVSize() int
	Encrypter(iv []byte) cipher.Stream
	Decrypter(iv []byte) cipher.Stream
}
