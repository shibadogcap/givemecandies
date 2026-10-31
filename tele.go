package main

import (
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"
)

// paths that scanners always try. answering 200 keeps them coming, and every
// hit is counted so the dashboard shows what the internet is looking for.
var baitPaths = map[string]bool{
	"/wp-login.php": true, "/wp-admin": true, "/xmlrpc.php": true,
	"/.env": true, "/.git/config": true, "/config.json": true,
	"/phpmyadmin": true, "/admin": true, "/server-status": true,
	"/actuator/health": true, "/api/v1/ping": true, "/shell.php": true,
}

func isBait(p string) bool { return baitPaths[p] }

type recorder struct {
	http.ResponseWriter
	status int
}

func (r *recorder) WriteHeader(c int) {
	r.status = c
	r.ResponseWriter.WriteHeader(c)
}

// aggregates only: no per-request disk log, so a 4k req/s flood costs nothing
// on the disk. everything here is bounded.
type tele struct {
	mu    sync.Mutex
	ip    map[string]int
	ua    map[string]int
	path  map[string]int
	ctry  map[string]int
	st    map[int]int
	bait  int64
	total int64
}

func newTele() *tele {
	t := &tele{ip: map[string]int{}, ua: map[string]int{}, path: map[string]int{},
		ctry: map[string]int{}, st: map[int]int{}}
	go func() {
		for range time.Tick(30 * time.Second) {
			t.mu.Lock()
			t.trimLocked(t.ip, 4000)
			t.trimLocked(t.ua, 400)
			t.trimLocked(t.path, 400)
			t.mu.Unlock()
		}
	}()
	return t
}

func (t *tele) trimLocked(m map[string]int, keep int) {
	if len(m) <= keep {
		return
	}
	type kv struct {
		k string
		v int
	}
	all := make([]kv, 0, len(m))
	for k, v := range m {
		all = append(all, kv{k, v})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].v > all[j].v })
	for _, e := range all[keep:] {
		delete(m, e.k)
	}
}

func (t *tele) record(r *http.Request, status int, bait bool) {
	ua := r.UserAgent()
	if len(ua) > 90 {
		ua = ua[:90]
	}
	if ua == "" {
		ua = "(none)"
	}
	p := r.URL.Path
	if len(p) > 60 {
		p = p[:60]
	}
	t.mu.Lock()
	t.total++
	t.st[status]++
	t.ip[clientIP(r)]++
	t.ua[ua]++
	t.path[p]++
	if c := r.Header.Get("CF-IPCountry"); c != "" {
		t.ctry[c]++
	}
	if bait {
		t.bait++
	}
	t.mu.Unlock()
}

type kv struct {
	K string `json:"k"`
	N int    `json:"n"`
}

func (t *tele) top(m map[string]int, n int) []kv {
	type pair struct {
		k string
		v int
	}
	all := make([]pair, 0, len(m))
	for k, v := range m {
		all = append(all, pair{k, v})
	}
	sort.Slice(all, func(i, j int) bool { return all[i].v > all[j].v })
	out := []kv{}
	for i, e := range all {
		if i >= n {
			break
		}
		out = append(out, kv{e.k, e.v})
	}
	return out
}

func (t *tele) snapshot() map[string]any {
	t.mu.Lock()
	defer t.mu.Unlock()
	st := map[string]int{}
	for k, v := range t.st {
		st[strconv.Itoa(k)] = v
	}
	ct := map[string]int{}
	for k, v := range t.ctry {
		ct[k] = v
	}
	return map[string]any{
		"total":    t.total,
		"bait":     t.bait,
		"ips":      len(t.ip),
		"top_ip":   t.top(t.ip, 10),
		"top_ua":   t.top(t.ua, 8),
		"top_path": t.top(t.path, 12),
		"country":  ct,
		"status":   st,
	}
}
