package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hyperion/printfarm/internal/model"
)

// --- Dispatch Repository Tests ---

func TestDispatchRepository_CreateAndGetByID(t *testing.T) {
	db := openTestDB(t)
	dispatchRepo := NewDispatchRepository(db)
	printerRepo := &PrinterRepository{db: db}
	ctx := context.Background()

	// Create a printer first (for FK)
	printer := &model.Printer{Name: "Test Printer"}
	if err := printerRepo.Create(ctx, printer); err != nil {
		t.Fatalf("Create printer failed: %v", err)
	}

	// Create necessary test data for job
	project, design := setupPrintJobTestData(t, db)
	printJobRepo := &PrintJobRepository{db: db}
	job := &model.PrintJob{
		DesignID:  design.ID,
		ProjectID: &project.ID,
		Status:    model.PrintJobStatusQueued,
	}
	if err := printJobRepo.Create(ctx, job); err != nil {
		t.Fatalf("Create job failed: %v", err)
	}

	// Create dispatch request
	req := &model.DispatchRequest{
		JobID:     job.ID,
		PrinterID: printer.ID,
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}

	if err := dispatchRepo.Create(ctx, req); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if req.ID == uuid.Nil {
		t.Error("Create should set request ID")
	}
	if req.Status != model.DispatchPending {
		t.Errorf("Create should set Status to 'pending', got %q", req.Status)
	}
	if req.CreatedAt.IsZero() {
		t.Error("Create should set CreatedAt")
	}

	// Retrieve the request
	got, err := dispatchRepo.GetByID(ctx, req.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetByID returned nil")
	}

	if got.JobID != job.ID {
		t.Errorf("JobID = %v, want %v", got.JobID, job.ID)
	}
	if got.PrinterID != printer.ID {
		t.Errorf("PrinterID = %v, want %v", got.PrinterID, printer.ID)
	}
	if got.Status != model.DispatchPending {
		t.Errorf("Status = %q, want %q", got.Status, model.DispatchPending)
	}
}

func TestDispatchRepository_GetByID_NotFound(t *testing.T) {
	db := openTestDB(t)
	repo := NewDispatchRepository(db)
	ctx := context.Background()

	got, err := repo.GetByID(ctx, uuid.New())
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got != nil {
		t.Error("GetByID should return nil for non-existent request")
	}
}

func TestDispatchRepository_GetPendingForPrinter(t *testing.T) {
	db := openTestDB(t)
	dispatchRepo := NewDispatchRepository(db)
	printerRepo := &PrinterRepository{db: db}
	ctx := context.Background()

	// Create printer
	printer := &model.Printer{Name: "Test Printer"}
	if err := printerRepo.Create(ctx, printer); err != nil {
		t.Fatalf("Create printer failed: %v", err)
	}

	// Create test job data
	project, design := setupPrintJobTestData(t, db)
	printJobRepo := &PrintJobRepository{db: db}
	job := &model.PrintJob{
		DesignID:  design.ID,
		ProjectID: &project.ID,
		Status:    model.PrintJobStatusQueued,
	}
	if err := printJobRepo.Create(ctx, job); err != nil {
		t.Fatalf("Create job failed: %v", err)
	}

	// Create pending dispatch request
	req := &model.DispatchRequest{
		JobID:     job.ID,
		PrinterID: printer.ID,
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}
	if err := dispatchRepo.Create(ctx, req); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Get pending request
	got, err := dispatchRepo.GetPendingForPrinter(ctx, printer.ID)
	if err != nil {
		t.Fatalf("GetPendingForPrinter failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetPendingForPrinter returned nil")
	}
	if got.ID != req.ID {
		t.Errorf("Got request ID %v, want %v", got.ID, req.ID)
	}

	// Should return nil for different printer
	got, err = dispatchRepo.GetPendingForPrinter(ctx, uuid.New())
	if err != nil {
		t.Fatalf("GetPendingForPrinter failed: %v", err)
	}
	if got != nil {
		t.Error("GetPendingForPrinter should return nil for different printer")
	}
}

func TestDispatchRepository_GetPendingForJob(t *testing.T) {
	db := openTestDB(t)
	dispatchRepo := NewDispatchRepository(db)
	printerRepo := &PrinterRepository{db: db}
	ctx := context.Background()

	// Create printer
	printer := &model.Printer{Name: "Test Printer"}
	if err := printerRepo.Create(ctx, printer); err != nil {
		t.Fatalf("Create printer failed: %v", err)
	}

	// Create test job data
	project, design := setupPrintJobTestData(t, db)
	printJobRepo := &PrintJobRepository{db: db}
	job := &model.PrintJob{
		DesignID:  design.ID,
		ProjectID: &project.ID,
		Status:    model.PrintJobStatusQueued,
	}
	if err := printJobRepo.Create(ctx, job); err != nil {
		t.Fatalf("Create job failed: %v", err)
	}

	// Create pending dispatch request
	req := &model.DispatchRequest{
		JobID:     job.ID,
		PrinterID: printer.ID,
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}
	if err := dispatchRepo.Create(ctx, req); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Get pending request for job
	got, err := dispatchRepo.GetPendingForJob(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetPendingForJob failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetPendingForJob returned nil")
	}
	if got.ID != req.ID {
		t.Errorf("Got request ID %v, want %v", got.ID, req.ID)
	}

	// Should return nil for different job
	got, err = dispatchRepo.GetPendingForJob(ctx, uuid.New())
	if err != nil {
		t.Fatalf("GetPendingForJob failed: %v", err)
	}
	if got != nil {
		t.Error("GetPendingForJob should return nil for different job")
	}
}

func TestDispatchRepository_ListPending(t *testing.T) {
	db := openTestDB(t)
	dispatchRepo := NewDispatchRepository(db)
	printerRepo := &PrinterRepository{db: db}
	ctx := context.Background()

	// Create printer
	printer := &model.Printer{Name: "Test Printer"}
	if err := printerRepo.Create(ctx, printer); err != nil {
		t.Fatalf("Create printer failed: %v", err)
	}

	// Create test jobs
	project, design := setupPrintJobTestData(t, db)
	printJobRepo := &PrintJobRepository{db: db}

	// Create multiple jobs and dispatch requests
	for i := 0; i < 3; i++ {
		job := &model.PrintJob{
			DesignID:  design.ID,
			ProjectID: &project.ID,
			Status:    model.PrintJobStatusQueued,
		}
		if err := printJobRepo.Create(ctx, job); err != nil {
			t.Fatalf("Create job %d failed: %v", i, err)
		}

		req := &model.DispatchRequest{
			JobID:     job.ID,
			PrinterID: printer.ID,
			ExpiresAt: time.Now().Add(30 * time.Minute),
		}
		if err := dispatchRepo.Create(ctx, req); err != nil {
			t.Fatalf("Create request %d failed: %v", i, err)
		}
		time.Sleep(10 * time.Millisecond) // ensure different timestamps
	}

	// List pending
	requests, err := dispatchRepo.ListPending(ctx)
	if err != nil {
		t.Fatalf("ListPending failed: %v", err)
	}
	if len(requests) != 3 {
		t.Errorf("len(requests) = %d, want 3", len(requests))
	}
}

func TestDispatchRepository_UpdateStatus(t *testing.T) {
	db := openTestDB(t)
	dispatchRepo := NewDispatchRepository(db)
	printerRepo := &PrinterRepository{db: db}
	ctx := context.Background()

	// Create printer
	printer := &model.Printer{Name: "Test Printer"}
	if err := printerRepo.Create(ctx, printer); err != nil {
		t.Fatalf("Create printer failed: %v", err)
	}

	// Create test job data
	project, design := setupPrintJobTestData(t, db)
	printJobRepo := &PrintJobRepository{db: db}
	job := &model.PrintJob{
		DesignID:  design.ID,
		ProjectID: &project.ID,
		Status:    model.PrintJobStatusQueued,
	}
	if err := printJobRepo.Create(ctx, job); err != nil {
		t.Fatalf("Create job failed: %v", err)
	}

	// Create dispatch request
	req := &model.DispatchRequest{
		JobID:     job.ID,
		PrinterID: printer.ID,
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}
	if err := dispatchRepo.Create(ctx, req); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Update status to confirmed
	if err := dispatchRepo.UpdateStatus(ctx, req.ID, model.DispatchConfirmed, ""); err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	got, _ := dispatchRepo.GetByID(ctx, req.ID)
	if got.Status != model.DispatchConfirmed {
		t.Errorf("Status = %q, want %q", got.Status, model.DispatchConfirmed)
	}
	if got.RespondedAt == nil {
		t.Error("RespondedAt should be set after UpdateStatus")
	}

	// No longer pending
	pending, _ := dispatchRepo.GetPendingForPrinter(ctx, printer.ID)
	if pending != nil {
		t.Error("Confirmed request should not be returned by GetPendingForPrinter")
	}
}

func TestDispatchRepository_UpdateStatus_WithReason(t *testing.T) {
	db := openTestDB(t)
	dispatchRepo := NewDispatchRepository(db)
	printerRepo := &PrinterRepository{db: db}
	ctx := context.Background()

	// Create printer
	printer := &model.Printer{Name: "Test Printer"}
	printerRepo.Create(ctx, printer)

	// Create test job data
	project, design := setupPrintJobTestData(t, db)
	printJobRepo := &PrintJobRepository{db: db}
	job := &model.PrintJob{
		DesignID:  design.ID,
		ProjectID: &project.ID,
		Status:    model.PrintJobStatusQueued,
	}
	printJobRepo.Create(ctx, job)

	// Create dispatch request
	req := &model.DispatchRequest{
		JobID:     job.ID,
		PrinterID: printer.ID,
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}
	dispatchRepo.Create(ctx, req)

	// Reject with reason
	reason := "Bed not clear"
	if err := dispatchRepo.UpdateStatus(ctx, req.ID, model.DispatchRejected, reason); err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	got, _ := dispatchRepo.GetByID(ctx, req.ID)
	if got.Status != model.DispatchRejected {
		t.Errorf("Status = %q, want %q", got.Status, model.DispatchRejected)
	}
	if got.Reason != reason {
		t.Errorf("Reason = %q, want %q", got.Reason, reason)
	}
}

func TestDispatchRepository_ExpireOld(t *testing.T) {
	db := openTestDB(t)
	dispatchRepo := NewDispatchRepository(db)
	printerRepo := &PrinterRepository{db: db}
	ctx := context.Background()

	// Create printer
	printer := &model.Printer{Name: "Test Printer"}
	printerRepo.Create(ctx, printer)

	// Create test job data
	project, design := setupPrintJobTestData(t, db)
	printJobRepo := &PrintJobRepository{db: db}

	// Create expired request
	job1 := &model.PrintJob{
		DesignID:  design.ID,
		ProjectID: &project.ID,
		Status:    model.PrintJobStatusQueued,
	}
	printJobRepo.Create(ctx, job1)

	expiredReq := &model.DispatchRequest{
		JobID:     job1.ID,
		PrinterID: printer.ID,
		ExpiresAt: time.Now().Add(-1 * time.Hour), // Already expired
	}
	dispatchRepo.Create(ctx, expiredReq)

	// Create non-expired request
	job2 := &model.PrintJob{
		DesignID:  design.ID,
		ProjectID: &project.ID,
		Status:    model.PrintJobStatusQueued,
	}
	printJobRepo.Create(ctx, job2)

	validReq := &model.DispatchRequest{
		JobID:     job2.ID,
		PrinterID: printer.ID,
		ExpiresAt: time.Now().Add(1 * time.Hour), // Still valid
	}
	dispatchRepo.Create(ctx, validReq)

	// Run expiration
	count, err := dispatchRepo.ExpireOld(ctx)
	if err != nil {
		t.Fatalf("ExpireOld failed: %v", err)
	}
	if count != 1 {
		t.Errorf("ExpireOld count = %d, want 1", count)
	}

	// Check expired request
	got, _ := dispatchRepo.GetByID(ctx, expiredReq.ID)
	if got.Status != model.DispatchExpired {
		t.Errorf("Expired request status = %q, want %q", got.Status, model.DispatchExpired)
	}

	// Check valid request still pending
	got, _ = dispatchRepo.GetByID(ctx, validReq.ID)
	if got.Status != model.DispatchPending {
		t.Errorf("Valid request status = %q, want %q", got.Status, model.DispatchPending)
	}

	// ListPending should only return the valid one
	pending, _ := dispatchRepo.ListPending(ctx)
	if len(pending) != 1 {
		t.Errorf("len(pending) = %d, want 1", len(pending))
	}
}

// --- Auto Dispatch Settings Repository Tests ---

func TestAutoDispatchSettingsRepository_GetDefaults(t *testing.T) {
	db := openTestDB(t)
	repo := NewAutoDispatchSettingsRepository(db)
	printerRepo := &PrinterRepository{db: db}
	ctx := context.Background()

	// Create a printer
	printer := &model.Printer{Name: "Test Printer"}
	if err := printerRepo.Create(ctx, printer); err != nil {
		t.Fatalf("Create printer failed: %v", err)
	}

	// Get settings for printer without explicit settings (should return defaults)
	settings, err := repo.Get(ctx, printer.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if settings.PrinterID != printer.ID {
		t.Errorf("PrinterID = %v, want %v", settings.PrinterID, printer.ID)
	}
	if settings.Enabled != false {
		t.Errorf("Enabled should default to false, got %v", settings.Enabled)
	}
	if settings.RequireConfirmation != true {
		t.Errorf("RequireConfirmation should default to true, got %v", settings.RequireConfirmation)
	}
	if settings.AutoStart != false {
		t.Errorf("AutoStart should default to false, got %v", settings.AutoStart)
	}
	if settings.TimeoutMinutes != 30 {
		t.Errorf("TimeoutMinutes should default to 30, got %d", settings.TimeoutMinutes)
	}
}

func TestAutoDispatchSettingsRepository_Upsert(t *testing.T) {
	db := openTestDB(t)
	repo := NewAutoDispatchSettingsRepository(db)
	printerRepo := &PrinterRepository{db: db}
	ctx := context.Background()

	// Create a printer
	printer := &model.Printer{Name: "Test Printer"}
	if err := printerRepo.Create(ctx, printer); err != nil {
		t.Fatalf("Create printer failed: %v", err)
	}

	// Insert new settings
	settings := &model.AutoDispatchSettings{
		PrinterID:           printer.ID,
		Enabled:             true,
		RequireConfirmation: false,
		AutoStart:           true,
		TimeoutMinutes:      60,
	}

	if err := repo.Upsert(ctx, settings); err != nil {
		t.Fatalf("Upsert (insert) failed: %v", err)
	}

	// Verify
	got, _ := repo.Get(ctx, printer.ID)
	if !got.Enabled {
		t.Error("Enabled = false, want true")
	}
	if got.RequireConfirmation {
		t.Error("RequireConfirmation = true, want false")
	}
	if !got.AutoStart {
		t.Error("AutoStart = false, want true")
	}
	if got.TimeoutMinutes != 60 {
		t.Errorf("TimeoutMinutes = %d, want 60", got.TimeoutMinutes)
	}

	// Update existing settings
	settings.Enabled = false
	settings.TimeoutMinutes = 15

	if err := repo.Upsert(ctx, settings); err != nil {
		t.Fatalf("Upsert (update) failed: %v", err)
	}

	// Verify update
	got, _ = repo.Get(ctx, printer.ID)
	if got.Enabled {
		t.Error("Enabled = true after update, want false")
	}
	if got.TimeoutMinutes != 15 {
		t.Errorf("TimeoutMinutes = %d after update, want 15", got.TimeoutMinutes)
	}
}

func TestAutoDispatchSettingsRepository_Delete(t *testing.T) {
	db := openTestDB(t)
	repo := NewAutoDispatchSettingsRepository(db)
	printerRepo := &PrinterRepository{db: db}
	ctx := context.Background()

	// Create a printer
	printer := &model.Printer{Name: "Test Printer"}
	printerRepo.Create(ctx, printer)

	// Insert settings
	settings := &model.AutoDispatchSettings{
		PrinterID:           printer.ID,
		Enabled:             true,
		RequireConfirmation: true,
		AutoStart:           true,
		TimeoutMinutes:      60,
	}
	repo.Upsert(ctx, settings)

	// Delete
	if err := repo.Delete(ctx, printer.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Should return defaults after delete
	got, _ := repo.Get(ctx, printer.ID)
	if got.Enabled != false {
		t.Error("After delete, should get default Enabled=false")
	}
	if got.TimeoutMinutes != 30 {
		t.Errorf("After delete, should get default TimeoutMinutes=30, got %d", got.TimeoutMinutes)
	}
}

// --- PrintJobRepository Dispatch Extension Tests ---

func TestPrintJobRepository_ListQueued(t *testing.T) {
	db := openTestDB(t)
	printJobRepo := &PrintJobRepository{db: db}
	ctx := context.Background()

	project, design := setupPrintJobTestData(t, db)

	// Create jobs with different statuses and priorities
	for i := 0; i < 3; i++ {
		job := &model.PrintJob{
			DesignID:            design.ID,
			ProjectID:           &project.ID,
			Status:              model.PrintJobStatusQueued,
			Priority:            i,
			AutoDispatchEnabled: true,
		}
		if err := printJobRepo.Create(ctx, job); err != nil {
			t.Fatalf("Create job %d failed: %v", i, err)
		}
		time.Sleep(10 * time.Millisecond) // ensure different timestamps
	}

	// Create a non-queued job (should not appear)
	completedJob := &model.PrintJob{
		DesignID:  design.ID,
		ProjectID: &project.ID,
		Status:    model.PrintJobStatusQueued, // Will be changed via event
		Priority:  100,
	}
	printJobRepo.Create(ctx, completedJob)

	// Mark as completed via event
	status := model.PrintJobStatusCompleted
	event := model.NewJobEvent(completedJob.ID, model.JobEventCompleted, &status)
	printJobRepo.AppendEvent(ctx, event)

	// Create a job with auto_dispatch_enabled=false
	// Note: Create() defaults auto_dispatch_enabled to true, so we must update it after
	noDispatchJob := &model.PrintJob{
		DesignID:  design.ID,
		ProjectID: &project.ID,
		Status:    model.PrintJobStatusQueued,
		Priority:  50,
	}
	printJobRepo.Create(ctx, noDispatchJob)
	// Update to set auto_dispatch_enabled=false
	noDispatchJob.AutoDispatchEnabled = false
	printJobRepo.Update(ctx, noDispatchJob)

	// List queued
	jobs, err := printJobRepo.ListQueued(ctx)
	if err != nil {
		t.Fatalf("ListQueued failed: %v", err)
	}

	// Should only get the 3 queued jobs with auto_dispatch_enabled=true
	if len(jobs) != 3 {
		t.Errorf("len(jobs) = %d, want 3", len(jobs))
	}

	// Should be ordered by priority DESC
	if len(jobs) >= 2 && jobs[0].Priority < jobs[1].Priority {
		t.Error("Jobs should be ordered by priority DESC")
	}
}

func TestPrintJobRepository_UpdatePriority(t *testing.T) {
	db := openTestDB(t)
	printJobRepo := &PrintJobRepository{db: db}
	ctx := context.Background()

	project, design := setupPrintJobTestData(t, db)

	job := &model.PrintJob{
		DesignID:  design.ID,
		ProjectID: &project.ID,
		Status:    model.PrintJobStatusQueued,
		Priority:  0,
	}
	if err := printJobRepo.Create(ctx, job); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Update priority
	if err := printJobRepo.UpdatePriority(ctx, job.ID, 10); err != nil {
		t.Fatalf("UpdatePriority failed: %v", err)
	}

	// Verify
	got, _ := printJobRepo.GetByID(ctx, job.ID)
	if got.Priority != 10 {
		t.Errorf("Priority = %d, want 10", got.Priority)
	}
}
