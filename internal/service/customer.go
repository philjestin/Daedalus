package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/philjestin/daedalus/internal/model"
	"github.com/philjestin/daedalus/internal/realtime"
	"github.com/philjestin/daedalus/internal/repository"
)

// CustomerService handles customer business logic.
type CustomerService struct {
	repo *repository.CustomerRepository
	hub  *realtime.Hub
}

// NewCustomerService creates a new CustomerService.
func NewCustomerService(repo *repository.CustomerRepository, hub *realtime.Hub) *CustomerService {
	return &CustomerService{repo: repo, hub: hub}
}

// Create creates a new customer.
func (s *CustomerService) Create(ctx context.Context, customer *model.Customer) error {
	if customer.Name == "" {
		return fmt.Errorf("customer name is required")
	}

	if err := s.repo.Create(ctx, customer); err != nil {
		return err
	}

	s.broadcastUpdate("customer_created", customer)
	slog.Info("customer created", "id", customer.ID, "name", customer.Name)
	return nil
}

// GetByID retrieves a customer by ID.
func (s *CustomerService) GetByID(ctx context.Context, id uuid.UUID) (*model.Customer, error) {
	return s.repo.GetByID(ctx, id)
}

// List retrieves customers with optional search filtering.
func (s *CustomerService) List(ctx context.Context, filters model.CustomerFilters) ([]model.Customer, error) {
	return s.repo.List(ctx, filters)
}

// Update updates a customer.
func (s *CustomerService) Update(ctx context.Context, customer *model.Customer) error {
	if customer.Name == "" {
		return fmt.Errorf("customer name is required")
	}

	if err := s.repo.Update(ctx, customer); err != nil {
		return err
	}

	s.broadcastUpdate("customer_updated", customer)
	return nil
}

// Delete removes a customer.
func (s *CustomerService) Delete(ctx context.Context, id uuid.UUID) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}
	s.broadcastUpdate("customer_deleted", map[string]interface{}{"id": id})
	return nil
}

func (s *CustomerService) broadcastUpdate(eventType string, data interface{}) {
	if s.hub != nil {
		s.hub.Broadcast(model.BroadcastEvent{
			Type: eventType,
			Data: data,
		})
	}
}
