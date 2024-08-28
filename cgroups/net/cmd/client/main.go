package main

import (
	"fmt"
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

// Write to a connection until it closes, and print the write rate.
func write(conn net.Conn) {
	log.Printf("[%v] writing", name)
	nByte := 0
	t := time.Now()
	b := make([]byte, BUF_SZ)
	for {
		n, err := conn.Write(b)
		if err != nil {
			log.Fatalf("[%v] Err writ buf: %v", name, err)
		}
		if n != len(b) {
			log.Fatalf("[%v] Err short write: %v", name, n)
		}
		nByte += n
		if time.Since(t) > PRINT_FREQ {
			elapsed := time.Since(t)
			tpt := uint64(float64(nByte) / float64(elapsed/time.Second))
			log.Printf("[%v] write throughput: %v/s", name, humanize.Bytes(tpt))
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
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatalf("[%v] Error Dial: %v", name, err)
	}
	write(conn)
}
