package main

import (
	"bufio"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// The hot path never touches the disk. Once a second the pending delta is
// appended as one line and fsynced: 3k req/s becomes 1 write/s.
type store struct {
	mu      sync.Mutex
	f       *os.File
	total   *counter
	last    *big.Int
	written *big.Int
	flushes atomic.Uint64
}

func openStore(path string, total *counter) (*store, error) {
	if err := os.MkdirAll(path[:strings.LastIndex(path, "/")], 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	s := &store{f: f, total: total, last: new(big.Int), written: new(big.Int)}
	// the counter is monotonic: the highest total in the log is the current one
	if rf, err := os.Open(path); err == nil {
		sc := bufio.NewScanner(rf)
		sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
		sum := new(big.Int)
		for sc.Scan() {
			parts := strings.Fields(sc.Text())
			if len(parts) == 3 {
				if v, ok := new(big.Int).SetString(parts[1], 10); ok {
					sum.Add(sum, v)
				}
			}
		}
		rf.Close()
		if sum.Sign() > 0 {
			total.setString(sum.String())
		}
	}
	s.last.Set(total.get())
	return s, nil
}

func (s *store) tick(now time.Time) {
	cur := s.total.get()
	if cur.Cmp(s.last) == 0 {
		return
	}
	delta := new(big.Int).Sub(cur, s.last)
	s.last.Set(cur)
	line := fmt.Sprintf("%d %s %s\n", now.Unix(), delta.String(), cur.String())
	s.mu.Lock()
	_, err := s.f.WriteString(line)
	if err == nil {
		err = s.f.Sync()
	}
	s.mu.Unlock()
	if err != nil {
		log.Printf("flush: %v", err)
		return
	}
	s.flushes.Add(1)
	s.mu.Lock()
	s.written.Add(s.written, delta)
	s.mu.Unlock()
}

func (s *store) start() {
	go func() {
		for t := range time.Tick(time.Second) {
			s.tick(t)
		}
	}()
}
