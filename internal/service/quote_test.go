package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/philjestin/daedalus/internal/database"
	"github.com/philjestin/daedalus/internal/model"
	"github.com/philjestin/daedalus/internal/repository"
)

func newQuoteTestServices(t *testing.T) (*QuoteService, *CustomerService, *OrderService, *repository.Repositories) {
	t.Helper()
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	repos := repository.NewRepositories(db)
	customerSvc := NewCustomerService(repos.Customers, nil)
	orderSvc := NewOrderService(repos.Orders, repos.Projects, repos.PrintJobs, nil, nil)
	quoteSvc := NewQuoteService(repos.Quotes, repos.Customers, repos.Orders, repos, nil)
	return quoteSvc, customerSvc, orderSvc, repos
}

// createQuoteTestCustomer creates a customer for use in quote tests.
func createQuoteTestCustomer(t *testing.T, customerSvc *CustomerService, ctx context.Context) *model.Customer {
	t.Helper()
	customer := &model.Customer{
		Name:  "Test Customer",
		Email: "test@example.com",
	}
	if err := customerSvc.Create(ctx, customer); err != nil {
		t.Fatalf("Create customer failed: %v", err)
	}
	return customer
}

func TestQuoteService_Create(t *testing.T) {
	quoteSvc, customerSvc, _, _ := newQuoteTestServices(t)
	ctx := context.Background()

	customer := createQuoteTestCustomer(t, customerSvc, ctx)

	quote := &model.Quote{
		CustomerID: customer.ID,
		Title:      "New Widget Quote",
		Notes:      "Rush order",
	}

	if err := quoteSvc.Create(ctx, quote); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if quote.ID == uuid.Nil {
		t.Error("Create should set quote ID")
	}
	if quote.QuoteNumber != "Q-0001" {
		t.Errorf("QuoteNumber = %q, want %q", quote.QuoteNumber, "Q-0001")
	}
	if quote.Status != model.QuoteStatusDraft {
		t.Errorf("Status = %q, want %q", quote.Status, model.QuoteStatusDraft)
	}

	// Verify events were added
	got, err := quoteSvc.GetByID(ctx, quote.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if len(got.Events) == 0 {
		t.Error("Create should add a creation event")
	}
	foundCreated := false
	for _, e := range got.Events {
		if e.EventType == "created" {
			foundCreated = true
			break
		}
	}
	if !foundCreated {
		t.Error("Expected a 'created' event")
	}

	// Second quote should get Q-0002
	quote2 := &model.Quote{
		CustomerID: customer.ID,
		Title:      "Second Quote",
	}
	if err := quoteSvc.Create(ctx, quote2); err != nil {
		t.Fatalf("Create second quote failed: %v", err)
	}
	if quote2.QuoteNumber != "Q-0002" {
		t.Errorf("Second QuoteNumber = %q, want %q", quote2.QuoteNumber, "Q-0002")
	}
}

func TestQuoteService_Create_Validation(t *testing.T) {
	quoteSvc, customerSvc, _, _ := newQuoteTestServices(t)
	ctx := context.Background()

	customer := createQuoteTestCustomer(t, customerSvc, ctx)

	t.Run("empty title", func(t *testing.T) {
		q := &model.Quote{
			CustomerID: customer.ID,
			Title:      "",
		}
		if err := quoteSvc.Create(ctx, q); err == nil {
			t.Error("Expected error for empty title")
		}
	})

	t.Run("nil customer ID", func(t *testing.T) {
		q := &model.Quote{
			CustomerID: uuid.Nil,
			Title:      "Test",
		}
		if err := quoteSvc.Create(ctx, q); err == nil {
			t.Error("Expected error for nil customer ID")
		}
	})

	t.Run("non-existent customer", func(t *testing.T) {
		q := &model.Quote{
			CustomerID: uuid.New(),
			Title:      "Test",
		}
		if err := quoteSvc.Create(ctx, q); err == nil {
			t.Error("Expected error for non-existent customer")
		}
	})
}

func TestQuoteService_Send(t *testing.T) {
	quoteSvc, customerSvc, _, _ := newQuoteTestServices(t)
	ctx := context.Background()

	customer := createQuoteTestCustomer(t, customerSvc, ctx)

	quote := &model.Quote{
		CustomerID: customer.ID,
		Title:      "Quote to Send",
	}
	if err := quoteSvc.Create(ctx, quote); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Send the quote
	sent, err := quoteSvc.Send(ctx, quote.ID)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	if sent.Status != model.QuoteStatusSent {
		t.Errorf("Status = %q, want %q", sent.Status, model.QuoteStatusSent)
	}
	if sent.SentAt == nil {
		t.Error("SentAt should be set after Send")
	}

	// Verify via GetByID
	got, err := quoteSvc.GetByID(ctx, quote.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.Status != model.QuoteStatusSent {
		t.Errorf("Persisted status = %q, want %q", got.Status, model.QuoteStatusSent)
	}

	// Verify a "sent" event was added
	foundSent := false
	for _, e := range got.Events {
		if e.EventType == "sent" {
			foundSent = true
			break
		}
	}
	if !foundSent {
		t.Error("Expected a 'sent' event after Send")
	}
}

func TestQuoteService_Send_InvalidStatus(t *testing.T) {
	quoteSvc, customerSvc, _, repos := newQuoteTestServices(t)
	ctx := context.Background()

	customer := createQuoteTestCustomer(t, customerSvc, ctx)

	// Create and manually set to "sent" status
	quote := &model.Quote{
		CustomerID: customer.ID,
		Title:      "Already Sent",
	}
	if err := quoteSvc.Create(ctx, quote); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Force status to sent
	quote.Status = model.QuoteStatusSent
	if err := repos.Quotes.Update(ctx, quote); err != nil {
		t.Fatalf("Update status failed: %v", err)
	}

	// Try to send again
	_, err := quoteSvc.Send(ctx, quote.ID)
	if err == nil {
		t.Error("Expected error when sending a non-draft quote")
	}
}

func TestQuoteService_Accept(t *testing.T) {
	quoteSvc, customerSvc, _, repos := newQuoteTestServices(t)
	ctx := context.Background()

	// Create customer
	customer := createQuoteTestCustomer(t, customerSvc, ctx)

	// Create quote
	quote := &model.Quote{
		CustomerID: customer.ID,
		Title:      "Quote to Accept",
	}
	if err := quoteSvc.Create(ctx, quote); err != nil {
		t.Fatalf("Create quote failed: %v", err)
	}

	// Add an option
	option := &model.QuoteOption{
		QuoteID: quote.ID,
		Name:    "Standard",
	}
	if err := quoteSvc.CreateOption(ctx, option); err != nil {
		t.Fatalf("CreateOption failed: %v", err)
	}

	// Add line items to the option
	item1 := &model.QuoteLineItem{
		OptionID:       option.ID,
		Type:           model.QuoteLineItemTypePrinting,
		Description:    "Widget print",
		Quantity:       3,
		Unit:           "each",
		UnitPriceCents: 1000,
		TotalCents:     3000,
	}
	if err := quoteSvc.CreateLineItem(ctx, item1); err != nil {
		t.Fatalf("CreateLineItem (1) failed: %v", err)
	}

	item2 := &model.QuoteLineItem{
		OptionID:       option.ID,
		Type:           model.QuoteLineItemTypeDesign,
		Description:    "Design work",
		Quantity:       1,
		Unit:           "hour",
		UnitPriceCents: 5000,
		TotalCents:     5000,
	}
	if err := quoteSvc.CreateLineItem(ctx, item2); err != nil {
		t.Fatalf("CreateLineItem (2) failed: %v", err)
	}

	// Send the quote first (required before accept)
	_, err := quoteSvc.Send(ctx, quote.ID)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Accept the quote with the option
	accepted, err := quoteSvc.Accept(ctx, quote.ID, option.ID)
	if err != nil {
		t.Fatalf("Accept failed: %v", err)
	}

	if accepted.Status != model.QuoteStatusAccepted {
		t.Errorf("Status = %q, want %q", accepted.Status, model.QuoteStatusAccepted)
	}
	if accepted.AcceptedOptionID == nil || *accepted.AcceptedOptionID != option.ID {
		t.Errorf("AcceptedOptionID = %v, want %v", accepted.AcceptedOptionID, option.ID)
	}
	if accepted.OrderID == nil {
		t.Fatal("OrderID should be set after Accept")
	}
	if accepted.AcceptedAt == nil {
		t.Error("AcceptedAt should be set after Accept")
	}

	// Verify the order was created
	order, err := repos.Orders.GetByID(ctx, *accepted.OrderID)
	if err != nil {
		t.Fatalf("GetByID order failed: %v", err)
	}
	if order == nil {
		t.Fatal("Order should exist after Accept")
	}
	if order.Source != model.OrderSourceQuote {
		t.Errorf("Order.Source = %q, want %q", order.Source, model.OrderSourceQuote)
	}
	if order.SourceOrderID != quote.QuoteNumber {
		t.Errorf("Order.SourceOrderID = %q, want %q", order.SourceOrderID, quote.QuoteNumber)
	}
	if order.CustomerName != customer.Name {
		t.Errorf("Order.CustomerName = %q, want %q", order.CustomerName, customer.Name)
	}
	if order.CustomerEmail != customer.Email {
		t.Errorf("Order.CustomerEmail = %q, want %q", order.CustomerEmail, customer.Email)
	}

	// Verify quote events include "accepted"
	got, err := quoteSvc.GetByID(ctx, quote.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	foundAccepted := false
	for _, e := range got.Events {
		if e.EventType == "accepted" {
			foundAccepted = true
			break
		}
	}
	if !foundAccepted {
		t.Error("Expected an 'accepted' event")
	}
}

func TestQuoteService_Accept_InvalidStatus(t *testing.T) {
	quoteSvc, customerSvc, _, _ := newQuoteTestServices(t)
	ctx := context.Background()

	customer := createQuoteTestCustomer(t, customerSvc, ctx)

	// Create a draft quote (not sent)
	quote := &model.Quote{
		CustomerID: customer.ID,
		Title:      "Draft Quote",
	}
	if err := quoteSvc.Create(ctx, quote); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	option := &model.QuoteOption{
		QuoteID: quote.ID,
		Name:    "Option A",
	}
	if err := quoteSvc.CreateOption(ctx, option); err != nil {
		t.Fatalf("CreateOption failed: %v", err)
	}

	// Try to accept a draft quote (should fail, must be sent first)
	_, err := quoteSvc.Accept(ctx, quote.ID, option.ID)
	if err == nil {
		t.Error("Expected error when accepting a draft quote")
	}
}

func TestQuoteService_Reject(t *testing.T) {
	quoteSvc, customerSvc, _, _ := newQuoteTestServices(t)
	ctx := context.Background()

	customer := createQuoteTestCustomer(t, customerSvc, ctx)

	quote := &model.Quote{
		CustomerID: customer.ID,
		Title:      "Quote to Reject",
	}
	if err := quoteSvc.Create(ctx, quote); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Send first
	_, err := quoteSvc.Send(ctx, quote.ID)
	if err != nil {
		t.Fatalf("Send failed: %v", err)
	}

	// Reject
	rejected, err := quoteSvc.Reject(ctx, quote.ID)
	if err != nil {
		t.Fatalf("Reject failed: %v", err)
	}

	if rejected.Status != model.QuoteStatusRejected {
		t.Errorf("Status = %q, want %q", rejected.Status, model.QuoteStatusRejected)
	}

	// Verify persisted state
	got, err := quoteSvc.GetByID(ctx, quote.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got.Status != model.QuoteStatusRejected {
		t.Errorf("Persisted status = %q, want %q", got.Status, model.QuoteStatusRejected)
	}

	// Verify a "rejected" event was added
	foundRejected := false
	for _, e := range got.Events {
		if e.EventType == "rejected" {
			foundRejected = true
			break
		}
	}
	if !foundRejected {
		t.Error("Expected a 'rejected' event")
	}
}

func TestQuoteService_LineItem_RecalculatesTotal(t *testing.T) {
	quoteSvc, customerSvc, _, repos := newQuoteTestServices(t)
	ctx := context.Background()

	customer := createQuoteTestCustomer(t, customerSvc, ctx)

	quote := &model.Quote{
		CustomerID: customer.ID,
		Title:      "Recalc Test",
	}
	if err := quoteSvc.Create(ctx, quote); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	option := &model.QuoteOption{
		QuoteID: quote.ID,
		Name:    "Recalc Option",
	}
	if err := quoteSvc.CreateOption(ctx, option); err != nil {
		t.Fatalf("CreateOption failed: %v", err)
	}

	// Add first line item
	item1 := &model.QuoteLineItem{
		OptionID:       option.ID,
		Type:           model.QuoteLineItemTypePrinting,
		Description:    "Item A",
		Quantity:       1,
		Unit:           "each",
		UnitPriceCents: 2000,
		TotalCents:     2000,
	}
	if err := quoteSvc.CreateLineItem(ctx, item1); err != nil {
		t.Fatalf("CreateLineItem (1) failed: %v", err)
	}

	// Check option total after first item
	opt1, err := repos.Quotes.GetOption(ctx, option.ID)
	if err != nil {
		t.Fatalf("GetOption failed: %v", err)
	}
	if opt1.TotalCents != 2000 {
		t.Errorf("TotalCents after item 1 = %d, want 2000", opt1.TotalCents)
	}

	// Add second line item
	item2 := &model.QuoteLineItem{
		OptionID:       option.ID,
		Type:           model.QuoteLineItemTypePostProcessing,
		Description:    "Item B",
		Quantity:       1,
		Unit:           "each",
		UnitPriceCents: 3500,
		TotalCents:     3500,
	}
	if err := quoteSvc.CreateLineItem(ctx, item2); err != nil {
		t.Fatalf("CreateLineItem (2) failed: %v", err)
	}

	// Check option total after second item
	opt2, err := repos.Quotes.GetOption(ctx, option.ID)
	if err != nil {
		t.Fatalf("GetOption failed: %v", err)
	}
	if opt2.TotalCents != 5500 {
		t.Errorf("TotalCents after item 2 = %d, want 5500 (2000+3500)", opt2.TotalCents)
	}

	// Update first item's total
	item1.TotalCents = 4000
	if err := quoteSvc.UpdateLineItem(ctx, item1); err != nil {
		t.Fatalf("UpdateLineItem failed: %v", err)
	}

	// Check option total after update
	opt3, err := repos.Quotes.GetOption(ctx, option.ID)
	if err != nil {
		t.Fatalf("GetOption failed: %v", err)
	}
	if opt3.TotalCents != 7500 {
		t.Errorf("TotalCents after update = %d, want 7500 (4000+3500)", opt3.TotalCents)
	}

	// Delete second item
	if err := quoteSvc.DeleteLineItem(ctx, option.ID, item2.ID); err != nil {
		t.Fatalf("DeleteLineItem failed: %v", err)
	}

	// Check option total after delete
	opt4, err := repos.Quotes.GetOption(ctx, option.ID)
	if err != nil {
		t.Fatalf("GetOption failed: %v", err)
	}
	if opt4.TotalCents != 4000 {
		t.Errorf("TotalCents after delete = %d, want 4000", opt4.TotalCents)
	}
}
