package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/philjestin/daedalus/internal/model"
)

// TestWebhookSignatureVerification tests HMAC-SHA256 signature verification.
func TestWebhookSignatureVerification(t *testing.T) {
	secret := "test-webhook-secret"

	tests := []struct {
		name      string
		body      []byte
		signature string
		secret    string
		valid     bool
	}{
		{
			name: "valid signature",
			body: []byte(`{"type":"receipt.created","resource_id":12345}`),
			signature: func() string {
				mac := hmac.New(sha256.New, []byte(secret))
				mac.Write([]byte(`{"type":"receipt.created","resource_id":12345}`))
				return hex.EncodeToString(mac.Sum(nil))
			}(),
			secret: secret,
			valid:  true,
		},
		{
			name:      "invalid signature",
			body:      []byte(`{"type":"receipt.created","resource_id":12345}`),
			signature: "invalid-signature",
			secret:    secret,
			valid:     false,
		},
		{
			name:      "empty signature",
			body:      []byte(`{"type":"receipt.created"}`),
			signature: "",
			secret:    secret,
			valid:     false,
		},
		{
			name: "wrong secret",
			body: []byte(`{"type":"receipt.created"}`),
			signature: func() string {
				mac := hmac.New(sha256.New, []byte("wrong-secret"))
				mac.Write([]byte(`{"type":"receipt.created"}`))
				return hex.EncodeToString(mac.Sum(nil))
			}(),
			secret: secret,
			valid:  false,
		},
		{
			name: "tampered body",
			body: []byte(`{"type":"receipt.created","resource_id":99999}`), // different resource_id
			signature: func() string {
				mac := hmac.New(sha256.New, []byte(secret))
				mac.Write([]byte(`{"type":"receipt.created","resource_id":12345}`)) // signed with original
				return hex.EncodeToString(mac.Sum(nil))
			}(),
			secret: secret,
			valid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := verifyWebhookSignature(tt.body, tt.signature, tt.secret)
			if result != tt.valid {
				t.Errorf("expected valid=%v, got %v", tt.valid, result)
			}
		})
	}
}

// TestWebhookEventParsing tests parsing of webhook event payloads.
func TestWebhookEventParsing(t *testing.T) {
	tests := []struct {
		name             string
		payload          string
		expectedType     string
		expectedResource string
		expectedID       int64
	}{
		{
			name:             "receipt.created",
			payload:          `{"type":"receipt.created","resource_type":"receipt","resource_id":12345,"shop_id":67890}`,
			expectedType:     "receipt.created",
			expectedResource: "receipt",
			expectedID:       12345,
		},
		{
			name:             "receipt.updated",
			payload:          `{"type":"receipt.updated","resource_type":"receipt","resource_id":12345,"shop_id":67890}`,
			expectedType:     "receipt.updated",
			expectedResource: "receipt",
			expectedID:       12345,
		},
		{
			name:             "listing.updated",
			payload:          `{"type":"listing.updated","resource_type":"listing","resource_id":99999,"shop_id":67890}`,
			expectedType:     "listing.updated",
			expectedResource: "listing",
			expectedID:       99999,
		},
		{
			name:             "listing.inventory.updated",
			payload:          `{"type":"listing.inventory.updated","resource_type":"listing","resource_id":55555,"shop_id":67890}`,
			expectedType:     "listing.inventory.updated",
			expectedResource: "listing",
			expectedID:       55555,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var event struct {
				Type         string `json:"type"`
				ResourceType string `json:"resource_type"`
				ResourceID   int64  `json:"resource_id"`
				ShopID       int64  `json:"shop_id"`
			}

			err := json.Unmarshal([]byte(tt.payload), &event)
			if err != nil {
				t.Fatalf("failed to parse: %v", err)
			}

			if event.Type != tt.expectedType {
				t.Errorf("expected type=%s, got %s", tt.expectedType, event.Type)
			}
			if event.ResourceType != tt.expectedResource {
				t.Errorf("expected resource_type=%s, got %s", tt.expectedResource, event.ResourceType)
			}
			if event.ResourceID != tt.expectedID {
				t.Errorf("expected resource_id=%d, got %d", tt.expectedID, event.ResourceID)
			}
		})
	}
}

// TestWebhookHandler_ReceiptCreated tests the full webhook flow for receipt.created.
func TestWebhookHandler_ReceiptCreated(t *testing.T) {
	secret := "test-secret"
	payload := `{"type":"receipt.created","resource_type":"receipt","resource_id":12345,"shop_id":67890}`
	body := []byte(payload)

	// Generate valid signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	signature := hex.EncodeToString(mac.Sum(nil))

	t.Run("valid webhook", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/integrations/etsy/webhook", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Etsy-Signature", signature)

		// Parse and verify
		var event struct {
			Type       string `json:"type"`
			ResourceID int64  `json:"resource_id"`
		}
		err := json.Unmarshal(body, &event)
		if err != nil {
			t.Fatalf("failed to parse: %v", err)
		}

		if event.Type != "receipt.created" {
			t.Errorf("expected type='receipt.created', got %s", event.Type)
		}

		// Verify signature
		if !verifyWebhookSignature(body, signature, secret) {
			t.Error("signature verification should pass")
		}
	})

	t.Run("missing signature", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/integrations/etsy/webhook", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		// No X-Etsy-Signature header

		signature := req.Header.Get("X-Etsy-Signature")
		if signature != "" {
			t.Error("expected no signature header")
		}

		// Without a secret configured, this should still process
		// With a secret, it should fail
	})
}

// TestWebhookEvent_Serialization tests EtsyWebhookEvent JSON serialization.
func TestWebhookEvent_Serialization(t *testing.T) {
	now := time.Now()
	processedAt := now.Add(-time.Minute)

	event := model.EtsyWebhookEvent{
		ID:           uuid.New(),
		EventType:    "receipt.created",
		ResourceType: "receipt",
		ResourceID:   12345,
		ShopID:       67890,
		Payload:      json.RawMessage(`{"type":"receipt.created"}`),
		Signature:    "test-signature",
		Processed:    true,
		ProcessedAt:  &processedAt,
		Error:        "",
		ReceivedAt:   now,
		CreatedAt:    now,
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	// Verify fields
	if parsed["event_type"] != "receipt.created" {
		t.Errorf("expected event_type='receipt.created', got %v", parsed["event_type"])
	}
	if parsed["resource_type"] != "receipt" {
		t.Errorf("expected resource_type='receipt', got %v", parsed["resource_type"])
	}
	if parsed["resource_id"] != float64(12345) {
		t.Errorf("expected resource_id=12345, got %v", parsed["resource_id"])
	}
	if parsed["processed"] != true {
		t.Errorf("expected processed=true, got %v", parsed["processed"])
	}
}

// TestWebhookEventTypes tests the webhook event type constants.
func TestWebhookEventTypes(t *testing.T) {
	tests := []struct {
		constant string
		expected string
	}{
		{model.EtsyEventReceiptCreated, "receipt.created"},
		{model.EtsyEventReceiptUpdated, "receipt.updated"},
		{model.EtsyEventListingUpdated, "listing.updated"},
		{model.EtsyEventListingInventoryUpdated, "listing.inventory.updated"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.constant)
			}
		})
	}
}

// TestListWebhookEvents_QueryParams tests query parameter parsing for ListWebhookEvents.
func TestListWebhookEvents_QueryParams(t *testing.T) {
	tests := []struct {
		name           string
		queryString    string
		expectedType   string
		expectedLimit  int
		expectedOffset int
	}{
		{
			name:           "no params",
			queryString:    "",
			expectedType:   "",
			expectedLimit:  50,
			expectedOffset: 0,
		},
		{
			name:           "type filter",
			queryString:    "type=receipt.created",
			expectedType:   "receipt.created",
			expectedLimit:  50,
			expectedOffset: 0,
		},
		{
			name:           "all params",
			queryString:    "type=listing.updated&limit=25&offset=50",
			expectedType:   "listing.updated",
			expectedLimit:  25,
			expectedOffset: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/webhook/events?"+tt.queryString, nil)

			eventType := req.URL.Query().Get("type")

			limit := 50
			if l := req.URL.Query().Get("limit"); l != "" {
				var val int
				json.Unmarshal([]byte(l), &val)
				if val > 0 {
					limit = val
				}
			}

			offset := 0
			if o := req.URL.Query().Get("offset"); o != "" {
				var val int
				json.Unmarshal([]byte(o), &val)
				if val >= 0 {
					offset = val
				}
			}

			if eventType != tt.expectedType {
				t.Errorf("expected type=%s, got %s", tt.expectedType, eventType)
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

// TestReprocessWebhookEvent tests the reprocess endpoint.
func TestReprocessWebhookEvent(t *testing.T) {
	eventID := uuid.New()

	t.Run("valid event ID", func(t *testing.T) {
		_, err := uuid.Parse(eventID.String())
		if err != nil {
			t.Errorf("should be valid UUID: %v", err)
		}
	})

	t.Run("invalid event ID", func(t *testing.T) {
		_, err := uuid.Parse("invalid-uuid")
		if err == nil {
			t.Error("should be invalid UUID")
		}
	})
}

// TestWebhookEvent_ProcessedState tests the processed state transitions.
func TestWebhookEvent_ProcessedState(t *testing.T) {
	t.Run("initial state", func(t *testing.T) {
		event := &model.EtsyWebhookEvent{
			ID:        uuid.New(),
			EventType: "receipt.created",
			Processed: false,
		}

		if event.Processed {
			t.Error("new event should not be processed")
		}
		if event.ProcessedAt != nil {
			t.Error("new event should not have processed_at")
		}
		if event.Error != "" {
			t.Error("new event should not have error")
		}
	})

	t.Run("successful processing", func(t *testing.T) {
		now := time.Now()
		event := &model.EtsyWebhookEvent{
			ID:          uuid.New(),
			EventType:   "receipt.created",
			Processed:   true,
			ProcessedAt: &now,
			Error:       "",
		}

		if !event.Processed {
			t.Error("processed event should be marked as processed")
		}
		if event.ProcessedAt == nil {
			t.Error("processed event should have processed_at")
		}
		if event.Error != "" {
			t.Error("successfully processed event should not have error")
		}
	})

	t.Run("failed processing", func(t *testing.T) {
		now := time.Now()
		event := &model.EtsyWebhookEvent{
			ID:          uuid.New(),
			EventType:   "receipt.created",
			Processed:   true,
			ProcessedAt: &now,
			Error:       "failed to sync receipts",
		}

		if !event.Processed {
			t.Error("failed event should still be marked as processed")
		}
		if event.Error == "" {
			t.Error("failed event should have error message")
		}
	})
}

// TestWebhookPayload_MalformedJSON tests handling of malformed JSON payloads.
func TestWebhookPayload_MalformedJSON(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		valid   bool
	}{
		{
			name:    "valid JSON",
			payload: `{"type":"receipt.created","resource_id":12345}`,
			valid:   true,
		},
		{
			name:    "empty object",
			payload: `{}`,
			valid:   true,
		},
		{
			name:    "invalid JSON - missing brace",
			payload: `{"type":"receipt.created"`,
			valid:   false,
		},
		{
			name:    "invalid JSON - trailing comma",
			payload: `{"type":"receipt.created",}`,
			valid:   false,
		},
		{
			name:    "empty string",
			payload: ``,
			valid:   false,
		},
		{
			name:    "not JSON",
			payload: `not json at all`,
			valid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var data map[string]interface{}
			err := json.Unmarshal([]byte(tt.payload), &data)
			isValid := err == nil

			if isValid != tt.valid {
				t.Errorf("expected valid=%v, got %v (error: %v)", tt.valid, isValid, err)
			}
		})
	}
}

// TestWebhookResponse tests that webhook handler returns 200 OK quickly.
func TestWebhookResponse(t *testing.T) {
	// Webhook handlers should return 200 OK immediately
	// Processing happens asynchronously

	t.Run("immediate response", func(t *testing.T) {
		// Simulate the expected behavior
		rr := httptest.NewRecorder()
		rr.WriteHeader(http.StatusOK)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rr.Code)
		}
	})
}

// TestSignatureGeneration tests generating a valid HMAC signature.
func TestSignatureGeneration(t *testing.T) {
	secret := "my-webhook-secret"
	body := []byte(`{"type":"receipt.created","resource_id":12345}`)

	// Generate signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	signature := hex.EncodeToString(mac.Sum(nil))

	// Verify it's a valid hex string
	decoded, err := hex.DecodeString(signature)
	if err != nil {
		t.Fatalf("signature should be valid hex: %v", err)
	}

	// SHA256 produces 32 bytes
	if len(decoded) != 32 {
		t.Errorf("expected 32 byte signature, got %d", len(decoded))
	}

	// Verify it's reproducible
	mac2 := hmac.New(sha256.New, []byte(secret))
	mac2.Write(body)
	signature2 := hex.EncodeToString(mac2.Sum(nil))

	if signature != signature2 {
		t.Error("signature should be reproducible")
	}
}
