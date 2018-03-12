// +build !linux

package main

func redirLocal(addr string, d Dialer) {
	logf("TCP redirect not supported")
}

func redir6Local(addr string, d Dialer) {
	logf("TCP6 redirect not supported")
}
