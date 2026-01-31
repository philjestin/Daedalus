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

// OrderService handles unified order business logic.
type OrderService struct {
	orderRepo    *repository.OrderRepository
	projectRepo  *repository.ProjectRepository
	printJobRepo *repository.PrintJobRepository
	taskRepo     *repository.TaskRepository
	templateSvc  *TemplateService
	hub          *realtime.Hub
}

// NewOrderService creates a new OrderService.
func NewOrderService(
	orderRepo *repository.OrderRepository,
	projectRepo *repository.ProjectRepository,
	printJobRepo *repository.PrintJobRepository,
	templateSvc *TemplateService,
	hub *realtime.Hub,
) *OrderService {
	return &OrderService{
		orderRepo:    orderRepo,
		projectRepo:  projectRepo,
		printJobRepo: printJobRepo,
		templateSvc:  templateSvc,
		hub:          hub,
	}
}

// SetTaskRepo sets the task repository (called after initialization to avoid circular dependency).
func (s *OrderService) SetTaskRepo(taskRepo *repository.TaskRepository) {
	s.taskRepo = taskRepo
}

// Create creates a new order.
func (s *OrderService) Create(ctx context.Context, order *model.Order) error {
	if order.CustomerName == "" {
		return fmt.Errorf("customer name is required")
	}
	if order.Source == "" {
		order.Source = model.OrderSourceManual
	}

	if err := s.orderRepo.Create(ctx, order); err != nil {
		return err
	}

	// Add creation event
	event := &model.OrderEvent{
		OrderID:   order.ID,
		EventType: "created",
		Message:   fmt.Sprintf("Order created from %s", order.Source),
	}
	s.orderRepo.AddEvent(ctx, event)

	s.broadcastUpdate("order_created", order)
	slog.Info("order created", "id", order.ID, "source", order.Source, "customer", order.CustomerName)
	return nil
}

// GetByID retrieves an order by ID with items and tasks.
func (s *OrderService) GetByID(ctx context.Context, id uuid.UUID) (*model.Order, error) {
	order, err := s.orderRepo.GetByID(ctx, id)
	if err != nil || order == nil {
		return order, err
	}

	// Load items
	items, err := s.orderRepo.GetItems(ctx, id)
	if err != nil {
		return nil, err
	}
	order.Items = items

	// Load tasks
	if s.taskRepo != nil {
		tasks, err := s.orderRepo.GetTasksByOrderID(ctx, id)
		if err != nil {
			slog.Warn("failed to load order tasks", "order_id", id, "error", err)
		}
		order.Tasks = tasks
	}

	// Load events
	events, err := s.orderRepo.GetEvents(ctx, id)
	if err != nil {
		return nil, err
	}
	order.Events = events

	return order, nil
}

// GetBySourceID retrieves an order by source and external ID.
func (s *OrderService) GetBySourceID(ctx context.Context, source model.OrderSource, sourceID string) (*model.Order, error) {
	return s.orderRepo.GetBySourceID(ctx, source, sourceID)
}

// List retrieves orders with optional filtering.
func (s *OrderService) List(ctx context.Context, filters model.OrderFilters) ([]model.Order, error) {
	orders, err := s.orderRepo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	// Load items for each order
	for i := range orders {
		items, err := s.orderRepo.GetItems(ctx, orders[i].ID)
		if err != nil {
			slog.Warn("failed to load order items", "order_id", orders[i].ID, "error", err)
			continue
		}
		orders[i].Items = items
	}

	return orders, nil
}

// Update updates an order.
func (s *OrderService) Update(ctx context.Context, order *model.Order) error {
	if err := s.orderRepo.Update(ctx, order); err != nil {
		return err
	}
	s.broadcastUpdate("order_updated", order)
	return nil
}

// UpdateStatus updates the status of an order.
func (s *OrderService) UpdateStatus(ctx context.Context, id uuid.UUID, status model.OrderStatus) error {
	order, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if order == nil {
		return fmt.Errorf("order not found")
	}

	oldStatus := order.Status
	if err := s.orderRepo.UpdateStatus(ctx, id, status); err != nil {
		return err
	}

	// Add status change event
	event := &model.OrderEvent{
		OrderID:   id,
		EventType: "status_changed",
		Message:   fmt.Sprintf("Status changed from %s to %s", oldStatus, status),
	}
	s.orderRepo.AddEvent(ctx, event)

	// Reload and broadcast
	order, _ = s.orderRepo.GetByID(ctx, id)
	s.broadcastUpdate("order_status_updated", order)

	slog.Info("order status updated", "id", id, "old_status", oldStatus, "new_status", status)
	return nil
}

// Delete removes an order.
func (s *OrderService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.orderRepo.Delete(ctx, id); err != nil {
		return err
	}
	s.broadcastUpdate("order_deleted", map[string]interface{}{"id": id})
	return nil
}

// AddItem adds an item to an order.
func (s *OrderService) AddItem(ctx context.Context, orderID uuid.UUID, item *model.OrderItem) error {
	item.OrderID = orderID
	if err := s.orderRepo.AddItem(ctx, item); err != nil {
		return err
	}

	event := &model.OrderEvent{
		OrderID:   orderID,
		EventType: "item_added",
		Message:   fmt.Sprintf("Added item: %s (qty: %d)", item.SKU, item.Quantity),
	}
	s.orderRepo.AddEvent(ctx, event)

	return nil
}

// RemoveItem removes an item from an order.
func (s *OrderService) RemoveItem(ctx context.Context, orderID uuid.UUID, itemID uuid.UUID) error {
	if err := s.orderRepo.DeleteItem(ctx, itemID); err != nil {
		return err
	}

	event := &model.OrderEvent{
		OrderID:   orderID,
		EventType: "item_removed",
		Message:   "Item removed from order",
	}
	s.orderRepo.AddEvent(ctx, event)

	return nil
}

// ProcessItem creates a task from a project (product catalog entry) for an order item.
func (s *OrderService) ProcessItem(ctx context.Context, orderID, itemID uuid.UUID) (*model.Task, error) {
	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if order == nil {
		return nil, fmt.Errorf("order not found")
	}

	item, err := s.orderRepo.GetItem(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, fmt.Errorf("item not found")
	}

	// Get the project (product catalog entry) - check ProjectID first, then fall back to template
	var projectID uuid.UUID
	if item.ProjectID != nil {
		projectID = *item.ProjectID
	} else if item.TemplateID != nil {
		// Legacy: look up projects by template
		projects, err := s.projectRepo.ListByTemplateID(ctx, *item.TemplateID)
		if err != nil {
			return nil, fmt.Errorf("failed to find project for template: %w", err)
		}
		if len(projects) == 0 {
			return nil, fmt.Errorf("no project found for template %s", item.TemplateID)
		}
		projectID = projects[0].ID
	} else {
		return nil, fmt.Errorf("item has no project or template assigned")
	}

	project, err := s.projectRepo.GetByID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, fmt.Errorf("project not found")
	}

	// Create task from project
	task := &model.Task{
		ProjectID:   projectID,
		OrderID:     &orderID,
		OrderItemID: &itemID,
		Name:        fmt.Sprintf("%s (Order %s)", project.Name, order.SourceOrderID),
		Status:      model.TaskStatusPending,
		Quantity:    item.Quantity,
		Notes:       order.Notes,
	}

	if err := s.taskRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("creating task: %w", err)
	}

	// Add event
	event := &model.OrderEvent{
		OrderID:   orderID,
		EventType: "task_created",
		Message:   fmt.Sprintf("Created task %s from item %s", task.Name, item.SKU),
	}
	s.orderRepo.AddEvent(ctx, event)

	// Update order status to in_progress if pending
	if order.Status == model.OrderStatusPending {
		s.UpdateStatus(ctx, orderID, model.OrderStatusInProgress)
	}

	slog.Info("processed order item", "order_id", orderID, "item_id", itemID, "task_id", task.ID)
	return task, nil
}

// GetProgress calculates the completion progress of an order.
func (s *OrderService) GetProgress(ctx context.Context, id uuid.UUID) (*model.OrderProgress, error) {
	order, err := s.GetByID(ctx, id)
	if err != nil || order == nil {
		return nil, err
	}

	progress := &model.OrderProgress{
		OrderID:    id,
		TotalItems: len(order.Items),
	}

	// Count completed items based on linked tasks
	var completedTasks int
	for _, task := range order.Tasks {
		if task.Status == model.TaskStatusCompleted {
			completedTasks++
		}
		// Count jobs for each task
		jobs, err := s.printJobRepo.ListByTask(ctx, task.ID)
		if err != nil {
			continue
		}
		for _, job := range jobs {
			progress.TotalJobs++
			if job.Status == model.PrintJobStatusCompleted {
				progress.CompletedJobs++
			}
		}
	}
	progress.CompletedItems = completedTasks

	// Calculate percentage
	if progress.TotalItems > 0 {
		progress.ProgressPercent = float64(progress.CompletedItems) / float64(progress.TotalItems) * 100
	}

	return progress, nil
}

// GetCounts returns counts of orders by status.
func (s *OrderService) GetCounts(ctx context.Context) (map[model.OrderStatus]int, error) {
	return s.orderRepo.CountByStatus(ctx)
}

// CreateFromExternalOrder creates a unified order from an external source (Etsy, Squarespace, Shopify).
func (s *OrderService) CreateFromExternalOrder(ctx context.Context, source model.OrderSource, sourceOrderID string, customerName, customerEmail string, items []model.OrderItem) (*model.Order, error) {
	// Check if order already exists
	existing, err := s.orderRepo.GetBySourceID(ctx, source, sourceOrderID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil // Already imported
	}

	order := &model.Order{
		Source:        source,
		SourceOrderID: sourceOrderID,
		CustomerName:  customerName,
		CustomerEmail: customerEmail,
		Status:        model.OrderStatusPending,
	}

	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, err
	}

	// Add items
	for i := range items {
		items[i].OrderID = order.ID
		if err := s.orderRepo.AddItem(ctx, &items[i]); err != nil {
			slog.Warn("failed to add order item", "order_id", order.ID, "sku", items[i].SKU, "error", err)
		}
	}

	// Add creation event
	event := &model.OrderEvent{
		OrderID:   order.ID,
		EventType: "synced",
		Message:   fmt.Sprintf("Imported from %s (ID: %s)", source, sourceOrderID),
	}
	s.orderRepo.AddEvent(ctx, event)

	slog.Info("created order from external source", "id", order.ID, "source", source, "source_order_id", sourceOrderID)
	return order, nil
}

// broadcastUpdate sends an order update via WebSocket.
func (s *OrderService) broadcastUpdate(eventType string, data interface{}) {
	if s.hub != nil {
		s.hub.Broadcast(model.BroadcastEvent{
			Type: eventType,
			Data: data,
		})
	}
}

// CheckOrderCompletion checks if all tasks for an order are complete and updates status.
func (s *OrderService) CheckOrderCompletion(ctx context.Context, orderID uuid.UUID) error {
	order, err := s.GetByID(ctx, orderID)
	if err != nil || order == nil {
		return err
	}

	// If all tasks are completed, mark order as completed
	if len(order.Tasks) > 0 {
		allCompleted := true
		for _, task := range order.Tasks {
			if task.Status != model.TaskStatusCompleted {
				allCompleted = false
				break
			}
		}
		if allCompleted {
			return s.UpdateStatus(ctx, orderID, model.OrderStatusCompleted)
		}
	}

	return nil
}

// MarkShipped marks an order as shipped.
func (s *OrderService) MarkShipped(ctx context.Context, id uuid.UUID, trackingNumber string) error {
	order, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if order == nil {
		return fmt.Errorf("order not found")
	}

	now := time.Now()
	order.ShippedAt = &now
	order.Status = model.OrderStatusShipped

	if err := s.orderRepo.Update(ctx, order); err != nil {
		return err
	}

	event := &model.OrderEvent{
		OrderID:   id,
		EventType: "shipped",
		Message:   fmt.Sprintf("Order shipped%s", func() string {
			if trackingNumber != "" {
				return fmt.Sprintf(" (tracking: %s)", trackingNumber)
			}
			return ""
		}()),
	}
	s.orderRepo.AddEvent(ctx, event)

	s.broadcastUpdate("order_shipped", order)
	return nil
}
