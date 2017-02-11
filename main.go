package main

import (
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
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
		Password  string
		Keygen    int
		Socks     string
		RedirTCP  string
		RedirTCP6 string
		TCPTun    string
		UDPTun    string
	}

	flag.BoolVar(&config.Verbose, "verbose", false, "verbose mode")
	flag.StringVar(&flags.Cipher, "cipher", "chacha20-ietf-poly1305", "available ciphers: "+strings.Join(listCipher(), " "))
	flag.StringVar(&flags.Key, "key", "", "base64url-encoded key (derive from password if empty)")
	flag.IntVar(&flags.Keygen, "keygen", 0, "generate a base64url-encoded random key of given length in byte")
	flag.StringVar(&flags.Password, "password", "", "password")
	flag.StringVar(&flags.Server, "s", "", "server listen address")
	flag.StringVar(&flags.Client, "c", "", "client connect address")
	flag.StringVar(&flags.Socks, "socks", ":1080", "(client-only) SOCKS listen address")
	flag.StringVar(&flags.RedirTCP, "redir", "", "(client-only) redirect TCP from this address")
	flag.StringVar(&flags.RedirTCP6, "redir6", "", "(client-only) redirect TCP IPv6 from this address")
	flag.StringVar(&flags.TCPTun, "tcptun", "", "(client-only) TCP tunnel (laddr1=raddr1,laddr2=raddr2,...)")
	flag.StringVar(&flags.UDPTun, "udptun", "", "(client-only) UDP tunnel (laddr1=raddr1,laddr2=raddr2,...)")
	flag.DurationVar(&config.UDPTimeout, "udptimeout", 5*time.Minute, "UDP tunnel timeout")
	flag.Parse()

	if flags.Keygen > 0 {
		key := make([]byte, flags.Keygen)
		io.ReadFull(rand.Reader, key)
		fmt.Println(base64.URLEncoding.EncodeToString(key))
		return
	}

	if flags.Client == "" && flags.Server == "" {
		flag.Usage()
		return
	}

	var key []byte
	if flags.Key != "" {
		k, err := base64.URLEncoding.DecodeString(flags.Key)
		if err != nil {
			log.Fatal(err)
		}
		key = k
	}

	ciph, err := pickCipher(flags.Cipher, key, flags.Password)
	if err != nil {
		log.Fatal(err)
	}

	if flags.Client != "" { // client mode
		if flags.UDPTun != "" {
			for _, tun := range strings.Split(flags.UDPTun, ",") {
				p := strings.Split(tun, "=")
				go udpLocal(p[0], flags.Client, p[1], ciph)
			}
		}

		if flags.TCPTun != "" {
			for _, tun := range strings.Split(flags.TCPTun, ",") {
				p := strings.Split(tun, "=")
				go tcpTun(p[0], flags.Client, p[1], ciph)
			}
		}

		if flags.Socks != "" {
			go socksLocal(flags.Socks, flags.Client, ciph)
		}

		if flags.RedirTCP != "" {
			go redirLocal(flags.RedirTCP, flags.Client, ciph)
		}

		if flags.RedirTCP6 != "" {
			go redir6Local(flags.RedirTCP6, flags.Client, ciph)
		}
	}

	if flags.Server != "" { // server mode
		go udpRemote(flags.Server, ciph)
		go tcpRemote(flags.Server, ciph)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}
