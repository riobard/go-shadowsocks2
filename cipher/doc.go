// Package cipher provides ciphers for Shadowsocks
package cipher

import "strconv"

type KeySizeError int

func (e KeySizeError) Error() string {
	return "key size error: need " + strconv.Itoa(int(e)) + " bytes"
}
