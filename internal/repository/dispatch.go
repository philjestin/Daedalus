package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/philjestin/daedalus/internal/model"
)

// DispatchRepository handles dispatch request database operations.
type DispatchRepository struct {
	db *sql.DB
}

// NewDispatchRepository creates a new dispatch repository.
func NewDispatchRepository(db *sql.DB) *DispatchRepository {
	return &DispatchRepository{db: db}
}

// Create inserts a new dispatch request.
func (r *DispatchRepository) Create(ctx context.Context, req *model.DispatchRequest) error {
	req.ID = uuid.New()
	req.CreatedAt = time.Now()
	req.Status = model.DispatchPending

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO dispatch_requests (id, job_id, printer_id, status, created_at, expires_at, responded_at, reason)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, req.ID, req.JobID, req.PrinterID, req.Status, req.CreatedAt, req.ExpiresAt, req.RespondedAt, req.Reason)
	return err
}

// GetByID retrieves a dispatch request by ID.
func (r *DispatchRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.DispatchRequest, error) {
	var req model.DispatchRequest
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, job_id, printer_id, status, created_at, expires_at, responded_at, reason
		FROM dispatch_requests WHERE id = ?
	`, id), &req.ID, &req.JobID, &req.PrinterID, &req.Status, &req.CreatedAt, &req.ExpiresAt, &req.RespondedAt, &req.Reason)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &req, err
}

// GetPendingForPrinter retrieves the pending dispatch request for a printer, if any.
func (r *DispatchRepository) GetPendingForPrinter(ctx context.Context, printerID uuid.UUID) (*model.DispatchRequest, error) {
	var req model.DispatchRequest
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, job_id, printer_id, status, created_at, expires_at, responded_at, reason
		FROM dispatch_requests
		WHERE printer_id = ? AND status = 'pending'
		ORDER BY created_at DESC
		LIMIT 1
	`, printerID), &req.ID, &req.JobID, &req.PrinterID, &req.Status, &req.CreatedAt, &req.ExpiresAt, &req.RespondedAt, &req.Reason)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &req, err
}

// GetPendingForJob retrieves the pending dispatch request for a job, if any.
func (r *DispatchRepository) GetPendingForJob(ctx context.Context, jobID uuid.UUID) (*model.DispatchRequest, error) {
	var req model.DispatchRequest
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, job_id, printer_id, status, created_at, expires_at, responded_at, reason
		FROM dispatch_requests
		WHERE job_id = ? AND status = 'pending'
		ORDER BY created_at DESC
		LIMIT 1
	`, jobID), &req.ID, &req.JobID, &req.PrinterID, &req.Status, &req.CreatedAt, &req.ExpiresAt, &req.RespondedAt, &req.Reason)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &req, err
}

// ListPending retrieves all pending dispatch requests.
func (r *DispatchRepository) ListPending(ctx context.Context) ([]model.DispatchRequest, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, job_id, printer_id, status, created_at, expires_at, responded_at, reason
		FROM dispatch_requests
		WHERE status = 'pending'
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []model.DispatchRequest
	for rows.Next() {
		var req model.DispatchRequest
		if err := scanRow(rows, &req.ID, &req.JobID, &req.PrinterID, &req.Status, &req.CreatedAt, &req.ExpiresAt, &req.RespondedAt, &req.Reason); err != nil {
			return nil, err
		}
		requests = append(requests, req)
	}
	return requests, rows.Err()
}

// UpdateStatus updates the status of a dispatch request.
func (r *DispatchRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.DispatchRequestStatus, reason string) error {
	now := time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE dispatch_requests
		SET status = ?, responded_at = ?, reason = ?
		WHERE id = ?
	`, status, now, reason, id)
	return err
}

// ExpireOld marks all expired pending requests as expired. Returns the number of expired requests.
func (r *DispatchRepository) ExpireOld(ctx context.Context) (int, error) {
	now := time.Now()
	result, err := r.db.ExecContext(ctx, `
		UPDATE dispatch_requests
		SET status = 'expired', responded_at = ?
		WHERE status = 'pending' AND expires_at < ?
	`, now, now)
	if err != nil {
		return 0, err
	}
	n, _ := result.RowsAffected()
	return int(n), nil
}

// AutoDispatchSettingsRepository handles auto-dispatch settings database operations.
type AutoDispatchSettingsRepository struct {
	db *sql.DB
}

// NewAutoDispatchSettingsRepository creates a new auto-dispatch settings repository.
func NewAutoDispatchSettingsRepository(db *sql.DB) *AutoDispatchSettingsRepository {
	return &AutoDispatchSettingsRepository{db: db}
}

// Get retrieves auto-dispatch settings for a printer.
func (r *AutoDispatchSettingsRepository) Get(ctx context.Context, printerID uuid.UUID) (*model.AutoDispatchSettings, error) {
	var s model.AutoDispatchSettings
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT printer_id, enabled, require_confirmation, auto_start, timeout_minutes, updated_at
		FROM auto_dispatch_settings WHERE printer_id = ?
	`, printerID), &s.PrinterID, &s.Enabled, &s.RequireConfirmation, &s.AutoStart, &s.TimeoutMinutes, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		// Return defaults if no settings exist
		return &model.AutoDispatchSettings{
			PrinterID:           printerID,
			Enabled:             false,
			RequireConfirmation: true,
			AutoStart:           false,
			TimeoutMinutes:      30,
			UpdatedAt:           time.Now(),
		}, nil
	}
	return &s, err
}

// Upsert creates or updates auto-dispatch settings for a printer.
func (r *AutoDispatchSettingsRepository) Upsert(ctx context.Context, settings *model.AutoDispatchSettings) error {
	settings.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO auto_dispatch_settings (printer_id, enabled, require_confirmation, auto_start, timeout_minutes, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(printer_id) DO UPDATE SET
			enabled = excluded.enabled,
			require_confirmation = excluded.require_confirmation,
			auto_start = excluded.auto_start,
			timeout_minutes = excluded.timeout_minutes,
			updated_at = excluded.updated_at
	`, settings.PrinterID, settings.Enabled, settings.RequireConfirmation, settings.AutoStart, settings.TimeoutMinutes, settings.UpdatedAt)
	return err
}

// Delete removes auto-dispatch settings for a printer.
func (r *AutoDispatchSettingsRepository) Delete(ctx context.Context, printerID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM auto_dispatch_settings WHERE printer_id = ?`, printerID)
	return err
}
