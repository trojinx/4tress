package main

import (
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRateLimiterResetsAfterWindow(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	for request := 0; request < 2; request++ {
		allowed, retryAfter := limiter.allowAt("203.0.113.7", start)
		if !allowed || retryAfter != 0 {
			t.Fatalf("request %d: allowed=%v retryAfter=%s, want allowed", request+1, allowed, retryAfter)
		}
	}

	allowed, retryAfter := limiter.allowAt("203.0.113.7", start)
	if allowed {
		t.Fatal("third request was allowed; want rate limit")
	}
	if retryAfter != time.Minute {
		t.Fatalf("retryAfter=%s, want %s", retryAfter, time.Minute)
	}

	allowed, _ = limiter.allowAt("203.0.113.7", start.Add(time.Minute))
	if !allowed {
		t.Fatal("request after window reset was blocked")
	}
}

func TestRateLimiterBoundsTrackedClients(t *testing.T) {
	limiter := NewRateLimiter(1, time.Hour)
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

	for client := 0; client <= maximumTrackedClients; client++ {
		limiter.allowAt(strconv.Itoa(client), start)
	}

	if got := len(limiter.clients); got > maximumTrackedClients {
		t.Fatalf("tracked clients=%d, limit=%d", got, maximumTrackedClients)
	}
}

func TestRateLimiterIsSafeForConcurrentHandlers(t *testing.T) {
	limiter := NewRateLimiter(5, time.Minute)
	var allowed atomic.Int32
	var waitGroup sync.WaitGroup

	for request := 0; request < 100; request++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			if permitted, _ := limiter.Allow("203.0.113.7"); permitted {
				allowed.Add(1)
			}
		}()
	}
	waitGroup.Wait()

	if got := allowed.Load(); got != 5 {
		t.Fatalf("allowed %d concurrent requests, want 5", got)
	}
}
