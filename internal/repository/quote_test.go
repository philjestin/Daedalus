package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/philjestin/daedalus/internal/model"
)

// createTestCustomer is a helper that creates a customer for quote FK requirements.
func createTestCustomer(t *testing.T, repo *CustomerRepository, ctx context.Context) *model.Customer {
	t.Helper()
	customer := &model.Customer{
		Name:  "Test Customer",
		Email: "test@example.com",
	}
	if err := repo.Create(ctx, customer); err != nil {
		t.Fatalf("Create test customer failed: %v", err)
	}
	return customer
}

func TestQuoteRepository_NextQuoteNumber(t *testing.T) {
	db := openTestDB(t)
	customerRepo := &CustomerRepository{db: db}
	quoteRepo := &QuoteRepository{db: db}
	ctx := context.Background()

	customer := createTestCustomer(t, customerRepo, ctx)

	// First quote number should be Q-0001
	num1, err := quoteRepo.NextQuoteNumber(ctx)
	if err != nil {
		t.Fatalf("NextQuoteNumber (1) failed: %v", err)
	}
	if num1 != "Q-0001" {
		t.Errorf("First quote number = %q, want %q", num1, "Q-0001")
	}

	// Create a quote with that number
	q1 := &model.Quote{
		QuoteNumber: num1,
		CustomerID:  customer.ID,
		Title:       "Quote 1",
	}
	if err := quoteRepo.Create(ctx, q1); err != nil {
		t.Fatalf("Create quote 1 failed: %v", err)
	}

	// Second quote number should be Q-0002
	num2, err := quoteRepo.NextQuoteNumber(ctx)
	if err != nil {
		t.Fatalf("NextQuoteNumber (2) failed: %v", err)
	}
	if num2 != "Q-0002" {
		t.Errorf("Second quote number = %q, want %q", num2, "Q-0002")
	}

	// Create second quote and verify third number
	q2 := &model.Quote{
		QuoteNumber: num2,
		CustomerID:  customer.ID,
		Title:       "Quote 2",
	}
	if err := quoteRepo.Create(ctx, q2); err != nil {
		t.Fatalf("Create quote 2 failed: %v", err)
	}

	num3, err := quoteRepo.NextQuoteNumber(ctx)
	if err != nil {
		t.Fatalf("NextQuoteNumber (3) failed: %v", err)
	}
	if num3 != "Q-0003" {
		t.Errorf("Third quote number = %q, want %q", num3, "Q-0003")
	}
}

func TestQuoteRepository_CreateAndGetByID(t *testing.T) {
	db := openTestDB(t)
	customerRepo := &CustomerRepository{db: db}
	quoteRepo := &QuoteRepository{db: db}
	ctx := context.Background()

	customer := createTestCustomer(t, customerRepo, ctx)

	validUntil := time.Now().Add(30 * 24 * time.Hour)
	quote := &model.Quote{
		QuoteNumber: "Q-0001",
		CustomerID:  customer.ID,
		Title:       "Test Quote",
		Notes:       "Some notes",
		ValidUntil:  &validUntil,
	}

	if err := quoteRepo.Create(ctx, quote); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if quote.ID == uuid.Nil {
		t.Error("Create should set quote ID")
	}
	if quote.CreatedAt.IsZero() {
		t.Error("Create should set CreatedAt")
	}
	if quote.Status != model.QuoteStatusDraft {
		t.Errorf("Create should default Status to 'draft', got %q", quote.Status)
	}

	got, err := quoteRepo.GetByID(ctx, quote.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetByID returned nil")
	}

	if got.QuoteNumber != "Q-0001" {
		t.Errorf("QuoteNumber = %q, want %q", got.QuoteNumber, "Q-0001")
	}
	if got.CustomerID != customer.ID {
		t.Errorf("CustomerID = %v, want %v", got.CustomerID, customer.ID)
	}
	if got.Title != "Test Quote" {
		t.Errorf("Title = %q, want %q", got.Title, "Test Quote")
	}
	if got.Notes != "Some notes" {
		t.Errorf("Notes = %q, want %q", got.Notes, "Some notes")
	}
	if got.ValidUntil == nil {
		t.Error("ValidUntil should not be nil")
	}
}

func TestQuoteRepository_GetByID_NotFound(t *testing.T) {
	db := openTestDB(t)
	quoteRepo := &QuoteRepository{db: db}
	ctx := context.Background()

	got, err := quoteRepo.GetByID(ctx, uuid.New())
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got != nil {
		t.Error("GetByID should return nil for non-existent quote")
	}
}

func TestQuoteRepository_List(t *testing.T) {
	db := openTestDB(t)
	customerRepo := &CustomerRepository{db: db}
	quoteRepo := &QuoteRepository{db: db}
	ctx := context.Background()

	customer1 := createTestCustomer(t, customerRepo, ctx)
	customer2 := &model.Customer{Name: "Second Customer", Email: "second@example.com"}
	if err := customerRepo.Create(ctx, customer2); err != nil {
		t.Fatalf("Create customer2 failed: %v", err)
	}

	// Create quotes with different statuses and customers
	quotes := []*model.Quote{
		{QuoteNumber: "Q-0001", CustomerID: customer1.ID, Title: "Draft 1", Status: model.QuoteStatusDraft},
		{QuoteNumber: "Q-0002", CustomerID: customer1.ID, Title: "Sent 1", Status: model.QuoteStatusSent},
		{QuoteNumber: "Q-0003", CustomerID: customer2.ID, Title: "Draft 2", Status: model.QuoteStatusDraft},
	}
	for _, q := range quotes {
		if err := quoteRepo.Create(ctx, q); err != nil {
			t.Fatalf("Create quote %q failed: %v", q.QuoteNumber, err)
		}
		time.Sleep(10 * time.Millisecond) // ensure different timestamps
	}

	// List all
	all, err := quoteRepo.List(ctx, model.QuoteFilters{})
	if err != nil {
		t.Fatalf("List all failed: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("List all: len = %d, want 3", len(all))
	}

	// Filter by status
	draftStatus := model.QuoteStatusDraft
	drafts, err := quoteRepo.List(ctx, model.QuoteFilters{Status: &draftStatus})
	if err != nil {
		t.Fatalf("List drafts failed: %v", err)
	}
	if len(drafts) != 2 {
		t.Errorf("List drafts: len = %d, want 2", len(drafts))
	}

	sentStatus := model.QuoteStatusSent
	sents, err := quoteRepo.List(ctx, model.QuoteFilters{Status: &sentStatus})
	if err != nil {
		t.Fatalf("List sent failed: %v", err)
	}
	if len(sents) != 1 {
		t.Errorf("List sent: len = %d, want 1", len(sents))
	}

	// Filter by customer_id
	c1Quotes, err := quoteRepo.List(ctx, model.QuoteFilters{CustomerID: &customer1.ID})
	if err != nil {
		t.Fatalf("List by customer1 failed: %v", err)
	}
	if len(c1Quotes) != 2 {
		t.Errorf("List by customer1: len = %d, want 2", len(c1Quotes))
	}

	c2Quotes, err := quoteRepo.List(ctx, model.QuoteFilters{CustomerID: &customer2.ID})
	if err != nil {
		t.Fatalf("List by customer2 failed: %v", err)
	}
	if len(c2Quotes) != 1 {
		t.Errorf("List by customer2: len = %d, want 1", len(c2Quotes))
	}
}

func TestQuoteRepository_Update(t *testing.T) {
	db := openTestDB(t)
	customerRepo := &CustomerRepository{db: db}
	quoteRepo := &QuoteRepository{db: db}
	ctx := context.Background()

	customer := createTestCustomer(t, customerRepo, ctx)

	quote := &model.Quote{
		QuoteNumber: "Q-0001",
		CustomerID:  customer.ID,
		Title:       "Original Title",
	}
	if err := quoteRepo.Create(ctx, quote); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	quote.Title = "Updated Title"
	quote.Notes = "Updated notes"
	quote.Status = model.QuoteStatusSent
	now := time.Now()
	quote.SentAt = &now

	if err := quoteRepo.Update(ctx, quote); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, err := quoteRepo.GetByID(ctx, quote.ID)
	if err != nil {
		t.Fatalf("GetByID after update failed: %v", err)
	}

	if got.Title != "Updated Title" {
		t.Errorf("Title = %q, want %q", got.Title, "Updated Title")
	}
	if got.Notes != "Updated notes" {
		t.Errorf("Notes = %q, want %q", got.Notes, "Updated notes")
	}
	if got.Status != model.QuoteStatusSent {
		t.Errorf("Status = %q, want %q", got.Status, model.QuoteStatusSent)
	}
	if got.SentAt == nil {
		t.Error("SentAt should be set after update")
	}
}

func TestQuoteRepository_Delete(t *testing.T) {
	db := openTestDB(t)
	customerRepo := &CustomerRepository{db: db}
	quoteRepo := &QuoteRepository{db: db}
	ctx := context.Background()

	customer := createTestCustomer(t, customerRepo, ctx)

	quote := &model.Quote{
		QuoteNumber: "Q-0001",
		CustomerID:  customer.ID,
		Title:       "To Delete",
	}
	if err := quoteRepo.Create(ctx, quote); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := quoteRepo.Delete(ctx, quote.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, err := quoteRepo.GetByID(ctx, quote.ID)
	if err != nil {
		t.Fatalf("GetByID after delete failed: %v", err)
	}
	if got != nil {
		t.Error("Quote should be nil after delete")
	}
}

func TestQuoteRepository_Options_CRUD(t *testing.T) {
	db := openTestDB(t)
	customerRepo := &CustomerRepository{db: db}
	quoteRepo := &QuoteRepository{db: db}
	ctx := context.Background()

	customer := createTestCustomer(t, customerRepo, ctx)

	quote := &model.Quote{
		QuoteNumber: "Q-0001",
		CustomerID:  customer.ID,
		Title:       "Quote with Options",
	}
	if err := quoteRepo.Create(ctx, quote); err != nil {
		t.Fatalf("Create quote failed: %v", err)
	}

	// Create option
	option := &model.QuoteOption{
		QuoteID:     quote.ID,
		Name:        "Standard Option",
		Description: "Standard turnaround",
		SortOrder:   1,
		TotalCents:  5000,
	}
	if err := quoteRepo.CreateOption(ctx, option); err != nil {
		t.Fatalf("CreateOption failed: %v", err)
	}

	if option.ID == uuid.Nil {
		t.Error("CreateOption should set option ID")
	}
	if option.CreatedAt.IsZero() {
		t.Error("CreateOption should set CreatedAt")
	}

	// GetOption
	got, err := quoteRepo.GetOption(ctx, option.ID)
	if err != nil {
		t.Fatalf("GetOption failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetOption returned nil")
	}
	if got.Name != "Standard Option" {
		t.Errorf("Name = %q, want %q", got.Name, "Standard Option")
	}
	if got.Description != "Standard turnaround" {
		t.Errorf("Description = %q, want %q", got.Description, "Standard turnaround")
	}
	if got.TotalCents != 5000 {
		t.Errorf("TotalCents = %d, want 5000", got.TotalCents)
	}

	// GetOptionsByQuoteID
	option2 := &model.QuoteOption{
		QuoteID:   quote.ID,
		Name:      "Rush Option",
		SortOrder: 2,
	}
	if err := quoteRepo.CreateOption(ctx, option2); err != nil {
		t.Fatalf("CreateOption (2) failed: %v", err)
	}

	options, err := quoteRepo.GetOptionsByQuoteID(ctx, quote.ID)
	if err != nil {
		t.Fatalf("GetOptionsByQuoteID failed: %v", err)
	}
	if len(options) != 2 {
		t.Errorf("len(options) = %d, want 2", len(options))
	}
	// Should be ordered by sort_order ASC
	if options[0].Name != "Standard Option" {
		t.Errorf("First option = %q, want %q", options[0].Name, "Standard Option")
	}
	if options[1].Name != "Rush Option" {
		t.Errorf("Second option = %q, want %q", options[1].Name, "Rush Option")
	}

	// UpdateOption
	option.Name = "Updated Option"
	option.TotalCents = 7500
	if err := quoteRepo.UpdateOption(ctx, option); err != nil {
		t.Fatalf("UpdateOption failed: %v", err)
	}

	updated, err := quoteRepo.GetOption(ctx, option.ID)
	if err != nil {
		t.Fatalf("GetOption after update failed: %v", err)
	}
	if updated.Name != "Updated Option" {
		t.Errorf("Name after update = %q, want %q", updated.Name, "Updated Option")
	}
	if updated.TotalCents != 7500 {
		t.Errorf("TotalCents after update = %d, want 7500", updated.TotalCents)
	}

	// DeleteOption
	if err := quoteRepo.DeleteOption(ctx, option2.ID); err != nil {
		t.Fatalf("DeleteOption failed: %v", err)
	}

	remaining, err := quoteRepo.GetOptionsByQuoteID(ctx, quote.ID)
	if err != nil {
		t.Fatalf("GetOptionsByQuoteID after delete failed: %v", err)
	}
	if len(remaining) != 1 {
		t.Errorf("len(options) after delete = %d, want 1", len(remaining))
	}
}

func TestQuoteRepository_LineItems_CRUD(t *testing.T) {
	db := openTestDB(t)
	customerRepo := &CustomerRepository{db: db}
	quoteRepo := &QuoteRepository{db: db}
	ctx := context.Background()

	customer := createTestCustomer(t, customerRepo, ctx)

	quote := &model.Quote{
		QuoteNumber: "Q-0001",
		CustomerID:  customer.ID,
		Title:       "Quote with Line Items",
	}
	if err := quoteRepo.Create(ctx, quote); err != nil {
		t.Fatalf("Create quote failed: %v", err)
	}

	option := &model.QuoteOption{
		QuoteID: quote.ID,
		Name:    "Option A",
	}
	if err := quoteRepo.CreateOption(ctx, option); err != nil {
		t.Fatalf("CreateOption failed: %v", err)
	}

	// Create line item
	item := &model.QuoteLineItem{
		OptionID:       option.ID,
		Type:           model.QuoteLineItemTypePrinting,
		Description:    "3D Print Widget",
		Quantity:       2,
		Unit:           "each",
		UnitPriceCents: 1500,
		TotalCents:     3000,
		SortOrder:      1,
	}
	if err := quoteRepo.CreateLineItem(ctx, item); err != nil {
		t.Fatalf("CreateLineItem failed: %v", err)
	}

	if item.ID == uuid.Nil {
		t.Error("CreateLineItem should set item ID")
	}
	if item.CreatedAt.IsZero() {
		t.Error("CreateLineItem should set CreatedAt")
	}

	// GetLineItem
	got, err := quoteRepo.GetLineItem(ctx, item.ID)
	if err != nil {
		t.Fatalf("GetLineItem failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetLineItem returned nil")
	}
	if got.Type != model.QuoteLineItemTypePrinting {
		t.Errorf("Type = %q, want %q", got.Type, model.QuoteLineItemTypePrinting)
	}
	if got.Description != "3D Print Widget" {
		t.Errorf("Description = %q, want %q", got.Description, "3D Print Widget")
	}
	if got.Quantity != 2 {
		t.Errorf("Quantity = %f, want 2", got.Quantity)
	}
	if got.UnitPriceCents != 1500 {
		t.Errorf("UnitPriceCents = %d, want 1500", got.UnitPriceCents)
	}
	if got.TotalCents != 3000 {
		t.Errorf("TotalCents = %d, want 3000", got.TotalCents)
	}

	// Create second line item
	item2 := &model.QuoteLineItem{
		OptionID:       option.ID,
		Type:           model.QuoteLineItemTypePostProcessing,
		Description:    "Sanding and painting",
		Quantity:       1,
		Unit:           "hour",
		UnitPriceCents: 5000,
		TotalCents:     5000,
		SortOrder:      2,
	}
	if err := quoteRepo.CreateLineItem(ctx, item2); err != nil {
		t.Fatalf("CreateLineItem (2) failed: %v", err)
	}

	// GetLineItemsByOptionID
	items, err := quoteRepo.GetLineItemsByOptionID(ctx, option.ID)
	if err != nil {
		t.Fatalf("GetLineItemsByOptionID failed: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("len(items) = %d, want 2", len(items))
	}
	// Should be ordered by sort_order ASC
	if items[0].Description != "3D Print Widget" {
		t.Errorf("First item = %q, want %q", items[0].Description, "3D Print Widget")
	}
	if items[1].Description != "Sanding and painting" {
		t.Errorf("Second item = %q, want %q", items[1].Description, "Sanding and painting")
	}

	// UpdateLineItem
	item.Description = "Updated Widget Print"
	item.TotalCents = 4000
	if err := quoteRepo.UpdateLineItem(ctx, item); err != nil {
		t.Fatalf("UpdateLineItem failed: %v", err)
	}

	updated, err := quoteRepo.GetLineItem(ctx, item.ID)
	if err != nil {
		t.Fatalf("GetLineItem after update failed: %v", err)
	}
	if updated.Description != "Updated Widget Print" {
		t.Errorf("Description after update = %q, want %q", updated.Description, "Updated Widget Print")
	}
	if updated.TotalCents != 4000 {
		t.Errorf("TotalCents after update = %d, want 4000", updated.TotalCents)
	}

	// DeleteLineItem
	if err := quoteRepo.DeleteLineItem(ctx, item2.ID); err != nil {
		t.Fatalf("DeleteLineItem failed: %v", err)
	}

	remaining, err := quoteRepo.GetLineItemsByOptionID(ctx, option.ID)
	if err != nil {
		t.Fatalf("GetLineItemsByOptionID after delete failed: %v", err)
	}
	if len(remaining) != 1 {
		t.Errorf("len(items) after delete = %d, want 1", len(remaining))
	}
}

func TestQuoteRepository_RecalculateOptionTotal(t *testing.T) {
	db := openTestDB(t)
	customerRepo := &CustomerRepository{db: db}
	quoteRepo := &QuoteRepository{db: db}
	ctx := context.Background()

	customer := createTestCustomer(t, customerRepo, ctx)

	quote := &model.Quote{
		QuoteNumber: "Q-0001",
		CustomerID:  customer.ID,
		Title:       "Recalc Test",
	}
	if err := quoteRepo.Create(ctx, quote); err != nil {
		t.Fatalf("Create quote failed: %v", err)
	}

	option := &model.QuoteOption{
		QuoteID:    quote.ID,
		Name:       "Option for Recalc",
		TotalCents: 0,
	}
	if err := quoteRepo.CreateOption(ctx, option); err != nil {
		t.Fatalf("CreateOption failed: %v", err)
	}

	// Add two line items
	item1 := &model.QuoteLineItem{
		OptionID:       option.ID,
		Type:           model.QuoteLineItemTypePrinting,
		Description:    "Item 1",
		Quantity:       1,
		Unit:           "each",
		UnitPriceCents: 2000,
		TotalCents:     2000,
		SortOrder:      1,
	}
	if err := quoteRepo.CreateLineItem(ctx, item1); err != nil {
		t.Fatalf("CreateLineItem (1) failed: %v", err)
	}

	item2 := &model.QuoteLineItem{
		OptionID:       option.ID,
		Type:           model.QuoteLineItemTypeDesign,
		Description:    "Item 2",
		Quantity:       1,
		Unit:           "each",
		UnitPriceCents: 3000,
		TotalCents:     3000,
		SortOrder:      2,
	}
	if err := quoteRepo.CreateLineItem(ctx, item2); err != nil {
		t.Fatalf("CreateLineItem (2) failed: %v", err)
	}

	// Recalculate
	if err := quoteRepo.RecalculateOptionTotal(ctx, option.ID); err != nil {
		t.Fatalf("RecalculateOptionTotal failed: %v", err)
	}

	got, err := quoteRepo.GetOption(ctx, option.ID)
	if err != nil {
		t.Fatalf("GetOption after recalculate failed: %v", err)
	}
	if got.TotalCents != 5000 {
		t.Errorf("TotalCents after recalculate = %d, want 5000 (2000+3000)", got.TotalCents)
	}
}

func TestQuoteRepository_Events(t *testing.T) {
	db := openTestDB(t)
	customerRepo := &CustomerRepository{db: db}
	quoteRepo := &QuoteRepository{db: db}
	ctx := context.Background()

	customer := createTestCustomer(t, customerRepo, ctx)

	quote := &model.Quote{
		QuoteNumber: "Q-0001",
		CustomerID:  customer.ID,
		Title:       "Events Test",
	}
	if err := quoteRepo.Create(ctx, quote); err != nil {
		t.Fatalf("Create quote failed: %v", err)
	}

	// Add events
	events := []*model.QuoteEvent{
		{QuoteID: quote.ID, EventType: "created", Message: "Quote created"},
		{QuoteID: quote.ID, EventType: "sent", Message: "Quote sent to customer"},
		{QuoteID: quote.ID, EventType: "accepted", Message: "Quote accepted"},
	}
	for i, e := range events {
		if err := quoteRepo.AddEvent(ctx, e); err != nil {
			t.Fatalf("AddEvent %d failed: %v", i, err)
		}
		if e.ID == uuid.Nil {
			t.Errorf("AddEvent %d should set event ID", i)
		}
		if e.CreatedAt.IsZero() {
			t.Errorf("AddEvent %d should set CreatedAt", i)
		}
		time.Sleep(10 * time.Millisecond) // ensure different timestamps
	}

	// GetEvents
	gotEvents, err := quoteRepo.GetEvents(ctx, quote.ID)
	if err != nil {
		t.Fatalf("GetEvents failed: %v", err)
	}
	if len(gotEvents) != 3 {
		t.Errorf("len(events) = %d, want 3", len(gotEvents))
	}

	// Events should be ordered by created_at DESC (most recent first)
	if gotEvents[0].EventType != "accepted" {
		t.Errorf("First event type = %q, want %q", gotEvents[0].EventType, "accepted")
	}
	if gotEvents[1].EventType != "sent" {
		t.Errorf("Second event type = %q, want %q", gotEvents[1].EventType, "sent")
	}
	if gotEvents[2].EventType != "created" {
		t.Errorf("Third event type = %q, want %q", gotEvents[2].EventType, "created")
	}

	// Verify events for a different quote returns empty
	otherEvents, err := quoteRepo.GetEvents(ctx, uuid.New())
	if err != nil {
		t.Fatalf("GetEvents for other quote failed: %v", err)
	}
	if len(otherEvents) != 0 {
		t.Errorf("Events for non-existent quote: len = %d, want 0", len(otherEvents))
	}
}
