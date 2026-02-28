package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/philjestin/daedalus/internal/model"
)

// QuoteRepository handles quote database operations.
type QuoteRepository struct {
	db *sql.DB
}

// NextQuoteNumber generates the next sequential quote number (Q-0001, Q-0002, ...).
func (r *QuoteRepository) NextQuoteNumber(ctx context.Context) (string, error) {
	var maxNum int
	row := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(MAX(CAST(SUBSTR(quote_number, 3) AS INTEGER)), 0)
		FROM quotes
	`)
	if err := row.Scan(&maxNum); err != nil {
		return "", err
	}
	return fmt.Sprintf("Q-%04d", maxNum+1), nil
}

// quoteColumns lists all quote columns for SELECT queries.
const quoteColumns = `id, quote_number, customer_id, status, title, notes, valid_until, accepted_option_id, order_id,
	discount_type, discount_value, rush_fee_cents, tax_rate, terms, requested_due_date,
	billing_address_json, shipping_address_json, share_token,
	created_at, updated_at, sent_at, accepted_at`

// scanQuote scans a row into a Quote struct, handling JSON address fields and time conversion.
func scanQuote(s scannable) (*model.Quote, error) {
	var q model.Quote
	var billingJSON, shippingJSON, shareToken *string
	err := scanRow(s,
		&q.ID, &q.QuoteNumber, &q.CustomerID, &q.Status, &q.Title, &q.Notes, &q.ValidUntil,
		&q.AcceptedOptionID, &q.OrderID,
		&q.DiscountType, &q.DiscountValue, &q.RushFeeCents, &q.TaxRate, &q.Terms, &q.RequestedDueDate,
		&billingJSON, &shippingJSON, &shareToken,
		&q.CreatedAt, &q.UpdatedAt, &q.SentAt, &q.AcceptedAt,
	)
	if err != nil {
		return nil, err
	}
	q.BillingAddress = unmarshalAddress(billingJSON)
	q.ShippingAddress = unmarshalAddress(shippingJSON)
	if shareToken != nil {
		q.ShareToken = *shareToken
	}
	if q.DiscountType == "" {
		q.DiscountType = model.DiscountTypeNone
	}
	return &q, nil
}

// Create inserts a new quote.
func (r *QuoteRepository) Create(ctx context.Context, quote *model.Quote) error {
	quote.ID = uuid.New()
	quote.CreatedAt = time.Now()
	quote.UpdatedAt = time.Now()
	if quote.Status == "" {
		quote.Status = model.QuoteStatusDraft
	}
	if quote.DiscountType == "" {
		quote.DiscountType = model.DiscountTypeNone
	}

	// Store NULL instead of empty string for share_token to allow UNIQUE index
	var shareToken *string
	if quote.ShareToken != "" {
		shareToken = &quote.ShareToken
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO quotes (id, quote_number, customer_id, status, title, notes, valid_until, accepted_option_id, order_id,
			discount_type, discount_value, rush_fee_cents, tax_rate, terms, requested_due_date,
			billing_address_json, shipping_address_json, share_token,
			created_at, updated_at, sent_at, accepted_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, quote.ID, quote.QuoteNumber, quote.CustomerID, quote.Status, quote.Title, quote.Notes, quote.ValidUntil,
		quote.AcceptedOptionID, quote.OrderID,
		quote.DiscountType, quote.DiscountValue, quote.RushFeeCents, quote.TaxRate, quote.Terms, quote.RequestedDueDate,
		marshalAddress(quote.BillingAddress), marshalAddress(quote.ShippingAddress), shareToken,
		quote.CreatedAt, quote.UpdatedAt, quote.SentAt, quote.AcceptedAt)
	return err
}

// GetByID retrieves a quote by ID.
func (r *QuoteRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Quote, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+quoteColumns+` FROM quotes WHERE id = ?`, id)
	q, err := scanQuote(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return q, err
}

// GetByShareToken retrieves a quote by its share token.
func (r *QuoteRepository) GetByShareToken(ctx context.Context, token string) (*model.Quote, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+quoteColumns+` FROM quotes WHERE share_token = ?`, token)
	q, err := scanQuote(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return q, err
}

// List retrieves quotes with optional filtering.
func (r *QuoteRepository) List(ctx context.Context, filters model.QuoteFilters) ([]model.Quote, error) {
	query := `SELECT ` + quoteColumns + ` FROM quotes WHERE 1=1`
	args := []interface{}{}

	if filters.Status != nil {
		query += " AND status = ?"
		args = append(args, *filters.Status)
	}
	if filters.CustomerID != nil {
		query += " AND customer_id = ?"
		args = append(args, *filters.CustomerID)
	}

	query += " ORDER BY created_at DESC"

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

	var quotes []model.Quote
	for rows.Next() {
		q, err := scanQuote(rows)
		if err != nil {
			return nil, err
		}
		quotes = append(quotes, *q)
	}
	return quotes, rows.Err()
}

// Update updates a quote.
func (r *QuoteRepository) Update(ctx context.Context, quote *model.Quote) error {
	quote.UpdatedAt = time.Now()

	var shareToken *string
	if quote.ShareToken != "" {
		shareToken = &quote.ShareToken
	}

	_, err := r.db.ExecContext(ctx, `
		UPDATE quotes SET customer_id = ?, status = ?, title = ?, notes = ?, valid_until = ?,
			accepted_option_id = ?, order_id = ?,
			discount_type = ?, discount_value = ?, rush_fee_cents = ?, tax_rate = ?,
			terms = ?, requested_due_date = ?,
			billing_address_json = ?, shipping_address_json = ?, share_token = ?,
			updated_at = ?, sent_at = ?, accepted_at = ?
		WHERE id = ?
	`, quote.CustomerID, quote.Status, quote.Title, quote.Notes, quote.ValidUntil,
		quote.AcceptedOptionID, quote.OrderID,
		quote.DiscountType, quote.DiscountValue, quote.RushFeeCents, quote.TaxRate,
		quote.Terms, quote.RequestedDueDate,
		marshalAddress(quote.BillingAddress), marshalAddress(quote.ShippingAddress), shareToken,
		quote.UpdatedAt, quote.SentAt, quote.AcceptedAt,
		quote.ID)
	return err
}

// Delete removes a quote.
func (r *QuoteRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM quotes WHERE id = ?`, id)
	return err
}

// CreateOption inserts a new quote option.
func (r *QuoteRepository) CreateOption(ctx context.Context, option *model.QuoteOption) error {
	option.ID = uuid.New()
	option.CreatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO quote_options (id, quote_id, name, description, sort_order, total_cents, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, option.ID, option.QuoteID, option.Name, option.Description, option.SortOrder, option.TotalCents, option.CreatedAt)
	return err
}

// GetOption retrieves a quote option by ID.
func (r *QuoteRepository) GetOption(ctx context.Context, id uuid.UUID) (*model.QuoteOption, error) {
	var o model.QuoteOption
	row := r.db.QueryRowContext(ctx, `
		SELECT id, quote_id, name, description, sort_order, total_cents, created_at
		FROM quote_options WHERE id = ?
	`, id)
	err := scanRow(row, &o.ID, &o.QuoteID, &o.Name, &o.Description, &o.SortOrder, &o.TotalCents, &o.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &o, err
}

// GetOptionsByQuoteID retrieves all options for a quote.
func (r *QuoteRepository) GetOptionsByQuoteID(ctx context.Context, quoteID uuid.UUID) ([]model.QuoteOption, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, quote_id, name, description, sort_order, total_cents, created_at
		FROM quote_options WHERE quote_id = ? ORDER BY sort_order ASC
	`, quoteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var options []model.QuoteOption
	for rows.Next() {
		var o model.QuoteOption
		if err := scanRow(rows, &o.ID, &o.QuoteID, &o.Name, &o.Description, &o.SortOrder, &o.TotalCents, &o.CreatedAt); err != nil {
			return nil, err
		}
		options = append(options, o)
	}
	return options, rows.Err()
}

// UpdateOption updates a quote option.
func (r *QuoteRepository) UpdateOption(ctx context.Context, option *model.QuoteOption) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE quote_options SET name = ?, description = ?, sort_order = ?, total_cents = ?
		WHERE id = ?
	`, option.Name, option.Description, option.SortOrder, option.TotalCents, option.ID)
	return err
}

// DeleteOption removes a quote option.
func (r *QuoteRepository) DeleteOption(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM quote_options WHERE id = ?`, id)
	return err
}

// CreateLineItem inserts a new line item.
func (r *QuoteRepository) CreateLineItem(ctx context.Context, item *model.QuoteLineItem) error {
	item.ID = uuid.New()
	item.CreatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO quote_line_items (id, option_id, type, description, quantity, unit, unit_price_cents, total_cents, sort_order, project_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, item.ID, item.OptionID, item.Type, item.Description, item.Quantity, item.Unit, item.UnitPriceCents, item.TotalCents, item.SortOrder, item.ProjectID, item.CreatedAt)
	return err
}

// GetLineItem retrieves a line item by ID.
func (r *QuoteRepository) GetLineItem(ctx context.Context, id uuid.UUID) (*model.QuoteLineItem, error) {
	var item model.QuoteLineItem
	row := r.db.QueryRowContext(ctx, `
		SELECT id, option_id, type, description, quantity, unit, unit_price_cents, total_cents, sort_order, project_id, created_at
		FROM quote_line_items WHERE id = ?
	`, id)
	err := scanRow(row, &item.ID, &item.OptionID, &item.Type, &item.Description, &item.Quantity, &item.Unit, &item.UnitPriceCents, &item.TotalCents, &item.SortOrder, &item.ProjectID, &item.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &item, err
}

// GetLineItemsByOptionID retrieves all line items for an option.
func (r *QuoteRepository) GetLineItemsByOptionID(ctx context.Context, optionID uuid.UUID) ([]model.QuoteLineItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, option_id, type, description, quantity, unit, unit_price_cents, total_cents, sort_order, project_id, created_at
		FROM quote_line_items WHERE option_id = ? ORDER BY sort_order ASC
	`, optionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.QuoteLineItem
	for rows.Next() {
		var item model.QuoteLineItem
		if err := scanRow(rows, &item.ID, &item.OptionID, &item.Type, &item.Description, &item.Quantity, &item.Unit, &item.UnitPriceCents, &item.TotalCents, &item.SortOrder, &item.ProjectID, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// UpdateLineItem updates a line item.
func (r *QuoteRepository) UpdateLineItem(ctx context.Context, item *model.QuoteLineItem) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE quote_line_items SET type = ?, description = ?, quantity = ?, unit = ?, unit_price_cents = ?, total_cents = ?, sort_order = ?, project_id = ?
		WHERE id = ?
	`, item.Type, item.Description, item.Quantity, item.Unit, item.UnitPriceCents, item.TotalCents, item.SortOrder, item.ProjectID, item.ID)
	return err
}

// DeleteLineItem removes a line item.
func (r *QuoteRepository) DeleteLineItem(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM quote_line_items WHERE id = ?`, id)
	return err
}

// RecalculateOptionTotal updates the option's total_cents to the sum of its line items.
func (r *QuoteRepository) RecalculateOptionTotal(ctx context.Context, optionID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE quote_options SET total_cents = (
			SELECT COALESCE(SUM(total_cents), 0) FROM quote_line_items WHERE option_id = ?
		) WHERE id = ?
	`, optionID, optionID)
	return err
}

// AddEvent adds an event to the quote history.
func (r *QuoteRepository) AddEvent(ctx context.Context, event *model.QuoteEvent) error {
	event.ID = uuid.New()
	event.CreatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO quote_events (id, quote_id, event_type, message, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, event.ID, event.QuoteID, event.EventType, event.Message, event.CreatedAt)
	return err
}

// GetEvents retrieves all events for a quote.
func (r *QuoteRepository) GetEvents(ctx context.Context, quoteID uuid.UUID) ([]model.QuoteEvent, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, quote_id, event_type, message, created_at
		FROM quote_events WHERE quote_id = ? ORDER BY created_at DESC
	`, quoteID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []model.QuoteEvent
	for rows.Next() {
		var event model.QuoteEvent
		if err := scanRow(rows, &event.ID, &event.QuoteID, &event.EventType, &event.Message, &event.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, rows.Err()
}
