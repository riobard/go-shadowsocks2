package main

import (
	"encoding/hex"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var config struct {
	Verbose    bool
	UDPTimeout time.Duration
}

func logf(f string, v ...interface{}) {
	if config.Verbose {
		log.Printf(f, v...)
	}
}

func main() {

	var flags struct {
		Client    string
		Server    string
		Cipher    string
		Key       string
		Socks     string
		RedirTCP  string
		RedirTCP6 string
		TCPTun    string
		UDPTun    string
	}

	flag.BoolVar(&config.Verbose, "verbose", false, "verbose mode")
	flag.StringVar(&flags.Cipher, "cipher", "", "cipher")
	flag.StringVar(&flags.Key, "key", "", "secret key in hexadecimal")
	flag.StringVar(&flags.Server, "s", "", "server listen address")
	flag.StringVar(&flags.Client, "c", "", "client connect address")
	flag.StringVar(&flags.Socks, "socks", ":1080", "(client-only) SOCKS listen address")
	flag.StringVar(&flags.RedirTCP, "redir", "", "(client-only) redirect TCP from this address")
	flag.StringVar(&flags.RedirTCP6, "redir6", "", "(client-only) redirect TCP IPv6 from this address")
	flag.StringVar(&flags.TCPTun, "tcptun", "", "(client-only) TCP tunnel (laddr1=raddr1,laddr2=raddr2,...)")
	flag.StringVar(&flags.UDPTun, "udptun", "", "(client-only) UDP tunnel (laddr1=raddr1,laddr2=raddr2,...)")
	flag.DurationVar(&config.UDPTimeout, "udptimeout", 5*time.Minute, "UDP tunnel timeout")
	flag.Parse()

	if flags.Cipher == "" {
		printCiphers(os.Stderr)
		return
	}

	key, err := hex.DecodeString(flags.Key)
	if err != nil {
		log.Fatalf("key: %v", err)
	}

	streamCipher, packetCipher, err := pickCipher(flags.Cipher, key)
	if err != nil {
		log.Fatalf("cipher: %v", err)
	}

	if flags.Client != "" { // client mode
		if flags.UDPTun != "" {
			for _, tun := range strings.Split(flags.UDPTun, ",") {
				p := strings.Split(tun, "=")
				go udpLocal(p[0], flags.Client, p[1], packetCipher)
			}
		}

		if flags.TCPTun != "" {
			for _, tun := range strings.Split(flags.TCPTun, ",") {
				p := strings.Split(tun, "=")
				go tcpTun(p[0], flags.Client, p[1], streamCipher)
			}
		}

		if flags.Socks != "" {
			go socksLocal(flags.Socks, flags.Client, streamCipher)
		}

		if flags.RedirTCP != "" {
			go redirLocal(flags.RedirTCP, flags.Client, streamCipher)
		}

		if flags.RedirTCP6 != "" {
			go redir6Local(flags.RedirTCP6, flags.Client, streamCipher)
		}
	} else if flags.Server != "" { // server mode
		go udpRemote(flags.Server, packetCipher)
		go tcpRemote(flags.Server, streamCipher)
	} else {
		flag.Usage()
		return
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}
