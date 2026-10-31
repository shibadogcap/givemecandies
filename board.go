package main

import (
	"encoding/json"
	"math/big"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
)

type sender struct {
	Name  string
	Total *big.Int
}

type entry struct {
	total *big.Int
	last  int64
}

type ipWin struct {
	n    int
	from int64
}

type board struct {
	mu    sync.Mutex
	m     map[string]*entry
	ip    map[string]*ipWin
	max   int
	top   atomic.Pointer[[]sender]
	dirty atomic.Bool

	newPerIP   int // new names allowed per window
	winSeconds int64
	ttlBelow   uint64 // totals under this expire after ttlSeconds of silence
	ttlSeconds int64
}

func newBoard(max int) *board {
	return &board{
		m: make(map[string]*entry), ip: make(map[string]*ipWin), max: max,
		newPerIP: 5, winSeconds: 60, ttlBelow: 1000, ttlSeconds: 600,
	}
}

// sanitize: full-width -> ascii, drop control/format/combining and invisible spaces,
// collapse runs of spaces, cap length.
func sanitizeName(s string) string {
	var b strings.Builder
	space := false
	n := 0
	for _, r := range s {
		switch {
		case r >= 0xFF01 && r <= 0xFF5E:
			r -= 0xFEE0
		case r == 0x3000:
			r = ' '
		}
		if r == ' ' || r == '\t' {
			space = true
			continue
		}
		if unicode.Is(unicode.Cf, r) || unicode.Is(unicode.Cc, r) || unicode.Is(unicode.Mn, r) ||
			r == 0x00AD || (r >= 0x200B && r <= 0x200F) || (r >= 0x2060 && r <= 0x206F) ||
			(r >= 0xFE00 && r <= 0xFE0F) || r == 0xFEFF {
			continue
		}
		if space && b.Len() > 0 {
			b.WriteRune(' ')
		}
		space = false
		b.WriteRune(r)
		n++
		if n >= 24 {
			break
		}
	}
	return strings.TrimSpace(b.String())
}

func (b *board) allowNew(ip string, now int64) bool {
	if b.newPerIP <= 0 {
		return true
	}
	w, ok := b.ip[ip]
	if !ok || now-w.from >= b.winSeconds {
		b.ip[ip] = &ipWin{n: 1, from: now}
		return true
	}
	if w.n >= b.newPerIP {
		return false
	}
	w.n++
	return true
}

func (b *board) add(ip, name string, n uint64) {
	if name == "" {
		return
	}
	now := time.Now().Unix()
	b.mu.Lock()
	e, ok := b.m[name]
	if ok {
		e.total.Add(e.total, new(big.Int).SetUint64(n))
		e.last = now
		b.mu.Unlock()
		b.dirty.Store(true)
		return
	}
	// a brand new name: bounded by per-IP budget and by the table cap
	if !b.allowNew(ip, now) || len(b.m) >= b.max {
		b.mu.Unlock()
		return
	}
	b.m[name] = &entry{total: new(big.Int).SetUint64(n), last: now}
	b.mu.Unlock()
	b.dirty.Store(true)
}

func (b *board) save(path string) {
	b.mu.Lock()
	type saved struct {
		Total string `json:"t"`
		Last  int64  `json:"l"`
	}
	out := make(map[string]saved, len(b.m))
	for k, e := range b.m {
		out[k] = saved{Total: e.total.String(), Last: e.last}
	}
	b.mu.Unlock()
	data, err := json.Marshal(out)
	if err != nil {
		return
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return
	}
	os.Rename(tmp, path)
}

func (b *board) load(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	type saved struct {
		Total string `json:"t"`
		Last  int64  `json:"l"`
	}
	var in map[string]saved
	if json.Unmarshal(data, &in) != nil {
		return
	}
	b.mu.Lock()
	for k, v := range in {
		if len(b.m) >= b.max {
			break
		}
		n, ok := new(big.Int).SetString(v.Total, 10)
		if !ok {
			continue
		}
		b.m[k] = &entry{total: n, last: v.Last}
	}
	b.mu.Unlock()
	b.dirty.Store(true)
}

func (b *board) rebuild() {
	now := time.Now().Unix()
	b.mu.Lock()
	thr := new(big.Int).SetUint64(b.ttlBelow)
	for k, e := range b.m {
		if e.total.Cmp(thr) < 0 && now-e.last > b.ttlSeconds {
			delete(b.m, k)
		}
	}
	if len(b.ip) > 4096 {
		for k, w := range b.ip {
			if now-w.from > 300 {
				delete(b.ip, k)
			}
		}
	}
	list := make([]sender, 0, len(b.m))
	for k, e := range b.m {
		list = append(list, sender{k, new(big.Int).Set(e.total)})
	}
	b.mu.Unlock()
	sort.Slice(list, func(i, j int) bool {
		if c := list[i].Total.Cmp(list[j].Total); c != 0 {
			return c > 0
		}
		return list[i].Name < list[j].Name
	})
	if len(list) > 100 {
		list = list[:100]
	}
	b.top.Store(&list)
}
