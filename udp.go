package main

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/riobard/go-shadowsocks2/socks"
)

const udpBufSize = 64 * 1024

var bufPool = sync.Pool{New: func() interface{} { return make([]byte, udpBufSize) }}

// Listen on laddr for UDP packets, encrypt and send to server to reach target.
func udpLocal(laddr, server, target string, shadow func(net.PacketConn) net.PacketConn) {
	srvAddr, err := net.ResolveUDPAddr("udp", server)
	if err != nil {
		logf("UDP server address error: %v", err)
		return
	}

	tgt := socks.ParseAddr(target)
	if tgt == nil {
		err = fmt.Errorf("invalid target address: %q", target)
		logf("UDP target address error: %v", err)
		return
	}

	c, err := net.ListenPacket("udp", laddr)
	if err != nil {
		logf("UDP local listen error: %v", err)
		return
	}
	defer c.Close()

	m := make(map[string]chan []byte)
	var lock sync.Mutex

	logf("UDP tunnel %s <-> %s <-> %s", laddr, server, target)
	for {
		buf := bufPool.Get().([]byte)
		copy(buf, tgt)
		n, raddr, err := c.ReadFrom(buf[len(tgt):])
		if err != nil {
			logf("UDP local read error: %v", err)
			continue
		}

		lock.Lock()
		k := raddr.String()
		ch := m[k]
		if ch == nil {
			pc, err := net.ListenPacket("udp", "")
			if err != nil {
				logf("failed to create UDP socket: %v", err)
				goto Unlock
			}
			pc = shadow(pc)
			ch = make(chan []byte, 1) // must use buffered chan
			m[k] = ch

			go func() { // recv from user and send to udpRemote
				for buf := range ch {
					pc.SetReadDeadline(time.Now().Add(config.UDPTimeout)) // extend read timeout
					if _, err := pc.WriteTo(buf, srvAddr); err != nil {
						logf("UDP local write error: %v", err)
					}
					bufPool.Put(buf[:cap(buf)])
				}
			}()

			go func() { // recv from udpRemote and send to user
				if err := timedCopy(raddr, c, pc, config.UDPTimeout, false); err != nil {
					if err, ok := err.(net.Error); ok && err.Timeout() {
						// ignore i/o timeout
					} else {
						logf("timedCopy error: %v", err)
					}
				}
				pc.Close()
				lock.Lock()
				if ch := m[k]; ch != nil {
					close(ch)
				}
				delete(m, k)
				lock.Unlock()
			}()
		}
	Unlock:
		lock.Unlock()

		select {
		case ch <- buf[:len(tgt)+n]: // send
		default: // drop
			bufPool.Put(buf)
		}
	}
}

// Listen on addr for encrypted packets and basically do UDP NAT.
func udpRemote(addr string, shadow func(net.PacketConn) net.PacketConn) {
	c, err := net.ListenPacket("udp", addr)
	if err != nil {
		logf("UDP remote listen error: %v", err)
		return
	}
	defer c.Close()
	c = shadow(c)

	m := make(map[string]chan []byte)
	var lock sync.Mutex

	logf("listening UDP on %s", addr)
	for {
		buf := bufPool.Get().([]byte)
		n, raddr, err := c.ReadFrom(buf)
		if err != nil {
			logf("UDP remote read error: %v", err)
			continue
		}

		lock.Lock()
		k := raddr.String()
		ch := m[k]
		if ch == nil {
			pc, err := net.ListenPacket("udp", "")
			if err != nil {
				logf("failed to create UDP socket: %v", err)
				goto Unlock
			}
			ch = make(chan []byte, 1) // must use buffered chan
			m[k] = ch

			go func() { // receive from udpLocal and send to target
				var tgtUDPAddr *net.UDPAddr
				var err error

				for buf := range ch {
					tgtAddr := socks.SplitAddr(buf)
					if tgtAddr == nil {
						logf("failed to split target address from packet: %q", buf)
						goto End
					}
					tgtUDPAddr, err = net.ResolveUDPAddr("udp", tgtAddr.String())
					if err != nil {
						logf("failed to resolve target UDP address: %v", err)
						goto End
					}
					pc.SetReadDeadline(time.Now().Add(config.UDPTimeout))
					if _, err = pc.WriteTo(buf[len(tgtAddr):], tgtUDPAddr); err != nil {
						logf("UDP remote write error: %v", err)
						goto End
					}
				End:
					bufPool.Put(buf[:cap(buf)])
				}
			}()

			go func() { // receive from udpLocal and send to client
				if err := timedCopy(raddr, c, pc, config.UDPTimeout, true); err != nil {
					if err, ok := err.(net.Error); ok && err.Timeout() {
						// ignore i/o timeout
					} else {
						logf("timedCopy error: %v", err)
					}
				}
				pc.Close()
				lock.Lock()
				if ch := m[k]; ch != nil {
					close(ch)
				}
				delete(m, k)
				lock.Unlock()
			}()
		}
	Unlock:
		lock.Unlock()

		select {
		case ch <- buf[:n]: // sent
		default: // drop
			bufPool.Put(buf)
		}
	}
}

// copy from src to dst at target with read timeout
func timedCopy(target net.Addr, dst, src net.PacketConn, timeout time.Duration, prependSrcAddr bool) error {
	buf := bufPool.Get().([]byte)
	defer bufPool.Put(buf)

	for {
		src.SetReadDeadline(time.Now().Add(timeout))
		n, raddr, err := src.ReadFrom(buf)
		if err != nil {
			return err
		}

		if prependSrcAddr { // server -> client: prepend original packet source address
			srcAddr := socks.ParseAddr(raddr.String())
			copy(buf[len(srcAddr):], buf[:n])
			copy(buf, srcAddr)
			if _, err = dst.WriteTo(buf[:len(srcAddr)+n], target); err != nil {
				return err
			}
		} else { // client -> user: strip original packet source address
			srcAddr := socks.SplitAddr(buf[:n])
			if _, err = dst.WriteTo(buf[len(srcAddr):n], target); err != nil {
				return err
			}
		}
	}
}
