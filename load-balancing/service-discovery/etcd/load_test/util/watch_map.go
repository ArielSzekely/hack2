package util

import (
	"sync"
)

type WatchMap struct {
	sync.Mutex
	wgs map[string]*sync.WaitGroup
}

func NewWatchMap() *WatchMap {
	return &WatchMap{
		wgs: make(map[string]*sync.WaitGroup),
	}
}

func (wm *WatchMap) Add(key string, n int) *sync.WaitGroup {
	var wg sync.WaitGroup

	wm.Lock()
	wm.wgs[key] = &wg
	wm.Unlock()

	wg.Add(n)

	return &wg
}

func (wm *WatchMap) Get(key string) *sync.WaitGroup {
	wm.Lock()
	defer wm.Unlock()

	return wm.wgs[key]
}
