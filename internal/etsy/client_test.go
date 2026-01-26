package etsy

import (
	"net/url"
	"testing"
)

func TestMoneyToCents(t *testing.T) {
	tests := []struct {
		name     string
		money    EtsyMoney
		expected int
	}{
		{
			name:     "standard USD cents",
			money:    EtsyMoney{Amount: 2500, Divisor: 100, CurrencyCode: "USD"},
			expected: 2500,
		},
		{
			name:     "divisor 1",
			money:    EtsyMoney{Amount: 25, Divisor: 1, CurrencyCode: "USD"},
			expected: 2500,
		},
		{
			name:     "zero divisor defaults to amount",
			money:    EtsyMoney{Amount: 1500, Divisor: 0, CurrencyCode: "USD"},
			expected: 1500,
		},
		{
			name:     "fractional with divisor 100",
			money:    EtsyMoney{Amount: 1599, Divisor: 100, CurrencyCode: "USD"},
			expected: 1599,
		},
		{
			name:     "divisor 10",
			money:    EtsyMoney{Amount: 250, Divisor: 10, CurrencyCode: "USD"},
			expected: 2500,
		},
		{
			name:     "divisor 1000 (3 decimal places)",
			money:    EtsyMoney{Amount: 25000, Divisor: 1000, CurrencyCode: "USD"},
			expected: 2500,
		},
		{
			name:     "zero amount",
			money:    EtsyMoney{Amount: 0, Divisor: 100, CurrencyCode: "USD"},
			expected: 0,
		},
		{
			name:     "EUR currency",
			money:    EtsyMoney{Amount: 1999, Divisor: 100, CurrencyCode: "EUR"},
			expected: 1999,
		},
		{
			name:     "GBP currency",
			money:    EtsyMoney{Amount: 1499, Divisor: 100, CurrencyCode: "GBP"},
			expected: 1499,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MoneyToCents(tt.money)
			if result != tt.expected {
				t.Errorf("MoneyToCents() = %d, expected %d", result, tt.expected)
			}
		})
	}
}

func TestReceiptQueryOptions_ToURLParams(t *testing.T) {
	tests := []struct {
		name     string
		opts     ReceiptQueryOptions
		expected map[string]string
	}{
		{
			name:     "empty options",
			opts:     ReceiptQueryOptions{},
			expected: map[string]string{},
		},
		{
			name: "min_created only",
			opts: ReceiptQueryOptions{
				MinCreated: 1700000000,
			},
			expected: map[string]string{
				"min_created": "1700000000",
			},
		},
		{
			name: "max_created only",
			opts: ReceiptQueryOptions{
				MaxCreated: 1700100000,
			},
			expected: map[string]string{
				"max_created": "1700100000",
			},
		},
		{
			name: "was_paid true",
			opts: ReceiptQueryOptions{
				WasPaid: boolPtr(true),
			},
			expected: map[string]string{
				"was_paid": "true",
			},
		},
		{
			name: "was_paid false",
			opts: ReceiptQueryOptions{
				WasPaid: boolPtr(false),
			},
			expected: map[string]string{
				"was_paid": "false",
			},
		},
		{
			name: "was_shipped true",
			opts: ReceiptQueryOptions{
				WasShipped: boolPtr(true),
			},
			expected: map[string]string{
				"was_shipped": "true",
			},
		},
		{
			name: "limit only",
			opts: ReceiptQueryOptions{
				Limit: 50,
			},
			expected: map[string]string{
				"limit": "50",
			},
		},
		{
			name: "offset only",
			opts: ReceiptQueryOptions{
				Offset: 100,
			},
			expected: map[string]string{
				"offset": "100",
			},
		},
		{
			name: "all options",
			opts: ReceiptQueryOptions{
				MinCreated: 1700000000,
				MaxCreated: 1700100000,
				WasPaid:    boolPtr(true),
				WasShipped: boolPtr(false),
				Limit:      25,
				Offset:     50,
			},
			expected: map[string]string{
				"min_created": "1700000000",
				"max_created": "1700100000",
				"was_paid":    "true",
				"was_shipped": "false",
				"limit":       "25",
				"offset":      "50",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := tt.opts.ToURLParams()

			// Check expected params are present
			for key, expectedVal := range tt.expected {
				if params.Get(key) != expectedVal {
					t.Errorf("param %s = %q, expected %q", key, params.Get(key), expectedVal)
				}
			}

			// Check no unexpected params
			for key := range params {
				if _, ok := tt.expected[key]; !ok {
					t.Errorf("unexpected param %s = %q", key, params.Get(key))
				}
			}
		})
	}
}

func TestListingQueryOptions_ToURLParams(t *testing.T) {
	tests := []struct {
		name     string
		opts     ListingQueryOptions
		expected map[string]string
	}{
		{
			name:     "empty options",
			opts:     ListingQueryOptions{},
			expected: map[string]string{},
		},
		{
			name: "state only",
			opts: ListingQueryOptions{
				State: "active",
			},
			expected: map[string]string{
				"state": "active",
			},
		},
		{
			name: "all options",
			opts: ListingQueryOptions{
				State:  "inactive",
				Limit:  100,
				Offset: 25,
			},
			expected: map[string]string{
				"state":  "inactive",
				"limit":  "100",
				"offset": "25",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := tt.opts.ToURLParams()

			for key, expectedVal := range tt.expected {
				if params.Get(key) != expectedVal {
					t.Errorf("param %s = %q, expected %q", key, params.Get(key), expectedVal)
				}
			}
		})
	}
}

func TestGenerateAuthURL(t *testing.T) {
	client := NewClient("test-client-id", "http://localhost:8080/callback")

	state := "test-state-123"
	codeChallenge := "test-challenge-456"

	authURL := client.GenerateAuthURL(state, codeChallenge)

	// Parse the URL
	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("failed to parse auth URL: %v", err)
	}

	// Check base URL
	if parsed.Scheme != "https" {
		t.Errorf("expected https scheme, got %s", parsed.Scheme)
	}
	if parsed.Host != "www.etsy.com" {
		t.Errorf("expected host www.etsy.com, got %s", parsed.Host)
	}
	if parsed.Path != "/oauth/connect" {
		t.Errorf("expected path /oauth/connect, got %s", parsed.Path)
	}

	// Check query params
	query := parsed.Query()
	if query.Get("response_type") != "code" {
		t.Errorf("expected response_type=code, got %s", query.Get("response_type"))
	}
	if query.Get("client_id") != "test-client-id" {
		t.Errorf("expected client_id=test-client-id, got %s", query.Get("client_id"))
	}
	if query.Get("redirect_uri") != "http://localhost:8080/callback" {
		t.Errorf("expected redirect_uri=http://localhost:8080/callback, got %s", query.Get("redirect_uri"))
	}
	if query.Get("state") != state {
		t.Errorf("expected state=%s, got %s", state, query.Get("state"))
	}
	if query.Get("code_challenge") != codeChallenge {
		t.Errorf("expected code_challenge=%s, got %s", codeChallenge, query.Get("code_challenge"))
	}
	if query.Get("code_challenge_method") != "S256" {
		t.Errorf("expected code_challenge_method=S256, got %s", query.Get("code_challenge_method"))
	}
}

func TestAPIVariation_Serialization(t *testing.T) {
	variation := APIVariation{
		PropertyID:     123,
		ValueID:        456,
		FormattedName:  "Color",
		FormattedValue: "Blue",
	}

	if variation.PropertyID != 123 {
		t.Errorf("expected PropertyID 123, got %d", variation.PropertyID)
	}
	if variation.FormattedName != "Color" {
		t.Errorf("expected FormattedName 'Color', got %s", variation.FormattedName)
	}
}

func TestAPIReceipt_Fields(t *testing.T) {
	receipt := APIReceipt{
		ReceiptID:       12345,
		BuyerUserID:     67890,
		BuyerEmail:      "test@example.com",
		Name:            "Test Buyer",
		Status:          "paid",
		IsPaid:          true,
		IsShipped:       false,
		CreateTimestamp: 1700000000,
		Grandtotal: EtsyMoney{
			Amount:       5000,
			Divisor:      100,
			CurrencyCode: "USD",
		},
	}

	if receipt.ReceiptID != 12345 {
		t.Errorf("expected ReceiptID 12345, got %d", receipt.ReceiptID)
	}
	if !receipt.IsPaid {
		t.Error("expected IsPaid to be true")
	}
	if receipt.IsShipped {
		t.Error("expected IsShipped to be false")
	}
	if MoneyToCents(receipt.Grandtotal) != 5000 {
		t.Errorf("expected Grandtotal 5000 cents, got %d", MoneyToCents(receipt.Grandtotal))
	}
}

func TestAPITransaction_Fields(t *testing.T) {
	tx := APITransaction{
		TransactionID: 111,
		Title:         "Test Product",
		ListingID:     222,
		Quantity:      2,
		SKU:           "TEST-SKU-001",
		IsDigital:     false,
		Price: EtsyMoney{
			Amount:       2500,
			Divisor:      100,
			CurrencyCode: "USD",
		},
	}

	if tx.TransactionID != 111 {
		t.Errorf("expected TransactionID 111, got %d", tx.TransactionID)
	}
	if tx.SKU != "TEST-SKU-001" {
		t.Errorf("expected SKU 'TEST-SKU-001', got %s", tx.SKU)
	}
	if tx.Quantity != 2 {
		t.Errorf("expected Quantity 2, got %d", tx.Quantity)
	}
	if MoneyToCents(tx.Price) != 2500 {
		t.Errorf("expected Price 2500 cents, got %d", MoneyToCents(tx.Price))
	}
}

func TestAPIListing_Fields(t *testing.T) {
	listing := APIListing{
		ListingID:        333,
		ShopID:           444,
		Title:            "Test Listing",
		State:            "active",
		Quantity:         10,
		URL:              "https://etsy.com/listing/333",
		NumFavorers:      50,
		Views:            1000,
		IsCustomizable:   true,
		IsPersonalizable: false,
		HasVariations:    true,
		Tags:             []string{"tag1", "tag2"},
		SKUs:             []string{"SKU-A", "SKU-B"},
		Price: EtsyMoney{
			Amount:       3500,
			Divisor:      100,
			CurrencyCode: "USD",
		},
	}

	if listing.ListingID != 333 {
		t.Errorf("expected ListingID 333, got %d", listing.ListingID)
	}
	if listing.State != "active" {
		t.Errorf("expected State 'active', got %s", listing.State)
	}
	if listing.Quantity != 10 {
		t.Errorf("expected Quantity 10, got %d", listing.Quantity)
	}
	if !listing.IsCustomizable {
		t.Error("expected IsCustomizable to be true")
	}
	if listing.IsPersonalizable {
		t.Error("expected IsPersonalizable to be false")
	}
	if len(listing.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(listing.Tags))
	}
	if len(listing.SKUs) != 2 {
		t.Errorf("expected 2 SKUs, got %d", len(listing.SKUs))
	}
}

// Helper function
func boolPtr(b bool) *bool {
	return &b
}
