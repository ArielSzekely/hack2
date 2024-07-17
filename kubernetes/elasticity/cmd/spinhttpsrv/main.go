package main

import (
	"fmt"
	"html"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const (
	ADDR = ":8080"
)

type request struct {
	replicaID  int
	id         string
	start      time.Time
	dur        time.Duration
	rps        float64
	qlat       time.Duration
	qlenStart  int
	qlenFinish int
	isDone     bool
	doneC      chan bool
}

func newRequest(replicaID int, id string, dur time.Duration) *request {
	return &request{
		replicaID: replicaID,
		start:     time.Now(),
		id:        id,
		dur:       dur,
		doneC:     make(chan bool),
	}
}

func (r *request) waitUntilDone() {
	<-r.doneC
}

func (r *request) done() {
	r.isDone = true
	r.doneC <- true
}

func (r *request) String() string {
	if r.isDone {
		return fmt.Sprintf("&{ id:%v replica_id:%03d rps:%0.2f dur:%v q_lat:%v e2e_lat:%v qlen_start:%v qlen_finish:%v }", r.id, r.replicaID, r.rps, r.dur.Round(10*time.Millisecond), r.qlat.Round(10*time.Millisecond), time.Since(r.start).Round(10*time.Millisecond), r.qlenStart, r.qlenFinish)
	} else {
		return fmt.Sprintf("&{ id:%v dur:%v }", r.id, r.dur)
	}
}

type srv struct {
	replicaID int
	nslots    int
	q         chan *request
	qlen      atomic.Int64
	mu        sync.Mutex
	reqCntIdx int
	reqCnts   []int // Slice to estimate request rate
}

func newSrv(nslots int) *srv {
	s := &srv{
		replicaID: rand.Intn(1000),
		q:         make(chan *request),
		nslots:    nslots,
		reqCnts:   make([]int, 2),
	}
	for i := 0; i < nslots; i++ {
		go s.worker()
	}
	return s
}

func getFirstSleep(nMSToSleep float64) time.Time {
	return time.Now().Add(time.Millisecond * time.Duration(rand.Int63n(int64(nMSToSleep))))
}

// Spin for a specified duration, consuming spinFrac * time.Second of the CPU
func spin(dur time.Duration, spinFrac float64) {
	log.Printf("Spin for %v frac %v", dur, spinFrac)
	defer log.Printf("Spin done for %v frac %v", dur, spinFrac)

	// Spin for spinFrac of each second
	nMSToSpin := spinFrac * 1000.0
	timeToSpin := time.Millisecond * time.Duration(nMSToSpin)
	// Sleep for the remaining 1 - spinFrac of each second
	nMSToSleep := 1000.0 - nMSToSpin
	timeToSleep := time.Millisecond * time.Duration(nMSToSleep)
	// The first time, sleep randomly somewhere between [now, now+timeToSleep)
	nextSleep := getFirstSleep(nMSToSleep)
	log.Printf("timeToSleep:%v timeToSpin:%v firstSleep:%v", timeToSleep, timeToSpin, nextSleep)
	start := time.Now()
	n := 1
	i := 1
	for {
		i++
		if i%10000 == 0 && time.Since(start) >= dur {
			break
		}
		if time.Now().After(nextSleep) {
			// Sleep to allow other spin workers to use the CPU, but don't sleep
			// past the expected completion time
			time.Sleep(min(timeToSleep, time.Until(start.Add(dur))))
			// After another timeToSpin period, sleep for timeToSleep
			nextSleep = time.Now().Add(timeToSpin)
		}
		n *= i*i + 2
	}
}

func (s *srv) maybeClearRequestRateL(idx int) {
	if s.reqCntIdx != idx {
		log.Printf("Switching request rate buckets buckets %v -> %v cnt %v", s.reqCntIdx, idx, s.reqCnts[idx])
		// Switch the bucket into which we accumulate, and clear the bucket
		s.reqCntIdx = idx
		s.reqCnts[idx] = 0
	}
}

func (s *srv) updateRequestRate() {
	t := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Switch idx every 5 seconds
	idx := (int(t.UnixMilli()/1000) % 10) / 5
	s.maybeClearRequestRateL(idx)
	s.reqCnts[idx]++
}

func (s *srv) getRequestRate() float64 {
	t := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Switch idx every 5 seconds
	idx := (int(t.UnixMilli()/1000) % 10) / 5
	s.maybeClearRequestRateL(idx)
	// Return the per-second rate from the previous bucket
	return float64(s.reqCnts[idx^1]) / 5.0
}

// Handle a spin request
func (s *srv) spinHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Please specify request ID", http.StatusBadRequest)
		return
	}
	dstr := r.URL.Query().Get("dur")
	if dstr == "" {
		http.Error(w, "Please specify spin duration", http.StatusBadRequest)
		return
	}
	dur, err := time.ParseDuration(dstr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing spin duration: %v", err), http.StatusBadRequest)
		return
	}
	s.updateRequestRate()
	req := newRequest(s.replicaID, id, dur)
	log.Printf("Spin request %v", req)
	s.enqueue(req)
	req.waitUntilDone()
	log.Printf("Request done %v", req)
	fmt.Fprintf(w, "Done spin %v\n", html.EscapeString(req.String()))
}

// Enqueue a new request
func (s *srv) enqueue(r *request) {
	s.qlen.Add(1)
	r.qlenStart = int(s.qlen.Load())
	log.Printf("Enqueue request %v qlen %v", r, r.qlenStart)
	s.q <- r
}

func (s *srv) worker() {
	for {
		r := <-s.q
		s.qlen.Add(-1)
		log.Printf("Dequeue request %v qlen %v", r, s.qlen.Load())
		r.qlat = time.Since(r.start)
		spin(r.dur, 1.0/float64(s.nslots))
		r.rps = s.getRequestRate()
		r.qlenFinish = int(s.qlen.Load())
		r.done()
	}
}

func main() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)
	if len(os.Args) != 2 {
		log.Fatalf("Usage: %v nslots\nArgs: %v", os.Args[0], os.Args)
	}
	nslots, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatalf("Err parse nslots: %v", err)
	}
	s := newSrv(nslots)
	http.HandleFunc("/spin", s.spinHandler)
	log.Printf("Start server at %v", ADDR)
	log.Fatal(http.ListenAndServe(ADDR, nil))
}
