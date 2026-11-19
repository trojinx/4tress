package main

import (
	"sync"
	"time"
)

type rateLimitEntry struct {
	count     int
	resetTime time.Time
}

// RateLimiter uses a fixed time window for each client IP. It is safe for
// concurrent HTTP handlers.
type RateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	clients map[string]rateLimitEntry
}

const maximumTrackedClients = 10_000

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		limit:   limit,
		window:  window,
		clients: make(map[string]rateLimitEntry),
	}
}

// Allow reports whether a request is permitted and, if it is blocked, how
// long the client must wait before its window resets.
func (limiter *RateLimiter) Allow(clientIP string) (bool, time.Duration) {
	return limiter.allowAt(clientIP, time.Now())
}

func (limiter *RateLimiter) allowAt(clientIP string, now time.Time) (bool, time.Duration) {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	entry, found := limiter.clients[clientIP]
	if !found && len(limiter.clients) >= maximumTrackedClients {
		limiter.removeExpired(now)
		if len(limiter.clients) >= maximumTrackedClients {
			limiter.removeOldest()
		}
	}

	entry, found = limiter.clients[clientIP]
	if !found || !now.Before(entry.resetTime) {
		entry = rateLimitEntry{resetTime: now.Add(limiter.window)}
	}

	if entry.count >= limiter.limit {
		limiter.clients[clientIP] = entry
		return false, entry.resetTime.Sub(now)
	}

	entry.count++
	limiter.clients[clientIP] = entry
	return true, 0
}

func (limiter *RateLimiter) removeExpired(now time.Time) {
	for clientIP, entry := range limiter.clients {
		if !now.Before(entry.resetTime) {
			delete(limiter.clients, clientIP)
		}
	}
}

func (limiter *RateLimiter) removeOldest() {
	var oldestClient string
	var oldestReset time.Time
	for clientIP, entry := range limiter.clients {
		if oldestClient == "" || entry.resetTime.Before(oldestReset) {
			oldestClient = clientIP
			oldestReset = entry.resetTime
		}
	}
	if oldestClient != "" {
		delete(limiter.clients, oldestClient)
	}
}
