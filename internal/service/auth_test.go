package service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/cadimartech/rankos/internal/database"
	"github.com/cadimartech/rankos/internal/repository"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func TestHashPassword_roundTrip(t *testing.T) {
	hash, err := HashPassword("correct-horse", bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if hash == "correct-horse" || !strings.HasPrefix(hash, "$2") {
		t.Fatalf("hash = %q", hash)
	}
	other, err := HashPassword("correct-horse", bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if hash == other {
		t.Fatal("bcrypt hashes of the same password should differ")
	}
	if !VerifyPassword(hash, "correct-horse") {
		t.Fatal("expected password to match")
	}
	if VerifyPassword(hash, "wrong-horse") || VerifyPassword("", "correct-horse") {
		t.Fatal("expected mismatch")
	}
}

func TestValidatePassword_length(t *testing.T) {
	if err := validatePassword("short"); err == nil {
		t.Fatal("expected short password to fail")
	}
	if err := validatePassword(strings.Repeat("a", 73)); err == nil {
		t.Fatal("expected 73-byte password to fail")
	}
	if err := validatePassword(strings.Repeat("a", 8)); err != nil {
		t.Fatal(err)
	}
	if err := validatePassword(strings.Repeat("a", 72)); err != nil {
		t.Fatal(err)
	}
}

func TestAuthService_defaultCostIsProduction(t *testing.T) {
	svc := NewAuthService(nil)
	if svc.HashCost() < bcrypt.DefaultCost {
		t.Fatalf("cost = %d, want at least %d", svc.HashCost(), bcrypt.DefaultCost)
	}
}

func TestAuthService_setupThenLocked(t *testing.T) {
	svc, _ := newAuthService(t)
	ctx := context.Background()

	user, token, err := svc.Setup(ctx, "Owner", "password12", "agent", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if !user.IsAdmin || user.PasswordHash != "" || token == "" {
		t.Fatalf("user=%+v token empty=%v", user, token == "")
	}

	_, _, err = svc.Setup(ctx, "Second", "password12", "", "")
	if !errors.Is(err, ErrSetupClosed) {
		t.Fatalf("second setup = %v", err)
	}

	signedIn, err := svc.ValidateSession(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(signedIn.Username, "Owner") {
		t.Fatalf("username = %s", signedIn.Username)
	}

	again, againToken, err := svc.Login(ctx, "owner", "password12", "", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != user.ID || againToken == "" || againToken == token {
		t.Fatal("login should open a new session for the same user")
	}
	if _, _, err := svc.Login(ctx, "owner", "nope-nope", "", "127.0.0.1"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("bad password = %v", err)
	}
}

func TestAuthService_sessionStoredHashedAndExpires(t *testing.T) {
	svc, db := newAuthService(t)
	ctx := context.Background()
	_, token, err := svc.Setup(ctx, "owner", "password12", "RankOS", "203.0.113.9")
	if err != nil {
		t.Fatal(err)
	}

	var stored string
	if err := db.QueryRow(`SELECT token_hash FROM sessions`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored == token || len(stored) != 64 {
		t.Fatalf("stored token hash = %q", stored)
	}

	past := time.Now().Add(-time.Hour).UTC()
	if _, err := db.Exec(`UPDATE sessions SET expires_at = ?`, past); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ValidateSession(ctx, token); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expired session = %v", err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("expired session still stored, count=%d", n)
	}
}

func TestAuthService_slidingSession(t *testing.T) {
	svc, db := newAuthService(t)
	ctx := context.Background()
	_, token, err := svc.Setup(ctx, "owner", "password12", "", "")
	if err != nil {
		t.Fatal(err)
	}
	soon := time.Now().Add(time.Hour).UTC()
	seen := time.Now().Add(-2 * time.Minute).UTC()
	if _, err := db.Exec(`UPDATE sessions SET expires_at = ?, last_seen_at = ?`, soon, seen); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ValidateSession(ctx, token); err != nil {
		t.Fatal(err)
	}
	var rawExp string
	if err := db.QueryRow(`SELECT expires_at FROM sessions`).Scan(&rawExp); err != nil {
		t.Fatal(err)
	}
	exp := parseDBTime(t, rawExp)
	if exp.Before(time.Now().Add(20 * 24 * time.Hour)) {
		t.Fatalf("expiry = %s, want about 30 days out", exp)
	}
}

func parseDBTime(t *testing.T, raw string) time.Time {
	t.Helper()
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02 15:04:05-07:00",
		"2006-01-02 15:04:05",
	}
	for _, format := range formats {
		if parsed, err := time.Parse(format, raw); err == nil {
			return parsed
		}
	}
	t.Fatalf("cannot parse time %q", raw)
	return time.Time{}
}

func TestAuthService_changePasswordDropsOtherSessions(t *testing.T) {
	svc, _ := newAuthService(t)
	ctx := context.Background()
	user, first, err := svc.Setup(ctx, "owner", "password12", "", "")
	if err != nil {
		t.Fatal(err)
	}
	_, second, err := svc.Login(ctx, "owner", "password12", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ChangePassword(ctx, user.ID, "password12", "new-password", first); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ValidateSession(ctx, first); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ValidateSession(ctx, second); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("old session = %v", err)
	}
	if _, _, err := svc.Login(ctx, "owner", "password12", "", ""); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("old password = %v", err)
	}
	if _, _, err := svc.Login(ctx, "owner", "new-password", "", ""); err != nil {
		t.Fatal(err)
	}
}

func TestAuthService_loginRateLimit(t *testing.T) {
	svc, _ := newAuthService(t)
	svc.ConfigureLoginLimit(LimitConfig{
		MaxAttempts: 2,
		Window:      time.Minute,
		BaseBackoff: time.Hour,
		MaxBackoff:  time.Hour,
	})
	ctx := context.Background()
	if _, _, err := svc.Setup(ctx, "owner", "password12", "", ""); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		_, _, err := svc.Login(ctx, "owner", "wrong-password", "", "198.51.100.8")
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("attempt %d = %v", i+1, err)
		}
	}
	_, _, err := svc.Login(ctx, "owner", "wrong-password", "", "198.51.100.8")
	var limited *RateLimitError
	if !errors.As(err, &limited) {
		t.Fatalf("expected rate limit, got %v", err)
	}
	if limited.RetryAfter < time.Minute {
		t.Fatalf("retry after = %s", limited.RetryAfter)
	}
	// A different username from the same IP is also blocked.
	_, _, err = svc.Login(ctx, "nobody", "wrong-password", "", "198.51.100.8")
	if !errors.As(err, &limited) {
		t.Fatalf("ip block = %v", err)
	}
}

func TestAuthService_userManagement(t *testing.T) {
	svc, _ := newAuthService(t)
	ctx := context.Background()
	admin, _, err := svc.Setup(ctx, "owner", "password12", "", "")
	if err != nil {
		t.Fatal(err)
	}
	created, err := svc.CreateUser(ctx, admin.ID, "helper", "password12", false)
	if err != nil {
		t.Fatal(err)
	}
	if created.IsAdmin || created.PasswordHash != "" {
		t.Fatalf("created = %+v", created)
	}
	if _, err := svc.CreateUser(ctx, created.ID, "nope", "password12", false); !errors.Is(err, ErrForbidden) {
		t.Fatalf("non-admin create = %v", err)
	}
	if _, err := svc.CreateUser(ctx, admin.ID, "Helper", "password12", false); !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("duplicate = %v", err)
	}
	if err := svc.DeleteUser(ctx, admin.ID, admin.ID); !errors.Is(err, ErrLastAdmin) {
		t.Fatalf("delete last admin = %v", err)
	}
	secondAdmin, err := svc.CreateUser(ctx, admin.ID, "boss", "password12", true)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteUser(ctx, admin.ID, admin.ID); !errors.Is(err, ErrDeleteSelf) {
		t.Fatalf("delete self = %v", err)
	}
	if err := svc.DeleteUser(ctx, admin.ID, secondAdmin.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteUser(ctx, admin.ID, uuid.New()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing user = %v", err)
	}
}

func newAuthService(t *testing.T) (*AuthService, *sql.DB) {
	t.Helper()
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	svc := NewAuthService(repository.NewRepositories(db).Users)
	svc.SetHashCost(bcrypt.MinCost)
	return svc, db
}
