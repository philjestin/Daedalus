package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/hyperion/printfarm/internal/model"
)

// OrderRepository handles order database operations.
type OrderRepository struct {
	db *sql.DB
}

// Create inserts a new order.
func (r *OrderRepository) Create(ctx context.Context, order *model.Order) error {
	order.ID = uuid.New()
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()
	if order.Status == "" {
		order.Status = model.OrderStatusPending
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO orders (id, source, source_order_id, customer_name, customer_email, status, priority, due_date, notes, created_at, updated_at, completed_at, shipped_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, order.ID, order.Source, order.SourceOrderID, order.CustomerName, order.CustomerEmail, order.Status, order.Priority, order.DueDate, order.Notes, order.CreatedAt, order.UpdatedAt, order.CompletedAt, order.ShippedAt)
	return err
}

// GetByID retrieves an order by ID.
func (r *OrderRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Order, error) {
	var order model.Order
	err := r.db.QueryRowContext(ctx, `
		SELECT id, source, source_order_id, customer_name, customer_email, status, priority, due_date, notes, created_at, updated_at, completed_at, shipped_at
		FROM orders WHERE id = ?
	`, id).Scan(&order.ID, &order.Source, &order.SourceOrderID, &order.CustomerName, &order.CustomerEmail, &order.Status, &order.Priority, &order.DueDate, &order.Notes, &order.CreatedAt, &order.UpdatedAt, &order.CompletedAt, &order.ShippedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &order, err
}

// GetBySourceID retrieves an order by source and source order ID.
func (r *OrderRepository) GetBySourceID(ctx context.Context, source model.OrderSource, sourceID string) (*model.Order, error) {
	var order model.Order
	err := r.db.QueryRowContext(ctx, `
		SELECT id, source, source_order_id, customer_name, customer_email, status, priority, due_date, notes, created_at, updated_at, completed_at, shipped_at
		FROM orders WHERE source = ? AND source_order_id = ?
	`, source, sourceID).Scan(&order.ID, &order.Source, &order.SourceOrderID, &order.CustomerName, &order.CustomerEmail, &order.Status, &order.Priority, &order.DueDate, &order.Notes, &order.CreatedAt, &order.UpdatedAt, &order.CompletedAt, &order.ShippedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &order, err
}

// List retrieves orders with optional filtering.
func (r *OrderRepository) List(ctx context.Context, filters model.OrderFilters) ([]model.Order, error) {
	query := `
		SELECT id, source, source_order_id, customer_name, customer_email, status, priority, due_date, notes, created_at, updated_at, completed_at, shipped_at
		FROM orders WHERE 1=1
	`
	args := []interface{}{}

	if filters.Status != nil {
		query += " AND status = ?"
		args = append(args, *filters.Status)
	}
	if filters.Source != nil {
		query += " AND source = ?"
		args = append(args, *filters.Source)
	}
	if filters.StartDate != nil {
		query += " AND created_at >= ?"
		args = append(args, *filters.StartDate)
	}
	if filters.EndDate != nil {
		query += " AND created_at <= ?"
		args = append(args, *filters.EndDate)
	}

	query += " ORDER BY COALESCE(due_date, '9999-12-31') ASC, priority DESC, created_at DESC"

	if filters.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filters.Limit)
	}
	if filters.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, filters.Offset)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []model.Order
	for rows.Next() {
		var order model.Order
		if err := rows.Scan(&order.ID, &order.Source, &order.SourceOrderID, &order.CustomerName, &order.CustomerEmail, &order.Status, &order.Priority, &order.DueDate, &order.Notes, &order.CreatedAt, &order.UpdatedAt, &order.CompletedAt, &order.ShippedAt); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

// Update updates an order.
func (r *OrderRepository) Update(ctx context.Context, order *model.Order) error {
	order.UpdatedAt = time.Now()
	_, err := r.db.ExecContext(ctx, `
		UPDATE orders SET source = ?, source_order_id = ?, customer_name = ?, customer_email = ?, status = ?, priority = ?, due_date = ?, notes = ?, updated_at = ?, completed_at = ?, shipped_at = ?
		WHERE id = ?
	`, order.Source, order.SourceOrderID, order.CustomerName, order.CustomerEmail, order.Status, order.Priority, order.DueDate, order.Notes, order.UpdatedAt, order.CompletedAt, order.ShippedAt, order.ID)
	return err
}

// UpdateStatus updates the status of an order.
func (r *OrderRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.OrderStatus) error {
	now := time.Now()
	var completedAt, shippedAt *time.Time
	if status == model.OrderStatusCompleted {
		completedAt = &now
	}
	if status == model.OrderStatusShipped {
		shippedAt = &now
	}

	_, err := r.db.ExecContext(ctx, `
		UPDATE orders SET status = ?, updated_at = ?, completed_at = COALESCE(?, completed_at), shipped_at = COALESCE(?, shipped_at)
		WHERE id = ?
	`, status, now, completedAt, shippedAt, id)
	return err
}

// Delete removes an order.
func (r *OrderRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM orders WHERE id = ?`, id)
	return err
}

// AddItem adds an item to an order.
func (r *OrderRepository) AddItem(ctx context.Context, item *model.OrderItem) error {
	item.ID = uuid.New()
	item.CreatedAt = time.Now()
	if item.Quantity == 0 {
		item.Quantity = 1
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO order_items (id, order_id, template_id, sku, quantity, notes, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, item.ID, item.OrderID, item.TemplateID, item.SKU, item.Quantity, item.Notes, item.CreatedAt)
	return err
}

// GetItems retrieves all items for an order.
func (r *OrderRepository) GetItems(ctx context.Context, orderID uuid.UUID) ([]model.OrderItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, order_id, template_id, sku, quantity, notes, created_at
		FROM order_items WHERE order_id = ?
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.OrderItem
	for rows.Next() {
		var item model.OrderItem
		if err := rows.Scan(&item.ID, &item.OrderID, &item.TemplateID, &item.SKU, &item.Quantity, &item.Notes, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetItem retrieves a single order item by ID.
func (r *OrderRepository) GetItem(ctx context.Context, itemID uuid.UUID) (*model.OrderItem, error) {
	var item model.OrderItem
	err := r.db.QueryRowContext(ctx, `
		SELECT id, order_id, template_id, sku, quantity, notes, created_at
		FROM order_items WHERE id = ?
	`, itemID).Scan(&item.ID, &item.OrderID, &item.TemplateID, &item.SKU, &item.Quantity, &item.Notes, &item.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &item, err
}

// DeleteItem removes an item from an order.
func (r *OrderRepository) DeleteItem(ctx context.Context, itemID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM order_items WHERE id = ?`, itemID)
	return err
}

// AddEvent adds an event to the order history.
func (r *OrderRepository) AddEvent(ctx context.Context, event *model.OrderEvent) error {
	event.ID = uuid.New()
	event.CreatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO order_events (id, order_id, event_type, message, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, event.ID, event.OrderID, event.EventType, event.Message, event.CreatedAt)
	return err
}

// GetEvents retrieves all events for an order.
func (r *OrderRepository) GetEvents(ctx context.Context, orderID uuid.UUID) ([]model.OrderEvent, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, order_id, event_type, message, created_at
		FROM order_events WHERE order_id = ? ORDER BY created_at DESC
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []model.OrderEvent
	for rows.Next() {
		var event model.OrderEvent
		if err := rows.Scan(&event.ID, &event.OrderID, &event.EventType, &event.Message, &event.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

// GetProjectsByOrderID retrieves all projects linked to an order.
func (r *OrderRepository) GetProjectsByOrderID(ctx context.Context, orderID uuid.UUID) ([]model.Project, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, description, target_date, tags, template_id, source, external_order_id, customer_notes, order_id, order_item_id, created_at, updated_at
		FROM projects WHERE order_id = ?
	`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []model.Project
	for rows.Next() {
		var p model.Project
		var tagsJSON []byte
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.TargetDate, &tagsJSON, &p.TemplateID, &p.Source, &p.ExternalOrderID, &p.CustomerNotes, &p.OrderID, &p.OrderItemID, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.Tags = unmarshalStringArray(tagsJSON)
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

// CountByStatus returns counts of orders by status.
func (r *OrderRepository) CountByStatus(ctx context.Context) (map[model.OrderStatus]int, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT status, COUNT(*) FROM orders GROUP BY status
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[model.OrderStatus]int)
	for rows.Next() {
		var status model.OrderStatus
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		counts[status] = count
	}
	return counts, rows.Err()
}
