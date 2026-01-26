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

// Parser handles receipt parsing using OpenAI Vision API.
type Parser struct {
	apiKey     string
	httpClient *http.Client
}

// NewParser creates a new receipt parser.
func NewParser() *Parser {
	apiKey := os.Getenv("OPENAI_API_KEY")
	return &Parser{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// OpenAI API request/response types
type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens"`
	Temperature float64         `json:"temperature"`
}

type openAIMessage struct {
	Role    string        `json:"role"`
	Content []interface{} `json:"content"`
}

type openAITextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type openAIImageContent struct {
	Type     string         `json:"type"`
	ImageURL openAIImageURL `json:"image_url"`
}

type openAIImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

type openAIResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
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
		return nil, fmt.Errorf("OPENAI_API_KEY not set")
	}

	// Convert image to base64
	base64Data := base64.StdEncoding.EncodeToString(data)

	// Determine the media type for the data URL
	mediaType := "image/jpeg"
	if strings.Contains(contentType, "png") {
		mediaType = "image/png"
	} else if strings.Contains(contentType, "pdf") {
		mediaType = "application/pdf"
	} else if strings.Contains(contentType, "webp") {
		mediaType = "image/webp"
	} else if strings.Contains(contentType, "gif") {
		mediaType = "image/gif"
	}

	dataURL := fmt.Sprintf("data:%s;base64,%s", mediaType, base64Data)

	// Build the prompt
	systemPrompt := `You are a receipt parser for a 3D printing business. Extract structured data from receipt images.

Your task is to analyze the receipt and extract:
1. Vendor/store name
2. Date of purchase
3. All line items with quantities and prices
4. Subtotal, tax, shipping (if applicable), and total
5. For filament/spool purchases, extract: brand, material type (PLA, PETG, ABS, etc.), color, weight in grams, diameter in mm

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
      "description": "Hatchbox PLA 1.75mm Black 1kg",
      "quantity": 1,
      "unit_price_cents": 2499,
      "total_price_cents": 2499,
      "category": "filament",
      "is_filament": true,
      "filament": {
        "brand": "Hatchbox",
        "material_type": "PLA",
        "color": "Black",
        "weight_grams": 1000,
        "diameter_mm": 1.75
      },
      "confidence": 95
    }
  ],
  "confidence": 90
}

Categories for items: filament, parts, tools, shipping, other

For prices, convert to cents (e.g., $24.99 = 2499 cents).
For filament, try to identify: brand, material (PLA/PETG/ABS/ASA/TPU/etc), color, weight, diameter.
If a field cannot be determined, use reasonable defaults or null.
Confidence should be 0-100 based on how certain you are about the extraction.`

	// Create the request
	req := openAIRequest{
		Model: "gpt-4o",
		Messages: []openAIMessage{
			{
				Role: "user",
				Content: []interface{}{
					openAITextContent{
						Type: "text",
						Text: systemPrompt,
					},
					openAIImageContent{
						Type: "image_url",
						ImageURL: openAIImageURL{
							URL:    dataURL,
							Detail: "high",
						},
					},
				},
			},
		},
		MaxTokens:   4096,
		Temperature: 0.1,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Make the API request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

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

	var apiResp openAIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if apiResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", apiResp.Error.Message)
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from API")
	}

	// Parse the JSON response
	content := apiResp.Choices[0].Message.Content
	content = cleanJSONResponse(content)

	var parsed model.ParsedReceipt
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse extracted data: %w (content: %s)", err, content)
	}

	// Store the raw response
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

// detectContentType determines the content type from file extension and magic bytes.
func detectContentType(filePath string, data []byte) string {
	// Check file extension first
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
