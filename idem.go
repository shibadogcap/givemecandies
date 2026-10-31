package main

import (
	"sync"
	"time"
)

type idem struct {
	mu   sync.Mutex
	m    map[string]int64
	max  int
	last int64
}

func newIdem(max int) *idem {
	if max <= 0 {
		return nil
	}
	i := &idem{m: make(map[string]int64), max: max}
	go func() {
		for range time.Tick(10 * time.Second) {
			cut := time.Now().Add(-10 * time.Minute).UnixNano()
			i.mu.Lock()
			if len(i.m) > 0 {
				for k, t := range i.m {
					if t < cut {
						delete(i.m, k)
					}
				}
			}
			i.mu.Unlock()
		}
	}()
	return i
}

func (i *idem) seen(key string) bool {
	if i == nil {
		return false
	}
	now := time.Now().UnixNano()
	i.mu.Lock()
	defer i.mu.Unlock()
	if _, ok := i.m[key]; ok {
		return true
	}
	if len(i.m) >= i.max {
		return false // shed when full, never grow unbounded
	}
	i.m[key] = now
	return false
}
