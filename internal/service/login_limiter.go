package service

import (
	"strings"
	"sync"
	"time"
)

// LimitConfig controls login backoff. Zero values are replaced with defaults
// by NewLoginLimiter.
type LimitConfig struct {
	MaxAttempts int
	Window      time.Duration
	BaseBackoff time.Duration
	MaxBackoff  time.Duration
}

// DefaultLimitConfig is five failures per 15 minutes, then exponential backoff.
func DefaultLimitConfig() LimitConfig {
	return LimitConfig{
		MaxAttempts: 5,
		Window:      15 * time.Minute,
		BaseBackoff: 2 * time.Second,
		MaxBackoff:  15 * time.Minute,
	}
}

type failureWindow struct {
	failures    []time.Time
	lockedUntil time.Time
}

// LoginLimiter tracks failed sign-ins per client IP and per username.
type LoginLimiter struct {
	mu          sync.Mutex
	states      map[string]*failureWindow
	maxAttempts int
	window      time.Duration
	baseBackoff time.Duration
	maxBackoff  time.Duration
}

// NewLoginLimiter builds a limiter. Invalid config falls back to defaults.
func NewLoginLimiter(cfg LimitConfig) *LoginLimiter {
	def := DefaultLimitConfig()
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = def.MaxAttempts
	}
	if cfg.Window <= 0 {
		cfg.Window = def.Window
	}
	if cfg.BaseBackoff <= 0 {
		cfg.BaseBackoff = def.BaseBackoff
	}
	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = def.MaxBackoff
	}
	return &LoginLimiter{
		states:      make(map[string]*failureWindow),
		maxAttempts: cfg.MaxAttempts,
		window:      cfg.Window,
		baseBackoff: cfg.BaseBackoff,
		maxBackoff:  cfg.MaxBackoff,
	}
}

// Allow reports whether a login attempt may proceed.
// retryAfter is how long the caller should wait when allowed is false.
func (l *LoginLimiter) Allow(ip, username string, now time.Time) (retryAfter time.Duration, allowed bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	var wait time.Duration
	blocked := false
	for _, key := range l.keys(ip, username) {
		st := l.states[key]
		if st == nil {
			continue
		}
		l.prune(st, now)
		if st.lockedUntil.After(now) {
			blocked = true
			d := st.lockedUntil.Sub(now)
			if d > wait {
				wait = d
			}
		}
	}
	return wait, !blocked
}

// Fail records a rejected attempt and starts backoff once the cap is reached.
func (l *LoginLimiter) Fail(ip, username string, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, key := range l.keys(ip, username) {
		l.bump(key, now)
	}
}

// Success clears counters for this IP and username.
func (l *LoginLimiter) Success(ip, username string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, key := range l.keys(ip, username) {
		delete(l.states, key)
	}
}

func (l *LoginLimiter) keys(ip, username string) []string {
	keys := []string{"ip:" + ip}
	username = strings.ToLower(strings.TrimSpace(username))
	if username != "" {
		keys = append(keys, "user:"+username)
	}
	return keys
}

func (l *LoginLimiter) bump(key string, now time.Time) {
	st := l.states[key]
	if st == nil {
		st = &failureWindow{}
		l.states[key] = st
	}
	l.prune(st, now)
	if len(st.failures) > 32 {
		st.failures = st.failures[len(st.failures)-32:]
	}
	st.failures = append(st.failures, now)
	if len(st.failures) < l.maxAttempts {
		return
	}
	exp := len(st.failures) - l.maxAttempts
	if exp > 16 {
		exp = 16
	}
	delay := l.baseBackoff
	for i := 0; i < exp; i++ {
		if delay > l.maxBackoff/2 {
			delay = l.maxBackoff
			break
		}
		delay *= 2
	}
	if delay > l.maxBackoff {
		delay = l.maxBackoff
	}
	st.lockedUntil = now.Add(delay)
}

func (l *LoginLimiter) prune(st *failureWindow, now time.Time) {
	cutoff := now.Add(-l.window)
	kept := st.failures[:0]
	for _, ts := range st.failures {
		if !ts.Before(cutoff) {
			kept = append(kept, ts)
		}
	}
	st.failures = kept
	if len(st.failures) < l.maxAttempts {
		st.lockedUntil = time.Time{}
	}
}
