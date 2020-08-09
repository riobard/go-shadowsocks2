// +build !linux,!darwin

package main

import (
	"net"
	"time"
)

func redirLocal(addr, server string, shadow func(net.Conn) net.Conn) {
	logf("TCP redirect not supported")
}

func redir6Local(addr, server string, shadow func(net.Conn) net.Conn) {
	logf("TCP6 redirect not supported")
}

func timedCork(c *net.TCPConn, d time.Duration) error { return nil }
