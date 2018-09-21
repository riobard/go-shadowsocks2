package main

import (
	"flag"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Dreamacro/go-shadowsocks2/core"
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
		Client    SpaceSeparatedList
		Server    SpaceSeparatedList
		TCPTun    PairList
		UDPTun    PairList
		Socks     string
		RedirTCP  string
		RedirTCP6 string
		TproxyTCP string
	}

	listCiphers := flag.Bool("cipher", false, "List supported ciphers")
	flag.BoolVar(&config.Verbose, "verbose", false, "verbose mode")
	flag.Var(&flags.Server, "s", "server listen url")
	flag.Var(&flags.Client, "c", "client connect url")
	flag.Var(&flags.TCPTun, "tcptun", "(client-only) TCP tunnel (laddr1=raddr1,laddr2=raddr2,...)")
	flag.Var(&flags.UDPTun, "udptun", "(client-only) UDP tunnel (laddr1=raddr1,laddr2=raddr2,...)")
	flag.StringVar(&flags.Socks, "socks", "", "(client-only) SOCKS listen address")
	flag.StringVar(&flags.RedirTCP, "redir", "", "(client-only) redirect TCP from this address")
	flag.StringVar(&flags.RedirTCP6, "redir6", "", "(client-only) redirect TCP IPv6 from this address")
	flag.StringVar(&flags.TproxyTCP, "tproxytcp", "", "(client-only) TPROXY TCP listen address")
	flag.DurationVar(&config.UDPTimeout, "udptimeout", 120*time.Second, "UDP tunnel timeout")
	flag.Parse()

	if *listCiphers {
		println(strings.Join(core.ListCipher(), " "))
		return
	}

	if len(flags.Client) == 0 && len(flags.Server) == 0 {
		flag.Usage()
		return
	}

	if len(flags.Client) > 0 { // client mode
		if len(flags.UDPTun) > 0 { // use first server for UDP
			addr, cipher, password, err := parseURL(flags.Client[0])
			if err != nil {
				log.Fatal(err)
			}

			ciph, err := core.PickCipher(cipher, nil, password)
			if err != nil {
				log.Fatal(err)
			}
			for _, p := range flags.UDPTun {
				go udpLocal(p[0], addr, p[1], ciph.PacketConn)
			}
		}

		d, err := fastdialer(flags.Client...)
		if err != nil {
			log.Fatalf("failed to create dialer: %v", err)
		}

		if len(flags.TCPTun) > 0 {
			for _, p := range flags.TCPTun {
				go tcpTun(p[0], p[1], d)
			}
		}

		if flags.Socks != "" {
			go socksLocal(flags.Socks, d)
		}

		if flags.RedirTCP != "" {
			go redirLocal(flags.RedirTCP, d)
		}

		if flags.RedirTCP6 != "" {
			go redir6Local(flags.RedirTCP6, d)
		}

		if flags.TproxyTCP != "" {
			go tproxyTCP(flags.TproxyTCP, d)
		}
	}

	if len(flags.Server) > 0 { // server mode
		for _, each := range flags.Server {
			addr, cipher, password, err := parseURL(each)
			if err != nil {
				log.Fatal(err)
			}

			ciph, err := core.PickCipher(cipher, nil, password)
			if err != nil {
				log.Fatal(err)
			}

			go udpRemote(addr, ciph.PacketConn)
			go tcpRemote(addr, ciph.StreamConn)
		}
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
}

func parseURL(s string) (addr, cipher, password string, err error) {
	u, err := url.Parse(s)
	if err != nil {
		return
	}

	addr = u.Host
	if u.User != nil {
		cipher = u.User.Username()
		password, _ = u.User.Password()
	}
	return
}

type PairList [][2]string // key1=val1,key2=val2,...

func (l PairList) String() string {
	s := make([]string, len(l))
	for i, pair := range l {
		s[i] = pair[0] + "=" + pair[1]
	}
	return strings.Join(s, ",")
}
func (l *PairList) Set(s string) error {
	for _, item := range strings.Split(s, ",") {
		pair := strings.Split(item, "=")
		if len(pair) != 2 {
			return nil
		}
		*l = append(*l, [2]string{pair[0], pair[1]})
	}
	return nil
}

type SpaceSeparatedList []string

func (l SpaceSeparatedList) String() string { return strings.Join(l, " ") }
func (l *SpaceSeparatedList) Set(s string) error {
	*l = strings.Split(s, " ")
	return nil
}
