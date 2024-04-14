package main

import (
	"log"
	"net"
)

const ADDR = ":12345"

func main() {
	l, err := net.Listen("tcp", ADDR)
	if err != nil {
		log.Fatalf("Listen on addr %v: err %v", ADDR, err)
	}
	log.Printf("Listening for clients!")
	_, err = l.Accept()
	if err != nil {
		log.Fatalf("Accept err %v", ADDR, err)
	}
	log.Printf("Conn received!")
}
