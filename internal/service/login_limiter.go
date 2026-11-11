package service

import (
	"strings"
	"sync"
	"time"
)

// LimitConfig controls login backoff. Zero values are replaced with defaults
// by NewLoginLimiter.
//
// MaxAttempts is the per-username cap. IPMaxAttempts is a separate, higher
// per-IP cap so one address cannot lock the instance as quickly as one account.
type LimitConfig struct {
	MaxAttempts   int
	IPMaxAttempts int
	Window        time.Duration
	BaseBackoff   time.Duration
	MaxBackoff    time.Duration
}

// Default username failures before backoff, and the higher per-IP cap.
const (
	defaultMaxAttempts   = 5
	defaultIPMaxAttempts = 20
)

// DefaultLimitConfig is five failures per username and twenty per IP inside
// 15 minutes, then exponential backoff.
func DefaultLimitConfig() LimitConfig {
	return LimitConfig{
		MaxAttempts:   defaultMaxAttempts,
		IPMaxAttempts: defaultIPMaxAttempts,
		Window:        15 * time.Minute,
		BaseBackoff:   2 * time.Second,
		MaxBackoff:    15 * time.Minute,
	}
}

type failureWindow struct {
	failures    []time.Time
	lockedUntil time.Time
}

// LoginLimiter tracks failed sign-ins per client IP and per username.
type LoginLimiter struct {
	mu            sync.Mutex
	states        map[string]*failureWindow
	maxAttempts   int
	ipMaxAttempts int
	window        time.Duration
	baseBackoff   time.Duration
	maxBackoff    time.Duration
}

// NewLoginLimiter builds a limiter. Invalid config falls back to defaults.
func NewLoginLimiter(cfg LimitConfig) *LoginLimiter {
	def := DefaultLimitConfig()
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = def.MaxAttempts
	}
	if cfg.IPMaxAttempts <= 0 {
		cfg.IPMaxAttempts = def.IPMaxAttempts
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
		states:        make(map[string]*failureWindow),
		maxAttempts:   cfg.MaxAttempts,
		ipMaxAttempts: cfg.IPMaxAttempts,
		window:        cfg.Window,
		baseBackoff:   cfg.BaseBackoff,
		maxBackoff:    cfg.MaxBackoff,
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
		l.prune(st, now, l.limitFor(key))
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
	var keys []string
	// An empty address means "do not apply the shared per-IP lock".
	// Callers pass that when the only known address is the proxy itself.
	if ip != "" {
		keys = append(keys, "ip:"+ip)
	}
	username = strings.ToLower(strings.TrimSpace(username))
	if username != "" {
		keys = append(keys, "user:"+username)
	}
	return keys
}

func (l *LoginLimiter) limitFor(key string) int {
	if strings.HasPrefix(key, "ip:") {
		return l.ipMaxAttempts
	}
	return l.maxAttempts
}

func (l *LoginLimiter) bump(key string, now time.Time) {
	st := l.states[key]
	if st == nil {
		st = &failureWindow{}
		l.states[key] = st
	}
	maxAttempts := l.limitFor(key)
	l.prune(st, now, maxAttempts)
	if len(st.failures) > 32 {
		st.failures = st.failures[len(st.failures)-32:]
	}
	st.failures = append(st.failures, now)
	if len(st.failures) < maxAttempts {
		return
	}
	exp := len(st.failures) - maxAttempts
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

func (l *LoginLimiter) prune(st *failureWindow, now time.Time, maxAttempts int) {
	cutoff := now.Add(-l.window)
	kept := st.failures[:0]
	for _, ts := range st.failures {
		if !ts.Before(cutoff) {
			kept = append(kept, ts)
		}
	}
	st.failures = kept
	if len(st.failures) < maxAttempts {
		st.lockedUntil = time.Time{}
	}
}
