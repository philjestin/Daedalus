package service

import (
	"context"
	"testing"

	"github.com/hyperion/printfarm/internal/database"
	"github.com/hyperion/printfarm/internal/model"
	"github.com/hyperion/printfarm/internal/repository"
)

func newFeedbackTestService(t *testing.T) *FeedbackService {
	t.Helper()
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	repos := repository.NewRepositories(db)
	return &FeedbackService{repo: repos.Feedback}
}

func TestFeedbackService_Submit(t *testing.T) {
	svc := newFeedbackTestService(t)
	ctx := context.Background()

	t.Run("valid submission", func(t *testing.T) {
		f := &model.Feedback{
			Type:    "bug",
			Message: "Something is broken",
			Contact: "user@example.com",
			Page:    "/dashboard",
		}
		if err := svc.Submit(ctx, f); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if f.ID.String() == "" || f.ID.String() == "00000000-0000-0000-0000-000000000000" {
			t.Error("expected ID to be set")
		}
		if f.CreatedAt.IsZero() {
			t.Error("expected CreatedAt to be set")
		}
		if f.AppVersion == "" {
			t.Error("expected AppVersion to be set")
		}
	})

	t.Run("empty message rejected", func(t *testing.T) {
		f := &model.Feedback{
			Type:    "bug",
			Message: "",
		}
		if err := svc.Submit(ctx, f); err == nil {
			t.Error("expected error for empty message")
		}
	})

	t.Run("invalid type defaults to general", func(t *testing.T) {
		f := &model.Feedback{
			Type:    "invalid",
			Message: "test message",
		}
		if err := svc.Submit(ctx, f); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if f.Type != "general" {
			t.Errorf("expected type 'general', got %q", f.Type)
		}
	})
}

func TestFeedbackService_List(t *testing.T) {
	svc := newFeedbackTestService(t)
	ctx := context.Background()

	t.Run("empty list", func(t *testing.T) {
		items, err := svc.List(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(items) != 0 {
			t.Errorf("expected 0 items, got %d", len(items))
		}
	})

	t.Run("returns submissions in reverse chronological order", func(t *testing.T) {
		for _, msg := range []string{"first", "second", "third"} {
			f := &model.Feedback{Type: "general", Message: msg}
			if err := svc.Submit(ctx, f); err != nil {
				t.Fatalf("submit: %v", err)
			}
		}

		items, err := svc.List(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(items) != 3 {
			t.Fatalf("expected 3 items, got %d", len(items))
		}
		if items[0].Message != "third" {
			t.Errorf("expected newest first, got %q", items[0].Message)
		}
	})
}

func TestFeedbackService_Delete(t *testing.T) {
	svc := newFeedbackTestService(t)
	ctx := context.Background()

	f := &model.Feedback{Type: "bug", Message: "delete me"}
	if err := svc.Submit(ctx, f); err != nil {
		t.Fatalf("submit: %v", err)
	}

	if err := svc.Delete(ctx, f.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}

	items, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 0 {
		t.Errorf("expected 0 items after delete, got %d", len(items))
	}
}
