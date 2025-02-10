package load_test

import (
	"flag"
	"fmt"
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

const (
	PORT_OFFSET = 10000
	SVC_NAME    = "mysvc"
)

var nClnt int
var minEntries int
var writeRPS int
var readRPS int
var dur time.Duration
var consulAddr string

func init() {
	flag.IntVar(&nClnt, "n_clnt", 1, "Number of clients")
	flag.IntVar(&minEntries, "min_entries", 2, "Minimum number of entries in the service registry at any one time")
	flag.DurationVar(&dur, "dur", 10*time.Second, "Duration")
	flag.StringVar(&consulAddr, "consul_addr", "127.0.0.1:8500", "Consul server addr")
	flag.IntVar(&writeRPS, "write_rps", 1, "Write requests per second")
	flag.IntVar(&readRPS, "read_rps", 1, "Read requests per second")
}

func populateRegistry(t *testing.T, c *consul.Client) error {
	db.DPrintf(db.TEST, "Populating service registry with %v entries", minEntries)
	for i := 0; i < minEntries; i++ {
		reg := &consul.AgentServiceRegistration{
			ID:      "min-svc-replica-" + strconv.Itoa(i),
			Name:    SVC_NAME,
			Port:    i,
			Address: "127.0.0.1",
		}
		if err := c.Agent().ServiceRegister(reg); err != nil {
			return err
		}
	}
	// Check that right number of services was registered
	svcs, err := c.Agent().ServicesWithFilter("Service==" + SVC_NAME)
	if !assert.Nil(t, err, "Err get first: %v", err) {
		return err
	}
	if !assert.Equal(t, minEntries, len(svcs), "Wrong num entries") {
		return fmt.Errorf("Wrong num entries %v != %v", minEntries, len(svcs))
	}
	db.DPrintf(db.TEST, "Done populating service registry with %v entries", minEntries)
	return nil
}

func TestCompile(t *testing.T) {
}

func TestRegisterOnly(t *testing.T) {
	cfg := consul.DefaultConfig()
	cfg.Address = consulAddr

	c, err := consul.NewClient(cfg)
	if !assert.Nil(t, err, "Err consul NewClient: %v", err) {
		return
	}

	db.DPrintf(db.TEST, "Created client")

	var idx atomic.Int32
	idx.Add(2)
	writeLG := loadgen.NewLoadGenerator(dur, writeRPS, func(r *rand.Rand) (time.Duration, bool) {
		i := int(idx.Add(1))
		id := "svc-replica-" + strconv.Itoa(i)
		reg := &consul.AgentServiceRegistration{
			ID:      id,
			Name:    SVC_NAME,
			Port:    PORT_OFFSET + i,
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

func TestRegisterDeregisterOnly(t *testing.T) {
	cfg := consul.DefaultConfig()
	cfg.Address = consulAddr

	c, err := consul.NewClient(cfg)
	if !assert.Nil(t, err, "Err consul NewClient: %v", err) {
		return
	}
	db.DPrintf(db.TEST, "Created client")

	if err := populateRegistry(t, c); !assert.Nil(t, err, "Err register first: %v", err) {
		return
	}

	var idx atomic.Int32
	idx.Add(1)
	writeLG := loadgen.NewLoadGenerator(dur, writeRPS, func(r *rand.Rand) (time.Duration, bool) {
		i := int(idx.Add(1))
		id := "svc-replica-" + strconv.Itoa(i/2)
		if i%2 == 0 {
			reg := &consul.AgentServiceRegistration{
				ID:      id,
				Name:    SVC_NAME,
				Port:    PORT_OFFSET + i,
				Address: "127.0.0.1",
			}
			err := c.Agent().ServiceRegister(reg)
			assert.Nil(t, err, "Err register: %v", err)
		} else {
			err := c.Agent().ServiceDeregister(id)
			assert.Nil(t, err, "Err deregister: %v", err)
		}
		return 0, false
	})
	db.DPrintf(db.TEST, "Calibrating write load generator")
	writeLG.Calibrate()
	db.DPrintf(db.TEST, "Done calibrating read load generator")
	writeLG.Run()
	writeLG.Stats()
	db.DPrintf(db.TEST, "Done")
}

func TestRegisterDeregisterGet(t *testing.T) {
	cfg := consul.DefaultConfig()
	cfg.Address = consulAddr

	c, err := consul.NewClient(cfg)
	if !assert.Nil(t, err, "Err consul NewClient: %v", err) {
		return
	}

	db.DPrintf(db.TEST, "Created client")

	if err := populateRegistry(t, c); !assert.Nil(t, err, "Err register first: %v", err) {
		return
	}

	var wg sync.WaitGroup

	var idx atomic.Int32
	idx.Add(1)
	writeLG := loadgen.NewLoadGenerator(dur, writeRPS, func(r *rand.Rand) (time.Duration, bool) {
		i := int(idx.Add(1))
		id := "svc-replica-" + strconv.Itoa(i/2)
		if i%2 == 0 {
			reg := &consul.AgentServiceRegistration{
				ID:      id,
				Name:    SVC_NAME,
				Port:    PORT_OFFSET + i,
				Address: "127.0.0.1",
			}
			err := c.Agent().ServiceRegister(reg)
			assert.Nil(t, err, "Err register: %v", err)
		} else {
			err := c.Agent().ServiceDeregister(id)
			assert.Nil(t, err, "Err deregister: %v", err)
		}
		return 0, false
	})
	readLG := loadgen.NewLoadGenerator(dur, readRPS, func(r *rand.Rand) (time.Duration, bool) {
		svcs, err := c.Agent().ServicesWithFilter("Service==" + SVC_NAME)
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
