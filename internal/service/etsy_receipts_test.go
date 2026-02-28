package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/philjestin/daedalus/internal/etsy"
	"github.com/philjestin/daedalus/internal/model"
)

func TestConvertAPIReceiptToModel(t *testing.T) {
	shopID := int64(12345)

	apiReceipt := etsy.APIReceipt{
		ReceiptID:       67890,
		BuyerUserID:     11111,
		BuyerEmail:      "buyer@example.com",
		Name:            "John Doe",
		FirstLine:       "123 Main St",
		SecondLine:      "Apt 4",
		City:            "Anytown",
		State:           "CA",
		Zip:             "12345",
		CountryISO:      "US",
		Status:          "paid",
		MessageFromBuyer: "Please ship quickly!",
		IsPaid:          true,
		IsShipped:       false,
		IsGift:          true,
		GiftMessage:     "Happy Birthday!",
		CreateTimestamp: 1700000000,
		UpdateTimestamp: 1700001000,
		Grandtotal: etsy.EtsyMoney{
			Amount:       5000,
			Divisor:      100,
			CurrencyCode: "USD",
		},
		Subtotal: etsy.EtsyMoney{
			Amount:       4500,
			Divisor:      100,
			CurrencyCode: "USD",
		},
		TotalPrice: etsy.EtsyMoney{
			Amount:       4500,
			Divisor:      100,
			CurrencyCode: "USD",
		},
		TotalShippingCost: etsy.EtsyMoney{
			Amount:       500,
			Divisor:      100,
			CurrencyCode: "USD",
		},
		TotalTaxCost: etsy.EtsyMoney{
			Amount:       0,
			Divisor:      100,
			CurrencyCode: "USD",
		},
		DiscountAmt: etsy.EtsyMoney{
			Amount:       0,
			Divisor:      100,
			CurrencyCode: "USD",
		},
	}

	receipt := convertAPIReceiptToModel(apiReceipt, shopID)

	// Verify basic fields
	if receipt.EtsyReceiptID != apiReceipt.ReceiptID {
		t.Errorf("expected EtsyReceiptID %d, got %d", apiReceipt.ReceiptID, receipt.EtsyReceiptID)
	}
	if receipt.EtsyShopID != shopID {
		t.Errorf("expected EtsyShopID %d, got %d", shopID, receipt.EtsyShopID)
	}
	if receipt.BuyerUserID != apiReceipt.BuyerUserID {
		t.Errorf("expected BuyerUserID %d, got %d", apiReceipt.BuyerUserID, receipt.BuyerUserID)
	}
	if receipt.BuyerEmail != apiReceipt.BuyerEmail {
		t.Errorf("expected BuyerEmail %s, got %s", apiReceipt.BuyerEmail, receipt.BuyerEmail)
	}
	if receipt.Name != apiReceipt.Name {
		t.Errorf("expected Name %s, got %s", apiReceipt.Name, receipt.Name)
	}

	// Verify status fields
	if receipt.Status != apiReceipt.Status {
		t.Errorf("expected Status %s, got %s", apiReceipt.Status, receipt.Status)
	}
	if receipt.IsPaid != apiReceipt.IsPaid {
		t.Errorf("expected IsPaid %v, got %v", apiReceipt.IsPaid, receipt.IsPaid)
	}
	if receipt.IsShipped != apiReceipt.IsShipped {
		t.Errorf("expected IsShipped %v, got %v", apiReceipt.IsShipped, receipt.IsShipped)
	}
	if receipt.IsGift != apiReceipt.IsGift {
		t.Errorf("expected IsGift %v, got %v", apiReceipt.IsGift, receipt.IsGift)
	}
	if receipt.GiftMessage != apiReceipt.GiftMessage {
		t.Errorf("expected GiftMessage %s, got %s", apiReceipt.GiftMessage, receipt.GiftMessage)
	}
	if receipt.MessageFromBuyer != apiReceipt.MessageFromBuyer {
		t.Errorf("expected MessageFromBuyer %s, got %s", apiReceipt.MessageFromBuyer, receipt.MessageFromBuyer)
	}

	// Verify money fields
	if receipt.GrandtotalCents != 5000 {
		t.Errorf("expected GrandtotalCents 5000, got %d", receipt.GrandtotalCents)
	}
	if receipt.SubtotalCents != 4500 {
		t.Errorf("expected SubtotalCents 4500, got %d", receipt.SubtotalCents)
	}
	if receipt.TotalPriceCents != 4500 {
		t.Errorf("expected TotalPriceCents 4500, got %d", receipt.TotalPriceCents)
	}
	if receipt.TotalShippingCostCents != 500 {
		t.Errorf("expected TotalShippingCostCents 500, got %d", receipt.TotalShippingCostCents)
	}
	if receipt.Currency != "USD" {
		t.Errorf("expected Currency USD, got %s", receipt.Currency)
	}

	// Verify shipping address
	if receipt.ShippingAddressFirstLine != apiReceipt.FirstLine {
		t.Errorf("expected ShippingAddressFirstLine %s, got %s", apiReceipt.FirstLine, receipt.ShippingAddressFirstLine)
	}
	if receipt.ShippingAddressSecondLine != apiReceipt.SecondLine {
		t.Errorf("expected ShippingAddressSecondLine %s, got %s", apiReceipt.SecondLine, receipt.ShippingAddressSecondLine)
	}
	if receipt.ShippingCity != apiReceipt.City {
		t.Errorf("expected ShippingCity %s, got %s", apiReceipt.City, receipt.ShippingCity)
	}
	if receipt.ShippingState != apiReceipt.State {
		t.Errorf("expected ShippingState %s, got %s", apiReceipt.State, receipt.ShippingState)
	}
	if receipt.ShippingZip != apiReceipt.Zip {
		t.Errorf("expected ShippingZip %s, got %s", apiReceipt.Zip, receipt.ShippingZip)
	}
	if receipt.ShippingCountryCode != apiReceipt.CountryISO {
		t.Errorf("expected ShippingCountryCode %s, got %s", apiReceipt.CountryISO, receipt.ShippingCountryCode)
	}

	// Verify timestamps
	if receipt.CreateTimestamp == nil {
		t.Error("expected CreateTimestamp to be set")
	} else {
		expectedTime := time.Unix(apiReceipt.CreateTimestamp, 0)
		if !receipt.CreateTimestamp.Equal(expectedTime) {
			t.Errorf("expected CreateTimestamp %v, got %v", expectedTime, *receipt.CreateTimestamp)
		}
	}
	if receipt.UpdateTimestamp == nil {
		t.Error("expected UpdateTimestamp to be set")
	}

	// Verify defaults
	if receipt.IsProcessed {
		t.Error("expected IsProcessed to be false by default")
	}
	if receipt.ProjectID != nil {
		t.Error("expected ProjectID to be nil by default")
	}
}

func TestConvertAPITransactionToItem(t *testing.T) {
	receiptID := uuid.New()

	apiTx := etsy.APITransaction{
		TransactionID: 12345,
		Title:         "Custom Widget",
		Description:   "A handmade widget",
		ListingID:     67890,
		Quantity:      2,
		SKU:           "WIDGET-001",
		IsDigital:     false,
		Price: etsy.EtsyMoney{
			Amount:       2500,
			Divisor:      100,
			CurrencyCode: "USD",
		},
		ShippingCost: etsy.EtsyMoney{
			Amount:       500,
			Divisor:      100,
			CurrencyCode: "USD",
		},
		Variations: []etsy.APIVariation{
			{
				PropertyID:     1,
				FormattedName:  "Color",
				FormattedValue: "Blue",
			},
			{
				PropertyID:     2,
				FormattedName:  "Size",
				FormattedValue: "Large",
			},
		},
	}

	item := convertAPITransactionToItem(apiTx, receiptID)

	// Verify basic fields
	if item.EtsyReceiptItemID != apiTx.TransactionID {
		t.Errorf("expected EtsyReceiptItemID %d, got %d", apiTx.TransactionID, item.EtsyReceiptItemID)
	}
	if item.ReceiptID != receiptID {
		t.Errorf("expected ReceiptID %s, got %s", receiptID, item.ReceiptID)
	}
	if item.EtsyListingID != apiTx.ListingID {
		t.Errorf("expected EtsyListingID %d, got %d", apiTx.ListingID, item.EtsyListingID)
	}
	if item.EtsyTransactionID != apiTx.TransactionID {
		t.Errorf("expected EtsyTransactionID %d, got %d", apiTx.TransactionID, item.EtsyTransactionID)
	}

	// Verify content fields
	if item.Title != apiTx.Title {
		t.Errorf("expected Title %s, got %s", apiTx.Title, item.Title)
	}
	if item.Description != apiTx.Description {
		t.Errorf("expected Description %s, got %s", apiTx.Description, item.Description)
	}
	if item.Quantity != apiTx.Quantity {
		t.Errorf("expected Quantity %d, got %d", apiTx.Quantity, item.Quantity)
	}
	if item.SKU != apiTx.SKU {
		t.Errorf("expected SKU %s, got %s", apiTx.SKU, item.SKU)
	}
	if item.IsDigital != apiTx.IsDigital {
		t.Errorf("expected IsDigital %v, got %v", apiTx.IsDigital, item.IsDigital)
	}

	// Verify money fields
	if item.PriceCents != 2500 {
		t.Errorf("expected PriceCents 2500, got %d", item.PriceCents)
	}
	if item.ShippingCostCents != 500 {
		t.Errorf("expected ShippingCostCents 500, got %d", item.ShippingCostCents)
	}

	// Verify variations were serialized
	if item.Variations == nil {
		t.Error("expected Variations to be set")
	}

	// Verify defaults
	if item.TemplateID != nil {
		t.Error("expected TemplateID to be nil by default")
	}
}

func TestConvertAPITransactionToItem_NoVariations(t *testing.T) {
	receiptID := uuid.New()

	apiTx := etsy.APITransaction{
		TransactionID: 99999,
		Title:         "Simple Item",
		ListingID:     88888,
		Quantity:      1,
		Price: etsy.EtsyMoney{
			Amount:       1000,
			Divisor:      100,
			CurrencyCode: "USD",
		},
	}

	item := convertAPITransactionToItem(apiTx, receiptID)

	// Variations should be nil when empty
	if item.Variations != nil {
		t.Error("expected Variations to be nil when no variations")
	}
}

func TestSyncResult_Aggregation(t *testing.T) {
	t.Run("initial state", func(t *testing.T) {
		result := &model.SyncResult{}

		if result.TotalFetched != 0 {
			t.Errorf("expected TotalFetched 0, got %d", result.TotalFetched)
		}
		if result.Created != 0 {
			t.Errorf("expected Created 0, got %d", result.Created)
		}
		if result.Updated != 0 {
			t.Errorf("expected Updated 0, got %d", result.Updated)
		}
		if result.Skipped != 0 {
			t.Errorf("expected Skipped 0, got %d", result.Skipped)
		}
		if result.Errors != 0 {
			t.Errorf("expected Errors 0, got %d", result.Errors)
		}
	})

	t.Run("counts add up correctly", func(t *testing.T) {
		result := &model.SyncResult{
			TotalFetched: 10,
			Created:      5,
			Updated:      3,
			Skipped:      1,
			Errors:       1,
		}

		// Total fetched should equal sum of created + updated + skipped + errors
		sum := result.Created + result.Updated + result.Skipped + result.Errors
		if result.TotalFetched != sum {
			t.Errorf("TotalFetched (%d) should equal sum of outcomes (%d)", result.TotalFetched, sum)
		}
	})

	t.Run("increment operations", func(t *testing.T) {
		result := &model.SyncResult{}

		result.TotalFetched = 5
		result.Created++
		result.Created++
		result.Updated++
		result.Errors++
		result.Skipped++

		if result.TotalFetched != 5 {
			t.Errorf("expected TotalFetched 5, got %d", result.TotalFetched)
		}
		if result.Created != 2 {
			t.Errorf("expected Created 2, got %d", result.Created)
		}
		if result.Updated != 1 {
			t.Errorf("expected Updated 1, got %d", result.Updated)
		}
		if result.Errors != 1 {
			t.Errorf("expected Errors 1, got %d", result.Errors)
		}
		if result.Skipped != 1 {
			t.Errorf("expected Skipped 1, got %d", result.Skipped)
		}
	})
}

func TestEtsyReceipt_ProcessedState(t *testing.T) {
	receipt := &model.EtsyReceipt{
		ID:              uuid.New(),
		EtsyReceiptID:   12345,
		EtsyShopID:      67890,
		Name:            "Test Buyer",
		Status:          "paid",
		GrandtotalCents: 5000,
	}

	// Initial state
	if receipt.IsProcessed {
		t.Error("new receipt should not be processed")
	}
	if receipt.ProjectID != nil {
		t.Error("new receipt should not have a project ID")
	}

	// After processing
	projectID := uuid.New()
	receipt.IsProcessed = true
	receipt.ProjectID = &projectID

	if !receipt.IsProcessed {
		t.Error("processed receipt should be marked as processed")
	}
	if receipt.ProjectID == nil {
		t.Error("processed receipt should have a project ID")
	}
	if *receipt.ProjectID != projectID {
		t.Errorf("expected project ID %s, got %s", projectID, *receipt.ProjectID)
	}
}

func TestEtsyReceiptItem_TemplateLinking(t *testing.T) {
	item := &model.EtsyReceiptItem{
		ID:                uuid.New(),
		EtsyReceiptItemID: 12345,
		ReceiptID:         uuid.New(),
		EtsyListingID:     67890,
		EtsyTransactionID: 12345,
		Title:             "Test Item",
		Quantity:          1,
		PriceCents:        2500,
		SKU:               "TEST-SKU",
	}

	// Initial state - no template
	if item.TemplateID != nil {
		t.Error("new item should not have a template ID")
	}

	// Link to template
	templateID := uuid.New()
	item.TemplateID = &templateID

	if item.TemplateID == nil {
		t.Error("linked item should have a template ID")
	}
	if *item.TemplateID != templateID {
		t.Errorf("expected template ID %s, got %s", templateID, *item.TemplateID)
	}
}
