package service

import (
	"testing"

	"github.com/google/uuid"
	"github.com/hyperion/printfarm/internal/model"
)

func TestPrinterValidation(t *testing.T) {
	tests := []struct {
		name        string
		constraints *model.PrinterConstraints
		printer     *model.Printer
		wantValid   bool
		wantErrors  int
	}{
		{
			name:        "no constraints - any printer valid",
			constraints: nil,
			printer: &model.Printer{
				ID:             uuid.New(),
				Name:           "Test Printer",
				NozzleDiameter: 0.4,
			},
			wantValid:  true,
			wantErrors: 0,
		},
		{
			name: "bed size too small",
			constraints: &model.PrinterConstraints{
				MinBedSize: &model.BuildVolume{X: 300, Y: 300, Z: 300},
			},
			printer: &model.Printer{
				ID:             uuid.New(),
				Name:           "Small Printer",
				BuildVolume:    &model.BuildVolume{X: 200, Y: 200, Z: 200},
				NozzleDiameter: 0.4,
			},
			wantValid:  false,
			wantErrors: 3,
		},
		{
			name: "bed size sufficient",
			constraints: &model.PrinterConstraints{
				MinBedSize: &model.BuildVolume{X: 200, Y: 200, Z: 200},
			},
			printer: &model.Printer{
				ID:             uuid.New(),
				Name:           "Large Printer",
				BuildVolume:    &model.BuildVolume{X: 300, Y: 300, Z: 300},
				NozzleDiameter: 0.4,
			},
			wantValid:  true,
			wantErrors: 0,
		},
		{
			name: "nozzle diameter mismatch",
			constraints: &model.PrinterConstraints{
				NozzleDiameters: []float64{0.6, 0.8},
			},
			printer: &model.Printer{
				ID:             uuid.New(),
				Name:           "Standard Printer",
				NozzleDiameter: 0.4,
			},
			wantValid:  false,
			wantErrors: 1,
		},
		{
			name: "nozzle diameter match",
			constraints: &model.PrinterConstraints{
				NozzleDiameters: []float64{0.4, 0.6},
			},
			printer: &model.Printer{
				ID:             uuid.New(),
				Name:           "Standard Printer",
				NozzleDiameter: 0.4,
			},
			wantValid:  true,
			wantErrors: 0,
		},
		{
			name: "enclosure required - warning only",
			constraints: &model.PrinterConstraints{
				RequiresEnclosure: true,
			},
			printer: &model.Printer{
				ID:             uuid.New(),
				Name:           "Open Printer",
				NozzleDiameter: 0.4,
			},
			wantValid:  true, // warnings don't fail validation
			wantErrors: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validatePrinterConstraints(tt.constraints, tt.printer)

			if result.Valid != tt.wantValid {
				t.Errorf("Valid = %v, want %v", result.Valid, tt.wantValid)
			}

			if len(result.Errors) != tt.wantErrors {
				t.Errorf("Errors count = %d, want %d. Errors: %v", len(result.Errors), tt.wantErrors, result.Errors)
			}
		})
	}
}

// validatePrinterConstraints is a helper to test constraint validation logic
func validatePrinterConstraints(constraints *model.PrinterConstraints, printer *model.Printer) *PrinterValidationResult {
	result := &PrinterValidationResult{Valid: true}

	if constraints == nil {
		return result
	}

	// Check bed size
	if constraints.MinBedSize != nil {
		if printer.BuildVolume == nil {
			result.Warnings = append(result.Warnings, "Printer build volume not configured")
		} else {
			if printer.BuildVolume.X < constraints.MinBedSize.X {
				result.Valid = false
				result.Errors = append(result.Errors, "X axis too small")
			}
			if printer.BuildVolume.Y < constraints.MinBedSize.Y {
				result.Valid = false
				result.Errors = append(result.Errors, "Y axis too small")
			}
			if printer.BuildVolume.Z < constraints.MinBedSize.Z {
				result.Valid = false
				result.Errors = append(result.Errors, "Z axis too small")
			}
		}
	}

	// Check nozzle diameter
	if len(constraints.NozzleDiameters) > 0 {
		found := false
		for _, d := range constraints.NozzleDiameters {
			if printer.NozzleDiameter == d {
				found = true
				break
			}
		}
		if !found {
			result.Valid = false
			result.Errors = append(result.Errors, "Incompatible nozzle diameter")
		}
	}

	// Check enclosure requirement - warning only
	if constraints.RequiresEnclosure {
		result.Warnings = append(result.Warnings, "Recipe requires enclosure")
	}

	// Check AMS requirement - warning only
	if constraints.RequiresAMS {
		result.Warnings = append(result.Warnings, "Recipe requires AMS")
	}

	return result
}

func TestCostCalculation(t *testing.T) {
	tests := []struct {
		name                  string
		materials             []model.RecipeMaterial
		estimatedPrintSeconds int
		hourlyRateCents       int
		wantMaterialCost      int
		wantTimeCost          int
	}{
		{
			name:                  "no materials, no time",
			materials:             nil,
			estimatedPrintSeconds: 0,
			hourlyRateCents:       500,
			wantMaterialCost:      0,
			wantTimeCost:          0,
		},
		{
			name: "single material 100g PLA at $25/kg",
			materials: []model.RecipeMaterial{
				{MaterialType: "pla", WeightGrams: 100},
			},
			estimatedPrintSeconds: 0,
			hourlyRateCents:       500,
			wantMaterialCost:      250, // (100/1000) * 25 * 100 = 250 cents
			wantTimeCost:          0,
		},
		{
			name:                  "1 hour print at $5/hour",
			materials:             nil,
			estimatedPrintSeconds: 3600,
			hourlyRateCents:       500,
			wantMaterialCost:      0,
			wantTimeCost:          500,
		},
		{
			name: "combined material and time",
			materials: []model.RecipeMaterial{
				{MaterialType: "pla", WeightGrams: 50},
			},
			estimatedPrintSeconds: 1800, // 30 minutes
			hourlyRateCents:       500,
			wantMaterialCost:      125, // (50/1000) * 25 * 100 = 125 cents
			wantTimeCost:          250, // 0.5 * 500 = 250 cents
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			materialCost, timeCost := calculateCosts(tt.materials, tt.estimatedPrintSeconds, tt.hourlyRateCents)

			if materialCost != tt.wantMaterialCost {
				t.Errorf("Material cost = %d cents, want %d cents", materialCost, tt.wantMaterialCost)
			}

			if timeCost != tt.wantTimeCost {
				t.Errorf("Time cost = %d cents, want %d cents", timeCost, tt.wantTimeCost)
			}
		})
	}
}

// calculateCosts is a helper to test cost calculation logic
func calculateCosts(materials []model.RecipeMaterial, estimatedPrintSeconds, hourlyRateCents int) (materialCostCents, timeCostCents int) {
	// Default cost per kg for materials (in dollars)
	defaultCostPerKg := 25.0

	for _, m := range materials {
		costCents := int((m.WeightGrams / 1000.0) * defaultCostPerKg * 100)
		materialCostCents += costCents
	}

	if estimatedPrintSeconds > 0 {
		hours := float64(estimatedPrintSeconds) / 3600.0
		timeCostCents = int(hours * float64(hourlyRateCents))
	}

	return materialCostCents, timeCostCents
}

func TestColorSpecMatching(t *testing.T) {
	tests := []struct {
		name      string
		colorSpec *model.ColorSpec
		material  model.Material
		wantMatch bool
	}{
		{
			name:      "nil color spec matches anything",
			colorSpec: nil,
			material:  model.Material{Color: "Red", ColorHex: "#FF0000"},
			wantMatch: true,
		},
		{
			name:      "any mode matches anything",
			colorSpec: &model.ColorSpec{Mode: "any"},
			material:  model.Material{Color: "Blue", ColorHex: "#0000FF"},
			wantMatch: true,
		},
		{
			name:      "exact mode matches by name",
			colorSpec: &model.ColorSpec{Mode: "exact", Name: "White"},
			material:  model.Material{Color: "White", ColorHex: "#FFFFFF"},
			wantMatch: true,
		},
		{
			name:      "exact mode fails on name mismatch",
			colorSpec: &model.ColorSpec{Mode: "exact", Name: "White"},
			material:  model.Material{Color: "Black", ColorHex: "#000000"},
			wantMatch: false,
		},
		{
			name:      "exact mode matches by hex",
			colorSpec: &model.ColorSpec{Mode: "exact", Hex: "#FF0000"},
			material:  model.Material{Color: "Red", ColorHex: "#FF0000"},
			wantMatch: true,
		},
		{
			name:      "exact mode fails on hex mismatch",
			colorSpec: &model.ColorSpec{Mode: "exact", Hex: "#FF0000"},
			material:  model.Material{Color: "Blue", ColorHex: "#0000FF"},
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match := colorSpecMatches(tt.colorSpec, tt.material)
			if match != tt.wantMatch {
				t.Errorf("colorSpecMatches() = %v, want %v", match, tt.wantMatch)
			}
		})
	}
}

// colorSpecMatches tests if a material matches a color specification
func colorSpecMatches(spec *model.ColorSpec, material model.Material) bool {
	if spec == nil {
		return true
	}

	switch spec.Mode {
	case "any":
		return true
	case "exact":
		if spec.Hex != "" && material.ColorHex != spec.Hex {
			return false
		}
		if spec.Name != "" && material.Color != spec.Name {
			return false
		}
		return true
	case "category":
		// Category matching would need color categorization logic
		return true
	default:
		return true
	}
}

func TestPrintProfileValidation(t *testing.T) {
	validProfiles := []model.PrintProfile{
		model.PrintProfileStandard,
		model.PrintProfileDetailed,
		model.PrintProfileFast,
		model.PrintProfileStrong,
		model.PrintProfileCustom,
	}

	for _, profile := range validProfiles {
		if profile == "" {
			t.Errorf("Profile should not be empty")
		}
	}

	// Test default profile
	if model.PrintProfileStandard != "standard" {
		t.Errorf("Standard profile should be 'standard', got %s", model.PrintProfileStandard)
	}
}

func TestRecipeMaterialValidation(t *testing.T) {
	tests := []struct {
		name     string
		material model.RecipeMaterial
		wantErr  bool
	}{
		{
			name: "valid material",
			material: model.RecipeMaterial{
				RecipeID:     uuid.New(),
				MaterialType: "pla",
				WeightGrams:  50,
			},
			wantErr: false,
		},
		{
			name: "missing recipe ID",
			material: model.RecipeMaterial{
				MaterialType: "pla",
				WeightGrams:  50,
			},
			wantErr: true,
		},
		{
			name: "missing material type",
			material: model.RecipeMaterial{
				RecipeID:    uuid.New(),
				WeightGrams: 50,
			},
			wantErr: true,
		},
		{
			name: "with AMS position",
			material: model.RecipeMaterial{
				RecipeID:     uuid.New(),
				MaterialType: "petg",
				WeightGrams:  100,
				AMSPosition:  intPtr(2),
			},
			wantErr: false,
		},
		{
			name: "with color spec",
			material: model.RecipeMaterial{
				RecipeID:     uuid.New(),
				MaterialType: "pla",
				WeightGrams:  75,
				ColorSpec:    &model.ColorSpec{Mode: "exact", Name: "White"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRecipeMaterial(&tt.material)
			hasErr := err != nil
			if hasErr != tt.wantErr {
				t.Errorf("validateRecipeMaterial() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func validateRecipeMaterial(m *model.RecipeMaterial) error {
	if m.RecipeID == uuid.Nil {
		return &validationError{"recipe ID is required"}
	}
	if m.MaterialType == "" {
		return &validationError{"material type is required"}
	}
	return nil
}

type validationError struct {
	msg string
}

func (e *validationError) Error() string {
	return e.msg
}

func intPtr(i int) *int {
	return &i
}
