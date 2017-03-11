package main

import (
	"io"
	"net"
	"time"

	"github.com/shadowsocks/go-shadowsocks2/core"
	"github.com/shadowsocks/go-shadowsocks2/socks"
)

// Create a SOCKS server listening on addr and proxy to server.
func socksLocal(addr, server string, ciph core.StreamConnCipher) {
	logf("SOCKS proxy %s <-> %s", addr, server)
	tcpLocal(addr, server, ciph, func(c net.Conn) (socks.Addr, error) { return socks.Handshake(c) })
}

// Create a TCP tunnel from addr to target via server.
func tcpTun(addr, server, target string, ciph core.StreamConnCipher) {
	tgt := socks.ParseAddr(target)
	if tgt == nil {
		logf("invalid target address %q", target)
		return
	}
	logf("TCP tunnel %s <-> %s <-> %s", addr, server, target)
	tcpLocal(addr, server, ciph, func(net.Conn) (socks.Addr, error) { return tgt, nil })
}

// Listen on addr and proxy to server to reach target from getAddr.
func tcpLocal(addr, server string, ciph core.StreamConnCipher, getAddr func(net.Conn) (socks.Addr, error)) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		logf("failed to listen on %s: %v", addr, err)
		return
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			logf("failed to accept: %s", err)
			continue
		}

		tgt, err := getAddr(conn)
		if err != nil {
			logf("failed to get target address: %v", err)
			continue
		}

		go tcpLocalHandle(conn, server, tgt, ciph)
	}
}

func tcpLocalHandle(c net.Conn, server string, target socks.Addr, ciph core.StreamConnCipher) {
	logf("proxy %s <-> %s <-> %s", c.RemoteAddr(), server, target)

	defer c.Close()

	sc, err := core.Dial("tcp", server, ciph)
	if err != nil {
		logf("failed to connect to server %v: %v", server, err)
		return
	}
	defer sc.Close()

	if _, err = sc.Write(target); err != nil {
		logf("failed to send target address: %v", err)
		return
	}

	_, _, err = relay(sc, c)
	if err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return // ignore i/o timeout
		}
		logf("relay error: %v", err)
	}
}

// Listen on addr for incoming connections.
func tcpRemote(addr string, ciph core.StreamConnCipher) {
	ln, err := core.Listen("tcp", addr, ciph)
	if err != nil {
		logf("failed to listen on %s: %v", addr, err)
		return
	}

	logf("listening TCP on %s", addr)
	for {
		conn, err := ln.Accept()
		if err != nil {
			logf("failed to accept: %s", err)
			continue
		}
		go tcpRemoteHandle(conn)
	}
}

func tcpRemoteHandle(c net.Conn) {
	defer c.Close()

	addr, err := socks.ReadAddr(c)
	if err != nil {
		logf("failed to read address: %v", err)
		return
	}
	logf("proxy %s <-> %s", c.RemoteAddr(), addr)

	conn, err := net.Dial("tcp", addr.String())
	if err != nil {
		logf("failed to connect to target: %s", err)
		return
	}
	defer conn.Close()

	_, _, err = relay(c, conn)
	if err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return // ignore i/o timeout
		}
		logf("relay error: %v", err)
	}
}

// relay copies between left and right bidirectionally. Returns number of
// bytes copied from right to left, from left to right, and any error occurred.
func relay(left, right net.Conn) (int64, int64, error) {
	type res struct {
		N   int64
		Err error
	}
	ch := make(chan res)

	go func() {
		n, err := io.Copy(right, left)
		right.SetDeadline(time.Now()) // wake up the other goroutine blocking on right
		left.SetDeadline(time.Now())  // wake up the other goroutine blocking on left
		ch <- res{n, err}
	}()

	n, err := io.Copy(left, right)
	right.SetDeadline(time.Now()) // wake up the other goroutine blocking on right
	left.SetDeadline(time.Now())  // wake up the other goroutine blocking on left
	rs := <-ch

	if err == nil {
		err = rs.Err
	}
	return n, rs.N, err
}
