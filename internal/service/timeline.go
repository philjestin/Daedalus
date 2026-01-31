package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hyperion/printfarm/internal/model"
	"github.com/hyperion/printfarm/internal/repository"
)

// TimelineService handles timeline/Gantt view business logic.
type TimelineService struct {
	orderRepo    *repository.OrderRepository
	projectRepo  *repository.ProjectRepository
	printJobRepo *repository.PrintJobRepository
}

// NewTimelineService creates a new TimelineService.
func NewTimelineService(
	orderRepo *repository.OrderRepository,
	projectRepo *repository.ProjectRepository,
	printJobRepo *repository.PrintJobRepository,
) *TimelineService {
	return &TimelineService{
		orderRepo:    orderRepo,
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

		// Get projects for this order
		projects, err := s.projectRepo.ListByOrderID(ctx, order.ID)
		if err != nil {
			continue
		}

		for _, project := range projects {
			projectItem := s.projectToTimelineItem(project, &order.ID)

			// Get jobs for this project
			jobs, err := s.printJobRepo.ListByProject(ctx, project.ID)
			if err != nil {
				continue
			}

			for _, job := range jobs {
				jobItem := s.jobToTimelineItem(job, &project.ID)
				projectItem.Children = append(projectItem.Children, jobItem)
			}

			item.Children = append(item.Children, projectItem)
		}

		items = append(items, item)
	}

	// Also get standalone projects (not linked to orders)
	allProjects, err := s.projectRepo.List(ctx)
	if err != nil {
		return items, err // Return what we have
	}

	for _, project := range allProjects {
		if project.OrderID != nil {
			continue // Already included under orders
		}

		item := s.projectToTimelineItem(project, nil)

		// Get jobs for this project
		jobs, err := s.printJobRepo.ListByProject(ctx, project.ID)
		if err != nil {
			continue
		}

		for _, job := range jobs {
			jobItem := s.jobToTimelineItem(job, &project.ID)
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

// projectToTimelineItem converts a project to a timeline item.
func (s *TimelineService) projectToTimelineItem(project model.Project, parentID *uuid.UUID) model.TimelineItem {
	progress := 0.0
	// Calculate progress based on job status could be added here

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

	// Get projects for this order
	projects, err := s.projectRepo.ListByOrderID(ctx, orderID)
	if err != nil {
		return &item, nil
	}

	totalJobs := 0
	completedJobs := 0

	for _, project := range projects {
		projectItem := s.projectToTimelineItem(project, &order.ID)

		// Get jobs for this project
		jobs, err := s.printJobRepo.ListByProject(ctx, project.ID)
		if err != nil {
			continue
		}

		projectTotalJobs := 0
		projectCompletedJobs := 0

		for _, job := range jobs {
			jobItem := s.jobToTimelineItem(job, &project.ID)
			projectItem.Children = append(projectItem.Children, jobItem)
			projectTotalJobs++
			totalJobs++
			if job.Status == model.PrintJobStatusCompleted {
				projectCompletedJobs++
				completedJobs++
			}
		}

		// Calculate project progress
		if projectTotalJobs > 0 {
			projectItem.Progress = float64(projectCompletedJobs) / float64(projectTotalJobs) * 100
		}

		item.Children = append(item.Children, projectItem)
	}

	// Calculate order progress
	if totalJobs > 0 {
		item.Progress = float64(completedJobs) / float64(totalJobs) * 100
	}

	return &item, nil
}

// GetProjectTimeline retrieves a detailed timeline for a single project.
func (s *TimelineService) GetProjectTimeline(ctx context.Context, projectID uuid.UUID) (*model.TimelineItem, error) {
	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil || project == nil {
		return nil, err
	}

	item := s.projectToTimelineItem(*project, project.OrderID)

	// Get jobs for this project
	jobs, err := s.printJobRepo.ListByProject(ctx, projectID)
	if err != nil {
		return &item, nil
	}

	totalJobs := 0
	completedJobs := 0

	for _, job := range jobs {
		jobItem := s.jobToTimelineItem(job, &project.ID)
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
