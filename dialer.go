package main

import (
	"errors"
	"net"

	"github.com/riobard/go-shadowsocks2/core"
	"github.com/riobard/go-shadowsocks2/socks"
	"github.com/riobard/go-shadowsocks2/speeddial"
)

type Dialer interface {
	Dial(network, address string) (net.Conn, error)
}

type dialer struct {
	*speeddial.Dialer
}

func (d dialer) Dial(network, address string) (net.Conn, error) {
	if network != "tcp" {
		return nil, errors.New("only TCP network is supported")
	}
	c, err := d.Dialer.Dial()
	if err != nil {
		return c, err
	}
	_, err = c.Write(socks.ParseAddr(address))
	if err != nil {
		c.Close()
	}
	return c, err
}

func fastdialer(u ...string) (*dialer, error) {
	rs := make([]speeddial.Dial, len(u))
	for i := range u {
		addr, cipher, password, err := parseURL(u[i])
		if err != nil {
			return nil, err
		}

		ciph, err := core.PickCipher(cipher, nil, password)
		if err != nil {
			return nil, err
		}

		rs[i] = func() (net.Conn, error) {
			c, err := net.Dial("tcp", addr)
			if err != nil {
				return c, err
			}
			c = ciph.StreamConn(c)
			return c, nil
		}
	}
	return &dialer{speeddial.New(rs...)}, nil
}
