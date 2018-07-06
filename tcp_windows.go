package main

func redirLocal(addr string, d Dialer)  { panic("TCP redirect not supported") }
func redir6Local(addr string, d Dialer) { panic("TCP6 redirect not supported") }
func tproxyTCP(addr string, d Dialer)   { panic("TPROXY TCP not supported") }
