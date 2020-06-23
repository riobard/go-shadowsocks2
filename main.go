package main

import (
	"flag"
	"log"
	"strings"

	"github.com/riobard/go-shadowsocks2/core"
	"github.com/riobard/go-shadowsocks2/listen"
)

func main() {
	listCiphers := flag.Bool("cipher", false, "List supported ciphers")
	flag.Parse()

	if *listCiphers {
		println(strings.Join(core.ListCipher(), " "))
		return
	}

	if len(config.Client) == 0 && len(config.Server) == 0 {
		flag.Usage()
		return
	}

	if len(config.Client) > 0 {
		client()
	}

	if len(config.Server) > 0 {
		server()
	}

	select {}
}

func client() {
	if len(config.UDPTun) > 0 { // use first server for UDP
		addr, cipher, password, err := parseURL(config.Client[0])
		if err != nil {
			log.Fatal(err)
		}

		ciph, err := core.PickCipher(cipher, nil, password)
		if err != nil {
			log.Fatal(err)
		}
		for _, p := range config.UDPTun {
			go udpLocal(p[0], addr, p[1], ciph.PacketConn)
		}
	}

	d, err := fastdialer(config.Client...)
	if err != nil {
		log.Fatalf("failed to create dialer: %v", err)
	}

	if len(config.TCPTun) > 0 {
		for _, p := range config.TCPTun {
			l, err := listen.ListenTo("tcp", p[0], p[1])
			if err != nil {
				log.Fatal(err)
			}
			logf("tcptun %v --> %v", p[0], p[1])
			go tcpLocal(l, d)
		}
	}

	if config.Socks != "" {
		l, err := listen.Listen("socks", "tcp", config.Socks)
		if err != nil {
			log.Fatal(err)
		}
		logf("socks %v", config.Socks)
		go tcpLocal(l, d)
	}

	if config.RedirTCP != "" {
		l, err := listen.Listen("redir", "tcp", config.RedirTCP)
		if err != nil {
			log.Fatal(err)
		}
		logf("redir tcp %v", config.RedirTCP)
		go tcpLocal(l, d)
	}

	if config.TproxyTCP != "" {
		l, err := listen.Listen("tproxy", "tcp", config.TproxyTCP)
		if err != nil {
			log.Fatal(err)
		}
		logf("tproxy tcp %v", config.TproxyTCP)
		go tcpLocal(l, d)
	}
}

func server() {
	for _, each := range config.Server {
		addr, cipher, password, err := parseURL(each)
		if err != nil {
			log.Fatal(err)
		}

		ciph, err := core.PickCipher(cipher, nil, password)
		if err != nil {
			log.Fatal(err)
		}

		if config.UDP {
			go udpRemote(addr, ciph.PacketConn)
		}
		go tcpRemote(addr, ciph.StreamConn)
	}
}

func logf(f string, v ...interface{}) {
	if config.Verbose {
		log.Printf(f, v...)
	}
}
