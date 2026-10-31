package main

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type server struct {
	cfg   cfg
	total *counter
	met   *meter
	tele  *tele

	reqs     atomic.Uint64
	rejected atomic.Uint64
	limited  atomic.Uint64
	badow    atomic.Uint64

	lim   *limiter
	idem  *idem
	store *store
	board *board
	start time.Time

	// last 5 seconds of accepted candy, for the rate display
	histMu sync.Mutex
	hist   [5]uint64
	histAt int64
}

func (s *server) rollHist(now int64) {
	s.histMu.Lock()
	if now != s.histAt {
		s.hist[now%5] = 0
		s.histAt = now
	}
	s.histMu.Unlock()
}

func (s *server) addHist(n uint64) {
	now := time.Now().Unix()
	s.histMu.Lock()
	if now != s.histAt {
		s.hist[now%5] = 0
		s.histAt = now
	}
	s.hist[now%5] += n
	s.histMu.Unlock()
}

func (s *server) rate5() uint64 {
	now := time.Now().Unix()
	s.histMu.Lock()
	defer s.histMu.Unlock()
	var sum uint64
	for i := int64(0); i < 5; i++ {
		if now-i == s.histAt || (s.histAt > now-i && now-i > s.histAt-5) {
		}
		sum += s.hist[(now-i)%5]
	}
	return sum / 5
}

func clientIP(r *http.Request) string {
	if v := r.Header.Get("CF-Connecting-IP"); v != "" {
		return v
	}
	h, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return h
}

type voteReq struct {
	Name   string `json:"name"`
	Amount uint64 `json:"amount"`
	RID    string `json:"rid"`
	Nonce  string `json:"nonce"`
	Sol    string `json:"sol"`
}

func (s *server) handleVote(w http.ResponseWriter, r *http.Request) {
	s.reqs.Add(1)
	body, err := io.ReadAll(io.LimitReader(r.Body, 512))
	if err != nil {
		s.badow.Add(1)
		http.Error(w, "bad", 400)
		return
	}
	var v voteReq
	if err := json.Unmarshal(body, &v); err != nil || v.Amount < 1 || v.Amount > s.cfg.maxAmount || len(v.RID) < 8 || len(v.RID) > 64 {
		s.badow.Add(1)
		http.Error(w, "bad request", 400)
		return
	}

	if s.idem.seen(v.RID) {
		w.WriteHeader(204)
		return
	}

	if !s.lim.allow(clientIP(r), float64(v.Amount)) {
		s.limited.Add(1)
		s.rejected.Add(1)
		// no body: a 429 here must cost almost nothing
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(429)
		return
	}

	s.total.add(v.Amount)
	s.addHist(v.Amount)
	if v.Name != "" {
		s.board.add(clientIP(r), sanitizeName(v.Name), v.Amount)
	}
	w.WriteHeader(204)
}

func (s *server) handleStats(w http.ResponseWriter, r *http.Request) {
	rps, peak, ips := s.met.snapshot()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(map[string]any{
		"total":    s.total.String(),
		"rate":     s.rate5(),
		"requests": s.reqs.Load(),
		"limited":  s.limited.Load(),
		"bad":      s.badow.Load(),
		"uptime":   int64(time.Since(s.start).Seconds()),
		"written":  s.store.written.String(),
		"flushes":  s.store.flushes.Load(),
		"senders":  s.board.size(),
		"rps":      rps,
		"peak":     peak,
		"ips":      ips,
	})
}

func (s *server) handleRanking(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	top := s.board.top.Load()
	out := []map[string]any{}
	if top != nil {
		for i, e := range *top {
			if i >= 100 {
				break
			}
			out = append(out, map[string]any{"name": e.Name, "total": e.Total.String()})
		}
	}
	json.NewEncoder(w).Encode(map[string]any{"senders": out, "count": s.board.size()})
}

func (b *board) size() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.m)
}

// SO_REUSEPORT on linux; not exported by syscall on every arch.
const soReusePort = 0xf
