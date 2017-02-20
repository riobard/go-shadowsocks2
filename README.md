# go-shadowsocks2

A fresh implementation of Shadowsocks in Go.

GoDoc at https://godoc.org/github.com/shadowsocks/go-shadowsocks2/


## Features

- SOCKS5 proxy 
- Support for Netfilter TCP redirect (IPv6 should work but not tested)
- UDP tunneling (e.g. relay DNS packets)
- TCP tunneling (e.g. benchmark with iperf3)


## Install

```sh
go get -u -v github.com/shadowsocks/go-shadowsocks2
```


## Basic Usage


### Server

Start a server listening on port 8488 using `aes-128-gcm` AEAD cipher with password `your-password`.

```sh
go-shadowsocks2 -s ss://aes-128-gcm:your-password@:8488 -verbose
```


### Client

Start a client connecting to the above server. The client listens on port 1080 for incoming SOCKS5 
connections, and tunnels UDP packets received on port 1080 and port 1081 to 8.8.8.8:53 and 8.8.4.4:53 
respectively. 

```sh
go-shadowsocks2 -c ss://aes-128-gcm:your-password@[server_address]:8488 \
    -socks :1080 -udptun :1080=8.8.8.8:53,:1081=8.8.4.4:53 -verbose
```

Replace `[server_address]` with the server's public address.


## Advanced Usage


### Use random keys instead of passwords

A random key is almost always better than a password. Generate a base64url-encoded 16-byte random key

```sh
go-shadowsocks2 -keygen 16
```

Start a server listening on port 8848 using `aes-128-gcm` AEAD cipher with the key generated above.

```sh
go-shadowsocks2 -s :8488 -cipher aes-128-gcm -key k5yEIX5ciUDpkpdtvZm7zQ== -verbose
```

And the corresponding client to connect to it.

```sh
go-shadowsocks2 -c [server_address]:8488 -cipher aes-128-gcm -key k5yEIX5ciUDpkpdtvZm7zQ== -verbose
```


### Netfilter TCP redirect (Linux only)

The client offers `-redir` and `-redir6` (for IPv6) options to handle TCP connections 
redirected by Netfilter on Linux. The feature works similar to `ss-redir` from `shadowsocks-libev`.


Start a client listening on port 1082 for redirected TCP connections and port 1083 for redirected
TCP IPv6 connections.

```sh
go-shadowsocks2 -c [server_address]:8488 -cipher aes-128-gcm -key k5yEIX5ciUDpkpdtvZm7zQ== \
    -redir :1082 -redir6 :1083
```


### TCP tunneling

The client offers `-tcptun [local_addr]:[local_port]=[remote_addr]:[remote_port]` option to tunnel TCP.
For example it can be used to proxy iperf3 for benchmarking.

Start iperf3 on the same machine with the server.

```sh
iperf3 -s
```

By default iperf3 listens on port 5201.

Start a client on the same machine with the server. The client listens on port 1090 for incoming connections
and tunnels to localhost:5201 where iperf3 is listening.

```sh
go-shadowsocks2 -c [server_address]:8488 -cipher aes-128-gcm -key k5yEIX5ciUDpkpdtvZm7zQ== \
    -tcptun :1090=localhost:5201
```

Start iperf3 client to connect to the tunneld port instead

```sh
iperf3 -c localhost -p 1090
```


## TODO

- Test coverage
- Binary releases for common platforms



## Design Principles

The code base strives to

- be idiomatic Go and well organized;
- use fewer external dependences as reasonably possible;
- only include proven modern ciphers;

