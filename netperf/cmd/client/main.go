package main

import (
	"log"
	"net"
	"os"
	"time"
)

const ADDR = ":12345"

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("Wrong num args: %v", os.Args)
	}
	log.Printf("Dialing!")
	start := time.Now()
	_, err := net.DialTimeout("tcp", os.Args[1]+ADDR, 1000000)
	if err != nil {
		log.Fatalf("Dial addr %v: err %v", ADDR, err)
	}
	log.Printf("Dialed!\nDial latency: %v", time.Since(start))
}
