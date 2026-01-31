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
	taskRepo     *repository.TaskRepository
	projectRepo  *repository.ProjectRepository
	printJobRepo *repository.PrintJobRepository
	hub          *realtime.Hub
}

// NewTaskService creates a new TaskService.
func NewTaskService(
	taskRepo *repository.TaskRepository,
	projectRepo *repository.ProjectRepository,
	printJobRepo *repository.PrintJobRepository,
	hub *realtime.Hub,
) *TaskService {
	return &TaskService{
		taskRepo:     taskRepo,
		projectRepo:  projectRepo,
		printJobRepo: printJobRepo,
		hub:          hub,
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

// calculateProgress computes the progress percentage based on job completion.
func (s *TaskService) calculateProgress(task *model.Task) float64 {
	if len(task.Jobs) == 0 {
		return 0
	}

	var completed int
	for _, job := range task.Jobs {
		if job.Status == model.PrintJobStatusCompleted {
			completed++
		}
	}

	return float64(completed) / float64(len(task.Jobs)) * 100
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
