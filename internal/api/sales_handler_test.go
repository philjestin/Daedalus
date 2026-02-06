package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hyperion/printfarm/internal/model"
	"github.com/hyperion/printfarm/internal/service"
)

func TestGetWeeklyInsights_Empty(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/sales/weekly-insights", nil)
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var insights service.WeeklyInsights
	if err := json.NewDecoder(rr.Body).Decode(&insights); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if insights.ThisWeek.Count != 0 {
		t.Errorf("expected 0 sales this week, got %d", insights.ThisWeek.Count)
	}
	if insights.LastWeek.Count != 0 {
		t.Errorf("expected 0 sales last week, got %d", insights.LastWeek.Count)
	}
	if insights.ThisWeek.GrossCents != 0 {
		t.Errorf("expected 0 gross this week, got %d", insights.ThisWeek.GrossCents)
	}
	if len(insights.Channels) != 0 {
		t.Errorf("expected 0 channels, got %d", len(insights.Channels))
	}
	if insights.WeekStart == "" {
		t.Error("expected week_start to be set")
	}
	if insights.PendingCount != 0 {
		t.Errorf("expected 0 pending count, got %d", insights.PendingCount)
	}
	if insights.PendingRevenueCents != 0 {
		t.Errorf("expected 0 pending revenue, got %d", insights.PendingRevenueCents)
	}
}

func TestGetWeeklyInsights_WithSales(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	now := time.Now().UTC()
	// Calculate this Monday
	weekday := now.Weekday()
	if weekday == time.Sunday {
		weekday = 7
	}
	thisMonday := time.Date(now.Year(), now.Month(), now.Day()-int(weekday-time.Monday), 12, 0, 0, 0, time.UTC)

	// Create sales in this week
	sale1 := &model.Sale{
		OccurredAt:      thisMonday,
		Channel:         model.SalesChannelEtsy,
		GrossCents:      5000,
		FeesCents:       500,
		NetCents:        4500,
		ItemDescription: "Widget A",
		Quantity:        1,
	}
	sale2 := &model.Sale{
		OccurredAt:      thisMonday.Add(24 * time.Hour),
		Channel:         model.SalesChannelEtsy,
		GrossCents:      3000,
		FeesCents:       300,
		NetCents:        2700,
		ItemDescription: "Widget B",
		Quantity:        1,
	}
	sale3 := &model.Sale{
		OccurredAt:      thisMonday.Add(24 * time.Hour),
		Channel:         model.SalesChannelWebsite,
		GrossCents:      2000,
		FeesCents:       0,
		NetCents:        2000,
		ItemDescription: "Widget C",
		Quantity:        1,
	}

	for _, s := range []*model.Sale{sale1, sale2, sale3} {
		if err := env.services.Sales.Create(ctx, s); err != nil {
			t.Fatalf("create sale: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/sales/weekly-insights", nil)
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var insights service.WeeklyInsights
	if err := json.NewDecoder(rr.Body).Decode(&insights); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if insights.ThisWeek.Count != 3 {
		t.Errorf("expected 3 sales this week, got %d", insights.ThisWeek.Count)
	}
	if insights.ThisWeek.GrossCents != 10000 {
		t.Errorf("expected 10000 gross this week, got %d", insights.ThisWeek.GrossCents)
	}
	if insights.ThisWeek.NetCents != 9200 {
		t.Errorf("expected 9200 net this week, got %d", insights.ThisWeek.NetCents)
	}
	if insights.ThisWeek.FeesCents != 800 {
		t.Errorf("expected 800 fees this week, got %d", insights.ThisWeek.FeesCents)
	}
	if insights.LastWeek.Count != 0 {
		t.Errorf("expected 0 sales last week, got %d", insights.LastWeek.Count)
	}
	if len(insights.Channels) != 2 {
		t.Errorf("expected 2 channels, got %d", len(insights.Channels))
	}
}

func TestGetWeeklyInsights_PendingSales(t *testing.T) {
	env := newTestEnv(t)
	ctx := context.Background()

	// Project A: has price_cents set
	priceCents := 2500
	projectA := &model.Project{
		Name:       "Priced Widget",
		PriceCents: &priceCents,
	}
	if err := env.services.Projects.Create(ctx, projectA); err != nil {
		t.Fatalf("create projectA: %v", err)
	}

	// Project B: no price_cents, but has sales history (avg $50 = 5000c)
	projectB := &model.Project{Name: "Widget With History"}
	if err := env.services.Projects.Create(ctx, projectB); err != nil {
		t.Fatalf("create projectB: %v", err)
	}
	for _, gross := range []int{4000, 6000} {
		sale := &model.Sale{
			OccurredAt:      time.Now().UTC().Add(-48 * time.Hour),
			Channel:         model.SalesChannelEtsy,
			GrossCents:      gross,
			NetCents:        gross,
			ProjectID:       &projectB.ID,
			ItemDescription: "past sale",
			Quantity:        1,
		}
		if err := env.services.Sales.Create(ctx, sale); err != nil {
			t.Fatalf("create sale: %v", err)
		}
	}

	// Project C: no price, no sales history
	projectC := &model.Project{Name: "Unknown Value Widget"}
	if err := env.services.Projects.Create(ctx, projectC); err != nil {
		t.Fatalf("create projectC: %v", err)
	}

	// Tasks:
	// A: pending qty 2 → 2 * 2500 = 5000
	// B: in_progress qty 3 → 3 * 5000 (avg from sales) = 15000
	// A: completed qty 1 → should NOT count
	// C: pending qty 4 → 4 * 0 (no price info) = 0
	tasks := []*model.Task{
		{ProjectID: projectA.ID, Name: "Task A", Status: model.TaskStatusPending, Quantity: 2},
		{ProjectID: projectB.ID, Name: "Task B", Status: model.TaskStatusInProgress, Quantity: 3},
		{ProjectID: projectA.ID, Name: "Task A done", Status: model.TaskStatusCompleted, Quantity: 1},
		{ProjectID: projectC.ID, Name: "Task C", Status: model.TaskStatusPending, Quantity: 4},
	}
	for _, task := range tasks {
		if err := env.services.Tasks.Create(ctx, task); err != nil {
			t.Fatalf("create task: %v", err)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/api/sales/weekly-insights", nil)
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var insights service.WeeklyInsights
	if err := json.NewDecoder(rr.Body).Decode(&insights); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	// task A (qty 2) + task B (qty 3) + task C (qty 4) = 9 pending units
	if insights.PendingCount != 9 {
		t.Errorf("expected 9 pending units, got %d", insights.PendingCount)
	}
	// A: 2*2500=5000, B: 3*5000=15000, C: 4*0=0 → total 20000
	if insights.PendingRevenueCents != 20000 {
		t.Errorf("expected 20000 pending revenue cents, got %d", insights.PendingRevenueCents)
	}
}
