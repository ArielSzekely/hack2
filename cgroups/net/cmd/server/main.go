package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/dustin/go-humanize"
)

const (
	BUF_SZ     = 8192
	PRINT_FREQ = 5 * time.Second
)

var name string

// Read from a connection until it closes, and print the read rate.
func read(conn net.Conn) {
	log.Printf("[%v] reading", name)
	nByte := 0
	t := time.Now()
	b := make([]byte, BUF_SZ)
	for {
		n, err := io.ReadFull(conn, b)
		if err != nil {
			log.Fatalf("[%v] Err read buf: %v", name, err)
		}
		if n != len(b) {
			log.Fatalf("[%v] Err short read: %v", name, n)
		}
		nByte += n
		if time.Since(t) > PRINT_FREQ {
			elapsed := time.Since(t)
			tpt := uint64(float64(nByte) / float64(elapsed/time.Second))
			log.Printf("[%v] read throughput: %v/s", name, humanize.Bytes(tpt))
			nByte = 0
			t = time.Now()
		}
	}
}

func main() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)
	if len(os.Args) != 3 {
		log.Fatalf("Usage: %v id addr\nArgs: %v", os.Args[0], os.Args)
	}
	name = fmt.Sprintf("%s:%v", os.Args[1], os.Getpid())
	addr := os.Args[2]
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("[%v] Error Listen: %v", name, err)
	}
	conn, err := l.Accept()
	if err != nil {
		log.Fatalf("[%v] Error Accept: %v", name, err)
	}
	read(conn)
}
