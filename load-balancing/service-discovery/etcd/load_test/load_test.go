package load_test

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.etcd.io/etcd/client/v3"

	"sigmaos/benchmarks/loadgen"
	db "sigmaos/debug"
)

const (
	PORT_OFFSET = 10000
	SVC_NAME    = "mysvc"
	SVC2_NAME   = "notmysvc"
)

var nClnt int
var nWatchers int
var minEntries int
var unrelatedEntries int
var writeRPS int
var readRPS int
var dur time.Duration
var etcdAddr string

func init() {
	flag.IntVar(&nClnt, "n_clnt", 1, "Number of clients")
	flag.IntVar(&nWatchers, "n_watchers", 1, "Number of watchers")
	flag.IntVar(&minEntries, "min_entries", 2, "Minimum number of entries in the service registry at any one time")
	flag.IntVar(&unrelatedEntries, "n_other_entries", 0, "Entries belonging to unrelated services")
	flag.DurationVar(&dur, "dur", 10*time.Second, "Duration")
	flag.StringVar(&etcdAddr, "etcd_addr", "127.0.0.1:4379", "Etcd port")
	flag.IntVar(&writeRPS, "write_rps", 1, "Write requests per second")
	flag.IntVar(&readRPS, "read_rps", 1, "Read requests per second")
}

func populateRegistry(t *testing.T, c *clientv3.Client, svcname string, ninstances int) error {
	db.DPrintf(db.TEST, "Populating service registry with %v entries of svc %v", ninstances, svcname)
	for i := 0; i < ninstances; i++ {
		id := "min-svc-replica-" + strconv.Itoa(i)
		addr := "127.0.0.1:XXXX"
		key := filepath.Join(svcname, id)
		if _, err := c.Put(context.TODO(), key, addr); err != nil {
			return err
		}
	}
	// Check that right number of services was registered
	kvs, err := c.Get(context.TODO(), svcname, clientv3.WithPrefix())
	if !assert.Nil(t, err, "Err get check initial: %v", err) {
		return err
	}
	if !assert.Equal(t, ninstances, len(kvs.Kvs), "Wrong num entries initial") {
		return fmt.Errorf("Wrong num entries %v != %v", ninstances, len(kvs.Kvs))
	}
	db.DPrintf(db.TEST, "Done populating service registry with %v entries", ninstances)
	return nil
}

func checkRegistry(t *testing.T, c *clientv3.Client) {
	db.DPrintf(db.TEST, "Checking final registry contents")
	// Check that right number of services was registered
	kvs, err := c.Get(context.TODO(), SVC_NAME, clientv3.WithPrefix())
	if !assert.Nil(t, err, "Err get check: %v", err) {
		return
	}
	if !assert.True(t, minEntries <= len(kvs.Kvs), "Wrong num entries final") {
		return
	}
	db.DPrintf(db.TEST, "Done checking final registry contents")
}

func TestCompile(t *testing.T) {
}

func TestRegisterOnly(t *testing.T) {
	c, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{etcdAddr},
		DialTimeout: 2 * time.Second,
	})
	if !assert.Nil(t, err, "Err etcd NewClient: %v", err) {
		return
	}

	db.DPrintf(db.TEST, "Created client")

	var idx atomic.Int32
	idx.Add(2)
	writeLG := loadgen.NewLoadGenerator(dur, writeRPS, func(r *rand.Rand) (time.Duration, bool) {
		i := int(idx.Add(1))
		id := "svc-replica-" + strconv.Itoa(i)
		addr := "127.0.0.1:" + strconv.Itoa(PORT_OFFSET+i)
		key := filepath.Join(SVC_NAME, id)
		_, err := c.Put(context.TODO(), key, addr)
		assert.Nil(t, err, "Err Put: %v", err)
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
	c, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{etcdAddr},
		DialTimeout: 2 * time.Second,
	})
	if !assert.Nil(t, err, "Err etcd NewClient: %v", err) {
		return
	}

	db.DPrintf(db.TEST, "Created client")

	if err := populateRegistry(t, c, SVC_NAME, minEntries); !assert.Nil(t, err, "Err populate registry: %v", err) {
		return
	}
	if err := populateRegistry(t, c, SVC2_NAME, unrelatedEntries); !assert.Nil(t, err, "Err populate registry unrelated: %v", err) {
		return
	}
	defer checkRegistry(t, c)

	var idx atomic.Int32
	idx.Add(1)
	writeLG := loadgen.NewLoadGenerator(dur, writeRPS, func(r *rand.Rand) (time.Duration, bool) {
		i := int(idx.Add(1))
		id := "svc-replica-" + strconv.Itoa(i)
		addr := "127.0.0.1:" + strconv.Itoa(PORT_OFFSET+i)
		key := filepath.Join(SVC_NAME, id)
		_, err := c.Put(context.TODO(), key, addr)
		assert.Nil(t, err, "Err Put: %v", err)
		resp, err := c.Delete(context.TODO(), key)
		assert.Nil(t, err, "Err Delete: %v", err)
		assert.Equal(t, int(resp.Deleted), 1, "Err wrong number deleted: %v != 1", resp.Deleted)
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
	c, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{etcdAddr},
		DialTimeout: 2 * time.Second,
	})
	if !assert.Nil(t, err, "Err etcd NewClient: %v", err) {
		return
	}

	db.DPrintf(db.TEST, "Created client")

	if err := populateRegistry(t, c, SVC_NAME, minEntries); !assert.Nil(t, err, "Err populate registry: %v", err) {
		return
	}
	if err := populateRegistry(t, c, SVC2_NAME, unrelatedEntries); !assert.Nil(t, err, "Err populate registry unrelated: %v", err) {
		return
	}
	defer checkRegistry(t, c)

	var wg sync.WaitGroup

	var idx atomic.Int32
	idx.Add(1)
	writeLG := loadgen.NewLoadGenerator(dur, writeRPS, func(r *rand.Rand) (time.Duration, bool) {
		i := int(idx.Add(1))
		id := "svc-replica-" + strconv.Itoa(i)
		addr := "127.0.0.1:" + strconv.Itoa(PORT_OFFSET+i)
		key := filepath.Join(SVC_NAME, id)
		_, err := c.Put(context.TODO(), key, addr)
		assert.Nil(t, err, "Err Put: %v", err)
		resp, err := c.Delete(context.TODO(), key)
		assert.Nil(t, err, "Err Delete: %v", err)
		assert.Equal(t, int(resp.Deleted), 1, "Err wrong number deleted: %v != 1", resp.Deleted)
		return 0, false
	})
	readLG := loadgen.NewLoadGenerator(dur, readRPS, func(r *rand.Rand) (time.Duration, bool) {
		kvs, err := c.Get(context.TODO(), SVC_NAME, clientv3.WithPrefix())
		assert.Nil(t, err, "Err get check initial: %v", err)
		assert.Nil(t, err, "Err get: %v", err)
		assert.True(t, len(kvs.Kvs) > 1, "No services returned")
		assert.True(t, len(kvs.Kvs) >= minEntries, "Too few services returned")
		return 0, false
	})
	db.DPrintf(db.TEST, "Calibrating write load generator")
	writeLG.Calibrate()
	db.DPrintf(db.TEST, "Done calibrating write load generator")
	db.DPrintf(db.TEST, "Calibrating read load generator")
	readLG.Calibrate()
	db.DPrintf(db.TEST, "Done calibrating read load generator")
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

func watchManyWriters(t *testing.T, wc clientv3.WatchChan, wg *sync.WaitGroup) {
	resp := <-wc
	ncreate := 0
	for _, e := range resp.Events {
		if e.IsCreate() {
			ncreate++
		}
	}
	if ncreate > 0 {
		// Consume the next creation event, and ack that it has been consumed
		wg.Done()
	}
}

func TestRegisterDeregisterWatchManyWriters(t *testing.T) {
	c, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{etcdAddr},
		DialTimeout: 2 * time.Second,
	})
	if !assert.Nil(t, err, "Err etcd NewClient: %v", err) {
		return
	}

	db.DPrintf(db.TEST, "Created client")

	if err := populateRegistry(t, c, SVC_NAME, minEntries); !assert.Nil(t, err, "Err populate registry: %v", err) {
		return
	}
	if err := populateRegistry(t, c, SVC2_NAME, unrelatedEntries); !assert.Nil(t, err, "Err populate registry unrelated: %v", err) {
		return
	}
	defer checkRegistry(t, c)

	var wg sync.WaitGroup

	clnts := make([]*clientv3.Client, nWatchers)

	for i := 0; i < len(clnts); i++ {
		// Create a new client for this watcher
		c, err := clientv3.New(clientv3.Config{
			Endpoints:   []string{etcdAddr},
			DialTimeout: 2 * time.Second,
		})
		if !assert.Nil(t, err, "Err etcd NewClient: %v", err) {
			return
		}
		clnts[i] = c
	}

	var idx atomic.Int32
	idx.Add(1)
	writeLG := loadgen.NewLoadGenerator(dur, writeRPS, func(r *rand.Rand) (time.Duration, bool) {
		var watchers sync.WaitGroup
		watchers.Add(nWatchers)

		// Start a bunch of goroutines to watch for a change
		for _, c := range clnts {
			wc := c.Watch(context.TODO(), SVC_NAME, clientv3.WithPrefix())
			go watchManyWriters(t, wc, &watchers)
		}

		i := int(idx.Add(1))
		id := "svc-replica-" + strconv.Itoa(i)
		addr := "127.0.0.1:" + strconv.Itoa(PORT_OFFSET+i)
		key := filepath.Join(SVC_NAME, id)

		// Note the put timestamp
		start := time.Now()

		// Put a value, releasing watchers
		_, err := c.Put(context.TODO(), key, addr)
		assert.Nil(t, err, "Err Put: %v", err)

		// Wait for the watchers to all be notified of the change, and measure how
		// long it took
		watchers.Wait()
		elapsed := time.Since(start)

		resp, err := c.Delete(context.TODO(), key)
		assert.Nil(t, err, "Err Delete: %v", err)
		assert.Equal(t, int(resp.Deleted), 1, "Err wrong number deleted: %v != 1", resp.Deleted)
		return elapsed, true
	})
	db.DPrintf(db.TEST, "Calibrating write load generator")
	writeLG.Calibrate()
	db.DPrintf(db.TEST, "Done calibrating write load generator")
	// Start write load-generator
	wg.Add(1)
	go func() {
		defer wg.Done()
		writeLG.Run()
	}()
	// Start read load-generator
	wg.Wait()
	db.DPrintf(db.TEST, "Write load-generator stats:")
	writeLG.Stats()
	db.DPrintf(db.TEST, "Read load-generator stats:")
	db.DPrintf(db.TEST, "Stop watchers")

	db.DPrintf(db.TEST, "Done")
}

func watcherOneWriter(t *testing.T, wc clientv3.WatchChan, done chan bool, wg *sync.WaitGroup) {
	for {
		select {
		case resp := <-wc:
			ncreate := 0
			for _, e := range resp.Events {
				if e.IsCreate() {
					ncreate++
				}
			}
			if ncreate > 0 {
				// Consume the next creation event, and ack that it has been consumed
				wg.Done()
				assert.Equal(t, ncreate, 1, "Wrong num create events")
			}
		case <-done:
			return
		}
	}
}

func TestRegisterDeregisterWatchOneWriter(t *testing.T) {
	c, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{etcdAddr},
		DialTimeout: 2 * time.Second,
	})
	if !assert.Nil(t, err, "Err etcd NewClient: %v", err) {
		return
	}

	db.DPrintf(db.TEST, "Created client")

	if err := populateRegistry(t, c, SVC_NAME, minEntries); !assert.Nil(t, err, "Err populate registry: %v", err) {
		return
	}
	if err := populateRegistry(t, c, SVC2_NAME, unrelatedEntries); !assert.Nil(t, err, "Err populate registry unrelated: %v", err) {
		return
	}
	defer checkRegistry(t, c)

	var watchers sync.WaitGroup
	var wg sync.WaitGroup
	done := make(chan bool)

	for i := 0; i < nWatchers; i++ {
		// Create a new client for this watcher
		c, err := clientv3.New(clientv3.Config{
			Endpoints:   []string{etcdAddr},
			DialTimeout: 2 * time.Second,
		})
		if !assert.Nil(t, err, "Err etcd NewClient: %v", err) {
			return
		}
		wc := c.Watch(context.TODO(), SVC_NAME, clientv3.WithPrefix())
		go watcherOneWriter(t, wc, done, &watchers)
	}

	var idx atomic.Int32
	idx.Add(1)
	writeLG := loadgen.NewLoadGenerator(dur, writeRPS, func(r *rand.Rand) (time.Duration, bool) {
		watchers.Add(nWatchers)
		i := int(idx.Add(1))
		id := "svc-replica-" + strconv.Itoa(i)
		addr := "127.0.0.1:" + strconv.Itoa(PORT_OFFSET+i)
		key := filepath.Join(SVC_NAME, id)

		// Note the put timestamp
		start := time.Now()

		// Put a value, releasing watchers
		_, err := c.Put(context.TODO(), key, addr)
		assert.Nil(t, err, "Err Put: %v", err)

		// Wait for the watchers to all be notified of the change, and measure how
		// long it took
		watchers.Wait()
		elapsed := time.Since(start)

		resp, err := c.Delete(context.TODO(), key)
		assert.Nil(t, err, "Err Delete: %v", err)
		assert.Equal(t, int(resp.Deleted), 1, "Err wrong number deleted: %v != 1", resp.Deleted)
		return elapsed, true
	})
	db.DPrintf(db.TEST, "Calibrating write load generator")
	writeLG.Calibrate()
	db.DPrintf(db.TEST, "Done calibrating write load generator")
	// Start write load-generator
	wg.Add(1)
	go func() {
		defer wg.Done()
		writeLG.Run()
	}()
	// Start read load-generator
	wg.Wait()
	db.DPrintf(db.TEST, "Write load-generator stats:")
	writeLG.Stats()
	db.DPrintf(db.TEST, "Read load-generator stats:")
	db.DPrintf(db.TEST, "Stop watchers")
	for i := 0; i < nWatchers; i++ {
		done <- true
	}
	db.DPrintf(db.TEST, "Done")
}
