package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/philjestin/daedalus/internal/crypto"
	"github.com/philjestin/daedalus/internal/model"
)

// DBTX is an interface for database operations that works with both *sql.DB and *sql.Tx.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Repositories holds all repository instances.
type Repositories struct {
	db                   *sql.DB
	Projects             *ProjectRepository
	Parts                *PartRepository
	Designs              *DesignRepository
	Printers             *PrinterRepository
	Materials            *MaterialRepository
	Spools               *SpoolRepository
	PrintJobs            *PrintJobRepository
	Files                *FileRepository
	Expenses             *ExpenseRepository
	Sales                *SaleRepository
	Templates            *TemplateRepository
	Etsy                 *EtsyRepository
	Squarespace          *SquarespaceRepository
	BambuCloud           *BambuCloudRepository
	Settings             *SettingsRepository
	ProjectSupplies      *ProjectSupplyRepository
	Dispatch             *DispatchRepository
	AutoDispatchSettings *AutoDispatchSettingsRepository
	// New repositories for feature gaps
	Orders          *OrderRepository
	Tags            *TagRepository
	AlertDismissals *AlertDismissalRepository
	Shopify         *ShopifyRepository
	Tasks           *TaskRepository
	TaskChecklist   *TaskChecklistRepository
	Feedback        *FeedbackRepository
	Customers       *CustomerRepository
	Quotes          *QuoteRepository
}

// WithTransaction executes a function within a database transaction.
// If the function returns an error, the transaction is rolled back.
// If the function succeeds, the transaction is committed.
func (r *Repositories) WithTransaction(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	err = fn(tx)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return rbErr
		}
		return err
	}

	return tx.Commit()
}

// NewRepositories creates all repository instances.
func NewRepositories(db *sql.DB) *Repositories {
	return &Repositories{
		db:                   db,
		Projects:             &ProjectRepository{db: db},
		Parts:                &PartRepository{db: db},
		Designs:              &DesignRepository{db: db},
		Printers:             &PrinterRepository{db: db},
		Materials:            &MaterialRepository{db: db},
		Spools:               &SpoolRepository{db: db},
		PrintJobs:            &PrintJobRepository{db: db},
		Files:                &FileRepository{db: db},
		Expenses:             &ExpenseRepository{db: db},
		Sales:                &SaleRepository{db: db},
		Templates:            &TemplateRepository{db: db},
		Etsy:                 &EtsyRepository{db: db},
		Squarespace:          &SquarespaceRepository{db: db},
		BambuCloud:           &BambuCloudRepository{db: db},
		Settings:             &SettingsRepository{db: db},
		ProjectSupplies:      &ProjectSupplyRepository{db: db},
		Dispatch:             &DispatchRepository{db: db},
		AutoDispatchSettings: &AutoDispatchSettingsRepository{db: db},
		// New repositories for feature gaps
		Orders:          &OrderRepository{db: db},
		Tags:            &TagRepository{db: db},
		AlertDismissals: &AlertDismissalRepository{db: db},
		Shopify:         &ShopifyRepository{db: db},
		Tasks:           &TaskRepository{db: db},
		TaskChecklist:   &TaskChecklistRepository{db: db},
		Feedback:        &FeedbackRepository{db: db},
		Customers:       &CustomerRepository{db: db},
		Quotes:          &QuoteRepository{db: db},
	}
}

// marshalStringArray serializes a []string to a JSON string for SQLite TEXT storage.
func marshalStringArray(arr []string) string {
	if arr == nil {
		return "[]"
	}
	b, _ := json.Marshal(arr)
	return string(b)
}

// unmarshalStringArray deserializes a JSON string from SQLite TEXT into []string.
func unmarshalStringArray(data []byte) []string {
	if data == nil {
		return []string{}
	}
	var arr []string
	json.Unmarshal(data, &arr)
	if arr == nil {
		return []string{}
	}
	return arr
}

// ProjectRepository handles project database operations.
type ProjectRepository struct {
	db *sql.DB
}

// Create inserts a new project.
func (r *ProjectRepository) Create(ctx context.Context, p *model.Project) error {
	p.ID = uuid.New()
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	if p.Tags == nil {
		p.Tags = []string{}
	}
	if p.Source == "" {
		p.Source = "manual"
	}

	tagsJSON := marshalStringArray(p.Tags)
	allowedPrinterIDsJSON, _ := json.Marshal(p.AllowedPrinterIDs)
	defaultSettingsJSON, _ := json.Marshal(p.DefaultSettings)

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO projects (id, name, description, target_date, tags, template_id, source, external_order_id, customer_notes, sku, price_cents, printer_type, allowed_printer_ids, default_settings, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, p.ID, p.Name, p.Description, p.TargetDate, tagsJSON, p.TemplateID, p.Source, p.ExternalOrderID, p.CustomerNotes, p.SKU, p.PriceCents, p.PrinterType, allowedPrinterIDsJSON, defaultSettingsJSON, p.Notes, p.CreatedAt, p.UpdatedAt)
	return err
}

// GetByID retrieves a project by ID.
func (r *ProjectRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Project, error) {
	var p model.Project
	var tagsJSON, allowedPrinterIDsJSON, defaultSettingsJSON []byte
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, name, description, target_date, tags, template_id, source, external_order_id, customer_notes, sku, price_cents, printer_type, allowed_printer_ids, default_settings, notes, created_at, updated_at
		FROM projects WHERE id = ?
	`, id), &p.ID, &p.Name, &p.Description, &p.TargetDate, &tagsJSON, &p.TemplateID, &p.Source, &p.ExternalOrderID, &p.CustomerNotes, &p.SKU, &p.PriceCents, &p.PrinterType, &allowedPrinterIDsJSON, &defaultSettingsJSON, &p.Notes, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.Tags = unmarshalStringArray(tagsJSON)
	if allowedPrinterIDsJSON != nil {
		json.Unmarshal(allowedPrinterIDsJSON, &p.AllowedPrinterIDs)
	}
	if defaultSettingsJSON != nil {
		json.Unmarshal(defaultSettingsJSON, &p.DefaultSettings)
	}
	return &p, nil
}

// List retrieves all projects.
func (r *ProjectRepository) List(ctx context.Context) ([]model.Project, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, description, target_date, tags, template_id, source, external_order_id, customer_notes, sku, price_cents, printer_type, allowed_printer_ids, default_settings, notes, created_at, updated_at
		FROM projects ORDER BY updated_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []model.Project
	for rows.Next() {
		var p model.Project
		var tagsJSON, allowedPrinterIDsJSON, defaultSettingsJSON []byte
		if err := scanRow(rows, &p.ID, &p.Name, &p.Description, &p.TargetDate, &tagsJSON, &p.TemplateID, &p.Source, &p.ExternalOrderID, &p.CustomerNotes, &p.SKU, &p.PriceCents, &p.PrinterType, &allowedPrinterIDsJSON, &defaultSettingsJSON, &p.Notes, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.Tags = unmarshalStringArray(tagsJSON)
		if allowedPrinterIDsJSON != nil {
			json.Unmarshal(allowedPrinterIDsJSON, &p.AllowedPrinterIDs)
		}
		if defaultSettingsJSON != nil {
			json.Unmarshal(defaultSettingsJSON, &p.DefaultSettings)
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

// Update updates a project.
func (r *ProjectRepository) Update(ctx context.Context, p *model.Project) error {
	p.UpdatedAt = time.Now()
	tagsJSON := marshalStringArray(p.Tags)
	allowedPrinterIDsJSON, _ := json.Marshal(p.AllowedPrinterIDs)
	defaultSettingsJSON, _ := json.Marshal(p.DefaultSettings)

	_, err := r.db.ExecContext(ctx, `
		UPDATE projects SET name = ?, description = ?, target_date = ?, tags = ?, template_id = ?, source = ?, external_order_id = ?, customer_notes = ?, sku = ?, price_cents = ?, printer_type = ?, allowed_printer_ids = ?, default_settings = ?, notes = ?, updated_at = ?
		WHERE id = ?
	`, p.Name, p.Description, p.TargetDate, tagsJSON, p.TemplateID, p.Source, p.ExternalOrderID, p.CustomerNotes, p.SKU, p.PriceCents, p.PrinterType, allowedPrinterIDsJSON, defaultSettingsJSON, p.Notes, p.UpdatedAt, p.ID)
	return err
}

// Delete removes a project.
func (r *ProjectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, id)
	return err
}

// ListByTemplateID retrieves all projects created from a given template (legacy).
func (r *ProjectRepository) ListByTemplateID(ctx context.Context, templateID uuid.UUID) ([]model.Project, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, description, target_date, tags, template_id, source, external_order_id, customer_notes, sku, price_cents, printer_type, allowed_printer_ids, default_settings, notes, created_at, updated_at
		FROM projects WHERE template_id = ? ORDER BY created_at DESC
	`, templateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []model.Project
	for rows.Next() {
		var p model.Project
		var tagsJSON, allowedPrinterIDsJSON, defaultSettingsJSON []byte
		if err := scanRow(rows, &p.ID, &p.Name, &p.Description, &p.TargetDate, &tagsJSON, &p.TemplateID, &p.Source, &p.ExternalOrderID, &p.CustomerNotes, &p.SKU, &p.PriceCents, &p.PrinterType, &allowedPrinterIDsJSON, &defaultSettingsJSON, &p.Notes, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.Tags = unmarshalStringArray(tagsJSON)
		if allowedPrinterIDsJSON != nil {
			json.Unmarshal(allowedPrinterIDsJSON, &p.AllowedPrinterIDs)
		}
		if defaultSettingsJSON != nil {
			json.Unmarshal(defaultSettingsJSON, &p.DefaultSettings)
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

// GetBySKU retrieves a project by SKU.
func (r *ProjectRepository) GetBySKU(ctx context.Context, sku string) (*model.Project, error) {
	var p model.Project
	var tagsJSON, allowedPrinterIDsJSON, defaultSettingsJSON []byte
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, name, description, target_date, tags, template_id, source, external_order_id, customer_notes, sku, price_cents, printer_type, allowed_printer_ids, default_settings, notes, created_at, updated_at
		FROM projects WHERE sku = ?
	`, sku), &p.ID, &p.Name, &p.Description, &p.TargetDate, &tagsJSON, &p.TemplateID, &p.Source, &p.ExternalOrderID, &p.CustomerNotes, &p.SKU, &p.PriceCents, &p.PrinterType, &allowedPrinterIDsJSON, &defaultSettingsJSON, &p.Notes, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	p.Tags = unmarshalStringArray(tagsJSON)
	if allowedPrinterIDsJSON != nil {
		json.Unmarshal(allowedPrinterIDsJSON, &p.AllowedPrinterIDs)
	}
	if defaultSettingsJSON != nil {
		json.Unmarshal(defaultSettingsJSON, &p.DefaultSettings)
	}
	return &p, nil
}

// PartRepository handles part database operations.
type PartRepository struct {
	db *sql.DB
}

// Create inserts a new part.
func (r *PartRepository) Create(ctx context.Context, p *model.Part) error {
	p.ID = uuid.New()
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	if p.Status == "" {
		p.Status = model.PartStatusDesign
	}
	if p.Quantity == 0 {
		p.Quantity = 1
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO parts (id, project_id, name, description, quantity, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, p.ID, p.ProjectID, p.Name, p.Description, p.Quantity, p.Status, p.CreatedAt, p.UpdatedAt)
	return err
}

// GetByID retrieves a part by ID.
func (r *PartRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Part, error) {
	var p model.Part
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, project_id, name, description, quantity, status, created_at, updated_at
		FROM parts WHERE id = ?
	`, id), &p.ID, &p.ProjectID, &p.Name, &p.Description, &p.Quantity, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &p, err
}

// ListByProject retrieves all parts for a project.
func (r *PartRepository) ListByProject(ctx context.Context, projectID uuid.UUID) ([]model.Part, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, name, description, quantity, status, created_at, updated_at
		FROM parts WHERE project_id = ? ORDER BY created_at ASC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var parts []model.Part
	for rows.Next() {
		var p model.Part
		if err := scanRow(rows, &p.ID, &p.ProjectID, &p.Name, &p.Description, &p.Quantity, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		parts = append(parts, p)
	}
	return parts, rows.Err()
}

// Update updates a part.
func (r *PartRepository) Update(ctx context.Context, p *model.Part) error {
	p.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE parts SET name = ?, description = ?, quantity = ?, status = ?, updated_at = ?
		WHERE id = ?
	`, p.Name, p.Description, p.Quantity, p.Status, p.UpdatedAt, p.ID)
	return err
}

// Delete removes a part.
func (r *PartRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM parts WHERE id = ?`, id)
	return err
}

// TaskRepository handles task database operations.
type TaskRepository struct {
	db *sql.DB
}

// Create inserts a new task.
func (r *TaskRepository) Create(ctx context.Context, t *model.Task) error {
	t.ID = uuid.New()
	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()
	if t.Status == "" {
		t.Status = model.TaskStatusPending
	}
	if t.Quantity == 0 {
		t.Quantity = 1
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO tasks (id, project_id, order_id, order_item_id, name, status, quantity, notes, pickup_date, created_at, updated_at, started_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, t.ID, t.ProjectID, t.OrderID, t.OrderItemID, t.Name, t.Status, t.Quantity, t.Notes, t.PickupDate, t.CreatedAt, t.UpdatedAt, t.StartedAt, t.CompletedAt)
	return err
}

// GetByID retrieves a task by ID.
func (r *TaskRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Task, error) {
	var t model.Task
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, project_id, order_id, order_item_id, name, status, quantity, notes, pickup_date, created_at, updated_at, started_at, completed_at
		FROM tasks WHERE id = ?
	`, id), &t.ID, &t.ProjectID, &t.OrderID, &t.OrderItemID, &t.Name, &t.Status, &t.Quantity, &t.Notes, &t.PickupDate, &t.CreatedAt, &t.UpdatedAt, &t.StartedAt, &t.CompletedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &t, err
}

// List retrieves tasks with optional filters.
func (r *TaskRepository) List(ctx context.Context, filters model.TaskFilters) ([]model.Task, error) {
	query := `SELECT id, project_id, order_id, order_item_id, name, status, quantity, notes, pickup_date, created_at, updated_at, started_at, completed_at FROM tasks WHERE 1=1`
	args := []interface{}{}

	if filters.ProjectID != nil {
		query += ` AND project_id = ?`
		args = append(args, *filters.ProjectID)
	}
	if filters.OrderID != nil {
		query += ` AND order_id = ?`
		args = append(args, *filters.OrderID)
	}
	if filters.Status != nil {
		query += ` AND status = ?`
		args = append(args, *filters.Status)
	}

	query += ` ORDER BY created_at DESC`

	if filters.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, filters.Limit)
	}
	if filters.Offset > 0 {
		query += ` OFFSET ?`
		args = append(args, filters.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []model.Task
	for rows.Next() {
		var t model.Task
		if err := scanRow(rows, &t.ID, &t.ProjectID, &t.OrderID, &t.OrderItemID, &t.Name, &t.Status, &t.Quantity, &t.Notes, &t.PickupDate, &t.CreatedAt, &t.UpdatedAt, &t.StartedAt, &t.CompletedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	return tasks, rows.Err()
}

// ListByProject retrieves all tasks for a project.
func (r *TaskRepository) ListByProject(ctx context.Context, projectID uuid.UUID) ([]model.Task, error) {
	return r.List(ctx, model.TaskFilters{ProjectID: &projectID})
}

// ListByOrder retrieves all tasks for an order.
func (r *TaskRepository) ListByOrder(ctx context.Context, orderID uuid.UUID) ([]model.Task, error) {
	return r.List(ctx, model.TaskFilters{OrderID: &orderID})
}

// Update updates a task.
func (r *TaskRepository) Update(ctx context.Context, t *model.Task) error {
	t.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE tasks SET project_id = ?, order_id = ?, order_item_id = ?, name = ?, status = ?, quantity = ?, notes = ?, pickup_date = ?, updated_at = ?, started_at = ?, completed_at = ?
		WHERE id = ?
	`, t.ProjectID, t.OrderID, t.OrderItemID, t.Name, t.Status, t.Quantity, t.Notes, t.PickupDate, t.UpdatedAt, t.StartedAt, t.CompletedAt, t.ID)
	return err
}

// TaskChecklistRepository handles task checklist item database operations.
type TaskChecklistRepository struct {
	db *sql.DB
}

// Create inserts a single checklist item.
func (r *TaskChecklistRepository) Create(ctx context.Context, item *model.TaskChecklistItem) error {
	item.ID = uuid.New()
	item.CreatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO task_checklist_items (id, task_id, name, part_id, sort_order, completed, completed_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, item.ID, item.TaskID, item.Name, item.PartID, item.SortOrder, item.Completed, item.CompletedAt, item.CreatedAt)
	return err
}

// CreateBatch inserts multiple checklist items.
func (r *TaskChecklistRepository) CreateBatch(ctx context.Context, items []model.TaskChecklistItem) error {
	for i := range items {
		if err := r.Create(ctx, &items[i]); err != nil {
			return err
		}
	}
	return nil
}

// ListByTask retrieves all checklist items for a task, ordered by sort_order.
func (r *TaskChecklistRepository) ListByTask(ctx context.Context, taskID uuid.UUID) ([]model.TaskChecklistItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, task_id, name, part_id, sort_order, completed, completed_at, created_at
		FROM task_checklist_items WHERE task_id = ? ORDER BY sort_order ASC
	`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.TaskChecklistItem
	for rows.Next() {
		var item model.TaskChecklistItem
		if err := scanRow(rows, &item.ID, &item.TaskID, &item.Name, &item.PartID, &item.SortOrder, &item.Completed, &item.CompletedAt, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// UpdateCompleted toggles the completed state of a checklist item.
func (r *TaskChecklistRepository) UpdateCompleted(ctx context.Context, id uuid.UUID, completed bool) error {
	var completedAt interface{}
	if completed {
		now := time.Now()
		completedAt = now
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE task_checklist_items SET completed = ?, completed_at = ? WHERE id = ?
	`, completed, completedAt, id)
	return err
}

// DeleteByTask removes all checklist items for a task.
func (r *TaskChecklistRepository) DeleteByTask(ctx context.Context, taskID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM task_checklist_items WHERE task_id = ?`, taskID)
	return err
}

// UpdateStatus updates only the task status and related timestamps.
func (r *TaskRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.TaskStatus) error {
	now := time.Now()
	var startedAt, completedAt interface{}

	switch status {
	case model.TaskStatusInProgress:
		startedAt = now
	case model.TaskStatusCompleted, model.TaskStatusCancelled:
		completedAt = now
	}

	_, err := r.db.ExecContext(ctx, `
		UPDATE tasks SET status = ?, updated_at = ?,
			started_at = COALESCE(?, started_at),
			completed_at = COALESCE(?, completed_at)
		WHERE id = ?
	`, status, now, startedAt, completedAt, id)
	return err
}

// Delete removes a task.
func (r *TaskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, id)
	return err
}

// GetProjectTaskStats returns task statistics for a project.
func (r *TaskRepository) GetProjectTaskStats(ctx context.Context, projectID uuid.UUID) (total int, completed int, err error) {
	err = r.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END), 0)
		FROM tasks WHERE project_id = ?
	`, projectID).Scan(&total, &completed)
	return
}

// GetPendingSalesStats returns the count and estimated revenue of tasks in pending/in_progress status.
// Revenue is estimated from project price_cents if set, otherwise from average gross_cents of past sales.
func (r *TaskRepository) GetPendingSalesStats(ctx context.Context) (count int, revenueCents int, err error) {
	err = r.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(t.quantity), 0),
			COALESCE(SUM(t.quantity * COALESCE(
				p.price_cents,
				(SELECT CASE WHEN COUNT(*) > 0 THEN SUM(s.gross_cents) / COUNT(*) ELSE 0 END
				 FROM sales s WHERE s.project_id = p.id),
				0
			)), 0)
		FROM tasks t
		JOIN projects p ON t.project_id = p.id
		WHERE t.status IN ('pending', 'in_progress')
	`).Scan(&count, &revenueCents)
	return
}

// DesignRepository handles design database operations.
type DesignRepository struct {
	db *sql.DB
}

// Create inserts a new design version.
func (r *DesignRepository) Create(ctx context.Context, d *model.Design) error {
	d.ID = uuid.New()
	d.CreatedAt = time.Now()

	// Get next version number for this part
	var maxVersion int
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(version), 0) FROM designs WHERE part_id = ?
	`, d.PartID), &maxVersion)
	if err != nil {
		return err
	}
	d.Version = maxVersion + 1

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO designs (id, part_id, version, file_id, file_name, file_hash, file_size_bytes, file_type, notes, slice_profile, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, d.ID, d.PartID, d.Version, d.FileID, d.FileName, d.FileHash, d.FileSizeBytes, d.FileType, d.Notes, d.SliceProfile, d.CreatedAt)
	return err
}

// GetByID retrieves a design by ID.
func (r *DesignRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Design, error) {
	var d model.Design
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, part_id, version, file_id, file_name, file_hash, file_size_bytes, file_type, notes, slice_profile, created_at
		FROM designs WHERE id = ?
	`, id), &d.ID, &d.PartID, &d.Version, &d.FileID, &d.FileName, &d.FileHash, &d.FileSizeBytes, &d.FileType, &d.Notes, &d.SliceProfile, &d.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &d, err
}

// ListByPart retrieves all designs for a part.
func (r *DesignRepository) ListByPart(ctx context.Context, partID uuid.UUID) ([]model.Design, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, part_id, version, file_id, file_name, file_hash, file_size_bytes, file_type, notes, slice_profile, created_at
		FROM designs WHERE part_id = ? ORDER BY version DESC
	`, partID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var designs []model.Design
	for rows.Next() {
		var d model.Design
		if err := scanRow(rows, &d.ID, &d.PartID, &d.Version, &d.FileID, &d.FileName, &d.FileHash, &d.FileSizeBytes, &d.FileType, &d.Notes, &d.SliceProfile, &d.CreatedAt); err != nil {
			return nil, err
		}
		designs = append(designs, d)
	}
	return designs, rows.Err()
}

// PrinterRepository handles printer database operations.
type PrinterRepository struct {
	db *sql.DB
}

// Create inserts a new printer.
func (r *PrinterRepository) Create(ctx context.Context, p *model.Printer) error {
	p.ID = uuid.New()
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	if p.Status == "" {
		p.Status = model.PrinterStatusOffline
	}
	if p.MinMaterialPercent == 0 {
		p.MinMaterialPercent = 10 // Default 10%
	}

	buildVolumeJSON, _ := json.Marshal(p.BuildVolume)

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO printers (id, name, model, manufacturer, connection_type, connection_uri, api_key, serial_number, status, build_volume, nozzle_diameter, location, notes, min_material_percent, cost_per_hour_cents, purchase_price_cents, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, p.ID, p.Name, p.Model, p.Manufacturer, p.ConnectionType, p.ConnectionURI, p.APIKey, p.SerialNumber, p.Status, buildVolumeJSON, p.NozzleDiameter, p.Location, p.Notes, p.MinMaterialPercent, p.CostPerHourCents, p.PurchasePriceCents, p.CreatedAt, p.UpdatedAt)
	return err
}

// GetByID retrieves a printer by ID.
func (r *PrinterRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Printer, error) {
	var p model.Printer
	var buildVolumeJSON []byte
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, name, model, manufacturer, connection_type, connection_uri, api_key, serial_number, status, build_volume, nozzle_diameter, location, notes, min_material_percent, cost_per_hour_cents, purchase_price_cents, created_at, updated_at
		FROM printers WHERE id = ?
	`, id), &p.ID, &p.Name, &p.Model, &p.Manufacturer, &p.ConnectionType, &p.ConnectionURI, &p.APIKey, &p.SerialNumber, &p.Status, &buildVolumeJSON, &p.NozzleDiameter, &p.Location, &p.Notes, &p.MinMaterialPercent, &p.CostPerHourCents, &p.PurchasePriceCents, &p.CreatedAt, &p.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if buildVolumeJSON != nil {
		json.Unmarshal(buildVolumeJSON, &p.BuildVolume)
	}
	return &p, nil
}

// List retrieves all printers.
func (r *PrinterRepository) List(ctx context.Context) ([]model.Printer, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, model, manufacturer, connection_type, connection_uri, api_key, serial_number, status, build_volume, nozzle_diameter, location, notes, min_material_percent, cost_per_hour_cents, purchase_price_cents, created_at, updated_at
		FROM printers ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var printers []model.Printer
	for rows.Next() {
		var p model.Printer
		var buildVolumeJSON []byte
		if err := scanRow(rows, &p.ID, &p.Name, &p.Model, &p.Manufacturer, &p.ConnectionType, &p.ConnectionURI, &p.APIKey, &p.SerialNumber, &p.Status, &buildVolumeJSON, &p.NozzleDiameter, &p.Location, &p.Notes, &p.MinMaterialPercent, &p.CostPerHourCents, &p.PurchasePriceCents, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		if buildVolumeJSON != nil {
			json.Unmarshal(buildVolumeJSON, &p.BuildVolume)
		}
		printers = append(printers, p)
	}
	return printers, rows.Err()
}

// Update updates a printer.
func (r *PrinterRepository) Update(ctx context.Context, p *model.Printer) error {
	p.UpdatedAt = time.Now()
	buildVolumeJSON, _ := json.Marshal(p.BuildVolume)

	_, err := r.db.ExecContext(ctx, `
		UPDATE printers SET name = ?, model = ?, manufacturer = ?, connection_type = ?, connection_uri = ?, api_key = ?, serial_number = ?, status = ?, build_volume = ?, nozzle_diameter = ?, location = ?, notes = ?, min_material_percent = ?, cost_per_hour_cents = ?, purchase_price_cents = ?, updated_at = ?
		WHERE id = ?
	`, p.Name, p.Model, p.Manufacturer, p.ConnectionType, p.ConnectionURI, p.APIKey, p.SerialNumber, p.Status, buildVolumeJSON, p.NozzleDiameter, p.Location, p.Notes, p.MinMaterialPercent, p.CostPerHourCents, p.PurchasePriceCents, p.UpdatedAt, p.ID)
	return err
}

// UpdateStatus updates only the printer status.
func (r *PrinterRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.PrinterStatus) error {
	_, err := r.db.ExecContext(ctx, `UPDATE printers SET status = ?, updated_at = ? WHERE id = ?`, status, time.Now(), id)
	return err
}

// Delete removes a printer.
func (r *PrinterRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM printers WHERE id = ?`, id)
	return err
}

// PrinterUtilizationData holds raw data for computing utilization metrics.
type PrinterUtilizationData struct {
	CompletedSeconds int
	FailedSeconds    int
	CompletedJobs    int
	FailedJobs       int
	TotalJobs        int
}

// GetPrinterUtilizationData retrieves utilization data for a printer since a given time.
func (r *PrinterRepository) GetPrinterUtilizationData(ctx context.Context, printerID uuid.UUID, since time.Time) (*PrinterUtilizationData, error) {
	var data PrinterUtilizationData
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COALESCE(SUM(CASE WHEN status = 'completed' THEN COALESCE(actual_seconds, 0) ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'failed' THEN COALESCE(actual_seconds, estimated_seconds, 0) ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0),
			COUNT(*)
		FROM print_jobs
		WHERE printer_id = ? AND created_at >= ?
	`, printerID, since).Scan(&data.CompletedSeconds, &data.FailedSeconds, &data.CompletedJobs, &data.FailedJobs, &data.TotalJobs)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

// PrinterHealthData holds raw data for computing health metrics.
type PrinterHealthData struct {
	TotalJobs          int
	CompletedJobs      int
	FailedJobs         int
	TotalSeconds       int
	TotalCostCents     int
	TotalMaterialGrams float64
}

// GetPrinterHealthData retrieves lifetime health data for a printer.
func (r *PrinterRepository) GetPrinterHealthData(ctx context.Context, printerID uuid.UUID) (*PrinterHealthData, error) {
	var data PrinterHealthData
	err := r.db.QueryRowContext(ctx, `
		SELECT
			COUNT(*),
			COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'completed' THEN COALESCE(actual_seconds, 0) ELSE 0 END), 0),
			COALESCE(SUM(COALESCE(cost_cents, 0)), 0),
			COALESCE(SUM(COALESCE(material_used_grams, 0)), 0)
		FROM print_jobs
		WHERE printer_id = ?
	`, printerID).Scan(&data.TotalJobs, &data.CompletedJobs, &data.FailedJobs, &data.TotalSeconds, &data.TotalCostCents, &data.TotalMaterialGrams)
	if err != nil {
		return nil, err
	}
	return &data, nil
}

// GetPrinterFailureBreakdown retrieves failure category counts for a printer.
func (r *PrinterRepository) GetPrinterFailureBreakdown(ctx context.Context, printerID uuid.UUID) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT COALESCE(failure_category, 'unknown'), COUNT(*)
		FROM print_jobs
		WHERE printer_id = ? AND status = 'failed'
		GROUP BY failure_category
	`, printerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	breakdown := make(map[string]int)
	for rows.Next() {
		var category string
		var count int
		if err := rows.Scan(&category, &count); err != nil {
			return nil, err
		}
		breakdown[category] = count
	}
	return breakdown, rows.Err()
}

// GetPrinterRevenueAttribution computes attributed revenue for a printer.
// This traces: Printer → Jobs → Projects → Sales with proportional attribution.
func (r *PrinterRepository) GetPrinterRevenueAttribution(ctx context.Context, printerID uuid.UUID) (int, error) {
	var revenueCents int
	err := r.db.QueryRowContext(ctx, `
		WITH printer_project_jobs AS (
			SELECT project_id, COUNT(*) as printer_jobs
			FROM print_jobs
			WHERE printer_id = ? AND project_id IS NOT NULL AND status = 'completed'
			GROUP BY project_id
		),
		project_total_jobs AS (
			SELECT project_id, COUNT(*) as total_jobs
			FROM print_jobs
			WHERE project_id IS NOT NULL AND status = 'completed'
			GROUP BY project_id
		),
		project_sales AS (
			SELECT project_id, SUM(gross_cents) as gross
			FROM sales
			WHERE project_id IS NOT NULL
			GROUP BY project_id
		)
		SELECT COALESCE(SUM(
			CAST(ps.gross AS REAL) * CAST(ppj.printer_jobs AS REAL) / CAST(ptj.total_jobs AS REAL)
		), 0)
		FROM printer_project_jobs ppj
		JOIN project_total_jobs ptj ON ppj.project_id = ptj.project_id
		JOIN project_sales ps ON ppj.project_id = ps.project_id
	`, printerID).Scan(&revenueCents)
	if err != nil {
		return 0, err
	}
	return revenueCents, nil
}

// MaterialRepository handles material database operations.
type MaterialRepository struct {
	db *sql.DB
}

// Create inserts a new material.
func (r *MaterialRepository) Create(ctx context.Context, m *model.Material) error {
	return r.CreateTx(ctx, r.db, m)
}

// CreateTx inserts a new material using the provided DBTX (supports transactions).
func (r *MaterialRepository) CreateTx(ctx context.Context, db DBTX, m *model.Material) error {
	m.ID = uuid.New()
	m.CreatedAt = time.Now()
	m.UpdatedAt = time.Now()

	printTempJSON, _ := json.Marshal(m.PrintTemp)
	bedTempJSON, _ := json.Marshal(m.BedTemp)

	_, err := db.ExecContext(ctx, `
		INSERT INTO materials (id, name, type, manufacturer, color, color_hex, density, cost_per_kg, print_temp, bed_temp, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, m.ID, m.Name, m.Type, m.Manufacturer, m.Color, m.ColorHex, m.Density, m.CostPerKg, printTempJSON, bedTempJSON, m.Notes, m.CreatedAt, m.UpdatedAt)
	return err
}

// GetByID retrieves a material by ID.
func (r *MaterialRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Material, error) {
	var m model.Material
	var printTempJSON, bedTempJSON []byte
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, name, type, manufacturer, color, color_hex, density, cost_per_kg, print_temp, bed_temp, notes, low_threshold_grams, created_at, updated_at
		FROM materials WHERE id = ?
	`, id), &m.ID, &m.Name, &m.Type, &m.Manufacturer, &m.Color, &m.ColorHex, &m.Density, &m.CostPerKg, &printTempJSON, &bedTempJSON, &m.Notes, &m.LowThresholdGrams, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if printTempJSON != nil {
		json.Unmarshal(printTempJSON, &m.PrintTemp)
	}
	if bedTempJSON != nil {
		json.Unmarshal(bedTempJSON, &m.BedTemp)
	}
	return &m, nil
}

// List retrieves all materials.
func (r *MaterialRepository) List(ctx context.Context) ([]model.Material, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, type, manufacturer, color, color_hex, density, cost_per_kg, print_temp, bed_temp, notes, low_threshold_grams, created_at, updated_at
		FROM materials ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var materials []model.Material
	for rows.Next() {
		var m model.Material
		var printTempJSON, bedTempJSON []byte
		if err := scanRow(rows, &m.ID, &m.Name, &m.Type, &m.Manufacturer, &m.Color, &m.ColorHex, &m.Density, &m.CostPerKg, &printTempJSON, &bedTempJSON, &m.Notes, &m.LowThresholdGrams, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		if printTempJSON != nil {
			json.Unmarshal(printTempJSON, &m.PrintTemp)
		}
		if bedTempJSON != nil {
			json.Unmarshal(bedTempJSON, &m.BedTemp)
		}
		materials = append(materials, m)
	}
	return materials, rows.Err()
}

// Delete removes a material by ID, clearing any foreign key references first.
func (r *MaterialRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Clear FK references from expense items
	if _, err := tx.ExecContext(ctx, `UPDATE expense_items SET matched_material_id = NULL WHERE matched_material_id = ?`, id); err != nil {
		return err
	}
	// Clear FK references from project supplies
	if _, err := tx.ExecContext(ctx, `UPDATE project_supplies SET material_id = NULL WHERE material_id = ?`, id); err != nil {
		return err
	}
	// Delete any spools tied to this material
	if _, err := tx.ExecContext(ctx, `DELETE FROM material_spools WHERE material_id = ?`, id); err != nil {
		return err
	}
	// Delete the material
	if _, err := tx.ExecContext(ctx, `DELETE FROM materials WHERE id = ?`, id); err != nil {
		return err
	}

	return tx.Commit()
}

// Update updates an existing material.
func (r *MaterialRepository) Update(ctx context.Context, m *model.Material) error {
	printTempJSON, _ := json.Marshal(m.PrintTemp)
	bedTempJSON, _ := json.Marshal(m.BedTemp)
	m.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE materials SET name = ?, type = ?, manufacturer = ?, color = ?, color_hex = ?, density = ?,
			cost_per_kg = ?, print_temp = ?, bed_temp = ?, notes = ?, low_threshold_grams = ?, updated_at = ?
		WHERE id = ?
	`, m.Name, m.Type, m.Manufacturer, m.Color, m.ColorHex, m.Density, m.CostPerKg, printTempJSON, bedTempJSON, m.Notes, m.LowThresholdGrams, m.UpdatedAt, m.ID)
	return err
}

// FindByTypeManufacturerColor finds a material matching the given type, manufacturer, and color.
// Returns nil if no match is found.
func (r *MaterialRepository) FindByTypeManufacturerColor(ctx context.Context, matType model.MaterialType, manufacturer, color string) (*model.Material, error) {
	var m model.Material
	var printTempJSON, bedTempJSON []byte
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, name, type, manufacturer, color, color_hex, density, cost_per_kg, print_temp, bed_temp, notes, low_threshold_grams, created_at, updated_at
		FROM materials WHERE LOWER(type) = LOWER(?) AND LOWER(manufacturer) = LOWER(?) AND LOWER(color) = LOWER(?)
		LIMIT 1
	`, matType, manufacturer, color), &m.ID, &m.Name, &m.Type, &m.Manufacturer, &m.Color, &m.ColorHex, &m.Density, &m.CostPerKg, &printTempJSON, &bedTempJSON, &m.Notes, &m.LowThresholdGrams, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if printTempJSON != nil {
		json.Unmarshal(printTempJSON, &m.PrintTemp)
	}
	if bedTempJSON != nil {
		json.Unmarshal(bedTempJSON, &m.BedTemp)
	}
	return &m, nil
}

// FindByTypeAndName finds a material matching the given type and name (case-insensitive).
// Used for deduplicating supply materials.
func (r *MaterialRepository) FindByTypeAndName(ctx context.Context, matType model.MaterialType, name string) (*model.Material, error) {
	var m model.Material
	var printTempJSON, bedTempJSON []byte
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, name, type, manufacturer, color, color_hex, density, cost_per_kg, print_temp, bed_temp, notes, low_threshold_grams, created_at, updated_at
		FROM materials WHERE LOWER(type) = LOWER(?) AND LOWER(name) = LOWER(?)
		LIMIT 1
	`, matType, name), &m.ID, &m.Name, &m.Type, &m.Manufacturer, &m.Color, &m.ColorHex, &m.Density, &m.CostPerKg, &printTempJSON, &bedTempJSON, &m.Notes, &m.LowThresholdGrams, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if printTempJSON != nil {
		json.Unmarshal(printTempJSON, &m.PrintTemp)
	}
	if bedTempJSON != nil {
		json.Unmarshal(bedTempJSON, &m.BedTemp)
	}
	return &m, nil
}

// ListByType retrieves all materials of a given type, ordered by name.
func (r *MaterialRepository) ListByType(ctx context.Context, matType model.MaterialType) ([]model.Material, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, type, manufacturer, color, color_hex, density, cost_per_kg, print_temp, bed_temp, notes, low_threshold_grams, created_at, updated_at
		FROM materials WHERE LOWER(type) = LOWER(?) ORDER BY name ASC
	`, matType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var materials []model.Material
	for rows.Next() {
		var m model.Material
		var printTempJSON, bedTempJSON []byte
		if err := scanRow(rows, &m.ID, &m.Name, &m.Type, &m.Manufacturer, &m.Color, &m.ColorHex, &m.Density, &m.CostPerKg, &printTempJSON, &bedTempJSON, &m.Notes, &m.LowThresholdGrams, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, err
		}
		if printTempJSON != nil {
			json.Unmarshal(printTempJSON, &m.PrintTemp)
		}
		if bedTempJSON != nil {
			json.Unmarshal(bedTempJSON, &m.BedTemp)
		}
		materials = append(materials, m)
	}
	return materials, rows.Err()
}

// SpoolRepository handles material spool database operations.
type SpoolRepository struct {
	db *sql.DB
}

// Create inserts a new spool.
func (r *SpoolRepository) Create(ctx context.Context, s *model.MaterialSpool) error {
	return r.CreateTx(ctx, r.db, s)
}

// CreateTx inserts a new spool using the provided DBTX (supports transactions).
func (r *SpoolRepository) CreateTx(ctx context.Context, db DBTX, s *model.MaterialSpool) error {
	s.ID = uuid.New()
	s.CreatedAt = time.Now()
	s.UpdatedAt = time.Now()
	if s.Status == "" {
		s.Status = model.SpoolStatusNew
	}

	_, err := db.ExecContext(ctx, `
		INSERT INTO material_spools (id, material_id, initial_weight, remaining_weight, purchase_date, purchase_cost, location, status, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, s.ID, s.MaterialID, s.InitialWeight, s.RemainingWeight, s.PurchaseDate, s.PurchaseCost, s.Location, s.Status, s.Notes, s.CreatedAt, s.UpdatedAt)
	return err
}

// GetByID retrieves a spool by ID.
func (r *SpoolRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.MaterialSpool, error) {
	var s model.MaterialSpool
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, material_id, initial_weight, remaining_weight, purchase_date, purchase_cost, location, status, notes, created_at, updated_at
		FROM material_spools WHERE id = ?
	`, id), &s.ID, &s.MaterialID, &s.InitialWeight, &s.RemainingWeight, &s.PurchaseDate, &s.PurchaseCost, &s.Location, &s.Status, &s.Notes, &s.CreatedAt, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &s, err
}

// List retrieves all spools.
func (r *SpoolRepository) List(ctx context.Context) ([]model.MaterialSpool, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, material_id, initial_weight, remaining_weight, purchase_date, purchase_cost, location, status, notes, created_at, updated_at
		FROM material_spools WHERE status != 'archived' ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var spools []model.MaterialSpool
	for rows.Next() {
		var s model.MaterialSpool
		if err := scanRow(rows, &s.ID, &s.MaterialID, &s.InitialWeight, &s.RemainingWeight, &s.PurchaseDate, &s.PurchaseCost, &s.Location, &s.Status, &s.Notes, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		spools = append(spools, s)
	}
	return spools, rows.Err()
}

// Update updates a spool.
func (r *SpoolRepository) Update(ctx context.Context, s *model.MaterialSpool) error {
	s.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE material_spools SET
			material_id = ?,
			initial_weight = ?,
			remaining_weight = ?,
			purchase_date = ?,
			purchase_cost = ?,
			location = ?,
			status = ?,
			notes = ?,
			updated_at = ?
		WHERE id = ?
	`, s.MaterialID, s.InitialWeight, s.RemainingWeight, s.PurchaseDate, s.PurchaseCost, s.Location, s.Status, s.Notes, s.UpdatedAt, s.ID)
	return err
}

// PrintJobRepository handles print job database operations.
type PrintJobRepository struct {
	db *sql.DB
}

// Create inserts a new print job and records the initial "queued" event.
func (r *PrintJobRepository) Create(ctx context.Context, j *model.PrintJob) error {
	j.ID = uuid.New()
	j.CreatedAt = time.Now()
	if j.AttemptNumber == 0 {
		j.AttemptNumber = 1
	}
	j.Status = model.PrintJobStatusQueued // Always start as queued
	j.AutoDispatchEnabled = true          // Default to enabled

	outcomeJSON, _ := json.Marshal(j.Outcome)
	snapshotJSON, _ := json.Marshal(j.MaterialSnapshot)

	// Insert the job record
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO print_jobs (id, design_id, printer_id, material_spool_id, project_id, task_id, status, progress, started_at, completed_at, outcome, notes, created_at, recipe_id, attempt_number, parent_job_id, estimated_seconds, material_snapshot, priority, auto_dispatch_enabled)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, j.ID, j.DesignID, j.PrinterID, j.MaterialSpoolID, j.ProjectID, j.TaskID, j.Status, j.Progress, j.StartedAt, j.CompletedAt, outcomeJSON, j.Notes, j.CreatedAt, j.RecipeID, j.AttemptNumber, j.ParentJobID, j.EstimatedSeconds, snapshotJSON, j.Priority, j.AutoDispatchEnabled)
	if err != nil {
		return err
	}

	// Record the initial queued event
	status := model.PrintJobStatusQueued
	event := model.NewJobEvent(j.ID, model.JobEventQueued, &status)
	return r.AppendEvent(ctx, event)
}

// GetByID retrieves a print job by ID with current status computed from events.
func (r *PrintJobRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.PrintJob, error) {
	var j model.PrintJob
	var outcomeJSON, snapshotJSON []byte
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, design_id, printer_id, material_spool_id, project_id, task_id, status, progress, started_at, completed_at, outcome, notes, created_at,
		       recipe_id, attempt_number, parent_job_id, failure_category, estimated_seconds, actual_seconds, material_used_grams, cost_cents, printer_time_cost_cents, material_cost_cents, material_snapshot, priority, auto_dispatch_enabled
		FROM print_jobs WHERE id = ?
	`, id), &j.ID, &j.DesignID, &j.PrinterID, &j.MaterialSpoolID, &j.ProjectID, &j.TaskID, &j.Status, &j.Progress, &j.StartedAt, &j.CompletedAt, &outcomeJSON, &j.Notes, &j.CreatedAt,
		&j.RecipeID, &j.AttemptNumber, &j.ParentJobID, &j.FailureCategory, &j.EstimatedSeconds, &j.ActualSeconds, &j.MaterialUsedGrams, &j.CostCents, &j.PrinterTimeCostCents, &j.MaterialCostCents, &snapshotJSON, &j.Priority, &j.AutoDispatchEnabled)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if outcomeJSON != nil {
		json.Unmarshal(outcomeJSON, &j.Outcome)
	}
	if snapshotJSON != nil {
		json.Unmarshal(snapshotJSON, &j.MaterialSnapshot)
	}

	// Get current status from latest event
	currentStatus, currentProgress, err := r.GetCurrentStatus(ctx, id)
	if err == nil && currentStatus != nil {
		j.Status = *currentStatus
		if currentProgress != nil {
			j.Progress = *currentProgress
		}
	}

	return &j, nil
}

// GetByIDWithEvents retrieves a print job with its full event timeline.
func (r *PrintJobRepository) GetByIDWithEvents(ctx context.Context, id uuid.UUID) (*model.PrintJob, error) {
	j, err := r.GetByID(ctx, id)
	if err != nil || j == nil {
		return j, err
	}

	events, err := r.GetEvents(ctx, id)
	if err != nil {
		return nil, err
	}
	j.Events = events

	return j, nil
}

// List retrieves print jobs with optional filters.
func (r *PrintJobRepository) List(ctx context.Context, printerID *uuid.UUID, status *model.PrintJobStatus) ([]model.PrintJob, error) {
	query := `SELECT pj.id, pj.design_id, pj.printer_id, pj.material_spool_id, pj.project_id, pj.task_id, pj.status, pj.progress, pj.started_at, pj.completed_at, pj.outcome, pj.notes, pj.created_at,
	                 pj.recipe_id, pj.attempt_number, pj.parent_job_id, pj.failure_category, pj.estimated_seconds, pj.actual_seconds, pj.material_used_grams, pj.cost_cents, pj.printer_time_cost_cents, pj.material_cost_cents, pj.material_snapshot, pj.priority, pj.auto_dispatch_enabled
	          FROM print_jobs pj WHERE 1=1`
	args := []interface{}{}

	if printerID != nil {
		query += " AND pj.printer_id = ?"
		args = append(args, *printerID)
	}
	if status != nil {
		query += " AND pj.status = ?"
		args = append(args, *status)
	}
	query += ` ORDER BY pj.created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []model.PrintJob
	for rows.Next() {
		var j model.PrintJob
		var outcomeJSON, snapshotJSON []byte
		if err := scanRow(rows, &j.ID, &j.DesignID, &j.PrinterID, &j.MaterialSpoolID, &j.ProjectID, &j.TaskID, &j.Status, &j.Progress, &j.StartedAt, &j.CompletedAt, &outcomeJSON, &j.Notes, &j.CreatedAt,
			&j.RecipeID, &j.AttemptNumber, &j.ParentJobID, &j.FailureCategory, &j.EstimatedSeconds, &j.ActualSeconds, &j.MaterialUsedGrams, &j.CostCents, &j.PrinterTimeCostCents, &j.MaterialCostCents, &snapshotJSON, &j.Priority, &j.AutoDispatchEnabled); err != nil {
			return nil, err
		}
		if outcomeJSON != nil {
			json.Unmarshal(outcomeJSON, &j.Outcome)
		}
		if snapshotJSON != nil {
			json.Unmarshal(snapshotJSON, &j.MaterialSnapshot)
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// ListCompletedSince retrieves completed print jobs that were completed on or after the given time.
func (r *PrintJobRepository) ListCompletedSince(ctx context.Context, since time.Time) ([]model.PrintJob, error) {
	query := `SELECT pj.id, pj.design_id, pj.printer_id, pj.material_spool_id, pj.project_id, pj.task_id, pj.status, pj.progress, pj.started_at, pj.completed_at, pj.outcome, pj.notes, pj.created_at,
	                 pj.recipe_id, pj.attempt_number, pj.parent_job_id, pj.failure_category, pj.estimated_seconds, pj.actual_seconds, pj.material_used_grams, pj.cost_cents, pj.printer_time_cost_cents, pj.material_cost_cents, pj.material_snapshot, pj.priority, pj.auto_dispatch_enabled
	          FROM print_jobs pj WHERE pj.status = ? AND pj.completed_at >= ? ORDER BY pj.created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, model.PrintJobStatusCompleted, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []model.PrintJob
	for rows.Next() {
		var j model.PrintJob
		var outcomeJSON, snapshotJSON []byte
		if err := scanRow(rows, &j.ID, &j.DesignID, &j.PrinterID, &j.MaterialSpoolID, &j.ProjectID, &j.TaskID, &j.Status, &j.Progress, &j.StartedAt, &j.CompletedAt, &outcomeJSON, &j.Notes, &j.CreatedAt,
			&j.RecipeID, &j.AttemptNumber, &j.ParentJobID, &j.FailureCategory, &j.EstimatedSeconds, &j.ActualSeconds, &j.MaterialUsedGrams, &j.CostCents, &j.PrinterTimeCostCents, &j.MaterialCostCents, &snapshotJSON, &j.Priority, &j.AutoDispatchEnabled); err != nil {
			return nil, err
		}
		if outcomeJSON != nil {
			json.Unmarshal(outcomeJSON, &j.Outcome)
		}
		if snapshotJSON != nil {
			json.Unmarshal(snapshotJSON, &j.MaterialSnapshot)
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// ListByDesign retrieves all print jobs for a design.
func (r *PrintJobRepository) ListByDesign(ctx context.Context, designID uuid.UUID) ([]model.PrintJob, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, design_id, printer_id, material_spool_id, project_id, task_id, status, progress, started_at, completed_at, outcome, notes, created_at,
		       recipe_id, attempt_number, parent_job_id, failure_category, estimated_seconds, actual_seconds, material_used_grams, cost_cents, printer_time_cost_cents, material_cost_cents, material_snapshot, priority, auto_dispatch_enabled
		FROM print_jobs WHERE design_id = ? ORDER BY created_at DESC
	`, designID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []model.PrintJob
	for rows.Next() {
		var j model.PrintJob
		var outcomeJSON, snapshotJSON []byte
		if err := scanRow(rows, &j.ID, &j.DesignID, &j.PrinterID, &j.MaterialSpoolID, &j.ProjectID, &j.TaskID, &j.Status, &j.Progress, &j.StartedAt, &j.CompletedAt, &outcomeJSON, &j.Notes, &j.CreatedAt,
			&j.RecipeID, &j.AttemptNumber, &j.ParentJobID, &j.FailureCategory, &j.EstimatedSeconds, &j.ActualSeconds, &j.MaterialUsedGrams, &j.CostCents, &j.PrinterTimeCostCents, &j.MaterialCostCents, &snapshotJSON, &j.Priority, &j.AutoDispatchEnabled); err != nil {
			return nil, err
		}
		if outcomeJSON != nil {
			json.Unmarshal(outcomeJSON, &j.Outcome)
		}
		if snapshotJSON != nil {
			json.Unmarshal(snapshotJSON, &j.MaterialSnapshot)
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// Update updates denormalized fields on a print job.
// NOTE: Status changes should go through AppendEvent, not Update.
// This method is for updating computed/summary fields only.
func (r *PrintJobRepository) Update(ctx context.Context, j *model.PrintJob) error {
	outcomeJSON, _ := json.Marshal(j.Outcome)
	snapshotJSON, _ := json.Marshal(j.MaterialSnapshot)

	_, err := r.db.ExecContext(ctx, `
		UPDATE print_jobs SET printer_id = ?, material_spool_id = ?, status = ?, progress = ?, started_at = ?, completed_at = ?, outcome = ?, notes = ?,
		       failure_category = ?, actual_seconds = ?, material_used_grams = ?, cost_cents = ?, printer_time_cost_cents = ?, material_cost_cents = ?, material_snapshot = ?,
		       priority = ?, auto_dispatch_enabled = ?
		WHERE id = ?
	`, j.PrinterID, j.MaterialSpoolID, j.Status, j.Progress, j.StartedAt, j.CompletedAt, outcomeJSON, j.Notes,
		j.FailureCategory, j.ActualSeconds, j.MaterialUsedGrams, j.CostCents, j.PrinterTimeCostCents, j.MaterialCostCents, snapshotJSON,
		j.Priority, j.AutoDispatchEnabled, j.ID)
	return err
}

// AppendEvent records a new event for a job. Events are immutable once created.
func (r *PrintJobRepository) AppendEvent(ctx context.Context, e *model.JobEvent) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now()
	}
	if e.OccurredAt.IsZero() {
		e.OccurredAt = e.CreatedAt
	}

	metadataJSON, _ := json.Marshal(e.Metadata)

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO job_events (id, job_id, event_type, occurred_at, status, progress, printer_id, error_code, error_message, actor_type, actor_id, metadata, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, e.ID, e.JobID, e.EventType, e.OccurredAt, e.Status, e.Progress, e.PrinterID, e.ErrorCode, e.ErrorMessage, e.ActorType, e.ActorID, metadataJSON, e.CreatedAt)

	if err != nil {
		return err
	}

	// Update denormalized status on print_jobs table for query efficiency
	if e.Status != nil {
		_, err = r.db.ExecContext(ctx, `UPDATE print_jobs SET status = ?, progress = COALESCE(?, progress) WHERE id = ?`, *e.Status, e.Progress, e.JobID)
	} else if e.Progress != nil {
		_, err = r.db.ExecContext(ctx, `UPDATE print_jobs SET progress = ? WHERE id = ?`, *e.Progress, e.JobID)
	}

	return err
}

// GetEvents retrieves all events for a job in chronological order.
func (r *PrintJobRepository) GetEvents(ctx context.Context, jobID uuid.UUID) ([]model.JobEvent, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, job_id, event_type, occurred_at, status, progress, printer_id, error_code, error_message, actor_type, actor_id, metadata, created_at
		FROM job_events WHERE job_id = ? ORDER BY occurred_at ASC
	`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []model.JobEvent
	for rows.Next() {
		var e model.JobEvent
		var metadataJSON []byte
		if err := scanRow(rows, &e.ID, &e.JobID, &e.EventType, &e.OccurredAt, &e.Status, &e.Progress, &e.PrinterID, &e.ErrorCode, &e.ErrorMessage, &e.ActorType, &e.ActorID, &metadataJSON, &e.CreatedAt); err != nil {
			return nil, err
		}
		if metadataJSON != nil {
			json.Unmarshal(metadataJSON, &e.Metadata)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// GetCurrentStatus retrieves the current status and progress for a job from the latest event.
func (r *PrintJobRepository) GetCurrentStatus(ctx context.Context, jobID uuid.UUID) (*model.PrintJobStatus, *float64, error) {
	var status model.PrintJobStatus
	var progress *float64
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT status, progress FROM job_events
		WHERE job_id = ? AND status IS NOT NULL
		ORDER BY occurred_at DESC LIMIT 1
	`, jobID), &status, &progress)
	if err == sql.ErrNoRows {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	return &status, progress, nil
}

// GetRetryChain retrieves all jobs in a retry chain (original + all retries).
func (r *PrintJobRepository) GetRetryChain(ctx context.Context, jobID uuid.UUID) ([]model.PrintJob, error) {
	// First, find the root job (the one with no parent)
	rootID := jobID
	for {
		var parentID *uuid.UUID
		err := scanRow(r.db.QueryRowContext(ctx, `SELECT parent_job_id FROM print_jobs WHERE id = ?`, rootID), &parentID)
		if err != nil {
			return nil, err
		}
		if parentID == nil {
			break
		}
		rootID = *parentID
	}

	// Now get all jobs in the chain
	rows, err := r.db.QueryContext(ctx, `
		WITH RECURSIVE chain AS (
			SELECT id, design_id, printer_id, material_spool_id, project_id, task_id, status, progress, started_at, completed_at, outcome, notes, created_at,
			       recipe_id, attempt_number, parent_job_id, failure_category, estimated_seconds, actual_seconds, material_used_grams, cost_cents, printer_time_cost_cents, material_cost_cents, material_snapshot, priority, auto_dispatch_enabled
			FROM print_jobs WHERE id = ?
			UNION ALL
			SELECT pj.id, pj.design_id, pj.printer_id, pj.material_spool_id, pj.project_id, pj.task_id, pj.status, pj.progress, pj.started_at, pj.completed_at, pj.outcome, pj.notes, pj.created_at,
			       pj.recipe_id, pj.attempt_number, pj.parent_job_id, pj.failure_category, pj.estimated_seconds, pj.actual_seconds, pj.material_used_grams, pj.cost_cents, pj.printer_time_cost_cents, pj.material_cost_cents, pj.material_snapshot, pj.priority, pj.auto_dispatch_enabled
			FROM print_jobs pj INNER JOIN chain c ON pj.parent_job_id = c.id
		)
		SELECT * FROM chain ORDER BY attempt_number ASC
	`, rootID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []model.PrintJob
	for rows.Next() {
		var j model.PrintJob
		var outcomeJSON, snapshotJSON []byte
		if err := scanRow(rows, &j.ID, &j.DesignID, &j.PrinterID, &j.MaterialSpoolID, &j.ProjectID, &j.TaskID, &j.Status, &j.Progress, &j.StartedAt, &j.CompletedAt, &outcomeJSON, &j.Notes, &j.CreatedAt,
			&j.RecipeID, &j.AttemptNumber, &j.ParentJobID, &j.FailureCategory, &j.EstimatedSeconds, &j.ActualSeconds, &j.MaterialUsedGrams, &j.CostCents, &j.PrinterTimeCostCents, &j.MaterialCostCents, &snapshotJSON, &j.Priority, &j.AutoDispatchEnabled); err != nil {
			return nil, err
		}
		if outcomeJSON != nil {
			json.Unmarshal(outcomeJSON, &j.Outcome)
		}
		if snapshotJSON != nil {
			json.Unmarshal(snapshotJSON, &j.MaterialSnapshot)
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// ListByRecipe retrieves all print jobs for a recipe/template.
func (r *PrintJobRepository) ListByRecipe(ctx context.Context, recipeID uuid.UUID) ([]model.PrintJob, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, design_id, printer_id, material_spool_id, project_id, task_id, status, progress, started_at, completed_at, outcome, notes, created_at,
		       recipe_id, attempt_number, parent_job_id, failure_category, estimated_seconds, actual_seconds, material_used_grams, cost_cents, printer_time_cost_cents, material_cost_cents, material_snapshot, priority, auto_dispatch_enabled
		FROM print_jobs WHERE recipe_id = ? ORDER BY created_at DESC
	`, recipeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []model.PrintJob
	for rows.Next() {
		var j model.PrintJob
		var outcomeJSON, snapshotJSON []byte
		if err := scanRow(rows, &j.ID, &j.DesignID, &j.PrinterID, &j.MaterialSpoolID, &j.ProjectID, &j.TaskID, &j.Status, &j.Progress, &j.StartedAt, &j.CompletedAt, &outcomeJSON, &j.Notes, &j.CreatedAt,
			&j.RecipeID, &j.AttemptNumber, &j.ParentJobID, &j.FailureCategory, &j.EstimatedSeconds, &j.ActualSeconds, &j.MaterialUsedGrams, &j.CostCents, &j.PrinterTimeCostCents, &j.MaterialCostCents, &snapshotJSON, &j.Priority, &j.AutoDispatchEnabled); err != nil {
			return nil, err
		}
		if outcomeJSON != nil {
			json.Unmarshal(outcomeJSON, &j.Outcome)
		}
		if snapshotJSON != nil {
			json.Unmarshal(snapshotJSON, &j.MaterialSnapshot)
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// ListByProject retrieves all print jobs for a project.
func (r *PrintJobRepository) ListByProject(ctx context.Context, projectID uuid.UUID) ([]model.PrintJob, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, design_id, printer_id, material_spool_id, project_id, task_id, status, progress, started_at, completed_at, outcome, notes, created_at,
		       recipe_id, attempt_number, parent_job_id, failure_category, estimated_seconds, actual_seconds, material_used_grams, cost_cents, printer_time_cost_cents, material_cost_cents, material_snapshot, priority, auto_dispatch_enabled
		FROM print_jobs WHERE project_id = ? ORDER BY created_at DESC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []model.PrintJob
	for rows.Next() {
		var j model.PrintJob
		var outcomeJSON, snapshotJSON []byte
		if err := scanRow(rows, &j.ID, &j.DesignID, &j.PrinterID, &j.MaterialSpoolID, &j.ProjectID, &j.TaskID, &j.Status, &j.Progress, &j.StartedAt, &j.CompletedAt, &outcomeJSON, &j.Notes, &j.CreatedAt,
			&j.RecipeID, &j.AttemptNumber, &j.ParentJobID, &j.FailureCategory, &j.EstimatedSeconds, &j.ActualSeconds, &j.MaterialUsedGrams, &j.CostCents, &j.PrinterTimeCostCents, &j.MaterialCostCents, &snapshotJSON, &j.Priority, &j.AutoDispatchEnabled); err != nil {
			return nil, err
		}
		if outcomeJSON != nil {
			json.Unmarshal(outcomeJSON, &j.Outcome)
		}
		if snapshotJSON != nil {
			json.Unmarshal(snapshotJSON, &j.MaterialSnapshot)
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// ListByTask retrieves all print jobs for a task.
func (r *PrintJobRepository) ListByTask(ctx context.Context, taskID uuid.UUID) ([]model.PrintJob, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, design_id, printer_id, material_spool_id, project_id, task_id, status, progress, started_at, completed_at, outcome, notes, created_at,
		       recipe_id, attempt_number, parent_job_id, failure_category, estimated_seconds, actual_seconds, material_used_grams, cost_cents, printer_time_cost_cents, material_cost_cents, material_snapshot, priority, auto_dispatch_enabled
		FROM print_jobs WHERE task_id = ? ORDER BY created_at DESC
	`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []model.PrintJob
	for rows.Next() {
		var j model.PrintJob
		var outcomeJSON, snapshotJSON []byte
		if err := scanRow(rows, &j.ID, &j.DesignID, &j.PrinterID, &j.MaterialSpoolID, &j.ProjectID, &j.TaskID, &j.Status, &j.Progress, &j.StartedAt, &j.CompletedAt, &outcomeJSON, &j.Notes, &j.CreatedAt,
			&j.RecipeID, &j.AttemptNumber, &j.ParentJobID, &j.FailureCategory, &j.EstimatedSeconds, &j.ActualSeconds, &j.MaterialUsedGrams, &j.CostCents, &j.PrinterTimeCostCents, &j.MaterialCostCents, &snapshotJSON, &j.Priority, &j.AutoDispatchEnabled); err != nil {
			return nil, err
		}
		if outcomeJSON != nil {
			json.Unmarshal(outcomeJSON, &j.Outcome)
		}
		if snapshotJSON != nil {
			json.Unmarshal(snapshotJSON, &j.MaterialSnapshot)
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// JobStats contains job statistics for a project.
type JobStats struct {
	Total     int `json:"total"`
	Queued    int `json:"queued"`
	Assigned  int `json:"assigned"`
	Printing  int `json:"printing"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
	Cancelled int `json:"cancelled"`
}

// GetProjectJobStats retrieves job statistics for a project.
func (r *PrintJobRepository) GetProjectJobStats(ctx context.Context, projectID uuid.UUID) (*JobStats, error) {
	stats := &JobStats{}
	rows, err := r.db.QueryContext(ctx, `
		SELECT status, COUNT(*) FROM print_jobs WHERE project_id = ? GROUP BY status
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var status model.PrintJobStatus
		var count int
		if err := scanRow(rows, &status, &count); err != nil {
			return nil, err
		}
		stats.Total += count
		switch status {
		case model.PrintJobStatusQueued:
			stats.Queued = count
		case model.PrintJobStatusAssigned:
			stats.Assigned = count
		case model.PrintJobStatusPrinting, model.PrintJobStatusUploaded:
			stats.Printing += count
		case model.PrintJobStatusCompleted:
			stats.Completed = count
		case model.PrintJobStatusFailed:
			stats.Failed = count
		case model.PrintJobStatusCancelled:
			stats.Cancelled = count
		}
	}
	return stats, rows.Err()
}

// GetPrinterJobStats retrieves job statistics for a printer.
func (r *PrintJobRepository) GetPrinterJobStats(ctx context.Context, printerID uuid.UUID) (*JobStats, error) {
	stats := &JobStats{}
	rows, err := r.db.QueryContext(ctx, `
		SELECT status, COUNT(*) FROM print_jobs WHERE printer_id = ? GROUP BY status
	`, printerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var status model.PrintJobStatus
		var count int
		if err := scanRow(rows, &status, &count); err != nil {
			return nil, err
		}
		stats.Total += count
		switch status {
		case model.PrintJobStatusQueued:
			stats.Queued = count
		case model.PrintJobStatusAssigned:
			stats.Assigned = count
		case model.PrintJobStatusPrinting, model.PrintJobStatusUploaded:
			stats.Printing += count
		case model.PrintJobStatusCompleted:
			stats.Completed = count
		case model.PrintJobStatusFailed:
			stats.Failed = count
		case model.PrintJobStatusCancelled:
			stats.Cancelled = count
		}
	}
	return stats, rows.Err()
}

// ListQueued retrieves queued jobs ordered by priority DESC, created_at ASC.
// Only returns jobs with auto_dispatch_enabled = true.
func (r *PrintJobRepository) ListQueued(ctx context.Context) ([]model.PrintJob, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, design_id, printer_id, material_spool_id, project_id, status, progress, started_at, completed_at, outcome, notes, created_at,
		       recipe_id, attempt_number, parent_job_id, failure_category, estimated_seconds, actual_seconds, material_used_grams, cost_cents, printer_time_cost_cents, material_cost_cents, material_snapshot, priority, auto_dispatch_enabled
		FROM print_jobs
		WHERE status = 'queued' AND auto_dispatch_enabled = 1
		ORDER BY priority DESC, created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []model.PrintJob
	for rows.Next() {
		var j model.PrintJob
		var outcomeJSON, snapshotJSON []byte
		if err := scanRow(rows, &j.ID, &j.DesignID, &j.PrinterID, &j.MaterialSpoolID, &j.ProjectID, &j.Status, &j.Progress, &j.StartedAt, &j.CompletedAt, &outcomeJSON, &j.Notes, &j.CreatedAt,
			&j.RecipeID, &j.AttemptNumber, &j.ParentJobID, &j.FailureCategory, &j.EstimatedSeconds, &j.ActualSeconds, &j.MaterialUsedGrams, &j.CostCents, &j.PrinterTimeCostCents, &j.MaterialCostCents, &snapshotJSON, &j.Priority, &j.AutoDispatchEnabled); err != nil {
			return nil, err
		}
		if outcomeJSON != nil {
			json.Unmarshal(outcomeJSON, &j.Outcome)
		}
		if snapshotJSON != nil {
			json.Unmarshal(snapshotJSON, &j.MaterialSnapshot)
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// UpdatePriority updates a job's priority.
func (r *PrintJobRepository) UpdatePriority(ctx context.Context, id uuid.UUID, priority int) error {
	_, err := r.db.ExecContext(ctx, `UPDATE print_jobs SET priority = ? WHERE id = ?`, priority, id)
	return err
}

// FileRepository handles file metadata database operations.
type FileRepository struct {
	db *sql.DB
}

// Create inserts a new file record.
func (r *FileRepository) Create(ctx context.Context, f *model.File) error {
	f.ID = uuid.New()
	f.CreatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO files (id, hash, original_name, content_type, size_bytes, storage_path, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, f.ID, f.Hash, f.OriginalName, f.ContentType, f.SizeBytes, f.StoragePath, f.CreatedAt)
	return err
}

// GetByID retrieves a file by ID.
func (r *FileRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.File, error) {
	var f model.File
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, hash, original_name, content_type, size_bytes, storage_path, created_at
		FROM files WHERE id = ?
	`, id), &f.ID, &f.Hash, &f.OriginalName, &f.ContentType, &f.SizeBytes, &f.StoragePath, &f.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &f, err
}

// GetByHash retrieves a file by hash (for deduplication).
func (r *FileRepository) GetByHash(ctx context.Context, hash string) (*model.File, error) {
	var f model.File
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, hash, original_name, content_type, size_bytes, storage_path, created_at
		FROM files WHERE hash = ?
	`, hash), &f.ID, &f.Hash, &f.OriginalName, &f.ContentType, &f.SizeBytes, &f.StoragePath, &f.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &f, err
}

// ExpenseRepository handles expense database operations.
type ExpenseRepository struct {
	db *sql.DB
}

// Create inserts a new expense.
func (r *ExpenseRepository) Create(ctx context.Context, e *model.Expense) error {
	e.ID = uuid.New()
	e.CreatedAt = time.Now()
	e.UpdatedAt = time.Now()
	if e.Status == "" {
		e.Status = model.ExpenseStatusPending
	}
	if e.Currency == "" {
		e.Currency = "USD"
	}
	if e.Category == "" {
		e.Category = model.ExpenseCategoryOther
	}

	rawAIJSON, _ := json.Marshal(e.RawAIResponse)

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO expenses (id, occurred_at, vendor, subtotal_cents, tax_cents, shipping_cents, total_cents, currency, category, notes, receipt_file_id, receipt_file_path, status, raw_ocr_text, raw_ai_response, confidence, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, e.ID, e.OccurredAt, e.Vendor, e.SubtotalCents, e.TaxCents, e.ShippingCents, e.TotalCents, e.Currency, e.Category, e.Notes, e.ReceiptFileID, e.ReceiptFilePath, e.Status, e.RawOCRText, rawAIJSON, e.Confidence, e.CreatedAt, e.UpdatedAt)
	return err
}

// GetByID retrieves an expense by ID.
func (r *ExpenseRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Expense, error) {
	var e model.Expense
	var rawAIJSON []byte
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, occurred_at, vendor, subtotal_cents, tax_cents, shipping_cents, total_cents, currency, category, notes, receipt_file_id, receipt_file_path, status, raw_ocr_text, raw_ai_response, confidence, created_at, updated_at
		FROM expenses WHERE id = ?
	`, id), &e.ID, &e.OccurredAt, &e.Vendor, &e.SubtotalCents, &e.TaxCents, &e.ShippingCents, &e.TotalCents, &e.Currency, &e.Category, &e.Notes, &e.ReceiptFileID, &e.ReceiptFilePath, &e.Status, &e.RawOCRText, &rawAIJSON, &e.Confidence, &e.CreatedAt, &e.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if rawAIJSON != nil {
		e.RawAIResponse = rawAIJSON
	}
	return &e, err
}

// List retrieves all expenses.
func (r *ExpenseRepository) List(ctx context.Context, status *model.ExpenseStatus) ([]model.Expense, error) {
	query := `SELECT id, occurred_at, vendor, subtotal_cents, tax_cents, shipping_cents, total_cents, currency, category, notes, receipt_file_id, receipt_file_path, status, confidence, created_at, updated_at FROM expenses`
	args := []interface{}{}

	if status != nil {
		query += ` WHERE status = ?`
		args = append(args, *status)
	}
	query += ` ORDER BY occurred_at DESC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expenses []model.Expense
	for rows.Next() {
		var e model.Expense
		if err := scanRow(rows, &e.ID, &e.OccurredAt, &e.Vendor, &e.SubtotalCents, &e.TaxCents, &e.ShippingCents, &e.TotalCents, &e.Currency, &e.Category, &e.Notes, &e.ReceiptFileID, &e.ReceiptFilePath, &e.Status, &e.Confidence, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		expenses = append(expenses, e)
	}
	return expenses, rows.Err()
}

// ListSince retrieves expenses with the given status that occurred on or after the given time.
func (r *ExpenseRepository) ListSince(ctx context.Context, status *model.ExpenseStatus, since time.Time) ([]model.Expense, error) {
	query := `SELECT id, occurred_at, vendor, subtotal_cents, tax_cents, shipping_cents, total_cents, currency, category, notes, receipt_file_id, receipt_file_path, status, confidence, created_at, updated_at FROM expenses WHERE occurred_at >= ?`
	args := []interface{}{since}

	if status != nil {
		query += ` AND status = ?`
		args = append(args, *status)
	}
	query += ` ORDER BY occurred_at DESC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expenses []model.Expense
	for rows.Next() {
		var e model.Expense
		if err := scanRow(rows, &e.ID, &e.OccurredAt, &e.Vendor, &e.SubtotalCents, &e.TaxCents, &e.ShippingCents, &e.TotalCents, &e.Currency, &e.Category, &e.Notes, &e.ReceiptFileID, &e.ReceiptFilePath, &e.Status, &e.Confidence, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		expenses = append(expenses, e)
	}
	return expenses, rows.Err()
}

// Update updates an expense.
func (r *ExpenseRepository) Update(ctx context.Context, e *model.Expense) error {
	return r.UpdateTx(ctx, r.db, e)
}

// UpdateTx updates an expense using the provided DBTX (supports transactions).
func (r *ExpenseRepository) UpdateTx(ctx context.Context, db DBTX, e *model.Expense) error {
	e.UpdatedAt = time.Now()
	rawAIJSON, _ := json.Marshal(e.RawAIResponse)

	_, err := db.ExecContext(ctx, `
		UPDATE expenses SET occurred_at = ?, vendor = ?, subtotal_cents = ?, tax_cents = ?, shipping_cents = ?, total_cents = ?, currency = ?, category = ?, notes = ?, status = ?, raw_ocr_text = ?, raw_ai_response = ?, confidence = ?, updated_at = ?
		WHERE id = ?
	`, e.OccurredAt, e.Vendor, e.SubtotalCents, e.TaxCents, e.ShippingCents, e.TotalCents, e.Currency, e.Category, e.Notes, e.Status, e.RawOCRText, rawAIJSON, e.Confidence, e.UpdatedAt, e.ID)
	return err
}

// Delete deletes an expense.
func (r *ExpenseRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM expenses WHERE id = ?`, id)
	return err
}

// ExpenseItemRepository handles expense item database operations (embedded in ExpenseRepository).

// CreateItem inserts a new expense item.
func (r *ExpenseRepository) CreateItem(ctx context.Context, item *model.ExpenseItem) error {
	item.ID = uuid.New()
	item.CreatedAt = time.Now()
	if item.ActionTaken == "" {
		item.ActionTaken = model.ExpenseItemActionNone
	}

	metadataJSON, _ := json.Marshal(item.Metadata)

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO expense_items (id, expense_id, description, quantity, unit_price_cents, total_price_cents, sku, vendor_item_id, category, metadata, matched_spool_id, matched_material_id, confidence, action_taken, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, item.ID, item.ExpenseID, item.Description, item.Quantity, item.UnitPriceCents, item.TotalPriceCents, item.SKU, item.VendorItemID, item.Category, metadataJSON, item.MatchedSpoolID, item.MatchedMaterialID, item.Confidence, item.ActionTaken, item.CreatedAt)
	return err
}

// GetItemsByExpenseID retrieves all items for an expense.
func (r *ExpenseRepository) GetItemsByExpenseID(ctx context.Context, expenseID uuid.UUID) ([]model.ExpenseItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, expense_id, description, quantity, unit_price_cents, total_price_cents, sku, vendor_item_id, category, metadata, matched_spool_id, matched_material_id, confidence, action_taken, created_at
		FROM expense_items WHERE expense_id = ? ORDER BY created_at
	`, expenseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.ExpenseItem
	for rows.Next() {
		var item model.ExpenseItem
		var metadataJSON []byte
		if err := scanRow(rows, &item.ID, &item.ExpenseID, &item.Description, &item.Quantity, &item.UnitPriceCents, &item.TotalPriceCents, &item.SKU, &item.VendorItemID, &item.Category, &metadataJSON, &item.MatchedSpoolID, &item.MatchedMaterialID, &item.Confidence, &item.ActionTaken, &item.CreatedAt); err != nil {
			return nil, err
		}
		if metadataJSON != nil {
			json.Unmarshal(metadataJSON, &item.Metadata)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// UpdateItem updates an expense item.
func (r *ExpenseRepository) UpdateItem(ctx context.Context, item *model.ExpenseItem) error {
	return r.UpdateItemTx(ctx, r.db, item)
}

// UpdateItemTx updates an expense item using the provided DBTX (supports transactions).
func (r *ExpenseRepository) UpdateItemTx(ctx context.Context, db DBTX, item *model.ExpenseItem) error {
	metadataJSON, _ := json.Marshal(item.Metadata)

	_, err := db.ExecContext(ctx, `
		UPDATE expense_items SET description = ?, quantity = ?, unit_price_cents = ?, total_price_cents = ?, sku = ?, vendor_item_id = ?, category = ?, metadata = ?, matched_spool_id = ?, matched_material_id = ?, confidence = ?, action_taken = ?
		WHERE id = ?
	`, item.Description, item.Quantity, item.UnitPriceCents, item.TotalPriceCents, item.SKU, item.VendorItemID, item.Category, metadataJSON, item.MatchedSpoolID, item.MatchedMaterialID, item.Confidence, item.ActionTaken, item.ID)
	return err
}

// DeleteItem deletes an expense item by ID.
func (r *ExpenseRepository) DeleteItem(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM expense_items WHERE id = ?`, id)
	return err
}

// DeleteItemsByExpenseID deletes all expense items for an expense.
func (r *ExpenseRepository) DeleteItemsByExpenseID(ctx context.Context, expenseID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM expense_items WHERE expense_id = ?`, expenseID)
	return err
}

// SaleRepository handles sale database operations.
type SaleRepository struct {
	db *sql.DB
}

// Create inserts a new sale.
func (r *SaleRepository) Create(ctx context.Context, s *model.Sale) error {
	s.ID = uuid.New()
	s.CreatedAt = time.Now()
	s.UpdatedAt = time.Now()
	if s.Currency == "" {
		s.Currency = "USD"
	}
	if s.Channel == "" {
		s.Channel = model.SalesChannelOther
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO sales (id, occurred_at, channel, platform, gross_cents, fees_cents, shipping_charged_cents, shipping_cost_cents, tax_collected_cents, net_cents, currency, project_id, order_reference, customer_name, item_description, quantity, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, s.ID, s.OccurredAt, s.Channel, s.Platform, s.GrossCents, s.FeesCents, s.ShippingChargedCents, s.ShippingCostCents, s.TaxCollectedCents, s.NetCents, s.Currency, s.ProjectID, s.OrderReference, s.CustomerName, s.ItemDescription, s.Quantity, s.Notes, s.CreatedAt, s.UpdatedAt)
	return err
}

// GetByID retrieves a sale by ID.
func (r *SaleRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Sale, error) {
	var s model.Sale
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, occurred_at, channel, platform, gross_cents, fees_cents, shipping_charged_cents, shipping_cost_cents, tax_collected_cents, net_cents, currency, project_id, order_reference, customer_name, item_description, quantity, notes, created_at, updated_at
		FROM sales WHERE id = ?
	`, id), &s.ID, &s.OccurredAt, &s.Channel, &s.Platform, &s.GrossCents, &s.FeesCents, &s.ShippingChargedCents, &s.ShippingCostCents, &s.TaxCollectedCents, &s.NetCents, &s.Currency, &s.ProjectID, &s.OrderReference, &s.CustomerName, &s.ItemDescription, &s.Quantity, &s.Notes, &s.CreatedAt, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &s, err
}

// List retrieves all sales.
func (r *SaleRepository) List(ctx context.Context, projectID *uuid.UUID) ([]model.Sale, error) {
	query := `SELECT id, occurred_at, channel, platform, gross_cents, fees_cents, shipping_charged_cents, shipping_cost_cents, tax_collected_cents, net_cents, currency, project_id, order_reference, customer_name, item_description, quantity, notes, created_at, updated_at FROM sales`
	args := []interface{}{}

	if projectID != nil {
		query += ` WHERE project_id = ?`
		args = append(args, *projectID)
	}
	query += ` ORDER BY occurred_at DESC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sales []model.Sale
	for rows.Next() {
		var s model.Sale
		if err := scanRow(rows, &s.ID, &s.OccurredAt, &s.Channel, &s.Platform, &s.GrossCents, &s.FeesCents, &s.ShippingChargedCents, &s.ShippingCostCents, &s.TaxCollectedCents, &s.NetCents, &s.Currency, &s.ProjectID, &s.OrderReference, &s.CustomerName, &s.ItemDescription, &s.Quantity, &s.Notes, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		sales = append(sales, s)
	}
	return sales, rows.Err()
}

// ListSince retrieves sales that occurred on or after the given time.
func (r *SaleRepository) ListSince(ctx context.Context, since time.Time) ([]model.Sale, error) {
	query := `SELECT id, occurred_at, channel, platform, gross_cents, fees_cents, shipping_charged_cents, shipping_cost_cents, tax_collected_cents, net_cents, currency, project_id, order_reference, customer_name, item_description, quantity, notes, created_at, updated_at FROM sales WHERE occurred_at >= ? ORDER BY occurred_at DESC`

	rows, err := r.db.QueryContext(ctx, query, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sales []model.Sale
	for rows.Next() {
		var s model.Sale
		if err := scanRow(rows, &s.ID, &s.OccurredAt, &s.Channel, &s.Platform, &s.GrossCents, &s.FeesCents, &s.ShippingChargedCents, &s.ShippingCostCents, &s.TaxCollectedCents, &s.NetCents, &s.Currency, &s.ProjectID, &s.OrderReference, &s.CustomerName, &s.ItemDescription, &s.Quantity, &s.Notes, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		sales = append(sales, s)
	}
	return sales, rows.Err()
}

// Update updates a sale.
func (r *SaleRepository) Update(ctx context.Context, s *model.Sale) error {
	s.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE sales SET occurred_at = ?, channel = ?, platform = ?, gross_cents = ?, fees_cents = ?, shipping_charged_cents = ?, shipping_cost_cents = ?, tax_collected_cents = ?, net_cents = ?, currency = ?, project_id = ?, order_reference = ?, customer_name = ?, item_description = ?, quantity = ?, notes = ?, updated_at = ?
		WHERE id = ?
	`, s.OccurredAt, s.Channel, s.Platform, s.GrossCents, s.FeesCents, s.ShippingChargedCents, s.ShippingCostCents, s.TaxCollectedCents, s.NetCents, s.Currency, s.ProjectID, s.OrderReference, s.CustomerName, s.ItemDescription, s.Quantity, s.Notes, s.UpdatedAt, s.ID)
	return err
}

// Delete deletes a sale.
func (r *SaleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sales WHERE id = ?`, id)
	return err
}

// GetTotalsByDateRange calculates totals for a date range.
func (r *SaleRepository) GetTotalsByDateRange(ctx context.Context, start, end time.Time) (grossCents, netCents, feesCents int, count int, err error) {
	err = scanRow(r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(gross_cents), 0), COALESCE(SUM(net_cents), 0), COALESCE(SUM(fees_cents), 0), COUNT(*)
		FROM sales WHERE occurred_at >= ? AND occurred_at < ?
	`, start, end), &grossCents, &netCents, &feesCents, &count)
	return
}

// ProjectSalesRow represents aggregated sales data for a project.
type ProjectSalesRow struct {
	ProjectID   string
	ProjectName string
	GrossCents  int
	NetCents    int
	Count       int
	AvgCents    int
	FirstSale   string
	LastSale    string
}

// GetSalesByProject returns sales aggregated by project.
func (r *SaleRepository) GetSalesByProject(ctx context.Context) ([]ProjectSalesRow, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			COALESCE(s.project_id, ''),
			COALESCE(p.name, s.item_description),
			COALESCE(SUM(s.gross_cents), 0),
			COALESCE(SUM(s.net_cents), 0),
			COUNT(*),
			CAST(COALESCE(AVG(s.gross_cents), 0) AS INTEGER),
			MIN(s.occurred_at),
			MAX(s.occurred_at)
		FROM sales s
		LEFT JOIN projects p ON s.project_id = p.id
		GROUP BY COALESCE(s.project_id, s.item_description)
		ORDER BY SUM(s.gross_cents) DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ProjectSalesRow
	for rows.Next() {
		var row ProjectSalesRow
		if err := rows.Scan(&row.ProjectID, &row.ProjectName, &row.GrossCents, &row.NetCents, &row.Count, &row.AvgCents, &row.FirstSale, &row.LastSale); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// TimeSeriesRow represents a single data point in a time series.
type TimeSeriesRow struct {
	DateBucket string
	Total      int
}

// ChannelBreakdownRow represents an aggregation by sales channel.
type ChannelBreakdownRow struct {
	Channel string
	Total   int
	Count   int
}

// GetSalesOverTime returns gross_cents grouped by date bucket.
func (r *SaleRepository) GetSalesOverTime(ctx context.Context, since time.Time, strftimeFmt string) ([]TimeSeriesRow, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT strftime(?, occurred_at) AS bucket, COALESCE(SUM(gross_cents), 0)
		FROM sales WHERE occurred_at >= ?
		GROUP BY bucket ORDER BY bucket
	`, strftimeFmt, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []TimeSeriesRow
	for rows.Next() {
		var row TimeSeriesRow
		if err := rows.Scan(&row.DateBucket, &row.Total); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// GetNetOverTime returns net_cents grouped by date bucket.
func (r *SaleRepository) GetNetOverTime(ctx context.Context, since time.Time, strftimeFmt string) ([]TimeSeriesRow, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT strftime(?, occurred_at) AS bucket, COALESCE(SUM(net_cents), 0)
		FROM sales WHERE occurred_at >= ?
		GROUP BY bucket ORDER BY bucket
	`, strftimeFmt, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []TimeSeriesRow
	for rows.Next() {
		var row TimeSeriesRow
		if err := rows.Scan(&row.DateBucket, &row.Total); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// GetSalesByChannel returns gross_cents and count grouped by channel.
func (r *SaleRepository) GetSalesByChannel(ctx context.Context, since time.Time) ([]ChannelBreakdownRow, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT channel, COALESCE(SUM(gross_cents), 0), COUNT(*)
		FROM sales WHERE occurred_at >= ?
		GROUP BY channel ORDER BY SUM(gross_cents) DESC
	`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ChannelBreakdownRow
	for rows.Next() {
		var row ChannelBreakdownRow
		if err := rows.Scan(&row.Channel, &row.Total, &row.Count); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// CategoryBreakdownRow represents an aggregation by expense category.
type CategoryBreakdownRow struct {
	Category string
	Total    int
	Count    int
}

// GetExpensesOverTime returns total_cents for confirmed expenses grouped by date bucket.
func (r *ExpenseRepository) GetExpensesOverTime(ctx context.Context, since time.Time, strftimeFmt string) ([]TimeSeriesRow, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT strftime(?, occurred_at) AS bucket, COALESCE(SUM(total_cents), 0)
		FROM expenses WHERE status = 'confirmed' AND occurred_at >= ?
		GROUP BY bucket ORDER BY bucket
	`, strftimeFmt, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []TimeSeriesRow
	for rows.Next() {
		var row TimeSeriesRow
		if err := rows.Scan(&row.DateBucket, &row.Total); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// GetExpensesByCategory returns total_cents and count for confirmed expenses grouped by category.
func (r *ExpenseRepository) GetExpensesByCategory(ctx context.Context, since time.Time) ([]CategoryBreakdownRow, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT category, COALESCE(SUM(total_cents), 0), COUNT(*)
		FROM expenses WHERE status = 'confirmed' AND occurred_at >= ?
		GROUP BY category ORDER BY SUM(total_cents) DESC
	`, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []CategoryBreakdownRow
	for rows.Next() {
		var row CategoryBreakdownRow
		if err := rows.Scan(&row.Category, &row.Total, &row.Count); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// TemplateRepository handles template database operations.
type TemplateRepository struct {
	db *sql.DB
}

// Create inserts a new template.
func (r *TemplateRepository) Create(ctx context.Context, t *model.Template) error {
	t.ID = uuid.New()
	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()
	if t.Tags == nil {
		t.Tags = []string{}
	}
	if t.PostProcessChecklist == nil {
		t.PostProcessChecklist = []string{}
	}
	if t.QuantityPerOrder == 0 {
		t.QuantityPerOrder = 1
	}
	if t.Version == 0 {
		t.Version = 1
	}
	if t.PrintProfile == "" {
		t.PrintProfile = model.PrintProfileStandard
	}

	tagsJSON := marshalStringArray(t.Tags)
	checklistJSON, _ := json.Marshal(t.PostProcessChecklist)
	constraintsJSON, _ := json.Marshal(t.PrinterConstraints)

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO templates (id, name, description, sku, tags, material_type, estimated_material_grams, preferred_printer_id, allow_any_printer, quantity_per_order, post_process_checklist, is_active, printer_constraints, print_profile, estimated_print_seconds, labor_minutes, sale_price_cents, material_cost_per_gram_cents, version, archived_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, t.ID, t.Name, t.Description, t.SKU, tagsJSON, t.MaterialType, t.EstimatedMaterialGrams, t.PreferredPrinterID, t.AllowAnyPrinter, t.QuantityPerOrder, checklistJSON, t.IsActive, constraintsJSON, t.PrintProfile, t.EstimatedPrintSeconds, t.LaborMinutes, t.SalePriceCents, t.MaterialCostPerGramCents, t.Version, t.ArchivedAt, t.CreatedAt, t.UpdatedAt)
	return err
}

// GetByID retrieves a template by ID.
func (r *TemplateRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Template, error) {
	var t model.Template
	var tagsJSON, checklistJSON, constraintsJSON []byte
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, name, description, sku, tags, material_type, estimated_material_grams, preferred_printer_id, allow_any_printer, quantity_per_order, post_process_checklist, is_active, COALESCE(printer_constraints, '{}'), COALESCE(print_profile, 'standard'), COALESCE(estimated_print_seconds, 0), COALESCE(labor_minutes, 0), COALESCE(sale_price_cents, 0), COALESCE(material_cost_per_gram_cents, 0), COALESCE(version, 1), archived_at, created_at, updated_at
		FROM templates WHERE id = ?
	`, id), &t.ID, &t.Name, &t.Description, &t.SKU, &tagsJSON, &t.MaterialType, &t.EstimatedMaterialGrams, &t.PreferredPrinterID, &t.AllowAnyPrinter, &t.QuantityPerOrder, &checklistJSON, &t.IsActive, &constraintsJSON, &t.PrintProfile, &t.EstimatedPrintSeconds, &t.LaborMinutes, &t.SalePriceCents, &t.MaterialCostPerGramCents, &t.Version, &t.ArchivedAt, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t.Tags = unmarshalStringArray(tagsJSON)
	if checklistJSON != nil {
		json.Unmarshal(checklistJSON, &t.PostProcessChecklist)
	}
	if constraintsJSON != nil && len(constraintsJSON) > 2 {
		var constraints model.PrinterConstraints
		if err := json.Unmarshal(constraintsJSON, &constraints); err == nil {
			t.PrinterConstraints = &constraints
		}
	}
	return &t, nil
}

// GetBySKU retrieves a template by SKU.
func (r *TemplateRepository) GetBySKU(ctx context.Context, sku string) (*model.Template, error) {
	var t model.Template
	var tagsJSON, checklistJSON, constraintsJSON []byte
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, name, description, sku, tags, material_type, estimated_material_grams, preferred_printer_id, allow_any_printer, quantity_per_order, post_process_checklist, is_active, COALESCE(printer_constraints, '{}'), COALESCE(print_profile, 'standard'), COALESCE(estimated_print_seconds, 0), COALESCE(labor_minutes, 0), COALESCE(sale_price_cents, 0), COALESCE(material_cost_per_gram_cents, 0), COALESCE(version, 1), archived_at, created_at, updated_at
		FROM templates WHERE sku = ?
	`, sku), &t.ID, &t.Name, &t.Description, &t.SKU, &tagsJSON, &t.MaterialType, &t.EstimatedMaterialGrams, &t.PreferredPrinterID, &t.AllowAnyPrinter, &t.QuantityPerOrder, &checklistJSON, &t.IsActive, &constraintsJSON, &t.PrintProfile, &t.EstimatedPrintSeconds, &t.LaborMinutes, &t.SalePriceCents, &t.MaterialCostPerGramCents, &t.Version, &t.ArchivedAt, &t.CreatedAt, &t.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	t.Tags = unmarshalStringArray(tagsJSON)
	if checklistJSON != nil {
		json.Unmarshal(checklistJSON, &t.PostProcessChecklist)
	}
	if constraintsJSON != nil && len(constraintsJSON) > 2 {
		var constraints model.PrinterConstraints
		if err := json.Unmarshal(constraintsJSON, &constraints); err == nil {
			t.PrinterConstraints = &constraints
		}
	}
	return &t, nil
}

// List retrieves all templates with optional active filter.
func (r *TemplateRepository) List(ctx context.Context, activeOnly bool) ([]model.Template, error) {
	query := `SELECT id, name, description, sku, tags, material_type, estimated_material_grams, preferred_printer_id, allow_any_printer, quantity_per_order, post_process_checklist, is_active, COALESCE(printer_constraints, '{}'), COALESCE(print_profile, 'standard'), COALESCE(estimated_print_seconds, 0), COALESCE(labor_minutes, 0), COALESCE(sale_price_cents, 0), COALESCE(material_cost_per_gram_cents, 0), COALESCE(version, 1), archived_at, created_at, updated_at FROM templates`
	if activeOnly {
		query += ` WHERE is_active = 1 AND archived_at IS NULL`
	}
	query += ` ORDER BY name ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []model.Template
	for rows.Next() {
		var t model.Template
		var tagsJSON, checklistJSON, constraintsJSON []byte
		if err := scanRow(rows, &t.ID, &t.Name, &t.Description, &t.SKU, &tagsJSON, &t.MaterialType, &t.EstimatedMaterialGrams, &t.PreferredPrinterID, &t.AllowAnyPrinter, &t.QuantityPerOrder, &checklistJSON, &t.IsActive, &constraintsJSON, &t.PrintProfile, &t.EstimatedPrintSeconds, &t.LaborMinutes, &t.SalePriceCents, &t.MaterialCostPerGramCents, &t.Version, &t.ArchivedAt, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		t.Tags = unmarshalStringArray(tagsJSON)
		if checklistJSON != nil {
			json.Unmarshal(checklistJSON, &t.PostProcessChecklist)
		}
		if constraintsJSON != nil && len(constraintsJSON) > 2 {
			var constraints model.PrinterConstraints
			if err := json.Unmarshal(constraintsJSON, &constraints); err == nil {
				t.PrinterConstraints = &constraints
			}
		}
		templates = append(templates, t)
	}
	return templates, rows.Err()
}

// Update updates a template.
func (r *TemplateRepository) Update(ctx context.Context, t *model.Template) error {
	t.UpdatedAt = time.Now()
	tagsJSON := marshalStringArray(t.Tags)
	checklistJSON, _ := json.Marshal(t.PostProcessChecklist)
	constraintsJSON, _ := json.Marshal(t.PrinterConstraints)

	_, err := r.db.ExecContext(ctx, `
		UPDATE templates SET name = ?, description = ?, sku = ?, tags = ?, material_type = ?, estimated_material_grams = ?, preferred_printer_id = ?, allow_any_printer = ?, quantity_per_order = ?, post_process_checklist = ?, is_active = ?, printer_constraints = ?, print_profile = ?, estimated_print_seconds = ?, labor_minutes = ?, sale_price_cents = ?, material_cost_per_gram_cents = ?, version = ?, archived_at = ?, updated_at = ?
		WHERE id = ?
	`, t.Name, t.Description, t.SKU, tagsJSON, t.MaterialType, t.EstimatedMaterialGrams, t.PreferredPrinterID, t.AllowAnyPrinter, t.QuantityPerOrder, checklistJSON, t.IsActive, constraintsJSON, t.PrintProfile, t.EstimatedPrintSeconds, t.LaborMinutes, t.SalePriceCents, t.MaterialCostPerGramCents, t.Version, t.ArchivedAt, t.UpdatedAt, t.ID)
	return err
}

// Delete removes a template.
func (r *TemplateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM templates WHERE id = ?`, id)
	return err
}

// AddDesign adds a design to a template.
func (r *TemplateRepository) AddDesign(ctx context.Context, td *model.TemplateDesign) error {
	td.ID = uuid.New()
	td.CreatedAt = time.Now()
	if td.Quantity == 0 {
		td.Quantity = 1
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO template_designs (id, template_id, design_id, is_primary, quantity, sequence_order, notes, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, td.ID, td.TemplateID, td.DesignID, td.IsPrimary, td.Quantity, td.SequenceOrder, td.Notes, td.CreatedAt)
	return err
}

// RemoveDesign removes a design from a template.
func (r *TemplateRepository) RemoveDesign(ctx context.Context, templateID, designID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM template_designs WHERE template_id = ? AND design_id = ?`, templateID, designID)
	return err
}

// GetDesigns retrieves all designs for a template.
func (r *TemplateRepository) GetDesigns(ctx context.Context, templateID uuid.UUID) ([]model.TemplateDesign, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT td.id, td.template_id, td.design_id, td.is_primary, td.quantity, td.sequence_order, td.notes, td.created_at,
		       d.id, d.part_id, d.version, d.file_id, d.file_name, d.file_hash, d.file_size_bytes, d.file_type, d.notes, d.slice_profile, d.created_at
		FROM template_designs td
		JOIN designs d ON d.id = td.design_id
		WHERE td.template_id = ?
		ORDER BY td.sequence_order ASC, td.created_at ASC
	`, templateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var designs []model.TemplateDesign
	for rows.Next() {
		var td model.TemplateDesign
		var d model.Design
		if err := scanRow(rows,
			&td.ID, &td.TemplateID, &td.DesignID, &td.IsPrimary, &td.Quantity, &td.SequenceOrder, &td.Notes, &td.CreatedAt,
			&d.ID, &d.PartID, &d.Version, &d.FileID, &d.FileName, &d.FileHash, &d.FileSizeBytes, &d.FileType, &d.Notes, &d.SliceProfile, &d.CreatedAt,
		); err != nil {
			return nil, err
		}
		td.Design = &d
		designs = append(designs, td)
	}
	return designs, rows.Err()
}

// CreateRecipeMaterial inserts a new material requirement.
func (r *TemplateRepository) CreateRecipeMaterial(ctx context.Context, m *model.RecipeMaterial) error {
	m.ID = uuid.New()
	m.CreatedAt = time.Now()

	colorSpecJSON, _ := json.Marshal(m.ColorSpec)

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO recipe_materials (id, recipe_id, material_type, color_spec, weight_grams, ams_position, sequence_order, notes, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, m.ID, m.RecipeID, m.MaterialType, colorSpecJSON, m.WeightGrams, m.AMSPosition, m.SequenceOrder, m.Notes, m.CreatedAt)
	return err
}

// GetRecipeMaterials retrieves all materials for a recipe.
func (r *TemplateRepository) GetRecipeMaterials(ctx context.Context, recipeID uuid.UUID) ([]model.RecipeMaterial, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, recipe_id, material_type, color_spec, weight_grams, ams_position, sequence_order, notes, created_at
		FROM recipe_materials WHERE recipe_id = ? ORDER BY sequence_order ASC
	`, recipeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var materials []model.RecipeMaterial
	for rows.Next() {
		var m model.RecipeMaterial
		var colorSpecJSON []byte
		if err := scanRow(rows, &m.ID, &m.RecipeID, &m.MaterialType, &colorSpecJSON, &m.WeightGrams, &m.AMSPosition, &m.SequenceOrder, &m.Notes, &m.CreatedAt); err != nil {
			return nil, err
		}
		if colorSpecJSON != nil && len(colorSpecJSON) > 2 {
			var colorSpec model.ColorSpec
			if err := json.Unmarshal(colorSpecJSON, &colorSpec); err == nil {
				m.ColorSpec = &colorSpec
			}
		}
		materials = append(materials, m)
	}
	return materials, rows.Err()
}

// GetRecipeMaterialByID retrieves a single material by ID.
func (r *TemplateRepository) GetRecipeMaterialByID(ctx context.Context, id uuid.UUID) (*model.RecipeMaterial, error) {
	var m model.RecipeMaterial
	var colorSpecJSON []byte
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, recipe_id, material_type, color_spec, weight_grams, ams_position, sequence_order, notes, created_at
		FROM recipe_materials WHERE id = ?
	`, id), &m.ID, &m.RecipeID, &m.MaterialType, &colorSpecJSON, &m.WeightGrams, &m.AMSPosition, &m.SequenceOrder, &m.Notes, &m.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if colorSpecJSON != nil && len(colorSpecJSON) > 2 {
		var colorSpec model.ColorSpec
		if err := json.Unmarshal(colorSpecJSON, &colorSpec); err == nil {
			m.ColorSpec = &colorSpec
		}
	}
	return &m, nil
}

// UpdateRecipeMaterial updates a material requirement.
func (r *TemplateRepository) UpdateRecipeMaterial(ctx context.Context, m *model.RecipeMaterial) error {
	colorSpecJSON, _ := json.Marshal(m.ColorSpec)

	_, err := r.db.ExecContext(ctx, `
		UPDATE recipe_materials SET material_type = ?, color_spec = ?, weight_grams = ?, ams_position = ?, sequence_order = ?, notes = ?
		WHERE id = ?
	`, m.MaterialType, colorSpecJSON, m.WeightGrams, m.AMSPosition, m.SequenceOrder, m.Notes, m.ID)
	return err
}

// DeleteRecipeMaterial removes a material requirement.
func (r *TemplateRepository) DeleteRecipeMaterial(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM recipe_materials WHERE id = ?`, id)
	return err
}

// AddRecipeSupply inserts a new supply item for a recipe.
func (r *TemplateRepository) AddRecipeSupply(ctx context.Context, s *model.RecipeSupply) error {
	s.ID = uuid.New()
	s.CreatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO recipe_supplies (id, recipe_id, name, unit_cost_cents, quantity, material_id, notes, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, s.ID, s.RecipeID, s.Name, s.UnitCostCents, s.Quantity, s.MaterialID, s.Notes, s.CreatedAt)
	return err
}

// GetRecipeSupplies retrieves all supplies for a recipe.
func (r *TemplateRepository) GetRecipeSupplies(ctx context.Context, recipeID uuid.UUID) ([]model.RecipeSupply, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, recipe_id, name, unit_cost_cents, quantity, material_id, notes, created_at
		FROM recipe_supplies WHERE recipe_id = ? ORDER BY created_at ASC
	`, recipeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var supplies []model.RecipeSupply
	for rows.Next() {
		var s model.RecipeSupply
		if err := scanRow(rows, &s.ID, &s.RecipeID, &s.Name, &s.UnitCostCents, &s.Quantity, &s.MaterialID, &s.Notes, &s.CreatedAt); err != nil {
			return nil, err
		}
		supplies = append(supplies, s)
	}
	return supplies, rows.Err()
}

// UpdateRecipeSupply updates a supply item.
func (r *TemplateRepository) UpdateRecipeSupply(ctx context.Context, s *model.RecipeSupply) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE recipe_supplies SET name = ?, unit_cost_cents = ?, quantity = ?, notes = ?
		WHERE id = ?
	`, s.Name, s.UnitCostCents, s.Quantity, s.Notes, s.ID)
	return err
}

// DeleteRecipeSupply removes a supply item.
func (r *TemplateRepository) DeleteRecipeSupply(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM recipe_supplies WHERE id = ?`, id)
	return err
}

// GetRecipeSupplyByID retrieves a single supply by ID.
func (r *TemplateRepository) GetRecipeSupplyByID(ctx context.Context, id uuid.UUID) (*model.RecipeSupply, error) {
	var s model.RecipeSupply
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, recipe_id, name, unit_cost_cents, quantity, material_id, notes, created_at
		FROM recipe_supplies WHERE id = ?
	`, id), &s.ID, &s.RecipeID, &s.Name, &s.UnitCostCents, &s.Quantity, &s.MaterialID, &s.Notes, &s.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// FindCompatiblePrinters queries printers matching recipe constraints.
func (r *TemplateRepository) FindCompatiblePrinters(ctx context.Context, recipeID uuid.UUID) ([]model.Printer, error) {
	// Get the recipe to check its constraints
	template, err := r.GetByID(ctx, recipeID)
	if err != nil {
		return nil, err
	}
	if template == nil {
		return nil, nil
	}

	// Start with all printers
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, model, manufacturer, connection_type, connection_uri, api_key, status, build_volume, nozzle_diameter, location, notes, created_at, updated_at
		FROM printers ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var allPrinters []model.Printer
	for rows.Next() {
		var p model.Printer
		var buildVolumeJSON []byte
		if err := scanRow(rows, &p.ID, &p.Name, &p.Model, &p.Manufacturer, &p.ConnectionType, &p.ConnectionURI, &p.APIKey, &p.Status, &buildVolumeJSON, &p.NozzleDiameter, &p.Location, &p.Notes, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		if buildVolumeJSON != nil {
			json.Unmarshal(buildVolumeJSON, &p.BuildVolume)
		}
		allPrinters = append(allPrinters, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// If no constraints, return all printers
	if template.PrinterConstraints == nil {
		return allPrinters, nil
	}

	constraints := template.PrinterConstraints

	// Filter printers based on constraints
	var compatible []model.Printer
	for _, p := range allPrinters {
		// Check bed size
		if constraints.MinBedSize != nil && p.BuildVolume != nil {
			if p.BuildVolume.X < constraints.MinBedSize.X ||
				p.BuildVolume.Y < constraints.MinBedSize.Y ||
				p.BuildVolume.Z < constraints.MinBedSize.Z {
				continue
			}
		}

		// Check nozzle diameter
		if len(constraints.NozzleDiameters) > 0 {
			found := false
			for _, d := range constraints.NozzleDiameters {
				if p.NozzleDiameter == d {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		compatible = append(compatible, p)
	}

	return compatible, nil
}

// BambuCloudRepository handles Bambu Cloud auth token storage.
type BambuCloudRepository struct {
	db *sql.DB
}

// Upsert stores or updates Bambu Cloud auth credentials.
// Only one row is expected (single account). Tokens are encrypted before storage.
func (r *BambuCloudRepository) Upsert(ctx context.Context, auth *model.BambuCloudAuth) error {
	if auth.ID == uuid.Nil {
		auth.ID = uuid.New()
	}
	auth.UpdatedAt = time.Now()
	if auth.CreatedAt.IsZero() {
		auth.CreatedAt = auth.UpdatedAt
	}

	// Encrypt access token before storing
	accessToken := auth.AccessToken
	if accessToken != "" {
		if encrypted, err := crypto.Encrypt(accessToken); err == nil {
			accessToken = encrypted
		} else {
			slog.Warn("failed to encrypt bambu cloud access token", "error", err)
		}
	}

	// Encrypt refresh token before storing
	refreshToken := auth.RefreshToken
	if refreshToken != "" {
		if encrypted, err := crypto.Encrypt(refreshToken); err == nil {
			refreshToken = encrypted
		} else {
			slog.Warn("failed to encrypt bambu cloud refresh token", "error", err)
		}
	}

	// Delete existing then insert (simpler than upsert for single-row table)
	_, _ = r.db.ExecContext(ctx, `DELETE FROM bambu_cloud_auth`)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO bambu_cloud_auth (id, email, access_token, refresh_token, mqtt_username, expires_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, auth.ID, auth.Email, accessToken, refreshToken, auth.MQTTUsername, auth.ExpiresAt, auth.CreatedAt, auth.UpdatedAt)
	return err
}

// Get retrieves the stored Bambu Cloud auth (if any).
// Tokens are decrypted before returning.
func (r *BambuCloudRepository) Get(ctx context.Context) (*model.BambuCloudAuth, error) {
	var auth model.BambuCloudAuth
	err := scanRow(r.db.QueryRowContext(ctx, `
		SELECT id, email, access_token, refresh_token, mqtt_username, expires_at, created_at, updated_at
		FROM bambu_cloud_auth LIMIT 1
	`), &auth.ID, &auth.Email, &auth.AccessToken, &auth.RefreshToken, &auth.MQTTUsername, &auth.ExpiresAt, &auth.CreatedAt, &auth.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Decrypt access token
	if decrypted, err := crypto.Decrypt(auth.AccessToken); err == nil {
		auth.AccessToken = decrypted
	}

	// Decrypt refresh token
	if auth.RefreshToken != "" {
		if decrypted, err := crypto.Decrypt(auth.RefreshToken); err == nil {
			auth.RefreshToken = decrypted
		}
	}

	return &auth, nil
}

// Delete removes the stored Bambu Cloud auth.
func (r *BambuCloudRepository) Delete(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM bambu_cloud_auth`)
	return err
}

// SettingsRepository handles settings key-value storage.
type SettingsRepository struct {
	db *sql.DB
}

// Setting represents a single key-value setting.
type Setting struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Get retrieves a setting by key. Returns nil if not found.
func (r *SettingsRepository) Get(ctx context.Context, key string) (*Setting, error) {
	var s Setting
	var updatedAt string
	err := r.db.QueryRowContext(ctx, `SELECT key, value, updated_at FROM settings WHERE key = ?`, key).
		Scan(&s.Key, &s.Value, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	return &s, nil
}

// Set upserts a setting. Creates it if it doesn't exist, updates it if it does.
func (r *SettingsRepository) Set(ctx context.Context, key, value string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at
	`, key, value, time.Now().UTC().Format(time.RFC3339))
	return err
}

// List retrieves all settings.
func (r *SettingsRepository) List(ctx context.Context) ([]Setting, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT key, value, updated_at FROM settings ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settings []Setting
	for rows.Next() {
		var s Setting
		var updatedAt string
		if err := rows.Scan(&s.Key, &s.Value, &updatedAt); err != nil {
			return nil, err
		}
		s.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		settings = append(settings, s)
	}
	return settings, nil
}

// Delete removes a setting by key.
func (r *SettingsRepository) Delete(ctx context.Context, key string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM settings WHERE key = ?`, key)
	return err
}

// ProjectSupplyRepository handles project supply database operations.
type ProjectSupplyRepository struct {
	db *sql.DB
}

// Create inserts a new project supply.
func (r *ProjectSupplyRepository) Create(ctx context.Context, s *model.ProjectSupply) error {
	s.ID = uuid.New()
	now := time.Now()
	s.CreatedAt = now
	s.UpdatedAt = now
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO project_supplies (id, project_id, name, unit_cost_cents, quantity, notes, material_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, s.ID, s.ProjectID, s.Name, s.UnitCostCents, s.Quantity, s.Notes, s.MaterialID, s.CreatedAt, s.UpdatedAt)
	return err
}

// ListByProject retrieves all supplies for a project.
func (r *ProjectSupplyRepository) ListByProject(ctx context.Context, projectID uuid.UUID) ([]model.ProjectSupply, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, project_id, name, unit_cost_cents, quantity, notes, material_id, created_at, updated_at
		FROM project_supplies WHERE project_id = ? ORDER BY created_at ASC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var supplies []model.ProjectSupply
	for rows.Next() {
		var s model.ProjectSupply
		if err := scanRow(rows, &s.ID, &s.ProjectID, &s.Name, &s.UnitCostCents, &s.Quantity, &s.Notes, &s.MaterialID, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		supplies = append(supplies, s)
	}
	return supplies, rows.Err()
}

// Delete removes a project supply by ID.
func (r *ProjectSupplyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM project_supplies WHERE id = ?`, id)
	return err
}
