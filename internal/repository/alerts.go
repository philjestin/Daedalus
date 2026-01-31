package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/hyperion/printfarm/internal/model"
)

// AlertDismissalRepository handles alert dismissal database operations.
type AlertDismissalRepository struct {
	db *sql.DB
}

// Create inserts a new alert dismissal.
func (r *AlertDismissalRepository) Create(ctx context.Context, dismissal *model.AlertDismissal) error {
	dismissal.ID = uuid.New()
	dismissal.DismissedAt = time.Now()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO alert_dismissals (id, alert_type, entity_id, dismissed_at, dismissed_until)
		VALUES (?, ?, ?, ?, ?)
	`, dismissal.ID, dismissal.AlertType, dismissal.EntityID, dismissal.DismissedAt, dismissal.DismissedUntil)
	return err
}

// GetByEntity retrieves a dismissal for a specific alert type and entity.
func (r *AlertDismissalRepository) GetByEntity(ctx context.Context, alertType model.AlertType, entityID string) (*model.AlertDismissal, error) {
	var dismissal model.AlertDismissal
	err := r.db.QueryRowContext(ctx, `
		SELECT id, alert_type, entity_id, dismissed_at, dismissed_until
		FROM alert_dismissals WHERE alert_type = ? AND entity_id = ?
		ORDER BY dismissed_at DESC LIMIT 1
	`, alertType, entityID).Scan(&dismissal.ID, &dismissal.AlertType, &dismissal.EntityID, &dismissal.DismissedAt, &dismissal.DismissedUntil)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &dismissal, err
}

// IsDismissed checks if an alert is currently dismissed.
func (r *AlertDismissalRepository) IsDismissed(ctx context.Context, alertType model.AlertType, entityID string) (bool, error) {
	dismissal, err := r.GetByEntity(ctx, alertType, entityID)
	if err != nil {
		return false, err
	}
	if dismissal == nil {
		return false, nil
	}

	// If dismissed_until is nil, it's permanently dismissed
	if dismissal.DismissedUntil == nil {
		return true, nil
	}

	// Check if the snooze has expired
	return time.Now().Before(*dismissal.DismissedUntil), nil
}

// Delete removes a dismissal by ID.
func (r *AlertDismissalRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM alert_dismissals WHERE id = ?`, id)
	return err
}

// DeleteByEntity removes all dismissals for a specific entity.
func (r *AlertDismissalRepository) DeleteByEntity(ctx context.Context, alertType model.AlertType, entityID string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM alert_dismissals WHERE alert_type = ? AND entity_id = ?
	`, alertType, entityID)
	return err
}

// CleanupExpired removes all expired snooze dismissals.
func (r *AlertDismissalRepository) CleanupExpired(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM alert_dismissals
		WHERE dismissed_until IS NOT NULL AND dismissed_until < datetime('now')
	`)
	return err
}

// List retrieves all active dismissals.
func (r *AlertDismissalRepository) List(ctx context.Context) ([]model.AlertDismissal, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, alert_type, entity_id, dismissed_at, dismissed_until
		FROM alert_dismissals
		WHERE dismissed_until IS NULL OR dismissed_until >= datetime('now')
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dismissals []model.AlertDismissal
	for rows.Next() {
		var dismissal model.AlertDismissal
		if err := rows.Scan(&dismissal.ID, &dismissal.AlertType, &dismissal.EntityID, &dismissal.DismissedAt, &dismissal.DismissedUntil); err != nil {
			return nil, err
		}
		dismissals = append(dismissals, dismissal)
	}
	return dismissals, rows.Err()
}
