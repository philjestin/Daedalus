package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/philjestin/daedalus/internal/model"
	"github.com/philjestin/daedalus/internal/printer"
	"github.com/philjestin/daedalus/internal/realtime"
	"github.com/philjestin/daedalus/internal/repository"
)

// DispatcherService orchestrates auto-dispatch of jobs to printers.
type DispatcherService struct {
	dispatchRepo    *repository.DispatchRepository
	settingsRepo    *repository.AutoDispatchSettingsRepository
	printJobRepo    *repository.PrintJobRepository
	printerRepo     *repository.PrinterRepository
	templateSvc     *TemplateService
	printJobSvc     *PrintJobService
	printerMgr      *printer.Manager
	hub             *realtime.Hub
	settingsService *SettingsService

	mu              sync.Mutex
	cleanupStopCh   chan struct{}
	cleanupInterval time.Duration
}

// NewDispatcherService creates a new dispatcher service.
func NewDispatcherService(
	dispatchRepo *repository.DispatchRepository,
	settingsRepo *repository.AutoDispatchSettingsRepository,
	printJobRepo *repository.PrintJobRepository,
	printerRepo *repository.PrinterRepository,
	templateSvc *TemplateService,
	printJobSvc *PrintJobService,
	printerMgr *printer.Manager,
	hub *realtime.Hub,
	settingsService *SettingsService,
) *DispatcherService {
	return &DispatcherService{
		dispatchRepo:    dispatchRepo,
		settingsRepo:    settingsRepo,
		printJobRepo:    printJobRepo,
		printerRepo:     printerRepo,
		templateSvc:     templateSvc,
		printJobSvc:     printJobSvc,
		printerMgr:      printerMgr,
		hub:             hub,
		settingsService: settingsService,
		cleanupInterval: 1 * time.Minute,
	}
}

// Init registers printer status callback and starts cleanup goroutine.
func (s *DispatcherService) Init() {
	s.printerMgr.OnStatusChange(s.handlePrinterStatusChange)
	slog.Info("DispatcherService: registered for printer status changes")

	// Start cleanup goroutine
	s.cleanupStopCh = make(chan struct{})
	go s.cleanupLoop()
}

// Stop stops the cleanup goroutine.
func (s *DispatcherService) Stop() {
	if s.cleanupStopCh != nil {
		close(s.cleanupStopCh)
	}
}

// handlePrinterStatusChange is called when a printer's status changes.
func (s *DispatcherService) handlePrinterStatusChange(newState, oldState *model.PrinterState) {
	if newState == nil {
		return
	}

	// Check for transition to idle from an active state
	if newState.Status == model.PrinterStatusIdle {
		wasActive := oldState != nil && (oldState.Status == model.PrinterStatusPrinting || oldState.Status == model.PrinterStatusPaused)
		if wasActive {
			go func() {
				if err := s.OnPrinterIdle(newState.PrinterID); err != nil {
					slog.Error("DispatcherService: failed to handle printer idle", "printer_id", newState.PrinterID, "error", err)
				}
			}()
		}
	}
}

// OnPrinterIdle is called when a printer transitions to idle state.
// It checks if auto-dispatch is enabled and creates a dispatch request if appropriate.
func (s *DispatcherService) OnPrinterIdle(printerID uuid.UUID) error {
	ctx := context.Background()

	// Check if global auto-dispatch is enabled
	if !s.IsGloballyEnabled(ctx) {
		slog.Debug("DispatcherService: auto-dispatch globally disabled")
		return nil
	}

	// Check printer-specific settings
	settings, err := s.settingsRepo.Get(ctx, printerID)
	if err != nil {
		return fmt.Errorf("failed to get printer settings: %w", err)
	}
	if !settings.Enabled {
		slog.Debug("DispatcherService: auto-dispatch disabled for printer", "printer_id", printerID)
		return nil
	}

	// Check if there's already a pending request for this printer
	existing, err := s.dispatchRepo.GetPendingForPrinter(ctx, printerID)
	if err != nil {
		return fmt.Errorf("failed to check existing requests: %w", err)
	}
	if existing != nil {
		slog.Debug("DispatcherService: pending request already exists", "printer_id", printerID, "request_id", existing.ID)
		return nil
	}

	// Find next compatible job
	job, err := s.FindNextJob(ctx, printerID)
	if err != nil {
		return fmt.Errorf("failed to find next job: %w", err)
	}
	if job == nil {
		slog.Debug("DispatcherService: no compatible job found", "printer_id", printerID)
		return nil
	}

	// Create dispatch request
	request, err := s.CreateDispatchRequest(ctx, job.ID, printerID)
	if err != nil {
		return fmt.Errorf("failed to create dispatch request: %w", err)
	}

	slog.Info("DispatcherService: created dispatch request", "request_id", request.ID, "job_id", job.ID, "printer_id", printerID)

	// Broadcast WebSocket event
	s.broadcastDispatchRequest(request)

	return nil
}

// FindNextJob finds the highest-priority compatible job for a printer.
func (s *DispatcherService) FindNextJob(ctx context.Context, printerID uuid.UUID) (*model.PrintJob, error) {
	// Get all queued jobs with auto_dispatch_enabled, ordered by priority DESC, created_at ASC
	jobs, err := s.printJobRepo.ListQueued(ctx)
	if err != nil {
		return nil, err
	}

	for _, job := range jobs {
		// Skip if job already has a pending dispatch request
		pendingReq, err := s.dispatchRepo.GetPendingForJob(ctx, job.ID)
		if err != nil {
			slog.Warn("DispatcherService: failed to check pending request for job", "job_id", job.ID, "error", err)
			continue
		}
		if pendingReq != nil {
			continue
		}

		// If job has a recipe, validate printer compatibility
		if job.RecipeID != nil {
			result, err := s.templateSvc.ValidatePrinterForRecipe(ctx, *job.RecipeID, printerID)
			if err != nil {
				slog.Warn("DispatcherService: failed to validate printer for recipe", "job_id", job.ID, "error", err)
				continue
			}
			if !result.Valid {
				continue
			}
		}

		// Check material compatibility if printer has AMS
		printerState, _ := s.printerMgr.GetState(printerID)
		if printerState != nil && printerState.AMS != nil && job.RecipeID != nil {
			compatible := s.checkMaterialCompatibility(ctx, *job.RecipeID, printerState.AMS)
			if !compatible {
				continue
			}
		}

		// Found a compatible job
		return &job, nil
	}

	return nil, nil
}

// checkMaterialCompatibility checks if the printer's AMS has compatible materials for the recipe.
func (s *DispatcherService) checkMaterialCompatibility(ctx context.Context, recipeID uuid.UUID, ams *model.AMSState) bool {
	// Get recipe materials
	template, err := s.templateSvc.GetByID(ctx, recipeID)
	if err != nil || template == nil {
		return false
	}

	if template.Materials == nil || len(template.Materials) == 0 {
		return true // No material requirements
	}

	// Build a map of available AMS materials
	availableMaterials := make(map[string]bool)
	for _, unit := range ams.Units {
		for _, tray := range unit.Trays {
			if !tray.Empty && tray.Remain > 10 { // At least 10% remaining
				key := fmt.Sprintf("%s:%s", tray.MaterialType, tray.Color)
				availableMaterials[key] = true
				// Also add just the type for looser matching
				availableMaterials[tray.MaterialType] = true
			}
		}
	}

	// Check if all required materials are available
	for _, rm := range template.Materials {
		// Check by type
		if !availableMaterials[string(rm.MaterialType)] {
			return false
		}
	}

	return true
}

// CreateDispatchRequest creates a new pending dispatch request.
func (s *DispatcherService) CreateDispatchRequest(ctx context.Context, jobID, printerID uuid.UUID) (*model.DispatchRequest, error) {
	// Get printer settings for timeout
	settings, err := s.settingsRepo.Get(ctx, printerID)
	if err != nil {
		return nil, err
	}

	request := &model.DispatchRequest{
		JobID:     jobID,
		PrinterID: printerID,
		ExpiresAt: time.Now().Add(time.Duration(settings.TimeoutMinutes) * time.Minute),
	}

	if err := s.dispatchRepo.Create(ctx, request); err != nil {
		return nil, err
	}

	// Enrich with job and printer data
	job, _ := s.printJobRepo.GetByID(ctx, jobID)
	printer, _ := s.printerRepo.GetByID(ctx, printerID)
	request.Job = job
	request.Printer = printer

	return request, nil
}

// ConfirmDispatch confirms a dispatch request and starts the job.
func (s *DispatcherService) ConfirmDispatch(ctx context.Context, requestID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	request, err := s.dispatchRepo.GetByID(ctx, requestID)
	if err != nil {
		return err
	}
	if request == nil {
		return fmt.Errorf("dispatch request not found")
	}
	if request.Status != model.DispatchPending {
		return fmt.Errorf("dispatch request is not pending")
	}

	// Update request status
	if err := s.dispatchRepo.UpdateStatus(ctx, requestID, model.DispatchConfirmed, ""); err != nil {
		return err
	}

	// Get the job
	job, err := s.printJobRepo.GetByID(ctx, request.JobID)
	if err != nil {
		return err
	}
	if job == nil {
		return fmt.Errorf("job not found")
	}

	// Assign printer to job
	job.PrinterID = &request.PrinterID
	if err := s.printJobRepo.Update(ctx, job); err != nil {
		return err
	}

	// Record assignment event
	status := model.PrintJobStatusAssigned
	event := model.NewJobEvent(job.ID, model.JobEventAssigned, &status).WithPrinter(request.PrinterID)
	if err := s.printJobRepo.AppendEvent(ctx, event); err != nil {
		slog.Warn("DispatcherService: failed to record assignment event", "job_id", job.ID, "error", err)
	}

	// Check if auto-start is enabled
	settings, err := s.settingsRepo.Get(ctx, request.PrinterID)
	if err != nil {
		return err
	}

	if settings.AutoStart {
		// Start the job
		if err := s.printJobSvc.Start(ctx, job.ID); err != nil {
			slog.Warn("DispatcherService: failed to auto-start job", "job_id", job.ID, "error", err)
			// Don't return error - the job is assigned, just not started
		}
	}

	// Broadcast confirmation
	s.broadcastDispatchConfirmed(request)

	return nil
}

// RejectDispatch rejects a dispatch request.
func (s *DispatcherService) RejectDispatch(ctx context.Context, requestID uuid.UUID, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	request, err := s.dispatchRepo.GetByID(ctx, requestID)
	if err != nil {
		return err
	}
	if request == nil {
		return fmt.Errorf("dispatch request not found")
	}
	if request.Status != model.DispatchPending {
		return fmt.Errorf("dispatch request is not pending")
	}

	if err := s.dispatchRepo.UpdateStatus(ctx, requestID, model.DispatchRejected, reason); err != nil {
		return err
	}

	// Broadcast rejection
	s.broadcastDispatchRejected(request, reason)

	return nil
}

// SkipJob skips the current job and tries to find the next compatible one.
func (s *DispatcherService) SkipJob(ctx context.Context, requestID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	request, err := s.dispatchRepo.GetByID(ctx, requestID)
	if err != nil {
		return err
	}
	if request == nil {
		return fmt.Errorf("dispatch request not found")
	}
	if request.Status != model.DispatchPending {
		return fmt.Errorf("dispatch request is not pending")
	}

	// Mark current request as rejected (skipped)
	if err := s.dispatchRepo.UpdateStatus(ctx, requestID, model.DispatchRejected, "skipped"); err != nil {
		return err
	}

	// Disable auto-dispatch for this specific job
	job, err := s.printJobRepo.GetByID(ctx, request.JobID)
	if err != nil {
		return err
	}
	if job != nil {
		job.AutoDispatchEnabled = false
		if err := s.printJobRepo.Update(ctx, job); err != nil {
			slog.Warn("DispatcherService: failed to disable auto-dispatch for job", "job_id", job.ID, "error", err)
		}
	}

	// Try to find the next job
	s.mu.Unlock() // Release lock before calling OnPrinterIdle
	if err := s.OnPrinterIdle(request.PrinterID); err != nil {
		slog.Warn("DispatcherService: failed to find next job after skip", "printer_id", request.PrinterID, "error", err)
	}
	s.mu.Lock()

	return nil
}

// ListPending returns all pending dispatch requests.
func (s *DispatcherService) ListPending(ctx context.Context) ([]model.DispatchRequest, error) {
	requests, err := s.dispatchRepo.ListPending(ctx)
	if err != nil {
		return nil, err
	}

	// Enrich with job and printer data
	for i := range requests {
		job, _ := s.printJobRepo.GetByID(ctx, requests[i].JobID)
		printer, _ := s.printerRepo.GetByID(ctx, requests[i].PrinterID)
		requests[i].Job = job
		requests[i].Printer = printer
	}

	return requests, nil
}

// GetSettings returns auto-dispatch settings for a printer.
func (s *DispatcherService) GetSettings(ctx context.Context, printerID uuid.UUID) (*model.AutoDispatchSettings, error) {
	return s.settingsRepo.Get(ctx, printerID)
}

// UpdateSettings updates auto-dispatch settings for a printer.
func (s *DispatcherService) UpdateSettings(ctx context.Context, settings *model.AutoDispatchSettings) error {
	return s.settingsRepo.Upsert(ctx, settings)
}

// IsGloballyEnabled returns whether auto-dispatch is globally enabled.
func (s *DispatcherService) IsGloballyEnabled(ctx context.Context) bool {
	setting, err := s.settingsService.Get(ctx, "auto_dispatch_enabled")
	if err != nil || setting == nil {
		return false // Disabled by default
	}
	return setting.Value == "true" || setting.Value == "1"
}

// SetGlobalEnabled enables or disables auto-dispatch globally.
func (s *DispatcherService) SetGlobalEnabled(ctx context.Context, enabled bool) error {
	val := "false"
	if enabled {
		val = "true"
	}
	return s.settingsService.Set(ctx, "auto_dispatch_enabled", val)
}

// cleanupLoop periodically expires old dispatch requests.
func (s *DispatcherService) cleanupLoop() {
	ticker := time.NewTicker(s.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.cleanupStopCh:
			return
		case <-ticker.C:
			ctx := context.Background()
			n, err := s.dispatchRepo.ExpireOld(ctx)
			if err != nil {
				slog.Warn("DispatcherService: failed to expire old requests", "error", err)
			} else if n > 0 {
				slog.Info("DispatcherService: expired old dispatch requests", "count", n)
				// Broadcast expiration events
				s.hub.Broadcast(model.BroadcastEvent{
					Type: "dispatch_expired",
					Data: map[string]interface{}{"count": n},
				})
			}
		}
	}
}

// broadcastDispatchRequest sends a dispatch_request WebSocket event.
func (s *DispatcherService) broadcastDispatchRequest(request *model.DispatchRequest) {
	s.hub.Broadcast(model.BroadcastEvent{
		Type: "dispatch_request",
		Data: request,
	})
}

// broadcastDispatchConfirmed sends a dispatch_confirmed WebSocket event.
func (s *DispatcherService) broadcastDispatchConfirmed(request *model.DispatchRequest) {
	s.hub.Broadcast(model.BroadcastEvent{
		Type: "dispatch_confirmed",
		Data: request,
	})
}

// broadcastDispatchRejected sends a dispatch_rejected WebSocket event.
func (s *DispatcherService) broadcastDispatchRejected(request *model.DispatchRequest, reason string) {
	s.hub.Broadcast(model.BroadcastEvent{
		Type: "dispatch_rejected",
		Data: map[string]interface{}{
			"request": request,
			"reason":  reason,
		},
	})
}
