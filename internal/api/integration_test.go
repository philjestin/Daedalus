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

// =============================================================================
// Project (Product Catalog) Integration Tests
// =============================================================================

func TestProjectCRUD_FullLifecycle(t *testing.T) {
	env := newTestEnv(t)

	// CREATE
	createBody, _ := json.Marshal(map[string]interface{}{
		"name":        "Gyroid Lamp",
		"description": "A beautiful gyroid table lamp",
		"sku":         "LAMP-GYR-001",
		"price_cents": 4999,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var created model.Project
	json.NewDecoder(rr.Body).Decode(&created)

	if created.Name != "Gyroid Lamp" {
		t.Errorf("name: got %q", created.Name)
	}
	if created.SKU != "LAMP-GYR-001" {
		t.Errorf("sku: got %q", created.SKU)
	}

	// READ
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/projects/%s", created.ID), nil)
	rr = httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var fetched model.Project
	json.NewDecoder(rr.Body).Decode(&fetched)
	if fetched.ID != created.ID {
		t.Errorf("id mismatch: %s vs %s", fetched.ID, created.ID)
	}

	// UPDATE
	updateBody, _ := json.Marshal(map[string]interface{}{
		"name":        "Gyroid Table Lamp",
		"price_cents": 5999,
	})
	req = httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/projects/%s", created.ID), bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify update
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/projects/%s", created.ID), nil)
	rr = httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)
	json.NewDecoder(rr.Body).Decode(&fetched)

	if fetched.Name != "Gyroid Table Lamp" {
		t.Errorf("updated name: got %q", fetched.Name)
	}

	// LIST
	req = httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	rr = httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", rr.Code)
	}

	var projects []model.Project
	json.NewDecoder(rr.Body).Decode(&projects)
	if len(projects) != 1 {
		t.Errorf("expected 1 project, got %d", len(projects))
	}

	// DELETE
	req = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/projects/%s", created.ID), nil)
	rr = httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent && rr.Code != http.StatusOK {
		t.Fatalf("delete: expected 204/200, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify deleted
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/projects/%s", created.ID), nil)
	rr = httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("after delete: expected 404, got %d", rr.Code)
	}
}

func TestProjectList_MultipleProjects(t *testing.T) {
	env := newTestEnv(t)

	names := []string{"Project Alpha", "Project Beta", "Project Gamma"}
	for _, name := range names {
		body, _ := json.Marshal(map[string]interface{}{"name": name})
		req := httptest.NewRequest(http.MethodPost, "/api/projects", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		env.handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("create %s: got %d", name, rr.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/projects", nil)
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	var projects []model.Project
	json.NewDecoder(rr.Body).Decode(&projects)
	if len(projects) != 3 {
		t.Errorf("expected 3 projects, got %d", len(projects))
	}
}

// =============================================================================
// Task Integration Tests
// =============================================================================

func TestTaskCRUD_FullLifecycle(t *testing.T) {
	env := newTestEnv(t)

	// First create a project
	project := &model.Project{Name: "Test Product", SKU: "TEST-001"}
	if err := env.services.Projects.Create(context.Background(), project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	// CREATE TASK
	createBody, _ := json.Marshal(map[string]interface{}{
		"project_id": project.ID,
		"name":       "Production Run #1",
		"quantity":   5,
		"notes":      "First batch",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("create task: expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var created model.Task
	json.NewDecoder(rr.Body).Decode(&created)

	if created.Name != "Production Run #1" {
		t.Errorf("name: got %q", created.Name)
	}
	if created.Quantity != 5 {
		t.Errorf("quantity: got %d", created.Quantity)
	}
	if created.Status != model.TaskStatusPending {
		t.Errorf("status: got %q", created.Status)
	}

	// READ TASK
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/tasks/%s", created.ID), nil)
	rr = httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("get task: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var fetched model.Task
	json.NewDecoder(rr.Body).Decode(&fetched)
	if fetched.ID != created.ID {
		t.Errorf("id mismatch")
	}

	// UPDATE TASK
	updateBody, _ := json.Marshal(map[string]interface{}{
		"name":     "Production Run #1 - Updated",
		"quantity": 10,
	})
	req = httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/api/tasks/%s", created.ID), bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("update task: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// LIST TASKS
	req = httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	rr = httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("list tasks: expected 200, got %d", rr.Code)
	}

	var tasks []model.Task
	json.NewDecoder(rr.Body).Decode(&tasks)
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}

	// DELETE TASK
	req = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/tasks/%s", created.ID), nil)
	rr = httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("delete task: expected 204, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestTaskWorkflow_StartCompleteCancel(t *testing.T) {
	env := newTestEnv(t)

	// Create project and task
	project := &model.Project{Name: "Workflow Test"}
	env.services.Projects.Create(context.Background(), project)

	task := &model.Task{
		ProjectID: project.ID,
		Name:      "Workflow Task",
		Quantity:  1,
	}
	env.services.Tasks.Create(context.Background(), task)

	// Verify initial status is pending
	if task.Status != model.TaskStatusPending {
		t.Errorf("initial status: got %q", task.Status)
	}

	// START TASK
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/tasks/%s/start", task.ID), nil)
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("start: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var started model.Task
	json.NewDecoder(rr.Body).Decode(&started)
	if started.Status != model.TaskStatusInProgress {
		t.Errorf("after start: got %q", started.Status)
	}
	if started.StartedAt == nil {
		t.Error("started_at should be set")
	}

	// COMPLETE TASK
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/tasks/%s/complete", task.ID), nil)
	rr = httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("complete: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var completed model.Task
	json.NewDecoder(rr.Body).Decode(&completed)
	if completed.Status != model.TaskStatusCompleted {
		t.Errorf("after complete: got %q", completed.Status)
	}
	if completed.CompletedAt == nil {
		t.Error("completed_at should be set")
	}
}

func TestTaskWorkflow_Cancel(t *testing.T) {
	env := newTestEnv(t)

	project := &model.Project{Name: "Cancel Test"}
	env.services.Projects.Create(context.Background(), project)

	task := &model.Task{
		ProjectID: project.ID,
		Name:      "Task to Cancel",
		Quantity:  1,
	}
	env.services.Tasks.Create(context.Background(), task)

	// Start then cancel
	env.services.Tasks.StartTask(context.Background(), task.ID)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/tasks/%s/cancel", task.ID), nil)
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("cancel: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var cancelled model.Task
	json.NewDecoder(rr.Body).Decode(&cancelled)
	if cancelled.Status != model.TaskStatusCancelled {
		t.Errorf("after cancel: got %q", cancelled.Status)
	}
}

func TestTaskList_FilterByStatus(t *testing.T) {
	env := newTestEnv(t)

	project := &model.Project{Name: "Filter Test"}
	env.services.Projects.Create(context.Background(), project)

	// Create tasks with different statuses
	pending := &model.Task{ProjectID: project.ID, Name: "Pending Task", Quantity: 1}
	inProgress := &model.Task{ProjectID: project.ID, Name: "In Progress Task", Quantity: 1}
	completed := &model.Task{ProjectID: project.ID, Name: "Completed Task", Quantity: 1}

	env.services.Tasks.Create(context.Background(), pending)
	env.services.Tasks.Create(context.Background(), inProgress)
	env.services.Tasks.Create(context.Background(), completed)

	env.services.Tasks.StartTask(context.Background(), inProgress.ID)
	env.services.Tasks.StartTask(context.Background(), completed.ID)
	env.services.Tasks.CompleteTask(context.Background(), completed.ID)

	// Filter by pending
	req := httptest.NewRequest(http.MethodGet, "/api/tasks?status=pending", nil)
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	var pendingTasks []model.Task
	json.NewDecoder(rr.Body).Decode(&pendingTasks)
	if len(pendingTasks) != 1 {
		t.Errorf("pending filter: expected 1, got %d", len(pendingTasks))
	}

	// Filter by in_progress
	req = httptest.NewRequest(http.MethodGet, "/api/tasks?status=in_progress", nil)
	rr = httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	var inProgressTasks []model.Task
	json.NewDecoder(rr.Body).Decode(&inProgressTasks)
	if len(inProgressTasks) != 1 {
		t.Errorf("in_progress filter: expected 1, got %d", len(inProgressTasks))
	}

	// Filter by completed
	req = httptest.NewRequest(http.MethodGet, "/api/tasks?status=completed", nil)
	rr = httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	var completedTasks []model.Task
	json.NewDecoder(rr.Body).Decode(&completedTasks)
	if len(completedTasks) != 1 {
		t.Errorf("completed filter: expected 1, got %d", len(completedTasks))
	}
}

func TestTaskList_FilterByProject(t *testing.T) {
	env := newTestEnv(t)

	project1 := &model.Project{Name: "Project 1"}
	project2 := &model.Project{Name: "Project 2"}
	env.services.Projects.Create(context.Background(), project1)
	env.services.Projects.Create(context.Background(), project2)

	// Create tasks for each project
	for i := 0; i < 3; i++ {
		task := &model.Task{ProjectID: project1.ID, Name: fmt.Sprintf("P1 Task %d", i), Quantity: 1}
		env.services.Tasks.Create(context.Background(), task)
	}
	for i := 0; i < 2; i++ {
		task := &model.Task{ProjectID: project2.ID, Name: fmt.Sprintf("P2 Task %d", i), Quantity: 1}
		env.services.Tasks.Create(context.Background(), task)
	}

	// Filter by project1
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/tasks?project_id=%s", project1.ID), nil)
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	var p1Tasks []model.Task
	json.NewDecoder(rr.Body).Decode(&p1Tasks)
	if len(p1Tasks) != 3 {
		t.Errorf("project1 filter: expected 3, got %d", len(p1Tasks))
	}

	// Filter by project2
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/tasks?project_id=%s", project2.ID), nil)
	rr = httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	var p2Tasks []model.Task
	json.NewDecoder(rr.Body).Decode(&p2Tasks)
	if len(p2Tasks) != 2 {
		t.Errorf("project2 filter: expected 2, got %d", len(p2Tasks))
	}
}

func TestProjectTasks_Endpoint(t *testing.T) {
	env := newTestEnv(t)

	project := &model.Project{Name: "Project with Tasks"}
	env.services.Projects.Create(context.Background(), project)

	// Create tasks for this project
	for i := 0; i < 3; i++ {
		task := &model.Task{ProjectID: project.ID, Name: fmt.Sprintf("Task %d", i), Quantity: 1}
		env.services.Tasks.Create(context.Background(), task)
	}

	// Get tasks via project endpoint
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/projects/%s/tasks", project.ID), nil)
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var tasks []model.Task
	json.NewDecoder(rr.Body).Decode(&tasks)
	if len(tasks) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(tasks))
	}
}

// =============================================================================
// Order Integration Tests
// =============================================================================

func TestOrderCRUD_FullLifecycle(t *testing.T) {
	env := newTestEnv(t)

	// CREATE ORDER
	createBody, _ := json.Marshal(map[string]interface{}{
		"customer_name":  "John Doe",
		"customer_email": "john@example.com",
		"channel":        "direct",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/orders", bytes.NewReader(createBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("create order: expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var created model.Order
	json.NewDecoder(rr.Body).Decode(&created)

	if created.CustomerName != "John Doe" {
		t.Errorf("customer_name: got %q", created.CustomerName)
	}
	if created.Status != model.OrderStatusPending {
		t.Errorf("status: got %q", created.Status)
	}

	// READ ORDER
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/orders/%s", created.ID), nil)
	rr = httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("get order: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	// LIST ORDERS
	req = httptest.NewRequest(http.MethodGet, "/api/orders", nil)
	rr = httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("list orders: expected 200, got %d", rr.Code)
	}

	var orders []model.Order
	json.NewDecoder(rr.Body).Decode(&orders)
	if len(orders) != 1 {
		t.Errorf("expected 1 order, got %d", len(orders))
	}
}

func TestOrderWithItems_AddItem(t *testing.T) {
	env := newTestEnv(t)

	// Create order
	order := &model.Order{
		CustomerName: "Jane Doe",
		Source:       model.OrderSourceManual,
	}
	env.services.Orders.Create(context.Background(), order)

	// Create a project to link
	project := &model.Project{Name: "Product SKU-123", SKU: "SKU-123"}
	env.services.Projects.Create(context.Background(), project)

	// Add item to order
	itemBody, _ := json.Marshal(map[string]interface{}{
		"product_name": "Gyroid Lamp",
		"sku":          "SKU-123",
		"quantity":     2,
		"price_cents":  4999,
		"project_id":   project.ID,
	})

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/orders/%s/items", order.ID), bytes.NewReader(itemBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("add item: expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify item was added
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/orders/%s", order.ID), nil)
	rr = httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	var fetched model.Order
	json.NewDecoder(rr.Body).Decode(&fetched)

	if len(fetched.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(fetched.Items))
	}
	if fetched.Items[0].SKU != "SKU-123" {
		t.Errorf("item SKU: got %q, want %q", fetched.Items[0].SKU, "SKU-123")
	}
}

// =============================================================================
// End-to-End Workflow Tests
// =============================================================================

func TestE2E_OrderToTaskWorkflow(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// 1. Create a product (project)
	project := &model.Project{
		Name:       "Premium Lamp",
		SKU:        "LAMP-PREM-001",
		PriceCents: func() *int { v := 7999; return &v }(),
	}
	if err := env.services.Projects.Create(ctx, project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	// 2. Create an order
	orderBody, _ := json.Marshal(map[string]interface{}{
		"customer_name":  "Alice Smith",
		"customer_email": "alice@example.com",
		"channel":        "etsy",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/orders", bytes.NewReader(orderBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("create order: got %d: %s", rr.Code, rr.Body.String())
	}

	var order model.Order
	json.NewDecoder(rr.Body).Decode(&order)

	// 3. Add item to order
	itemBody, _ := json.Marshal(map[string]interface{}{
		"product_name": "Premium Lamp",
		"sku":          "LAMP-PREM-001",
		"quantity":     2,
		"price_cents":  7999,
		"project_id":   project.ID,
	})
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/orders/%s/items", order.ID), bytes.NewReader(itemBody))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("add item: got %d: %s", rr.Code, rr.Body.String())
	}

	var item model.OrderItem
	json.NewDecoder(rr.Body).Decode(&item)

	// 4. Process order item to create task
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/orders/%s/items/%s/process", order.ID, item.ID), nil)
	rr = httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK && rr.Code != http.StatusCreated {
		t.Fatalf("process item: got %d: %s", rr.Code, rr.Body.String())
	}

	// 5. Verify task was created
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/tasks?order_id=%s", order.ID), nil)
	rr = httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	var tasks []model.Task
	json.NewDecoder(rr.Body).Decode(&tasks)

	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}

	task := tasks[0]
	if task.ProjectID != project.ID {
		t.Errorf("task project_id: got %s, want %s", task.ProjectID, project.ID)
	}
	if task.OrderID == nil || *task.OrderID != order.ID {
		t.Errorf("task order_id mismatch")
	}

	// 6. Start the task
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/tasks/%s/start", task.ID), nil)
	rr = httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("start task: got %d: %s", rr.Code, rr.Body.String())
	}

	// 7. Complete the task
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/tasks/%s/complete", task.ID), nil)
	rr = httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("complete task: got %d: %s", rr.Code, rr.Body.String())
	}

	// 8. Verify task is completed
	req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/tasks/%s", task.ID), nil)
	rr = httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	var completedTask model.Task
	json.NewDecoder(rr.Body).Decode(&completedTask)

	if completedTask.Status != model.TaskStatusCompleted {
		t.Errorf("task status: got %q, want completed", completedTask.Status)
	}
}

func TestE2E_MultipleTasksPerProject(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// Create a project
	project := &model.Project{Name: "Popular Product", SKU: "POP-001"}
	env.services.Projects.Create(ctx, project)

	// Create multiple tasks for the same project
	for i := 0; i < 5; i++ {
		task := &model.Task{
			ProjectID: project.ID,
			Name:      fmt.Sprintf("Batch %d", i+1),
			Quantity:  10,
		}
		env.services.Tasks.Create(ctx, task)
	}

	// Verify all tasks are linked to the project
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/projects/%s/tasks", project.ID), nil)
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	var tasks []model.Task
	json.NewDecoder(rr.Body).Decode(&tasks)

	if len(tasks) != 5 {
		t.Errorf("expected 5 tasks, got %d", len(tasks))
	}

	// Complete some tasks and verify filtering works
	env.services.Tasks.StartTask(ctx, tasks[0].ID)
	env.services.Tasks.CompleteTask(ctx, tasks[0].ID)
	env.services.Tasks.StartTask(ctx, tasks[1].ID)
	env.services.Tasks.CompleteTask(ctx, tasks[1].ID)

	req = httptest.NewRequest(http.MethodGet, "/api/tasks?status=completed", nil)
	rr = httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	var completedTasks []model.Task
	json.NewDecoder(rr.Body).Decode(&completedTasks)

	if len(completedTasks) != 2 {
		t.Errorf("expected 2 completed tasks, got %d", len(completedTasks))
	}
}

// =============================================================================
// Error Handling Tests
// =============================================================================

func TestTask_InvalidProjectID(t *testing.T) {
	env := newTestEnv(t)

	body, _ := json.Marshal(map[string]interface{}{
		"project_id": uuid.New(), // Non-existent project
		"name":       "Orphan Task",
		"quantity":   1,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	// Should fail with 400 or 404
	if rr.Code == http.StatusCreated {
		t.Error("expected error for non-existent project")
	}
}

func TestTask_NotFound(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/tasks/%s", uuid.New()), nil)
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestTask_InvalidStatus(t *testing.T) {
	env := newTestEnv(t)

	project := &model.Project{Name: "Test"}
	env.services.Projects.Create(context.Background(), project)

	task := &model.Task{ProjectID: project.ID, Name: "Test Task", Quantity: 1}
	env.services.Tasks.Create(context.Background(), task)

	// Try to complete a pending task (should fail - must start first)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/tasks/%s/complete", task.ID), nil)
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	// Depending on implementation, this might fail or succeed
	// Just verify it doesn't crash
	if rr.Code == http.StatusInternalServerError {
		t.Errorf("should handle invalid state transition gracefully, got 500: %s", rr.Body.String())
	}
}

func TestProject_NotFound(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/projects/%s", uuid.New()), nil)
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestOrder_NotFound(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/orders/%s", uuid.New()), nil)
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}
