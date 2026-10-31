package main

import (
	"math/big"
	"sync"
)

// A plain uint64 wraps at 1.8e19, which this workload can reach in about
// 40 minutes. big.Int has no ceiling and is still ~100ns per add.
type counter struct {
	mu sync.Mutex
	v  *big.Int
}

func newCounter() *counter { return &counter{v: new(big.Int)} }

func (c *counter) add(n uint64) {
	c.mu.Lock()
	c.v.Add(c.v, new(big.Int).SetUint64(n))
	c.mu.Unlock()
}

func (c *counter) get() *big.Int {
	c.mu.Lock()
	out := new(big.Int).Set(c.v)
	c.mu.Unlock()
	return out
}

func (c *counter) setString(v string) {
	if n, ok := new(big.Int).SetString(v, 10); ok {
		c.mu.Lock()
		c.v.Set(n)
		c.mu.Unlock()
	}
}

func (c *counter) String() string { return c.get().String() }
