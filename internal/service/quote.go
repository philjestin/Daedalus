package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/philjestin/daedalus/internal/model"
	"github.com/philjestin/daedalus/internal/realtime"
	"github.com/philjestin/daedalus/internal/repository"
)

// QuoteService handles quote business logic.
type QuoteService struct {
	quoteRepo    *repository.QuoteRepository
	customerRepo *repository.CustomerRepository
	orderRepo    *repository.OrderRepository
	repos        *repository.Repositories
	hub          *realtime.Hub
}

// NewQuoteService creates a new QuoteService.
func NewQuoteService(
	quoteRepo *repository.QuoteRepository,
	customerRepo *repository.CustomerRepository,
	orderRepo *repository.OrderRepository,
	repos *repository.Repositories,
	hub *realtime.Hub,
) *QuoteService {
	return &QuoteService{
		quoteRepo:    quoteRepo,
		customerRepo: customerRepo,
		orderRepo:    orderRepo,
		repos:        repos,
		hub:          hub,
	}
}

// Create creates a new quote with auto-generated quote number.
func (s *QuoteService) Create(ctx context.Context, quote *model.Quote) error {
	if quote.Title == "" {
		return fmt.Errorf("quote title is required")
	}
	if quote.CustomerID == uuid.Nil {
		return fmt.Errorf("customer is required")
	}

	// Verify customer exists
	customer, err := s.customerRepo.GetByID(ctx, quote.CustomerID)
	if err != nil {
		return fmt.Errorf("failed to verify customer: %w", err)
	}
	if customer == nil {
		return fmt.Errorf("customer not found")
	}

	// Auto-generate quote number
	quoteNumber, err := s.quoteRepo.NextQuoteNumber(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate quote number: %w", err)
	}
	quote.QuoteNumber = quoteNumber

	if err := s.quoteRepo.Create(ctx, quote); err != nil {
		return err
	}

	// Add creation event
	s.quoteRepo.AddEvent(ctx, &model.QuoteEvent{
		QuoteID:   quote.ID,
		EventType: "created",
		Message:   fmt.Sprintf("Quote %s created", quote.QuoteNumber),
	})

	s.broadcastUpdate("quote_created", quote)
	slog.Info("quote created", "id", quote.ID, "number", quote.QuoteNumber)
	return nil
}

// GetByID retrieves a quote by ID with customer, options (with line items), and events.
func (s *QuoteService) GetByID(ctx context.Context, id uuid.UUID) (*model.Quote, error) {
	quote, err := s.quoteRepo.GetByID(ctx, id)
	if err != nil || quote == nil {
		return quote, err
	}

	// Load customer
	customer, err := s.customerRepo.GetByID(ctx, quote.CustomerID)
	if err != nil {
		return nil, err
	}
	quote.Customer = customer

	// Load options with line items
	options, err := s.quoteRepo.GetOptionsByQuoteID(ctx, id)
	if err != nil {
		return nil, err
	}
	for i, opt := range options {
		items, err := s.quoteRepo.GetLineItemsByOptionID(ctx, opt.ID)
		if err != nil {
			return nil, err
		}
		options[i].Items = items
	}
	quote.Options = options

	// Load events
	events, err := s.quoteRepo.GetEvents(ctx, id)
	if err != nil {
		return nil, err
	}
	quote.Events = events

	return quote, nil
}

// List retrieves quotes with optional filtering, enriched with customer data.
func (s *QuoteService) List(ctx context.Context, filters model.QuoteFilters) ([]model.Quote, error) {
	quotes, err := s.quoteRepo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	// Enrich with customer data and option totals
	for i, q := range quotes {
		customer, err := s.customerRepo.GetByID(ctx, q.CustomerID)
		if err == nil && customer != nil {
			quotes[i].Customer = customer
		}
		options, err := s.quoteRepo.GetOptionsByQuoteID(ctx, q.ID)
		if err == nil {
			quotes[i].Options = options
		}
	}

	return quotes, nil
}

// Update updates a quote (only allowed in draft status).
func (s *QuoteService) Update(ctx context.Context, quote *model.Quote) error {
	existing, err := s.quoteRepo.GetByID(ctx, quote.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("quote not found")
	}
	if existing.Status != model.QuoteStatusDraft {
		return fmt.Errorf("can only edit quotes in draft status")
	}

	if err := s.quoteRepo.Update(ctx, quote); err != nil {
		return err
	}

	s.broadcastUpdate("quote_updated", quote)
	return nil
}

// Delete removes a quote.
func (s *QuoteService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.quoteRepo.Delete(ctx, id); err != nil {
		return err
	}
	s.broadcastUpdate("quote_deleted", map[string]interface{}{"id": id})
	return nil
}

// Send transitions a quote from draft to sent.
func (s *QuoteService) Send(ctx context.Context, id uuid.UUID) (*model.Quote, error) {
	quote, err := s.quoteRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if quote == nil {
		return nil, fmt.Errorf("quote not found")
	}
	if quote.Status != model.QuoteStatusDraft {
		return nil, fmt.Errorf("can only send quotes in draft status")
	}

	now := time.Now()
	quote.Status = model.QuoteStatusSent
	quote.SentAt = &now

	if err := s.quoteRepo.Update(ctx, quote); err != nil {
		return nil, err
	}

	s.quoteRepo.AddEvent(ctx, &model.QuoteEvent{
		QuoteID:   id,
		EventType: "sent",
		Message:   "Quote sent to customer",
	})

	s.broadcastUpdate("quote_updated", quote)
	slog.Info("quote sent", "id", id, "number", quote.QuoteNumber)
	return quote, nil
}

// Accept transitions a quote to accepted and creates an Order from the chosen option.
func (s *QuoteService) Accept(ctx context.Context, quoteID uuid.UUID, optionID uuid.UUID) (*model.Quote, error) {
	quote, err := s.quoteRepo.GetByID(ctx, quoteID)
	if err != nil {
		return nil, err
	}
	if quote == nil {
		return nil, fmt.Errorf("quote not found")
	}
	if quote.Status != model.QuoteStatusSent {
		return nil, fmt.Errorf("can only accept quotes in sent status")
	}

	// Verify option belongs to this quote
	option, err := s.quoteRepo.GetOption(ctx, optionID)
	if err != nil {
		return nil, err
	}
	if option == nil || option.QuoteID != quoteID {
		return nil, fmt.Errorf("option not found for this quote")
	}

	// Load customer for order creation
	customer, err := s.customerRepo.GetByID(ctx, quote.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("failed to load customer: %w", err)
	}
	if customer == nil {
		return nil, fmt.Errorf("customer not found")
	}

	// Load line items for the accepted option
	items, err := s.quoteRepo.GetLineItemsByOptionID(ctx, optionID)
	if err != nil {
		return nil, fmt.Errorf("failed to load line items: %w", err)
	}

	// Create the order
	order := &model.Order{
		Source:        model.OrderSourceQuote,
		SourceOrderID: quote.QuoteNumber,
		CustomerID:    &customer.ID,
		CustomerName:  customer.Name,
		CustomerEmail: customer.Email,
		Notes:         fmt.Sprintf("From quote %s - %s", quote.QuoteNumber, option.Name),
	}

	if err := s.orderRepo.Create(ctx, order); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	// Create order items from line items that are printing-type or linked to a project
	for _, item := range items {
		if item.Type == model.QuoteLineItemTypePrinting || item.ProjectID != nil {
			orderItem := &model.OrderItem{
				OrderID:   order.ID,
				ProjectID: item.ProjectID,
				Quantity:  int(item.Quantity),
				Notes:     item.Description,
			}
			if orderItem.Quantity < 1 {
				orderItem.Quantity = 1
			}
			s.orderRepo.AddItem(ctx, orderItem)
		}
	}

	// Add event to order
	s.orderRepo.AddEvent(ctx, &model.OrderEvent{
		OrderID:   order.ID,
		EventType: "created",
		Message:   fmt.Sprintf("Order created from quote %s", quote.QuoteNumber),
	})

	// Update quote status
	now := time.Now()
	quote.Status = model.QuoteStatusAccepted
	quote.AcceptedOptionID = &optionID
	quote.OrderID = &order.ID
	quote.AcceptedAt = &now

	if err := s.quoteRepo.Update(ctx, quote); err != nil {
		return nil, err
	}

	s.quoteRepo.AddEvent(ctx, &model.QuoteEvent{
		QuoteID:   quoteID,
		EventType: "accepted",
		Message:   fmt.Sprintf("Quote accepted with option '%s', order %s created", option.Name, order.ID),
	})

	s.broadcastUpdate("quote_updated", quote)
	s.broadcastUpdate("order_created", order)
	slog.Info("quote accepted", "id", quoteID, "option", optionID, "order", order.ID)
	return quote, nil
}

// Reject transitions a quote to rejected.
func (s *QuoteService) Reject(ctx context.Context, id uuid.UUID) (*model.Quote, error) {
	quote, err := s.quoteRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if quote == nil {
		return nil, fmt.Errorf("quote not found")
	}
	if quote.Status != model.QuoteStatusSent {
		return nil, fmt.Errorf("can only reject quotes in sent status")
	}

	quote.Status = model.QuoteStatusRejected
	if err := s.quoteRepo.Update(ctx, quote); err != nil {
		return nil, err
	}

	s.quoteRepo.AddEvent(ctx, &model.QuoteEvent{
		QuoteID:   id,
		EventType: "rejected",
		Message:   "Quote rejected by customer",
	})

	s.broadcastUpdate("quote_updated", quote)
	slog.Info("quote rejected", "id", id, "number", quote.QuoteNumber)
	return quote, nil
}

// CreateOption adds an option to a quote.
func (s *QuoteService) CreateOption(ctx context.Context, option *model.QuoteOption) error {
	if option.Name == "" {
		return fmt.Errorf("option name is required")
	}

	quote, err := s.quoteRepo.GetByID(ctx, option.QuoteID)
	if err != nil {
		return err
	}
	if quote == nil {
		return fmt.Errorf("quote not found")
	}

	if err := s.quoteRepo.CreateOption(ctx, option); err != nil {
		return err
	}

	s.broadcastUpdate("quote_updated", map[string]interface{}{"id": option.QuoteID})
	return nil
}

// UpdateOption updates a quote option.
func (s *QuoteService) UpdateOption(ctx context.Context, option *model.QuoteOption) error {
	if err := s.quoteRepo.UpdateOption(ctx, option); err != nil {
		return err
	}
	s.broadcastUpdate("quote_updated", map[string]interface{}{"id": option.QuoteID})
	return nil
}

// DeleteOption removes a quote option.
func (s *QuoteService) DeleteOption(ctx context.Context, quoteID uuid.UUID, optionID uuid.UUID) error {
	if err := s.quoteRepo.DeleteOption(ctx, optionID); err != nil {
		return err
	}
	s.broadcastUpdate("quote_updated", map[string]interface{}{"id": quoteID})
	return nil
}

// CreateLineItem adds a line item to an option and recalculates the total.
func (s *QuoteService) CreateLineItem(ctx context.Context, item *model.QuoteLineItem) error {
	if item.Description == "" {
		return fmt.Errorf("line item description is required")
	}
	if item.Type == "" {
		item.Type = model.QuoteLineItemTypeOther
	}
	if item.Unit == "" {
		item.Unit = "each"
	}

	if err := s.quoteRepo.CreateLineItem(ctx, item); err != nil {
		return err
	}

	// Recalculate option total
	if err := s.quoteRepo.RecalculateOptionTotal(ctx, item.OptionID); err != nil {
		slog.Warn("failed to recalculate option total", "option_id", item.OptionID, "error", err)
	}

	return nil
}

// UpdateLineItem updates a line item and recalculates the option total.
func (s *QuoteService) UpdateLineItem(ctx context.Context, item *model.QuoteLineItem) error {
	if err := s.quoteRepo.UpdateLineItem(ctx, item); err != nil {
		return err
	}

	// Recalculate option total
	if err := s.quoteRepo.RecalculateOptionTotal(ctx, item.OptionID); err != nil {
		slog.Warn("failed to recalculate option total", "option_id", item.OptionID, "error", err)
	}

	return nil
}

// DeleteLineItem removes a line item and recalculates the option total.
func (s *QuoteService) DeleteLineItem(ctx context.Context, optionID uuid.UUID, itemID uuid.UUID) error {
	if err := s.quoteRepo.DeleteLineItem(ctx, itemID); err != nil {
		return err
	}

	// Recalculate option total
	if err := s.quoteRepo.RecalculateOptionTotal(ctx, optionID); err != nil {
		slog.Warn("failed to recalculate option total", "option_id", optionID, "error", err)
	}

	return nil
}

func (s *QuoteService) broadcastUpdate(eventType string, data interface{}) {
	if s.hub != nil {
		s.hub.Broadcast(model.BroadcastEvent{
			Type: eventType,
			Data: data,
		})
	}
}
