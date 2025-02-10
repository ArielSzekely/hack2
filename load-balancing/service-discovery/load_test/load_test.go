package load_test

import (
	"flag"
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	consul "github.com/hashicorp/consul/api"
	"github.com/stretchr/testify/assert"

	"sigmaos/benchmarks/loadgen"
	db "sigmaos/debug"
)

var nClnt int
var writeRPS int
var readRPS int
var dur time.Duration
var consulAddr string

func init() {
	flag.IntVar(&nClnt, "n_clnt", 1, "Number of clients")
	flag.DurationVar(&dur, "dur", 10*time.Second, "Duration")
	flag.StringVar(&consulAddr, "consul_addr", "127.0.0.1:8500", "Consul server addr")
	flag.IntVar(&writeRPS, "write_rps", 1, "Write requests per second")
	flag.IntVar(&readRPS, "read_rps", 1, "Read requests per second")
}

func TestCompile(t *testing.T) {
}

func TestRegister(t *testing.T) {
	cfg := consul.DefaultConfig()
	cfg.Address = consulAddr

	c, err := consul.NewClient(cfg)
	if !assert.Nil(t, err, "Err consul NewClient: %v", err) {
		return
	}

	db.DPrintf(db.TEST, "Created client")

	var idx atomic.Int32
	writeLG := loadgen.NewLoadGenerator(dur, writeRPS, func(r *rand.Rand) (time.Duration, bool) {
		i := int(idx.Add(1))
		id := "svc-replica-" + strconv.Itoa(i)
		reg := &consul.AgentServiceRegistration{
			ID:      id,
			Name:    "svc",
			Port:    i,
			Address: "127.0.0.1",
		}
		err := c.Agent().ServiceRegister(reg)
		assert.Nil(t, err, "Err register: %v", err)
		return 0, false
	})
	db.DPrintf(db.TEST, "Calibrating write load generator")
	writeLG.Calibrate()
	db.DPrintf(db.TEST, "Done calibrating read load generator")
	writeLG.Run()
	writeLG.Stats()
	db.DPrintf(db.TEST, "Done")
}

func TestRegisterAndGet(t *testing.T) {
	cfg := consul.DefaultConfig()
	cfg.Address = consulAddr

	c, err := consul.NewClient(cfg)
	if !assert.Nil(t, err, "Err consul NewClient: %v", err) {
		return
	}

	db.DPrintf(db.TEST, "Created client")

	reg := &consul.AgentServiceRegistration{
		ID:      "svc-replica-0",
		Name:    "svc",
		Port:    0,
		Address: "127.0.0.1",
	}
	err = c.Agent().ServiceRegister(reg)
	if !assert.Nil(t, err, "Err register first: %v", err) {
		return
	}
	db.DPrintf(db.TEST, "Registered a service endpoint")

	svcs, err := c.Agent().ServicesWithFilter("Service==svc")
	if !assert.Nil(t, err, "Err get first: %v", err) {
		return
	}
	if !assert.Equal(t, 1, len(svcs), "Wrong num services") {
		return
	}
	db.DPrintf(db.TEST, "Read a service endpoint")

	var wg sync.WaitGroup

	var idx atomic.Int32
	idx.Add(1)
	writeLG := loadgen.NewLoadGenerator(dur, writeRPS, func(r *rand.Rand) (time.Duration, bool) {
		i := int(idx.Add(1))
		id := "svc-replica-" + strconv.Itoa(i)
		reg := &consul.AgentServiceRegistration{
			ID:      id,
			Name:    "svc",
			Port:    i,
			Address: "127.0.0.1",
		}
		err := c.Agent().ServiceRegister(reg)
		assert.Nil(t, err, "Err register: %v", err)
		return 0, false
	})
	readLG := loadgen.NewLoadGenerator(dur, readRPS, func(r *rand.Rand) (time.Duration, bool) {
		svcs, err := c.Agent().ServicesWithFilter("Service==svc")
		assert.Nil(t, err, "Err get: %v", err)
		assert.True(t, len(svcs) > 1, "No services returned")
		return 0, false
	})
	db.DPrintf(db.TEST, "Calibrating write load generator")
	writeLG.Calibrate()
	db.DPrintf(db.TEST, "Done calibrating write load generator")
	db.DPrintf(db.TEST, "Calibrating read load generator")
	readLG.Calibrate()
	db.DPrintf(db.TEST, "Done calibrating write load generator")
	// Start write load-generator
	wg.Add(1)
	go func() {
		defer wg.Done()
		writeLG.Run()
	}()
	// Start read load-generator
	wg.Add(1)
	go func() {
		defer wg.Done()
		readLG.Run()
	}()
	wg.Wait()
	db.DPrintf(db.TEST, "Write load-generator stats:")
	writeLG.Stats()
	db.DPrintf(db.TEST, "Read load-generator stats:")
	readLG.Stats()
	db.DPrintf(db.TEST, "Done")
}
