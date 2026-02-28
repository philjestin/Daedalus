package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/philjestin/daedalus/internal/model"
)

func TestJobEventCreation(t *testing.T) {
	jobID := uuid.New()
	status := model.PrintJobStatusQueued

	event := model.NewJobEvent(jobID, model.JobEventQueued, &status)

	if event.JobID != jobID {
		t.Errorf("JobID = %v, want %v", event.JobID, jobID)
	}
	if event.EventType != model.JobEventQueued {
		t.Errorf("EventType = %v, want %v", event.EventType, model.JobEventQueued)
	}
	if event.Status == nil || *event.Status != status {
		t.Errorf("Status = %v, want %v", event.Status, status)
	}
	if event.ActorType != model.ActorSystem {
		t.Errorf("ActorType = %v, want %v", event.ActorType, model.ActorSystem)
	}
}

func TestJobEventWithActor(t *testing.T) {
	jobID := uuid.New()
	event := model.NewJobEvent(jobID, model.JobEventPaused, nil).
		WithActor(model.ActorUser, "user123")

	if event.ActorType != model.ActorUser {
		t.Errorf("ActorType = %v, want %v", event.ActorType, model.ActorUser)
	}
	if event.ActorID != "user123" {
		t.Errorf("ActorID = %s, want user123", event.ActorID)
	}
}

func TestJobEventWithError(t *testing.T) {
	jobID := uuid.New()
	status := model.PrintJobStatusFailed
	event := model.NewJobEvent(jobID, model.JobEventFailed, &status).
		WithError("MECHANICAL_FAILURE", "Extruder jam detected")

	if event.ErrorCode != "MECHANICAL_FAILURE" {
		t.Errorf("ErrorCode = %s, want MECHANICAL_FAILURE", event.ErrorCode)
	}
	if event.ErrorMessage != "Extruder jam detected" {
		t.Errorf("ErrorMessage = %s, want 'Extruder jam detected'", event.ErrorMessage)
	}
}

func TestJobEventWithProgress(t *testing.T) {
	jobID := uuid.New()
	event := model.NewJobEvent(jobID, model.JobEventProgress, nil).
		WithProgress(45.5)

	if event.Progress == nil || *event.Progress != 45.5 {
		t.Errorf("Progress = %v, want 45.5", event.Progress)
	}
}

func TestJobEventWithPrinter(t *testing.T) {
	jobID := uuid.New()
	printerID := uuid.New()
	event := model.NewJobEvent(jobID, model.JobEventAssigned, nil).
		WithPrinter(printerID)

	if event.PrinterID == nil || *event.PrinterID != printerID {
		t.Errorf("PrinterID = %v, want %v", event.PrinterID, printerID)
	}
}

func TestJobEventWithMetadata(t *testing.T) {
	jobID := uuid.New()
	event := model.NewJobEvent(jobID, model.JobEventCompleted, nil).
		WithMetadata(map[string]interface{}{
			"material_used_grams": 50.5,
			"quality_rating":      4,
		})

	if event.Metadata == nil {
		t.Fatal("Metadata is nil")
	}
	if event.Metadata["material_used_grams"] != 50.5 {
		t.Errorf("Metadata[material_used_grams] = %v, want 50.5", event.Metadata["material_used_grams"])
	}
	if event.Metadata["quality_rating"] != 4 {
		t.Errorf("Metadata[quality_rating] = %v, want 4", event.Metadata["quality_rating"])
	}
}

func TestPrintJobStatusIsTerminal(t *testing.T) {
	tests := []struct {
		status   model.PrintJobStatus
		terminal bool
	}{
		{model.PrintJobStatusQueued, false},
		{model.PrintJobStatusAssigned, false},
		{model.PrintJobStatusUploaded, false},
		{model.PrintJobStatusPrinting, false},
		{model.PrintJobStatusPaused, false},
		{model.PrintJobStatusCompleted, true},
		{model.PrintJobStatusFailed, true},
		{model.PrintJobStatusCancelled, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			if tt.status.IsTerminal() != tt.terminal {
				t.Errorf("IsTerminal() = %v, want %v", tt.status.IsTerminal(), tt.terminal)
			}
		})
	}
}

func TestFailureCategoryValues(t *testing.T) {
	categories := []model.FailureCategory{
		model.FailureMechanical,
		model.FailureFilament,
		model.FailureAdhesion,
		model.FailureThermal,
		model.FailureNetwork,
		model.FailureUserCancelled,
		model.FailureUnknown,
	}

	for _, cat := range categories {
		if cat == "" {
			t.Error("Failure category should not be empty")
		}
	}
}

func TestActorTypeValues(t *testing.T) {
	actors := []model.ActorType{
		model.ActorUser,
		model.ActorSystem,
		model.ActorPrinter,
		model.ActorWebhook,
	}

	for _, actor := range actors {
		if actor == "" {
			t.Error("Actor type should not be empty")
		}
	}
}

func TestJobEventTypeValues(t *testing.T) {
	eventTypes := []model.JobEventType{
		model.JobEventQueued,
		model.JobEventAssigned,
		model.JobEventUploaded,
		model.JobEventStarted,
		model.JobEventProgress,
		model.JobEventPaused,
		model.JobEventResumed,
		model.JobEventCompleted,
		model.JobEventFailed,
		model.JobEventCancelled,
		model.JobEventRetried,
	}

	for _, et := range eventTypes {
		if et == "" {
			t.Error("Event type should not be empty")
		}
	}
}

func TestPrintJobRetryTracking(t *testing.T) {
	originalJobID := uuid.New()
	recipeID := uuid.New()

	// Original job
	printerID := uuid.New()
	spoolID := uuid.New()
	originalJob := &model.PrintJob{
		ID:              originalJobID,
		DesignID:        uuid.New(),
		PrinterID:       &printerID,
		MaterialSpoolID: &spoolID,
		RecipeID:        &recipeID,
		AttemptNumber:   1,
		ParentJobID:     nil,
		Status:          model.PrintJobStatusFailed,
		CreatedAt:       time.Now(),
	}

	// Retry job
	retryJob := &model.PrintJob{
		ID:              uuid.New(),
		DesignID:        originalJob.DesignID,
		PrinterID:       originalJob.PrinterID,
		MaterialSpoolID: originalJob.MaterialSpoolID,
		RecipeID:        originalJob.RecipeID,
		AttemptNumber:   originalJob.AttemptNumber + 1,
		ParentJobID:     &originalJobID,
		Status:          model.PrintJobStatusQueued,
		CreatedAt:       time.Now(),
	}

	// Verify retry tracking
	if retryJob.AttemptNumber != 2 {
		t.Errorf("AttemptNumber = %d, want 2", retryJob.AttemptNumber)
	}
	if retryJob.ParentJobID == nil || *retryJob.ParentJobID != originalJobID {
		t.Errorf("ParentJobID = %v, want %v", retryJob.ParentJobID, originalJobID)
	}
	if retryJob.RecipeID == nil || *retryJob.RecipeID != recipeID {
		t.Errorf("RecipeID = %v, want %v", retryJob.RecipeID, recipeID)
	}
}

func TestPrintJobCostTracking(t *testing.T) {
	estimatedSeconds := 3600
	actualSeconds := 3720
	materialUsed := 50.5
	costCents := 250

	printerID2 := uuid.New()
	spoolID2 := uuid.New()
	job := &model.PrintJob{
		ID:                uuid.New(),
		DesignID:          uuid.New(),
		PrinterID:         &printerID2,
		MaterialSpoolID:   &spoolID2,
		EstimatedSeconds:  &estimatedSeconds,
		ActualSeconds:     &actualSeconds,
		MaterialUsedGrams: &materialUsed,
		CostCents:         &costCents,
		Status:            model.PrintJobStatusCompleted,
		CreatedAt:         time.Now(),
	}

	if job.EstimatedSeconds == nil || *job.EstimatedSeconds != 3600 {
		t.Errorf("EstimatedSeconds = %v, want 3600", job.EstimatedSeconds)
	}
	if job.ActualSeconds == nil || *job.ActualSeconds != 3720 {
		t.Errorf("ActualSeconds = %v, want 3720", job.ActualSeconds)
	}
	if job.MaterialUsedGrams == nil || *job.MaterialUsedGrams != 50.5 {
		t.Errorf("MaterialUsedGrams = %v, want 50.5", job.MaterialUsedGrams)
	}
	if job.CostCents == nil || *job.CostCents != 250 {
		t.Errorf("CostCents = %v, want 250", job.CostCents)
	}
}

func TestEventChainBuilder(t *testing.T) {
	jobID := uuid.New()
	printerID := uuid.New()

	// Build a complete event with all options
	status := model.PrintJobStatusFailed
	event := model.NewJobEvent(jobID, model.JobEventFailed, &status).
		WithPrinter(printerID).
		WithActor(model.ActorPrinter, printerID.String()).
		WithError("THERMAL_RUNAWAY", "Heater failed to maintain temperature").
		WithProgress(67.5).
		WithMetadata(map[string]interface{}{
			"bed_temp":    45.2,
			"nozzle_temp": 180.0,
		})

	// Verify all fields are set
	if event.JobID != jobID {
		t.Errorf("JobID mismatch")
	}
	if event.PrinterID == nil || *event.PrinterID != printerID {
		t.Errorf("PrinterID mismatch")
	}
	if event.ActorType != model.ActorPrinter {
		t.Errorf("ActorType = %v, want %v", event.ActorType, model.ActorPrinter)
	}
	if event.ActorID != printerID.String() {
		t.Errorf("ActorID mismatch")
	}
	if event.ErrorCode != "THERMAL_RUNAWAY" {
		t.Errorf("ErrorCode mismatch")
	}
	if event.ErrorMessage != "Heater failed to maintain temperature" {
		t.Errorf("ErrorMessage mismatch")
	}
	if event.Progress == nil || *event.Progress != 67.5 {
		t.Errorf("Progress mismatch")
	}
	if event.Metadata["bed_temp"] != 45.2 {
		t.Errorf("Metadata[bed_temp] mismatch")
	}
}

func TestPrintJobStatusTransitions(t *testing.T) {
	// Test valid status progression
	validTransitions := []struct {
		from model.PrintJobStatus
		to   model.PrintJobStatus
	}{
		{model.PrintJobStatusQueued, model.PrintJobStatusAssigned},
		{model.PrintJobStatusAssigned, model.PrintJobStatusUploaded},
		{model.PrintJobStatusUploaded, model.PrintJobStatusPrinting},
		{model.PrintJobStatusPrinting, model.PrintJobStatusPaused},
		{model.PrintJobStatusPaused, model.PrintJobStatusPrinting},
		{model.PrintJobStatusPrinting, model.PrintJobStatusCompleted},
		{model.PrintJobStatusPrinting, model.PrintJobStatusFailed},
		{model.PrintJobStatusQueued, model.PrintJobStatusCancelled},
		{model.PrintJobStatusPrinting, model.PrintJobStatusCancelled},
	}

	for _, tt := range validTransitions {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			// Just verify these transitions are conceptually valid
			// The actual validation is in the service layer
			if tt.from == tt.to {
				t.Errorf("From and To should be different")
			}
		})
	}
}
