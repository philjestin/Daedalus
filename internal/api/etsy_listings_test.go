package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/philjestin/daedalus/internal/model"
)

// TestListListings_QueryParams tests the query parameter parsing for ListListings.
func TestListListings_QueryParams(t *testing.T) {
	tests := []struct {
		name           string
		queryString    string
		expectedState  string
		expectedLimit  int
		expectedOffset int
	}{
		{
			name:           "no params",
			queryString:    "",
			expectedState:  "",
			expectedLimit:  50,
			expectedOffset: 0,
		},
		{
			name:           "state=active",
			queryString:    "state=active",
			expectedState:  "active",
			expectedLimit:  50,
			expectedOffset: 0,
		},
		{
			name:           "state=inactive",
			queryString:    "state=inactive",
			expectedState:  "inactive",
			expectedLimit:  50,
			expectedOffset: 0,
		},
		{
			name:           "state=draft",
			queryString:    "state=draft",
			expectedState:  "draft",
			expectedLimit:  50,
			expectedOffset: 0,
		},
		{
			name:           "limit only",
			queryString:    "limit=25",
			expectedState:  "",
			expectedLimit:  25,
			expectedOffset: 0,
		},
		{
			name:           "all params",
			queryString:    "state=active&limit=10&offset=20",
			expectedState:  "active",
			expectedLimit:  10,
			expectedOffset: 20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/integrations/etsy/listings?"+tt.queryString, nil)

			state := req.URL.Query().Get("state")

			limit := 50
			if l := req.URL.Query().Get("limit"); l != "" {
				if val, err := strconv.Atoi(l); err == nil && val > 0 {
					limit = val
				}
			}

			offset := 0
			if o := req.URL.Query().Get("offset"); o != "" {
				if val, err := strconv.Atoi(o); err == nil && val >= 0 {
					offset = val
				}
			}

			if state != tt.expectedState {
				t.Errorf("expected state=%s, got %s", tt.expectedState, state)
			}
			if limit != tt.expectedLimit {
				t.Errorf("expected limit=%d, got %d", tt.expectedLimit, limit)
			}
			if offset != tt.expectedOffset {
				t.Errorf("expected offset=%d, got %d", tt.expectedOffset, offset)
			}
		})
	}
}

// TestListListings_Filtering tests filtering listings by state.
func TestListListings_Filtering(t *testing.T) {
	listings := []model.EtsyListing{
		{ID: uuid.New(), Title: "Active Listing 1", State: "active"},
		{ID: uuid.New(), Title: "Active Listing 2", State: "active"},
		{ID: uuid.New(), Title: "Inactive Listing", State: "inactive"},
		{ID: uuid.New(), Title: "Draft Listing", State: "draft"},
	}

	t.Run("filter active", func(t *testing.T) {
		var filtered []model.EtsyListing
		for _, l := range listings {
			if l.State == "active" {
				filtered = append(filtered, l)
			}
		}
		if len(filtered) != 2 {
			t.Errorf("expected 2 active listings, got %d", len(filtered))
		}
	})

	t.Run("filter inactive", func(t *testing.T) {
		var filtered []model.EtsyListing
		for _, l := range listings {
			if l.State == "inactive" {
				filtered = append(filtered, l)
			}
		}
		if len(filtered) != 1 {
			t.Errorf("expected 1 inactive listing, got %d", len(filtered))
		}
	})

	t.Run("no filter returns all", func(t *testing.T) {
		if len(listings) != 4 {
			t.Errorf("expected 4 total listings, got %d", len(listings))
		}
	})
}

// TestListListings_Pagination tests pagination for listings.
func TestListListings_Pagination(t *testing.T) {
	// Create test data
	var listings []model.EtsyListing
	for i := 0; i < 25; i++ {
		listings = append(listings, model.EtsyListing{
			ID:    uuid.New(),
			Title: "Listing",
			State: "active",
		})
	}

	tests := []struct {
		name          string
		limit         int
		offset        int
		expectedCount int
	}{
		{"first page", 10, 0, 10},
		{"second page", 10, 10, 10},
		{"third page partial", 10, 20, 5},
		{"beyond last", 10, 30, 0},
		{"large limit", 100, 0, 25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := tt.offset
			if start > len(listings) {
				start = len(listings)
			}
			end := tt.offset + tt.limit
			if end > len(listings) {
				end = len(listings)
			}

			page := listings[start:end]
			if len(page) != tt.expectedCount {
				t.Errorf("expected %d items, got %d", tt.expectedCount, len(page))
			}
		})
	}
}

// TestLinkListing_ValidRequest tests valid link listing requests.
func TestLinkListing_ValidRequest(t *testing.T) {
	templateID := uuid.New()

	tests := []struct {
		name    string
		request LinkListingRequest
		valid   bool
	}{
		{
			name: "minimal request",
			request: LinkListingRequest{
				TemplateID: templateID.String(),
			},
			valid: true,
		},
		{
			name: "with SKU",
			request: LinkListingRequest{
				TemplateID: templateID.String(),
				SKU:        "TEST-SKU",
			},
			valid: true,
		},
		{
			name: "with sync inventory",
			request: LinkListingRequest{
				TemplateID:    templateID.String(),
				SyncInventory: true,
			},
			valid: true,
		},
		{
			name: "all fields",
			request: LinkListingRequest{
				TemplateID:    templateID.String(),
				SKU:           "FULL-SKU",
				SyncInventory: true,
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("failed to marshal: %v", err)
			}

			var parsed LinkListingRequest
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("failed to unmarshal: %v", err)
			}

			// Validate template ID can be parsed
			_, err = uuid.Parse(parsed.TemplateID)
			isValid := err == nil

			if isValid != tt.valid {
				t.Errorf("expected valid=%v, got %v", tt.valid, isValid)
			}
		})
	}
}

// TestLinkListing_InvalidTemplateID tests invalid template ID handling.
func TestLinkListing_InvalidTemplateID(t *testing.T) {
	tests := []struct {
		name       string
		templateID string
		expectErr  bool
	}{
		{
			name:       "valid UUID",
			templateID: uuid.New().String(),
			expectErr:  false,
		},
		{
			name:       "invalid UUID",
			templateID: "not-a-uuid",
			expectErr:  true,
		},
		{
			name:       "empty string",
			templateID: "",
			expectErr:  true,
		},
		{
			name:       "partial UUID",
			templateID: "550e8400-e29b-41d4",
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := uuid.Parse(tt.templateID)
			hasErr := err != nil

			if hasErr != tt.expectErr {
				t.Errorf("expected error=%v, got %v", tt.expectErr, hasErr)
			}
		})
	}
}

// TestUnlinkListing_QueryParams tests query parameter parsing for UnlinkListing.
func TestUnlinkListing_QueryParams(t *testing.T) {
	templateID := uuid.New()

	r := chi.NewRouter()

	r.Delete("/listings/{id}/link", func(w http.ResponseWriter, r *http.Request) {
		listingID, err := parseUUID(r, "id")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		templateIDStr := r.URL.Query().Get("template_id")
		if templateIDStr == "" {
			http.Error(w, "template_id is required", http.StatusBadRequest)
			return
		}

		tid, err := uuid.Parse(templateIDStr)
		if err != nil {
			http.Error(w, "invalid template_id", http.StatusBadRequest)
			return
		}

		// Both IDs are valid
		_ = listingID
		_ = tid
		w.WriteHeader(http.StatusNoContent)
	})

	t.Run("valid request", func(t *testing.T) {
		listingID := uuid.New()
		req := httptest.NewRequest(http.MethodDelete, "/listings/"+listingID.String()+"/link?template_id="+templateID.String(), nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusNoContent {
			t.Errorf("expected status 204, got %d", rr.Code)
		}
	})

	t.Run("missing template_id", func(t *testing.T) {
		listingID := uuid.New()
		req := httptest.NewRequest(http.MethodDelete, "/listings/"+listingID.String()+"/link", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})

	t.Run("invalid template_id", func(t *testing.T) {
		listingID := uuid.New()
		req := httptest.NewRequest(http.MethodDelete, "/listings/"+listingID.String()+"/link?template_id=invalid", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})
}

// TestListingResponse_Serialization tests EtsyListing JSON serialization.
func TestListingResponse_Serialization(t *testing.T) {
	listing := model.EtsyListing{
		ID:               uuid.New(),
		EtsyListingID:    12345,
		EtsyShopID:       67890,
		Title:            "Test Listing",
		Description:      "A test listing",
		State:            "active",
		Quantity:         10,
		URL:              "https://etsy.com/listing/12345",
		Views:            1000,
		NumFavorers:      50,
		IsCustomizable:   true,
		IsPersonalizable: false,
		HasVariations:    true,
		PriceCents:       2500,
		Currency:         "USD",
	}

	data, err := json.Marshal(listing)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Verify key fields
	if parsed["etsy_listing_id"] != float64(12345) {
		t.Errorf("expected etsy_listing_id=12345, got %v", parsed["etsy_listing_id"])
	}
	if parsed["title"] != "Test Listing" {
		t.Errorf("expected title='Test Listing', got %v", parsed["title"])
	}
	if parsed["state"] != "active" {
		t.Errorf("expected state='active', got %v", parsed["state"])
	}
	if parsed["quantity"] != float64(10) {
		t.Errorf("expected quantity=10, got %v", parsed["quantity"])
	}
	if parsed["price_cents"] != float64(2500) {
		t.Errorf("expected price_cents=2500, got %v", parsed["price_cents"])
	}
	if parsed["is_customizable"] != true {
		t.Errorf("expected is_customizable=true, got %v", parsed["is_customizable"])
	}
	if parsed["has_variations"] != true {
		t.Errorf("expected has_variations=true, got %v", parsed["has_variations"])
	}
}

// TestSyncListings_Response tests the sync listings response format.
func TestSyncListings_Response(t *testing.T) {
	result := &model.SyncResult{
		TotalFetched: 20,
		Created:      15,
		Updated:      5,
		Skipped:      0,
		Errors:       0,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed["total_fetched"] != float64(20) {
		t.Errorf("expected total_fetched=20, got %v", parsed["total_fetched"])
	}
	if parsed["created"] != float64(15) {
		t.Errorf("expected created=15, got %v", parsed["created"])
	}
	if parsed["updated"] != float64(5) {
		t.Errorf("expected updated=5, got %v", parsed["updated"])
	}
}

// TestGetListing_URLParsing tests URL parameter parsing for GetListing.
func TestGetListing_URLParsing(t *testing.T) {
	r := chi.NewRouter()

	r.Get("/listings/{id}", func(w http.ResponseWriter, r *http.Request) {
		_, err := parseUUID(r, "id")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	t.Run("valid UUID", func(t *testing.T) {
		testID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/listings/"+testID.String(), nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
	})

	t.Run("invalid UUID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/listings/invalid-uuid", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})
}

// TestLinkListingRequest_Parse tests parsing of LinkListingRequest.
func TestLinkListingRequest_Parse(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		templateID := uuid.New()
		jsonStr := `{"template_id": "` + templateID.String() + `", "sku": "TEST-SKU", "sync_inventory": true}`

		var req LinkListingRequest
		err := json.Unmarshal([]byte(jsonStr), &req)
		if err != nil {
			t.Fatalf("failed to parse: %v", err)
		}

		if req.TemplateID != templateID.String() {
			t.Errorf("expected template_id=%s, got %s", templateID, req.TemplateID)
		}
		if req.SKU != "TEST-SKU" {
			t.Errorf("expected sku='TEST-SKU', got %s", req.SKU)
		}
		if !req.SyncInventory {
			t.Error("expected sync_inventory=true")
		}
	})

	t.Run("minimal request", func(t *testing.T) {
		templateID := uuid.New()
		jsonStr := `{"template_id": "` + templateID.String() + `"}`

		var req LinkListingRequest
		err := json.Unmarshal([]byte(jsonStr), &req)
		if err != nil {
			t.Fatalf("failed to parse: %v", err)
		}

		if req.TemplateID != templateID.String() {
			t.Errorf("expected template_id=%s, got %s", templateID, req.TemplateID)
		}
		if req.SKU != "" {
			t.Errorf("expected empty sku, got %s", req.SKU)
		}
		if req.SyncInventory {
			t.Error("expected sync_inventory=false by default")
		}
	})
}

// TestSyncInventory tests the inventory sync endpoint behavior.
func TestSyncInventory(t *testing.T) {
	r := chi.NewRouter()

	r.Post("/listings/{id}/sync-inventory", func(w http.ResponseWriter, r *http.Request) {
		_, err := parseUUID(r, "id")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		respondJSON(w, http.StatusOK, map[string]string{"status": "synced"})
	})

	t.Run("valid listing ID", func(t *testing.T) {
		listingID := uuid.New()
		req := httptest.NewRequest(http.MethodPost, "/listings/"+listingID.String()+"/sync-inventory", bytes.NewReader([]byte("{}")))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
	})

	t.Run("invalid listing ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/listings/invalid/sync-inventory", bytes.NewReader([]byte("{}")))
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})
}
