package api

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/philjestin/daedalus/internal/model"
)

func TestDispatchHandler_ListPending_Empty(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/dispatch/requests", nil)
	rr := httptest.NewRecorder()

	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusOK)
	}

	var requests []model.DispatchRequest
	if err := json.NewDecoder(rr.Body).Decode(&requests); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(requests) != 0 {
		t.Errorf("Expected empty list, got %d requests", len(requests))
	}
}

func TestDispatchHandler_ListPending_WithRequests(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// Create a printer
	printer := &model.Printer{Name: "Test Printer"}
	env.services.Printers.Create(ctx, printer)

	// Create required data for a job
	project := &model.Project{Name: "Test Project"}
	env.services.Projects.Create(ctx, project)

	part := &model.Part{ProjectID: project.ID, Name: "Test Part"}
	env.services.Parts.Create(ctx, part)

	// Create a design (using storage)
	design, _ := createTestDesign(t, env, part.ID)

	// Create a job
	job := &model.PrintJob{
		DesignID:            design.ID,
		ProjectID:           &project.ID,
		Status:              model.PrintJobStatusQueued,
		Priority:            0,
		AutoDispatchEnabled: true,
	}
	env.services.PrintJobs.Create(ctx, job)

	// Create dispatch request directly
	dispatchReq, _ := env.services.Dispatcher.CreateDispatchRequest(ctx, job.ID, printer.ID)

	req := httptest.NewRequest(http.MethodGet, "/api/dispatch/requests", nil)
	rr := httptest.NewRecorder()

	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusOK)
	}

	var requests []model.DispatchRequest
	json.NewDecoder(rr.Body).Decode(&requests)

	if len(requests) != 1 {
		t.Fatalf("Expected 1 request, got %d", len(requests))
	}
	if requests[0].ID != dispatchReq.ID {
		t.Error("Returned request ID doesn't match")
	}
}

func TestDispatchHandler_Confirm(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// Setup printer and job
	printer := &model.Printer{Name: "Test Printer"}
	env.services.Printers.Create(ctx, printer)

	project := &model.Project{Name: "Test Project"}
	env.services.Projects.Create(ctx, project)

	part := &model.Part{ProjectID: project.ID, Name: "Test Part"}
	env.services.Parts.Create(ctx, part)

	design, _ := createTestDesign(t, env, part.ID)

	job := &model.PrintJob{
		DesignID:            design.ID,
		ProjectID:           &project.ID,
		Status:              model.PrintJobStatusQueued,
		AutoDispatchEnabled: true,
	}
	env.services.PrintJobs.Create(ctx, job)

	// Create dispatch request
	dispatchReq, _ := env.services.Dispatcher.CreateDispatchRequest(ctx, job.ID, printer.ID)

	// Confirm the request
	req := httptest.NewRequest(http.MethodPost, "/api/dispatch/requests/"+dispatchReq.ID.String()+"/confirm", nil)
	rr := httptest.NewRecorder()

	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d. Body: %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	// Verify job is assigned
	updatedJob, _ := env.services.PrintJobs.GetByID(ctx, job.ID)
	if updatedJob.PrinterID == nil || *updatedJob.PrinterID != printer.ID {
		t.Error("Job should be assigned to printer after confirmation")
	}
}

func TestDispatchHandler_Confirm_InvalidID(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/api/dispatch/requests/invalid-uuid/confirm", nil)
	rr := httptest.NewRecorder()

	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestDispatchHandler_Confirm_NotFound(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/api/dispatch/requests/"+uuid.New().String()+"/confirm", nil)
	rr := httptest.NewRecorder()

	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d (not found should return bad request)", rr.Code, http.StatusBadRequest)
	}
}

func TestDispatchHandler_Reject(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// Setup
	printer := &model.Printer{Name: "Test Printer"}
	env.services.Printers.Create(ctx, printer)

	project := &model.Project{Name: "Test Project"}
	env.services.Projects.Create(ctx, project)

	part := &model.Part{ProjectID: project.ID, Name: "Test Part"}
	env.services.Parts.Create(ctx, part)

	design, _ := createTestDesign(t, env, part.ID)

	job := &model.PrintJob{
		DesignID:            design.ID,
		ProjectID:           &project.ID,
		Status:              model.PrintJobStatusQueued,
		AutoDispatchEnabled: true,
	}
	env.services.PrintJobs.Create(ctx, job)

	dispatchReq, _ := env.services.Dispatcher.CreateDispatchRequest(ctx, job.ID, printer.ID)

	// Reject with reason
	body := bytes.NewBufferString(`{"reason": "Bed not clear"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/dispatch/requests/"+dispatchReq.ID.String()+"/reject", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d. Body: %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	// Verify by listing pending - the rejected request should not appear
	pendingRequests, _ := env.services.Dispatcher.ListPending(ctx)
	for _, r := range pendingRequests {
		if r.ID == dispatchReq.ID {
			t.Error("Rejected request should not appear in pending list")
		}
	}
}

func TestDispatchHandler_Skip(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// Setup
	printer := &model.Printer{Name: "Test Printer"}
	env.services.Printers.Create(ctx, printer)

	project := &model.Project{Name: "Test Project"}
	env.services.Projects.Create(ctx, project)

	part := &model.Part{ProjectID: project.ID, Name: "Test Part"}
	env.services.Parts.Create(ctx, part)

	design, _ := createTestDesign(t, env, part.ID)

	job := &model.PrintJob{
		DesignID:            design.ID,
		ProjectID:           &project.ID,
		Status:              model.PrintJobStatusQueued,
		AutoDispatchEnabled: true,
	}
	env.services.PrintJobs.Create(ctx, job)

	dispatchReq, _ := env.services.Dispatcher.CreateDispatchRequest(ctx, job.ID, printer.ID)

	req := httptest.NewRequest(http.MethodPost, "/api/dispatch/requests/"+dispatchReq.ID.String()+"/skip", nil)
	rr := httptest.NewRecorder()

	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d. Body: %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	// Verify job has auto_dispatch_enabled=false
	updatedJob, _ := env.services.PrintJobs.GetByID(ctx, job.ID)
	if updatedJob.AutoDispatchEnabled {
		t.Error("Job should have auto_dispatch_enabled=false after skip")
	}
}

func TestDispatchHandler_GetGlobalSettings(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/dispatch/settings", nil)
	rr := httptest.NewRecorder()

	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusOK)
	}

	var result map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&result)

	if _, ok := result["enabled"]; !ok {
		t.Error("Response should include 'enabled' field")
	}
}

func TestDispatchHandler_UpdateGlobalSettings(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// Enable
	body := bytes.NewBufferString(`{"enabled": true}`)
	req := httptest.NewRequest(http.MethodPut, "/api/dispatch/settings", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusOK)
	}

	// Verify
	if !env.services.Dispatcher.IsGloballyEnabled(ctx) {
		t.Error("Auto-dispatch should be enabled")
	}

	// Disable
	body = bytes.NewBufferString(`{"enabled": false}`)
	req = httptest.NewRequest(http.MethodPut, "/api/dispatch/settings", body)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()

	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusOK)
	}

	if env.services.Dispatcher.IsGloballyEnabled(ctx) {
		t.Error("Auto-dispatch should be disabled")
	}
}

func TestDispatchHandler_GetPrinterSettings(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	printer := &model.Printer{Name: "Test Printer"}
	env.services.Printers.Create(ctx, printer)

	req := httptest.NewRequest(http.MethodGet, "/api/printers/"+printer.ID.String()+"/dispatch-settings", nil)
	rr := httptest.NewRecorder()

	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusOK)
	}

	var settings model.AutoDispatchSettings
	json.NewDecoder(rr.Body).Decode(&settings)

	if settings.PrinterID != printer.ID {
		t.Errorf("PrinterID = %v, want %v", settings.PrinterID, printer.ID)
	}
	// Should return defaults
	if settings.Enabled != false {
		t.Error("New printer should have Enabled=false by default")
	}
	if settings.TimeoutMinutes != 30 {
		t.Errorf("TimeoutMinutes = %d, want 30", settings.TimeoutMinutes)
	}
}

func TestDispatchHandler_UpdatePrinterSettings(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	printer := &model.Printer{Name: "Test Printer"}
	env.services.Printers.Create(ctx, printer)

	body := bytes.NewBufferString(`{
		"enabled": true,
		"require_confirmation": false,
		"auto_start": true,
		"timeout_minutes": 60
	}`)
	req := httptest.NewRequest(http.MethodPut, "/api/printers/"+printer.ID.String()+"/dispatch-settings", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status = %d, want %d. Body: %s", rr.Code, http.StatusOK, rr.Body.String())
	}

	// Verify
	settings, _ := env.services.Dispatcher.GetSettings(ctx, printer.ID)
	if !settings.Enabled {
		t.Error("Enabled should be true")
	}
	if settings.RequireConfirmation {
		t.Error("RequireConfirmation should be false")
	}
	if !settings.AutoStart {
		t.Error("AutoStart should be true")
	}
	if settings.TimeoutMinutes != 60 {
		t.Errorf("TimeoutMinutes = %d, want 60", settings.TimeoutMinutes)
	}
}

func TestDispatchHandler_UpdatePrinterSettings_InvalidID(t *testing.T) {
	env := newTestEnv(t)

	body := bytes.NewBufferString(`{"enabled": true}`)
	req := httptest.NewRequest(http.MethodPut, "/api/printers/invalid-uuid/dispatch-settings", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestDispatchHandler_UpdatePrinterSettings_InvalidBody(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	printer := &model.Printer{Name: "Test Printer"}
	env.services.Printers.Create(ctx, printer)

	body := bytes.NewBufferString(`invalid json`)
	req := httptest.NewRequest(http.MethodPut, "/api/printers/"+printer.ID.String()+"/dispatch-settings", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

// createTestDesign creates a part with a design using the multipart API.
func createTestDesign(t *testing.T, env *testEnv, partID uuid.UUID) (*model.Design, error) {
	t.Helper()

	// Upload a new design to the existing part via the API
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	w.WriteField("notes", "test design")
	fw, _ := w.CreateFormFile("file", "test.3mf")
	fw.Write([]byte("fake 3mf content"))
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/parts/"+partID.String()+"/designs", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Create design failed: %d: %s", rr.Code, rr.Body.String())
	}

	var design model.Design
	if err := json.NewDecoder(rr.Body).Decode(&design); err != nil {
		t.Fatalf("Decode design failed: %v", err)
	}

	return &design, nil
}
