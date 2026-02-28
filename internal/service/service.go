package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hyperion/printfarm/internal/bambu"
	"github.com/hyperion/printfarm/internal/crypto"
	"github.com/hyperion/printfarm/internal/model"
	"github.com/hyperion/printfarm/internal/printer"
	"github.com/hyperion/printfarm/internal/realtime"
	"github.com/hyperion/printfarm/internal/receipt"
	"github.com/hyperion/printfarm/internal/repository"
	"github.com/hyperion/printfarm/internal/storage"
	"github.com/hyperion/printfarm/internal/threemf"
)

// Services holds all service instances.
type Services struct {
	Projects        *ProjectService
	Parts           *PartService
	Designs         *DesignService
	Printers        *PrinterService
	Materials       *MaterialService
	Spools          *SpoolService
	PrintJobs       *PrintJobService
	Files           *FileService
	Expenses        *ExpenseService
	Sales           *SaleService
	Stats           *StatsService
	Templates       *TemplateService
	Etsy            *EtsyService
	Squarespace     *SquarespaceService
	BambuCloud      *BambuCloudService
	Settings        *SettingsService
	ProjectSupplies *ProjectSupplyService
	Backup          *BackupService
	Dispatcher      *DispatcherService
	// New services for feature gaps
	Orders   *OrderService
	Alerts   *AlertService
	Tags     *TagService
	Shopify  *ShopifyService
	Timeline *TimelineService
	Tasks    *TaskService
	Feedback *FeedbackService
}

// EtsyConfig holds Etsy OAuth configuration.
type EtsyConfig struct {
	ClientID    string
	RedirectURI string
}

// ServicesConfig holds all service configuration.
type ServicesConfig struct {
	Etsy EtsyConfig
}

// NewServices creates all service instances.
func NewServices(repos *repository.Repositories, store storage.Storage, printerMgr *printer.Manager, hub *realtime.Hub) *Services {
	// Create shared Bambu cloud client
	bambuCloudClient := bambu.NewCloudClient()

	services := &Services{
		Projects:  &ProjectService{repo: repos.Projects, printJobRepo: repos.PrintJobs, printerRepo: repos.Printers, spoolRepo: repos.Spools, templateRepo: repos.Templates, designRepo: repos.Designs, saleRepo: repos.Sales, partRepo: repos.Parts, supplyRepo: repos.ProjectSupplies, printerMgr: printerMgr, hub: hub, storage: store},
		Parts:     &PartService{repo: repos.Parts},
		Designs:   &DesignService{repo: repos.Designs, fileRepo: repos.Files, storage: store},
		Printers:  &PrinterService{repo: repos.Printers, printJobRepo: repos.PrintJobs, saleRepo: repos.Sales, manager: printerMgr, hub: hub, discovery: printer.NewDiscovery(), bambuCloudRepo: repos.BambuCloud, bambuCloud: bambuCloudClient},
		Materials: &MaterialService{repo: repos.Materials},
		Spools:    &SpoolService{repo: repos.Spools},
		PrintJobs: &PrintJobService{repo: repos.PrintJobs, printerRepo: repos.Printers, designRepo: repos.Designs, spoolRepo: repos.Spools, materialRepo: repos.Materials, projectRepo: repos.Projects, printerMgr: printerMgr, hub: hub, storage: store},
		Files:     &FileService{repo: repos.Files, storage: store},
		Expenses:  &ExpenseService{repo: repos.Expenses, materialRepo: repos.Materials, spoolRepo: repos.Spools, fileRepo: repos.Files, settingsRepo: repos.Settings, repos: repos, storage: store},
		Sales:     &SaleService{repo: repos.Sales, taskRepo: repos.Tasks},
		Stats:     &StatsService{expenseRepo: repos.Expenses, saleRepo: repos.Sales, printJobRepo: repos.PrintJobs},
		Templates: &TemplateService{repo: repos.Templates, projectRepo: repos.Projects, partRepo: repos.Parts, designRepo: repos.Designs, printJobRepo: repos.PrintJobs, spoolRepo: repos.Spools, materialRepo: repos.Materials, printerRepo: repos.Printers},
		Etsy:            nil, // Initialize separately with config
		Squarespace:     nil, // Initialize below after Templates is ready
		BambuCloud:      NewBambuCloudService(repos.BambuCloud, repos.Printers, printerMgr, bambuCloudClient),
		Settings:        &SettingsService{repo: repos.Settings},
		ProjectSupplies: &ProjectSupplyService{repo: repos.ProjectSupplies, materialRepo: repos.Materials},
	}
	// Wire cross-service dependencies
	services.Stats.projectService = services.Projects
	services.Templates.projectService = services.Projects
	// Initialize Squarespace with template service for order processing
	services.Squarespace = NewSquarespaceService(repos.Squarespace, services.Templates)
	// Initialize Dispatcher service
	services.Dispatcher = NewDispatcherService(
		repos.Dispatch,
		repos.AutoDispatchSettings,
		repos.PrintJobs,
		repos.Printers,
		services.Templates,
		services.PrintJobs,
		printerMgr,
		hub,
		services.Settings,
	)
	services.Dispatcher.Init()

	// Initialize new services for feature gaps
	services.Orders = NewOrderService(repos.Orders, repos.Projects, repos.PrintJobs, services.Templates, hub)
	services.Alerts = NewAlertService(repos.Spools, repos.Materials, repos.Orders, repos.AlertDismissals, hub)
	services.Tags = NewTagService(repos.Tags, repos.Parts, repos.Designs)
	services.Shopify = NewShopifyService(repos.Shopify, services.Orders, services.Templates, hub)
	services.Timeline = NewTimelineService(repos.Orders, repos.Tasks, repos.Projects, repos.PrintJobs)
	services.Tasks = NewTaskService(repos.Tasks, repos.Projects, repos.PrintJobs, repos.Parts, repos.TaskChecklist, repos.Designs, hub)
	services.Feedback = &FeedbackService{repo: repos.Feedback}

	// Wire job completion callback to auto-complete checklist items
	services.PrintJobs.SetOnJobCompleted(services.Tasks.HandleJobCompleted)

	// Wire task repo to order service (needed for ProcessItem)
	services.Orders.SetTaskRepo(repos.Tasks)

	return services
}

// NewServicesWithEtsy creates all service instances including Etsy integration.
func NewServicesWithEtsy(repos *repository.Repositories, store storage.Storage, printerMgr *printer.Manager, hub *realtime.Hub, etsyConfig EtsyConfig) *Services {
	services := NewServices(repos, store, printerMgr, hub)
	services.Etsy = NewEtsyService(repos.Etsy, etsyConfig.ClientID, etsyConfig.RedirectURI, services.Settings)
	return services
}

// NewServicesWithConfig creates all service instances with full configuration.
func NewServicesWithConfig(repos *repository.Repositories, store storage.Storage, printerMgr *printer.Manager, hub *realtime.Hub, config ServicesConfig) *Services {
	services := NewServices(repos, store, printerMgr, hub)
	services.Etsy = NewEtsyService(repos.Etsy, config.Etsy.ClientID, config.Etsy.RedirectURI, services.Settings)
	return services
}

// SetBackupService sets the backup service (must be called after DB is available).
func (s *Services) SetBackupService(backup *BackupService) {
	s.Backup = backup
}

// ProjectService handles project business logic.
type ProjectService struct {
	repo            *repository.ProjectRepository
	printJobRepo    *repository.PrintJobRepository
	printerRepo     *repository.PrinterRepository
	spoolRepo       *repository.SpoolRepository
	templateRepo    *repository.TemplateRepository
	designRepo      *repository.DesignRepository
	saleRepo        *repository.SaleRepository
	partRepo        *repository.PartRepository
	supplyRepo      *repository.ProjectSupplyRepository
	printerMgr      *printer.Manager
	hub             *realtime.Hub
	storage         storage.Storage
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
func (s *ProjectService) List(ctx context.Context) ([]model.Project, error) {
	return s.repo.List(ctx)
}

// Update updates a project.
func (s *ProjectService) Update(ctx context.Context, p *model.Project) error {
	return s.repo.Update(ctx, p)
}

// Delete removes a project.
func (s *ProjectService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// GetJobStats retrieves job statistics for a project.
func (s *ProjectService) GetJobStats(ctx context.Context, projectID uuid.UUID) (*repository.JobStats, error) {
	return s.printJobRepo.GetProjectJobStats(ctx, projectID)
}

// ListJobs retrieves all print jobs for a project.
func (s *ProjectService) ListJobs(ctx context.Context, projectID uuid.UUID) ([]model.PrintJob, error) {
	return s.printJobRepo.ListByProject(ctx, projectID)
}

// GetProjectSummary computes a derived analytics summary for a project.
// All values are computed from jobs and sales — nothing is stored.
func (s *ProjectService) GetProjectSummary(ctx context.Context, projectID uuid.UUID) (*model.ProjectSummary, error) {
	// Fetch jobs
	jobs, err := s.printJobRepo.ListByProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Fetch sales
	sales, err := s.saleRepo.List(ctx, &projectID)
	if err != nil {
		return nil, err
	}

	summary := &model.ProjectSummary{}

	// Revenue from sales
	for _, sale := range sales {
		summary.TotalRevenueCents += sale.GrossCents
		summary.TotalFeesCents += sale.FeesCents
		summary.NetRevenueCents += sale.NetCents
		summary.SalesCount++
	}

	// Cost and performance from jobs
	summary.JobCount = len(jobs)
	for _, job := range jobs {
		if job.Status == model.PrintJobStatusCompleted {
			summary.CompletedCount++
		}
		if job.Status == model.PrintJobStatusFailed {
			summary.FailedCount++
		}

		// Cost breakdown (from completed jobs with snapshots)
		if job.PrinterTimeCostCents != nil {
			summary.PrinterTimeCostCents += *job.PrinterTimeCostCents
		}
		if job.MaterialCostCents != nil {
			summary.MaterialCostCents += *job.MaterialCostCents
		}
		if job.CostCents != nil {
			summary.TotalCostCents += *job.CostCents
		}

		// Print time
		if job.ActualSeconds != nil && *job.ActualSeconds > 0 {
			summary.TotalPrintSeconds += *job.ActualSeconds
		}

		// Material
		if job.MaterialUsedGrams != nil {
			summary.TotalMaterialGrams += *job.MaterialUsedGrams
		}
	}

	// Derived metrics
	if summary.CompletedCount+summary.FailedCount > 0 {
		summary.SuccessRate = float64(summary.CompletedCount) / float64(summary.CompletedCount+summary.FailedCount) * 100
	}
	if summary.CompletedCount > 0 {
		summary.AvgPrintSeconds = summary.TotalPrintSeconds / summary.CompletedCount
	}

	// Estimated material cost, grams, and print time from slice profiles
	if s.partRepo != nil && s.designRepo != nil {
		parts, err := s.partRepo.ListByProject(ctx, projectID)
		if err == nil {
			for _, part := range parts {
				designs, err := s.designRepo.ListByPart(ctx, part.ID)
				if err != nil || len(designs) == 0 {
					continue
				}
				// Latest design is first (ordered by version DESC)
				latest := designs[0]
				if latest.SliceProfile != nil {
					var profile model.SliceProfileData
					if json.Unmarshal(latest.SliceProfile, &profile) == nil {
						if profile.WeightGrams > 0 {
							// Default cost: $19.99/kg
							costPerKg := 19.99
							costCents := int(profile.WeightGrams / 1000.0 * costPerKg * 100)
							summary.EstimatedMaterialCostCents += costCents * part.Quantity
							summary.EstimatedMaterialGrams += profile.WeightGrams * float64(part.Quantity)
						}
						if profile.PrintTimeSeconds > 0 {
							summary.EstimatedPrintSeconds += profile.PrintTimeSeconds * part.Quantity
						}
					}
				}
			}
		}
	}

	// Supply costs
	if s.supplyRepo != nil {
		supplies, err := s.supplyRepo.ListByProject(ctx, projectID)
		if err == nil {
			for _, supply := range supplies {
				summary.SupplyCostCents += supply.UnitCostCents * supply.Quantity
			}
		}
	}

	// Include estimated material and supply costs in total
	summary.TotalCostCents += summary.EstimatedMaterialCostCents + summary.SupplyCostCents

	// UnitCostCents is the per-unit cost of production
	summary.UnitCostCents = summary.TotalCostCents

	// TotalCostCents is total COGS: per-unit cost × number of sales
	if summary.SalesCount > 1 {
		summary.TotalCostCents = summary.UnitCostCents * summary.SalesCount
	}

	summary.GrossProfitCents = summary.NetRevenueCents - summary.TotalCostCents
	if summary.NetRevenueCents > 0 {
		summary.GrossMarginPercent = float64(summary.GrossProfitCents) / float64(summary.NetRevenueCents) * 100
	}

	// Profit per hour: (profit from one sale) / (print time for one unit in hours)
	printSeconds := summary.TotalPrintSeconds
	if printSeconds <= 0 {
		printSeconds = summary.EstimatedPrintSeconds
	}
	if printSeconds > 0 && summary.SalesCount > 0 {
		profitPerSale := float64(summary.GrossProfitCents) / float64(summary.SalesCount)
		hours := float64(printSeconds) / 3600.0
		summary.ProfitPerHourCents = int(profitPerSale / hours)
	}

	return summary, nil
}

// StartProductionResult contains the result of starting production.
type StartProductionResult struct {
	JobsStarted    int               `json:"jobs_started"`
	JobsSkipped    int               `json:"jobs_skipped"`
	FailedJobs     []StartJobFailure `json:"failed_jobs,omitempty"`
}

// StartJobFailure represents a job that failed to start.
type StartJobFailure struct {
	JobID   uuid.UUID `json:"job_id"`
	Reason  string    `json:"reason"`
}

// StartProduction auto-assigns resources and starts all queued jobs for a project.
func (s *ProjectService) StartProduction(ctx context.Context, projectID uuid.UUID) (*StartProductionResult, error) {
	project, err := s.repo.GetByID(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	if project == nil {
		return nil, fmt.Errorf("project not found")
	}

	// Get all jobs for this project
	jobs, err := s.printJobRepo.ListByProject(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project jobs: %w", err)
	}

	// Get template constraints if this project has a template
	var template *model.Template
	if project.TemplateID != nil {
		template, err = s.templateRepo.GetByID(ctx, *project.TemplateID)
		if err != nil {
			return nil, fmt.Errorf("failed to get template: %w", err)
		}
	}

	// Get available printers and spools
	printers, err := s.printerRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get printers: %w", err)
	}

	spools, err := s.spoolRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get spools: %w", err)
	}

	result := &StartProductionResult{}

	for _, job := range jobs {
		// Skip jobs that are not in queued status
		if job.Status != model.PrintJobStatusQueued {
			result.JobsSkipped++
			continue
		}

		// Find an idle printer
		var selectedPrinter *model.Printer
		for i := range printers {
			p := &printers[i]
			if p.Status != model.PrinterStatusIdle {
				continue
			}
			// Check template constraints
			if template != nil {
				if template.PreferredPrinterID != nil && !template.AllowAnyPrinter {
					if p.ID != *template.PreferredPrinterID {
						continue
					}
				}
				// Check printer constraints if specified
				if template.PrinterConstraints != nil {
					// Note: Printer model doesn't have HasEnclosure/HasAMS yet
					// These checks would need printer model updates to fully work
					// For now, we skip these checks
				}
			}
			selectedPrinter = p
			break
		}

		if selectedPrinter == nil {
			result.FailedJobs = append(result.FailedJobs, StartJobFailure{
				JobID:  job.ID,
				Reason: "no idle printer available",
			})
			continue
		}

		// Find a spool with matching material and enough weight
		var selectedSpool *model.MaterialSpool
		for i := range spools {
			sp := &spools[i]
			if sp.Status != model.SpoolStatusInUse && sp.Status != model.SpoolStatusNew && sp.Status != model.SpoolStatusLow {
				continue
			}
			// Check if spool has enough material (at least 50g as minimum)
			if sp.RemainingWeight < 50 {
				continue
			}
			// TODO: Check material type constraints from template if specified
			selectedSpool = sp
			break
		}

		if selectedSpool == nil {
			result.FailedJobs = append(result.FailedJobs, StartJobFailure{
				JobID:  job.ID,
				Reason: "no suitable spool available",
			})
			continue
		}

		// Assign resources to the job
		job.PrinterID = &selectedPrinter.ID
		job.MaterialSpoolID = &selectedSpool.ID

		if err := s.printJobRepo.Update(ctx, &job); err != nil {
			result.FailedJobs = append(result.FailedJobs, StartJobFailure{
				JobID:  job.ID,
				Reason: fmt.Sprintf("failed to assign resources: %v", err),
			})
			continue
		}

		// Record assignment event
		assignedStatus := model.PrintJobStatusAssigned
		event := model.NewJobEvent(job.ID, model.JobEventAssigned, &assignedStatus).
			WithPrinter(selectedPrinter.ID).
			WithActor(model.ActorSystem, "start_production")
		if err := s.printJobRepo.AppendEvent(ctx, event); err != nil {
			slog.Error("failed to record assignment event", "job_id", job.ID, "error", err)
		}

		// Get design and start the job
		design, err := s.designRepo.GetByID(ctx, job.DesignID)
		if err != nil || design == nil {
			result.FailedJobs = append(result.FailedJobs, StartJobFailure{
				JobID:  job.ID,
				Reason: "design not found",
			})
			continue
		}

		// Send to printer
		if err := s.printerMgr.StartJob(selectedPrinter.ID, design.FileName, s.storage.GetFullPath(design.FileName)); err != nil {
			// Record failure event
			failedStatus := model.PrintJobStatusFailed
			failEvent := model.NewJobEvent(job.ID, model.JobEventFailed, &failedStatus).
				WithError("UPLOAD_FAILED", err.Error()).
				WithActor(model.ActorSystem, "start_production")
			s.printJobRepo.AppendEvent(ctx, failEvent)

			result.FailedJobs = append(result.FailedJobs, StartJobFailure{
				JobID:  job.ID,
				Reason: fmt.Sprintf("failed to start print: %v", err),
			})
			continue
		}

		// Record uploaded event
		uploadedStatus := model.PrintJobStatusUploaded
		uploadEvent := model.NewJobEvent(job.ID, model.JobEventUploaded, &uploadedStatus).
			WithPrinter(selectedPrinter.ID).
			WithActor(model.ActorSystem, "start_production")
		s.printJobRepo.AppendEvent(ctx, uploadEvent)

		// Update started_at timestamp
		now := time.Now()
		job.StartedAt = &now
		s.printJobRepo.Update(ctx, &job)

		// Mark printer as printing
		selectedPrinter.Status = model.PrinterStatusPrinting
		result.JobsStarted++
	}

	s.hub.Broadcast(realtime.Event{
		Type: "production_started",
		Data: map[string]interface{}{
			"project_id":   projectID,
			"jobs_started": result.JobsStarted,
		},
	})

	return result, nil
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

	// Extract slicer metadata from 3MF files
	if fileType == model.FileType3MF {
		fullPath := s.storage.GetFullPath(storagePath)
		if profile, err := threemf.Parse(fullPath); err != nil {
			slog.Warn("failed to parse 3MF metadata", "file", filename, "error", err)
		} else if profile != nil {
			design.SliceProfile = profile
		}
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

// OpenInExternalApp opens a design file in an external application.
func (s *DesignService) OpenInExternalApp(ctx context.Context, id uuid.UUID, appName string) error {
	design, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if design == nil {
		return fmt.Errorf("design not found")
	}

	file, err := s.fileRepo.GetByID(ctx, design.FileID)
	if err != nil {
		return err
	}
	if file == nil {
		return fmt.Errorf("file not found")
	}

	fullPath := s.storage.GetFullPath(file.StoragePath)
	if _, err := os.Stat(fullPath); err != nil {
		return fmt.Errorf("file not found on disk: %w", err)
	}

	var cmd *exec.Cmd
	if appName != "" {
		cmd = exec.Command("open", "-a", appName, fullPath)
	} else {
		cmd = exec.Command("open", fullPath)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to open application: %w: %s", err, string(output))
	}

	return nil
}

// PrinterService handles printer business logic.
type PrinterService struct {
	repo           *repository.PrinterRepository
	printJobRepo   *repository.PrintJobRepository
	saleRepo       *repository.SaleRepository
	manager        *printer.Manager
	hub            *realtime.Hub
	discovery      *printer.Discovery
	bambuCloudRepo *repository.BambuCloudRepository
	bambuCloud     *bambu.CloudClient
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

// ListJobs retrieves all print jobs for a printer.
func (s *PrinterService) ListJobs(ctx context.Context, printerID uuid.UUID) ([]model.PrintJob, error) {
	return s.printJobRepo.List(ctx, &printerID, nil)
}

// GetJobStats retrieves job statistics for a printer.
func (s *PrinterService) GetJobStats(ctx context.Context, printerID uuid.UUID) (*repository.JobStats, error) {
	return s.printJobRepo.GetPrinterJobStats(ctx, printerID)
}

// GetPrinterAnalytics computes comprehensive analytics for a printer.
func (s *PrinterService) GetPrinterAnalytics(ctx context.Context, printerID uuid.UUID) (*model.PrinterAnalytics, error) {
	// Fetch printer metadata
	printer, err := s.repo.GetByID(ctx, printerID)
	if err != nil {
		return nil, fmt.Errorf("get printer: %w", err)
	}
	if printer == nil {
		return nil, fmt.Errorf("printer not found")
	}

	// Compute revenue attribution (lifetime)
	revenueCents, err := s.repo.GetPrinterRevenueAttribution(ctx, printerID)
	if err != nil {
		slog.Warn("failed to get revenue attribution", "printer_id", printerID, "error", err)
		revenueCents = 0
	}

	// Compute utilization for 7d, 30d, 90d periods
	now := time.Now()
	periods := []struct {
		label string
		since time.Time
	}{
		{"7d", now.AddDate(0, 0, -7)},
		{"30d", now.AddDate(0, 0, -30)},
		{"90d", now.AddDate(0, 0, -90)},
	}

	var utilizations []model.PrinterUtilization
	for _, p := range periods {
		data, err := s.repo.GetPrinterUtilizationData(ctx, printerID, p.since)
		if err != nil {
			slog.Warn("failed to get utilization data", "period", p.label, "error", err)
			continue
		}

		totalHours := now.Sub(p.since).Hours()
		printingHours := float64(data.CompletedSeconds) / 3600.0
		failedHours := float64(data.FailedSeconds) / 3600.0
		idleHours := totalHours - printingHours - failedHours
		if idleHours < 0 {
			idleHours = 0
		}

		var utilizationPercent float64
		if totalHours > 0 {
			utilizationPercent = (printingHours / totalHours) * 100
		}

		var actualRevenuePerHour int
		if printingHours > 0 {
			// Proportionally attribute revenue to this period based on total printing hours
			healthData, _ := s.repo.GetPrinterHealthData(ctx, printerID)
			if healthData != nil && healthData.TotalSeconds > 0 {
				totalPrintingHours := float64(healthData.TotalSeconds) / 3600.0
				periodRevenueShare := (printingHours / totalPrintingHours) * float64(revenueCents)
				actualRevenuePerHour = int(periodRevenueShare / printingHours)
			}
		}

		utilizations = append(utilizations, model.PrinterUtilization{
			Period:                     p.label,
			TotalHours:                 totalHours,
			PrintingHours:              printingHours,
			FailedHours:                failedHours,
			IdleHours:                  idleHours,
			UtilizationPercent:         utilizationPercent,
			ConfiguredCostPerHourCents: printer.CostPerHourCents,
			ActualRevenuePerHourCents:  actualRevenuePerHour,
		})
	}

	// Compute health metrics
	healthData, err := s.repo.GetPrinterHealthData(ctx, printerID)
	if err != nil {
		return nil, fmt.Errorf("get health data: %w", err)
	}

	failureBreakdown, err := s.repo.GetPrinterFailureBreakdown(ctx, printerID)
	if err != nil {
		slog.Warn("failed to get failure breakdown", "printer_id", printerID, "error", err)
		failureBreakdown = make(map[string]int)
	}

	var failureRate float64
	if healthData.TotalJobs > 0 {
		failureRate = float64(healthData.FailedJobs) / float64(healthData.TotalJobs) * 100
	}

	var avgJobDuration int
	var avgCost int
	if healthData.CompletedJobs > 0 {
		avgJobDuration = healthData.TotalSeconds / healthData.CompletedJobs
		avgCost = healthData.TotalCostCents / healthData.CompletedJobs
	}

	health := &model.PrinterHealth{
		TotalJobs:          healthData.TotalJobs,
		CompletedJobs:      healthData.CompletedJobs,
		FailedJobs:         healthData.FailedJobs,
		FailureRate:        failureRate,
		AvgJobDurationSec:  avgJobDuration,
		AvgCostCents:       avgCost,
		TotalMaterialGrams: healthData.TotalMaterialGrams,
		TotalCostCents:     healthData.TotalCostCents,
		TotalRevenueCents:  revenueCents,
		FailureBreakdown:   failureBreakdown,
	}

	// Compute ROI metrics
	totalPrintingHours := float64(healthData.TotalSeconds) / 3600.0
	printerAgeHours := now.Sub(printer.CreatedAt).Hours()

	var revenuePerHour, costPerHour, netPerHour int
	if totalPrintingHours > 0 {
		revenuePerHour = int(float64(revenueCents) / totalPrintingHours)
		costPerHour = int(float64(healthData.TotalCostCents) / totalPrintingHours)
		netPerHour = revenuePerHour - costPerHour
	}

	lifetimeProfit := revenueCents - healthData.TotalCostCents - printer.PurchasePriceCents

	var hoursToBreakEven float64
	breakEvenReached := lifetimeProfit >= 0
	if !breakEvenReached && netPerHour > 0 {
		remainingToBreakEven := -lifetimeProfit
		hoursToBreakEven = float64(remainingToBreakEven) / float64(netPerHour)
	} else if breakEvenReached {
		// Calculate when break-even was reached
		if netPerHour > 0 {
			hoursToBreakEven = float64(printer.PurchasePriceCents) / float64(netPerHour)
		}
	}

	roi := &model.PrinterROI{
		PurchasePriceCents:  printer.PurchasePriceCents,
		TotalRevenueCents:   revenueCents,
		TotalCostCents:      healthData.TotalCostCents,
		LifetimeProfitCents: lifetimeProfit,
		TotalPrintingHours:  totalPrintingHours,
		RevenuePerHourCents: revenuePerHour,
		CostPerHourCents:    costPerHour,
		NetPerHourCents:     netPerHour,
		HoursToBreakEven:    hoursToBreakEven,
		PrinterAgeHours:     printerAgeHours,
		BreakEvenReached:    breakEvenReached,
	}

	return &model.PrinterAnalytics{
		Utilization: utilizations,
		ROI:         roi,
		Health:      health,
	}, nil
}

// DiscoverPrinters scans the network for printers.
func (s *PrinterService) DiscoverPrinters(ctx context.Context) ([]printer.DiscoveredPrinter, error) {
	discovered, err := s.discovery.QuickScan(ctx)
	if err != nil {
		return nil, err
	}

	// Mark printers that are already added
	existing, _ := s.repo.List(ctx)
	existingURIs := make(map[string]bool)
	for _, p := range existing {
		existingURIs[p.ConnectionURI] = true
	}

	for i := range discovered {
		host := discovered[i].Host
		uri := fmt.Sprintf("http://%s:%d", host, discovered[i].Port)
		// Check both full URI and bare host (Bambu printers store just the IP)
		if existingURIs[uri] || existingURIs[host] {
			discovered[i].AlreadyAdded = true
		}
	}

	return discovered, nil
}

// ConnectAllPrinters loads all printers from the database and connects
// non-manual printers. Called at startup to restore connections.
// For bambu_cloud printers, credentials are refreshed from the stored
// cloud auth so that token updates from later logins are picked up.
func (s *PrinterService) ConnectAllPrinters(ctx context.Context) {
	printers, err := s.repo.List(ctx)
	if err != nil {
		slog.Error("failed to load printers for reconnection", "error", err)
		return
	}

	// Load and validate cloud auth once if any cloud printers exist
	var cloudAuth *model.BambuCloudAuth
	for _, p := range printers {
		if p.ConnectionType == model.ConnectionTypeBambuCloud {
			cloudAuth, _ = s.bambuCloudRepo.Get(ctx)
			if cloudAuth != nil {
				// Check if token needs refresh
				if cloudAuth.IsExpired(tokenRefreshBuffer) && cloudAuth.CanRefresh() {
					slog.Info("refreshing expired bambu cloud token at startup")
					refreshedAuth, err := s.refreshBambuToken(ctx, cloudAuth)
					if err != nil {
						slog.Error("failed to refresh bambu cloud token at startup", "error", err)
						cloudAuth = nil // Mark as unavailable
					} else {
						cloudAuth = refreshedAuth
						slog.Info("bambu cloud token refreshed successfully at startup")
					}
				} else if cloudAuth.IsExpired(0) {
					slog.Warn("bambu cloud token expired and no refresh token available")
					cloudAuth = nil
				}
			}
			break
		}
	}

	for i := range printers {
		p := &printers[i]
		if p.ConnectionType == model.ConnectionTypeManual {
			continue
		}

		// For cloud printers, inject the latest auth credentials
		if p.ConnectionType == model.ConnectionTypeBambuCloud {
			if cloudAuth == nil {
				slog.Warn("skipping cloud printer — no valid Bambu Cloud credentials",
					"printer_id", p.ID, "name", p.Name)
				continue
			}
			p.ConnectionURI = cloudAuth.MQTTUsername
			p.APIKey = cloudAuth.AccessToken
		}

		slog.Info("reconnecting printer", "id", p.ID, "name", p.Name, "type", p.ConnectionType)
		go s.manager.Connect(p)
	}
}

// refreshBambuToken refreshes the Bambu Cloud access token.
func (s *PrinterService) refreshBambuToken(ctx context.Context, auth *model.BambuCloudAuth) (*model.BambuCloudAuth, error) {
	if s.bambuCloud == nil {
		return nil, fmt.Errorf("bambu cloud client not configured")
	}

	resp, err := s.bambuCloud.RefreshToken(auth.RefreshToken)
	if err != nil {
		return nil, err
	}

	// Calculate new expiration time
	var expiresAt *time.Time
	if resp.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second)
		expiresAt = &t
	}

	// Update auth with new tokens
	auth.AccessToken = resp.AccessToken
	if resp.RefreshToken != "" {
		auth.RefreshToken = resp.RefreshToken
	}
	auth.ExpiresAt = expiresAt

	// Persist updated credentials
	if err := s.bambuCloudRepo.Upsert(ctx, auth); err != nil {
		return nil, fmt.Errorf("failed to save refreshed credentials: %w", err)
	}

	return auth, nil
}

// BambuCloudService handles Bambu Cloud authentication and device management.
type BambuCloudService struct {
	cloud      *bambu.CloudClient
	repo       *repository.BambuCloudRepository
	printerRepo *repository.PrinterRepository
	printerMgr *printer.Manager
}

// NewBambuCloudService creates a new BambuCloudService.
func NewBambuCloudService(repo *repository.BambuCloudRepository, printerRepo *repository.PrinterRepository, printerMgr *printer.Manager, cloud *bambu.CloudClient) *BambuCloudService {
	return &BambuCloudService{
		cloud:       cloud,
		repo:        repo,
		printerRepo: printerRepo,
		printerMgr:  printerMgr,
	}
}

// Login authenticates with Bambu Cloud. Returns true if a verification code is needed.
func (s *BambuCloudService) Login(ctx context.Context, email, password string) (needsCode bool, err error) {
	resp, err := s.cloud.Login(email, password)
	if err != nil {
		return false, err
	}

	if resp.LoginType == "verifyCode" || resp.LoginType == "tfa" {
		// Need email verification code
		if err := s.cloud.RequestVerifyCode(email); err != nil {
			return false, fmt.Errorf("failed to request verification code: %w", err)
		}
		return true, nil
	}

	// Direct login succeeded — store credentials
	return false, s.storeAuth(ctx, email, resp)
}

// VerifyCode completes login with a verification code.
func (s *BambuCloudService) VerifyCode(ctx context.Context, email, code string) error {
	resp, err := s.cloud.LoginWithCode(email, code)
	if err != nil {
		return err
	}
	return s.storeAuth(ctx, email, resp)
}

// storeAuth fetches the MQTT username and persists credentials.
func (s *BambuCloudService) storeAuth(ctx context.Context, email string, resp *bambu.LoginResponse) error {
	mqttUsername, err := s.cloud.GetUsername(resp.AccessToken)
	if err != nil {
		return fmt.Errorf("failed to get MQTT username: %w", err)
	}

	var expiresAt *time.Time
	if resp.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second)
		expiresAt = &t
	}

	auth := &model.BambuCloudAuth{
		Email:        email,
		AccessToken:  resp.AccessToken,
		RefreshToken: resp.RefreshToken,
		MQTTUsername: mqttUsername,
		ExpiresAt:    expiresAt,
	}
	return s.repo.Upsert(ctx, auth)
}

// tokenRefreshBuffer is how long before expiration we should refresh the token.
const tokenRefreshBuffer = 5 * time.Minute

// GetValidAuth retrieves auth credentials, automatically refreshing if expired.
// Returns an error if not authenticated or if refresh fails.
func (s *BambuCloudService) GetValidAuth(ctx context.Context) (*model.BambuCloudAuth, error) {
	auth, err := s.repo.Get(ctx)
	if err != nil {
		return nil, err
	}
	if auth == nil {
		return nil, fmt.Errorf("not authenticated with Bambu Cloud")
	}

	// Check if token is expired or about to expire
	if auth.IsExpired(tokenRefreshBuffer) {
		slog.Info("bambu cloud token expired or expiring soon, attempting refresh",
			"expires_at", auth.ExpiresAt,
			"can_refresh", auth.CanRefresh())

		if !auth.CanRefresh() {
			return nil, fmt.Errorf("bambu cloud token expired and no refresh token available - please re-login")
		}

		// Attempt refresh
		refreshedAuth, err := s.refreshToken(ctx, auth)
		if err != nil {
			slog.Error("failed to refresh bambu cloud token", "error", err)
			return nil, fmt.Errorf("failed to refresh token: %w - please re-login", err)
		}
		auth = refreshedAuth
		slog.Info("bambu cloud token refreshed successfully", "new_expires_at", auth.ExpiresAt)
	}

	return auth, nil
}

// refreshToken uses the refresh token to get new credentials.
func (s *BambuCloudService) refreshToken(ctx context.Context, auth *model.BambuCloudAuth) (*model.BambuCloudAuth, error) {
	resp, err := s.cloud.RefreshToken(auth.RefreshToken)
	if err != nil {
		return nil, err
	}

	// Calculate new expiration time
	var expiresAt *time.Time
	if resp.ExpiresIn > 0 {
		t := time.Now().Add(time.Duration(resp.ExpiresIn) * time.Second)
		expiresAt = &t
	}

	// Update auth with new tokens (keep existing email and mqtt username)
	auth.AccessToken = resp.AccessToken
	if resp.RefreshToken != "" {
		auth.RefreshToken = resp.RefreshToken
	}
	auth.ExpiresAt = expiresAt

	// Persist updated credentials
	if err := s.repo.Upsert(ctx, auth); err != nil {
		return nil, fmt.Errorf("failed to save refreshed credentials: %w", err)
	}

	return auth, nil
}

// GetDevices fetches the device list from Bambu Cloud.
func (s *BambuCloudService) GetDevices(ctx context.Context) ([]bambu.CloudDevice, error) {
	auth, err := s.GetValidAuth(ctx)
	if err != nil {
		return nil, err
	}
	return s.cloud.GetDevices(auth.AccessToken)
}

// GetStoredAuth returns the stored authentication info (without exposing the token).
func (s *BambuCloudService) GetStoredAuth(ctx context.Context) (*model.BambuCloudAuth, error) {
	return s.repo.Get(ctx)
}

// AddDevice creates a printer from a cloud device and connects it.
func (s *BambuCloudService) AddDevice(ctx context.Context, devID string) (*model.Printer, error) {
	auth, err := s.GetValidAuth(ctx)
	if err != nil {
		return nil, err
	}

	devices, err := s.cloud.GetDevices(auth.AccessToken)
	if err != nil {
		return nil, err
	}

	// Find the requested device
	var device *bambu.CloudDevice
	for i := range devices {
		if devices[i].DevID == devID {
			device = &devices[i]
			break
		}
	}
	if device == nil {
		return nil, fmt.Errorf("device %s not found in cloud account", devID)
	}

	// Create the printer record
	p := &model.Printer{
		Name:           device.Name,
		Model:          device.DevProductName,
		Manufacturer:   "Bambu Lab",
		ConnectionType: model.ConnectionTypeBambuCloud,
		ConnectionURI:  auth.MQTTUsername,   // Store MQTT username here
		APIKey:         auth.AccessToken,     // Store auth token here
		SerialNumber:   device.DevID,
		NozzleDiameter: device.NozzleDiameter,
	}
	if p.NozzleDiameter == 0 {
		p.NozzleDiameter = 0.4
	}

	if err := s.printerRepo.Create(ctx, p); err != nil {
		return nil, err
	}

	// Connect via cloud MQTT
	go s.printerMgr.Connect(p)

	return p, nil
}

// Logout clears stored Bambu Cloud credentials.
func (s *BambuCloudService) Logout(ctx context.Context) error {
	return s.repo.Delete(ctx)
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

// ListByType retrieves all materials of a given type.
func (s *MaterialService) ListByType(ctx context.Context, matType model.MaterialType) ([]model.Material, error) {
	return s.repo.ListByType(ctx, matType)
}

// Delete removes a material by ID.
func (s *MaterialService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
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
// Jobs are immutable once created - state changes are recorded as events.
type PrintJobService struct {
	repo            *repository.PrintJobRepository
	printerRepo     *repository.PrinterRepository
	designRepo      *repository.DesignRepository
	spoolRepo       *repository.SpoolRepository
	materialRepo    *repository.MaterialRepository
	projectRepo     *repository.ProjectRepository
	printerMgr      *printer.Manager
	hub             *realtime.Hub
	storage         storage.Storage
	onJobCompleted  func(ctx context.Context, job *model.PrintJob)
}

// SetOnJobCompleted sets a callback invoked when a print job completes successfully.
func (s *PrintJobService) SetOnJobCompleted(fn func(ctx context.Context, job *model.PrintJob)) {
	s.onJobCompleted = fn
}

// Create creates a new print job and records the initial "queued" event.
// Printer and spool can be nil - job will be queued pending assignment.
func (s *PrintJobService) Create(ctx context.Context, j *model.PrintJob) error {
	if j.DesignID == uuid.Nil {
		return fmt.Errorf("design ID is required")
	}
	// Printer and spool are optional - job can be created without assignment
	// The repository.Create already records the initial queued event
	return s.repo.Create(ctx, j)
}

// GetByID retrieves a print job by ID.
func (s *PrintJobService) GetByID(ctx context.Context, id uuid.UUID) (*model.PrintJob, error) {
	return s.repo.GetByID(ctx, id)
}

// GetByIDWithEvents retrieves a print job with its full event timeline.
func (s *PrintJobService) GetByIDWithEvents(ctx context.Context, id uuid.UUID) (*model.PrintJob, error) {
	return s.repo.GetByIDWithEvents(ctx, id)
}

// List retrieves print jobs.
func (s *PrintJobService) List(ctx context.Context, printerID *uuid.UUID, status *model.PrintJobStatus) ([]model.PrintJob, error) {
	return s.repo.List(ctx, printerID, status)
}

// ListByDesign retrieves print jobs for a design.
func (s *PrintJobService) ListByDesign(ctx context.Context, designID uuid.UUID) ([]model.PrintJob, error) {
	return s.repo.ListByDesign(ctx, designID)
}

// ListByRecipe retrieves print jobs for a recipe/template.
func (s *PrintJobService) ListByRecipe(ctx context.Context, recipeID uuid.UUID) ([]model.PrintJob, error) {
	return s.repo.ListByRecipe(ctx, recipeID)
}

// GetEvents retrieves all events for a job in chronological order.
func (s *PrintJobService) GetEvents(ctx context.Context, jobID uuid.UUID) ([]model.JobEvent, error) {
	return s.repo.GetEvents(ctx, jobID)
}

// GetRetryChain retrieves all jobs in a retry chain (original + all retries).
func (s *PrintJobService) GetRetryChain(ctx context.Context, jobID uuid.UUID) ([]model.PrintJob, error) {
	return s.repo.GetRetryChain(ctx, jobID)
}

// Start sends a print job to the printer and records the appropriate events.
func (s *PrintJobService) Start(ctx context.Context, id uuid.UUID) error {
	job, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if job == nil {
		return fmt.Errorf("job not found")
	}

	// Verify job is in a startable state
	if job.Status.IsTerminal() {
		return fmt.Errorf("cannot start job in %s status", job.Status)
	}

	// Verify resources are assigned before starting
	if job.NeedsAssignment() {
		return fmt.Errorf("job needs printer and spool assignment before starting")
	}

	// Get design and file
	design, err := s.designRepo.GetByID(ctx, job.DesignID)
	if err != nil {
		return err
	}

	// Get printer
	printerData, err := s.printerRepo.GetByID(ctx, *job.PrinterID)
	if err != nil {
		return err
	}
	if printerData == nil {
		return fmt.Errorf("printer not found")
	}

	// Get printer state and validate material (if AMS available)
	printerState, _ := s.printerMgr.GetState(*job.PrinterID)
	if printerState != nil && printerState.AMS != nil {
		validation := s.ValidateMaterial(printerState.AMS, printerData.MinMaterialPercent)
		if len(validation.Errors) > 0 {
			return fmt.Errorf("material validation failed: %s", validation.Errors[0])
		}

		// Capture material snapshot before starting
		job.MaterialSnapshot = s.captureMaterialSnapshot(printerState.AMS)
	}

	// Record assignment event if printer wasn't already assigned
	if job.Status == model.PrintJobStatusQueued {
		assignedStatus := model.PrintJobStatusAssigned
		assignEvent := model.NewJobEvent(job.ID, model.JobEventAssigned, &assignedStatus).
			WithPrinter(*job.PrinterID).
			WithActor(model.ActorSystem, "print_service")
		if err := s.repo.AppendEvent(ctx, assignEvent); err != nil {
			return fmt.Errorf("failed to record assignment: %w", err)
		}
	}

	// Send to printer via manager
	if err := s.printerMgr.StartJob(*job.PrinterID, design.FileName, s.storage.GetFullPath(design.FileName)); err != nil {
		// Record failure event
		failedStatus := model.PrintJobStatusFailed
		failEvent := model.NewJobEvent(job.ID, model.JobEventFailed, &failedStatus).
			WithError("UPLOAD_FAILED", err.Error()).
			WithActor(model.ActorSystem, "print_service")
		s.repo.AppendEvent(ctx, failEvent)
		return fmt.Errorf("failed to start print: %w", err)
	}

	// Record uploaded event
	uploadedStatus := model.PrintJobStatusUploaded
	uploadEvent := model.NewJobEvent(job.ID, model.JobEventUploaded, &uploadedStatus).
		WithPrinter(*job.PrinterID).
		WithActor(model.ActorSystem, "print_service")
	if err := s.repo.AppendEvent(ctx, uploadEvent); err != nil {
		return fmt.Errorf("failed to record upload: %w", err)
	}

	// Update started_at timestamp and material snapshot
	now := time.Now()
	job.StartedAt = &now
	s.repo.Update(ctx, job)

	// Broadcast update
	s.hub.Broadcast(realtime.Event{
		Type: "job_started",
		Data: job,
	})

	return nil
}

// ValidateMaterial validates the current AMS material state against thresholds.
// Returns warnings for low material and errors that should block job start.
func (s *PrintJobService) ValidateMaterial(ams *model.AMSState, minPercent int) *model.MaterialValidation {
	result := &model.MaterialValidation{Valid: true}

	if ams == nil {
		return result
	}

	// Find the currently selected tray
	var currentTray *model.AMSTray
	if ams.CurrentTray == "255" && ams.ExternalSpool != nil {
		currentTray = ams.ExternalSpool
	} else if ams.CurrentTray != "" {
		// Parse tray number (format: "X" where X is 0-15 for AMS trays)
		for _, unit := range ams.Units {
			for i := range unit.Trays {
				tray := &unit.Trays[i]
				// Calculate global tray ID: unit_id * 4 + tray_id
				globalID := unit.ID*4 + tray.ID
				if fmt.Sprintf("%d", globalID) == ams.CurrentTray {
					currentTray = tray
					break
				}
			}
			if currentTray != nil {
				break
			}
		}
	}

	if currentTray == nil {
		// No tray selected, can't validate
		return result
	}

	// Check if tray is empty
	if currentTray.Empty {
		result.Valid = false
		result.Errors = append(result.Errors, "selected tray is empty")
		return result
	}

	// Check remaining percentage against threshold
	if minPercent > 0 && currentTray.Remain < minPercent {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("material remaining (%d%%) is below minimum threshold (%d%%)", currentTray.Remain, minPercent))
	} else if currentTray.Remain < 20 {
		// Add warning for low material even if above threshold
		result.Warnings = append(result.Warnings, fmt.Sprintf("material low: %d%% remaining", currentTray.Remain))
	}

	return result
}

// captureMaterialSnapshot creates a snapshot of the current AMS state for a job.
func (s *PrintJobService) captureMaterialSnapshot(ams *model.AMSState) *model.MaterialSnapshot {
	if ams == nil {
		return nil
	}

	snapshot := &model.MaterialSnapshot{
		CapturedAt: time.Now(),
		AMSState:   ams,
	}

	// Find and record the currently selected tray info
	if ams.CurrentTray == "255" && ams.ExternalSpool != nil {
		snapshot.SelectedTray = 255
		snapshot.MaterialType = ams.ExternalSpool.MaterialType
		snapshot.Color = ams.ExternalSpool.Color
		snapshot.RemainPercent = ams.ExternalSpool.Remain
		snapshot.Brand = ams.ExternalSpool.Brand
	} else if ams.CurrentTray != "" {
		for _, unit := range ams.Units {
			for _, tray := range unit.Trays {
				globalID := unit.ID*4 + tray.ID
				if fmt.Sprintf("%d", globalID) == ams.CurrentTray {
					snapshot.SelectedTray = globalID
					snapshot.MaterialType = tray.MaterialType
					snapshot.Color = tray.Color
					snapshot.RemainPercent = tray.Remain
					snapshot.Brand = tray.Brand
					break
				}
			}
		}
	}

	return snapshot
}

// PreflightCheckResult contains the result of a preflight validation.
type PreflightCheckResult struct {
	Ready      bool                     `json:"ready"`
	Validation *model.MaterialValidation `json:"validation,omitempty"`
	AMSState   *model.AMSState          `json:"ams_state,omitempty"`
	Warnings   []string                  `json:"warnings,omitempty"`
	Errors     []string                  `json:"errors,omitempty"`
}

// PreflightCheck validates a job is ready to start.
// Returns current AMS state, validation results, and any warnings/errors.
func (s *PrintJobService) PreflightCheck(ctx context.Context, jobID uuid.UUID) (*PreflightCheckResult, error) {
	job, err := s.repo.GetByID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, fmt.Errorf("job not found")
	}

	result := &PreflightCheckResult{Ready: true}

	// Check job state
	if job.Status.IsTerminal() {
		result.Ready = false
		result.Errors = append(result.Errors, fmt.Sprintf("job is in terminal state: %s", job.Status))
		return result, nil
	}

	// Check resource assignment
	if job.NeedsAssignment() {
		result.Ready = false
		result.Errors = append(result.Errors, "job needs printer and spool assignment")
		return result, nil
	}

	// Get printer
	printerData, err := s.printerRepo.GetByID(ctx, *job.PrinterID)
	if err != nil || printerData == nil {
		result.Ready = false
		result.Errors = append(result.Errors, "printer not found")
		return result, nil
	}

	// Get printer state including AMS
	printerState, _ := s.printerMgr.GetState(*job.PrinterID)
	if printerState == nil {
		result.Warnings = append(result.Warnings, "printer state unavailable")
		return result, nil
	}

	// Include AMS state in result
	result.AMSState = printerState.AMS

	// Validate material if AMS available
	if printerState.AMS != nil {
		validation := s.ValidateMaterial(printerState.AMS, printerData.MinMaterialPercent)
		result.Validation = validation
		result.Warnings = append(result.Warnings, validation.Warnings...)
		result.Errors = append(result.Errors, validation.Errors...)
		if !validation.Valid {
			result.Ready = false
		}
	}

	return result, nil
}

// AssignResources assigns a printer and spool to a job.
func (s *PrintJobService) AssignResources(ctx context.Context, jobID, printerID, spoolID uuid.UUID) error {
	job, err := s.repo.GetByID(ctx, jobID)
	if err != nil {
		return err
	}
	if job == nil {
		return fmt.Errorf("job not found")
	}

	if job.Status.IsTerminal() {
		return fmt.Errorf("cannot assign resources to job in %s status", job.Status)
	}

	// Verify printer exists
	printer, err := s.printerRepo.GetByID(ctx, printerID)
	if err != nil {
		return fmt.Errorf("failed to get printer: %w", err)
	}
	if printer == nil {
		return fmt.Errorf("printer not found")
	}

	// Verify spool exists
	spool, err := s.spoolRepo.GetByID(ctx, spoolID)
	if err != nil {
		return fmt.Errorf("failed to get spool: %w", err)
	}
	if spool == nil {
		return fmt.Errorf("spool not found")
	}

	// Assign resources
	job.PrinterID = &printerID
	job.MaterialSpoolID = &spoolID

	if err := s.repo.Update(ctx, job); err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	// Record assignment event
	assignedStatus := model.PrintJobStatusAssigned
	event := model.NewJobEvent(job.ID, model.JobEventAssigned, &assignedStatus).
		WithPrinter(printerID).
		WithActor(model.ActorSystem, "resource_assignment")
	if err := s.repo.AppendEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to record assignment: %w", err)
	}

	return nil
}

// ListByProject retrieves print jobs for a project.
func (s *PrintJobService) ListByProject(ctx context.Context, projectID uuid.UUID) ([]model.PrintJob, error) {
	return s.repo.ListByProject(ctx, projectID)
}

// RecordPrintingStarted records that the printer has begun printing (called by printer callbacks).
func (s *PrintJobService) RecordPrintingStarted(ctx context.Context, id uuid.UUID) error {
	job, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if job == nil {
		return fmt.Errorf("job not found")
	}

	printingStatus := model.PrintJobStatusPrinting
	event := model.NewJobEvent(job.ID, model.JobEventStarted, &printingStatus).
		WithActor(model.ActorPrinter, "")
	if job.PrinterID != nil {
		event = event.WithPrinter(*job.PrinterID).
			WithActor(model.ActorPrinter, job.PrinterID.String())
	}

	if err := s.repo.AppendEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to record printing started: %w", err)
	}

	// Update started_at if not already set
	if job.StartedAt == nil {
		now := time.Now()
		job.StartedAt = &now
		s.repo.Update(ctx, job)
	}

	s.hub.Broadcast(realtime.Event{
		Type: "job_printing",
		Data: job,
	})

	return nil
}

// RecordProgress records a progress update for a job (called by printer callbacks).
func (s *PrintJobService) RecordProgress(ctx context.Context, id uuid.UUID, progress float64) error {
	event := model.NewJobEvent(id, model.JobEventProgress, nil).
		WithProgress(progress).
		WithActor(model.ActorPrinter, "")

	return s.repo.AppendEvent(ctx, event)
}

// Pause pauses a print job and records the event.
func (s *PrintJobService) Pause(ctx context.Context, id uuid.UUID) error {
	job, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if job == nil {
		return fmt.Errorf("job not found")
	}

	if job.Status != model.PrintJobStatusPrinting {
		return fmt.Errorf("can only pause printing jobs, current status: %s", job.Status)
	}

	if job.PrinterID == nil {
		return fmt.Errorf("job has no assigned printer")
	}

	if err := s.printerMgr.PauseJob(*job.PrinterID); err != nil {
		return err
	}

	// Record paused event
	pausedStatus := model.PrintJobStatusPaused
	event := model.NewJobEvent(job.ID, model.JobEventPaused, &pausedStatus).
		WithPrinter(*job.PrinterID).
		WithActor(model.ActorUser, "")

	if err := s.repo.AppendEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to record pause: %w", err)
	}

	s.hub.Broadcast(realtime.Event{
		Type: "job_paused",
		Data: job,
	})

	return nil
}

// Resume resumes a paused print job and records the event.
func (s *PrintJobService) Resume(ctx context.Context, id uuid.UUID) error {
	job, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if job == nil {
		return fmt.Errorf("job not found")
	}

	if job.Status != model.PrintJobStatusPaused {
		return fmt.Errorf("can only resume paused jobs, current status: %s", job.Status)
	}

	if job.PrinterID == nil {
		return fmt.Errorf("job has no assigned printer")
	}

	if err := s.printerMgr.ResumeJob(*job.PrinterID); err != nil {
		return err
	}

	// Record resumed event (goes back to printing status)
	printingStatus := model.PrintJobStatusPrinting
	event := model.NewJobEvent(job.ID, model.JobEventResumed, &printingStatus).
		WithPrinter(*job.PrinterID).
		WithActor(model.ActorUser, "")

	if err := s.repo.AppendEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to record resume: %w", err)
	}

	s.hub.Broadcast(realtime.Event{
		Type: "job_resumed",
		Data: job,
	})

	return nil
}

// Cancel cancels a print job and records the event.
func (s *PrintJobService) Cancel(ctx context.Context, id uuid.UUID) error {
	job, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if job == nil {
		return fmt.Errorf("job not found")
	}

	if job.Status.IsTerminal() {
		return fmt.Errorf("cannot cancel job in %s status", job.Status)
	}

	// Try to cancel on printer (may fail if not connected, but we still record cancellation)
	if job.PrinterID != nil {
		s.printerMgr.CancelJob(*job.PrinterID)
	}

	// Record cancelled event
	cancelledStatus := model.PrintJobStatusCancelled
	event := model.NewJobEvent(job.ID, model.JobEventCancelled, &cancelledStatus).
		WithActor(model.ActorUser, "")
	if job.PrinterID != nil {
		event = event.WithPrinter(*job.PrinterID)
	}

	if err := s.repo.AppendEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to record cancellation: %w", err)
	}

	// Update completed_at
	now := time.Now()
	job.CompletedAt = &now
	failureCategory := model.FailureUserCancelled
	job.FailureCategory = &failureCategory
	s.repo.Update(ctx, job)

	s.hub.Broadcast(realtime.Event{
		Type: "job_cancelled",
		Data: job,
	})

	return nil
}

// Update updates denormalized fields on a print job.
// NOTE: For status changes, use the specific methods (Start, Pause, Cancel, RecordOutcome).
func (s *PrintJobService) Update(ctx context.Context, j *model.PrintJob) error {
	return s.repo.Update(ctx, j)
}

// RecordOutcome records the outcome of a completed print job.
// It records the completion/failure event, deducts material from the spool, and calculates costs.
func (s *PrintJobService) RecordOutcome(ctx context.Context, id uuid.UUID, outcome *model.PrintOutcome) error {
	job, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}
	if job == nil {
		return fmt.Errorf("job not found")
	}

	// Get the spool and material to calculate cost (if assigned)
	var spool *model.MaterialSpool
	var material *model.Material
	if job.MaterialSpoolID != nil {
		spool, err = s.spoolRepo.GetByID(ctx, *job.MaterialSpoolID)
		if err != nil {
			return fmt.Errorf("failed to get spool: %w", err)
		}
	}
	if spool == nil && job.MaterialSpoolID != nil {
		return fmt.Errorf("spool not found")
	}

	// Get material for cost calculation (if spool is assigned)
	if spool != nil {
		material, err = s.materialRepo.GetByID(ctx, spool.MaterialID)
		if err != nil {
			return fmt.Errorf("failed to get material: %w", err)
		}
	}

	// Calculate material cost: (grams / 1000) * cost_per_kg
	if outcome.MaterialUsed > 0 && material != nil {
		outcome.MaterialCost = (outcome.MaterialUsed / 1000.0) * material.CostPerKg
	}

	// Compute printer time cost snapshot
	var printerTimeCostCents int
	if job.PrinterID != nil {
		printerObj, _ := s.printerRepo.GetByID(ctx, *job.PrinterID)
		if printerObj != nil && printerObj.CostPerHourCents > 0 {
			var durationSeconds int
			if outcome.ActualTime != nil && *outcome.ActualTime > 0 {
				durationSeconds = *outcome.ActualTime
			} else if job.ActualSeconds != nil && *job.ActualSeconds > 0 {
				durationSeconds = *job.ActualSeconds
			} else if job.StartedAt != nil {
				durationSeconds = int(time.Since(*job.StartedAt).Seconds())
			}
			if durationSeconds > 0 {
				printerTimeCostCents = (durationSeconds * printerObj.CostPerHourCents) / 3600
			}
		}
	}

	// Update spool remaining weight if material was used
	if outcome.MaterialUsed > 0 && spool != nil {
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

	// Record the completion/failure event
	now := time.Now()
	var eventType model.JobEventType
	var status model.PrintJobStatus
	var errorCode, errorMessage string

	if outcome.Success {
		eventType = model.JobEventCompleted
		status = model.PrintJobStatusCompleted
	} else {
		eventType = model.JobEventFailed
		status = model.PrintJobStatusFailed
		errorCode = "PRINT_FAILED"
		errorMessage = outcome.FailureReason
	}

	event := model.NewJobEvent(job.ID, eventType, &status).
		WithActor(model.ActorSystem, "outcome_recorder")
	if !outcome.Success {
		event = event.WithError(errorCode, errorMessage)
	}
	if job.Progress > 0 {
		event = event.WithProgress(job.Progress)
	}

	// Add outcome data to metadata
	event.Metadata = map[string]interface{}{
		"material_used_grams": outcome.MaterialUsed,
		"material_cost":       outcome.MaterialCost,
		"quality_rating":      outcome.QualityRating,
		"actual_time":         outcome.ActualTime,
	}

	if err := s.repo.AppendEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to record outcome event: %w", err)
	}

	// Persist cost snapshots
	materialCostCents := int(outcome.MaterialCost * 100)
	totalCostCents := materialCostCents + printerTimeCostCents

	// Update job with outcome data
	job.CompletedAt = &now
	job.Outcome = outcome
	job.MaterialUsedGrams = &outcome.MaterialUsed
	job.CostCents = &totalCostCents
	job.PrinterTimeCostCents = &printerTimeCostCents
	job.MaterialCostCents = &materialCostCents
	if outcome.ActualTime != nil {
		job.ActualSeconds = outcome.ActualTime
	}

	if err := s.repo.Update(ctx, job); err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	// Broadcast the completed event
	wsEventType := "job_completed"
	if !outcome.Success {
		wsEventType = "job_failed"
	}
	s.hub.Broadcast(realtime.Event{
		Type: wsEventType,
		Data: job,
	})

	// Notify task service on successful completion
	if outcome.Success && s.onJobCompleted != nil {
		s.onJobCompleted(ctx, job)
	}

	return nil
}

// RetryRequest contains parameters for retrying a failed job.
type RetryRequest struct {
	PrinterID       *uuid.UUID       // Optional: use different printer
	MaterialSpoolID *uuid.UUID       // Optional: use different spool
	FailureCategory *model.FailureCategory // Classify why the original failed
	Notes           string           // Notes for the retry
}

// Retry creates a new job from a failed job, linking them in a retry chain.
func (s *PrintJobService) Retry(ctx context.Context, id uuid.UUID, req *RetryRequest) (*model.PrintJob, error) {
	originalJob, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get original job: %w", err)
	}
	if originalJob == nil {
		return nil, fmt.Errorf("job not found")
	}

	// Only failed jobs can be retried
	if originalJob.Status != model.PrintJobStatusFailed && originalJob.Status != model.PrintJobStatusCancelled {
		return nil, fmt.Errorf("can only retry failed or cancelled jobs, current status: %s", originalJob.Status)
	}

	// Update the original job's failure category if provided
	if req != nil && req.FailureCategory != nil {
		originalJob.FailureCategory = req.FailureCategory
		s.repo.Update(ctx, originalJob)
	}

	// Create new job as retry
	newJob := &model.PrintJob{
		DesignID:         originalJob.DesignID,
		PrinterID:        originalJob.PrinterID,
		MaterialSpoolID:  originalJob.MaterialSpoolID,
		ProjectID:        originalJob.ProjectID,
		TaskID:           originalJob.TaskID,
		Notes:            originalJob.Notes,
		RecipeID:         originalJob.RecipeID,
		EstimatedSeconds: originalJob.EstimatedSeconds,
		AttemptNumber:    originalJob.AttemptNumber + 1,
		ParentJobID:      &originalJob.ID,
	}

	// Override with retry request values
	if req != nil {
		if req.PrinterID != nil {
			newJob.PrinterID = req.PrinterID
		}
		if req.MaterialSpoolID != nil {
			newJob.MaterialSpoolID = req.MaterialSpoolID
		}
		if req.Notes != "" {
			newJob.Notes = fmt.Sprintf("%s\n[Retry] %s", originalJob.Notes, req.Notes)
		}
	}

	if err := s.repo.Create(ctx, newJob); err != nil {
		return nil, fmt.Errorf("failed to create retry job: %w", err)
	}

	// Record retried event on original job
	retriedEvent := model.NewJobEvent(originalJob.ID, model.JobEventRetried, nil).
		WithActor(model.ActorUser, "").
		WithMetadata(map[string]interface{}{
			"retry_job_id": newJob.ID.String(),
		})
	s.repo.AppendEvent(ctx, retriedEvent)

	s.hub.Broadcast(realtime.Event{
		Type: "job_retried",
		Data: map[string]interface{}{
			"original_job": originalJob,
			"new_job":      newJob,
		},
	})

	return newJob, nil
}

// RecordFailure records a failure for a job (called by printer callbacks or error handlers).
func (s *PrintJobService) RecordFailure(ctx context.Context, id uuid.UUID, category model.FailureCategory, errorCode, errorMessage string) error {
	job, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if job == nil {
		return fmt.Errorf("job not found")
	}

	if job.Status.IsTerminal() {
		return fmt.Errorf("job already in terminal status: %s", job.Status)
	}

	// Record failed event
	failedStatus := model.PrintJobStatusFailed
	event := model.NewJobEvent(job.ID, model.JobEventFailed, &failedStatus).
		WithError(errorCode, errorMessage).
		WithActor(model.ActorPrinter, job.PrinterID.String())

	if err := s.repo.AppendEvent(ctx, event); err != nil {
		return fmt.Errorf("failed to record failure: %w", err)
	}

	// Update job with failure info
	now := time.Now()
	job.CompletedAt = &now
	job.FailureCategory = &category
	s.repo.Update(ctx, job)

	s.hub.Broadcast(realtime.Event{
		Type: "job_failed",
		Data: job,
	})

	return nil
}

// Init registers the printer status change callback to auto-detect job failures.
// Call this after services are created to enable automatic failure detection.
func (s *PrintJobService) Init() {
	s.printerMgr.OnStatusChange(s.handlePrinterStatusChange)
	slog.Info("PrintJobService: registered for printer status changes")
}

// handlePrinterStatusChange is called when any printer's status changes.
// It auto-detects job failures when a printer transitions to error state.
func (s *PrintJobService) handlePrinterStatusChange(newState *model.PrinterState, oldState *model.PrinterState) {
	if newState == nil {
		return
	}

	ctx := context.Background()

	// Detect failure: printer went from printing to error
	wasPrinting := oldState != nil && oldState.Status == model.PrinterStatusPrinting
	isError := newState.Status == model.PrinterStatusError

	if wasPrinting && isError {
		slog.Warn("printer failure detected", "printer_id", newState.PrinterID, "old_status", oldState.Status, "new_status", newState.Status)

		// Find active job for this printer
		job, err := s.GetActiveJobForPrinter(ctx, newState.PrinterID)
		if err != nil {
			slog.Error("failed to find active job for failed printer", "printer_id", newState.PrinterID, "error", err)
			return
		}
		if job == nil {
			slog.Debug("no active job found for failed printer", "printer_id", newState.PrinterID)
			return
		}

		// Auto-record the failure
		category := model.FailureUnknown
		errorCode := "PRINTER_ERROR"
		errorMessage := "Printer reported error during print"

		if err := s.RecordFailure(ctx, job.ID, category, errorCode, errorMessage); err != nil {
			slog.Error("failed to auto-record job failure", "job_id", job.ID, "error", err)
		} else {
			slog.Info("auto-recorded job failure", "job_id", job.ID, "printer_id", newState.PrinterID)
		}
	}
}

// GetActiveJobForPrinter finds the currently active (printing/paused) job for a printer.
func (s *PrintJobService) GetActiveJobForPrinter(ctx context.Context, printerID uuid.UUID) (*model.PrintJob, error) {
	// Get jobs for this printer that are in active states
	printingStatus := model.PrintJobStatusPrinting
	jobs, err := s.repo.List(ctx, &printerID, &printingStatus)
	if err != nil {
		return nil, err
	}
	if len(jobs) > 0 {
		return &jobs[0], nil
	}

	// Also check paused status
	pausedStatus := model.PrintJobStatusPaused
	jobs, err = s.repo.List(ctx, &printerID, &pausedStatus)
	if err != nil {
		return nil, err
	}
	if len(jobs) > 0 {
		return &jobs[0], nil
	}

	// Check uploaded status (job sent but not confirmed printing yet)
	uploadedStatus := model.PrintJobStatusUploaded
	jobs, err = s.repo.List(ctx, &printerID, &uploadedStatus)
	if err != nil {
		return nil, err
	}
	if len(jobs) > 0 {
		return &jobs[0], nil
	}

	return nil, nil
}

// MarkAsScrap marks a failed job as scrap (no retry intended).
// This is a user action to acknowledge the failure and move on.
type ScrapRequest struct {
	FailureCategory model.FailureCategory `json:"failure_category"`
	Notes           string                `json:"notes"`
}

func (s *PrintJobService) MarkAsScrap(ctx context.Context, id uuid.UUID, req *ScrapRequest) error {
	job, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if job == nil {
		return fmt.Errorf("job not found")
	}

	// Job must be in failed state to mark as scrap
	if job.Status != model.PrintJobStatusFailed {
		return fmt.Errorf("can only mark failed jobs as scrap, current status: %s", job.Status)
	}

	// Update job with scrap info
	if req != nil && req.FailureCategory != "" {
		job.FailureCategory = &req.FailureCategory
	}
	if req != nil && req.Notes != "" {
		if job.Notes != "" {
			job.Notes = job.Notes + "\n[Marked as Scrap] " + req.Notes
		} else {
			job.Notes = "[Marked as Scrap] " + req.Notes
		}
	} else {
		if job.Notes != "" {
			job.Notes = job.Notes + "\n[Marked as Scrap]"
		} else {
			job.Notes = "[Marked as Scrap]"
		}
	}

	// Record outcome as failed
	outcome := &model.PrintOutcome{
		Success:       false,
		FailureReason: "Marked as scrap by user",
	}
	if req != nil && req.Notes != "" {
		outcome.Notes = req.Notes
	}
	job.Outcome = outcome

	if err := s.repo.Update(ctx, job); err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	s.hub.Broadcast(realtime.Event{
		Type: "job_updated",
		Data: job,
	})

	return nil
}

// UpdatePriority updates a job's priority in the queue.
func (s *PrintJobService) UpdatePriority(ctx context.Context, id uuid.UUID, priority int) error {
	return s.repo.UpdatePriority(ctx, id, priority)
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
	settingsRepo *repository.SettingsRepository
	repos        *repository.Repositories // For transaction support
	storage      storage.Storage
	parser       *receipt.Parser
}

// initParser initializes the receipt parser, reading the API key from
// the settings DB first, then falling back to the ANTHROPIC_API_KEY env var.
func (s *ExpenseService) initParser(ctx context.Context) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if s.settingsRepo != nil {
		if setting, err := s.settingsRepo.Get(ctx, "anthropic_api_key"); err == nil && setting != nil && setting.Value != "" {
			apiKey = setting.Value
		}
	}
	s.parser = receipt.NewParserWithKey(apiKey)
}

// UploadReceipt uploads a receipt file and starts AI parsing.
func (s *ExpenseService) UploadReceipt(ctx context.Context, filename string, data []byte) (*model.Expense, error) {
	// Initialize parser lazily
	if s.parser == nil {
		s.initParser(ctx)
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

	// Parse receipt asynchronously if API key is configured
	if s.parser.HasAPIKey() {
		go func() {
			parseCtx := context.Background()
			s.parseReceiptAsync(parseCtx, expense.ID, storagePath, data)
		}()
	}

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
		// Surface the error to the user by updating the expense record
		if expense, getErr := s.repo.GetByID(ctx, expenseID); getErr == nil && expense != nil {
			expense.Notes = fmt.Sprintf("Parse failed: %s", err.Error())
			expense.Status = model.ExpenseStatusRejected
			_ = s.repo.Update(ctx, expense)
		}
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

	// Create expense items and auto-create materials + spools for filament
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
			continue
		}

		// Auto-create material + spools for filament items
		if item.IsFilament && item.Filament != nil {
			materialID, err := s.findOrCreateMaterial(ctx, item.Filament, item.UnitPriceCents)
			if err != nil {
				slog.Error("failed to find/create material", "expense_id", expenseID, "description", item.Description, "error", err)
				continue
			}

			weightGrams := item.Filament.WeightGrams
			if weightGrams == 0 {
				weightGrams = 1000
			}

			quantity := int(item.Quantity)
			if quantity < 1 {
				quantity = 1
			}

			for i := 0; i < quantity; i++ {
				spool := &model.MaterialSpool{
					MaterialID:      materialID,
					InitialWeight:   weightGrams,
					RemainingWeight: weightGrams,
					PurchaseDate:    &expense.OccurredAt,
					PurchaseCost:    float64(item.TotalPriceCents) / 100.0 / float64(quantity),
					Status:          model.SpoolStatusNew,
					Notes:           fmt.Sprintf("From receipt: %s", expense.Vendor),
				}

				if err := s.spoolRepo.Create(ctx, spool); err != nil {
					slog.Error("failed to create spool", "expense_id", expenseID, "error", err)
					continue
				}

				if i == 0 {
					expenseItem.MatchedSpoolID = &spool.ID
					expenseItem.MatchedMaterialID = &materialID
					expenseItem.ActionTaken = model.ExpenseItemActionCreatedSpool
				}
			}

			if err := s.repo.UpdateItem(ctx, expenseItem); err != nil {
				slog.Error("failed to update expense item", "expense_id", expenseID, "error", err)
			}

			slog.Info("auto-created material + spools", "expense_id", expenseID, "material_id", materialID, "quantity", quantity, "description", item.Description)
		} else if !item.IsFilament && item.Category != "shipping" {
			// Auto-create supply material for non-filament, non-shipping items
			materialID, err := s.findOrCreateSupplyMaterial(ctx, item.Description, parsed.Vendor, item.UnitPriceCents)
			if err != nil {
				slog.Error("failed to find/create supply material", "expense_id", expenseID, "description", item.Description, "error", err)
				continue
			}

			expenseItem.MatchedMaterialID = &materialID
			expenseItem.ActionTaken = model.ExpenseItemActionCreatedSupply

			if err := s.repo.UpdateItem(ctx, expenseItem); err != nil {
				slog.Error("failed to update expense item with supply material", "expense_id", expenseID, "error", err)
			}

			slog.Info("auto-created supply material", "expense_id", expenseID, "material_id", materialID, "description", item.Description)
		}
	}

	// Auto-confirm the expense since materials were processed
	expense.Status = model.ExpenseStatusConfirmed
	if err := s.repo.Update(ctx, expense); err != nil {
		slog.Error("failed to auto-confirm expense", "expense_id", expenseID, "error", err)
	}

	slog.Info("expense auto-confirmed", "expense_id", expenseID, "items", len(parsed.Items))
}

// findOrCreateMaterial finds an existing material matching the filament metadata,
// or creates a new one. Returns the material ID.
// filamentColorHex maps common filament color names to hex values.
var filamentColorHex = map[string]string{
	"black":          "#000000",
	"white":          "#FFFFFF",
	"red":            "#FF0000",
	"blue":           "#0000FF",
	"green":          "#008000",
	"yellow":         "#FFFF00",
	"orange":         "#FF8C00",
	"purple":         "#800080",
	"pink":           "#FF69B4",
	"gray":           "#808080",
	"grey":           "#808080",
	"silver":         "#C0C0C0",
	"gold":           "#FFD700",
	"brown":          "#8B4513",
	"beige":          "#F5DEB3",
	"ivory":          "#FFFFF0",
	"cream":          "#FFFDD0",
	"cyan":           "#00FFFF",
	"magenta":        "#FF00FF",
	"teal":           "#008080",
	"navy":           "#000080",
	"olive":          "#808000",
	"maroon":         "#800000",
	"coral":          "#FF7F50",
	"salmon":         "#FA8072",
	"turquoise":      "#40E0D0",
	"lavender":       "#E6E6FA",
	"lilac":          "#C8A2C8",
	"mint":           "#3EB489",
	"jade":           "#00A86B",
	"transparent":    "#E0E0E0",
	"natural":        "#F5F0E1",
	"matte black":    "#1A1A1A",
	"matte white":    "#F0F0F0",
	"charcoal":       "#36454F",
	"dark grey":      "#555555",
	"dark gray":      "#555555",
	"light grey":     "#BBBBBB",
	"light gray":     "#BBBBBB",
	"dark blue":      "#00008B",
	"light blue":     "#ADD8E6",
	"sky blue":       "#87CEEB",
	"dark green":     "#006400",
	"light green":    "#90EE90",
	"dark red":       "#8B0000",
	"bambu green":    "#00AE42",
	"jade white":     "#E8E0D8",
	"arctic blue":    "#6CB4EE",
}

func (s *ExpenseService) findOrCreateMaterial(ctx context.Context, fm *model.FilamentMetadata, unitPriceCents int) (uuid.UUID, error) {
	matType := model.MaterialType(strings.ToLower(fm.MaterialType))
	manufacturer := fm.Brand
	color := fm.Color

	// Try to find existing material
	existing, err := s.materialRepo.FindByTypeManufacturerColor(ctx, matType, manufacturer, color)
	if err != nil {
		return uuid.Nil, fmt.Errorf("find material: %w", err)
	}
	if existing != nil {
		return existing.ID, nil
	}

	// Build a descriptive name
	name := manufacturer
	if name != "" {
		name += " "
	}
	name += strings.ToUpper(string(matType))
	if color != "" {
		name += " - " + color
	}

	// Calculate cost per kg from the unit price
	weightKg := fm.WeightGrams / 1000.0
	if weightKg <= 0 {
		weightKg = 1.0
	}
	costPerKg := float64(unitPriceCents) / 100.0 / weightKg

	diameterMM := fm.DiameterMM
	if diameterMM == 0 {
		diameterMM = 1.75
	}

	// Resolve color hex: use AI-provided value, then fallback to color name lookup
	colorHex := fm.ColorHex
	if colorHex == "" && color != "" {
		colorHex = filamentColorHex[strings.ToLower(color)]
	}

	mat := &model.Material{
		Name:         name,
		Type:         matType,
		Manufacturer: manufacturer,
		Color:        color,
		ColorHex:     colorHex,
		Density:      1.24, // reasonable default for PLA/PETG
		CostPerKg:    costPerKg,
	}

	if err := s.materialRepo.Create(ctx, mat); err != nil {
		return uuid.Nil, fmt.Errorf("create material: %w", err)
	}

	return mat.ID, nil
}

// findOrCreateSupplyMaterial finds or creates a supply-type material for non-filament receipt items.
func (s *ExpenseService) findOrCreateSupplyMaterial(ctx context.Context, description string, vendor string, unitPriceCents int) (uuid.UUID, error) {
	// Try to find existing supply material by type + name
	existing, err := s.materialRepo.FindByTypeAndName(ctx, model.MaterialTypeSupply, description)
	if err != nil {
		return uuid.Nil, fmt.Errorf("find supply material: %w", err)
	}
	if existing != nil {
		return existing.ID, nil
	}

	// Create new supply material
	mat := &model.Material{
		Name:         description,
		Type:         model.MaterialTypeSupply,
		Manufacturer: vendor,
		CostPerKg:    float64(unitPriceCents) / 100.0, // repurposed as per-unit cost
	}

	if err := s.materialRepo.Create(ctx, mat); err != nil {
		return uuid.Nil, fmt.Errorf("create supply material: %w", err)
	}

	return mat.ID, nil
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
// All database operations are wrapped in a transaction for atomicity.
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

	// Execute all inventory changes within a transaction
	return s.repos.WithTransaction(ctx, func(tx *sql.Tx) error {
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
					if err := s.materialRepo.CreateTx(ctx, tx, confirmItem.NewMaterial); err != nil {
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

					if err := s.spoolRepo.CreateTx(ctx, tx, spool); err != nil {
						return fmt.Errorf("failed to create spool: %w", err)
					}

					// Update expense item with matched spool
					if i == 0 {
						expenseItem.MatchedSpoolID = &spool.ID
						expenseItem.MatchedMaterialID = &materialID
						expenseItem.ActionTaken = model.ExpenseItemActionCreatedSpool
					}
				}

				if err := s.repo.UpdateItemTx(ctx, tx, expenseItem); err != nil {
					return fmt.Errorf("failed to update expense item: %w", err)
				}
			} else {
				expenseItem.ActionTaken = model.ExpenseItemActionSkipped
				if err := s.repo.UpdateItemTx(ctx, tx, expenseItem); err != nil {
					return fmt.Errorf("failed to update expense item: %w", err)
				}
			}
		}

		// Mark expense as confirmed
		expense.Status = model.ExpenseStatusConfirmed
		if err := s.repo.UpdateTx(ctx, tx, expense); err != nil {
			return fmt.Errorf("failed to confirm expense: %w", err)
		}

		return nil
	})
}

// RetryParse re-reads the stored receipt file and re-triggers AI parsing.
func (s *ExpenseService) RetryParse(ctx context.Context, id uuid.UUID) (*model.Expense, error) {
	// Re-initialize parser on retry so it picks up any new API key from settings
	s.initParser(ctx)

	expense, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if expense == nil {
		return nil, fmt.Errorf("expense not found")
	}
	if expense.ReceiptFilePath == "" {
		return nil, fmt.Errorf("no receipt file stored for this expense")
	}

	// Read the file back from storage
	reader, err := s.storage.Get(expense.ReceiptFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read stored receipt: %w", err)
	}
	data, err := io.ReadAll(reader)
	reader.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read stored receipt: %w", err)
	}

	// Delete any old expense items from a previous parse attempt
	_ = s.repo.DeleteItemsByExpenseID(ctx, id)

	// Reset expense to pending state
	expense.Status = model.ExpenseStatusPending
	expense.Notes = ""
	expense.Vendor = ""
	expense.SubtotalCents = 0
	expense.TaxCents = 0
	expense.ShippingCents = 0
	expense.TotalCents = 0
	expense.Confidence = 0
	expense.RawOCRText = ""
	expense.RawAIResponse = nil
	expense.Category = model.ExpenseCategoryOther
	if err := s.repo.Update(ctx, expense); err != nil {
		return nil, fmt.Errorf("failed to reset expense: %w", err)
	}

	// Re-trigger parsing if API key is configured
	if s.parser.HasAPIKey() {
		go func() {
			parseCtx := context.Background()
			s.parseReceiptAsync(parseCtx, expense.ID, expense.ReceiptFilePath, data)
		}()
	}

	return expense, nil
}

// Delete deletes an expense.
func (s *ExpenseService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// SaleService handles sales and revenue tracking.
type SaleService struct {
	repo     *repository.SaleRepository
	taskRepo *repository.TaskRepository
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

// WeekSummary holds aggregated sales totals for a single week.
type WeekSummary struct {
	GrossCents int `json:"gross_cents"`
	NetCents   int `json:"net_cents"`
	FeesCents  int `json:"fees_cents"`
	Count      int `json:"count"`
}

// WeeklyInsights holds this-week vs last-week comparison data.
type WeeklyInsights struct {
	ThisWeek            WeekSummary        `json:"this_week"`
	LastWeek            WeekSummary        `json:"last_week"`
	Channels            []ChannelBreakdown `json:"channels"`
	WeekStart           string             `json:"week_start"`
	WeekEnd             string             `json:"week_end"`
	PendingCount        int                `json:"pending_count"`
	PendingRevenueCents int                `json:"pending_revenue_cents"`
}

// GetWeeklyInsights returns this week's sales metrics with last week comparison.
func (s *SaleService) GetWeeklyInsights(ctx context.Context) (*WeeklyInsights, error) {
	now := time.Now().UTC()

	// Monday of this week
	weekday := now.Weekday()
	if weekday == time.Sunday {
		weekday = 7
	}
	thisMonday := time.Date(now.Year(), now.Month(), now.Day()-int(weekday-time.Monday), 0, 0, 0, 0, time.UTC)
	thisSunday := thisMonday.AddDate(0, 0, 7) // exclusive end

	lastMonday := thisMonday.AddDate(0, 0, -7)

	// This week totals
	twGross, twNet, twFees, twCount, err := s.repo.GetTotalsByDateRange(ctx, thisMonday, thisSunday)
	if err != nil {
		return nil, fmt.Errorf("this week totals: %w", err)
	}

	// Last week totals
	lwGross, lwNet, lwFees, lwCount, err := s.repo.GetTotalsByDateRange(ctx, lastMonday, thisMonday)
	if err != nil {
		return nil, fmt.Errorf("last week totals: %w", err)
	}

	// Channel breakdown for this week
	channelRows, err := s.repo.GetSalesByChannel(ctx, thisMonday)
	if err != nil {
		return nil, fmt.Errorf("channel breakdown: %w", err)
	}
	channels := make([]ChannelBreakdown, len(channelRows))
	for i, r := range channelRows {
		channels[i] = ChannelBreakdown{Channel: r.Channel, Total: r.Total, Count: r.Count}
	}

	// Pending sales: tasks in pending/in_progress linked to priced projects
	var pendingCount, pendingRevenue int
	if s.taskRepo != nil {
		pendingCount, pendingRevenue, err = s.taskRepo.GetPendingSalesStats(ctx)
		if err != nil {
			return nil, fmt.Errorf("pending sales: %w", err)
		}
	}

	return &WeeklyInsights{
		ThisWeek:            WeekSummary{GrossCents: twGross, NetCents: twNet, FeesCents: twFees, Count: twCount},
		LastWeek:            WeekSummary{GrossCents: lwGross, NetCents: lwNet, FeesCents: lwFees, Count: lwCount},
		Channels:            channels,
		WeekStart:           thisMonday.Format("2006-01-02"),
		WeekEnd:             thisSunday.AddDate(0, 0, -1).Format("2006-01-02"),
		PendingCount:        pendingCount,
		PendingRevenueCents: pendingRevenue,
	}, nil
}

// FinancialSummary contains aggregated financial data.
type FinancialSummary struct {
	TotalExpensesCents     int     `json:"total_expenses_cents"`
	TotalSalesGrossCents   int     `json:"total_sales_gross_cents"`
	TotalSalesNetCents     int     `json:"total_sales_net_cents"`
	TotalFeesCents         int     `json:"total_fees_cents"`
	TotalMaterialCost      float64 `json:"total_material_cost"`
	TotalMaterialUsedGrams float64 `json:"total_material_used_grams"`
	TotalCOGSCents         int     `json:"total_cogs_cents"`
	NetProfitCents         int     `json:"net_profit_cents"`
	ConfirmedExpenseCount  int     `json:"confirmed_expense_count"`
	PendingExpenseCount    int     `json:"pending_expense_count"`
	SalesCount             int     `json:"sales_count"`
	CompletedPrintCount    int     `json:"completed_print_count"`
	SuccessfulPrintCount   int     `json:"successful_print_count"`
}

// StatsService handles financial statistics and aggregations.
type StatsService struct {
	expenseRepo    *repository.ExpenseRepository
	saleRepo       *repository.SaleRepository
	printJobRepo   *repository.PrintJobRepository
	projectService *ProjectService
}

// GetFinancialSummary returns aggregated financial data.
// If since is non-nil, only data from that time onward is included.
func (s *StatsService) GetFinancialSummary(ctx context.Context, since *time.Time) (*FinancialSummary, error) {
	summary := &FinancialSummary{}

	// Get expense totals
	confirmedStatus := model.ExpenseStatusConfirmed
	var expenses []model.Expense
	var err error
	if since != nil {
		expenses, err = s.expenseRepo.ListSince(ctx, &confirmedStatus, *since)
	} else {
		expenses, err = s.expenseRepo.List(ctx, &confirmedStatus)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get expenses: %w", err)
	}
	for _, exp := range expenses {
		summary.TotalExpensesCents += exp.TotalCents
		summary.ConfirmedExpenseCount++
	}

	// Get pending expense count
	pendingStatus := model.ExpenseStatusPending
	var pendingExpenses []model.Expense
	if since != nil {
		pendingExpenses, err = s.expenseRepo.ListSince(ctx, &pendingStatus, *since)
	} else {
		pendingExpenses, err = s.expenseRepo.List(ctx, &pendingStatus)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get pending expenses: %w", err)
	}
	summary.PendingExpenseCount = len(pendingExpenses)

	// Get sales totals
	var sales []model.Sale
	if since != nil {
		sales, err = s.saleRepo.ListSince(ctx, *since)
	} else {
		sales, err = s.saleRepo.List(ctx, nil)
	}
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
	var jobs []model.PrintJob
	if since != nil {
		jobs, err = s.printJobRepo.ListCompletedSince(ctx, *since)
	} else {
		completedStatus := model.PrintJobStatusCompleted
		jobs, err = s.printJobRepo.List(ctx, nil, &completedStatus)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get print jobs: %w", err)
	}
	for _, job := range jobs {
		summary.CompletedPrintCount++
		if job.Outcome != nil {
			if job.Outcome.Success {
				summary.SuccessfulPrintCount++
			}
		}
	}

	// Aggregate material and cost data from sales-linked project summaries
	if s.projectService != nil {
		projectIDs := make(map[uuid.UUID]bool)
		for _, sale := range sales {
			if sale.ProjectID != nil {
				projectIDs[*sale.ProjectID] = true
			}
		}

		for pid := range projectIDs {
			ps, err := s.projectService.GetProjectSummary(ctx, pid)
			if err != nil || ps == nil {
				continue
			}
			// Material grams: actual from jobs + estimated from slice profiles
			summary.TotalMaterialUsedGrams += ps.TotalMaterialGrams + ps.EstimatedMaterialGrams
			// Material cost: actual from jobs + estimated from profiles
			summary.TotalMaterialCost += float64(ps.MaterialCostCents+ps.EstimatedMaterialCostCents) / 100.0
			// Total COGS includes material + printer time + supplies (already × sales count)
			summary.TotalCOGSCents += ps.TotalCostCents
		}
	}

	// Net profit: sales net revenue - total COGS
	summary.NetProfitCents = summary.TotalSalesNetCents - summary.TotalCOGSCents

	return summary, nil
}

// TimeSeriesPoint represents a single data point in a time series.
type TimeSeriesPoint struct {
	Date     string `json:"date"`
	Revenue  int    `json:"revenue"`
	Expenses int    `json:"expenses"`
	Profit   int    `json:"profit"`
}

// TimeSeriesData represents the full time series response.
type TimeSeriesData struct {
	Points []TimeSeriesPoint `json:"points"`
	Period string            `json:"period"`
}

// CategoryBreakdown represents an expense category breakdown.
type CategoryBreakdown struct {
	Category string `json:"category"`
	Total    int    `json:"total"`
	Count    int    `json:"count"`
}

// ChannelBreakdown represents a sales channel breakdown.
type ChannelBreakdown struct {
	Channel string `json:"channel"`
	Total   int    `json:"total"`
	Count   int    `json:"count"`
}

// parsePeriod converts a period string to (since time, strftime format).
func parsePeriod(period string) (time.Time, string) {
	now := time.Now()
	switch period {
	case "90d":
		return now.AddDate(0, 0, -90), "%Y-W%W"
	case "12m":
		return now.AddDate(-1, 0, 0), "%Y-%m"
	default: // "30d"
		return now.AddDate(0, 0, -30), "%Y-%m-%d"
	}
}

// GetTimeSeriesData returns aligned revenue, expenses, and profit time series data.
func (s *StatsService) GetTimeSeriesData(ctx context.Context, period string) (*TimeSeriesData, error) {
	since, strftimeFmt := parsePeriod(period)

	revenueSeries, err := s.saleRepo.GetSalesOverTime(ctx, since, strftimeFmt)
	if err != nil {
		return nil, fmt.Errorf("failed to get revenue series: %w", err)
	}

	expenseSeries, err := s.expenseRepo.GetExpensesOverTime(ctx, since, strftimeFmt)
	if err != nil {
		return nil, fmt.Errorf("failed to get expense series: %w", err)
	}

	// Merge into aligned date buckets
	allDates := make(map[string]bool)
	revenueMap := make(map[string]int)
	expenseMap := make(map[string]int)

	for _, r := range revenueSeries {
		allDates[r.DateBucket] = true
		revenueMap[r.DateBucket] = r.Total
	}
	for _, e := range expenseSeries {
		allDates[e.DateBucket] = true
		expenseMap[e.DateBucket] = e.Total
	}

	// Sort dates
	dates := make([]string, 0, len(allDates))
	for d := range allDates {
		dates = append(dates, d)
	}
	sortStrings(dates)

	points := make([]TimeSeriesPoint, len(dates))
	for i, d := range dates {
		rev := revenueMap[d]
		exp := expenseMap[d]
		points[i] = TimeSeriesPoint{
			Date:     d,
			Revenue:  rev,
			Expenses: exp,
			Profit:   rev - exp,
		}
	}

	return &TimeSeriesData{
		Points: points,
		Period: period,
	}, nil
}

// GetExpensesByCategory returns expense totals grouped by category.
func (s *StatsService) GetExpensesByCategory(ctx context.Context, period string) ([]CategoryBreakdown, error) {
	since, _ := parsePeriod(period)
	rows, err := s.expenseRepo.GetExpensesByCategory(ctx, since)
	if err != nil {
		return nil, err
	}

	result := make([]CategoryBreakdown, len(rows))
	for i, r := range rows {
		result[i] = CategoryBreakdown{
			Category: r.Category,
			Total:    r.Total,
			Count:    r.Count,
		}
	}
	return result, nil
}

// GetSalesByChannel returns sales totals grouped by channel.
func (s *StatsService) GetSalesByChannel(ctx context.Context, period string) ([]ChannelBreakdown, error) {
	since, _ := parsePeriod(period)
	rows, err := s.saleRepo.GetSalesByChannel(ctx, since)
	if err != nil {
		return nil, err
	}

	result := make([]ChannelBreakdown, len(rows))
	for i, r := range rows {
		result[i] = ChannelBreakdown{
			Channel: r.Channel,
			Total:   r.Total,
			Count:   r.Count,
		}
	}
	return result, nil
}

// ProjectSales represents aggregated sales data for a project.
type ProjectSales struct {
	ProjectID            string `json:"project_id"`
	ProjectName          string `json:"project_name"`
	GrossCents           int    `json:"gross_cents"`
	NetCents             int    `json:"net_cents"`
	Count                int    `json:"count"`
	AvgCents             int    `json:"avg_cents"`
	UnitCostCents        int    `json:"unit_cost_cents"`
	TotalCOGS            int    `json:"total_cogs_cents"`
	ProfitCents          int    `json:"profit_cents"`
	EstimatedPrintSeconds int   `json:"estimated_print_seconds"`
	TotalPrintSeconds    int    `json:"total_print_seconds"`
	FirstSale            string `json:"first_sale"`
	LastSale             string `json:"last_sale"`
}

// GetSalesByProject returns sales aggregated by project.
func (s *StatsService) GetSalesByProject(ctx context.Context) ([]ProjectSales, error) {
	rows, err := s.saleRepo.GetSalesByProject(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]ProjectSales, len(rows))
	for i, r := range rows {
		ps := ProjectSales{
			ProjectID:   r.ProjectID,
			ProjectName: r.ProjectName,
			GrossCents:  r.GrossCents,
			NetCents:    r.NetCents,
			Count:       r.Count,
			AvgCents:    r.AvgCents,
			FirstSale:   r.FirstSale,
			LastSale:    r.LastSale,
		}

		// Enrich with cost data from project summary
		if s.projectService != nil && r.ProjectID != "" {
			projectID, err := uuid.Parse(r.ProjectID)
			if err == nil {
				summary, err := s.projectService.GetProjectSummary(ctx, projectID)
				if err == nil && summary != nil {
					ps.UnitCostCents = summary.UnitCostCents
					ps.TotalCOGS = summary.TotalCostCents
					ps.ProfitCents = ps.NetCents - ps.TotalCOGS
					ps.EstimatedPrintSeconds = summary.EstimatedPrintSeconds
					ps.TotalPrintSeconds = summary.TotalPrintSeconds
				}
			}
		}

		result[i] = ps
	}
	return result, nil
}

// sortStrings sorts a slice of strings in ascending order.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

// TemplateService handles template business logic.
type TemplateService struct {
	repo           *repository.TemplateRepository
	projectRepo    *repository.ProjectRepository
	partRepo       *repository.PartRepository
	designRepo     *repository.DesignRepository
	printJobRepo   *repository.PrintJobRepository
	spoolRepo      *repository.SpoolRepository
	materialRepo   *repository.MaterialRepository
	printerRepo    *repository.PrinterRepository
	projectService *ProjectService
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

// GetByID retrieves a template by ID with its designs, materials, and supplies.
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

	// Load materials
	materials, err := s.repo.GetRecipeMaterials(ctx, id)
	if err != nil {
		return nil, err
	}
	t.Materials = materials

	// Load supplies
	supplies, err := s.repo.GetRecipeSupplies(ctx, id)
	if err != nil {
		return nil, err
	}
	t.Supplies = supplies

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
					DesignID:  td.DesignID,
					ProjectID: &project.ID,
					RecipeID:  &templateID,
					Status:    model.PrintJobStatusQueued,
					Notes:     fmt.Sprintf("Part %d/%d for order", i+1, totalParts),
				}

				// Assign preferred printer if specified and not allowing any printer
				if template.PreferredPrinterID != nil && !template.AllowAnyPrinter {
					job.PrinterID = template.PreferredPrinterID
				}

				// Assign material spool if specified in options
				if opts.MaterialSpoolID != nil {
					job.MaterialSpoolID = opts.MaterialSpoolID
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

// GetByIDWithMaterials retrieves a template with its materials and supplies loaded.
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

	// Load supplies
	supplies, err := s.repo.GetRecipeSupplies(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("loading supplies: %w", err)
	}
	t.Supplies = supplies

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

// AddSupply adds a supply item to a recipe.
func (s *TemplateService) AddSupply(ctx context.Context, supply *model.RecipeSupply) error {
	if supply.RecipeID == uuid.Nil {
		return fmt.Errorf("recipe ID is required")
	}
	// If material_id is set and name is empty, auto-populate from the material
	if supply.MaterialID != nil && *supply.MaterialID != uuid.Nil && supply.Name == "" {
		mat, err := s.materialRepo.GetByID(ctx, *supply.MaterialID)
		if err != nil {
			return fmt.Errorf("failed to look up material: %w", err)
		}
		if mat != nil {
			supply.Name = mat.Name
			if supply.UnitCostCents == 0 {
				supply.UnitCostCents = int(mat.CostPerKg * 100) // CostPerKg repurposed as per-unit $ for supplies
			}
		}
	}
	if supply.Name == "" {
		return fmt.Errorf("supply name is required")
	}
	if supply.Quantity < 1 {
		supply.Quantity = 1
	}
	return s.repo.AddRecipeSupply(ctx, supply)
}

// UpdateSupply updates a supply item.
func (s *TemplateService) UpdateSupply(ctx context.Context, supply *model.RecipeSupply) error {
	return s.repo.UpdateRecipeSupply(ctx, supply)
}

// RemoveSupply removes a supply item from a recipe.
func (s *TemplateService) RemoveSupply(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteRecipeSupply(ctx, id)
}

// GetSupply retrieves a single supply by ID.
func (s *TemplateService) GetSupply(ctx context.Context, id uuid.UUID) (*model.RecipeSupply, error) {
	return s.repo.GetRecipeSupplyByID(ctx, id)
}

// ListSupplies retrieves all supplies for a recipe.
func (s *TemplateService) ListSupplies(ctx context.Context, recipeID uuid.UUID) ([]model.RecipeSupply, error) {
	return s.repo.GetRecipeSupplies(ctx, recipeID)
}

// GetTemplateAnalytics returns aggregated performance metrics from all projects created from a template.
func (s *TemplateService) GetTemplateAnalytics(ctx context.Context, templateID uuid.UUID) (*model.TemplateAnalytics, error) {
	// Verify template exists
	template, err := s.GetByID(ctx, templateID)
	if err != nil {
		return nil, err
	}
	if template == nil {
		return nil, fmt.Errorf("template not found")
	}

	// Get all projects linked to this template
	projects, err := s.projectRepo.ListByTemplateID(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("listing projects: %w", err)
	}

	analytics := &model.TemplateAnalytics{
		TemplateID:             templateID,
		ProjectCount:           len(projects),
		EstimatedPrintSeconds:  template.EstimatedPrintSeconds,
		EstimatedMaterialGrams: template.EstimatedMaterialGrams,
	}

	if len(projects) == 0 {
		return analytics, nil
	}

	// Aggregate summaries from each project
	var marginSum float64
	var marginCount int
	for _, proj := range projects {
		if s.projectService == nil {
			continue
		}
		summary, err := s.projectService.GetProjectSummary(ctx, proj.ID)
		if err != nil || summary == nil {
			continue
		}

		analytics.TotalRevenueCents += summary.TotalRevenueCents
		analytics.TotalFeesCents += summary.TotalFeesCents
		analytics.NetRevenueCents += summary.NetRevenueCents
		analytics.TotalSalesCount += summary.SalesCount

		analytics.TotalCostCents += summary.TotalCostCents
		analytics.TotalPrinterTimeCost += summary.PrinterTimeCostCents
		analytics.TotalMaterialCost += summary.MaterialCostCents
		analytics.TotalSupplyCost += summary.SupplyCostCents

		analytics.TotalJobCount += summary.JobCount
		analytics.TotalCompleted += summary.CompletedCount
		analytics.TotalFailed += summary.FailedCount

		analytics.TotalPrintSeconds += summary.TotalPrintSeconds
		analytics.TotalMaterialGrams += summary.TotalMaterialGrams

		if summary.GrossMarginPercent != 0 {
			marginSum += summary.GrossMarginPercent
			marginCount++
		}
	}

	// Derived metrics
	if analytics.TotalCompleted+analytics.TotalFailed > 0 {
		analytics.SuccessRate = float64(analytics.TotalCompleted) / float64(analytics.TotalCompleted+analytics.TotalFailed) * 100
	}
	if analytics.TotalCompleted > 0 {
		analytics.AvgPrintSeconds = analytics.TotalPrintSeconds / analytics.TotalCompleted
	}
	if analytics.ProjectCount > 0 {
		analytics.AvgUnitCostCents = analytics.TotalCostCents / analytics.ProjectCount
		analytics.AvgMaterialGrams = analytics.TotalMaterialGrams / float64(analytics.ProjectCount)
	}
	if marginCount > 0 {
		analytics.AvgGrossMarginPercent = marginSum / float64(marginCount)
	}

	analytics.TotalGrossProfitCents = analytics.NetRevenueCents - analytics.TotalCostCents

	// Profit per hour across all projects
	if analytics.TotalPrintSeconds > 0 {
		hours := float64(analytics.TotalPrintSeconds) / 3600.0
		analytics.ProfitPerHourCents = int(float64(analytics.TotalGrossProfitCents) / hours)
	}

	// Get estimated cost for comparison
	costEstimate, err := s.CalculateRecipeCost(ctx, templateID)
	if err == nil && costEstimate != nil {
		analytics.EstimatedCostCents = costEstimate.TotalCostCents
	}

	return analytics, nil
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

// DefaultLaborRateCents is the default manual labor cost per hour in cents.
const DefaultLaborRateCents = 1500 // $15.00/hour

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

	// Determine hourly rate: use preferred printer's rate if available
	hourlyRateCents := DefaultHourlyRateCents
	printerName := ""
	if template.PreferredPrinterID != nil {
		p, _ := s.printerRepo.GetByID(ctx, *template.PreferredPrinterID)
		if p != nil && p.CostPerHourCents > 0 {
			hourlyRateCents = p.CostPerHourCents
			printerName = p.Name
		}
	}

	estimate := &model.RecipeCostEstimate{
		EstimatedPrintTime: template.EstimatedPrintSeconds,
		HourlyRateCents:    hourlyRateCents,
		LaborRateCents:     DefaultLaborRateCents,
		LaborMinutes:       template.LaborMinutes,
		SalePriceCents:     template.SalePriceCents,
		PrinterName:        printerName,
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

	// Calculate time cost (machine time) using actual printer rate
	if template.EstimatedPrintSeconds > 0 {
		hours := float64(template.EstimatedPrintSeconds) / 3600.0
		estimate.TimeCostCents = int(hours * float64(hourlyRateCents))
	}

	// Calculate labor cost (manual labor time)
	if template.LaborMinutes > 0 {
		hours := float64(template.LaborMinutes) / 60.0
		estimate.LaborCostCents = int(hours * float64(DefaultLaborRateCents))
	}

	// Calculate supply costs
	for _, supply := range template.Supplies {
		totalCents := supply.UnitCostCents * supply.Quantity
		estimate.SupplyCostCents += totalCents
		estimate.SupplyBreakdown = append(estimate.SupplyBreakdown, model.RecipeSupplyCostBreakdown{
			Name:          supply.Name,
			UnitCostCents: supply.UnitCostCents,
			Quantity:      supply.Quantity,
			TotalCents:    totalCents,
		})
	}

	// Total cost = material + machine time + labor + supplies
	estimate.TotalCostCents = estimate.MaterialCostCents + estimate.TimeCostCents + estimate.LaborCostCents + estimate.SupplyCostCents

	// Calculate gross margin if sale price is set
	if template.SalePriceCents > 0 {
		estimate.GrossMarginCents = template.SalePriceCents - estimate.TotalCostCents
		estimate.GrossMarginPercent = float64(estimate.GrossMarginCents) / float64(template.SalePriceCents) * 100.0

		// Profit per hour: margin / estimated print hours
		if template.EstimatedPrintSeconds > 0 {
			hours := float64(template.EstimatedPrintSeconds) / 3600.0
			estimate.ProfitPerHourCents = int(float64(estimate.GrossMarginCents) / hours)
		}
	}

	return estimate, nil
}

// SettingsService handles application settings.
type SettingsService struct {
	repo *repository.SettingsRepository
}

// sensitiveKeys lists settings that should be encrypted at rest.
var sensitiveKeys = map[string]bool{
	"anthropic_api_key":       true,
	"etsy_client_id":          true,
	"etsy_access_token":       true,
	"etsy_refresh_token":      true,
	"bambu_cloud_token":       true,
	"bambu_cloud_password":    true,
}

// isSensitive checks if a key should be encrypted.
func isSensitive(key string) bool {
	return sensitiveKeys[key]
}

// Get retrieves a setting by key, decrypting if necessary.
func (s *SettingsService) Get(ctx context.Context, key string) (*repository.Setting, error) {
	setting, err := s.repo.Get(ctx, key)
	if err != nil || setting == nil {
		return setting, err
	}

	// Decrypt sensitive values
	if isSensitive(key) && crypto.IsEncrypted(setting.Value) {
		decrypted, err := crypto.Decrypt(setting.Value)
		if err != nil {
			slog.Warn("failed to decrypt setting", "key", key, "error", err)
			// Return original value if decryption fails (might be unencrypted legacy data)
			return setting, nil
		}
		setting.Value = decrypted
	}

	return setting, nil
}

// Set creates or updates a setting, encrypting sensitive values.
func (s *SettingsService) Set(ctx context.Context, key, value string) error {
	// Encrypt sensitive values
	if isSensitive(key) && value != "" {
		encrypted, err := crypto.Encrypt(value)
		if err != nil {
			slog.Warn("failed to encrypt setting, storing unencrypted", "key", key, "error", err)
			// Fall back to storing unencrypted if encryption fails
		} else {
			value = encrypted
		}
	}

	return s.repo.Set(ctx, key, value)
}

// List retrieves all settings, decrypting sensitive values.
func (s *SettingsService) List(ctx context.Context) ([]repository.Setting, error) {
	settings, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}

	// Decrypt sensitive values
	for i := range settings {
		if isSensitive(settings[i].Key) && crypto.IsEncrypted(settings[i].Value) {
			decrypted, err := crypto.Decrypt(settings[i].Value)
			if err != nil {
				slog.Warn("failed to decrypt setting", "key", settings[i].Key, "error", err)
				continue
			}
			settings[i].Value = decrypted
		}
	}

	return settings, nil
}

// Delete removes a setting.
func (s *SettingsService) Delete(ctx context.Context, key string) error {
	return s.repo.Delete(ctx, key)
}

// ProjectSupplyService handles project supply business logic.
type ProjectSupplyService struct {
	repo         *repository.ProjectSupplyRepository
	materialRepo *repository.MaterialRepository
}

// Create creates a new project supply.
func (s *ProjectSupplyService) Create(ctx context.Context, supply *model.ProjectSupply) error {
	// If material_id is set and name is empty, auto-populate from the material
	if supply.MaterialID != nil && *supply.MaterialID != uuid.Nil && supply.Name == "" {
		mat, err := s.materialRepo.GetByID(ctx, *supply.MaterialID)
		if err != nil {
			return fmt.Errorf("failed to look up material: %w", err)
		}
		if mat != nil {
			supply.Name = mat.Name
			if supply.UnitCostCents == 0 {
				supply.UnitCostCents = int(mat.CostPerKg * 100) // CostPerKg repurposed as per-unit $ for supplies
			}
		}
	}
	if supply.Name == "" {
		return fmt.Errorf("supply name is required")
	}
	if supply.Quantity < 1 {
		supply.Quantity = 1
	}
	return s.repo.Create(ctx, supply)
}

// ListByProject retrieves all supplies for a project.
func (s *ProjectSupplyService) ListByProject(ctx context.Context, projectID uuid.UUID) ([]model.ProjectSupply, error) {
	return s.repo.ListByProject(ctx, projectID)
}

// Delete removes a project supply.
func (s *ProjectSupplyService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
