package main

import (
	"flag"
	"net/url"
	"strings"
	"time"
)

var config struct {
	Verbose    bool
	UDP        bool
	UDPTimeout time.Duration
	Client     SpaceSeparatedList
	Server     SpaceSeparatedList
	TCPTun     PairList
	UDPTun     PairList
	Socks      string
	RedirTCP   string
	TproxyTCP  string
}

func init() {
	flag.BoolVar(&config.Verbose, "verbose", false, "verbose mode")
	flag.Var(&config.Server, "s", "server listen url")
	flag.Var(&config.Client, "c", "client connect url")
	flag.Var(&config.TCPTun, "tcptun", "(client-only) TCP tunnel (laddr1=raddr1,laddr2=raddr2,...)")
	flag.Var(&config.UDPTun, "udptun", "(client-only) UDP tunnel (laddr1=raddr1,laddr2=raddr2,...)")
	flag.StringVar(&config.Socks, "socks", "", "(client-only) SOCKS listen address")
	flag.StringVar(&config.RedirTCP, "redir", "", "(client-only) redirect TCP from this address")
	flag.StringVar(&config.TproxyTCP, "tproxytcp", "", "(Linux client-only) TPROXY TCP listen address")
	flag.BoolVar(&config.UDP, "udp", false, "(server-only) UDP support")
	flag.DurationVar(&config.UDPTimeout, "udptimeout", 120*time.Second, "UDP tunnel timeout")
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
