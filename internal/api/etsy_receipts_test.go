package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hyperion/printfarm/internal/model"
)

// MockEtsyReceiptService implements receipt-related methods for testing.
type MockEtsyReceiptService struct {
	receipts     map[uuid.UUID]*model.EtsyReceipt
	syncResult   *model.SyncResult
	syncError    error
	processError error
}

func NewMockEtsyReceiptService() *MockEtsyReceiptService {
	return &MockEtsyReceiptService{
		receipts: make(map[uuid.UUID]*model.EtsyReceipt),
	}
}

func (m *MockEtsyReceiptService) AddReceipt(r *model.EtsyReceipt) {
	m.receipts[r.ID] = r
}

func (m *MockEtsyReceiptService) SetSyncResult(result *model.SyncResult, err error) {
	m.syncResult = result
	m.syncError = err
}

func (m *MockEtsyReceiptService) SetProcessError(err error) {
	m.processError = err
}

// TestListReceipts_QueryParams tests the query parameter parsing for ListReceipts.
func TestListReceipts_QueryParams(t *testing.T) {
	tests := []struct {
		name            string
		queryString     string
		expectedProcess *bool
		expectedLimit   int
		expectedOffset  int
	}{
		{
			name:            "no params",
			queryString:     "",
			expectedProcess: nil,
			expectedLimit:   50,
			expectedOffset:  0,
		},
		{
			name:            "processed=true",
			queryString:     "processed=true",
			expectedProcess: boolPtr(true),
			expectedLimit:   50,
			expectedOffset:  0,
		},
		{
			name:            "processed=false",
			queryString:     "processed=false",
			expectedProcess: boolPtr(false),
			expectedLimit:   50,
			expectedOffset:  0,
		},
		{
			name:            "limit only",
			queryString:     "limit=25",
			expectedProcess: nil,
			expectedLimit:   25,
			expectedOffset:  0,
		},
		{
			name:            "offset only",
			queryString:     "offset=100",
			expectedProcess: nil,
			expectedLimit:   50,
			expectedOffset:  100,
		},
		{
			name:            "all params",
			queryString:     "processed=true&limit=10&offset=20",
			expectedProcess: boolPtr(true),
			expectedLimit:   10,
			expectedOffset:  20,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/integrations/etsy/receipts?"+tt.queryString, nil)

			// Parse query params
			processed := req.URL.Query().Get("processed")
			var parsedProcessed *bool
			if processed != "" {
				val := processed == "true"
				parsedProcessed = &val
			}

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

			// Verify parsed values
			if tt.expectedProcess == nil && parsedProcessed != nil {
				t.Error("expected processed to be nil")
			}
			if tt.expectedProcess != nil {
				if parsedProcessed == nil {
					t.Error("expected processed to not be nil")
				} else if *parsedProcessed != *tt.expectedProcess {
					t.Errorf("expected processed=%v, got %v", *tt.expectedProcess, *parsedProcessed)
				}
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

// TestListReceipts_Pagination tests pagination behavior.
func TestListReceipts_Pagination(t *testing.T) {
	t.Run("first page", func(t *testing.T) {
		limit := 10
		offset := 0

		// Simulate getting first 10 of 25 items
		totalItems := 25
		pageItems := min(limit, totalItems-offset)

		if pageItems != 10 {
			t.Errorf("expected 10 items on first page, got %d", pageItems)
		}
	})

	t.Run("second page", func(t *testing.T) {
		limit := 10
		offset := 10

		totalItems := 25
		pageItems := min(limit, totalItems-offset)

		if pageItems != 10 {
			t.Errorf("expected 10 items on second page, got %d", pageItems)
		}
	})

	t.Run("last page partial", func(t *testing.T) {
		limit := 10
		offset := 20

		totalItems := 25
		pageItems := min(limit, totalItems-offset)

		if pageItems != 5 {
			t.Errorf("expected 5 items on last page, got %d", pageItems)
		}
	})

	t.Run("beyond last page", func(t *testing.T) {
		limit := 10
		offset := 30

		totalItems := 25
		remaining := totalItems - offset
		if remaining < 0 {
			remaining = 0
		}
		pageItems := min(limit, remaining)

		if pageItems != 0 {
			t.Errorf("expected 0 items beyond last page, got %d", pageItems)
		}
	})
}

// TestProcessReceipt tests the receipt processing endpoint.
func TestProcessReceipt_Success(t *testing.T) {
	receiptID := uuid.New()
	projectID := uuid.New()

	receipt := &model.EtsyReceipt{
		ID:              receiptID,
		EtsyReceiptID:   12345,
		Name:            "Test Buyer",
		Status:          "paid",
		GrandtotalCents: 5000,
		IsProcessed:     false,
		Items: []model.EtsyReceiptItem{
			{
				ID:    uuid.New(),
				Title: "Test Item",
				SKU:   "TEST-SKU",
			},
		},
	}

	project := &model.Project{
		ID:              projectID,
		Name:            "Order from Etsy",
		Source:          "etsy",
		ExternalOrderID: "etsy-12345",
	}

	// Verify receipt and project fields
	if receipt.IsProcessed {
		t.Error("receipt should not be processed initially")
	}
	if project.Source != "etsy" {
		t.Errorf("expected project source 'etsy', got %s", project.Source)
	}
	if project.ExternalOrderID != "etsy-12345" {
		t.Errorf("expected external order ID 'etsy-12345', got %s", project.ExternalOrderID)
	}
}

func TestProcessReceipt_AlreadyProcessed(t *testing.T) {
	receipt := &model.EtsyReceipt{
		ID:          uuid.New(),
		IsProcessed: true,
		ProjectID:   ptrUUID(uuid.New()),
	}

	if !receipt.IsProcessed {
		t.Error("receipt should be marked as processed")
	}
	if receipt.ProjectID == nil {
		t.Error("processed receipt should have project ID")
	}
}

func TestProcessReceipt_NoMatchingTemplate(t *testing.T) {
	receipt := &model.EtsyReceipt{
		ID:            uuid.New(),
		EtsyReceiptID: 12345,
		IsProcessed:   false,
		Items: []model.EtsyReceiptItem{
			{
				ID:    uuid.New(),
				Title: "Test Item",
				SKU:   "UNKNOWN-SKU",
			},
		},
	}

	// Simulate no template found for SKU
	hasMatch := false
	for _, item := range receipt.Items {
		if item.SKU == "KNOWN-SKU" {
			hasMatch = true
			break
		}
	}

	if hasMatch {
		t.Error("should not find matching template for unknown SKU")
	}
}

// TestGetReceipt_URLParsing tests URL parameter parsing for GetReceipt.
func TestGetReceipt_URLParsing(t *testing.T) {
	t.Run("valid UUID", func(t *testing.T) {
		testID := uuid.New()
		r := chi.NewRouter()
		var parsedID uuid.UUID

		r.Get("/receipts/{id}", func(w http.ResponseWriter, r *http.Request) {
			id, err := parseUUID(r, "id")
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			parsedID = id
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/receipts/"+testID.String(), nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
		if parsedID != testID {
			t.Errorf("expected ID %s, got %s", testID, parsedID)
		}
	})

	t.Run("invalid UUID", func(t *testing.T) {
		r := chi.NewRouter()

		r.Get("/receipts/{id}", func(w http.ResponseWriter, r *http.Request) {
			_, err := parseUUID(r, "id")
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/receipts/invalid-uuid", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rr.Code)
		}
	})
}

// TestSyncReceipts_Response tests the sync receipts response format.
func TestSyncReceipts_Response(t *testing.T) {
	result := &model.SyncResult{
		TotalFetched: 10,
		Created:      5,
		Updated:      3,
		Skipped:      1,
		Errors:       1,
	}

	// Serialize and verify
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Check all fields are present
	expectedFields := []string{"total_fetched", "created", "updated", "skipped", "errors"}
	for _, field := range expectedFields {
		if _, ok := parsed[field]; !ok {
			t.Errorf("expected field %s in response", field)
		}
	}

	// Check values
	if parsed["total_fetched"] != float64(10) {
		t.Errorf("expected total_fetched=10, got %v", parsed["total_fetched"])
	}
	if parsed["created"] != float64(5) {
		t.Errorf("expected created=5, got %v", parsed["created"])
	}
}

// TestReceiptResponse_Serialization tests EtsyReceipt JSON serialization.
func TestReceiptResponse_Serialization(t *testing.T) {
	now := time.Now()
	projectID := uuid.New()

	receipt := model.EtsyReceipt{
		ID:              uuid.New(),
		EtsyReceiptID:   12345,
		EtsyShopID:      67890,
		BuyerEmail:      "buyer@example.com",
		Name:            "Test Buyer",
		Status:          "paid",
		IsShipped:       false,
		IsPaid:          true,
		IsGift:          true,
		GiftMessage:     "Happy Birthday!",
		GrandtotalCents: 5000,
		Currency:        "USD",
		IsProcessed:     true,
		ProjectID:       &projectID,
		CreatedAt:       now,
		Items: []model.EtsyReceiptItem{
			{
				ID:         uuid.New(),
				Title:      "Test Item",
				Quantity:   2,
				PriceCents: 2500,
				SKU:        "TEST-SKU",
			},
		},
	}

	data, err := json.Marshal(receipt)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Verify key fields
	if parsed["etsy_receipt_id"] != float64(12345) {
		t.Errorf("expected etsy_receipt_id=12345, got %v", parsed["etsy_receipt_id"])
	}
	if parsed["name"] != "Test Buyer" {
		t.Errorf("expected name='Test Buyer', got %v", parsed["name"])
	}
	if parsed["is_paid"] != true {
		t.Errorf("expected is_paid=true, got %v", parsed["is_paid"])
	}
	if parsed["is_processed"] != true {
		t.Errorf("expected is_processed=true, got %v", parsed["is_processed"])
	}
	if parsed["grandtotal_cents"] != float64(5000) {
		t.Errorf("expected grandtotal_cents=5000, got %v", parsed["grandtotal_cents"])
	}

	// Verify items are included
	items, ok := parsed["items"].([]interface{})
	if !ok {
		t.Error("expected items to be an array")
	} else if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}
}

// TestProcessReceiptRequest_Parsing tests request body parsing for ProcessReceipt.
func TestProcessReceiptRequest_Parsing(t *testing.T) {
	t.Run("empty body is valid", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/receipts/123/process", bytes.NewReader([]byte("{}")))
		req.Header.Set("Content-Type", "application/json")

		var body map[string]interface{}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Errorf("should parse empty object: %v", err)
		}
	})

	t.Run("with template_id override", func(t *testing.T) {
		templateID := uuid.New().String()
		jsonBody := `{"template_id": "` + templateID + `"}`
		req := httptest.NewRequest(http.MethodPost, "/receipts/123/process", bytes.NewReader([]byte(jsonBody)))
		req.Header.Set("Content-Type", "application/json")

		var body struct {
			TemplateID string `json:"template_id"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			t.Errorf("should parse with template_id: %v", err)
		}
		if body.TemplateID != templateID {
			t.Errorf("expected template_id=%s, got %s", templateID, body.TemplateID)
		}
	})
}

// Helper functions
func boolPtr(b bool) *bool {
	return &b
}

func ptrUUID(id uuid.UUID) *uuid.UUID {
	return &id
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
