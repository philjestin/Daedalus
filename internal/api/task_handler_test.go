package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/philjestin/daedalus/internal/model"
)

func TestPrintFromChecklist_Success(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// Create project, part, design, task
	project := &model.Project{Name: "Test Project"}
	env.services.Projects.Create(ctx, project)

	part := &model.Part{ProjectID: project.ID, Name: "Base Plate"}
	env.services.Parts.Create(ctx, part)

	design, _ := createTestDesign(t, env, part.ID)

	task := &model.Task{
		ProjectID: project.ID,
		Name:      "Build Test Project",
		Quantity:  1,
	}
	env.services.Tasks.Create(ctx, task)

	// Get checklist items (auto-generated from parts)
	checklist, err := env.services.Tasks.GetChecklist(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetChecklist: %v", err)
	}

	// Find the checklist item linked to our part
	var partItem *model.TaskChecklistItem
	for i := range checklist {
		if checklist[i].PartID != nil && *checklist[i].PartID == part.ID {
			partItem = &checklist[i]
			break
		}
	}
	if partItem == nil {
		t.Fatal("No checklist item found with part_id")
	}

	// POST /api/tasks/{id}/checklist/{itemId}/print
	url := fmt.Sprintf("/api/tasks/%s/checklist/%s/print", task.ID, partItem.ID)
	req := httptest.NewRequest(http.MethodPost, url, nil)
	rr := httptest.NewRecorder()

	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Status = %d, want %d, body: %s", rr.Code, http.StatusCreated, rr.Body.String())
	}

	var job model.PrintJob
	if err := json.NewDecoder(rr.Body).Decode(&job); err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if job.DesignID != design.ID {
		t.Errorf("DesignID = %s, want %s", job.DesignID, design.ID)
	}
	if job.TaskID == nil || *job.TaskID != task.ID {
		t.Errorf("TaskID = %v, want %s", job.TaskID, task.ID)
	}
	if job.ProjectID == nil || *job.ProjectID != project.ID {
		t.Errorf("ProjectID = %v, want %s", job.ProjectID, project.ID)
	}

	// Task should have been auto-started
	updatedTask, _ := env.services.Tasks.GetByID(ctx, task.ID)
	if updatedTask.Status != model.TaskStatusInProgress {
		t.Errorf("Task status = %s, want in_progress", updatedTask.Status)
	}
}

func TestPrintFromChecklist_NoPartID(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// Create project with a part so checklist is generated, but target the Assembly item (no part_id)
	project := &model.Project{Name: "Test Project"}
	env.services.Projects.Create(ctx, project)

	part := &model.Part{ProjectID: project.ID, Name: "Widget"}
	env.services.Parts.Create(ctx, part)

	createTestDesign(t, env, part.ID)

	task := &model.Task{
		ProjectID: project.ID,
		Name:      "Build Widget",
		Quantity:  1,
	}
	env.services.Tasks.Create(ctx, task)

	checklist, _ := env.services.Tasks.GetChecklist(ctx, task.ID)

	// Find the Assembly item (no part_id)
	var assemblyItem *model.TaskChecklistItem
	for i := range checklist {
		if checklist[i].PartID == nil {
			assemblyItem = &checklist[i]
			break
		}
	}
	if assemblyItem == nil {
		t.Fatal("No Assembly checklist item found")
	}

	url := fmt.Sprintf("/api/tasks/%s/checklist/%s/print", task.ID, assemblyItem.ID)
	req := httptest.NewRequest(http.MethodPost, url, nil)
	rr := httptest.NewRecorder()

	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestPrintFromChecklist_InvalidIDs(t *testing.T) {
	env := newTestEnv(t)

	// Invalid task ID
	req := httptest.NewRequest(http.MethodPost, "/api/tasks/not-a-uuid/checklist/"+uuid.New().String()+"/print", nil)
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Invalid task ID: Status = %d, want %d", rr.Code, http.StatusBadRequest)
	}

	// Invalid item ID
	req = httptest.NewRequest(http.MethodPost, "/api/tasks/"+uuid.New().String()+"/checklist/not-a-uuid/print", nil)
	rr = httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Errorf("Invalid item ID: Status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}

func TestHandleJobCompleted_AutoCompletesChecklist(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// Create project, part, design, task
	project := &model.Project{Name: "Test Project"}
	env.services.Projects.Create(ctx, project)

	part := &model.Part{ProjectID: project.ID, Name: "Frame"}
	env.services.Parts.Create(ctx, part)

	createTestDesign(t, env, part.ID)

	task := &model.Task{
		ProjectID: project.ID,
		Name:      "Build Frame",
		Quantity:  1,
	}
	env.services.Tasks.Create(ctx, task)

	// Get the part checklist item
	checklist, _ := env.services.Tasks.GetChecklist(ctx, task.ID)
	var partItem *model.TaskChecklistItem
	for i := range checklist {
		if checklist[i].PartID != nil && *checklist[i].PartID == part.ID {
			partItem = &checklist[i]
			break
		}
	}
	if partItem == nil {
		t.Fatal("No part checklist item found")
	}

	// Create a print job from checklist
	url := fmt.Sprintf("/api/tasks/%s/checklist/%s/print", task.ID, partItem.ID)
	req := httptest.NewRequest(http.MethodPost, url, nil)
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("PrintFromChecklist failed: %d: %s", rr.Code, rr.Body.String())
	}

	var job model.PrintJob
	json.NewDecoder(rr.Body).Decode(&job)

	// Record successful outcome
	outcomeURL := fmt.Sprintf("/api/print-jobs/%s/outcome", job.ID)
	outcomeBody, _ := json.Marshal(map[string]interface{}{
		"success":       true,
		"material_used": 15.0,
	})
	req = httptest.NewRequest(http.MethodPost, outcomeURL, bytes.NewReader(outcomeBody))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("RecordOutcome failed: %d: %s", rr.Code, rr.Body.String())
	}

	// Verify checklist item was auto-completed
	updatedChecklist, _ := env.services.Tasks.GetChecklist(ctx, task.ID)
	for _, item := range updatedChecklist {
		if item.ID == partItem.ID {
			if !item.Completed {
				t.Error("Checklist item should have been auto-completed")
			}
			return
		}
	}
	t.Error("Checklist item not found after job completion")
}
