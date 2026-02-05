package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/hyperion/printfarm/internal/model"
	"github.com/hyperion/printfarm/internal/realtime"
	"github.com/hyperion/printfarm/internal/repository"
)

// TaskService handles task business logic.
type TaskService struct {
	taskRepo      *repository.TaskRepository
	projectRepo   *repository.ProjectRepository
	printJobRepo  *repository.PrintJobRepository
	partRepo      *repository.PartRepository
	checklistRepo *repository.TaskChecklistRepository
	designRepo    *repository.DesignRepository
	hub           *realtime.Hub
}

// NewTaskService creates a new TaskService.
func NewTaskService(
	taskRepo *repository.TaskRepository,
	projectRepo *repository.ProjectRepository,
	printJobRepo *repository.PrintJobRepository,
	partRepo *repository.PartRepository,
	checklistRepo *repository.TaskChecklistRepository,
	designRepo *repository.DesignRepository,
	hub *realtime.Hub,
) *TaskService {
	return &TaskService{
		taskRepo:      taskRepo,
		projectRepo:   projectRepo,
		printJobRepo:  printJobRepo,
		partRepo:      partRepo,
		checklistRepo: checklistRepo,
		designRepo:    designRepo,
		hub:           hub,
	}
}

// Create creates a new task.
func (s *TaskService) Create(ctx context.Context, task *model.Task) error {
	if task.ProjectID == uuid.Nil {
		return fmt.Errorf("project_id is required")
	}
	if task.Name == "" {
		return fmt.Errorf("task name is required")
	}

	if err := s.taskRepo.Create(ctx, task); err != nil {
		return err
	}

	// Auto-generate checklist items from project parts
	s.generateChecklist(ctx, task)

	s.broadcastUpdate("task_created", task)
	slog.Info("task created", "id", task.ID, "project_id", task.ProjectID, "name", task.Name)
	return nil
}

// CreateFromProject creates a task from a project (product catalog entry).
func (s *TaskService) CreateFromProject(ctx context.Context, projectID uuid.UUID, orderID, orderItemID *uuid.UUID, quantity int) (*model.Task, error) {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, fmt.Errorf("project not found")
	}

	if quantity <= 0 {
		quantity = 1
	}

	task := &model.Task{
		ProjectID:   projectID,
		OrderID:     orderID,
		OrderItemID: orderItemID,
		Name:        project.Name,
		Status:      model.TaskStatusPending,
		Quantity:    quantity,
	}

	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, err
	}

	// Auto-generate checklist items from project parts
	s.generateChecklist(ctx, task)

	s.broadcastUpdate("task_created", task)
	slog.Info("task created from project", "id", task.ID, "project_id", projectID, "quantity", quantity)
	return task, nil
}

// GetByID retrieves a task by ID with optional relations.
func (s *TaskService) GetByID(ctx context.Context, id uuid.UUID) (*model.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil || task == nil {
		return task, err
	}

	// Load project
	project, err := s.projectRepo.GetByID(ctx, task.ProjectID)
	if err != nil {
		slog.Warn("failed to load task project", "task_id", id, "project_id", task.ProjectID, "error", err)
	}
	task.Project = project

	// Load jobs
	jobs, err := s.printJobRepo.ListByTask(ctx, id)
	if err != nil {
		slog.Warn("failed to load task jobs", "task_id", id, "error", err)
	}
	task.Jobs = jobs

	// Load checklist items
	if s.checklistRepo != nil {
		items, err := s.checklistRepo.ListByTask(ctx, id)
		if err != nil {
			slog.Warn("failed to load task checklist", "task_id", id, "error", err)
		}
		task.ChecklistItems = items
	}

	// Calculate progress
	task.Progress = s.calculateProgress(task)

	return task, nil
}

// List retrieves tasks with optional filters.
func (s *TaskService) List(ctx context.Context, filters model.TaskFilters) ([]model.Task, error) {
	tasks, err := s.taskRepo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	// Load progress for each task
	for i := range tasks {
		jobs, err := s.printJobRepo.ListByTask(ctx, tasks[i].ID)
		if err != nil {
			continue
		}
		tasks[i].Jobs = jobs
		if s.checklistRepo != nil {
			items, err := s.checklistRepo.ListByTask(ctx, tasks[i].ID)
			if err == nil {
				tasks[i].ChecklistItems = items
			}
		}
		tasks[i].Progress = s.calculateProgress(&tasks[i])
	}

	return tasks, nil
}

// ListByProject retrieves all tasks for a project.
func (s *TaskService) ListByProject(ctx context.Context, projectID uuid.UUID) ([]model.Task, error) {
	return s.List(ctx, model.TaskFilters{ProjectID: &projectID})
}

// ListByOrder retrieves all tasks for an order.
func (s *TaskService) ListByOrder(ctx context.Context, orderID uuid.UUID) ([]model.Task, error) {
	return s.List(ctx, model.TaskFilters{OrderID: &orderID})
}

// Update updates a task.
func (s *TaskService) Update(ctx context.Context, task *model.Task) error {
	if err := s.taskRepo.Update(ctx, task); err != nil {
		return err
	}
	s.broadcastUpdate("task_updated", task)
	return nil
}

// UpdateStatus updates the status of a task.
func (s *TaskService) UpdateStatus(ctx context.Context, id uuid.UUID, status model.TaskStatus) error {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if task == nil {
		return fmt.Errorf("task not found")
	}

	oldStatus := task.Status
	if err := s.taskRepo.UpdateStatus(ctx, id, status); err != nil {
		return err
	}

	// Reload for broadcasting
	task, _ = s.taskRepo.GetByID(ctx, id)
	s.broadcastUpdate("task_status_updated", task)

	slog.Info("task status updated", "id", id, "old_status", oldStatus, "new_status", status)
	return nil
}

// Delete removes a task.
func (s *TaskService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.taskRepo.Delete(ctx, id); err != nil {
		return err
	}
	s.broadcastUpdate("task_deleted", map[string]interface{}{"id": id})
	return nil
}

// GetProgress calculates the completion progress of a task based on its jobs.
func (s *TaskService) GetProgress(ctx context.Context, id uuid.UUID) (float64, error) {
	task, err := s.GetByID(ctx, id)
	if err != nil || task == nil {
		return 0, err
	}
	return task.Progress, nil
}

// AddJob adds a print job to a task.
func (s *TaskService) AddJob(ctx context.Context, taskID uuid.UUID, job *model.PrintJob) error {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return err
	}
	if task == nil {
		return fmt.Errorf("task not found")
	}

	job.TaskID = &taskID
	job.ProjectID = &task.ProjectID

	// The job will be created by PrintJobService, we just update task tracking
	s.broadcastUpdate("task_job_added", map[string]interface{}{
		"task_id": taskID,
		"job_id":  job.ID,
	})

	return nil
}

// CheckTaskCompletion checks if all jobs for a task are complete and updates status.
func (s *TaskService) CheckTaskCompletion(ctx context.Context, taskID uuid.UUID) error {
	task, err := s.GetByID(ctx, taskID)
	if err != nil || task == nil {
		return err
	}

	if task.Progress >= 100.0 && task.Status != model.TaskStatusCompleted {
		return s.UpdateStatus(ctx, taskID, model.TaskStatusCompleted)
	}

	if task.Progress > 0 && task.Status == model.TaskStatusPending {
		return s.UpdateStatus(ctx, taskID, model.TaskStatusInProgress)
	}

	return nil
}

// StartTask marks a task as in_progress.
func (s *TaskService) StartTask(ctx context.Context, id uuid.UUID) error {
	task, err := s.taskRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if task == nil {
		return fmt.Errorf("task not found")
	}

	if task.Status == model.TaskStatusPending {
		now := time.Now()
		task.Status = model.TaskStatusInProgress
		task.StartedAt = &now
		return s.taskRepo.Update(ctx, task)
	}

	return nil
}

// CompleteTask marks a task as completed.
func (s *TaskService) CompleteTask(ctx context.Context, id uuid.UUID) error {
	return s.UpdateStatus(ctx, id, model.TaskStatusCompleted)
}

// CancelTask marks a task as cancelled.
func (s *TaskService) CancelTask(ctx context.Context, id uuid.UUID) error {
	return s.UpdateStatus(ctx, id, model.TaskStatusCancelled)
}

// RegenerateChecklist removes existing checklist items and creates new ones from project parts.
func (s *TaskService) RegenerateChecklist(ctx context.Context, taskID uuid.UUID) ([]model.TaskChecklistItem, error) {
	if s.partRepo == nil || s.checklistRepo == nil {
		return nil, fmt.Errorf("checklist not available")
	}

	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task == nil {
		return nil, fmt.Errorf("task not found")
	}

	// Delete existing checklist items
	if err := s.checklistRepo.DeleteByTask(ctx, taskID); err != nil {
		return nil, err
	}

	// Generate new ones
	s.generateChecklist(ctx, task)

	// Return the new list
	return s.checklistRepo.ListByTask(ctx, taskID)
}

// generateChecklist creates checklist items from project parts for a task.
func (s *TaskService) generateChecklist(ctx context.Context, task *model.Task) {
	if s.partRepo == nil || s.checklistRepo == nil {
		return
	}
	parts, err := s.partRepo.ListByProject(ctx, task.ProjectID)
	if err != nil {
		slog.Warn("failed to load project parts for checklist", "project_id", task.ProjectID, "error", err)
		return
	}
	if len(parts) == 0 {
		return
	}
	var items []model.TaskChecklistItem
	for i, part := range parts {
		partID := part.ID
		items = append(items, model.TaskChecklistItem{
			TaskID:    task.ID,
			Name:      part.Name,
			PartID:    &partID,
			SortOrder: i,
		})
	}
	// Add "Assembly" as the final checklist item
	items = append(items, model.TaskChecklistItem{
		TaskID:    task.ID,
		Name:      "Assembly",
		SortOrder: len(parts),
	})
	if err := s.checklistRepo.CreateBatch(ctx, items); err != nil {
		slog.Warn("failed to create checklist items", "task_id", task.ID, "error", err)
	}
}

// ToggleChecklistItem toggles the completed state of a checklist item.
func (s *TaskService) ToggleChecklistItem(ctx context.Context, itemID uuid.UUID, completed bool) error {
	if s.checklistRepo == nil {
		return fmt.Errorf("checklist repository not available")
	}
	return s.checklistRepo.UpdateCompleted(ctx, itemID, completed)
}

// GetChecklist returns the checklist items for a task.
func (s *TaskService) GetChecklist(ctx context.Context, taskID uuid.UUID) ([]model.TaskChecklistItem, error) {
	if s.checklistRepo == nil {
		return nil, nil
	}
	return s.checklistRepo.ListByTask(ctx, taskID)
}

// calculateProgress computes the progress percentage based on job and checklist completion.
func (s *TaskService) calculateProgress(task *model.Task) float64 {
	hasJobs := len(task.Jobs) > 0
	hasChecklist := len(task.ChecklistItems) > 0

	if !hasJobs && !hasChecklist {
		return 0
	}

	var jobProgress float64
	if hasJobs {
		var completed int
		for _, job := range task.Jobs {
			if job.Status == model.PrintJobStatusCompleted {
				completed++
			}
		}
		jobProgress = float64(completed) / float64(len(task.Jobs)) * 100
	}

	var checklistProgress float64
	if hasChecklist {
		var completed int
		for _, item := range task.ChecklistItems {
			if item.Completed {
				completed++
			}
		}
		checklistProgress = float64(completed) / float64(len(task.ChecklistItems)) * 100
	}

	if hasJobs && hasChecklist {
		return (jobProgress + checklistProgress) / 2
	}
	if hasChecklist {
		return checklistProgress
	}
	return jobProgress
}

// PrintFromChecklist creates a print job for a checklist item's linked part.
func (s *TaskService) PrintFromChecklist(ctx context.Context, taskID, checklistItemID uuid.UUID) (*model.PrintJob, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil {
		return nil, fmt.Errorf("task not found")
	}
	if task.Status == model.TaskStatusCompleted || task.Status == model.TaskStatusCancelled {
		return nil, fmt.Errorf("cannot print from a %s task", task.Status)
	}

	// Find the checklist item
	items, err := s.checklistRepo.ListByTask(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to load checklist: %w", err)
	}
	var item *model.TaskChecklistItem
	for i := range items {
		if items[i].ID == checklistItemID {
			item = &items[i]
			break
		}
	}
	if item == nil {
		return nil, fmt.Errorf("checklist item not found")
	}
	if item.PartID == nil {
		return nil, fmt.Errorf("checklist item has no linked part")
	}

	// Get the latest design for this part
	designs, err := s.designRepo.ListByPart(ctx, *item.PartID)
	if err != nil {
		return nil, fmt.Errorf("failed to load designs: %w", err)
	}
	if len(designs) == 0 {
		return nil, fmt.Errorf("no designs found for part")
	}
	latestDesign := designs[0] // ListByPart orders by version DESC

	// Create a queued print job linked to this task
	job := &model.PrintJob{
		DesignID:  latestDesign.ID,
		TaskID:    &taskID,
		ProjectID: &task.ProjectID,
	}
	if err := s.printJobRepo.Create(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to create print job: %w", err)
	}

	// Auto-start task if pending
	if task.Status == model.TaskStatusPending {
		_ = s.StartTask(ctx, taskID)
	}

	s.broadcastUpdate("task_job_added", map[string]interface{}{
		"task_id": taskID,
		"job_id":  job.ID,
	})

	slog.Info("print job created from checklist", "task_id", taskID, "item_id", checklistItemID, "job_id", job.ID, "design_id", latestDesign.ID)
	return job, nil
}

// HandleJobCompleted auto-marks the matching checklist item as done when a print job succeeds.
func (s *TaskService) HandleJobCompleted(ctx context.Context, job *model.PrintJob) {
	if job.TaskID == nil {
		return
	}
	taskID := *job.TaskID

	// Get the design to find the part ID
	design, err := s.designRepo.GetByID(ctx, job.DesignID)
	if err != nil || design == nil {
		slog.Warn("HandleJobCompleted: failed to load design", "design_id", job.DesignID, "error", err)
		return
	}

	// Find uncompleted checklist item matching this part
	items, err := s.checklistRepo.ListByTask(ctx, taskID)
	if err != nil {
		slog.Warn("HandleJobCompleted: failed to load checklist", "task_id", taskID, "error", err)
		return
	}
	for _, item := range items {
		if item.PartID != nil && *item.PartID == design.PartID && !item.Completed {
			if err := s.checklistRepo.UpdateCompleted(ctx, item.ID, true); err != nil {
				slog.Warn("HandleJobCompleted: failed to mark checklist item", "item_id", item.ID, "error", err)
				return
			}
			slog.Info("checklist item auto-completed", "task_id", taskID, "item_id", item.ID, "part_id", design.PartID)
			break
		}
	}

	// Check if task should auto-complete
	if err := s.CheckTaskCompletion(ctx, taskID); err != nil {
		slog.Warn("HandleJobCompleted: failed to check task completion", "task_id", taskID, "error", err)
	}
}

// broadcastUpdate sends a task update via WebSocket.
func (s *TaskService) broadcastUpdate(eventType string, data interface{}) {
	if s.hub != nil {
		s.hub.Broadcast(model.BroadcastEvent{
			Type: eventType,
			Data: data,
		})
	}
}
