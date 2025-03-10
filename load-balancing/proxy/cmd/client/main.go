package main

import (
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"proxy/pipe"
	"proxy/rpc"
)

func init() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)
}

func main() {
	if len(os.Args) != 5 {
		log.Fatalf("Usage: %v <srvaddr> <bufsz> <nrpc> <noproxy|netproxy|pipeproxy>", os.Args[0])
	}
	srvAddr := os.Args[1]
	srvAddrB := []byte(os.Args[1])
	bufsz, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Fatalf("Can't parse bufsz: %v", err)
	}
	nrpc, err := strconv.Atoi(os.Args[3])
	if err != nil {
		log.Fatalf("Can't parse nrpc: %v", err)
	}
	proxytype := os.Args[4]
	log.Printf("Dialing!")
	var conn net.Conn
	switch proxytype {
	case "noproxy":
		conn, err = net.Dial("tcp", srvAddr)
		if err != nil {
			log.Fatalf("Dial srv addr %v: err %v", srvAddr, err)
		}
	case "netproxy":
		conn, err = net.Dial("tcp", srvAddr)
		if err != nil {
			log.Fatalf("Dial proxy addr %v: err %v", srvAddr, err)
		}
	case "pipeproxy":
		conn, err = net.Dial("unix", pipe.PATH)
		if err != nil {
			log.Fatalf("Dial pipe %v: err %v", pipe.PATH, err)
		}
	default:
		log.Fatalf("Unknown proxy type: %v", proxytype)
	}
	log.Printf("Dialed!")
	log.Printf("Client sending %v RPCs", nrpc)
	// Tell the server how many RPCs to expect
	if err := rpc.InformNRPC(conn, nrpc); err != nil {
		log.Fatalf("Err InformNRPC: %v", err)
	}
	n := 0
	start := time.Now()
	rtt := float64(0)
	b := make([]byte, bufsz)
	for i := 0; i < nrpc; i++ {
		rtStart := time.Now()
		if err := rpc.ClientRPC(srvAddrB, conn, b); err != nil {
			log.Fatalf("Err ClientRPC: %v", err)
		}
		rtt += time.Since(rtStart).Seconds()
		n += bufsz + len(srvAddrB)
	}
	dur := time.Since(start)
	log.Printf("Performed %v %v-KB RPCs in %0.3fs", nrpc, bufsz/1024, dur.Seconds())
	log.Printf("RPC throughput: %0.3fReq/s %0.3fMB/s", float64(nrpc)/dur.Seconds(), float64(bufsz*nrpc)/float64(1024*1024)/dur.Seconds())
	log.Printf("Average RPC RTT: %0.3fms", 1000.0*rtt/float64(nrpc))
}
