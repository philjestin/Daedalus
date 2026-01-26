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

	_, err := r.pool.Exec(ctx, `
		INSERT INTO projects (id, name, description, status, target_date, tags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, p.ID, p.Name, p.Description, p.Status, p.TargetDate, p.Tags, p.CreatedAt, p.UpdatedAt)
	return err
}

// GetByID retrieves a project by ID.
func (r *ProjectRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Project, error) {
	var p model.Project
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, description, status, target_date, tags, created_at, updated_at
		FROM projects WHERE id = $1
	`, id).Scan(&p.ID, &p.Name, &p.Description, &p.Status, &p.TargetDate, &p.Tags, &p.CreatedAt, &p.UpdatedAt)
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
			SELECT id, name, description, status, target_date, tags, created_at, updated_at
			FROM projects WHERE status = $1 ORDER BY updated_at DESC
		`, *status)
	} else {
		rows, err = r.pool.Query(ctx, `
			SELECT id, name, description, status, target_date, tags, created_at, updated_at
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
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Status, &p.TargetDate, &p.Tags, &p.CreatedAt, &p.UpdatedAt); err != nil {
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
		UPDATE projects SET name = $2, description = $3, status = $4, target_date = $5, tags = $6, updated_at = $7
		WHERE id = $1
	`, p.ID, p.Name, p.Description, p.Status, p.TargetDate, p.Tags, p.UpdatedAt)
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

// PrintJobRepository handles print job database operations.
type PrintJobRepository struct {
	pool *pgxpool.Pool
}

// Create inserts a new print job.
func (r *PrintJobRepository) Create(ctx context.Context, j *model.PrintJob) error {
	j.ID = uuid.New()
	j.CreatedAt = time.Now()
	if j.Status == "" {
		j.Status = model.PrintJobStatusQueued
	}

	outcomeJSON, _ := json.Marshal(j.Outcome)

	_, err := r.pool.Exec(ctx, `
		INSERT INTO print_jobs (id, design_id, printer_id, material_spool_id, status, progress, started_at, completed_at, outcome, notes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, j.ID, j.DesignID, j.PrinterID, j.MaterialSpoolID, j.Status, j.Progress, j.StartedAt, j.CompletedAt, outcomeJSON, j.Notes, j.CreatedAt)
	return err
}

// GetByID retrieves a print job by ID.
func (r *PrintJobRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.PrintJob, error) {
	var j model.PrintJob
	var outcomeJSON []byte
	err := r.pool.QueryRow(ctx, `
		SELECT id, design_id, printer_id, material_spool_id, status, progress, started_at, completed_at, outcome, notes, created_at
		FROM print_jobs WHERE id = $1
	`, id).Scan(&j.ID, &j.DesignID, &j.PrinterID, &j.MaterialSpoolID, &j.Status, &j.Progress, &j.StartedAt, &j.CompletedAt, &outcomeJSON, &j.Notes, &j.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if outcomeJSON != nil {
		json.Unmarshal(outcomeJSON, &j.Outcome)
	}
	return &j, nil
}

// List retrieves print jobs with optional filters.
func (r *PrintJobRepository) List(ctx context.Context, printerID *uuid.UUID, status *model.PrintJobStatus) ([]model.PrintJob, error) {
	query := `SELECT id, design_id, printer_id, material_spool_id, status, progress, started_at, completed_at, outcome, notes, created_at FROM print_jobs WHERE 1=1`
	args := []interface{}{}
	argNum := 1

	if printerID != nil {
		query += ` AND printer_id = $` + string(rune('0'+argNum))
		args = append(args, *printerID)
		argNum++
	}
	if status != nil {
		query += ` AND status = $` + string(rune('0'+argNum))
		args = append(args, *status)
		argNum++
	}
	query += ` ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []model.PrintJob
	for rows.Next() {
		var j model.PrintJob
		var outcomeJSON []byte
		if err := rows.Scan(&j.ID, &j.DesignID, &j.PrinterID, &j.MaterialSpoolID, &j.Status, &j.Progress, &j.StartedAt, &j.CompletedAt, &outcomeJSON, &j.Notes, &j.CreatedAt); err != nil {
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
		SELECT id, design_id, printer_id, material_spool_id, status, progress, started_at, completed_at, outcome, notes, created_at
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
		if err := rows.Scan(&j.ID, &j.DesignID, &j.PrinterID, &j.MaterialSpoolID, &j.Status, &j.Progress, &j.StartedAt, &j.CompletedAt, &outcomeJSON, &j.Notes, &j.CreatedAt); err != nil {
			return nil, err
		}
		if outcomeJSON != nil {
			json.Unmarshal(outcomeJSON, &j.Outcome)
		}
		jobs = append(jobs, j)
	}
	return jobs, rows.Err()
}

// Update updates a print job.
func (r *PrintJobRepository) Update(ctx context.Context, j *model.PrintJob) error {
	outcomeJSON, _ := json.Marshal(j.Outcome)

	_, err := r.pool.Exec(ctx, `
		UPDATE print_jobs SET status = $2, progress = $3, started_at = $4, completed_at = $5, outcome = $6, notes = $7
		WHERE id = $1
	`, j.ID, j.Status, j.Progress, j.StartedAt, j.CompletedAt, outcomeJSON, j.Notes)
	return err
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

