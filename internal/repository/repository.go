package repository

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/hyperion/printfarm/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repositories holds all repository instances.
type Repositories struct {
	Projects  *ProjectRepository
	Parts     *PartRepository
	Designs   *DesignRepository
	Printers  *PrinterRepository
	Materials *MaterialRepository
	Spools    *SpoolRepository
	PrintJobs *PrintJobRepository
	Files     *FileRepository
	Expenses  *ExpenseRepository
	Sales     *SaleRepository
	Templates *TemplateRepository
	Etsy      *EtsyRepository
}

// NewRepositories creates all repository instances.
func NewRepositories(pool *pgxpool.Pool) *Repositories {
	return &Repositories{
		Projects:  &ProjectRepository{pool: pool},
		Parts:     &PartRepository{pool: pool},
		Designs:   &DesignRepository{pool: pool},
		Printers:  &PrinterRepository{pool: pool},
		Materials: &MaterialRepository{pool: pool},
		Spools:    &SpoolRepository{pool: pool},
		PrintJobs: &PrintJobRepository{pool: pool},
		Files:     &FileRepository{pool: pool},
		Expenses:  &ExpenseRepository{pool: pool},
		Sales:     &SaleRepository{pool: pool},
		Templates: &TemplateRepository{pool: pool},
		Etsy:      &EtsyRepository{pool: pool},
	}
}

// ProjectRepository handles project database operations.
type ProjectRepository struct {
	pool *pgxpool.Pool
}

// Create inserts a new project.
func (r *ProjectRepository) Create(ctx context.Context, p *model.Project) error {
	p.ID = uuid.New()
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	if p.Status == "" {
		p.Status = model.ProjectStatusDraft
	}
	if p.Tags == nil {
		p.Tags = []string{}
	}
	if p.Source == "" {
		p.Source = "manual"
	}

	_, err := r.pool.Exec(ctx, `
		INSERT INTO projects (id, name, description, status, target_date, tags, template_id, source, external_order_id, customer_notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`, p.ID, p.Name, p.Description, p.Status, p.TargetDate, p.Tags, p.TemplateID, p.Source, p.ExternalOrderID, p.CustomerNotes, p.CreatedAt, p.UpdatedAt)
	return err
}

// GetByID retrieves a project by ID.
func (r *ProjectRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Project, error) {
	var p model.Project
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, description, status, target_date, tags, template_id, source, external_order_id, customer_notes, created_at, updated_at
		FROM projects WHERE id = $1
	`, id).Scan(&p.ID, &p.Name, &p.Description, &p.Status, &p.TargetDate, &p.Tags, &p.TemplateID, &p.Source, &p.ExternalOrderID, &p.CustomerNotes, &p.CreatedAt, &p.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &p, err
}

// List retrieves all projects with optional status filter.
func (r *ProjectRepository) List(ctx context.Context, status *model.ProjectStatus) ([]model.Project, error) {
	var rows pgx.Rows
	var err error

	if status != nil {
		rows, err = r.pool.Query(ctx, `
			SELECT id, name, description, status, target_date, tags, template_id, source, external_order_id, customer_notes, created_at, updated_at
			FROM projects WHERE status = $1 ORDER BY updated_at DESC
		`, *status)
	} else {
		rows, err = r.pool.Query(ctx, `
			SELECT id, name, description, status, target_date, tags, template_id, source, external_order_id, customer_notes, created_at, updated_at
			FROM projects ORDER BY updated_at DESC
		`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []model.Project
	for rows.Next() {
		var p model.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Status, &p.TargetDate, &p.Tags, &p.TemplateID, &p.Source, &p.ExternalOrderID, &p.CustomerNotes, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

// Update updates a project.
func (r *ProjectRepository) Update(ctx context.Context, p *model.Project) error {
	p.UpdatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		UPDATE projects SET name = $2, description = $3, status = $4, target_date = $5, tags = $6, template_id = $7, source = $8, external_order_id = $9, customer_notes = $10, updated_at = $11
		WHERE id = $1
	`, p.ID, p.Name, p.Description, p.Status, p.TargetDate, p.Tags, p.TemplateID, p.Source, p.ExternalOrderID, p.CustomerNotes, p.UpdatedAt)
	return err
}

// Delete removes a project.
func (r *ProjectRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM projects WHERE id = $1`, id)
	return err
}

// PartRepository handles part database operations.
type PartRepository struct {
	pool *pgxpool.Pool
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

	_, err := r.pool.Exec(ctx, `
		INSERT INTO parts (id, project_id, name, description, quantity, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, p.ID, p.ProjectID, p.Name, p.Description, p.Quantity, p.Status, p.CreatedAt, p.UpdatedAt)
	return err
}

// GetByID retrieves a part by ID.
func (r *PartRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Part, error) {
	var p model.Part
	err := r.pool.QueryRow(ctx, `
		SELECT id, project_id, name, description, quantity, status, created_at, updated_at
		FROM parts WHERE id = $1
	`, id).Scan(&p.ID, &p.ProjectID, &p.Name, &p.Description, &p.Quantity, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &p, err
}

// ListByProject retrieves all parts for a project.
func (r *PartRepository) ListByProject(ctx context.Context, projectID uuid.UUID) ([]model.Part, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, project_id, name, description, quantity, status, created_at, updated_at
		FROM parts WHERE project_id = $1 ORDER BY created_at ASC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var parts []model.Part
	for rows.Next() {
		var p model.Part
		if err := rows.Scan(&p.ID, &p.ProjectID, &p.Name, &p.Description, &p.Quantity, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		parts = append(parts, p)
	}
	return parts, rows.Err()
}

// Update updates a part.
func (r *PartRepository) Update(ctx context.Context, p *model.Part) error {
	p.UpdatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		UPDATE parts SET name = $2, description = $3, quantity = $4, status = $5, updated_at = $6
		WHERE id = $1
	`, p.ID, p.Name, p.Description, p.Quantity, p.Status, p.UpdatedAt)
	return err
}

// Delete removes a part.
func (r *PartRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM parts WHERE id = $1`, id)
	return err
}

// DesignRepository handles design database operations.
type DesignRepository struct {
	pool *pgxpool.Pool
}

// Create inserts a new design version.
func (r *DesignRepository) Create(ctx context.Context, d *model.Design) error {
	d.ID = uuid.New()
	d.CreatedAt = time.Now()

	// Get next version number for this part
	var maxVersion int
	err := r.pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(version), 0) FROM designs WHERE part_id = $1
	`, d.PartID).Scan(&maxVersion)
	if err != nil {
		return err
	}
	d.Version = maxVersion + 1

	_, err = r.pool.Exec(ctx, `
		INSERT INTO designs (id, part_id, version, file_id, file_name, file_hash, file_size_bytes, file_type, notes, slice_profile, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, d.ID, d.PartID, d.Version, d.FileID, d.FileName, d.FileHash, d.FileSizeBytes, d.FileType, d.Notes, d.SliceProfile, d.CreatedAt)
	return err
}

// GetByID retrieves a design by ID.
func (r *DesignRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Design, error) {
	var d model.Design
	err := r.pool.QueryRow(ctx, `
		SELECT id, part_id, version, file_id, file_name, file_hash, file_size_bytes, file_type, notes, slice_profile, created_at
		FROM designs WHERE id = $1
	`, id).Scan(&d.ID, &d.PartID, &d.Version, &d.FileID, &d.FileName, &d.FileHash, &d.FileSizeBytes, &d.FileType, &d.Notes, &d.SliceProfile, &d.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &d, err
}

// ListByPart retrieves all designs for a part.
func (r *DesignRepository) ListByPart(ctx context.Context, partID uuid.UUID) ([]model.Design, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, part_id, version, file_id, file_name, file_hash, file_size_bytes, file_type, notes, slice_profile, created_at
		FROM designs WHERE part_id = $1 ORDER BY version DESC
	`, partID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var designs []model.Design
	for rows.Next() {
		var d model.Design
		if err := rows.Scan(&d.ID, &d.PartID, &d.Version, &d.FileID, &d.FileName, &d.FileHash, &d.FileSizeBytes, &d.FileType, &d.Notes, &d.SliceProfile, &d.CreatedAt); err != nil {
			return nil, err
		}
		designs = append(designs, d)
	}
	return designs, rows.Err()
}

// PrinterRepository handles printer database operations.
type PrinterRepository struct {
	pool *pgxpool.Pool
}

// Create inserts a new printer.
func (r *PrinterRepository) Create(ctx context.Context, p *model.Printer) error {
	p.ID = uuid.New()
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	if p.Status == "" {
		p.Status = model.PrinterStatusOffline
	}

	buildVolumeJSON, _ := json.Marshal(p.BuildVolume)

	_, err := r.pool.Exec(ctx, `
		INSERT INTO printers (id, name, model, manufacturer, connection_type, connection_uri, api_key, status, build_volume, nozzle_diameter, location, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`, p.ID, p.Name, p.Model, p.Manufacturer, p.ConnectionType, p.ConnectionURI, p.APIKey, p.Status, buildVolumeJSON, p.NozzleDiameter, p.Location, p.Notes, p.CreatedAt, p.UpdatedAt)
	return err
}

// GetByID retrieves a printer by ID.
func (r *PrinterRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Printer, error) {
	var p model.Printer
	var buildVolumeJSON []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, model, manufacturer, connection_type, connection_uri, api_key, status, build_volume, nozzle_diameter, location, notes, created_at, updated_at
		FROM printers WHERE id = $1
	`, id).Scan(&p.ID, &p.Name, &p.Model, &p.Manufacturer, &p.ConnectionType, &p.ConnectionURI, &p.APIKey, &p.Status, &buildVolumeJSON, &p.NozzleDiameter, &p.Location, &p.Notes, &p.CreatedAt, &p.UpdatedAt)
	if err == pgx.ErrNoRows {
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
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, model, manufacturer, connection_type, connection_uri, api_key, status, build_volume, nozzle_diameter, location, notes, created_at, updated_at
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
		if err := rows.Scan(&p.ID, &p.Name, &p.Model, &p.Manufacturer, &p.ConnectionType, &p.ConnectionURI, &p.APIKey, &p.Status, &buildVolumeJSON, &p.NozzleDiameter, &p.Location, &p.Notes, &p.CreatedAt, &p.UpdatedAt); err != nil {
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

	_, err := r.pool.Exec(ctx, `
		UPDATE printers SET name = $2, model = $3, manufacturer = $4, connection_type = $5, connection_uri = $6, api_key = $7, status = $8, build_volume = $9, nozzle_diameter = $10, location = $11, notes = $12, updated_at = $13
		WHERE id = $1
	`, p.ID, p.Name, p.Model, p.Manufacturer, p.ConnectionType, p.ConnectionURI, p.APIKey, p.Status, buildVolumeJSON, p.NozzleDiameter, p.Location, p.Notes, p.UpdatedAt)
	return err
}

// UpdateStatus updates only the printer status.
func (r *PrinterRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.PrinterStatus) error {
	_, err := r.pool.Exec(ctx, `UPDATE printers SET status = $2, updated_at = $3 WHERE id = $1`, id, status, time.Now())
	return err
}

// Delete removes a printer.
func (r *PrinterRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM printers WHERE id = $1`, id)
	return err
}

// MaterialRepository handles material database operations.
type MaterialRepository struct {
	pool *pgxpool.Pool
}

// Create inserts a new material.
func (r *MaterialRepository) Create(ctx context.Context, m *model.Material) error {
	m.ID = uuid.New()
	m.CreatedAt = time.Now()
	m.UpdatedAt = time.Now()

	printTempJSON, _ := json.Marshal(m.PrintTemp)
	bedTempJSON, _ := json.Marshal(m.BedTemp)

	_, err := r.pool.Exec(ctx, `
		INSERT INTO materials (id, name, type, manufacturer, color, color_hex, density, cost_per_kg, print_temp, bed_temp, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, m.ID, m.Name, m.Type, m.Manufacturer, m.Color, m.ColorHex, m.Density, m.CostPerKg, printTempJSON, bedTempJSON, m.Notes, m.CreatedAt, m.UpdatedAt)
	return err
}

// GetByID retrieves a material by ID.
func (r *MaterialRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Material, error) {
	var m model.Material
	var printTempJSON, bedTempJSON []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, type, manufacturer, color, color_hex, density, cost_per_kg, print_temp, bed_temp, notes, created_at, updated_at
		FROM materials WHERE id = $1
	`, id).Scan(&m.ID, &m.Name, &m.Type, &m.Manufacturer, &m.Color, &m.ColorHex, &m.Density, &m.CostPerKg, &printTempJSON, &bedTempJSON, &m.Notes, &m.CreatedAt, &m.UpdatedAt)
	if err == pgx.ErrNoRows {
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
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, type, manufacturer, color, color_hex, density, cost_per_kg, print_temp, bed_temp, notes, created_at, updated_at
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
		if err := rows.Scan(&m.ID, &m.Name, &m.Type, &m.Manufacturer, &m.Color, &m.ColorHex, &m.Density, &m.CostPerKg, &printTempJSON, &bedTempJSON, &m.Notes, &m.CreatedAt, &m.UpdatedAt); err != nil {
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
	pool *pgxpool.Pool
}

// Create inserts a new spool.
func (r *SpoolRepository) Create(ctx context.Context, s *model.MaterialSpool) error {
	s.ID = uuid.New()
	s.CreatedAt = time.Now()
	s.UpdatedAt = time.Now()
	if s.Status == "" {
		s.Status = model.SpoolStatusNew
	}

	_, err := r.pool.Exec(ctx, `
		INSERT INTO material_spools (id, material_id, initial_weight, remaining_weight, purchase_date, purchase_cost, location, status, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, s.ID, s.MaterialID, s.InitialWeight, s.RemainingWeight, s.PurchaseDate, s.PurchaseCost, s.Location, s.Status, s.Notes, s.CreatedAt, s.UpdatedAt)
	return err
}

// GetByID retrieves a spool by ID.
func (r *SpoolRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.MaterialSpool, error) {
	var s model.MaterialSpool
	err := r.pool.QueryRow(ctx, `
		SELECT id, material_id, initial_weight, remaining_weight, purchase_date, purchase_cost, location, status, notes, created_at, updated_at
		FROM material_spools WHERE id = $1
	`, id).Scan(&s.ID, &s.MaterialID, &s.InitialWeight, &s.RemainingWeight, &s.PurchaseDate, &s.PurchaseCost, &s.Location, &s.Status, &s.Notes, &s.CreatedAt, &s.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &s, err
}

// List retrieves all spools.
func (r *SpoolRepository) List(ctx context.Context) ([]model.MaterialSpool, error) {
	rows, err := r.pool.Query(ctx, `
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
		if err := rows.Scan(&s.ID, &s.MaterialID, &s.InitialWeight, &s.RemainingWeight, &s.PurchaseDate, &s.PurchaseCost, &s.Location, &s.Status, &s.Notes, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		spools = append(spools, s)
	}
	return spools, rows.Err()
}

// Update updates a spool.
func (r *SpoolRepository) Update(ctx context.Context, s *model.MaterialSpool) error {
	s.UpdatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		UPDATE material_spools SET
			material_id = $2,
			initial_weight = $3,
			remaining_weight = $4,
			purchase_date = $5,
			purchase_cost = $6,
			location = $7,
			status = $8,
			notes = $9,
			updated_at = $10
		WHERE id = $1
	`, s.ID, s.MaterialID, s.InitialWeight, s.RemainingWeight, s.PurchaseDate, s.PurchaseCost, s.Location, s.Status, s.Notes, s.UpdatedAt)
	return err
}

// PrintJobRepository handles print job database operations.
type PrintJobRepository struct {
	pool *pgxpool.Pool
}

// Create inserts a new print job and records the initial "queued" event.
func (r *PrintJobRepository) Create(ctx context.Context, j *model.PrintJob) error {
	j.ID = uuid.New()
	j.CreatedAt = time.Now()
	if j.AttemptNumber == 0 {
		j.AttemptNumber = 1
	}
	j.Status = model.PrintJobStatusQueued // Always start as queued

	outcomeJSON, _ := json.Marshal(j.Outcome)

	// Insert the job record
	_, err := r.pool.Exec(ctx, `
		INSERT INTO print_jobs (id, design_id, printer_id, material_spool_id, status, progress, started_at, completed_at, outcome, notes, created_at, recipe_id, attempt_number, parent_job_id, estimated_seconds)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`, j.ID, j.DesignID, j.PrinterID, j.MaterialSpoolID, j.Status, j.Progress, j.StartedAt, j.CompletedAt, outcomeJSON, j.Notes, j.CreatedAt, j.RecipeID, j.AttemptNumber, j.ParentJobID, j.EstimatedSeconds)
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
	var outcomeJSON []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, design_id, printer_id, material_spool_id, status, progress, started_at, completed_at, outcome, notes, created_at,
		       recipe_id, attempt_number, parent_job_id, failure_category, estimated_seconds, actual_seconds, material_used_grams, cost_cents
		FROM print_jobs WHERE id = $1
	`, id).Scan(&j.ID, &j.DesignID, &j.PrinterID, &j.MaterialSpoolID, &j.Status, &j.Progress, &j.StartedAt, &j.CompletedAt, &outcomeJSON, &j.Notes, &j.CreatedAt,
		&j.RecipeID, &j.AttemptNumber, &j.ParentJobID, &j.FailureCategory, &j.EstimatedSeconds, &j.ActualSeconds, &j.MaterialUsedGrams, &j.CostCents)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if outcomeJSON != nil {
		json.Unmarshal(outcomeJSON, &j.Outcome)
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
	query := `SELECT pj.id, pj.design_id, pj.printer_id, pj.material_spool_id, pj.status, pj.progress, pj.started_at, pj.completed_at, pj.outcome, pj.notes, pj.created_at,
	                 pj.recipe_id, pj.attempt_number, pj.parent_job_id, pj.failure_category, pj.estimated_seconds, pj.actual_seconds, pj.material_used_grams, pj.cost_cents
	          FROM print_jobs pj WHERE 1=1`
	args := []interface{}{}
	argNum := 1

	if printerID != nil {
		query += fmt.Sprintf(" AND pj.printer_id = $%d", argNum)
		args = append(args, *printerID)
		argNum++
	}
	if status != nil {
		query += fmt.Sprintf(" AND pj.status = $%d", argNum)
		args = append(args, *status)
		argNum++
	}
	query += ` ORDER BY pj.created_at DESC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []model.PrintJob
	for rows.Next() {
		var j model.PrintJob
		var outcomeJSON []byte
		if err := rows.Scan(&j.ID, &j.DesignID, &j.PrinterID, &j.MaterialSpoolID, &j.Status, &j.Progress, &j.StartedAt, &j.CompletedAt, &outcomeJSON, &j.Notes, &j.CreatedAt,
			&j.RecipeID, &j.AttemptNumber, &j.ParentJobID, &j.FailureCategory, &j.EstimatedSeconds, &j.ActualSeconds, &j.MaterialUsedGrams, &j.CostCents); err != nil {
			return nil, err
		}
		if outcomeJSON != nil {
			json.Unmarshal(outcomeJSON, &j.Outcome)
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// ListByDesign retrieves all print jobs for a design.
func (r *PrintJobRepository) ListByDesign(ctx context.Context, designID uuid.UUID) ([]model.PrintJob, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, design_id, printer_id, material_spool_id, status, progress, started_at, completed_at, outcome, notes, created_at,
		       recipe_id, attempt_number, parent_job_id, failure_category, estimated_seconds, actual_seconds, material_used_grams, cost_cents
		FROM print_jobs WHERE design_id = $1 ORDER BY created_at DESC
	`, designID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []model.PrintJob
	for rows.Next() {
		var j model.PrintJob
		var outcomeJSON []byte
		if err := rows.Scan(&j.ID, &j.DesignID, &j.PrinterID, &j.MaterialSpoolID, &j.Status, &j.Progress, &j.StartedAt, &j.CompletedAt, &outcomeJSON, &j.Notes, &j.CreatedAt,
			&j.RecipeID, &j.AttemptNumber, &j.ParentJobID, &j.FailureCategory, &j.EstimatedSeconds, &j.ActualSeconds, &j.MaterialUsedGrams, &j.CostCents); err != nil {
			return nil, err
		}
		if outcomeJSON != nil {
			json.Unmarshal(outcomeJSON, &j.Outcome)
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

	_, err := r.pool.Exec(ctx, `
		UPDATE print_jobs SET status = $2, progress = $3, started_at = $4, completed_at = $5, outcome = $6, notes = $7,
		       failure_category = $8, actual_seconds = $9, material_used_grams = $10, cost_cents = $11
		WHERE id = $1
	`, j.ID, j.Status, j.Progress, j.StartedAt, j.CompletedAt, outcomeJSON, j.Notes,
		j.FailureCategory, j.ActualSeconds, j.MaterialUsedGrams, j.CostCents)
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

	_, err := r.pool.Exec(ctx, `
		INSERT INTO job_events (id, job_id, event_type, occurred_at, status, progress, printer_id, error_code, error_message, actor_type, actor_id, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, e.ID, e.JobID, e.EventType, e.OccurredAt, e.Status, e.Progress, e.PrinterID, e.ErrorCode, e.ErrorMessage, e.ActorType, e.ActorID, metadataJSON, e.CreatedAt)

	if err != nil {
		return err
	}

	// Update denormalized status on print_jobs table for query efficiency
	if e.Status != nil {
		_, err = r.pool.Exec(ctx, `UPDATE print_jobs SET status = $2, progress = COALESCE($3, progress) WHERE id = $1`, e.JobID, *e.Status, e.Progress)
	} else if e.Progress != nil {
		_, err = r.pool.Exec(ctx, `UPDATE print_jobs SET progress = $2 WHERE id = $1`, e.JobID, *e.Progress)
	}

	return err
}

// GetEvents retrieves all events for a job in chronological order.
func (r *PrintJobRepository) GetEvents(ctx context.Context, jobID uuid.UUID) ([]model.JobEvent, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, job_id, event_type, occurred_at, status, progress, printer_id, error_code, error_message, actor_type, actor_id, metadata, created_at
		FROM job_events WHERE job_id = $1 ORDER BY occurred_at ASC
	`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []model.JobEvent
	for rows.Next() {
		var e model.JobEvent
		var metadataJSON []byte
		if err := rows.Scan(&e.ID, &e.JobID, &e.EventType, &e.OccurredAt, &e.Status, &e.Progress, &e.PrinterID, &e.ErrorCode, &e.ErrorMessage, &e.ActorType, &e.ActorID, &metadataJSON, &e.CreatedAt); err != nil {
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
	err := r.pool.QueryRow(ctx, `
		SELECT status, progress FROM job_events
		WHERE job_id = $1 AND status IS NOT NULL
		ORDER BY occurred_at DESC LIMIT 1
	`, jobID).Scan(&status, &progress)
	if err == pgx.ErrNoRows {
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
		err := r.pool.QueryRow(ctx, `SELECT parent_job_id FROM print_jobs WHERE id = $1`, rootID).Scan(&parentID)
		if err != nil {
			return nil, err
		}
		if parentID == nil {
			break
		}
		rootID = *parentID
	}

	// Now get all jobs in the chain
	rows, err := r.pool.Query(ctx, `
		WITH RECURSIVE chain AS (
			SELECT id, design_id, printer_id, material_spool_id, status, progress, started_at, completed_at, outcome, notes, created_at,
			       recipe_id, attempt_number, parent_job_id, failure_category, estimated_seconds, actual_seconds, material_used_grams, cost_cents
			FROM print_jobs WHERE id = $1
			UNION ALL
			SELECT pj.id, pj.design_id, pj.printer_id, pj.material_spool_id, pj.status, pj.progress, pj.started_at, pj.completed_at, pj.outcome, pj.notes, pj.created_at,
			       pj.recipe_id, pj.attempt_number, pj.parent_job_id, pj.failure_category, pj.estimated_seconds, pj.actual_seconds, pj.material_used_grams, pj.cost_cents
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
		var outcomeJSON []byte
		if err := rows.Scan(&j.ID, &j.DesignID, &j.PrinterID, &j.MaterialSpoolID, &j.Status, &j.Progress, &j.StartedAt, &j.CompletedAt, &outcomeJSON, &j.Notes, &j.CreatedAt,
			&j.RecipeID, &j.AttemptNumber, &j.ParentJobID, &j.FailureCategory, &j.EstimatedSeconds, &j.ActualSeconds, &j.MaterialUsedGrams, &j.CostCents); err != nil {
			return nil, err
		}
		if outcomeJSON != nil {
			json.Unmarshal(outcomeJSON, &j.Outcome)
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// ListByRecipe retrieves all print jobs for a recipe/template.
func (r *PrintJobRepository) ListByRecipe(ctx context.Context, recipeID uuid.UUID) ([]model.PrintJob, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, design_id, printer_id, material_spool_id, status, progress, started_at, completed_at, outcome, notes, created_at,
		       recipe_id, attempt_number, parent_job_id, failure_category, estimated_seconds, actual_seconds, material_used_grams, cost_cents
		FROM print_jobs WHERE recipe_id = $1 ORDER BY created_at DESC
	`, recipeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []model.PrintJob
	for rows.Next() {
		var j model.PrintJob
		var outcomeJSON []byte
		if err := rows.Scan(&j.ID, &j.DesignID, &j.PrinterID, &j.MaterialSpoolID, &j.Status, &j.Progress, &j.StartedAt, &j.CompletedAt, &outcomeJSON, &j.Notes, &j.CreatedAt,
			&j.RecipeID, &j.AttemptNumber, &j.ParentJobID, &j.FailureCategory, &j.EstimatedSeconds, &j.ActualSeconds, &j.MaterialUsedGrams, &j.CostCents); err != nil {
			return nil, err
		}
		if outcomeJSON != nil {
			json.Unmarshal(outcomeJSON, &j.Outcome)
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// FileRepository handles file metadata database operations.
type FileRepository struct {
	pool *pgxpool.Pool
}

// Create inserts a new file record.
func (r *FileRepository) Create(ctx context.Context, f *model.File) error {
	f.ID = uuid.New()
	f.CreatedAt = time.Now()

	_, err := r.pool.Exec(ctx, `
		INSERT INTO files (id, hash, original_name, content_type, size_bytes, storage_path, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, f.ID, f.Hash, f.OriginalName, f.ContentType, f.SizeBytes, f.StoragePath, f.CreatedAt)
	return err
}

// GetByID retrieves a file by ID.
func (r *FileRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.File, error) {
	var f model.File
	err := r.pool.QueryRow(ctx, `
		SELECT id, hash, original_name, content_type, size_bytes, storage_path, created_at
		FROM files WHERE id = $1
	`, id).Scan(&f.ID, &f.Hash, &f.OriginalName, &f.ContentType, &f.SizeBytes, &f.StoragePath, &f.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &f, err
}

// GetByHash retrieves a file by hash (for deduplication).
func (r *FileRepository) GetByHash(ctx context.Context, hash string) (*model.File, error) {
	var f model.File
	err := r.pool.QueryRow(ctx, `
		SELECT id, hash, original_name, content_type, size_bytes, storage_path, created_at
		FROM files WHERE hash = $1
	`, hash).Scan(&f.ID, &f.Hash, &f.OriginalName, &f.ContentType, &f.SizeBytes, &f.StoragePath, &f.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &f, err
}

// ExpenseRepository handles expense database operations.
type ExpenseRepository struct {
	pool *pgxpool.Pool
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

	_, err := r.pool.Exec(ctx, `
		INSERT INTO expenses (id, occurred_at, vendor, subtotal_cents, tax_cents, shipping_cents, total_cents, currency, category, notes, receipt_file_id, receipt_file_path, status, raw_ocr_text, raw_ai_response, confidence, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`, e.ID, e.OccurredAt, e.Vendor, e.SubtotalCents, e.TaxCents, e.ShippingCents, e.TotalCents, e.Currency, e.Category, e.Notes, e.ReceiptFileID, e.ReceiptFilePath, e.Status, e.RawOCRText, rawAIJSON, e.Confidence, e.CreatedAt, e.UpdatedAt)
	return err
}

// GetByID retrieves an expense by ID.
func (r *ExpenseRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Expense, error) {
	var e model.Expense
	var rawAIJSON []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, occurred_at, vendor, subtotal_cents, tax_cents, shipping_cents, total_cents, currency, category, notes, receipt_file_id, receipt_file_path, status, raw_ocr_text, raw_ai_response, confidence, created_at, updated_at
		FROM expenses WHERE id = $1
	`, id).Scan(&e.ID, &e.OccurredAt, &e.Vendor, &e.SubtotalCents, &e.TaxCents, &e.ShippingCents, &e.TotalCents, &e.Currency, &e.Category, &e.Notes, &e.ReceiptFileID, &e.ReceiptFilePath, &e.Status, &e.RawOCRText, &rawAIJSON, &e.Confidence, &e.CreatedAt, &e.UpdatedAt)
	if err == pgx.ErrNoRows {
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
		query += ` WHERE status = $1`
		args = append(args, *status)
	}
	query += ` ORDER BY occurred_at DESC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var expenses []model.Expense
	for rows.Next() {
		var e model.Expense
		if err := rows.Scan(&e.ID, &e.OccurredAt, &e.Vendor, &e.SubtotalCents, &e.TaxCents, &e.ShippingCents, &e.TotalCents, &e.Currency, &e.Category, &e.Notes, &e.ReceiptFileID, &e.ReceiptFilePath, &e.Status, &e.Confidence, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		expenses = append(expenses, e)
	}
	return expenses, rows.Err()
}

// Update updates an expense.
func (r *ExpenseRepository) Update(ctx context.Context, e *model.Expense) error {
	e.UpdatedAt = time.Now()
	rawAIJSON, _ := json.Marshal(e.RawAIResponse)

	_, err := r.pool.Exec(ctx, `
		UPDATE expenses SET occurred_at = $2, vendor = $3, subtotal_cents = $4, tax_cents = $5, shipping_cents = $6, total_cents = $7, currency = $8, category = $9, notes = $10, status = $11, raw_ocr_text = $12, raw_ai_response = $13, confidence = $14, updated_at = $15
		WHERE id = $1
	`, e.ID, e.OccurredAt, e.Vendor, e.SubtotalCents, e.TaxCents, e.ShippingCents, e.TotalCents, e.Currency, e.Category, e.Notes, e.Status, e.RawOCRText, rawAIJSON, e.Confidence, e.UpdatedAt)
	return err
}

// Delete deletes an expense.
func (r *ExpenseRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM expenses WHERE id = $1`, id)
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

	_, err := r.pool.Exec(ctx, `
		INSERT INTO expense_items (id, expense_id, description, quantity, unit_price_cents, total_price_cents, sku, vendor_item_id, category, metadata, matched_spool_id, matched_material_id, confidence, action_taken, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`, item.ID, item.ExpenseID, item.Description, item.Quantity, item.UnitPriceCents, item.TotalPriceCents, item.SKU, item.VendorItemID, item.Category, metadataJSON, item.MatchedSpoolID, item.MatchedMaterialID, item.Confidence, item.ActionTaken, item.CreatedAt)
	return err
}

// GetItemsByExpenseID retrieves all items for an expense.
func (r *ExpenseRepository) GetItemsByExpenseID(ctx context.Context, expenseID uuid.UUID) ([]model.ExpenseItem, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, expense_id, description, quantity, unit_price_cents, total_price_cents, sku, vendor_item_id, category, metadata, matched_spool_id, matched_material_id, confidence, action_taken, created_at
		FROM expense_items WHERE expense_id = $1 ORDER BY created_at
	`, expenseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.ExpenseItem
	for rows.Next() {
		var item model.ExpenseItem
		var metadataJSON []byte
		if err := rows.Scan(&item.ID, &item.ExpenseID, &item.Description, &item.Quantity, &item.UnitPriceCents, &item.TotalPriceCents, &item.SKU, &item.VendorItemID, &item.Category, &metadataJSON, &item.MatchedSpoolID, &item.MatchedMaterialID, &item.Confidence, &item.ActionTaken, &item.CreatedAt); err != nil {
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
	metadataJSON, _ := json.Marshal(item.Metadata)

	_, err := r.pool.Exec(ctx, `
		UPDATE expense_items SET description = $2, quantity = $3, unit_price_cents = $4, total_price_cents = $5, sku = $6, vendor_item_id = $7, category = $8, metadata = $9, matched_spool_id = $10, matched_material_id = $11, confidence = $12, action_taken = $13
		WHERE id = $1
	`, item.ID, item.Description, item.Quantity, item.UnitPriceCents, item.TotalPriceCents, item.SKU, item.VendorItemID, item.Category, metadataJSON, item.MatchedSpoolID, item.MatchedMaterialID, item.Confidence, item.ActionTaken)
	return err
}

// SaleRepository handles sale database operations.
type SaleRepository struct {
	pool *pgxpool.Pool
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

	_, err := r.pool.Exec(ctx, `
		INSERT INTO sales (id, occurred_at, channel, platform, gross_cents, fees_cents, shipping_charged_cents, shipping_cost_cents, tax_collected_cents, net_cents, currency, project_id, order_reference, customer_name, item_description, quantity, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`, s.ID, s.OccurredAt, s.Channel, s.Platform, s.GrossCents, s.FeesCents, s.ShippingChargedCents, s.ShippingCostCents, s.TaxCollectedCents, s.NetCents, s.Currency, s.ProjectID, s.OrderReference, s.CustomerName, s.ItemDescription, s.Quantity, s.Notes, s.CreatedAt, s.UpdatedAt)
	return err
}

// GetByID retrieves a sale by ID.
func (r *SaleRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Sale, error) {
	var s model.Sale
	err := r.pool.QueryRow(ctx, `
		SELECT id, occurred_at, channel, platform, gross_cents, fees_cents, shipping_charged_cents, shipping_cost_cents, tax_collected_cents, net_cents, currency, project_id, order_reference, customer_name, item_description, quantity, notes, created_at, updated_at
		FROM sales WHERE id = $1
	`, id).Scan(&s.ID, &s.OccurredAt, &s.Channel, &s.Platform, &s.GrossCents, &s.FeesCents, &s.ShippingChargedCents, &s.ShippingCostCents, &s.TaxCollectedCents, &s.NetCents, &s.Currency, &s.ProjectID, &s.OrderReference, &s.CustomerName, &s.ItemDescription, &s.Quantity, &s.Notes, &s.CreatedAt, &s.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &s, err
}

// List retrieves all sales.
func (r *SaleRepository) List(ctx context.Context, projectID *uuid.UUID) ([]model.Sale, error) {
	query := `SELECT id, occurred_at, channel, platform, gross_cents, fees_cents, shipping_charged_cents, shipping_cost_cents, tax_collected_cents, net_cents, currency, project_id, order_reference, customer_name, item_description, quantity, notes, created_at, updated_at FROM sales`
	args := []interface{}{}

	if projectID != nil {
		query += ` WHERE project_id = $1`
		args = append(args, *projectID)
	}
	query += ` ORDER BY occurred_at DESC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sales []model.Sale
	for rows.Next() {
		var s model.Sale
		if err := rows.Scan(&s.ID, &s.OccurredAt, &s.Channel, &s.Platform, &s.GrossCents, &s.FeesCents, &s.ShippingChargedCents, &s.ShippingCostCents, &s.TaxCollectedCents, &s.NetCents, &s.Currency, &s.ProjectID, &s.OrderReference, &s.CustomerName, &s.ItemDescription, &s.Quantity, &s.Notes, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		sales = append(sales, s)
	}
	return sales, rows.Err()
}

// Update updates a sale.
func (r *SaleRepository) Update(ctx context.Context, s *model.Sale) error {
	s.UpdatedAt = time.Now()
	_, err := r.pool.Exec(ctx, `
		UPDATE sales SET occurred_at = $2, channel = $3, platform = $4, gross_cents = $5, fees_cents = $6, shipping_charged_cents = $7, shipping_cost_cents = $8, tax_collected_cents = $9, net_cents = $10, currency = $11, project_id = $12, order_reference = $13, customer_name = $14, item_description = $15, quantity = $16, notes = $17, updated_at = $18
		WHERE id = $1
	`, s.ID, s.OccurredAt, s.Channel, s.Platform, s.GrossCents, s.FeesCents, s.ShippingChargedCents, s.ShippingCostCents, s.TaxCollectedCents, s.NetCents, s.Currency, s.ProjectID, s.OrderReference, s.CustomerName, s.ItemDescription, s.Quantity, s.Notes, s.UpdatedAt)
	return err
}

// Delete deletes a sale.
func (r *SaleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sales WHERE id = $1`, id)
	return err
}

// GetTotalsByDateRange calculates totals for a date range.
func (r *SaleRepository) GetTotalsByDateRange(ctx context.Context, start, end time.Time) (grossCents, netCents, feesCents int, count int, err error) {
	err = r.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(gross_cents), 0), COALESCE(SUM(net_cents), 0), COALESCE(SUM(fees_cents), 0), COUNT(*)
		FROM sales WHERE occurred_at >= $1 AND occurred_at < $2
	`, start, end).Scan(&grossCents, &netCents, &feesCents, &count)
	return
}

// TemplateRepository handles template database operations.
type TemplateRepository struct {
	pool *pgxpool.Pool
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

	checklistJSON, _ := json.Marshal(t.PostProcessChecklist)
	constraintsJSON, _ := json.Marshal(t.PrinterConstraints)

	_, err := r.pool.Exec(ctx, `
		INSERT INTO templates (id, name, description, sku, tags, material_type, estimated_material_grams, preferred_printer_id, allow_any_printer, quantity_per_order, post_process_checklist, is_active, printer_constraints, print_profile, estimated_print_seconds, version, archived_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
	`, t.ID, t.Name, t.Description, t.SKU, t.Tags, t.MaterialType, t.EstimatedMaterialGrams, t.PreferredPrinterID, t.AllowAnyPrinter, t.QuantityPerOrder, checklistJSON, t.IsActive, constraintsJSON, t.PrintProfile, t.EstimatedPrintSeconds, t.Version, t.ArchivedAt, t.CreatedAt, t.UpdatedAt)
	return err
}

// GetByID retrieves a template by ID.
func (r *TemplateRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Template, error) {
	var t model.Template
	var checklistJSON, constraintsJSON []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, description, sku, tags, material_type, estimated_material_grams, preferred_printer_id, allow_any_printer, quantity_per_order, post_process_checklist, is_active, COALESCE(printer_constraints, '{}'), COALESCE(print_profile, 'standard'), COALESCE(estimated_print_seconds, 0), COALESCE(version, 1), archived_at, created_at, updated_at
		FROM templates WHERE id = $1
	`, id).Scan(&t.ID, &t.Name, &t.Description, &t.SKU, &t.Tags, &t.MaterialType, &t.EstimatedMaterialGrams, &t.PreferredPrinterID, &t.AllowAnyPrinter, &t.QuantityPerOrder, &checklistJSON, &t.IsActive, &constraintsJSON, &t.PrintProfile, &t.EstimatedPrintSeconds, &t.Version, &t.ArchivedAt, &t.CreatedAt, &t.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
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
	var checklistJSON, constraintsJSON []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, description, sku, tags, material_type, estimated_material_grams, preferred_printer_id, allow_any_printer, quantity_per_order, post_process_checklist, is_active, COALESCE(printer_constraints, '{}'), COALESCE(print_profile, 'standard'), COALESCE(estimated_print_seconds, 0), COALESCE(version, 1), archived_at, created_at, updated_at
		FROM templates WHERE sku = $1
	`, sku).Scan(&t.ID, &t.Name, &t.Description, &t.SKU, &t.Tags, &t.MaterialType, &t.EstimatedMaterialGrams, &t.PreferredPrinterID, &t.AllowAnyPrinter, &t.QuantityPerOrder, &checklistJSON, &t.IsActive, &constraintsJSON, &t.PrintProfile, &t.EstimatedPrintSeconds, &t.Version, &t.ArchivedAt, &t.CreatedAt, &t.UpdatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
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
	var rows pgx.Rows
	var err error

	query := `SELECT id, name, description, sku, tags, material_type, estimated_material_grams, preferred_printer_id, allow_any_printer, quantity_per_order, post_process_checklist, is_active, COALESCE(printer_constraints, '{}'), COALESCE(print_profile, 'standard'), COALESCE(estimated_print_seconds, 0), COALESCE(version, 1), archived_at, created_at, updated_at FROM templates`
	if activeOnly {
		query += ` WHERE is_active = true AND archived_at IS NULL`
	}
	query += ` ORDER BY name ASC`

	rows, err = r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []model.Template
	for rows.Next() {
		var t model.Template
		var checklistJSON, constraintsJSON []byte
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.SKU, &t.Tags, &t.MaterialType, &t.EstimatedMaterialGrams, &t.PreferredPrinterID, &t.AllowAnyPrinter, &t.QuantityPerOrder, &checklistJSON, &t.IsActive, &constraintsJSON, &t.PrintProfile, &t.EstimatedPrintSeconds, &t.Version, &t.ArchivedAt, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
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
	checklistJSON, _ := json.Marshal(t.PostProcessChecklist)
	constraintsJSON, _ := json.Marshal(t.PrinterConstraints)

	_, err := r.pool.Exec(ctx, `
		UPDATE templates SET name = $2, description = $3, sku = $4, tags = $5, material_type = $6, estimated_material_grams = $7, preferred_printer_id = $8, allow_any_printer = $9, quantity_per_order = $10, post_process_checklist = $11, is_active = $12, printer_constraints = $13, print_profile = $14, estimated_print_seconds = $15, version = $16, archived_at = $17, updated_at = $18
		WHERE id = $1
	`, t.ID, t.Name, t.Description, t.SKU, t.Tags, t.MaterialType, t.EstimatedMaterialGrams, t.PreferredPrinterID, t.AllowAnyPrinter, t.QuantityPerOrder, checklistJSON, t.IsActive, constraintsJSON, t.PrintProfile, t.EstimatedPrintSeconds, t.Version, t.ArchivedAt, t.UpdatedAt)
	return err
}

// Delete removes a template.
func (r *TemplateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM templates WHERE id = $1`, id)
	return err
}

// AddDesign adds a design to a template.
func (r *TemplateRepository) AddDesign(ctx context.Context, td *model.TemplateDesign) error {
	td.ID = uuid.New()
	td.CreatedAt = time.Now()
	if td.Quantity == 0 {
		td.Quantity = 1
	}

	_, err := r.pool.Exec(ctx, `
		INSERT INTO template_designs (id, template_id, design_id, is_primary, quantity, sequence_order, notes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, td.ID, td.TemplateID, td.DesignID, td.IsPrimary, td.Quantity, td.SequenceOrder, td.Notes, td.CreatedAt)
	return err
}

// RemoveDesign removes a design from a template.
func (r *TemplateRepository) RemoveDesign(ctx context.Context, templateID, designID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM template_designs WHERE template_id = $1 AND design_id = $2`, templateID, designID)
	return err
}

// GetDesigns retrieves all designs for a template.
func (r *TemplateRepository) GetDesigns(ctx context.Context, templateID uuid.UUID) ([]model.TemplateDesign, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT td.id, td.template_id, td.design_id, td.is_primary, td.quantity, td.sequence_order, td.notes, td.created_at,
		       d.id, d.part_id, d.version, d.file_id, d.file_name, d.file_hash, d.file_size_bytes, d.file_type, d.notes, d.slice_profile, d.created_at
		FROM template_designs td
		JOIN designs d ON d.id = td.design_id
		WHERE td.template_id = $1
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
		if err := rows.Scan(
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

	_, err := r.pool.Exec(ctx, `
		INSERT INTO recipe_materials (id, recipe_id, material_type, color_spec, weight_grams, ams_position, sequence_order, notes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, m.ID, m.RecipeID, m.MaterialType, colorSpecJSON, m.WeightGrams, m.AMSPosition, m.SequenceOrder, m.Notes, m.CreatedAt)
	return err
}

// GetRecipeMaterials retrieves all materials for a recipe.
func (r *TemplateRepository) GetRecipeMaterials(ctx context.Context, recipeID uuid.UUID) ([]model.RecipeMaterial, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, recipe_id, material_type, color_spec, weight_grams, ams_position, sequence_order, notes, created_at
		FROM recipe_materials WHERE recipe_id = $1 ORDER BY sequence_order ASC
	`, recipeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var materials []model.RecipeMaterial
	for rows.Next() {
		var m model.RecipeMaterial
		var colorSpecJSON []byte
		if err := rows.Scan(&m.ID, &m.RecipeID, &m.MaterialType, &colorSpecJSON, &m.WeightGrams, &m.AMSPosition, &m.SequenceOrder, &m.Notes, &m.CreatedAt); err != nil {
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
	err := r.pool.QueryRow(ctx, `
		SELECT id, recipe_id, material_type, color_spec, weight_grams, ams_position, sequence_order, notes, created_at
		FROM recipe_materials WHERE id = $1
	`, id).Scan(&m.ID, &m.RecipeID, &m.MaterialType, &colorSpecJSON, &m.WeightGrams, &m.AMSPosition, &m.SequenceOrder, &m.Notes, &m.CreatedAt)
	if err == pgx.ErrNoRows {
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

	_, err := r.pool.Exec(ctx, `
		UPDATE recipe_materials SET material_type = $2, color_spec = $3, weight_grams = $4, ams_position = $5, sequence_order = $6, notes = $7
		WHERE id = $1
	`, m.ID, m.MaterialType, colorSpecJSON, m.WeightGrams, m.AMSPosition, m.SequenceOrder, m.Notes)
	return err
}

// DeleteRecipeMaterial removes a material requirement.
func (r *TemplateRepository) DeleteRecipeMaterial(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM recipe_materials WHERE id = $1`, id)
	return err
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
	rows, err := r.pool.Query(ctx, `
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
		if err := rows.Scan(&p.ID, &p.Name, &p.Model, &p.Manufacturer, &p.ConnectionType, &p.ConnectionURI, &p.APIKey, &p.Status, &buildVolumeJSON, &p.NozzleDiameter, &p.Location, &p.Notes, &p.CreatedAt, &p.UpdatedAt); err != nil {
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

		// Additional constraints like enclosure, AMS would need printer metadata extensions
		// For now, we pass these constraints to the frontend for user validation

		compatible = append(compatible, p)
	}

	return compatible, nil
}

