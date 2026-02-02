package receipt

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hyperion/printfarm/internal/model"
)

// Parser handles receipt parsing using the Anthropic Messages API.
type Parser struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewParser creates a new receipt parser using the Anthropic API.
// It reads the API key from the ANTHROPIC_API_KEY env var.
func NewParser() *Parser {
	return NewParserWithKey(os.Getenv("ANTHROPIC_API_KEY"))
}

// NewParserWithKey creates a new receipt parser with an explicit API key.
// If apiKey is empty, parsing will fail with a descriptive error.
func NewParserWithKey(apiKey string) *Parser {
	model := os.Getenv("RECEIPT_PARSER_MODEL")
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	return &Parser{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// HasAPIKey returns true if the parser has an API key configured.
func (p *Parser) HasAPIKey() bool {
	return p.apiKey != ""
}

// Anthropic API request/response types

type anthropicRequest struct {
	Model     string              `json:"model"`
	MaxTokens int                 `json:"max_tokens"`
	Messages  []anthropicMessage  `json:"messages"`
}

type anthropicMessage struct {
	Role    string        `json:"role"`
	Content []interface{} `json:"content"`
}

type anthropicTextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicImageContent struct {
	Type   string               `json:"type"`
	Source anthropicMediaSource `json:"source"`
}

type anthropicDocumentContent struct {
	Type   string               `json:"type"`
	Source anthropicMediaSource `json:"source"`
}

type anthropicMediaSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type anthropicResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Error      *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// ParseFromFile parses a receipt from a file path.
func (p *Parser) ParseFromFile(ctx context.Context, filePath string) (*model.ParsedReceipt, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	contentType := detectContentType(filePath, data)
	return p.ParseFromBytes(ctx, data, contentType)
}

// ParseFromBytes parses a receipt from raw bytes.
func (p *Parser) ParseFromBytes(ctx context.Context, data []byte, contentType string) (*model.ParsedReceipt, error) {
	if p.apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not set — add it to your .env file to enable receipt parsing")
	}

	base64Data := base64.StdEncoding.EncodeToString(data)

	// Determine the media type
	mediaType := resolveMediaType(contentType)
	isPDF := strings.Contains(mediaType, "pdf")

	// Build the media content block: "document" for PDFs, "image" for images
	var mediaBlock interface{}
	if isPDF {
		mediaBlock = anthropicDocumentContent{
			Type: "document",
			Source: anthropicMediaSource{
				Type:      "base64",
				MediaType: mediaType,
				Data:      base64Data,
			},
		}
	} else {
		mediaBlock = anthropicImageContent{
			Type: "image",
			Source: anthropicMediaSource{
				Type:      "base64",
				MediaType: mediaType,
				Data:      base64Data,
			},
		}
	}

	prompt := `You are a receipt parser for a maker/3D printing business. Extract structured data from this receipt.

Your task is to analyze the receipt and extract:
1. Vendor/store name
2. Date of purchase (from invoice/payment date)
3. ALL line items with quantities and prices — include EVERY item on the receipt, not just filament
4. Subtotal, tax, shipping (if applicable), and total
5. For filament/spool purchases, extract: brand, material type (PLA, PETG, ABS, etc.), color, weight in grams, diameter in mm
6. For non-filament items (tools, supplies, parts, etc.), provide a clean, descriptive item name

Important rules:
- Include EVERY item from the receipt. Non-filament items (lamp cords, lightbulbs, tools, hardware, etc.) are important supply items
- For items with quantity > 1, the unit_price_cents should be the per-unit price, and total_price_cents should be quantity × unit_price (the "Items SubTotal" column value)
- MULTI-PACKS: If a product name says "6 Pack", "4 Pack", "50pk", "10-Pack", etc., the quantity MUST be the pack count (e.g., 6) and unit_price_cents MUST be the total line price divided by that pack count. Example: "6 Pack Lamp Cord" at $37.33 → quantity: 6, unit_price_cents: 622, total_price_cents: 3733
- Convert all prices to cents (e.g., $19.99 = 1999 cents)
- For Bambu Lab invoices: the "Items SubTotal" column is the final per-line total after discounts and before tax
- The vendor is the seller name (e.g., "Bambu Lab US", "Amazon.com"), not the buyer
- For filament items, extract variant info: color name, material type from the product name (PLA Basic = PLA, PETG Translucent = PETG, PLA Matte = PLA, PLA Silk+ = PLA)
- For filament items, also provide a hex color code (color_hex) that represents the filament color (e.g., Black = #000000, White = #FFFFFF, Red = #FF0000, Beige = #F5DEB3)
- Weight is typically in the variant description (e.g., "1kg" = 1000 grams, "1 kg" = 1000 grams)
- Diameter is typically 1.75mm for these products
- For non-filament items, use a short clean description of the INDIVIDUAL item, stripping pack count from the name (e.g., "6 Pack Plug in Hanging Light Kit..." → "Plug-in Hanging Light Cord 12ft", "LiteHistory 6W LED Bulb G16.5 2700K 6-Pack" → "LED Globe Bulb G16.5 6W 2700K")

Return ONLY valid JSON in this exact format (no markdown, no explanation):
{
  "vendor": "Store Name",
  "date": "YYYY-MM-DD",
  "subtotal_cents": 2499,
  "tax_cents": 200,
  "shipping_cents": 0,
  "total_cents": 2699,
  "currency": "USD",
  "items": [
    {
      "description": "PLA Basic - Black (Refill / 1kg)",
      "quantity": 1,
      "unit_price_cents": 1999,
      "total_price_cents": 1299,
      "category": "filament",
      "is_filament": true,
      "filament": {
        "brand": "Bambu Lab",
        "material_type": "PLA",
        "color": "Black",
        "color_hex": "#000000",
        "weight_grams": 1000,
        "diameter_mm": 1.75
      },
      "confidence": 95
    },
    {
      "description": "Plug-in Hanging Light Cord 12ft with Switch",
      "quantity": 6,
      "unit_price_cents": 622,
      "total_price_cents": 3733,
      "category": "parts",
      "is_filament": false,
      "confidence": 95
    }
  ],
  "confidence": 90
}

Categories for items: filament, parts, tools, shipping, other
- filament: 3D printer filament spools or refills
- parts: components, hardware, electrical parts, supplies used in projects
- tools: tools, equipment, instruments
- shipping: shipping or delivery charges
- other: anything that doesn't fit the above

If a field cannot be determined, use reasonable defaults or null.
Confidence should be 0-100 based on how certain you are about the extraction.`

	req := anthropicRequest{
		Model:     p.model,
		MaxTokens: 4096,
		Messages: []anthropicMessage{
			{
				Role: "user",
				Content: []interface{}{
					mediaBlock,
					anthropicTextContent{
						Type: "text",
						Text: prompt,
					},
				},
			},
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if apiResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", apiResp.Error.Message)
	}

	// Extract text from response content blocks
	var textContent string
	for _, block := range apiResp.Content {
		if block.Type == "text" {
			textContent += block.Text
		}
	}

	if textContent == "" {
		return nil, fmt.Errorf("no text content in API response")
	}

	// Parse the JSON response
	content := cleanJSONResponse(textContent)

	var parsed model.ParsedReceipt
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse extracted data: %w (content: %s)", err, content)
	}

	parsed.RawText = content

	return &parsed, nil
}

// cleanJSONResponse removes markdown code blocks and other formatting from the response.
func cleanJSONResponse(s string) string {
	s = strings.TrimSpace(s)

	// Remove markdown code blocks
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
	}
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
	}
	if strings.HasSuffix(s, "```") {
		s = strings.TrimSuffix(s, "```")
	}

	return strings.TrimSpace(s)
}

// resolveMediaType maps a content type string to the canonical MIME type.
func resolveMediaType(contentType string) string {
	switch {
	case strings.Contains(contentType, "pdf"):
		return "application/pdf"
	case strings.Contains(contentType, "png"):
		return "image/png"
	case strings.Contains(contentType, "webp"):
		return "image/webp"
	case strings.Contains(contentType, "gif"):
		return "image/gif"
	default:
		return "image/jpeg"
	}
}

// detectContentType determines the content type from file extension and magic bytes.
func detectContentType(filePath string, data []byte) string {
	lower := strings.ToLower(filePath)
	switch {
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(lower, ".pdf"):
		return "application/pdf"
	case strings.HasSuffix(lower, ".webp"):
		return "image/webp"
	case strings.HasSuffix(lower, ".gif"):
		return "image/gif"
	}

	// Fall back to magic bytes detection
	if len(data) >= 4 {
		if data[0] == 0x89 && data[1] == 'P' && data[2] == 'N' && data[3] == 'G' {
			return "image/png"
		}
		if data[0] == 0xFF && data[1] == 0xD8 {
			return "image/jpeg"
		}
		if data[0] == '%' && data[1] == 'P' && data[2] == 'D' && data[3] == 'F' {
			return "application/pdf"
		}
	}

	return "application/octet-stream"
}
