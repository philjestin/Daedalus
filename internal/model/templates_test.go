package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTemplate_JSONSerialization(t *testing.T) {
	template := Template{
		ID:                     uuid.New(),
		Name:                   "Test Product",
		Description:            "A test product template",
		SKU:                    "PROD-001",
		Tags:                   []string{"custom", "bestseller"},
		MaterialType:           MaterialTypePLA,
		EstimatedMaterialGrams: 45.5,
		PreferredPrinterID:     nil,
		AllowAnyPrinter:        true,
		QuantityPerOrder:       2,
		PostProcessChecklist:   []string{"Remove supports", "Sand edges", "Apply finish"},
		IsActive:               true,
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
	}

	t.Run("serialize to JSON", func(t *testing.T) {
		data, err := json.Marshal(template)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if result["name"] != "Test Product" {
			t.Errorf("expected name 'Test Product', got %v", result["name"])
		}
		if result["sku"] != "PROD-001" {
			t.Errorf("expected sku 'PROD-001', got %v", result["sku"])
		}
		if result["material_type"] != "pla" {
			t.Errorf("expected material_type 'pla', got %v", result["material_type"])
		}
		if result["quantity_per_order"] != float64(2) {
			t.Errorf("expected quantity_per_order 2, got %v", result["quantity_per_order"])
		}
		if result["allow_any_printer"] != true {
			t.Errorf("expected allow_any_printer true, got %v", result["allow_any_printer"])
		}
	})

	t.Run("deserialize from JSON", func(t *testing.T) {
		jsonStr := `{
			"id": "550e8400-e29b-41d4-a716-446655440000",
			"name": "JSON Template",
			"description": "From JSON",
			"sku": "JSON-001",
			"tags": ["tag1", "tag2"],
			"material_type": "petg",
			"estimated_material_grams": 100.5,
			"allow_any_printer": false,
			"quantity_per_order": 5,
			"post_process_checklist": ["Step 1", "Step 2"],
			"is_active": true
		}`

		var result Template
		if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if result.Name != "JSON Template" {
			t.Errorf("expected name 'JSON Template', got %s", result.Name)
		}
		if result.MaterialType != MaterialTypePETG {
			t.Errorf("expected material_type PETG, got %s", result.MaterialType)
		}
		if result.QuantityPerOrder != 5 {
			t.Errorf("expected quantity_per_order 5, got %d", result.QuantityPerOrder)
		}
		if len(result.Tags) != 2 {
			t.Errorf("expected 2 tags, got %d", len(result.Tags))
		}
		if len(result.PostProcessChecklist) != 2 {
			t.Errorf("expected 2 checklist items, got %d", len(result.PostProcessChecklist))
		}
	})
}

func TestTemplateDesign_JSONSerialization(t *testing.T) {
	templateID := uuid.New()
	designID := uuid.New()

	td := TemplateDesign{
		ID:            uuid.New(),
		TemplateID:    templateID,
		DesignID:      designID,
		IsPrimary:     true,
		Quantity:      3,
		SequenceOrder: 1,
		Notes:         "Main body component",
		CreatedAt:     time.Now(),
	}

	t.Run("serialize to JSON", func(t *testing.T) {
		data, err := json.Marshal(td)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if result["template_id"] != templateID.String() {
			t.Errorf("expected template_id %s, got %v", templateID, result["template_id"])
		}
		if result["design_id"] != designID.String() {
			t.Errorf("expected design_id %s, got %v", designID, result["design_id"])
		}
		if result["is_primary"] != true {
			t.Errorf("expected is_primary true, got %v", result["is_primary"])
		}
		if result["quantity"] != float64(3) {
			t.Errorf("expected quantity 3, got %v", result["quantity"])
		}
	})
}

func TestTemplate_WithDesigns(t *testing.T) {
	template := Template{
		ID:               uuid.New(),
		Name:             "Multi-Part Product",
		MaterialType:     MaterialTypePLA,
		QuantityPerOrder: 1,
		Designs: []TemplateDesign{
			{
				ID:            uuid.New(),
				DesignID:      uuid.New(),
				IsPrimary:     true,
				Quantity:      1,
				SequenceOrder: 0,
				Notes:         "Main part",
			},
			{
				ID:            uuid.New(),
				DesignID:      uuid.New(),
				IsPrimary:     false,
				Quantity:      2,
				SequenceOrder: 1,
				Notes:         "Support bracket",
			},
		},
	}

	t.Run("template has designs", func(t *testing.T) {
		if len(template.Designs) != 2 {
			t.Errorf("expected 2 designs, got %d", len(template.Designs))
		}
	})

	t.Run("serialize with designs", func(t *testing.T) {
		data, err := json.Marshal(template)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		designs, ok := result["designs"].([]interface{})
		if !ok {
			t.Fatal("designs should be an array")
		}
		if len(designs) != 2 {
			t.Errorf("expected 2 designs in JSON, got %d", len(designs))
		}
	})
}

func TestProject_TemplateFields(t *testing.T) {
	templateID := uuid.New()

	project := Project{
		ID:              uuid.New(),
		Name:            "Order from Template",
		Status:          ProjectStatusActive,
		TemplateID:      &templateID,
		Source:          "etsy",
		ExternalOrderID: "ETSY-12345",
		CustomerNotes:   "Please rush this order",
	}

	t.Run("project has template reference", func(t *testing.T) {
		if project.TemplateID == nil {
			t.Error("expected template_id to be set")
		}
		if *project.TemplateID != templateID {
			t.Errorf("expected template_id %s, got %s", templateID, *project.TemplateID)
		}
	})

	t.Run("project has source tracking", func(t *testing.T) {
		if project.Source != "etsy" {
			t.Errorf("expected source 'etsy', got %s", project.Source)
		}
		if project.ExternalOrderID != "ETSY-12345" {
			t.Errorf("expected external_order_id 'ETSY-12345', got %s", project.ExternalOrderID)
		}
	})

	t.Run("serialize with template fields", func(t *testing.T) {
		data, err := json.Marshal(project)
		if err != nil {
			t.Fatalf("failed to marshal: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("failed to unmarshal: %v", err)
		}

		if result["template_id"] != templateID.String() {
			t.Errorf("expected template_id in JSON, got %v", result["template_id"])
		}
		if result["source"] != "etsy" {
			t.Errorf("expected source 'etsy' in JSON, got %v", result["source"])
		}
		if result["external_order_id"] != "ETSY-12345" {
			t.Errorf("expected external_order_id in JSON, got %v", result["external_order_id"])
		}
		if result["customer_notes"] != "Please rush this order" {
			t.Errorf("expected customer_notes in JSON, got %v", result["customer_notes"])
		}
	})
}

func TestMaterialType_Values(t *testing.T) {
	tests := []struct {
		materialType MaterialType
		expected     string
	}{
		{MaterialTypePLA, "pla"},
		{MaterialTypePETG, "petg"},
		{MaterialTypeABS, "abs"},
		{MaterialTypeASA, "asa"},
		{MaterialTypeTPU, "tpu"},
	}

	for _, tt := range tests {
		t.Run(string(tt.materialType), func(t *testing.T) {
			if string(tt.materialType) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.materialType)
			}
		})
	}
}
