# go-shadowsocks2

A fresh implementation of Shadowsocks in Go.

GoDoc at https://godoc.org/github.com/riobard/go-shadowsocks2/


## Features

- SOCKS5 proxy 
- Netfilter TCP redirect (IPv6 should work but not tested)
- UDP tunneling (e.g. tunneling DNS)
- TCP tunneling (e.g. benchmark with iperf3)


## Install

```sh
go install github.com/riobard/go-shadowsocks2
```


## Usage


Server

```sh
go-shadowsocks2 -s :8488 -cipher aes-128-gcm -key 1234567890abcdef1234567890abcdef -verbose
```


Client

```sh
go-shadowsocks2 -c [server_address]:8488 -cipher aes-128-gcm -key 1234567890abcdef1234567890abcdef \
    -socks :1080 -udptun :1080=8.8.8.8:53,:1081=8.8.4.4:53 -tcptun :1082=localhost:5201 -verbose
```

Keys are in hexdecimal format.