package main

import (
	"sync"
	"time"
)

// requests per second for the last 60s, the peak since boot, and how many
// distinct IPs have been seen in the last minute. a flood shows up as peak up
// and ips up; one person mashing shows peak up and ips flat.
type meter struct {
	mu   sync.Mutex
	sec  [60]uint64
	at   int64
	peak uint64
	ips  map[string]int64
}

func newMeter() *meter { return &meter{ips: make(map[string]int64)} }

func (m *meter) hit(ip string) {
	now := time.Now().Unix()
	m.mu.Lock()
	if now != m.at {
		m.sec[now%60] = 0
		m.at = now
	}
	m.sec[now%60]++
	if n := m.sec[now%60]; n > m.peak {
		m.peak = n
	}
	m.ips[ip] = now
	m.mu.Unlock()
}

func (m *meter) snapshot() (rps, peak uint64, ips int) {
	now := time.Now().Unix()
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.ips) > 4096 {
		for k, t := range m.ips {
			if now-t > 60 {
				delete(m.ips, k)
			}
		}
	}
	if m.at == now || m.at == now-1 {
		rps = m.sec[m.at%60]
	}
	if m.at < now-1 {
		rps = 0
	}
	return rps, m.peak, len(m.ips)
}
