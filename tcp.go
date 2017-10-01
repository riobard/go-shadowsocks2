package main

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/riobard/go-shadowsocks2/socks"
)

// Create a SOCKS server listening on addr and proxy to server.
func socksLocal(addr string, d Dialer) {
	logf("SOCKS proxy %s", addr)
	tcpLocal(addr, d, func(c net.Conn) (socks.Addr, error) { return socks.Handshake(c) })
}

// Create a TCP tunnel from addr to target via server.
func tcpTun(addr, target string, d Dialer) {
	tgt := socks.ParseAddr(target)
	if tgt == nil {
		logf("invalid target address %q", target)
		return
	}
	logf("TCP tunnel %s <-> %s", addr, target)
	tcpLocal(addr, d, func(net.Conn) (socks.Addr, error) { return tgt, nil })
}

// Listen on addr and proxy to server to reach target from getAddr.
func tcpLocal(addr string, d Dialer, getAddr func(net.Conn) (socks.Addr, error)) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		logf("failed to listen on %s: %v", addr, err)
		return
	}

	for {
		c, err := l.Accept()
		if err != nil {
			logf("failed to accept: %s", err)
			continue
		}

		go func() {
			defer c.Close()
			c.(*net.TCPConn).SetKeepAlive(true)

			tgt, err := getAddr(c)
			if err != nil {
				logf("failed to get target address: %v", err)
				return
			}

			rc, err := d.Dial("tcp", tgt.String())
			if err != nil {
				logf("failed to connect: %v", err)
				return
			}

			logf("proxy %s <--[%s]--> %s", c.RemoteAddr(), rc.RemoteAddr(), tgt)
			_, _, err = relay(rc, c)
			if err != nil {
				if err, ok := err.(net.Error); ok && err.Timeout() {
					return // ignore i/o timeout
				}
				logf("relay error: %v", err)
			}
		}()
	}
}

// Listen on addr for incoming connections.
func tcpRemote(addr string, shadow func(net.Conn) net.Conn) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		logf("failed to listen on %s: %v", addr, err)
		return
	}

	logf("listening TCP on %s", addr)
	for {
		c, err := l.Accept()
		if err != nil {
			logf("failed to accept: %v", err)
			continue
		}

		go func() {
			defer c.Close()
			c.(*net.TCPConn).SetKeepAlive(true)
			c = shadow(c)

			tgt, err := socks.ReadAddr(c)
			if err != nil {
				logf("failed to get target address: %v", err)
				return
			}

			rc, err := net.Dial("tcp", tgt.String())
			if err != nil {
				logf("failed to connect to target: %v", err)
				return
			}
			defer rc.Close()
			rc.(*net.TCPConn).SetKeepAlive(true)

			logf("proxy %s <-> %s", c.RemoteAddr(), tgt)
			_, _, err = relay(c, rc)
			if err != nil {
				if err, ok := err.(net.Error); ok && err.Timeout() {
					return // ignore i/o timeout
				}
				logf("relay error: %v", err)
			}
		}()
	}
}

// relay copies between left and right bidirectionally. Returns number of
// bytes copied from right to left, from left to right, and any error occurred.
func relay(left, right net.Conn) (int64, int64, error) {
	var nLR, nRL int64
	var errLR, errRL error
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		nRL, errRL = io.Copy(right, left)
		right.SetDeadline(time.Now()) // wake up the other goroutine blocking on right
		left.SetDeadline(time.Now())  // wake up the other goroutine blocking on left
	}()

	nLR, errLR = io.Copy(left, right)
	right.SetDeadline(time.Now()) // wake up the other goroutine blocking on right
	left.SetDeadline(time.Now())  // wake up the other goroutine blocking on left
	wg.Wait()

	if errLR == nil {
		errLR = errRL
	}
	return nLR, nRL, errLR
}
