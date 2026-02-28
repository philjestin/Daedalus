package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/philjestin/daedalus/internal/model"
	"github.com/philjestin/daedalus/internal/realtime"
	"github.com/philjestin/daedalus/internal/repository"
)

// AlertService handles alert business logic including low-spool detection.
type AlertService struct {
	spoolRepo    *repository.SpoolRepository
	materialRepo *repository.MaterialRepository
	orderRepo    *repository.OrderRepository
	dismissRepo  *repository.AlertDismissalRepository
	hub          *realtime.Hub
}

// NewAlertService creates a new AlertService.
func NewAlertService(
	spoolRepo *repository.SpoolRepository,
	materialRepo *repository.MaterialRepository,
	orderRepo *repository.OrderRepository,
	dismissRepo *repository.AlertDismissalRepository,
	hub *realtime.Hub,
) *AlertService {
	return &AlertService{
		spoolRepo:    spoolRepo,
		materialRepo: materialRepo,
		orderRepo:    orderRepo,
		dismissRepo:  dismissRepo,
		hub:          hub,
	}
}

// GetActiveAlerts retrieves all active alerts.
func (s *AlertService) GetActiveAlerts(ctx context.Context) ([]model.Alert, error) {
	var alerts []model.Alert

	// Get low spool alerts
	lowSpoolAlerts, err := s.GetLowSpoolAlerts(ctx)
	if err != nil {
		return nil, err
	}
	alerts = append(alerts, lowSpoolAlerts...)

	// Get order due alerts
	orderDueAlerts, err := s.GetOrderDueAlerts(ctx)
	if err != nil {
		return nil, err
	}
	alerts = append(alerts, orderDueAlerts...)

	return alerts, nil
}

// GetLowSpoolAlerts retrieves alerts for spools below their material's low threshold.
func (s *AlertService) GetLowSpoolAlerts(ctx context.Context) ([]model.Alert, error) {
	spools, err := s.spoolRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	var alerts []model.Alert

	for _, spool := range spools {
		// Skip archived spools
		if spool.Status == model.SpoolStatusArchived || spool.Status == model.SpoolStatusEmpty {
			continue
		}

		// Get the material to check threshold
		material, err := s.materialRepo.GetByID(ctx, spool.MaterialID)
		if err != nil || material == nil {
			continue
		}

		threshold := material.LowThresholdGrams
		if threshold == 0 {
			threshold = 100 // Default threshold
		}

		// Check if spool is below threshold
		if spool.RemainingWeight <= float64(threshold) {
			// Check if dismissed
			dismissed, err := s.dismissRepo.IsDismissed(ctx, model.AlertTypeLowSpool, spool.ID.String())
			if err != nil || dismissed {
				continue
			}

			severity := model.AlertSeverityWarning
			if spool.RemainingWeight <= 0 {
				severity = model.AlertSeverityCritical
			}

			alertType := model.AlertTypeLowSpool
			if spool.RemainingWeight <= 0 {
				alertType = model.AlertTypeEmptySpool
			}

			alert := model.Alert{
				ID:         fmt.Sprintf("%s-%s", alertType, spool.ID.String()),
				Type:       alertType,
				Severity:   severity,
				EntityID:   spool.ID,
				EntityType: "spool",
				Message:    fmt.Sprintf("%s %s is low (%.0fg remaining)", material.Name, material.Color, spool.RemainingWeight),
				CreatedAt:  time.Now(),
			}
			alerts = append(alerts, alert)
		}
	}

	return alerts, nil
}

// GetOrderDueAlerts retrieves alerts for orders with approaching due dates.
func (s *AlertService) GetOrderDueAlerts(ctx context.Context) ([]model.Alert, error) {
	// Get pending and in-progress orders
	filters := model.OrderFilters{
		Limit: 100,
	}
	orders, err := s.orderRepo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	var alerts []model.Alert
	now := time.Now()

	for _, order := range orders {
		// Only check pending and in-progress orders with due dates
		if order.Status != model.OrderStatusPending && order.Status != model.OrderStatusInProgress {
			continue
		}
		if order.DueDate == nil {
			continue
		}

		// Check if dismissed
		dismissed, err := s.dismissRepo.IsDismissed(ctx, model.AlertTypeOrderDue, order.ID.String())
		if err != nil || dismissed {
			continue
		}

		hoursUntilDue := order.DueDate.Sub(now).Hours()

		var severity model.AlertSeverity
		var message string

		if hoursUntilDue < 0 {
			severity = model.AlertSeverityCritical
			message = fmt.Sprintf("Order for %s is overdue", order.CustomerName)
		} else if hoursUntilDue <= 24 {
			severity = model.AlertSeverityCritical
			message = fmt.Sprintf("Order for %s is due today", order.CustomerName)
		} else if hoursUntilDue <= 72 {
			severity = model.AlertSeverityWarning
			message = fmt.Sprintf("Order for %s is due in %.0f hours", order.CustomerName, hoursUntilDue)
		} else {
			continue // Not urgent enough to alert
		}

		alert := model.Alert{
			ID:         fmt.Sprintf("%s-%s", model.AlertTypeOrderDue, order.ID.String()),
			Type:       model.AlertTypeOrderDue,
			Severity:   severity,
			EntityID:   order.ID,
			EntityType: "order",
			Message:    message,
			CreatedAt:  time.Now(),
		}
		alerts = append(alerts, alert)
	}

	return alerts, nil
}

// DismissAlert dismisses an alert with optional snooze duration.
func (s *AlertService) DismissAlert(ctx context.Context, alertType model.AlertType, entityID string, until *time.Time) error {
	// Clean up any existing dismissals for this entity
	if err := s.dismissRepo.DeleteByEntity(ctx, alertType, entityID); err != nil {
		return err
	}

	dismissal := &model.AlertDismissal{
		AlertType:      alertType,
		EntityID:       entityID,
		DismissedUntil: until,
	}
	return s.dismissRepo.Create(ctx, dismissal)
}

// UndismissAlert removes a dismissal, allowing the alert to show again.
func (s *AlertService) UndismissAlert(ctx context.Context, alertType model.AlertType, entityID string) error {
	return s.dismissRepo.DeleteByEntity(ctx, alertType, entityID)
}

// CheckAndBroadcastAlerts checks for new alerts and broadcasts them via WebSocket.
func (s *AlertService) CheckAndBroadcastAlerts(ctx context.Context) error {
	alerts, err := s.GetActiveAlerts(ctx)
	if err != nil {
		return err
	}

	if len(alerts) > 0 && s.hub != nil {
		s.hub.Broadcast(model.BroadcastEvent{
			Type: "alerts_updated",
			Data: map[string]interface{}{
				"alerts": alerts,
				"count":  len(alerts),
			},
		})
	}

	return nil
}

// GetAlertCounts returns counts of alerts by severity.
func (s *AlertService) GetAlertCounts(ctx context.Context) (map[model.AlertSeverity]int, error) {
	alerts, err := s.GetActiveAlerts(ctx)
	if err != nil {
		return nil, err
	}

	counts := map[model.AlertSeverity]int{
		model.AlertSeverityInfo:     0,
		model.AlertSeverityWarning:  0,
		model.AlertSeverityCritical: 0,
	}

	for _, alert := range alerts {
		counts[alert.Severity]++
	}

	return counts, nil
}

// SnoozeDurations provides standard snooze options.
var SnoozeDurations = map[string]time.Duration{
	"1h":  1 * time.Hour,
	"4h":  4 * time.Hour,
	"24h": 24 * time.Hour,
}

// GetSnoozeUntil calculates the snooze end time for a given duration key.
func GetSnoozeUntil(durationKey string) *time.Time {
	if durationKey == "permanent" {
		return nil // nil = permanent dismissal
	}
	duration, ok := SnoozeDurations[durationKey]
	if !ok {
		duration = 1 * time.Hour // Default to 1 hour
	}
	until := time.Now().Add(duration)
	return &until
}

// CleanupExpiredDismissals removes expired snooze dismissals.
func (s *AlertService) CleanupExpiredDismissals(ctx context.Context) error {
	return s.dismissRepo.CleanupExpired(ctx)
}

// UpdateMaterialThreshold updates the low threshold for a material and checks for new alerts.
func (s *AlertService) UpdateMaterialThreshold(ctx context.Context, materialID uuid.UUID, thresholdGrams int) error {
	material, err := s.materialRepo.GetByID(ctx, materialID)
	if err != nil {
		return err
	}
	if material == nil {
		return fmt.Errorf("material not found")
	}

	material.LowThresholdGrams = thresholdGrams
	if err := s.materialRepo.Update(ctx, material); err != nil {
		return err
	}

	// Check for new alerts after threshold change
	return s.CheckAndBroadcastAlerts(ctx)
}
