package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	ADDR                     = ":8080"
	CLUSTER_MONITOR_INTERVAL = 5 * time.Second
)

type Tlb string

const (
	LB_RR           Tlb = "rr"
	LB_ClntSideQLen     = "clntsideqlen"
	LB_Random           = "random"
)

func parseLBType(s string) Tlb {
	switch s {
	case string(LB_RR):
		return LB_RR
	case string(LB_Random):
		return LB_Random
	default:
		log.Fatalf("Unknown LB type: %v", s)
		return Tlb("unknown")
	}
}

type LB struct {
	mu              sync.Mutex
	clnt            *http.Client
	t               Tlb
	podIPs          []string
	outstandingReqs map[string]int
	idx             int
}

func newLB(lbType Tlb, k8sclnt *kubernetes.Clientset) *LB {
	clnt := &http.Client{
		Timeout:   20 * time.Minute,
		Transport: http.DefaultTransport,
	}
	clnt.Transport.(*http.Transport).MaxIdleConnsPerHost = 100000
	clnt.Transport.(*http.Transport).MaxIdleConns = 100000
	lb := &LB{
		t:               lbType,
		clnt:            clnt,
		outstandingReqs: make(map[string]int),
	}
	go lb.monitorPods(k8sclnt)
	return lb
}

func (lb *LB) updatePodIPs(ips map[string]bool) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	// Remove any IPs which are no longer part of the service
	for i := 0; i < len(lb.podIPs); i++ {
		// If IP not present in set of current ready IPs, remove it
		if !ips[lb.podIPs[i]] {
			lb.podIPs = append(lb.podIPs[:i], lb.podIPs[i+1:]...)
			i--
			log.Printf("Remove pod IP from backends: %v", lb.podIPs[i])
		} else {
			// Otherwise, remove this IP from the set so we don't double-add it later.
			delete(ips, lb.podIPs[i])
		}
	}
	// Add any IPs which were missing
	for ip := range ips {
		log.Printf("Add pod IP to backends: %v", ip)
		lb.podIPs = append(lb.podIPs, ip)
		addr := ip + ":8080"
		lb.outstandingReqs[addr] = 0
	}
}

func (lb *LB) monitorPods(k8sclnt *kubernetes.Clientset) {
	for {
		ep, err := k8sclnt.CoreV1().Endpoints("default").Get(context.TODO(), "spinhttp", metav1.GetOptions{})
		if err != nil {
			log.Fatalf("Err get endpoints: %v", err)
		}
		ips := make(map[string]bool)
		for _, epSubset := range ep.Subsets {
			for _, a := range epSubset.Addresses {
				ips[a.IP] = true
			}
		}
		lb.updatePodIPs(ips)
		time.Sleep(CLUSTER_MONITOR_INTERVAL)
	}
}

func (lb *LB) getBackendPodAddrRR() string {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	if len(lb.podIPs) == 0 {
		log.Fatalf("No backend pod IPs available")
	}
	ip := lb.podIPs[lb.idx%len(lb.podIPs)]
	lb.idx++
	return ip + ":8080"
}

func (lb *LB) getBackendPodAddrClntSideQLen() string {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	// Find the backend pod with the shortes qlen
	addr := ""
	minQLen := -1
	for a, qlen := range lb.outstandingReqs {
		if qlen < minQLen || minQLen == -1 {
			minQLen = qlen
			addr = a
		}
	}
	lb.outstandingReqs[addr]++
	return addr
}

func (lb *LB) getBackendPodAddr() string {
	switch lb.t {
	case LB_RR:
		return lb.getBackendPodAddrRR()
	case LB_ClntSideQLen:
		return lb.getBackendPodAddrClntSideQLen()
	default:
		log.Fatalf("Unimplemented LB of type %v", lb.t)
		return "unimplemented-lb-type"
	}
}

// TODO: better naming scheme?
// Note that a request to addr has completed
func (lb *LB) putBackendPodAddr(addr string) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	lb.outstandingReqs[addr]--
}

// Load-balance requests across backend replicas, and forward the reply back to
// the caller.
func (lb *LB) lbHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("LB handler, URL:%v", r.URL)
	addr := lb.getBackendPodAddr()
	defer lb.putBackendPodAddr(addr)
	proxyURL := "http://" + addr + "/spin?" + r.URL.RawQuery
	log.Printf("proxy URL: %v", proxyURL)
	resp, err := lb.clnt.Get(proxyURL)
	if err != nil {
		http.Error(w, fmt.Sprintf("Err proxied request: %v", err), http.StatusBadRequest)
		return
	}
	defer resp.Body.Close()
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Err copy body proxied request: %v", err), http.StatusBadRequest)
		return
	}
}

func main() {
	log.SetFlags(log.Ltime | log.Lmicroseconds)
	if len(os.Args) != 2 {
		log.Fatalf("Usage: %v type\nArgs: %v", os.Args[0], os.Args)
	}
	lbType := Tlb(os.Args[1])
	// creates the in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf(err.Error())
	}
	// creates the k8s client
	k8sclnt, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf(err.Error())
	}
	lb := newLB(lbType, k8sclnt)
	http.HandleFunc("/spin", lb.lbHandler)
	log.Printf("Start server at %v", ADDR)
	log.Fatal(http.ListenAndServe(ADDR, nil))
}
