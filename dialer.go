package main

import (
	"net"
	"sync/atomic"
	"time"

	"github.com/riobard/go-shadowsocks2/core"
	"github.com/riobard/go-shadowsocks2/socks"
)

type Dialer interface {
	Dial(network, address string) (net.Conn, error)
}

type dialer struct {
	dialTime    int64        // time to dial in nanoseconds (exponetially smoothed)
	lastUpdated atomic.Value // of time.Time
	server      string
	shadow      func(net.Conn) net.Conn
}

func NewDialer(u string) (*dialer, error) {
	addr, cipher, password, err := parseURL(u)
	if err != nil {
		return nil, err
	}
	ciph, err := core.PickCipher(cipher, nil, password)
	if err != nil {
		return nil, err
	}
	d := &dialer{server: addr, shadow: ciph.StreamConn}
	d.lastUpdated.Store(time.Time{})
	return d, nil
}

func (d *dialer) Dial(network, address string) (net.Conn, error) {
	c, err := d.dial()
	if err != nil {
		return c, err
	}
	c.(*net.TCPConn).SetKeepAlive(true)
	c = d.shadow(c)
	_, err = c.Write(socks.ParseAddr(address))
	return c, err
}

func (d *dialer) dial() (net.Conn, error) {
	const timeout = 2 * time.Second
	const wt = 4

	t0 := time.Now()
	c, err := net.DialTimeout("tcp", d.server, timeout)
	td := time.Since(t0)
	if err != nil {
		td = timeout // penality
	}

	new := td.Nanoseconds()
	if old := atomic.LoadInt64(&d.dialTime); old > 0 {
		new = (wt*old + new) / (wt + 1) // Exponentially Weighted Moving Average
	}
	atomic.StoreInt64(&d.dialTime, new)
	logf("probe %s [%d ms] err=%v", d.server, new/1e6, err)
	d.lastUpdated.Store(time.Now())
	return c, err
}

// Actively measure average dial time
func (d *dialer) probe() {
	const interval = 10 * time.Second
	for {
		age := time.Since(d.lastUpdated.Load().(time.Time))
		if age > interval {
			if c, err := d.dial(); err == nil {
				c.Close()
			}
		} else {
			time.Sleep(interval - age)
		}
	}
}

type priorityDialer struct {
	dialers []*dialer
}

func NewPriorityDialer(u ...string) (*priorityDialer, error) {
	var dialers []*dialer

	for _, each := range u {
		d, err := NewDialer(each)
		if err != nil {
			return nil, err
		}
		dialers = append(dialers, d)
	}

	for _, d := range dialers {
		go d.probe()
	}

	return &priorityDialer{dialers}, nil
}

const maxInt64 = int64(1<<63 - 1)

func (d *priorityDialer) Dial(network, address string) (net.Conn, error) {
	tMin := maxInt64
	var dMin *dialer
	for _, d := range d.dialers {
		if t := atomic.LoadInt64(&d.dialTime); t < tMin {
			dMin, tMin = d, t
		}
	}
	logf("best server %s [%d ms]", dMin.server, tMin/1e6)
	return dMin.Dial(network, address)
}
