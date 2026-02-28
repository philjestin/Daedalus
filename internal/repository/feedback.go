package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/hyperion/printfarm/internal/model"
)

// FeedbackRepository handles feedback database operations.
type FeedbackRepository struct {
	db *sql.DB
}

// Create inserts a new feedback record.
func (r *FeedbackRepository) Create(ctx context.Context, f *model.Feedback) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO feedback (id, type, message, contact, page, app_version, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		f.ID.String(), f.Type, f.Message, f.Contact, f.Page, f.AppVersion, f.CreatedAt,
	)
	return err
}

// List returns all feedback ordered by newest first.
func (r *FeedbackRepository) List(ctx context.Context) ([]model.Feedback, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, type, message, contact, page, app_version, created_at
		 FROM feedback ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feedback []model.Feedback
	for rows.Next() {
		var f model.Feedback
		var id string
		var contact, page, appVersion sql.NullString
		if err := rows.Scan(&id, &f.Type, &f.Message, &contact, &page, &appVersion, &f.CreatedAt); err != nil {
			return nil, err
		}
		f.ID = uuid.MustParse(id)
		if contact.Valid {
			f.Contact = contact.String
		}
		if page.Valid {
			f.Page = page.String
		}
		if appVersion.Valid {
			f.AppVersion = appVersion.String
		}
		feedback = append(feedback, f)
	}
	return feedback, rows.Err()
}

// Delete removes a feedback record by ID.
func (r *FeedbackRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM feedback WHERE id = ?`, id.String())
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}
