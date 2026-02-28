package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/philjestin/daedalus/internal/database"
	"github.com/philjestin/daedalus/internal/model"
	"github.com/philjestin/daedalus/internal/printer"
	"github.com/philjestin/daedalus/internal/realtime"
	"github.com/philjestin/daedalus/internal/repository"
	"github.com/philjestin/daedalus/internal/storage"
)

// setupDispatcherTestEnv creates a test environment with all necessary services.
func setupDispatcherTestEnv(t *testing.T) (*DispatcherService, *repository.Repositories, *printer.Manager) {
	t.Helper()

	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	storageDir := t.TempDir()
	store := storage.NewLocalStorage(storageDir)

	repos := repository.NewRepositories(db)
	hub := realtime.NewHub()
	printerMgr := printer.NewManager()

	services := NewServices(repos, store, printerMgr, hub)

	return services.Dispatcher, repos, printerMgr
}

// createTestPrintJob creates a print job with required dependencies for testing.
func createTestPrintJob(t *testing.T, repos *repository.Repositories, status model.PrintJobStatus, priority int, autoDispatch bool) *model.PrintJob {
	t.Helper()
	ctx := context.Background()

	// Create project
	project := &model.Project{Name: "Test Project " + uuid.New().String()}
	if err := repos.Projects.Create(ctx, project); err != nil {
		t.Fatalf("Create project failed: %v", err)
	}

	// Create part
	part := &model.Part{ProjectID: project.ID, Name: "Test Part"}
	if err := repos.Parts.Create(ctx, part); err != nil {
		t.Fatalf("Create part failed: %v", err)
	}

	// Create file
	file := &model.File{
		Hash:         uuid.New().String(),
		OriginalName: "test.3mf",
		ContentType:  "application/3mf",
		SizeBytes:    1024,
		StoragePath:  "test/path",
	}
	if err := repos.Files.Create(ctx, file); err != nil {
		t.Fatalf("Create file failed: %v", err)
	}

	// Create design
	design := &model.Design{
		PartID:        part.ID,
		FileName:      "test.3mf",
		FileID:        file.ID,
		FileHash:      file.Hash,
		FileSizeBytes: file.SizeBytes,
		FileType:      "3mf",
	}
	if err := repos.Designs.Create(ctx, design); err != nil {
		t.Fatalf("Create design failed: %v", err)
	}

	// Create job (note: Create defaults AutoDispatchEnabled to true)
	job := &model.PrintJob{
		DesignID:  design.ID,
		ProjectID: &project.ID,
		Status:    status,
		Priority:  priority,
	}
	if err := repos.PrintJobs.Create(ctx, job); err != nil {
		t.Fatalf("Create job failed: %v", err)
	}

	// Update auto_dispatch_enabled to the desired value if not true (since Create defaults to true)
	if !autoDispatch {
		job.AutoDispatchEnabled = false
		if err := repos.PrintJobs.Update(ctx, job); err != nil {
			t.Fatalf("Update job auto_dispatch_enabled failed: %v", err)
		}
	}

	return job
}

// createTestPrinter creates a printer for testing.
func createTestPrinter(t *testing.T, repos *repository.Repositories, name string) *model.Printer {
	t.Helper()
	ctx := context.Background()

	printerObj := &model.Printer{Name: name}
	if err := repos.Printers.Create(ctx, printerObj); err != nil {
		t.Fatalf("Create printer failed: %v", err)
	}
	return printerObj
}

func TestDispatcherService_IsGloballyEnabled(t *testing.T) {
	dispatcher, repos, _ := setupDispatcherTestEnv(t)
	ctx := context.Background()

	// Default: disabled
	if dispatcher.IsGloballyEnabled(ctx) {
		t.Error("Auto-dispatch should be disabled by default")
	}

	// Enable globally
	repos.Settings.Set(ctx, "auto_dispatch_enabled", "true")

	if !dispatcher.IsGloballyEnabled(ctx) {
		t.Error("Auto-dispatch should be enabled after setting to true")
	}

	// Disable again
	repos.Settings.Set(ctx, "auto_dispatch_enabled", "false")

	if dispatcher.IsGloballyEnabled(ctx) {
		t.Error("Auto-dispatch should be disabled after setting to false")
	}
}

func TestDispatcherService_SetGlobalEnabled(t *testing.T) {
	dispatcher, _, _ := setupDispatcherTestEnv(t)
	ctx := context.Background()

	// Enable
	if err := dispatcher.SetGlobalEnabled(ctx, true); err != nil {
		t.Fatalf("SetGlobalEnabled(true) failed: %v", err)
	}
	if !dispatcher.IsGloballyEnabled(ctx) {
		t.Error("Should be enabled after SetGlobalEnabled(true)")
	}

	// Disable
	if err := dispatcher.SetGlobalEnabled(ctx, false); err != nil {
		t.Fatalf("SetGlobalEnabled(false) failed: %v", err)
	}
	if dispatcher.IsGloballyEnabled(ctx) {
		t.Error("Should be disabled after SetGlobalEnabled(false)")
	}
}

func TestDispatcherService_GetSettings_DefaultsForNewPrinter(t *testing.T) {
	dispatcher, repos, _ := setupDispatcherTestEnv(t)
	ctx := context.Background()

	printerObj := createTestPrinter(t, repos, "Test Printer")

	settings, err := dispatcher.GetSettings(ctx, printerObj.ID)
	if err != nil {
		t.Fatalf("GetSettings failed: %v", err)
	}

	if settings.Enabled {
		t.Error("New printer should have Enabled=false by default")
	}
	if !settings.RequireConfirmation {
		t.Error("New printer should have RequireConfirmation=true by default")
	}
	if settings.AutoStart {
		t.Error("New printer should have AutoStart=false by default")
	}
	if settings.TimeoutMinutes != 30 {
		t.Errorf("New printer should have TimeoutMinutes=30 by default, got %d", settings.TimeoutMinutes)
	}
}

func TestDispatcherService_UpdateSettings(t *testing.T) {
	dispatcher, repos, _ := setupDispatcherTestEnv(t)
	ctx := context.Background()

	printerObj := createTestPrinter(t, repos, "Test Printer")

	settings := &model.AutoDispatchSettings{
		PrinterID:           printerObj.ID,
		Enabled:             true,
		RequireConfirmation: false,
		AutoStart:           true,
		TimeoutMinutes:      60,
	}

	if err := dispatcher.UpdateSettings(ctx, settings); err != nil {
		t.Fatalf("UpdateSettings failed: %v", err)
	}

	// Verify
	got, _ := dispatcher.GetSettings(ctx, printerObj.ID)
	if !got.Enabled {
		t.Error("Enabled should be true")
	}
	if got.RequireConfirmation {
		t.Error("RequireConfirmation should be false")
	}
	if !got.AutoStart {
		t.Error("AutoStart should be true")
	}
	if got.TimeoutMinutes != 60 {
		t.Errorf("TimeoutMinutes = %d, want 60", got.TimeoutMinutes)
	}
}

func TestDispatcherService_FindNextJob_NoJobs(t *testing.T) {
	dispatcher, repos, _ := setupDispatcherTestEnv(t)
	ctx := context.Background()

	printerObj := createTestPrinter(t, repos, "Test Printer")

	job, err := dispatcher.FindNextJob(ctx, printerObj.ID)
	if err != nil {
		t.Fatalf("FindNextJob failed: %v", err)
	}
	if job != nil {
		t.Error("FindNextJob should return nil when no queued jobs exist")
	}
}

func TestDispatcherService_FindNextJob_SkipsAutoDispatchDisabled(t *testing.T) {
	dispatcher, repos, _ := setupDispatcherTestEnv(t)
	ctx := context.Background()

	printerObj := createTestPrinter(t, repos, "Test Printer")

	// Create job with auto_dispatch_enabled=false
	createTestPrintJob(t, repos, model.PrintJobStatusQueued, 0, false)

	job, err := dispatcher.FindNextJob(ctx, printerObj.ID)
	if err != nil {
		t.Fatalf("FindNextJob failed: %v", err)
	}
	if job != nil {
		t.Error("FindNextJob should skip jobs with auto_dispatch_enabled=false")
	}
}

func TestDispatcherService_FindNextJob_ReturnsHighestPriority(t *testing.T) {
	dispatcher, repos, _ := setupDispatcherTestEnv(t)
	ctx := context.Background()

	printerObj := createTestPrinter(t, repos, "Test Printer")

	// Create jobs with different priorities
	lowPriorityJob := createTestPrintJob(t, repos, model.PrintJobStatusQueued, 1, true)
	highPriorityJob := createTestPrintJob(t, repos, model.PrintJobStatusQueued, 10, true)
	mediumPriorityJob := createTestPrintJob(t, repos, model.PrintJobStatusQueued, 5, true)

	_ = lowPriorityJob
	_ = mediumPriorityJob

	job, err := dispatcher.FindNextJob(ctx, printerObj.ID)
	if err != nil {
		t.Fatalf("FindNextJob failed: %v", err)
	}
	if job == nil {
		t.Fatal("FindNextJob should return a job")
	}
	if job.ID != highPriorityJob.ID {
		t.Errorf("FindNextJob should return highest priority job, got priority %d instead of 10", job.Priority)
	}
}

func TestDispatcherService_FindNextJob_SkipsJobsWithPendingRequest(t *testing.T) {
	dispatcher, repos, _ := setupDispatcherTestEnv(t)
	ctx := context.Background()

	printerObj := createTestPrinter(t, repos, "Test Printer")

	// Create two jobs
	highPriorityJob := createTestPrintJob(t, repos, model.PrintJobStatusQueued, 10, true)
	lowPriorityJob := createTestPrintJob(t, repos, model.PrintJobStatusQueued, 1, true)

	// Create pending dispatch request for high priority job
	req := &model.DispatchRequest{
		JobID:     highPriorityJob.ID,
		PrinterID: printerObj.ID,
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}
	if err := repos.Dispatch.Create(ctx, req); err != nil {
		t.Fatalf("Create dispatch request failed: %v", err)
	}

	// FindNextJob should skip the high priority job and return the low priority one
	job, err := dispatcher.FindNextJob(ctx, printerObj.ID)
	if err != nil {
		t.Fatalf("FindNextJob failed: %v", err)
	}
	if job == nil {
		t.Fatal("FindNextJob should return a job")
	}
	if job.ID != lowPriorityJob.ID {
		t.Error("FindNextJob should skip jobs with pending requests")
	}
}

func TestDispatcherService_CreateDispatchRequest(t *testing.T) {
	dispatcher, repos, _ := setupDispatcherTestEnv(t)
	ctx := context.Background()

	printerObj := createTestPrinter(t, repos, "Test Printer")
	job := createTestPrintJob(t, repos, model.PrintJobStatusQueued, 0, true)

	// Set timeout for printer
	settings := &model.AutoDispatchSettings{
		PrinterID:      printerObj.ID,
		Enabled:        true,
		TimeoutMinutes: 15,
	}
	repos.AutoDispatchSettings.Upsert(ctx, settings)

	request, err := dispatcher.CreateDispatchRequest(ctx, job.ID, printerObj.ID)
	if err != nil {
		t.Fatalf("CreateDispatchRequest failed: %v", err)
	}

	if request.ID == uuid.Nil {
		t.Error("Request ID should be set")
	}
	if request.JobID != job.ID {
		t.Errorf("JobID = %v, want %v", request.JobID, job.ID)
	}
	if request.PrinterID != printerObj.ID {
		t.Errorf("PrinterID = %v, want %v", request.PrinterID, printerObj.ID)
	}
	if request.Status != model.DispatchPending {
		t.Errorf("Status = %q, want %q", request.Status, model.DispatchPending)
	}

	// Check expiration is set correctly (15 minutes)
	expectedExpiry := time.Now().Add(15 * time.Minute)
	if request.ExpiresAt.Before(expectedExpiry.Add(-1*time.Minute)) || request.ExpiresAt.After(expectedExpiry.Add(1*time.Minute)) {
		t.Error("ExpiresAt should be approximately 15 minutes from now")
	}

	// Request should have enriched Job and Printer data
	if request.Job == nil {
		t.Error("Request should have Job populated")
	}
	if request.Printer == nil {
		t.Error("Request should have Printer populated")
	}
}

func TestDispatcherService_ConfirmDispatch(t *testing.T) {
	dispatcher, repos, _ := setupDispatcherTestEnv(t)
	ctx := context.Background()

	printerObj := createTestPrinter(t, repos, "Test Printer")
	job := createTestPrintJob(t, repos, model.PrintJobStatusQueued, 0, true)

	// Create dispatch request
	request, _ := dispatcher.CreateDispatchRequest(ctx, job.ID, printerObj.ID)

	// Confirm
	if err := dispatcher.ConfirmDispatch(ctx, request.ID); err != nil {
		t.Fatalf("ConfirmDispatch failed: %v", err)
	}

	// Check request status
	updatedReq, _ := repos.Dispatch.GetByID(ctx, request.ID)
	if updatedReq.Status != model.DispatchConfirmed {
		t.Errorf("Request status = %q, want %q", updatedReq.Status, model.DispatchConfirmed)
	}

	// Check job is assigned
	updatedJob, _ := repos.PrintJobs.GetByID(ctx, job.ID)
	if updatedJob.PrinterID == nil || *updatedJob.PrinterID != printerObj.ID {
		t.Error("Job should be assigned to printer after confirmation")
	}
}

func TestDispatcherService_ConfirmDispatch_NotFound(t *testing.T) {
	dispatcher, _, _ := setupDispatcherTestEnv(t)
	ctx := context.Background()

	err := dispatcher.ConfirmDispatch(ctx, uuid.New())
	if err == nil {
		t.Error("ConfirmDispatch should fail for non-existent request")
	}
}

func TestDispatcherService_ConfirmDispatch_NotPending(t *testing.T) {
	dispatcher, repos, _ := setupDispatcherTestEnv(t)
	ctx := context.Background()

	printerObj := createTestPrinter(t, repos, "Test Printer")
	job := createTestPrintJob(t, repos, model.PrintJobStatusQueued, 0, true)

	// Create and confirm a request
	request, _ := dispatcher.CreateDispatchRequest(ctx, job.ID, printerObj.ID)
	dispatcher.ConfirmDispatch(ctx, request.ID)

	// Try to confirm again
	err := dispatcher.ConfirmDispatch(ctx, request.ID)
	if err == nil {
		t.Error("ConfirmDispatch should fail for already confirmed request")
	}
}

func TestDispatcherService_RejectDispatch(t *testing.T) {
	dispatcher, repos, _ := setupDispatcherTestEnv(t)
	ctx := context.Background()

	printerObj := createTestPrinter(t, repos, "Test Printer")
	job := createTestPrintJob(t, repos, model.PrintJobStatusQueued, 0, true)

	// Create dispatch request
	request, _ := dispatcher.CreateDispatchRequest(ctx, job.ID, printerObj.ID)

	// Reject
	reason := "Bed not clear"
	if err := dispatcher.RejectDispatch(ctx, request.ID, reason); err != nil {
		t.Fatalf("RejectDispatch failed: %v", err)
	}

	// Check request status
	updatedReq, _ := repos.Dispatch.GetByID(ctx, request.ID)
	if updatedReq.Status != model.DispatchRejected {
		t.Errorf("Request status = %q, want %q", updatedReq.Status, model.DispatchRejected)
	}
	if updatedReq.Reason != reason {
		t.Errorf("Request reason = %q, want %q", updatedReq.Reason, reason)
	}

	// Job should not be assigned
	updatedJob, _ := repos.PrintJobs.GetByID(ctx, job.ID)
	if updatedJob.PrinterID != nil {
		t.Error("Job should not be assigned after rejection")
	}
}

func TestDispatcherService_SkipJob(t *testing.T) {
	dispatcher, repos, _ := setupDispatcherTestEnv(t)
	ctx := context.Background()

	printerObj := createTestPrinter(t, repos, "Test Printer")
	job := createTestPrintJob(t, repos, model.PrintJobStatusQueued, 0, true)

	// Enable auto-dispatch globally and for printer
	dispatcher.SetGlobalEnabled(ctx, true)
	dispatcher.UpdateSettings(ctx, &model.AutoDispatchSettings{
		PrinterID:           printerObj.ID,
		Enabled:             true,
		RequireConfirmation: true,
		TimeoutMinutes:      30,
	})

	// Create dispatch request
	request, _ := dispatcher.CreateDispatchRequest(ctx, job.ID, printerObj.ID)

	// Skip
	if err := dispatcher.SkipJob(ctx, request.ID); err != nil {
		t.Fatalf("SkipJob failed: %v", err)
	}

	// Check original request is rejected
	updatedReq, _ := repos.Dispatch.GetByID(ctx, request.ID)
	if updatedReq.Status != model.DispatchRejected {
		t.Errorf("Request status = %q, want %q", updatedReq.Status, model.DispatchRejected)
	}
	if updatedReq.Reason != "skipped" {
		t.Errorf("Request reason = %q, want 'skipped'", updatedReq.Reason)
	}

	// Job should have auto_dispatch_enabled=false
	updatedJob, _ := repos.PrintJobs.GetByID(ctx, job.ID)
	if updatedJob.AutoDispatchEnabled {
		t.Error("Job should have auto_dispatch_enabled=false after skip")
	}
}

func TestDispatcherService_ListPending(t *testing.T) {
	dispatcher, repos, _ := setupDispatcherTestEnv(t)
	ctx := context.Background()

	printerObj := createTestPrinter(t, repos, "Test Printer")

	// Create multiple pending requests
	for i := 0; i < 3; i++ {
		job := createTestPrintJob(t, repos, model.PrintJobStatusQueued, i, true)
		dispatcher.CreateDispatchRequest(ctx, job.ID, printerObj.ID)
	}

	// List pending
	requests, err := dispatcher.ListPending(ctx)
	if err != nil {
		t.Fatalf("ListPending failed: %v", err)
	}
	if len(requests) != 3 {
		t.Errorf("len(requests) = %d, want 3", len(requests))
	}

	// All should be enriched with Job and Printer
	for i, req := range requests {
		if req.Job == nil {
			t.Errorf("Request %d should have Job populated", i)
		}
		if req.Printer == nil {
			t.Errorf("Request %d should have Printer populated", i)
		}
	}
}

func TestDispatcherService_OnPrinterIdle_GloballyDisabled(t *testing.T) {
	dispatcher, repos, _ := setupDispatcherTestEnv(t)
	ctx := context.Background()

	printerObj := createTestPrinter(t, repos, "Test Printer")
	createTestPrintJob(t, repos, model.PrintJobStatusQueued, 0, true)

	// Enable printer-specific settings but keep global disabled
	dispatcher.UpdateSettings(ctx, &model.AutoDispatchSettings{
		PrinterID: printerObj.ID,
		Enabled:   true,
	})

	// Call OnPrinterIdle
	if err := dispatcher.OnPrinterIdle(printerObj.ID); err != nil {
		t.Fatalf("OnPrinterIdle failed: %v", err)
	}

	// No dispatch request should be created
	pending, _ := repos.Dispatch.ListPending(ctx)
	if len(pending) != 0 {
		t.Error("No dispatch request should be created when globally disabled")
	}
}

func TestDispatcherService_OnPrinterIdle_PrinterDisabled(t *testing.T) {
	dispatcher, repos, _ := setupDispatcherTestEnv(t)
	ctx := context.Background()

	printerObj := createTestPrinter(t, repos, "Test Printer")
	createTestPrintJob(t, repos, model.PrintJobStatusQueued, 0, true)

	// Enable global but keep printer disabled
	dispatcher.SetGlobalEnabled(ctx, true)

	// Call OnPrinterIdle
	if err := dispatcher.OnPrinterIdle(printerObj.ID); err != nil {
		t.Fatalf("OnPrinterIdle failed: %v", err)
	}

	// No dispatch request should be created
	pending, _ := repos.Dispatch.ListPending(ctx)
	if len(pending) != 0 {
		t.Error("No dispatch request should be created when printer is disabled")
	}
}

func TestDispatcherService_OnPrinterIdle_CreatesRequest(t *testing.T) {
	dispatcher, repos, _ := setupDispatcherTestEnv(t)
	ctx := context.Background()

	printerObj := createTestPrinter(t, repos, "Test Printer")
	job := createTestPrintJob(t, repos, model.PrintJobStatusQueued, 0, true)

	// Enable globally and for printer
	dispatcher.SetGlobalEnabled(ctx, true)
	dispatcher.UpdateSettings(ctx, &model.AutoDispatchSettings{
		PrinterID:      printerObj.ID,
		Enabled:        true,
		TimeoutMinutes: 30,
	})

	// Call OnPrinterIdle
	if err := dispatcher.OnPrinterIdle(printerObj.ID); err != nil {
		t.Fatalf("OnPrinterIdle failed: %v", err)
	}

	// A dispatch request should be created
	pending, _ := repos.Dispatch.ListPending(ctx)
	if len(pending) != 1 {
		t.Fatalf("Expected 1 dispatch request, got %d", len(pending))
	}
	if pending[0].JobID != job.ID {
		t.Error("Dispatch request should be for the queued job")
	}
}

func TestDispatcherService_OnPrinterIdle_SkipsIfPendingExists(t *testing.T) {
	dispatcher, repos, _ := setupDispatcherTestEnv(t)
	ctx := context.Background()

	printerObj := createTestPrinter(t, repos, "Test Printer")
	job := createTestPrintJob(t, repos, model.PrintJobStatusQueued, 0, true)

	// Enable globally and for printer
	dispatcher.SetGlobalEnabled(ctx, true)
	dispatcher.UpdateSettings(ctx, &model.AutoDispatchSettings{
		PrinterID:      printerObj.ID,
		Enabled:        true,
		TimeoutMinutes: 30,
	})

	// Create existing pending request
	existingReq := &model.DispatchRequest{
		JobID:     job.ID,
		PrinterID: printerObj.ID,
		ExpiresAt: time.Now().Add(30 * time.Minute),
	}
	repos.Dispatch.Create(ctx, existingReq)

	// Call OnPrinterIdle
	if err := dispatcher.OnPrinterIdle(printerObj.ID); err != nil {
		t.Fatalf("OnPrinterIdle failed: %v", err)
	}

	// Should still only have 1 pending request
	pending, _ := repos.Dispatch.ListPending(ctx)
	if len(pending) != 1 {
		t.Errorf("Expected 1 dispatch request (the existing one), got %d", len(pending))
	}
}

func TestDispatcherService_OnPrinterIdle_NoCompatibleJob(t *testing.T) {
	dispatcher, repos, _ := setupDispatcherTestEnv(t)
	ctx := context.Background()

	printerObj := createTestPrinter(t, repos, "Test Printer")

	// Create job with auto_dispatch_enabled=false
	createTestPrintJob(t, repos, model.PrintJobStatusQueued, 0, false)

	// Enable globally and for printer
	dispatcher.SetGlobalEnabled(ctx, true)
	dispatcher.UpdateSettings(ctx, &model.AutoDispatchSettings{
		PrinterID:      printerObj.ID,
		Enabled:        true,
		TimeoutMinutes: 30,
	})

	// Call OnPrinterIdle
	if err := dispatcher.OnPrinterIdle(printerObj.ID); err != nil {
		t.Fatalf("OnPrinterIdle failed: %v", err)
	}

	// No dispatch request should be created
	pending, _ := repos.Dispatch.ListPending(ctx)
	if len(pending) != 0 {
		t.Error("No dispatch request should be created when no compatible job exists")
	}
}
