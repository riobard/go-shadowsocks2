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


### Server

Start a server listening on port 8848 using `aes-128-gcm` AEAD cipher with a 128-bit key in hexdecimal.


```sh
go-shadowsocks2 -s :8488 -cipher aes-128-gcm -key 1234567890abcdef1234567890abcdef -verbose
```



### Client

Start a client connecting to the above server. The client listens on port 1080 for incoming SOCKS5 
connections, and tunnels UDP packets received on port 1080 and port 1081 to 8.8.8.8:53 and 8.8.4.4:53 
respectively. The client also tunnels TCP connection to port 1082 to port 5201 on localhost, which is
used to proxy iperf3 for benchmarking.

```sh
go-shadowsocks2 -c [server_address]:8488 -cipher aes-128-gcm -key 1234567890abcdef1234567890abcdef \
    -socks :1080 -udptun :1080=8.8.8.8:53,:1081=8.8.4.4:53 -tcptun :1082=localhost:5201 -verbose
```
