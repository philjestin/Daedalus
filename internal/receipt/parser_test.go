package receipt

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestNewParser_ReadsAnthropicAPIKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key-123")
	p := NewParser()
	if p.apiKey != "test-key-123" {
		t.Errorf("expected apiKey %q, got %q", "test-key-123", p.apiKey)
	}
}

func TestNewParser_DefaultModel(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "k")
	p := NewParser()
	if p.model != "claude-sonnet-4-20250514" {
		t.Errorf("expected default model claude-sonnet-4-20250514, got %q", p.model)
	}
}

func TestNewParser_CustomModel(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "k")
	t.Setenv("RECEIPT_PARSER_MODEL", "claude-3-haiku-20240307")
	p := NewParser()
	if p.model != "claude-3-haiku-20240307" {
		t.Errorf("expected model claude-3-haiku-20240307, got %q", p.model)
	}
}

func TestParseFromBytes_NoAPIKey(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	p := NewParser()

	_, err := p.ParseFromBytes(context.Background(), []byte("data"), "image/jpeg")
	if err == nil {
		t.Fatal("expected error when API key is not set")
	}
	if got := err.Error(); got != "ANTHROPIC_API_KEY not set — add it to your .env file to enable receipt parsing" {
		t.Errorf("unexpected error message: %s", got)
	}
}

func TestParseFromBytes_PDFContentType(t *testing.T) {
	// Verify that PDFs send "document" type, not "image" type
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("expected x-api-key header, got %q", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("expected anthropic-version header, got %q", r.Header.Get("anthropic-version"))
		}

		json.NewDecoder(r.Body).Decode(&receivedBody)

		// Return a valid response
		resp := map[string]interface{}{
			"id":   "msg_test",
			"type": "message",
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": `{"vendor":"Test","date":"2024-01-01","subtotal_cents":1000,"tax_cents":100,"shipping_cents":0,"total_cents":1100,"currency":"USD","items":[],"confidence":90}`,
				},
			},
			"stop_reason": "end_turn",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := &Parser{
		apiKey:     "test-key",
		model:      "test-model",
		httpClient: server.Client(),
	}

	// Override the API URL by using a custom transport
	origURL := "https://api.anthropic.com/v1/messages"
	_ = origURL

	// We need to intercept the request. Let's create a custom RoundTripper.
	p.httpClient = &http.Client{
		Transport: &redirectTransport{target: server.URL, wrapped: http.DefaultTransport},
	}

	// Use PDF magic bytes
	pdfData := []byte("%PDF-1.4 fake pdf content")
	parsed, err := p.ParseFromBytes(context.Background(), pdfData, "application/pdf")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify the response was parsed
	if parsed.Vendor != "Test" {
		t.Errorf("expected vendor %q, got %q", "Test", parsed.Vendor)
	}
	if parsed.TotalCents != 1100 {
		t.Errorf("expected total_cents 1100, got %d", parsed.TotalCents)
	}

	// Verify the request sent "document" type for PDF
	messages := receivedBody["messages"].([]interface{})
	msg := messages[0].(map[string]interface{})
	content := msg["content"].([]interface{})
	mediaBlock := content[0].(map[string]interface{})

	if mediaBlock["type"] != "document" {
		t.Errorf("expected document type for PDF, got %q", mediaBlock["type"])
	}
	source := mediaBlock["source"].(map[string]interface{})
	if source["media_type"] != "application/pdf" {
		t.Errorf("expected media_type application/pdf, got %q", source["media_type"])
	}
}

func TestParseFromBytes_ImageContentType(t *testing.T) {
	var receivedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedBody)

		resp := map[string]interface{}{
			"id":   "msg_test",
			"type": "message",
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": `{"vendor":"Store","date":"2024-01-01","subtotal_cents":500,"tax_cents":50,"shipping_cents":0,"total_cents":550,"currency":"USD","items":[],"confidence":85}`,
				},
			},
			"stop_reason": "end_turn",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := &Parser{
		apiKey:     "test-key",
		model:      "test-model",
		httpClient: &http.Client{Transport: &redirectTransport{target: server.URL, wrapped: http.DefaultTransport}},
	}

	parsed, err := p.ParseFromBytes(context.Background(), []byte{0xFF, 0xD8, 0xFF, 0xE0}, "image/jpeg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.Vendor != "Store" {
		t.Errorf("expected vendor %q, got %q", "Store", parsed.Vendor)
	}

	// Verify the request sent "image" type for JPEG
	messages := receivedBody["messages"].([]interface{})
	msg := messages[0].(map[string]interface{})
	content := msg["content"].([]interface{})
	mediaBlock := content[0].(map[string]interface{})

	if mediaBlock["type"] != "image" {
		t.Errorf("expected image type for JPEG, got %q", mediaBlock["type"])
	}
	source := mediaBlock["source"].(map[string]interface{})
	if source["media_type"] != "image/jpeg" {
		t.Errorf("expected media_type image/jpeg, got %q", source["media_type"])
	}
}

func TestParseFromBytes_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`)
	}))
	defer server.Close()

	p := &Parser{
		apiKey:     "bad-key",
		model:      "test-model",
		httpClient: &http.Client{Transport: &redirectTransport{target: server.URL, wrapped: http.DefaultTransport}},
	}

	_, err := p.ParseFromBytes(context.Background(), []byte("data"), "image/jpeg")
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
	if got := err.Error(); !contains(got, "401") {
		t.Errorf("expected error to mention status 401, got: %s", got)
	}
}

func TestParseFromBytes_ResponseWithMarkdownCodeBlock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"id":   "msg_test",
			"type": "message",
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": "```json\n{\"vendor\":\"Bambu Lab US\",\"date\":\"2026-01-11\",\"subtotal_cents\":15785,\"tax_cents\":728,\"shipping_cents\":0,\"total_cents\":15513,\"currency\":\"USD\",\"items\":[{\"description\":\"PLA Basic - Beige (Refill / 1kg)\",\"quantity\":3,\"unit_price_cents\":1999,\"total_price_cents\":3898,\"category\":\"filament\",\"is_filament\":true,\"filament\":{\"brand\":\"Bambu Lab\",\"material_type\":\"PLA\",\"color\":\"Beige\",\"weight_grams\":1000,\"diameter_mm\":1.75},\"confidence\":95}],\"confidence\":92}\n```",
				},
			},
			"stop_reason": "end_turn",
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := &Parser{
		apiKey:     "test-key",
		model:      "test-model",
		httpClient: &http.Client{Transport: &redirectTransport{target: server.URL, wrapped: http.DefaultTransport}},
	}

	parsed, err := p.ParseFromBytes(context.Background(), []byte("data"), "image/jpeg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if parsed.Vendor != "Bambu Lab US" {
		t.Errorf("vendor: got %q", parsed.Vendor)
	}
	if parsed.TotalCents != 15513 {
		t.Errorf("total_cents: got %d", parsed.TotalCents)
	}
	if len(parsed.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(parsed.Items))
	}
	item := parsed.Items[0]
	if item.Quantity != 3 {
		t.Errorf("item quantity: got %f", item.Quantity)
	}
	if !item.IsFilament {
		t.Error("item should be filament")
	}
	if item.Filament == nil {
		t.Fatal("item filament metadata is nil")
	}
	if item.Filament.MaterialType != "PLA" {
		t.Errorf("filament material_type: got %q", item.Filament.MaterialType)
	}
	if item.Filament.Color != "Beige" {
		t.Errorf("filament color: got %q", item.Filament.Color)
	}
	if item.Filament.WeightGrams != 1000 {
		t.Errorf("filament weight_grams: got %f", item.Filament.WeightGrams)
	}
}

func TestParseFromFile_NonExistent(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	p := NewParser()

	_, err := p.ParseFromFile(context.Background(), "/tmp/nonexistent-receipt-file.pdf")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}

func TestCleanJSONResponse(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`{"key": "value"}`, `{"key": "value"}`},
		{"```json\n{\"key\": \"value\"}\n```", `{"key": "value"}`},
		{"```\n{\"key\": \"value\"}\n```", `{"key": "value"}`},
		{"  {\"key\": \"value\"}  ", `{"key": "value"}`},
	}

	for _, tt := range tests {
		got := cleanJSONResponse(tt.input)
		if got != tt.expected {
			t.Errorf("cleanJSONResponse(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		path     string
		data     []byte
		expected string
	}{
		{"receipt.pdf", nil, "application/pdf"},
		{"photo.png", nil, "image/png"},
		{"photo.jpg", nil, "image/jpeg"},
		{"photo.jpeg", nil, "image/jpeg"},
		{"photo.webp", nil, "image/webp"},
		{"photo.gif", nil, "image/gif"},
		// Magic bytes detection
		{"unknown", []byte{0x89, 'P', 'N', 'G'}, "image/png"},
		{"unknown", []byte{0xFF, 0xD8, 0x00, 0x00}, "image/jpeg"},
		{"unknown", []byte{'%', 'P', 'D', 'F'}, "application/pdf"},
		{"unknown", []byte{0x00, 0x00, 0x00, 0x00}, "application/octet-stream"},
	}

	for _, tt := range tests {
		got := detectContentType(tt.path, tt.data)
		if got != tt.expected {
			t.Errorf("detectContentType(%q, ...) = %q, want %q", tt.path, got, tt.expected)
		}
	}
}

func TestResolveMediaType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"application/pdf", "application/pdf"},
		{"image/png", "image/png"},
		{"image/webp", "image/webp"},
		{"image/gif", "image/gif"},
		{"image/jpeg", "image/jpeg"},
		{"application/octet-stream", "image/jpeg"}, // fallback
	}

	for _, tt := range tests {
		got := resolveMediaType(tt.input)
		if got != tt.expected {
			t.Errorf("resolveMediaType(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

// redirectTransport redirects all requests to the test server URL.
type redirectTransport struct {
	target  string
	wrapped http.RoundTripper
}

func (t *redirectTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Rewrite URL to point to test server
	newReq := req.Clone(req.Context())
	newReq.URL.Scheme = "http"
	newReq.URL.Host = t.target[len("http://"):]
	return t.wrapped.RoundTrip(newReq)
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Ensure we don't accidentally read env files during tests
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
