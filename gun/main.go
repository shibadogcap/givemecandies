package main

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

func rid() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b) + "-" + fmt.Sprint(time.Now().UnixNano())
}

func main() {
	url := flag.String("url", "", "target base url")
	conc := flag.Int("c", 64, "concurrency")
	dur := flag.Int("d", 20, "seconds")
	amount := flag.Uint64("amount", 1, "candies per request")
	name := flag.String("name", "", "sender name")
	flag.Parse()
	if *url == "" {
		fmt.Fprintln(os.Stderr, "need -url")
		os.Exit(1)
	}

	var ok, fail int64
	codes := sync.Map{}
	var mu sync.Mutex
	lat := make([]time.Duration, 0, 1<<20)

	deadline := time.Now().Add(time.Duration(*dur) * time.Second)
	var wg sync.WaitGroup
	t0 := time.Now()

	for i := 0; i < *conc; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tr := &http.Transport{
				MaxIdleConnsPerHost: 1,
				DisableCompression:  true,
				DialContext:         (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
			}
			cl := &http.Client{Transport: tr, Timeout: 10 * time.Second}
			buf := &bytes.Buffer{}
			for time.Now().Before(deadline) {
				buf.Reset()
				if *name != "" {
					fmt.Fprintf(buf, `{"name":%q,"amount":%d,"rid":"%s"}`, *name, *amount, rid())
				} else {
					fmt.Fprintf(buf, `{"amount":%d,"rid":"%s"}`, *amount, rid())
				}
				st := time.Now()
				req, _ := http.NewRequest("POST", *url+"/c", bytes.NewReader(buf.Bytes()))
				req.Header.Set("Content-Type", "text/plain")
				resp, err := cl.Do(req)
				d := time.Since(st)
				if err != nil {
					atomic.AddInt64(&fail, 1)
					codes.Store("err", true)
					continue
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				if resp.StatusCode == 204 || resp.StatusCode == 200 {
					atomic.AddInt64(&ok, 1)
				} else {
					atomic.AddInt64(&fail, 1)
				}
				codes.Store(resp.StatusCode, true)
				mu.Lock()
				if len(lat) < 1<<20 {
					lat = append(lat, d)
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	el := time.Since(t0).Seconds()
	sort.Slice(lat, func(i, j int) bool { return lat[i] < lat[j] })
	p := func(q float64) time.Duration {
		if len(lat) == 0 {
			return 0
		}
		return lat[int(float64(len(lat)-1)*q)]
	}
	var st []string
	codes.Range(func(k, v any) bool { st = append(st, fmt.Sprint(k)); return true })
	fmt.Printf("conc=%d dur=%.0fs amount=%d\n", *conc, el, *amount)
	fmt.Printf("ok=%d fail=%d  %.0f req/s\n", ok, fail, float64(ok+fail)/el)
	fmt.Printf("lat p50=%dus p90=%dus p99=%dus max=%dus\n",
		p(.5).Microseconds(), p(.9).Microseconds(), p(.99).Microseconds(), p(1).Microseconds())
	fmt.Printf("candies sent = %d  status=%v\n", uint64(ok)**amount, st)
}
