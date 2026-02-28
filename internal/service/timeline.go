package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/philjestin/daedalus/internal/model"
	"github.com/philjestin/daedalus/internal/repository"
)

// TimelineService handles timeline/Gantt view business logic.
type TimelineService struct {
	orderRepo    *repository.OrderRepository
	taskRepo     *repository.TaskRepository
	projectRepo  *repository.ProjectRepository
	printJobRepo *repository.PrintJobRepository
}

// NewTimelineService creates a new TimelineService.
func NewTimelineService(
	orderRepo *repository.OrderRepository,
	taskRepo *repository.TaskRepository,
	projectRepo *repository.ProjectRepository,
	printJobRepo *repository.PrintJobRepository,
) *TimelineService {
	return &TimelineService{
		orderRepo:    orderRepo,
		taskRepo:     taskRepo,
		projectRepo:  projectRepo,
		printJobRepo: printJobRepo,
	}
}

// GetTimeline retrieves timeline items for a date range.
func (s *TimelineService) GetTimeline(ctx context.Context, startDate, endDate *time.Time) ([]model.TimelineItem, error) {
	var items []model.TimelineItem

	// Get orders in date range
	filters := model.OrderFilters{
		StartDate: startDate,
		EndDate:   endDate,
	}
	orders, err := s.orderRepo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	for _, order := range orders {
		item := s.orderToTimelineItem(order)

		// Get tasks for this order
		tasks, err := s.taskRepo.ListByOrder(ctx, order.ID)
		if err != nil {
			continue
		}

		for _, task := range tasks {
			taskItem := s.taskToTimelineItem(task, &order.ID)

			// Get jobs for this task
			jobs, err := s.printJobRepo.ListByTask(ctx, task.ID)
			if err != nil {
				continue
			}

			for _, job := range jobs {
				jobItem := s.jobToTimelineItem(job, &task.ID)
				taskItem.Children = append(taskItem.Children, jobItem)
			}

			item.Children = append(item.Children, taskItem)
		}

		items = append(items, item)
	}

	// Also get standalone tasks (not linked to orders)
	allTasks, err := s.taskRepo.List(ctx, model.TaskFilters{})
	if err != nil {
		return items, err // Return what we have
	}

	for _, task := range allTasks {
		if task.OrderID != nil {
			continue // Already included under orders
		}

		item := s.taskToTimelineItem(task, nil)

		// Get jobs for this task
		jobs, err := s.printJobRepo.ListByTask(ctx, task.ID)
		if err != nil {
			continue
		}

		for _, job := range jobs {
			jobItem := s.jobToTimelineItem(job, &task.ID)
			item.Children = append(item.Children, jobItem)
		}

		items = append(items, item)
	}

	return items, nil
}

// orderToTimelineItem converts an order to a timeline item.
func (s *TimelineService) orderToTimelineItem(order model.Order) model.TimelineItem {
	progress := 0.0
	if order.Status == model.OrderStatusCompleted || order.Status == model.OrderStatusShipped {
		progress = 100
	} else if order.Status == model.OrderStatusInProgress {
		progress = 50 // Will be refined based on jobs
	}

	var endDate *time.Time
	if order.CompletedAt != nil {
		endDate = order.CompletedAt
	} else if order.ShippedAt != nil {
		endDate = order.ShippedAt
	}

	return model.TimelineItem{
		ID:        order.ID,
		Type:      "order",
		Name:      order.CustomerName,
		Status:    string(order.Status),
		StartDate: &order.CreatedAt,
		DueDate:   order.DueDate,
		EndDate:   endDate,
		Progress:  progress,
	}
}

// taskToTimelineItem converts a task to a timeline item.
func (s *TimelineService) taskToTimelineItem(task model.Task, parentID *uuid.UUID) model.TimelineItem {
	var startDate *time.Time
	if task.StartedAt != nil {
		startDate = task.StartedAt
	} else {
		startDate = &task.CreatedAt
	}

	var endDate *time.Time
	if task.CompletedAt != nil {
		endDate = task.CompletedAt
	}

	return model.TimelineItem{
		ID:        task.ID,
		Type:      "task",
		Name:      task.Name,
		Status:    string(task.Status),
		StartDate: startDate,
		EndDate:   endDate,
		Progress:  task.Progress,
		ParentID:  parentID,
	}
}

// projectToTimelineItem converts a project to a timeline item.
func (s *TimelineService) projectToTimelineItem(project model.Project, parentID *uuid.UUID) model.TimelineItem {
	progress := 0.0
	// Calculate progress based on tasks if needed

	return model.TimelineItem{
		ID:        project.ID,
		Type:      "project",
		Name:      project.Name,
		Status:    "active",
		StartDate: &project.CreatedAt,
		DueDate:   project.TargetDate,
		Progress:  progress,
		ParentID:  parentID,
	}
}

// jobToTimelineItem converts a print job to a timeline item.
func (s *TimelineService) jobToTimelineItem(job model.PrintJob, parentID *uuid.UUID) model.TimelineItem {
	var startDate, endDate *time.Time
	if job.StartedAt != nil {
		startDate = job.StartedAt
	} else {
		startDate = &job.CreatedAt
	}
	if job.CompletedAt != nil {
		endDate = job.CompletedAt
	} else if job.EstimatedSeconds != nil && *job.EstimatedSeconds > 0 && job.StartedAt != nil {
		estimated := job.StartedAt.Add(time.Duration(*job.EstimatedSeconds) * time.Second)
		endDate = &estimated
	}

	return model.TimelineItem{
		ID:        job.ID,
		Type:      "job",
		Name:      job.Notes,
		Status:    string(job.Status),
		StartDate: startDate,
		EndDate:   endDate,
		Progress:  float64(job.Progress),
		ParentID:  parentID,
	}
}

// GetOrderTimeline retrieves a detailed timeline for a single order.
func (s *TimelineService) GetOrderTimeline(ctx context.Context, orderID uuid.UUID) (*model.TimelineItem, error) {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil || order == nil {
		return nil, err
	}

	item := s.orderToTimelineItem(*order)

	// Get tasks for this order
	tasks, err := s.taskRepo.ListByOrder(ctx, orderID)
	if err != nil {
		return &item, nil
	}

	totalJobs := 0
	completedJobs := 0

	for _, task := range tasks {
		taskItem := s.taskToTimelineItem(task, &order.ID)

		// Get jobs for this task
		jobs, err := s.printJobRepo.ListByTask(ctx, task.ID)
		if err != nil {
			continue
		}

		taskTotalJobs := 0
		taskCompletedJobs := 0

		for _, job := range jobs {
			jobItem := s.jobToTimelineItem(job, &task.ID)
			taskItem.Children = append(taskItem.Children, jobItem)
			taskTotalJobs++
			totalJobs++
			if job.Status == model.PrintJobStatusCompleted {
				taskCompletedJobs++
				completedJobs++
			}
		}

		// Calculate task progress
		if taskTotalJobs > 0 {
			taskItem.Progress = float64(taskCompletedJobs) / float64(taskTotalJobs) * 100
		}

		item.Children = append(item.Children, taskItem)
	}

	// Calculate order progress
	if totalJobs > 0 {
		item.Progress = float64(completedJobs) / float64(totalJobs) * 100
	}

	return &item, nil
}

// GetTaskTimeline retrieves a detailed timeline for a single task.
func (s *TimelineService) GetTaskTimeline(ctx context.Context, taskID uuid.UUID) (*model.TimelineItem, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil || task == nil {
		return nil, err
	}

	item := s.taskToTimelineItem(*task, task.OrderID)

	// Get jobs for this task
	jobs, err := s.printJobRepo.ListByTask(ctx, taskID)
	if err != nil {
		return &item, nil
	}

	totalJobs := 0
	completedJobs := 0

	for _, job := range jobs {
		jobItem := s.jobToTimelineItem(job, &task.ID)
		item.Children = append(item.Children, jobItem)
		totalJobs++
		if job.Status == model.PrintJobStatusCompleted {
			completedJobs++
		}
	}

	// Calculate progress
	if totalJobs > 0 {
		item.Progress = float64(completedJobs) / float64(totalJobs) * 100
	}

	return &item, nil
}

// GetProjectTimeline retrieves a detailed timeline for a project and its tasks.
func (s *TimelineService) GetProjectTimeline(ctx context.Context, projectID uuid.UUID) (*model.TimelineItem, error) {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil || project == nil {
		return nil, err
	}

	item := s.projectToTimelineItem(*project, nil)

	// Get tasks for this project
	tasks, err := s.taskRepo.ListByProject(ctx, projectID)
	if err != nil {
		return &item, nil
	}

	totalJobs := 0
	completedJobs := 0

	for _, task := range tasks {
		taskItem := s.taskToTimelineItem(task, &project.ID)

		// Get jobs for this task
		jobs, err := s.printJobRepo.ListByTask(ctx, task.ID)
		if err != nil {
			continue
		}

		for _, job := range jobs {
			jobItem := s.jobToTimelineItem(job, &task.ID)
			taskItem.Children = append(taskItem.Children, jobItem)
			totalJobs++
			if job.Status == model.PrintJobStatusCompleted {
				completedJobs++
			}
		}

		item.Children = append(item.Children, taskItem)
	}

	// Calculate progress
	if totalJobs > 0 {
		item.Progress = float64(completedJobs) / float64(totalJobs) * 100
	}

	return &item, nil
}
