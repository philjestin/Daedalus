package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/cadimartech/rankos/internal/model"
	"github.com/google/uuid"
)

func TestUserRepository_caseInsensitiveUsername(t *testing.T) {
	repo := &UserRepository{db: openTestDB(t)}
	ctx := context.Background()

	user := &model.User{Username: "Admin", PasswordHash: "hash-1", IsAdmin: true}
	if err := repo.Create(ctx, user); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.GetByUsername(ctx, "admin")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != user.ID || got.Username != "Admin" || !got.IsAdmin {
		t.Fatalf("got %+v", got)
	}

	dup := &model.User{Username: "ADMIN", PasswordHash: "hash-2"}
	if err := repo.Create(ctx, dup); !errors.Is(err, ErrUsernameTaken) {
		t.Fatalf("duplicate username error = %v", err)
	}
}

func TestUserRepository_CreateFirstUserLocks(t *testing.T) {
	repo := &UserRepository{db: openTestDB(t)}
	ctx := context.Background()

	first := &model.User{Username: "owner", PasswordHash: "hash"}
	if err := repo.CreateFirstUser(ctx, first); err != nil {
		t.Fatalf("first: %v", err)
	}
	if !first.IsAdmin {
		t.Fatal("first user should be admin")
	}

	second := &model.User{Username: "other", PasswordHash: "hash"}
	if err := repo.CreateFirstUser(ctx, second); !errors.Is(err, ErrAlreadySetup) {
		t.Fatalf("second setup error = %v", err)
	}

	n, err := repo.Count(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("user count = %d", n)
	}
}

func TestSessionRepository_createValidateExpire(t *testing.T) {
	repo := &UserRepository{db: openTestDB(t)}
	ctx := context.Background()
	user := &model.User{Username: "owner", PasswordHash: "hash"}
	if err := repo.CreateFirstUser(ctx, user); err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	session := &model.Session{
		UserID:     user.ID,
		TokenHash:  "abc123",
		ExpiresAt:  now.Add(time.Hour),
		CreatedAt:  now,
		LastSeenAt: now,
		UserAgent:  "test-agent",
		IP:         "203.0.113.5",
	}
	if err := repo.CreateSession(ctx, session); err != nil {
		t.Fatal(err)
	}
	if session.TokenHash == "raw-token" {
		t.Fatal("repository must store the hash it is given, not invent a raw token")
	}

	gotSession, gotUser, err := repo.GetSession(ctx, session.TokenHash)
	if err != nil {
		t.Fatal(err)
	}
	if gotUser.Username != "owner" || gotSession.IP != "203.0.113.5" || gotSession.UserAgent != "test-agent" {
		t.Fatalf("session = %+v user = %+v", gotSession, gotUser)
	}

	later := now.Add(30 * 24 * time.Hour)
	if err := repo.TouchSession(ctx, gotSession.ID, later, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	touched, _, err := repo.GetSession(ctx, session.TokenHash)
	if err != nil {
		t.Fatal(err)
	}
	if touched.ExpiresAt.Before(now.Add(29 * 24 * time.Hour)) {
		t.Fatalf("expiry did not slide: %s", touched.ExpiresAt)
	}

	if err := repo.Delete(ctx, user.ID); err != nil {
		t.Fatal(err)
	}
	_, _, err = repo.GetSession(ctx, session.TokenHash)
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("session after user delete = %v", err)
	}
	if _, err := repo.GetByID(ctx, user.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("user after delete = %v", err)
	}
}

func TestUserRepository_listOmitsNothingButKeepsHashForCaller(t *testing.T) {
	repo := &UserRepository{db: openTestDB(t)}
	ctx := context.Background()
	if err := repo.Create(ctx, &model.User{Username: "a", PasswordHash: "secret-hash"}); err != nil {
		t.Fatal(err)
	}
	users, err := repo.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 || users[0].PasswordHash != "secret-hash" || users[0].ID == uuid.Nil {
		t.Fatalf("users = %+v", users)
	}
}
