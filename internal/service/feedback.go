package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hyperion/printfarm/internal/model"
	"github.com/hyperion/printfarm/internal/repository"
	"github.com/hyperion/printfarm/internal/version"
)

// FeedbackService handles feedback business logic.
type FeedbackService struct {
	repo *repository.FeedbackRepository
}

// Submit validates and stores a feedback submission.
func (s *FeedbackService) Submit(ctx context.Context, f *model.Feedback) error {
	if f.Message == "" {
		return fmt.Errorf("message is required")
	}

	validTypes := map[string]bool{"bug": true, "feature": true, "general": true}
	if !validTypes[f.Type] {
		f.Type = "general"
	}

	f.ID = uuid.New()
	f.CreatedAt = time.Now().UTC()
	f.AppVersion = version.Version

	return s.repo.Create(ctx, f)
}

// List returns all feedback submissions.
func (s *FeedbackService) List(ctx context.Context) ([]model.Feedback, error) {
	return s.repo.List(ctx)
}

// Delete removes a feedback submission by ID.
func (s *FeedbackService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
