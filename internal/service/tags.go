package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/philjestin/daedalus/internal/model"
	"github.com/philjestin/daedalus/internal/repository"
)

// TagService handles tag business logic.
type TagService struct {
	tagRepo    *repository.TagRepository
	partRepo   *repository.PartRepository
	designRepo *repository.DesignRepository
}

// NewTagService creates a new TagService.
func NewTagService(
	tagRepo *repository.TagRepository,
	partRepo *repository.PartRepository,
	designRepo *repository.DesignRepository,
) *TagService {
	return &TagService{
		tagRepo:    tagRepo,
		partRepo:   partRepo,
		designRepo: designRepo,
	}
}

// Create creates a new tag.
func (s *TagService) Create(ctx context.Context, tag *model.Tag) error {
	if tag.Name == "" {
		return fmt.Errorf("tag name is required")
	}

	// Check if tag already exists
	existing, err := s.tagRepo.GetByName(ctx, tag.Name)
	if err != nil {
		return err
	}
	if existing != nil {
		return fmt.Errorf("tag with name '%s' already exists", tag.Name)
	}

	return s.tagRepo.Create(ctx, tag)
}

// GetByID retrieves a tag by ID.
func (s *TagService) GetByID(ctx context.Context, id uuid.UUID) (*model.Tag, error) {
	return s.tagRepo.GetByID(ctx, id)
}

// GetByName retrieves a tag by name.
func (s *TagService) GetByName(ctx context.Context, name string) (*model.Tag, error) {
	return s.tagRepo.GetByName(ctx, name)
}

// List retrieves all tags.
func (s *TagService) List(ctx context.Context) ([]model.Tag, error) {
	return s.tagRepo.List(ctx)
}

// Update updates a tag.
func (s *TagService) Update(ctx context.Context, tag *model.Tag) error {
	if tag.Name == "" {
		return fmt.Errorf("tag name is required")
	}
	return s.tagRepo.Update(ctx, tag)
}

// Delete removes a tag.
func (s *TagService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.tagRepo.Delete(ctx, id)
}

// GetOrCreate retrieves a tag by name, creating it if it doesn't exist.
func (s *TagService) GetOrCreate(ctx context.Context, name string, color string) (*model.Tag, error) {
	tag, err := s.tagRepo.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if tag != nil {
		return tag, nil
	}

	tag = &model.Tag{
		Name:  name,
		Color: color,
	}
	if err := s.tagRepo.Create(ctx, tag); err != nil {
		return nil, err
	}
	return tag, nil
}

// ---- Part Tags ----

// AddTagToPart adds a tag to a part.
func (s *TagService) AddTagToPart(ctx context.Context, partID, tagID uuid.UUID) error {
	// Verify part exists
	part, err := s.partRepo.GetByID(ctx, partID)
	if err != nil {
		return err
	}
	if part == nil {
		return fmt.Errorf("part not found")
	}

	// Verify tag exists
	tag, err := s.tagRepo.GetByID(ctx, tagID)
	if err != nil {
		return err
	}
	if tag == nil {
		return fmt.Errorf("tag not found")
	}

	return s.tagRepo.AddToPart(ctx, partID, tagID)
}

// RemoveTagFromPart removes a tag from a part.
func (s *TagService) RemoveTagFromPart(ctx context.Context, partID, tagID uuid.UUID) error {
	return s.tagRepo.RemoveFromPart(ctx, partID, tagID)
}

// GetTagsForPart retrieves all tags for a part.
func (s *TagService) GetTagsForPart(ctx context.Context, partID uuid.UUID) ([]model.Tag, error) {
	return s.tagRepo.GetForPart(ctx, partID)
}

// SetPartTags replaces all tags for a part.
func (s *TagService) SetPartTags(ctx context.Context, partID uuid.UUID, tagIDs []uuid.UUID) error {
	// Get current tags
	currentTags, err := s.tagRepo.GetForPart(ctx, partID)
	if err != nil {
		return err
	}

	// Remove tags not in new set
	currentTagIDs := make(map[uuid.UUID]bool)
	for _, tag := range currentTags {
		currentTagIDs[tag.ID] = true
	}

	newTagIDs := make(map[uuid.UUID]bool)
	for _, id := range tagIDs {
		newTagIDs[id] = true
	}

	// Remove tags that are no longer present
	for id := range currentTagIDs {
		if !newTagIDs[id] {
			if err := s.tagRepo.RemoveFromPart(ctx, partID, id); err != nil {
				return err
			}
		}
	}

	// Add new tags
	for id := range newTagIDs {
		if !currentTagIDs[id] {
			if err := s.tagRepo.AddToPart(ctx, partID, id); err != nil {
				return err
			}
		}
	}

	return nil
}

// ---- Design Tags ----

// AddTagToDesign adds a tag to a design.
func (s *TagService) AddTagToDesign(ctx context.Context, designID, tagID uuid.UUID) error {
	// Verify design exists
	design, err := s.designRepo.GetByID(ctx, designID)
	if err != nil {
		return err
	}
	if design == nil {
		return fmt.Errorf("design not found")
	}

	// Verify tag exists
	tag, err := s.tagRepo.GetByID(ctx, tagID)
	if err != nil {
		return err
	}
	if tag == nil {
		return fmt.Errorf("tag not found")
	}

	return s.tagRepo.AddToDesign(ctx, designID, tagID)
}

// RemoveTagFromDesign removes a tag from a design.
func (s *TagService) RemoveTagFromDesign(ctx context.Context, designID, tagID uuid.UUID) error {
	return s.tagRepo.RemoveFromDesign(ctx, designID, tagID)
}

// GetTagsForDesign retrieves all tags for a design.
func (s *TagService) GetTagsForDesign(ctx context.Context, designID uuid.UUID) ([]model.Tag, error) {
	return s.tagRepo.GetForDesign(ctx, designID)
}

// SetDesignTags replaces all tags for a design.
func (s *TagService) SetDesignTags(ctx context.Context, designID uuid.UUID, tagIDs []uuid.UUID) error {
	// Get current tags
	currentTags, err := s.tagRepo.GetForDesign(ctx, designID)
	if err != nil {
		return err
	}

	// Remove tags not in new set
	currentTagIDs := make(map[uuid.UUID]bool)
	for _, tag := range currentTags {
		currentTagIDs[tag.ID] = true
	}

	newTagIDs := make(map[uuid.UUID]bool)
	for _, id := range tagIDs {
		newTagIDs[id] = true
	}

	// Remove tags that are no longer present
	for id := range currentTagIDs {
		if !newTagIDs[id] {
			if err := s.tagRepo.RemoveFromDesign(ctx, designID, id); err != nil {
				return err
			}
		}
	}

	// Add new tags
	for id := range newTagIDs {
		if !currentTagIDs[id] {
			if err := s.tagRepo.AddToDesign(ctx, designID, id); err != nil {
				return err
			}
		}
	}

	return nil
}

// ---- Search ----

// ListPartsByTag retrieves all parts with a given tag.
func (s *TagService) ListPartsByTag(ctx context.Context, tagID uuid.UUID) ([]model.Part, error) {
	partIDs, err := s.tagRepo.ListPartsByTag(ctx, tagID)
	if err != nil {
		return nil, err
	}

	var parts []model.Part
	for _, id := range partIDs {
		part, err := s.partRepo.GetByID(ctx, id)
		if err != nil {
			continue
		}
		if part != nil {
			parts = append(parts, *part)
		}
	}

	return parts, nil
}

// ListDesignsByTag retrieves all designs with a given tag.
func (s *TagService) ListDesignsByTag(ctx context.Context, tagID uuid.UUID) ([]model.Design, error) {
	designIDs, err := s.tagRepo.ListDesignsByTag(ctx, tagID)
	if err != nil {
		return nil, err
	}

	var designs []model.Design
	for _, id := range designIDs {
		design, err := s.designRepo.GetByID(ctx, id)
		if err != nil {
			continue
		}
		if design != nil {
			designs = append(designs, *design)
		}
	}

	return designs, nil
}
