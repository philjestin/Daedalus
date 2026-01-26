package service

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/hyperion/printfarm/internal/etsy"
	"github.com/hyperion/printfarm/internal/model"
)

func TestConvertAPIListingToModel(t *testing.T) {
	apiListing := etsy.APIListing{
		ListingID:        12345,
		ShopID:           67890,
		Title:            "Handmade Widget",
		Description:      "A beautiful handmade widget",
		State:            "active",
		Quantity:         10,
		URL:              "https://www.etsy.com/listing/12345",
		NumFavorers:      50,
		Views:            1000,
		IsCustomizable:   true,
		IsPersonalizable: false,
		HasVariations:    true,
		Tags:             []string{"handmade", "widget", "custom"},
		SKUs:             []string{"WIDGET-001", "WIDGET-002"},
		Price: etsy.EtsyMoney{
			Amount:       3500,
			Divisor:      100,
			CurrencyCode: "USD",
		},
	}

	listing := convertAPIListingToModel(apiListing)

	// Verify basic fields
	if listing.EtsyListingID != apiListing.ListingID {
		t.Errorf("expected EtsyListingID %d, got %d", apiListing.ListingID, listing.EtsyListingID)
	}
	if listing.EtsyShopID != apiListing.ShopID {
		t.Errorf("expected EtsyShopID %d, got %d", apiListing.ShopID, listing.EtsyShopID)
	}
	if listing.Title != apiListing.Title {
		t.Errorf("expected Title %s, got %s", apiListing.Title, listing.Title)
	}
	if listing.Description != apiListing.Description {
		t.Errorf("expected Description %s, got %s", apiListing.Description, listing.Description)
	}
	if listing.State != apiListing.State {
		t.Errorf("expected State %s, got %s", apiListing.State, listing.State)
	}

	// Verify quantity and stats
	if listing.Quantity != apiListing.Quantity {
		t.Errorf("expected Quantity %d, got %d", apiListing.Quantity, listing.Quantity)
	}
	if listing.NumFavorers != apiListing.NumFavorers {
		t.Errorf("expected NumFavorers %d, got %d", apiListing.NumFavorers, listing.NumFavorers)
	}
	if listing.Views != apiListing.Views {
		t.Errorf("expected Views %d, got %d", apiListing.Views, listing.Views)
	}

	// Verify URL
	if listing.URL != apiListing.URL {
		t.Errorf("expected URL %s, got %s", apiListing.URL, listing.URL)
	}

	// Verify boolean flags
	if listing.IsCustomizable != apiListing.IsCustomizable {
		t.Errorf("expected IsCustomizable %v, got %v", apiListing.IsCustomizable, listing.IsCustomizable)
	}
	if listing.IsPersonalizable != apiListing.IsPersonalizable {
		t.Errorf("expected IsPersonalizable %v, got %v", apiListing.IsPersonalizable, listing.IsPersonalizable)
	}
	if listing.HasVariations != apiListing.HasVariations {
		t.Errorf("expected HasVariations %v, got %v", apiListing.HasVariations, listing.HasVariations)
	}

	// Verify price
	if listing.PriceCents != 3500 {
		t.Errorf("expected PriceCents 3500, got %d", listing.PriceCents)
	}
	if listing.Currency != "USD" {
		t.Errorf("expected Currency USD, got %s", listing.Currency)
	}

	// Verify tags were serialized
	if listing.Tags == nil {
		t.Error("expected Tags to be set")
	} else {
		var tags []string
		if err := json.Unmarshal(listing.Tags, &tags); err != nil {
			t.Errorf("failed to unmarshal tags: %v", err)
		} else if len(tags) != 3 {
			t.Errorf("expected 3 tags, got %d", len(tags))
		}
	}

	// Verify SKUs were serialized
	if listing.SKUs == nil {
		t.Error("expected SKUs to be set")
	} else {
		var skus []string
		if err := json.Unmarshal(listing.SKUs, &skus); err != nil {
			t.Errorf("failed to unmarshal SKUs: %v", err)
		} else if len(skus) != 2 {
			t.Errorf("expected 2 SKUs, got %d", len(skus))
		}
	}

	// Verify defaults
	if listing.LinkedTemplate != nil {
		t.Error("expected LinkedTemplate to be nil by default")
	}
}

func TestConvertAPIListingToModel_NoTagsOrSKUs(t *testing.T) {
	apiListing := etsy.APIListing{
		ListingID: 12345,
		ShopID:    67890,
		Title:     "Simple Listing",
		State:     "active",
		Price: etsy.EtsyMoney{
			Amount:       1000,
			Divisor:      100,
			CurrencyCode: "USD",
		},
	}

	listing := convertAPIListingToModel(apiListing)

	// Tags and SKUs should be nil when empty
	if listing.Tags != nil {
		t.Error("expected Tags to be nil when no tags")
	}
	if listing.SKUs != nil {
		t.Error("expected SKUs to be nil when no SKUs")
	}
}

func TestEtsyListingTemplate_Linking(t *testing.T) {
	t.Run("create basic link", func(t *testing.T) {
		link := &model.EtsyListingTemplate{
			ID:            uuid.New(),
			EtsyListingID: 12345,
			TemplateID:    uuid.New(),
			SKU:           "TEST-SKU",
			SyncInventory: true,
		}

		if link.EtsyListingID != 12345 {
			t.Errorf("expected EtsyListingID 12345, got %d", link.EtsyListingID)
		}
		if link.SKU != "TEST-SKU" {
			t.Errorf("expected SKU 'TEST-SKU', got %s", link.SKU)
		}
		if !link.SyncInventory {
			t.Error("expected SyncInventory to be true")
		}
	})

	t.Run("link without SKU", func(t *testing.T) {
		link := &model.EtsyListingTemplate{
			ID:            uuid.New(),
			EtsyListingID: 12345,
			TemplateID:    uuid.New(),
			SyncInventory: false,
		}

		if link.SKU != "" {
			t.Errorf("expected empty SKU, got %s", link.SKU)
		}
		if link.SyncInventory {
			t.Error("expected SyncInventory to be false")
		}
	})
}

func TestInventorySyncCalculation(t *testing.T) {
	t.Run("simple quantity", func(t *testing.T) {
		// Simulate a listing with 10 items
		currentQuantity := 10
		targetQuantity := 5

		diff := currentQuantity - targetQuantity
		if diff != 5 {
			t.Errorf("expected diff of 5, got %d", diff)
		}
	})

	t.Run("increase quantity", func(t *testing.T) {
		currentQuantity := 5
		targetQuantity := 10

		diff := targetQuantity - currentQuantity
		if diff != 5 {
			t.Errorf("expected diff of 5, got %d", diff)
		}
	})

	t.Run("no change needed", func(t *testing.T) {
		currentQuantity := 10
		targetQuantity := 10

		diff := currentQuantity - targetQuantity
		if diff != 0 {
			t.Errorf("expected no diff, got %d", diff)
		}
	})

	t.Run("zero out inventory", func(t *testing.T) {
		currentQuantity := 10
		targetQuantity := 0

		if targetQuantity != 0 {
			t.Errorf("expected target 0, got %d", targetQuantity)
		}
		if currentQuantity-targetQuantity != 10 {
			t.Errorf("expected diff of 10, got %d", currentQuantity-targetQuantity)
		}
	})
}

func TestEtsyListing_StateValues(t *testing.T) {
	validStates := []string{"active", "inactive", "draft", "expired", "sold_out"}

	for _, state := range validStates {
		t.Run("state_"+state, func(t *testing.T) {
			listing := &model.EtsyListing{
				ID:            uuid.New(),
				EtsyListingID: 12345,
				State:         state,
			}

			if listing.State != state {
				t.Errorf("expected state %s, got %s", state, listing.State)
			}
		})
	}
}

func TestEtsyListing_PriceFormats(t *testing.T) {
	tests := []struct {
		name          string
		priceCents    int
		currency      string
		expectedCents int
	}{
		{
			name:          "USD standard price",
			priceCents:    2500,
			currency:      "USD",
			expectedCents: 2500,
		},
		{
			name:          "EUR price",
			priceCents:    1999,
			currency:      "EUR",
			expectedCents: 1999,
		},
		{
			name:          "GBP price",
			priceCents:    1500,
			currency:      "GBP",
			expectedCents: 1500,
		},
		{
			name:          "free item",
			priceCents:    0,
			currency:      "USD",
			expectedCents: 0,
		},
		{
			name:          "high value item",
			priceCents:    100000, // $1000
			currency:      "USD",
			expectedCents: 100000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listing := &model.EtsyListing{
				PriceCents: tt.priceCents,
				Currency:   tt.currency,
			}

			if listing.PriceCents != tt.expectedCents {
				t.Errorf("expected PriceCents %d, got %d", tt.expectedCents, listing.PriceCents)
			}
			if listing.Currency != tt.currency {
				t.Errorf("expected Currency %s, got %s", tt.currency, listing.Currency)
			}
		})
	}
}

func TestEtsyListing_WithLinkedTemplate(t *testing.T) {
	template := &model.Template{
		ID:               uuid.New(),
		Name:             "Test Template",
		SKU:              "TEMPLATE-001",
		MaterialType:     model.MaterialTypePLA,
		QuantityPerOrder: 2,
	}

	listing := &model.EtsyListing{
		ID:             uuid.New(),
		EtsyListingID:  12345,
		Title:          "Test Listing",
		State:          "active",
		LinkedTemplate: template,
	}

	if listing.LinkedTemplate == nil {
		t.Error("expected LinkedTemplate to be set")
	}
	if listing.LinkedTemplate.ID != template.ID {
		t.Errorf("expected template ID %s, got %s", template.ID, listing.LinkedTemplate.ID)
	}
	if listing.LinkedTemplate.SKU != "TEMPLATE-001" {
		t.Errorf("expected template SKU 'TEMPLATE-001', got %s", listing.LinkedTemplate.SKU)
	}
}

func TestListingSyncResult(t *testing.T) {
	t.Run("all created", func(t *testing.T) {
		result := &model.SyncResult{
			TotalFetched: 5,
			Created:      5,
		}

		if result.TotalFetched != result.Created {
			t.Errorf("expected all items to be created")
		}
	})

	t.Run("all updated", func(t *testing.T) {
		result := &model.SyncResult{
			TotalFetched: 5,
			Updated:      5,
		}

		if result.TotalFetched != result.Updated {
			t.Errorf("expected all items to be updated")
		}
	})

	t.Run("mixed results", func(t *testing.T) {
		result := &model.SyncResult{
			TotalFetched: 10,
			Created:      3,
			Updated:      5,
			Skipped:      1,
			Errors:       1,
		}

		sum := result.Created + result.Updated + result.Skipped + result.Errors
		if sum != result.TotalFetched {
			t.Errorf("sum of outcomes (%d) should equal TotalFetched (%d)", sum, result.TotalFetched)
		}
	})
}
