package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hyperion/printfarm/internal/model"
	"github.com/hyperion/printfarm/internal/printer"
	"github.com/hyperion/printfarm/internal/realtime"
	"github.com/hyperion/printfarm/internal/receipt"
	"github.com/hyperion/printfarm/internal/repository"
	"github.com/hyperion/printfarm/internal/storage"
)

// Services holds all service instances.
type Services struct {
	Projects  *ProjectService
	Parts     *PartService
	Designs   *DesignService
	Printers  *PrinterService
	Materials *MaterialService
	Spools    *SpoolService
	PrintJobs *PrintJobService
	Files     *FileService
	Expenses  *ExpenseService
	Sales     *SaleService
	Stats     *StatsService
	Templates *TemplateService
	Etsy      *EtsyService
}

// EtsyConfig holds Etsy OAuth configuration.
type EtsyConfig struct {
	ClientID    string
	RedirectURI string
}

// NewServices creates all service instances.
func NewServices(repos *repository.Repositories, store storage.Storage, printerMgr *printer.Manager, hub *realtime.Hub) *Services {
	return &Services{
		Projects:  &ProjectService{repo: repos.Projects},
		Parts:     &PartService{repo: repos.Parts},
		Designs:   &DesignService{repo: repos.Designs, fileRepo: repos.Files, storage: store},
		Printers:  &PrinterService{repo: repos.Printers, manager: printerMgr, hub: hub, discovery: printer.NewDiscovery()},
		Materials: &MaterialService{repo: repos.Materials},
		Spools:    &SpoolService{repo: repos.Spools},
		PrintJobs: &PrintJobService{repo: repos.PrintJobs, printerRepo: repos.Printers, designRepo: repos.Designs, spoolRepo: repos.Spools, materialRepo: repos.Materials, printerMgr: printerMgr, hub: hub, storage: store},
		Files:     &FileService{repo: repos.Files, storage: store},
		Expenses:  &ExpenseService{repo: repos.Expenses, materialRepo: repos.Materials, spoolRepo: repos.Spools, fileRepo: repos.Files, storage: store},
		Sales:     &SaleService{repo: repos.Sales},
		Stats:     &StatsService{expenseRepo: repos.Expenses, saleRepo: repos.Sales, printJobRepo: repos.PrintJobs},
		Templates: &TemplateService{repo: repos.Templates, projectRepo: repos.Projects, partRepo: repos.Parts, designRepo: repos.Designs, printJobRepo: repos.PrintJobs, spoolRepo: repos.Spools, materialRepo: repos.Materials, printerRepo: repos.Printers},
		Etsy:      nil, // Initialize separately with NewServicesWithEtsy
	}
}

// NewServicesWithEtsy creates all service instances including Etsy integration.
func NewServicesWithEtsy(repos *repository.Repositories, store storage.Storage, printerMgr *printer.Manager, hub *realtime.Hub, etsyConfig EtsyConfig) *Services {
	services := NewServices(repos, store, printerMgr, hub)
	services.Etsy = NewEtsyService(repos.Etsy, etsyConfig.ClientID, etsyConfig.RedirectURI)
	return services
}

// ProjectService handles project business logic.
type ProjectService struct {
	repo *repository.ProjectRepository
}

// Create creates a new project.
func (s *ProjectService) Create(ctx context.Context, p *model.Project) error {
	if p.Name == "" {
		return fmt.Errorf("project name is required")
	}
	return s.repo.Create(ctx, p)
}

// GetByID retrieves a project by ID.
func (s *ProjectService) GetByID(ctx context.Context, id uuid.UUID) (*model.Project, error) {
	return s.repo.GetByID(ctx, id)
}

// List retrieves all projects.
func (s *ProjectService) List(ctx context.Context, status *model.ProjectStatus) ([]model.Project, error) {
	return s.repo.List(ctx, status)
}

// Update updates a project.
func (s *ProjectService) Update(ctx context.Context, p *model.Project) error {
	return s.repo.Update(ctx, p)
}

// Delete removes a project.
func (s *ProjectService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// PartService handles part business logic.
type PartService struct {
	repo *repository.PartRepository
}

// Create creates a new part.
func (s *PartService) Create(ctx context.Context, p *model.Part) error {
	if p.Name == "" {
		return fmt.Errorf("part name is required")
	}
	if p.ProjectID == uuid.Nil {
		return fmt.Errorf("project ID is required")
	}
	return s.repo.Create(ctx, p)
}

// GetByID retrieves a part by ID.
func (s *PartService) GetByID(ctx context.Context, id uuid.UUID) (*model.Part, error) {
	return s.repo.GetByID(ctx, id)
}

// ListByProject retrieves all parts for a project.
func (s *PartService) ListByProject(ctx context.Context, projectID uuid.UUID) ([]model.Part, error) {
	return s.repo.ListByProject(ctx, projectID)
}

// Update updates a part.
func (s *PartService) Update(ctx context.Context, p *model.Part) error {
	return s.repo.Update(ctx, p)
}

// Delete removes a part.
func (s *PartService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// DesignService handles design business logic.
type DesignService struct {
	repo     *repository.DesignRepository
	fileRepo *repository.FileRepository
	storage  storage.Storage
}

// Create creates a new design version with file upload.
func (s *DesignService) Create(ctx context.Context, partID uuid.UUID, filename string, reader io.Reader, notes string) (*model.Design, error) {
	if partID == uuid.Nil {
		return nil, fmt.Errorf("part ID is required")
	}

	// Determine file type from extension
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
	var fileType model.FileType
	switch ext {
	case "stl":
		fileType = model.FileTypeSTL
	case "3mf":
		fileType = model.FileType3MF
	case "gcode":
		fileType = model.FileTypeGCODE
	default:
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}

	// Save file to storage
	storagePath, hash, size, err := s.storage.Save(filename, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	// Check for existing file with same hash (deduplication)
	existingFile, err := s.fileRepo.GetByHash(ctx, hash)
	if err != nil {
		return nil, err
	}

	var fileID uuid.UUID
	if existingFile != nil {
		fileID = existingFile.ID
		// Remove duplicate file from storage
		s.storage.Delete(storagePath)
		storagePath = existingFile.StoragePath
	} else {
		// Create new file record
		file := &model.File{
			Hash:         hash,
			OriginalName: filename,
			ContentType:  getContentType(ext),
			SizeBytes:    size,
			StoragePath:  storagePath,
		}
		if err := s.fileRepo.Create(ctx, file); err != nil {
			return nil, fmt.Errorf("failed to create file record: %w", err)
		}
		fileID = file.ID
	}

	// Create design record
	design := &model.Design{
		PartID:        partID,
		FileID:        fileID,
		FileName:      filename,
		FileHash:      hash,
		FileSizeBytes: size,
		FileType:      fileType,
		Notes:         notes,
	}
	if err := s.repo.Create(ctx, design); err != nil {
		return nil, fmt.Errorf("failed to create design: %w", err)
	}

	return design, nil
}

// GetByID retrieves a design by ID.
func (s *DesignService) GetByID(ctx context.Context, id uuid.UUID) (*model.Design, error) {
	return s.repo.GetByID(ctx, id)
}

// ListByPart retrieves all designs for a part.
func (s *DesignService) ListByPart(ctx context.Context, partID uuid.UUID) ([]model.Design, error) {
	return s.repo.ListByPart(ctx, partID)
}

// GetFile retrieves the file for a design.
func (s *DesignService) GetFile(ctx context.Context, id uuid.UUID) (io.ReadCloser, *model.Design, error) {
	design, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if design == nil {
		return nil, nil, fmt.Errorf("design not found")
	}

	file, err := s.fileRepo.GetByID(ctx, design.FileID)
	if err != nil {
		return nil, nil, err
	}
	if file == nil {
		return nil, nil, fmt.Errorf("file not found")
	}

	reader, err := s.storage.Get(file.StoragePath)
	if err != nil {
		return nil, nil, err
	}

	return reader, design, nil
}

// PrinterService handles printer business logic.
type PrinterService struct {
	repo      *repository.PrinterRepository
	manager   *printer.Manager
	hub       *realtime.Hub
	discovery *printer.Discovery
}

// Create creates a new printer.
func (s *PrinterService) Create(ctx context.Context, p *model.Printer) error {
	if p.Name == "" {
		return fmt.Errorf("printer name is required")
	}
	if err := s.repo.Create(ctx, p); err != nil {
		return err
	}

	// Connect printer if not manual
	if p.ConnectionType != model.ConnectionTypeManual {
		go s.manager.Connect(p)
	}

	return nil
}

// GetByID retrieves a printer by ID.
func (s *PrinterService) GetByID(ctx context.Context, id uuid.UUID) (*model.Printer, error) {
	return s.repo.GetByID(ctx, id)
}

// List retrieves all printers.
func (s *PrinterService) List(ctx context.Context) ([]model.Printer, error) {
	return s.repo.List(ctx)
}

// Update updates a printer.
func (s *PrinterService) Update(ctx context.Context, p *model.Printer) error {
	return s.repo.Update(ctx, p)
}

// Delete removes a printer.
func (s *PrinterService) Delete(ctx context.Context, id uuid.UUID) error {
	s.manager.Disconnect(id)
	return s.repo.Delete(ctx, id)
}

// GetState retrieves real-time state for a printer.
func (s *PrinterService) GetState(ctx context.Context, id uuid.UUID) (*model.PrinterState, error) {
	return s.manager.GetState(id)
}

// GetAllStates retrieves real-time state for all printers.
func (s *PrinterService) GetAllStates(ctx context.Context) map[uuid.UUID]*model.PrinterState {
	return s.manager.GetAllStates()
}

// DiscoverPrinters scans the network for printers.
func (s *PrinterService) DiscoverPrinters(ctx context.Context) ([]printer.DiscoveredPrinter, error) {
	discovered, err := s.discovery.QuickScan(ctx)
	if err != nil {
		return nil, err
	}

	// Mark printers that are already added
	existing, _ := s.repo.List(ctx)
	existingHosts := make(map[string]bool)
	for _, p := range existing {
		// Extract host from connection URI
		existingHosts[p.ConnectionURI] = true
	}

	for i := range discovered {
		uri := fmt.Sprintf("http://%s:%d", discovered[i].Host, discovered[i].Port)
		if existingHosts[uri] {
			discovered[i].AlreadyAdded = true
		}
	}

	return discovered, nil
}

// MaterialService handles material business logic.
type MaterialService struct {
	repo *repository.MaterialRepository
}

// Create creates a new material.
func (s *MaterialService) Create(ctx context.Context, m *model.Material) error {
	if m.Name == "" {
		return fmt.Errorf("material name is required")
	}
	return s.repo.Create(ctx, m)
}

// GetByID retrieves a material by ID.
func (s *MaterialService) GetByID(ctx context.Context, id uuid.UUID) (*model.Material, error) {
	return s.repo.GetByID(ctx, id)
}

// List retrieves all materials.
func (s *MaterialService) List(ctx context.Context) ([]model.Material, error) {
	return s.repo.List(ctx)
}

// SpoolService handles spool business logic.
type SpoolService struct {
	repo *repository.SpoolRepository
}

// Create creates a new spool.
func (s *SpoolService) Create(ctx context.Context, sp *model.MaterialSpool) error {
	if sp.MaterialID == uuid.Nil {
		return fmt.Errorf("material ID is required")
	}
	if sp.RemainingWeight == 0 {
		sp.RemainingWeight = sp.InitialWeight
	}
	return s.repo.Create(ctx, sp)
}

// GetByID retrieves a spool by ID.
func (s *SpoolService) GetByID(ctx context.Context, id uuid.UUID) (*model.MaterialSpool, error) {
	return s.repo.GetByID(ctx, id)
}

// List retrieves all spools.
func (s *SpoolService) List(ctx context.Context) ([]model.MaterialSpool, error) {
	return s.repo.List(ctx)
}

// PrintJobService handles print job business logic.
type PrintJobService struct {
	repo         *repository.PrintJobRepository
	printerRepo  *repository.PrinterRepository
	designRepo   *repository.DesignRepository
	spoolRepo    *repository.SpoolRepository
	materialRepo *repository.MaterialRepository
	printerMgr   *printer.Manager
	hub          *realtime.Hub
	storage      storage.Storage
}

// Create creates a new print job.
func (s *PrintJobService) Create(ctx context.Context, j *model.PrintJob) error {
	if j.DesignID == uuid.Nil {
		return fmt.Errorf("design ID is required")
	}
	if j.PrinterID == uuid.Nil {
		return fmt.Errorf("printer ID is required")
	}
	if j.MaterialSpoolID == uuid.Nil {
		return fmt.Errorf("material spool ID is required")
	}
	return s.repo.Create(ctx, j)
}

// GetByID retrieves a print job by ID.
func (s *PrintJobService) GetByID(ctx context.Context, id uuid.UUID) (*model.PrintJob, error) {
	return s.repo.GetByID(ctx, id)
}

// List retrieves print jobs.
func (s *PrintJobService) List(ctx context.Context, printerID *uuid.UUID, status *model.PrintJobStatus) ([]model.PrintJob, error) {
	return s.repo.List(ctx, printerID, status)
}

// ListByDesign retrieves print jobs for a design.
func (s *PrintJobService) ListByDesign(ctx context.Context, designID uuid.UUID) ([]model.PrintJob, error) {
	return s.repo.ListByDesign(ctx, designID)
}

// Start sends a print job to the printer.
func (s *PrintJobService) Start(ctx context.Context, id uuid.UUID) error {
	job, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if job == nil {
		return fmt.Errorf("job not found")
	}

	// Get design and file
	design, err := s.designRepo.GetByID(ctx, job.DesignID)
	if err != nil {
		return err
	}

	// Get printer
	printerData, err := s.printerRepo.GetByID(ctx, job.PrinterID)
	if err != nil {
		return err
	}
	if printerData == nil {
		return fmt.Errorf("printer not found")
	}

	// Send to printer via manager
	if err := s.printerMgr.StartJob(job.PrinterID, design.FileName, s.storage.GetFullPath(design.FileName)); err != nil {
		return fmt.Errorf("failed to start print: %w", err)
	}

	// Update job status
	job.Status = model.PrintJobStatusSending
	if err := s.repo.Update(ctx, job); err != nil {
		return err
	}

	// Broadcast update
	s.hub.Broadcast(realtime.Event{
		Type: "job_started",
		Data: job,
	})

	return nil
}

// Pause pauses a print job.
func (s *PrintJobService) Pause(ctx context.Context, id uuid.UUID) error {
	job, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if job == nil {
		return fmt.Errorf("job not found")
	}

	if err := s.printerMgr.PauseJob(job.PrinterID); err != nil {
		return err
	}

	return nil
}

// Resume resumes a paused print job.
func (s *PrintJobService) Resume(ctx context.Context, id uuid.UUID) error {
	job, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if job == nil {
		return fmt.Errorf("job not found")
	}

	if err := s.printerMgr.ResumeJob(job.PrinterID); err != nil {
		return err
	}

	return nil
}

// Cancel cancels a print job.
func (s *PrintJobService) Cancel(ctx context.Context, id uuid.UUID) error {
	job, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if job == nil {
		return fmt.Errorf("job not found")
	}

	if err := s.printerMgr.CancelJob(job.PrinterID); err != nil {
		return err
	}

	job.Status = model.PrintJobStatusCancelled
	return s.repo.Update(ctx, job)
}

// Update updates a print job (for outcome logging).
func (s *PrintJobService) Update(ctx context.Context, j *model.PrintJob) error {
	return s.repo.Update(ctx, j)
}

// RecordOutcome records the outcome of a completed print job.
// It updates the job status, records the outcome, and deducts material from the spool.
func (s *PrintJobService) RecordOutcome(ctx context.Context, id uuid.UUID, outcome *model.PrintOutcome) error {
	job, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}
	if job == nil {
		return fmt.Errorf("job not found")
	}

	// Get the spool and material to calculate cost
	spool, err := s.spoolRepo.GetByID(ctx, job.MaterialSpoolID)
	if err != nil {
		return fmt.Errorf("failed to get spool: %w", err)
	}
	if spool == nil {
		return fmt.Errorf("spool not found")
	}

	// Get material for cost calculation
	material, err := s.materialRepo.GetByID(ctx, spool.MaterialID)
	if err != nil {
		return fmt.Errorf("failed to get material: %w", err)
	}

	// Calculate material cost: (grams / 1000) * cost_per_kg
	if outcome.MaterialUsed > 0 && material != nil {
		outcome.MaterialCost = (outcome.MaterialUsed / 1000.0) * material.CostPerKg
	}

	// Update spool remaining weight if material was used
	if outcome.MaterialUsed > 0 {
		newWeight := spool.RemainingWeight - outcome.MaterialUsed
		if newWeight < 0 {
			newWeight = 0
		}
		spool.RemainingWeight = newWeight

		// Update spool status based on remaining weight
		if spool.RemainingWeight <= 0 {
			spool.Status = model.SpoolStatusEmpty
		} else if spool.RemainingWeight < 100 {
			spool.Status = model.SpoolStatusLow
		}

		if err := s.spoolRepo.Update(ctx, spool); err != nil {
			return fmt.Errorf("failed to update spool: %w", err)
		}
	}

	// Update job with outcome
	now := time.Now()
	job.CompletedAt = &now
	job.Status = model.PrintJobStatusCompleted
	if !outcome.Success {
		job.Status = model.PrintJobStatusFailed
	}
	job.Outcome = outcome

	if err := s.repo.Update(ctx, job); err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	// Broadcast the completed event
	eventType := "job_completed"
	if !outcome.Success {
		eventType = "job_failed"
	}
	s.hub.Broadcast(realtime.Event{
		Type: eventType,
		Data: job,
	})

	return nil
}

// FileService handles file operations.
type FileService struct {
	repo    *repository.FileRepository
	storage storage.Storage
}

// GetByID retrieves a file by ID.
func (s *FileService) GetByID(ctx context.Context, id uuid.UUID) (*model.File, error) {
	return s.repo.GetByID(ctx, id)
}

// GetReader retrieves a file reader.
func (s *FileService) GetReader(ctx context.Context, id uuid.UUID) (io.ReadCloser, *model.File, error) {
	file, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if file == nil {
		return nil, nil, fmt.Errorf("file not found")
	}

	reader, err := s.storage.Get(file.StoragePath)
	if err != nil {
		return nil, nil, err
	}

	return reader, file, nil
}

// getContentType returns MIME type for file extension.
func getContentType(ext string) string {
	switch ext {
	case "stl":
		return "model/stl"
	case "3mf":
		return "model/3mf"
	case "gcode":
		return "text/x-gcode"
	default:
		return "application/octet-stream"
	}
}

// ExpenseService handles expense and receipt processing.
type ExpenseService struct {
	repo         *repository.ExpenseRepository
	materialRepo *repository.MaterialRepository
	spoolRepo    *repository.SpoolRepository
	fileRepo     *repository.FileRepository
	storage      storage.Storage
	parser       *receipt.Parser
}

// UploadReceipt uploads a receipt file and starts AI parsing.
func (s *ExpenseService) UploadReceipt(ctx context.Context, filename string, data []byte) (*model.Expense, error) {
	// Initialize parser lazily
	if s.parser == nil {
		s.parser = receipt.NewParser()
	}

	// Store the file using Save with a bytes reader
	reader := bytes.NewReader(data)
	storagePath, _, _, err := s.storage.Save(filename, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to store receipt: %w", err)
	}

	// Create initial expense record
	expense := &model.Expense{
		OccurredAt:      time.Now(),
		Status:          model.ExpenseStatusPending,
		ReceiptFilePath: storagePath,
		Category:        model.ExpenseCategoryOther,
	}

	if err := s.repo.Create(ctx, expense); err != nil {
		return nil, fmt.Errorf("failed to create expense: %w", err)
	}

	// Parse receipt asynchronously (or synchronously for now)
	go func() {
		parseCtx := context.Background()
		s.parseReceiptAsync(parseCtx, expense.ID, storagePath, data)
	}()

	return expense, nil
}

// parseReceiptAsync parses a receipt and updates the expense record.
func (s *ExpenseService) parseReceiptAsync(ctx context.Context, expenseID uuid.UUID, filePath string, data []byte) {
	slog.Info("starting receipt parsing", "expense_id", expenseID)

	// Detect content type
	contentType := "image/jpeg"
	if len(data) >= 4 {
		if data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G' {
			contentType = "image/png"
		} else if data[0] == '%' && data[1] == 'P' && data[2] == 'D' && data[3] == 'F' {
			contentType = "application/pdf"
		}
	}

	parsed, err := s.parser.ParseFromBytes(ctx, data, contentType)
	if err != nil {
		slog.Error("failed to parse receipt", "expense_id", expenseID, "error", err)
		return
	}

	slog.Info("receipt parsed successfully", "expense_id", expenseID, "vendor", parsed.Vendor, "total_cents", parsed.TotalCents)

	// Update expense with parsed data
	expense, err := s.repo.GetByID(ctx, expenseID)
	if err != nil || expense == nil {
		slog.Error("failed to get expense for update", "expense_id", expenseID, "error", err)
		return
	}

	expense.Vendor = parsed.Vendor
	expense.SubtotalCents = parsed.SubtotalCents
	expense.TaxCents = parsed.TaxCents
	expense.ShippingCents = parsed.ShippingCents
	expense.TotalCents = parsed.TotalCents
	expense.Currency = parsed.Currency
	expense.Confidence = parsed.Confidence
	expense.RawOCRText = parsed.RawText

	// Determine primary category based on items
	hasFilament := false
	for _, item := range parsed.Items {
		if item.IsFilament {
			hasFilament = true
			break
		}
	}
	if hasFilament {
		expense.Category = model.ExpenseCategoryFilament
	}

	// Store raw AI response
	rawJSON, _ := json.Marshal(parsed)
	expense.RawAIResponse = rawJSON

	// Parse date
	if parsed.Date != "" {
		if t, err := time.Parse("2006-01-02", parsed.Date); err == nil {
			expense.OccurredAt = t
		}
	}

	if err := s.repo.Update(ctx, expense); err != nil {
		slog.Error("failed to update expense", "expense_id", expenseID, "error", err)
		return
	}

	// Create expense items
	for _, item := range parsed.Items {
		expenseItem := &model.ExpenseItem{
			ExpenseID:       expenseID,
			Description:     item.Description,
			Quantity:        item.Quantity,
			UnitPriceCents:  item.UnitPriceCents,
			TotalPriceCents: item.TotalPriceCents,
			Category:        item.Category,
			Confidence:      item.Confidence,
		}

		if item.IsFilament && item.Filament != nil {
			expenseItem.Metadata = item.Filament
		}

		if err := s.repo.CreateItem(ctx, expenseItem); err != nil {
			slog.Error("failed to create expense item", "expense_id", expenseID, "error", err)
		}
	}

	slog.Info("expense items created", "expense_id", expenseID, "count", len(parsed.Items))
}

// GetByID retrieves an expense by ID with its items.
func (s *ExpenseService) GetByID(ctx context.Context, id uuid.UUID) (*model.Expense, error) {
	expense, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if expense == nil {
		return nil, nil
	}

	// Load items
	items, err := s.repo.GetItemsByExpenseID(ctx, id)
	if err != nil {
		return nil, err
	}
	expense.Items = items

	return expense, nil
}

// List retrieves all expenses.
func (s *ExpenseService) List(ctx context.Context, status *model.ExpenseStatus) ([]model.Expense, error) {
	return s.repo.List(ctx, status)
}

// ConfirmExpenseRequest contains the data to confirm an expense.
type ConfirmExpenseRequest struct {
	Items []ConfirmExpenseItem `json:"items"`
}

// ConfirmExpenseItem contains the user's decisions for each expense item.
type ConfirmExpenseItem struct {
	ItemID       uuid.UUID `json:"item_id"`
	CreateSpool  bool      `json:"create_spool"`
	MaterialID   *uuid.UUID `json:"material_id,omitempty"` // Use existing material
	NewMaterial  *model.Material `json:"new_material,omitempty"` // Create new material
	WeightGrams  float64   `json:"weight_grams,omitempty"`
	DiameterMM   float64   `json:"diameter_mm,omitempty"`
}

// ConfirmExpense confirms an expense and applies inventory changes.
func (s *ExpenseService) ConfirmExpense(ctx context.Context, id uuid.UUID, req *ConfirmExpenseRequest) error {
	expense, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if expense == nil {
		return fmt.Errorf("expense not found")
	}

	if expense.Status != model.ExpenseStatusPending {
		return fmt.Errorf("expense is not pending")
	}

	// Process each item
	for _, confirmItem := range req.Items {
		// Find the expense item
		var expenseItem *model.ExpenseItem
		for i := range expense.Items {
			if expense.Items[i].ID == confirmItem.ItemID {
				expenseItem = &expense.Items[i]
				break
			}
		}
		if expenseItem == nil {
			continue
		}

		if confirmItem.CreateSpool {
			var materialID uuid.UUID

			// Create new material if specified
			if confirmItem.NewMaterial != nil {
				if err := s.materialRepo.Create(ctx, confirmItem.NewMaterial); err != nil {
					return fmt.Errorf("failed to create material: %w", err)
				}
				materialID = confirmItem.NewMaterial.ID
			} else if confirmItem.MaterialID != nil {
				materialID = *confirmItem.MaterialID
			} else {
				continue // No material specified, skip
			}

			// Determine weight
			weightGrams := confirmItem.WeightGrams
			if weightGrams == 0 && expenseItem.Metadata != nil {
				weightGrams = expenseItem.Metadata.WeightGrams
			}
			if weightGrams == 0 {
				weightGrams = 1000 // Default 1kg
			}

			// Create spools (one per quantity)
			quantity := int(expenseItem.Quantity)
			if quantity < 1 {
				quantity = 1
			}

			for i := 0; i < quantity; i++ {
				spool := &model.MaterialSpool{
					MaterialID:      materialID,
					InitialWeight:   weightGrams,
					RemainingWeight: weightGrams,
					PurchaseDate:    &expense.OccurredAt,
					PurchaseCost:    float64(expenseItem.TotalPriceCents) / 100.0 / float64(quantity),
					Status:          model.SpoolStatusNew,
					Notes:           fmt.Sprintf("From receipt: %s", expense.Vendor),
				}

				if err := s.spoolRepo.Create(ctx, spool); err != nil {
					return fmt.Errorf("failed to create spool: %w", err)
				}

				// Update expense item with matched spool
				if i == 0 {
					expenseItem.MatchedSpoolID = &spool.ID
					expenseItem.MatchedMaterialID = &materialID
					expenseItem.ActionTaken = model.ExpenseItemActionCreatedSpool
				}
			}

			if err := s.repo.UpdateItem(ctx, expenseItem); err != nil {
				return fmt.Errorf("failed to update expense item: %w", err)
			}
		} else {
			expenseItem.ActionTaken = model.ExpenseItemActionSkipped
			if err := s.repo.UpdateItem(ctx, expenseItem); err != nil {
				return fmt.Errorf("failed to update expense item: %w", err)
			}
		}
	}

	// Mark expense as confirmed
	expense.Status = model.ExpenseStatusConfirmed
	if err := s.repo.Update(ctx, expense); err != nil {
		return fmt.Errorf("failed to confirm expense: %w", err)
	}

	return nil
}

// Delete deletes an expense.
func (s *ExpenseService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// SaleService handles sales and revenue tracking.
type SaleService struct {
	repo *repository.SaleRepository
}

// Create creates a new sale.
func (s *SaleService) Create(ctx context.Context, sale *model.Sale) error {
	// Calculate net if not provided
	if sale.NetCents == 0 {
		sale.NetCents = sale.GrossCents - sale.FeesCents - sale.ShippingCostCents
	}
	return s.repo.Create(ctx, sale)
}

// GetByID retrieves a sale by ID.
func (s *SaleService) GetByID(ctx context.Context, id uuid.UUID) (*model.Sale, error) {
	return s.repo.GetByID(ctx, id)
}

// List retrieves all sales.
func (s *SaleService) List(ctx context.Context, projectID *uuid.UUID) ([]model.Sale, error) {
	return s.repo.List(ctx, projectID)
}

// Update updates a sale.
func (s *SaleService) Update(ctx context.Context, sale *model.Sale) error {
	// Recalculate net
	sale.NetCents = sale.GrossCents - sale.FeesCents - sale.ShippingCostCents
	return s.repo.Update(ctx, sale)
}

// Delete deletes a sale.
func (s *SaleService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// GetProfitSummary calculates profit for a date range.
func (s *SaleService) GetProfitSummary(ctx context.Context, start, end time.Time) (grossCents, netCents, feesCents int, count int, err error) {
	return s.repo.GetTotalsByDateRange(ctx, start, end)
}

// FinancialSummary contains aggregated financial data.
type FinancialSummary struct {
	TotalExpensesCents     int     `json:"total_expenses_cents"`
	TotalSalesGrossCents   int     `json:"total_sales_gross_cents"`
	TotalSalesNetCents     int     `json:"total_sales_net_cents"`
	TotalFeesCents         int     `json:"total_fees_cents"`
	TotalMaterialCost      float64 `json:"total_material_cost"`
	TotalMaterialUsedGrams float64 `json:"total_material_used_grams"`
	NetProfitCents         int     `json:"net_profit_cents"`
	ConfirmedExpenseCount  int     `json:"confirmed_expense_count"`
	PendingExpenseCount    int     `json:"pending_expense_count"`
	SalesCount             int     `json:"sales_count"`
	CompletedPrintCount    int     `json:"completed_print_count"`
	SuccessfulPrintCount   int     `json:"successful_print_count"`
}

// StatsService handles financial statistics and aggregations.
type StatsService struct {
	expenseRepo  *repository.ExpenseRepository
	saleRepo     *repository.SaleRepository
	printJobRepo *repository.PrintJobRepository
}

// GetFinancialSummary returns aggregated financial data.
func (s *StatsService) GetFinancialSummary(ctx context.Context) (*FinancialSummary, error) {
	summary := &FinancialSummary{}

	// Get expense totals
	confirmedStatus := model.ExpenseStatusConfirmed
	expenses, err := s.expenseRepo.List(ctx, &confirmedStatus)
	if err != nil {
		return nil, fmt.Errorf("failed to get expenses: %w", err)
	}
	for _, exp := range expenses {
		summary.TotalExpensesCents += exp.TotalCents
		summary.ConfirmedExpenseCount++
	}

	// Get pending expense count
	pendingStatus := model.ExpenseStatusPending
	pendingExpenses, err := s.expenseRepo.List(ctx, &pendingStatus)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending expenses: %w", err)
	}
	summary.PendingExpenseCount = len(pendingExpenses)

	// Get sales totals
	sales, err := s.saleRepo.List(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get sales: %w", err)
	}
	for _, sale := range sales {
		summary.TotalSalesGrossCents += sale.GrossCents
		summary.TotalSalesNetCents += sale.NetCents
		summary.TotalFeesCents += sale.FeesCents
		summary.SalesCount++
	}

	// Get print job stats (completed jobs with outcomes)
	completedStatus := model.PrintJobStatusCompleted
	jobs, err := s.printJobRepo.List(ctx, nil, &completedStatus)
	if err != nil {
		return nil, fmt.Errorf("failed to get print jobs: %w", err)
	}
	for _, job := range jobs {
		summary.CompletedPrintCount++
		if job.Outcome != nil {
			summary.TotalMaterialUsedGrams += job.Outcome.MaterialUsed
			summary.TotalMaterialCost += job.Outcome.MaterialCost
			if job.Outcome.Success {
				summary.SuccessfulPrintCount++
			}
		}
	}

	// Calculate net profit: sales net - expenses - material cost (already in spool cost)
	materialCostCents := int(summary.TotalMaterialCost * 100)
	summary.NetProfitCents = summary.TotalSalesNetCents - summary.TotalExpensesCents - materialCostCents

	return summary, nil
}

// TemplateService handles template business logic.
type TemplateService struct {
	repo         *repository.TemplateRepository
	projectRepo  *repository.ProjectRepository
	partRepo     *repository.PartRepository
	designRepo   *repository.DesignRepository
	printJobRepo *repository.PrintJobRepository
	spoolRepo    *repository.SpoolRepository
	materialRepo *repository.MaterialRepository
	printerRepo  *repository.PrinterRepository
}

// Create creates a new template.
func (s *TemplateService) Create(ctx context.Context, t *model.Template) error {
	if t.Name == "" {
		return fmt.Errorf("template name is required")
	}
	if t.MaterialType == "" {
		return fmt.Errorf("material type is required")
	}
	t.IsActive = true
	return s.repo.Create(ctx, t)
}

// GetByID retrieves a template by ID with its designs.
func (s *TemplateService) GetByID(ctx context.Context, id uuid.UUID) (*model.Template, error) {
	t, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, nil
	}

	// Load designs
	designs, err := s.repo.GetDesigns(ctx, id)
	if err != nil {
		return nil, err
	}
	t.Designs = designs

	return t, nil
}

// GetBySKU retrieves a template by SKU.
func (s *TemplateService) GetBySKU(ctx context.Context, sku string) (*model.Template, error) {
	return s.repo.GetBySKU(ctx, sku)
}

// List retrieves all templates.
func (s *TemplateService) List(ctx context.Context, activeOnly bool) ([]model.Template, error) {
	return s.repo.List(ctx, activeOnly)
}

// Update updates a template.
func (s *TemplateService) Update(ctx context.Context, t *model.Template) error {
	return s.repo.Update(ctx, t)
}

// Delete removes a template.
func (s *TemplateService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// AddDesign adds a design to a template.
func (s *TemplateService) AddDesign(ctx context.Context, td *model.TemplateDesign) error {
	if td.TemplateID == uuid.Nil {
		return fmt.Errorf("template ID is required")
	}
	if td.DesignID == uuid.Nil {
		return fmt.Errorf("design ID is required")
	}

	// Verify design exists
	design, err := s.designRepo.GetByID(ctx, td.DesignID)
	if err != nil {
		return err
	}
	if design == nil {
		return fmt.Errorf("design not found")
	}

	return s.repo.AddDesign(ctx, td)
}

// RemoveDesign removes a design from a template.
func (s *TemplateService) RemoveDesign(ctx context.Context, templateID, designID uuid.UUID) error {
	return s.repo.RemoveDesign(ctx, templateID, designID)
}

// GetDesigns retrieves all designs for a template.
func (s *TemplateService) GetDesigns(ctx context.Context, templateID uuid.UUID) ([]model.TemplateDesign, error) {
	return s.repo.GetDesigns(ctx, templateID)
}

// CreateFromTemplateOptions contains options for creating a project from a template.
type CreateFromTemplateOptions struct {
	OrderQuantity   int        // Multiplier for quantity_per_order
	ExternalOrderID string     // For Etsy orders later
	CustomerNotes   string
	Source          string     // "manual", "etsy", "api"
	MaterialSpoolID *uuid.UUID // Optional spool override
}

// CreateProjectFromTemplate creates a new project from a template.
func (s *TemplateService) CreateProjectFromTemplate(ctx context.Context, templateID uuid.UUID, opts CreateFromTemplateOptions) (*model.Project, []model.PrintJob, error) {
	// Fetch template with designs
	template, err := s.GetByID(ctx, templateID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get template: %w", err)
	}
	if template == nil {
		return nil, nil, fmt.Errorf("template not found")
	}

	if !template.IsActive {
		return nil, nil, fmt.Errorf("template is not active")
	}

	// Set defaults
	if opts.OrderQuantity <= 0 {
		opts.OrderQuantity = 1
	}
	if opts.Source == "" {
		opts.Source = "manual"
	}

	// Create project
	project := &model.Project{
		Name:            template.Name,
		Description:     template.Description,
		Status:          model.ProjectStatusActive,
		Tags:            template.Tags,
		TemplateID:      &templateID,
		Source:          opts.Source,
		ExternalOrderID: opts.ExternalOrderID,
		CustomerNotes:   opts.CustomerNotes,
	}
	if err := s.projectRepo.Create(ctx, project); err != nil {
		return nil, nil, fmt.Errorf("failed to create project: %w", err)
	}

	var printJobs []model.PrintJob

	// For each template design, create parts and print jobs
	totalParts := template.QuantityPerOrder * opts.OrderQuantity
	for _, td := range template.Designs {
		for i := 0; i < totalParts; i++ {
			// Create part for this design
			partName := td.Design.FileName
			if td.Notes != "" {
				partName = td.Notes
			}

			part := &model.Part{
				ProjectID:   project.ID,
				Name:        partName,
				Description: fmt.Sprintf("From template: %s", template.Name),
				Quantity:    td.Quantity,
				Status:      model.PartStatusDesign,
			}
			if err := s.partRepo.Create(ctx, part); err != nil {
				return nil, nil, fmt.Errorf("failed to create part: %w", err)
			}

			// Create print job for each part quantity
			for j := 0; j < td.Quantity; j++ {
				job := &model.PrintJob{
					DesignID: td.DesignID,
					Status:   model.PrintJobStatusQueued,
					Notes:    fmt.Sprintf("Part %d/%d for order", i+1, totalParts),
				}

				// Assign preferred printer if specified and not allowing any printer
				if template.PreferredPrinterID != nil && !template.AllowAnyPrinter {
					job.PrinterID = *template.PreferredPrinterID
				}

				// Assign material spool if specified in options
				if opts.MaterialSpoolID != nil {
					job.MaterialSpoolID = *opts.MaterialSpoolID
				}

				if err := s.printJobRepo.Create(ctx, job); err != nil {
					return nil, nil, fmt.Errorf("failed to create print job: %w", err)
				}
				printJobs = append(printJobs, *job)
			}
		}
	}

	return project, printJobs, nil
}

// GetByIDWithMaterials retrieves a template with its materials loaded.
func (s *TemplateService) GetByIDWithMaterials(ctx context.Context, id uuid.UUID) (*model.Template, error) {
	t, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, nil
	}

	// Load materials
	materials, err := s.repo.GetRecipeMaterials(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("loading materials: %w", err)
	}
	t.Materials = materials

	return t, nil
}

// AddMaterial adds a material requirement to a recipe.
func (s *TemplateService) AddMaterial(ctx context.Context, m *model.RecipeMaterial) error {
	if m.RecipeID == uuid.Nil {
		return fmt.Errorf("recipe ID is required")
	}
	if m.MaterialType == "" {
		return fmt.Errorf("material type is required")
	}
	return s.repo.CreateRecipeMaterial(ctx, m)
}

// UpdateMaterial updates a material requirement.
func (s *TemplateService) UpdateMaterial(ctx context.Context, m *model.RecipeMaterial) error {
	return s.repo.UpdateRecipeMaterial(ctx, m)
}

// RemoveMaterial removes a material requirement from a recipe.
func (s *TemplateService) RemoveMaterial(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteRecipeMaterial(ctx, id)
}

// GetMaterial retrieves a single material by ID.
func (s *TemplateService) GetMaterial(ctx context.Context, id uuid.UUID) (*model.RecipeMaterial, error) {
	return s.repo.GetRecipeMaterialByID(ctx, id)
}

// ListMaterials retrieves all materials for a recipe.
func (s *TemplateService) ListMaterials(ctx context.Context, recipeID uuid.UUID) ([]model.RecipeMaterial, error) {
	return s.repo.GetRecipeMaterials(ctx, recipeID)
}

// PrinterValidationResult contains the result of printer validation.
type PrinterValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// ValidatePrinterForRecipe checks if a printer meets recipe constraints.
func (s *TemplateService) ValidatePrinterForRecipe(ctx context.Context, recipeID, printerID uuid.UUID) (*PrinterValidationResult, error) {
	template, err := s.repo.GetByID(ctx, recipeID)
	if err != nil {
		return nil, err
	}
	if template == nil {
		return nil, fmt.Errorf("recipe not found")
	}

	printer, err := s.printerRepo.GetByID(ctx, printerID)
	if err != nil {
		return nil, err
	}
	if printer == nil {
		return nil, fmt.Errorf("printer not found")
	}

	result := &PrinterValidationResult{Valid: true}

	if template.PrinterConstraints == nil {
		return result, nil
	}

	constraints := template.PrinterConstraints

	// Check bed size
	if constraints.MinBedSize != nil {
		if printer.BuildVolume == nil {
			result.Warnings = append(result.Warnings, "Printer build volume not configured")
		} else {
			if printer.BuildVolume.X < constraints.MinBedSize.X {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("X axis too small: need %.0fmm, have %.0fmm", constraints.MinBedSize.X, printer.BuildVolume.X))
			}
			if printer.BuildVolume.Y < constraints.MinBedSize.Y {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("Y axis too small: need %.0fmm, have %.0fmm", constraints.MinBedSize.Y, printer.BuildVolume.Y))
			}
			if printer.BuildVolume.Z < constraints.MinBedSize.Z {
				result.Valid = false
				result.Errors = append(result.Errors, fmt.Sprintf("Z axis too small: need %.0fmm, have %.0fmm", constraints.MinBedSize.Z, printer.BuildVolume.Z))
			}
		}
	}

	// Check nozzle diameter
	if len(constraints.NozzleDiameters) > 0 {
		found := false
		for _, d := range constraints.NozzleDiameters {
			if printer.NozzleDiameter == d {
				found = true
				break
			}
		}
		if !found {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Incompatible nozzle: need one of %v mm, have %.2f mm", constraints.NozzleDiameters, printer.NozzleDiameter))
		}
	}

	// Check enclosure requirement
	if constraints.RequiresEnclosure {
		result.Warnings = append(result.Warnings, "Recipe requires enclosure - verify printer has enclosure")
	}

	// Check AMS requirement
	if constraints.RequiresAMS {
		result.Warnings = append(result.Warnings, "Recipe requires AMS - verify printer has AMS configured")
	}

	return result, nil
}

// FindCompatiblePrinters returns printers that match recipe constraints.
func (s *TemplateService) FindCompatiblePrinters(ctx context.Context, recipeID uuid.UUID) ([]model.Printer, error) {
	return s.repo.FindCompatiblePrinters(ctx, recipeID)
}

// CompatibleSpool represents a spool that matches recipe requirements.
type CompatibleSpool struct {
	Spool       model.MaterialSpool `json:"spool"`
	Material    model.Material      `json:"material"`
	MatchReason string              `json:"match_reason"`
}

// FindCompatibleSpools finds spools matching recipe material requirements.
func (s *TemplateService) FindCompatibleSpools(ctx context.Context, recipeID uuid.UUID) ([]CompatibleSpool, error) {
	materials, err := s.repo.GetRecipeMaterials(ctx, recipeID)
	if err != nil {
		return nil, err
	}

	// Get all available spools
	spools, err := s.spoolRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	// Get all materials for lookup
	allMaterials, err := s.materialRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	materialMap := make(map[uuid.UUID]model.Material)
	for _, m := range allMaterials {
		materialMap[m.ID] = m
	}

	var compatible []CompatibleSpool

	for _, rm := range materials {
		for _, spool := range spools {
			// Skip empty or archived spools
			if spool.Status == model.SpoolStatusEmpty || spool.Status == model.SpoolStatusArchived {
				continue
			}

			// Check if spool has enough material
			if spool.RemainingWeight < rm.WeightGrams {
				continue
			}

			material, ok := materialMap[spool.MaterialID]
			if !ok {
				continue
			}

			// Check material type match
			if material.Type != rm.MaterialType {
				continue
			}

			// Check color match based on color spec mode
			matchReason := fmt.Sprintf("Type match: %s", rm.MaterialType)

			if rm.ColorSpec != nil {
				switch rm.ColorSpec.Mode {
				case "exact":
					if rm.ColorSpec.Hex != "" && material.ColorHex != rm.ColorSpec.Hex {
						continue
					}
					if rm.ColorSpec.Name != "" && material.Color != rm.ColorSpec.Name {
						continue
					}
					matchReason += fmt.Sprintf(", Color: %s", material.Color)
				case "category":
					// Allow any color in the same category (would need color categorization)
					matchReason += fmt.Sprintf(", Color (any): %s", material.Color)
				case "any":
					matchReason += " (any color)"
				}
			}

			compatible = append(compatible, CompatibleSpool{
				Spool:       spool,
				Material:    material,
				MatchReason: matchReason,
			})
		}
	}

	return compatible, nil
}

// DefaultHourlyRateCents is the default machine time cost per hour in cents.
const DefaultHourlyRateCents = 500 // $5.00/hour

// CalculateRecipeCost calculates the cost breakdown for a recipe.
func (s *TemplateService) CalculateRecipeCost(ctx context.Context, recipeID uuid.UUID) (*model.RecipeCostEstimate, error) {
	template, err := s.GetByIDWithMaterials(ctx, recipeID)
	if err != nil {
		return nil, err
	}
	if template == nil {
		return nil, fmt.Errorf("recipe not found")
	}

	// Get all materials for cost lookup
	allMaterials, err := s.materialRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	materialMap := make(map[model.MaterialType]model.Material)
	for _, m := range allMaterials {
		materialMap[m.Type] = m
	}

	estimate := &model.RecipeCostEstimate{
		EstimatedPrintTime: template.EstimatedPrintSeconds,
		HourlyRateCents:    DefaultHourlyRateCents,
	}

	// Calculate material costs
	for _, rm := range template.Materials {
		material, ok := materialMap[rm.MaterialType]
		if !ok {
			// Use a default cost if material not found
			material = model.Material{CostPerKg: 25.0} // $25/kg default
		}

		// Cost = (weight_grams / 1000) * cost_per_kg * 100 (to cents)
		costCents := int((rm.WeightGrams / 1000.0) * material.CostPerKg * 100)
		estimate.MaterialCostCents += costCents

		colorName := ""
		if rm.ColorSpec != nil {
			colorName = rm.ColorSpec.Name
		}

		estimate.MaterialBreakdown = append(estimate.MaterialBreakdown, model.RecipeMaterialCostBreakdown{
			MaterialType: string(rm.MaterialType),
			WeightGrams:  rm.WeightGrams,
			CostCents:    costCents,
			ColorName:    colorName,
		})
	}

	// If no materials defined but we have estimated grams, use that
	if len(template.Materials) == 0 && template.EstimatedMaterialGrams > 0 {
		material, ok := materialMap[template.MaterialType]
		if !ok {
			material = model.Material{CostPerKg: 25.0}
		}
		costCents := int((template.EstimatedMaterialGrams / 1000.0) * material.CostPerKg * 100)
		estimate.MaterialCostCents = costCents
		estimate.MaterialBreakdown = append(estimate.MaterialBreakdown, model.RecipeMaterialCostBreakdown{
			MaterialType: string(template.MaterialType),
			WeightGrams:  template.EstimatedMaterialGrams,
			CostCents:    costCents,
		})
	}

	// Calculate time cost
	if template.EstimatedPrintSeconds > 0 {
		hours := float64(template.EstimatedPrintSeconds) / 3600.0
		estimate.TimeCostCents = int(hours * float64(DefaultHourlyRateCents))
	}

	estimate.TotalCostCents = estimate.MaterialCostCents + estimate.TimeCostCents

	return estimate, nil
}

