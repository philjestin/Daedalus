package service

import (
	"testing"
	"time"
)

func TestLoginLimiter_backoffPerIPAndUsername(t *testing.T) {
	limiter := NewLoginLimiter(LimitConfig{
		MaxAttempts:   3,
		IPMaxAttempts: 3,
		Window:        time.Minute,
		BaseBackoff:   time.Second,
		MaxBackoff:    time.Minute,
	})
	now := time.Now()

	for i := 0; i < 3; i++ {
		if _, ok := limiter.Allow("198.51.100.10", "sam", now); !ok {
			t.Fatalf("attempt %d should be allowed", i+1)
		}
		limiter.Fail("198.51.100.10", "sam", now)
	}

	if _, ok := limiter.Allow("198.51.100.10", "sam", now); ok {
		t.Fatal("same ip and username should be locked")
	}
	if _, ok := limiter.Allow("198.51.100.10", "other", now); ok {
		t.Fatal("same ip should be locked for a different username")
	}
	if _, ok := limiter.Allow("198.51.100.11", "sam", now); ok {
		t.Fatal("same username should be locked from a different ip")
	}
	if _, ok := limiter.Allow("198.51.100.12", "other", now); !ok {
		t.Fatal("unrelated ip and username should be allowed")
	}

	retry, ok := limiter.Allow("198.51.100.10", "sam", now)
	if ok || retry < time.Second {
		t.Fatalf("retry=%s allowed=%v", retry, ok)
	}

	// Backoff elapsed, one more attempt is allowed, then the lock returns.
	later := now.Add(time.Second + time.Millisecond)
	if _, ok := limiter.Allow("198.51.100.10", "sam", later); !ok {
		t.Fatal("attempt after backoff should be allowed once")
	}
	limiter.Fail("198.51.100.10", "sam", later)
	if _, ok := limiter.Allow("198.51.100.10", "sam", later); ok {
		t.Fatal("failure after backoff should lock again")
	}

	limiter.Success("198.51.100.10", "Sam")
	if _, ok := limiter.Allow("198.51.100.10", "sam", later); !ok {
		t.Fatal("success should clear the ip and username")
	}
}

func TestLoginLimiter_usernameLocksBeforeSharedIP(t *testing.T) {
	limiter := NewLoginLimiter(DefaultLimitConfig())
	now := time.Now()
	for i := 0; i < 5; i++ {
		if _, ok := limiter.Allow("198.51.100.10", "sam", now); !ok {
			t.Fatalf("attempt %d should be allowed", i+1)
		}
		limiter.Fail("198.51.100.10", "sam", now)
	}
	if _, ok := limiter.Allow("198.51.100.10", "sam", now); ok {
		t.Fatal("username should lock at the default attempt cap")
	}
	if _, ok := limiter.Allow("198.51.100.10", "other", now); !ok {
		t.Fatal("per-IP limit should be higher than the per-username limit")
	}
	if _, ok := limiter.Allow("198.51.100.11", "other", now); !ok {
		t.Fatal("unrelated client should be allowed")
	}
}

func TestLoginLimiter_blankIPIsNotSharedBucket(t *testing.T) {
	limiter := NewLoginLimiter(LimitConfig{
		MaxAttempts: 2,
		Window:      time.Minute,
		BaseBackoff: time.Hour,
		MaxBackoff:  time.Hour,
	})
	now := time.Now()
	limiter.Fail("", "ann", now)
	limiter.Fail("", "ann", now)
	if _, ok := limiter.Allow("", "ann", now); ok {
		t.Fatal("username should still lock when no client IP is available")
	}
	if _, ok := limiter.Allow("", "bob", now); !ok {
		t.Fatal("blank IP must not lock every username")
	}
}

func TestLoginLimiter_unsetIPMaxAttemptsUsesHigherDefault(t *testing.T) {
	limiter := NewLoginLimiter(LimitConfig{
		MaxAttempts: 2,
		Window:      time.Minute,
		BaseBackoff: time.Hour,
		MaxBackoff:  time.Hour,
	})
	now := time.Now()
	limiter.Fail("198.51.100.10", "sam", now)
	limiter.Fail("198.51.100.10", "sam", now)
	if _, ok := limiter.Allow("198.51.100.10", "sam", now); ok {
		t.Fatal("username should lock at MaxAttempts")
	}
	if _, ok := limiter.Allow("198.51.100.10", "other", now); !ok {
		t.Fatal("omitted IP cap should stay above the username cap")
	}
}

func TestLoginLimiter_windowDropsOldFailures(t *testing.T) {
	limiter := NewLoginLimiter(LimitConfig{
		MaxAttempts: 2,
		Window:      time.Minute,
		BaseBackoff: time.Hour,
		MaxBackoff:  time.Hour,
	})
	start := time.Now()
	limiter.Fail("ip", "user", start)
	limiter.Fail("ip", "user", start)
	if _, ok := limiter.Allow("ip", "user", start); ok {
		t.Fatal("expected lock inside the window")
	}
	outside := start.Add(time.Minute + time.Second)
	if _, ok := limiter.Allow("ip", "user", outside); !ok {
		t.Fatal("failures outside the window should not lock")
	}
}
