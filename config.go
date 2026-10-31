package main

import (
	"os"
	"strconv"
)

type cfg struct {
	maxAmount   uint64
	limitRate   float64
	limitBurst  float64
	idempotency bool
}

func envF(k string, d float64) float64 {
	if v := os.Getenv(k); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return d
}

func envS(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func envI(k string, d int) int {
	if v := os.Getenv(k); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return d
}
