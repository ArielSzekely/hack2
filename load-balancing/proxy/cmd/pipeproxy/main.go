package main

import (
	"log"
	"net"
	"os"

	"proxy/pipe"
	"proxy/rpc"
)

func init() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)
}

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Usage: %v <srv-ip>", os.Args[0])
	}
	srvAddr := os.Args[1]
	// Remove any old pipes
	os.Remove(pipe.PATH)
	// Create the pipe proxy socket
	l, err := net.Listen("unix", pipe.PATH)
	if err != nil {
		log.Fatalf("Err listen on pipe [%v]: %v", pipe.PATH, err)
	}
	if err := os.Chmod(pipe.PATH, 0777); err != nil {
		log.Fatalf("Err chmod sigmasocket: %v", err)
	}
	log.Printf("Listening for clients!")
	log.Printf("Connecting to server %v", srvAddr)
	srvConn, err := net.Dial("tcp", srvAddr)
	if err != nil {
		log.Fatalf("Err dial [%v]: %v", srvAddr, err)
	}
	log.Printf("Connected to server %v", srvAddr)
	cliConn, err := l.Accept()
	if err != nil {
		log.Fatalf("Err accept [%v]: %v", pipe.PATH, err)
	}
	log.Printf("Conn received!")
	addr := make([]byte, 100)
	b := make([]byte, 10*1024*1024)
	nrpc, err := rpc.GetNRPC(cliConn)
	if err != nil {
		log.Fatalf("Err GetNRPC [%v]: %v", pipe.PATH, err)
	}
	log.Printf("Proxy proxying %v RPCs", nrpc)
	// Tell the server how many RPCs to expect
	if err := rpc.InformNRPC(srvConn, nrpc); err != nil {
		log.Fatalf("Err InformNRPC: %v", err)
	}
	for i := 0; i < nrpc; i++ {
		if err := rpc.ProxyRPC(addr, cliConn, srvConn, b); err != nil {
			log.Fatalf("Err ReadFrameInto: %v", err)
		}
	}
	// Remove any old pipes
	os.Remove(pipe.PATH)
}
