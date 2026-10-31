package main

import (
	"sync"
	"time"
)

type bucket struct {
	tokens float64
	last   time.Time
}

type limiter struct {
	mu    sync.Mutex
	m     map[string]*bucket
	rate  float64
	burst float64
}

func newLimiter(rate, burst float64) *limiter {
	if rate <= 0 {
		return nil
	}
	l := &limiter{m: make(map[string]*bucket), rate: rate, burst: burst}
	go func() {
		for range time.Tick(30 * time.Second) {
			cut := time.Now().Add(-5 * time.Minute)
			l.mu.Lock()
			for k, b := range l.m {
				if b.last.Before(cut) {
					delete(l.m, k)
				}
			}
			l.mu.Unlock()
		}
	}()
	return l
}

func (l *limiter) allow(ip string, cost float64) bool {
	if l == nil {
		return true
	}
	now := time.Now()
	l.mu.Lock()
	b, ok := l.m[ip]
	if !ok {
		b = &bucket{tokens: l.burst, last: now}
		l.m[ip] = b
	}
	b.tokens += now.Sub(b.last).Seconds() * l.rate
	if b.tokens > l.burst {
		b.tokens = l.burst
	}
	b.last = now
	if b.tokens < cost {
		l.mu.Unlock()
		return false
	}
	b.tokens -= cost
	l.mu.Unlock()
	return true
}
