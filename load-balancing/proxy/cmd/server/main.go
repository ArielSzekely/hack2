package main

import (
	"log"
	"net"
	"time"

	"proxy/rpc"
)

const ADDR = ":8088"

func main() {
	l, err := net.Listen("tcp", ADDR)
	if err != nil {
		log.Fatalf("Err listen on addr [%v]: %v", ADDR, err)
	}
	log.Printf("Listening for clients!")
	conn, err := l.Accept()
	if err != nil {
		log.Fatalf("Err accept [%v]: %v", ADDR, err)
	}
	log.Printf("Conn received!")
	nrpc, err := rpc.GetNRPC(conn)
	if err != nil {
		log.Fatalf("Err GetNRPC [%v]: %v", ADDR, err)
	}
	log.Printf("Server expecting %v RPCs", nrpc)
	start := time.Now()
	n := 0
	addr := make([]byte, 100)
	b := make([]byte, 10*1024*1024)
	for i := 0; i < nrpc; i++ {
		addrNByte, nbyte, err := rpc.ServeRPC(addr, conn, b)
		if err != nil {
			log.Fatalf("Err ServeRPC: %v", err)
		}
		n += nbyte + addrNByte
	}
	dur := time.Since(start)
	bufsz := n / nrpc
	log.Printf("Served %v %v-KB RPCs in %0.3fs", nrpc, bufsz/1024, dur.Seconds())
	log.Printf("RPC throughput: %0.3fReq/s %0.3fMB/s", float64(nrpc)/dur.Seconds(), float64(n)/float64(1024*1024)/dur.Seconds())
}
