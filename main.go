package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	addr := envS("CANDY_ADDR", ":8080")
	data := envS("CANDY_DATA", "/var/lib/candy")
	logPath := data + "/candy.log"
	senderPath := data + "/senders.json"

	c := cfg{
		maxAmount:   uint64(envF("CANDY_MAX_AMOUNT", 2e12)),
		limitRate:   envF("CANDY_LIMIT_RATE", 0),
		limitBurst:  envF("CANDY_LIMIT_BURST", 2000),
		idempotency: envI("CANDY_IDEMPOTENCY", 1) == 1,
	}
	s := &server{
		cfg:   c,
		lim:   newLimiter(c.limitRate, c.limitBurst),
		idem:  newIdem(map[bool]int{true: 2000000, false: 0}[c.idempotency]),
		start: time.Now(),
	}
	s.total = newCounter()
	s.met = newMeter()
	s.tele = newTele()
	st, err := openStore(logPath, s.total)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	s.store = st
	st.start()
	s.board = newBoard(50000)
	s.board.load(senderPath)
	go func() {
		last := time.Now()
		for range time.Tick(5 * time.Second) {
			s.board.rebuild()
			if time.Since(last) > 30*time.Second {
				s.board.save(senderPath)
				last = time.Now()
			}
		}
	}()
	defer st.f.Close()

	mux := http.NewServeMux()
	idx, _ := page.ReadFile("web/index.html")
	var gz bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&gz, gzip.BestCompression)
	zw.Write(idx)
	zw.Close()
	pageGz := gz.Bytes()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=60")
		w.Header().Set("Vary", "Accept-Encoding")
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			w.Header().Set("Content-Encoding", "gzip")
			w.Write(pageGz)
			return
		}
		w.Write(idx)
	})
	mux.HandleFunc("/c", s.handleVote)
	mux.HandleFunc("/api/stats", s.handleStats)
	mux.HandleFunc("/api/attack", func(w http.ResponseWriter, r *http.Request) {
		rps, peak, ips := s.met.snapshot()
		out := s.tele.snapshot()
		out["rps"] = rps
		out["peak"] = peak
		out["ips60"] = ips
		out["total_candy"] = s.total.String()
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		json.NewEncoder(w).Encode(out)
	})
	for p := range baitPaths {
		p := p
		mux.HandleFunc(p, func(w http.ResponseWriter, r *http.Request) {
			// every scanner hit is a candy. symbolic, but it counts.
			s.total.add(1)
			s.addHist(1)
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte("ok"))
		})
	}
	atk, _ := page.ReadFile("web/attack.html")
	mux.HandleFunc("/attack", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("X-Robots-Tag", "noindex")
		w.Write(atk)
	})
	mux.HandleFunc("/api/ranking", s.handleRanking)
	fav, _ := page.ReadFile("web/favicon.svg")
	mux.HandleFunc("/favicon.svg", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Write(fav)
	})
	icon, _ := page.ReadFile("web/icon.webp")
	mux.HandleFunc("/icon.webp", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/webp")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Write(icon)
	})
	og, _ := page.ReadFile("web/og.png")
	mux.HandleFunc("/og.png", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Write(og)
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })

	hit := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.met.hit(clientIP(r))
			rec := &recorder{ResponseWriter: w, status: 200}
			next.ServeHTTP(rec, r)
			s.tele.record(r, rec.status, isBait(r.URL.Path))
		})
	}

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           hit(mux),
		ReadHeaderTimeout: 2 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1024,
	}
	lc := net.ListenConfig{Control: func(network, address string, rc syscall.RawConn) error {
		var serr error
		rc.Control(func(fd uintptr) {
			serr = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, soReusePort, 1)
		})
		return serr
	}}
	ln, err := lc.Listen(context.Background(), "tcp", addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	log.Printf("candy listening %s max=%d rate=%.0f/s burst=%.0f idem=%v",
		addr, c.maxAmount, c.limitRate, c.limitBurst, c.idempotency)

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGTERM, syscall.SIGINT)
		<-sig
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()
	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
