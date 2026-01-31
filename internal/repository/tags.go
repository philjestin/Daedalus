package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/hyperion/printfarm/internal/model"
)

// TagRepository handles tag database operations.
type TagRepository struct {
	db *sql.DB
}

// Create inserts a new tag.
func (r *TagRepository) Create(ctx context.Context, tag *model.Tag) error {
	tag.ID = uuid.New()
	tag.CreatedAt = time.Now()
	if tag.Color == "" {
		tag.Color = "#6b7280"
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO tags (id, name, color, created_at)
		VALUES (?, ?, ?, ?)
	`, tag.ID, tag.Name, tag.Color, tag.CreatedAt)
	return err
}

// GetByID retrieves a tag by ID.
func (r *TagRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Tag, error) {
	var tag model.Tag
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, color, created_at
		FROM tags WHERE id = ?
	`, id).Scan(&tag.ID, &tag.Name, &tag.Color, &tag.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &tag, err
}

// GetByName retrieves a tag by name.
func (r *TagRepository) GetByName(ctx context.Context, name string) (*model.Tag, error) {
	var tag model.Tag
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, color, created_at
		FROM tags WHERE name = ?
	`, name).Scan(&tag.ID, &tag.Name, &tag.Color, &tag.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &tag, err
}

// List retrieves all tags.
func (r *TagRepository) List(ctx context.Context) ([]model.Tag, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, color, created_at
		FROM tags ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []model.Tag
	for rows.Next() {
		var tag model.Tag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.Color, &tag.CreatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

// Update updates a tag.
func (r *TagRepository) Update(ctx context.Context, tag *model.Tag) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE tags SET name = ?, color = ?
		WHERE id = ?
	`, tag.Name, tag.Color, tag.ID)
	return err
}

// Delete removes a tag.
func (r *TagRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM tags WHERE id = ?`, id)
	return err
}

// AddToPart adds a tag to a part.
func (r *TagRepository) AddToPart(ctx context.Context, partID, tagID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO part_tags (part_id, tag_id)
		VALUES (?, ?)
	`, partID, tagID)
	return err
}

// RemoveFromPart removes a tag from a part.
func (r *TagRepository) RemoveFromPart(ctx context.Context, partID, tagID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM part_tags WHERE part_id = ? AND tag_id = ?
	`, partID, tagID)
	return err
}

// GetForPart retrieves all tags for a part.
func (r *TagRepository) GetForPart(ctx context.Context, partID uuid.UUID) ([]model.Tag, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT t.id, t.name, t.color, t.created_at
		FROM tags t
		JOIN part_tags pt ON t.id = pt.tag_id
		WHERE pt.part_id = ?
		ORDER BY t.name ASC
	`, partID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []model.Tag
	for rows.Next() {
		var tag model.Tag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.Color, &tag.CreatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

// AddToDesign adds a tag to a design.
func (r *TagRepository) AddToDesign(ctx context.Context, designID, tagID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO design_tags (design_id, tag_id)
		VALUES (?, ?)
	`, designID, tagID)
	return err
}

// RemoveFromDesign removes a tag from a design.
func (r *TagRepository) RemoveFromDesign(ctx context.Context, designID, tagID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM design_tags WHERE design_id = ? AND tag_id = ?
	`, designID, tagID)
	return err
}

// GetForDesign retrieves all tags for a design.
func (r *TagRepository) GetForDesign(ctx context.Context, designID uuid.UUID) ([]model.Tag, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT t.id, t.name, t.color, t.created_at
		FROM tags t
		JOIN design_tags dt ON t.id = dt.tag_id
		WHERE dt.design_id = ?
		ORDER BY t.name ASC
	`, designID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []model.Tag
	for rows.Next() {
		var tag model.Tag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.Color, &tag.CreatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, tag)
	}
	return tags, rows.Err()
}

// ListPartsByTag retrieves all part IDs with a given tag.
func (r *TagRepository) ListPartsByTag(ctx context.Context, tagID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT part_id FROM part_tags WHERE tag_id = ?
	`, tagID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// ListDesignsByTag retrieves all design IDs with a given tag.
func (r *TagRepository) ListDesignsByTag(ctx context.Context, tagID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT design_id FROM design_tags WHERE tag_id = ?
	`, tagID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
