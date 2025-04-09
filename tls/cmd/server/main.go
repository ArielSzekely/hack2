package main

import (
	"crypto/tls"
	"log"
	"net"
)

func init() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)
}

const ADDR = ":8088"

func main() {
	cer, err := tls.LoadX509KeyPair("keys/server.crt", "keys/server.key")
	if err != nil {
		log.Fatalf("Err load cert: %v", err)
	}
	config := &tls.Config{
		Certificates: []tls.Certificate{cer},
	}
	l, err := tls.Listen("tcp", ADDR, config)
	if err != nil {
		log.Fatalf("Err listen on addr [%v]: %v", ADDR, err)
	}
	defer l.Close()
	log.Printf("Listening for clients!")
	var conn net.Conn
	for {
		conn, err = l.Accept()
		if err != nil {
			log.Fatalf("Err accept [%v]: %v", ADDR, err)
		}
		log.Printf("Conn received!")
		if err := conn.(*tls.Conn).Handshake(); err != nil {
			log.Fatalf("Err TLS handshake: %v", err)
		}
	}
}
