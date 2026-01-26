package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hyperion/printfarm/internal/model"
)

func TestAddMaterialRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request AddMaterialRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: AddMaterialRequest{
				MaterialType:  "pla",
				WeightGrams:   50,
				SequenceOrder: 0,
			},
			wantErr: false,
		},
		{
			name: "with color spec",
			request: AddMaterialRequest{
				MaterialType: "petg",
				WeightGrams:  100,
				ColorSpec: &model.ColorSpec{
					Mode: "exact",
					Name: "White",
					Hex:  "#FFFFFF",
				},
				SequenceOrder: 1,
			},
			wantErr: false,
		},
		{
			name: "with AMS position",
			request: AddMaterialRequest{
				MaterialType:  "pla",
				WeightGrams:   75,
				AMSPosition:   intPtr(2),
				SequenceOrder: 0,
			},
			wantErr: false,
		},
		{
			name: "with notes",
			request: AddMaterialRequest{
				MaterialType:  "abs",
				WeightGrams:   120,
				Notes:         "Main body color",
				SequenceOrder: 0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			var parsed AddMaterialRequest
			err = json.Unmarshal(data, &parsed)
			hasErr := err != nil
			if hasErr != tt.wantErr {
				t.Errorf("JSON unmarshal error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if parsed.MaterialType != tt.request.MaterialType {
					t.Errorf("MaterialType = %s, want %s", parsed.MaterialType, tt.request.MaterialType)
				}
				if parsed.WeightGrams != tt.request.WeightGrams {
					t.Errorf("WeightGrams = %f, want %f", parsed.WeightGrams, tt.request.WeightGrams)
				}
			}
		})
	}
}

func TestRespondJSON(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		data       interface{}
		wantStatus int
		wantBody   string
	}{
		{
			name:       "success with data",
			status:     http.StatusOK,
			data:       map[string]string{"message": "ok"},
			wantStatus: http.StatusOK,
			wantBody:   `{"message":"ok"}`,
		},
		{
			name:       "created with data",
			status:     http.StatusCreated,
			data:       map[string]int{"id": 123},
			wantStatus: http.StatusCreated,
			wantBody:   `{"id":123}`,
		},
		{
			name:       "nil data",
			status:     http.StatusOK,
			data:       nil,
			wantStatus: http.StatusOK,
			wantBody:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			respondJSON(w, tt.status, tt.data)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantBody != "" {
				body := bytes.TrimSpace(w.Body.Bytes())
				if string(body) != tt.wantBody {
					t.Errorf("Body = %s, want %s", body, tt.wantBody)
				}
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Content-Type = %s, want application/json", contentType)
			}
		})
	}
}

func TestRespondError(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		message    string
		wantStatus int
	}{
		{
			name:       "bad request",
			status:     http.StatusBadRequest,
			message:    "invalid input",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "not found",
			status:     http.StatusNotFound,
			message:    "resource not found",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "internal error",
			status:     http.StatusInternalServerError,
			message:    "something went wrong",
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			respondError(w, tt.status, tt.message)

			if w.Code != tt.wantStatus {
				t.Errorf("Status = %d, want %d", w.Code, tt.wantStatus)
			}

			var response map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if response["error"] != tt.message {
				t.Errorf("Error message = %s, want %s", response["error"], tt.message)
			}
		})
	}
}

func TestPrinterConstraintsJSON(t *testing.T) {
	tests := []struct {
		name        string
		constraints model.PrinterConstraints
	}{
		{
			name: "empty constraints",
			constraints: model.PrinterConstraints{
				RequiresEnclosure: false,
				RequiresAMS:       false,
			},
		},
		{
			name: "with bed size",
			constraints: model.PrinterConstraints{
				MinBedSize: &model.BuildVolume{X: 256, Y: 256, Z: 256},
			},
		},
		{
			name: "with nozzle diameters",
			constraints: model.PrinterConstraints{
				NozzleDiameters: []float64{0.4, 0.6},
			},
		},
		{
			name: "full constraints",
			constraints: model.PrinterConstraints{
				MinBedSize:        &model.BuildVolume{X: 300, Y: 300, Z: 300},
				NozzleDiameters:   []float64{0.4},
				RequiresEnclosure: true,
				RequiresAMS:       true,
				PrinterTags:       []string{"production", "enclosed"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test JSON round-trip
			data, err := json.Marshal(tt.constraints)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			var parsed model.PrinterConstraints
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			// Verify key fields
			if parsed.RequiresEnclosure != tt.constraints.RequiresEnclosure {
				t.Errorf("RequiresEnclosure = %v, want %v", parsed.RequiresEnclosure, tt.constraints.RequiresEnclosure)
			}
			if parsed.RequiresAMS != tt.constraints.RequiresAMS {
				t.Errorf("RequiresAMS = %v, want %v", parsed.RequiresAMS, tt.constraints.RequiresAMS)
			}
		})
	}
}

func TestColorSpecJSON(t *testing.T) {
	tests := []struct {
		name      string
		colorSpec model.ColorSpec
	}{
		{
			name: "any mode",
			colorSpec: model.ColorSpec{
				Mode: "any",
			},
		},
		{
			name: "exact with name",
			colorSpec: model.ColorSpec{
				Mode: "exact",
				Name: "White",
			},
		},
		{
			name: "exact with hex",
			colorSpec: model.ColorSpec{
				Mode: "exact",
				Hex:  "#FF5733",
			},
		},
		{
			name: "exact with both",
			colorSpec: model.ColorSpec{
				Mode: "exact",
				Name: "Red",
				Hex:  "#FF0000",
			},
		},
		{
			name: "category mode",
			colorSpec: model.ColorSpec{
				Mode: "category",
				Name: "warm",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.colorSpec)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			var parsed model.ColorSpec
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			if parsed.Mode != tt.colorSpec.Mode {
				t.Errorf("Mode = %s, want %s", parsed.Mode, tt.colorSpec.Mode)
			}
			if parsed.Name != tt.colorSpec.Name {
				t.Errorf("Name = %s, want %s", parsed.Name, tt.colorSpec.Name)
			}
			if parsed.Hex != tt.colorSpec.Hex {
				t.Errorf("Hex = %s, want %s", parsed.Hex, tt.colorSpec.Hex)
			}
		})
	}
}

func TestRecipeCostEstimateJSON(t *testing.T) {
	estimate := model.RecipeCostEstimate{
		MaterialCostCents:  250,
		TimeCostCents:      500,
		TotalCostCents:     750,
		EstimatedPrintTime: 3600,
		HourlyRateCents:    500,
		MaterialBreakdown: []model.RecipeMaterialCostBreakdown{
			{
				MaterialType: "pla",
				WeightGrams:  100,
				CostCents:    250,
				ColorName:    "White",
			},
		},
	}

	data, err := json.Marshal(estimate)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed model.RecipeCostEstimate
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if parsed.TotalCostCents != estimate.TotalCostCents {
		t.Errorf("TotalCostCents = %d, want %d", parsed.TotalCostCents, estimate.TotalCostCents)
	}
	if parsed.MaterialCostCents != estimate.MaterialCostCents {
		t.Errorf("MaterialCostCents = %d, want %d", parsed.MaterialCostCents, estimate.MaterialCostCents)
	}
	if parsed.TimeCostCents != estimate.TimeCostCents {
		t.Errorf("TimeCostCents = %d, want %d", parsed.TimeCostCents, estimate.TimeCostCents)
	}
	if len(parsed.MaterialBreakdown) != len(estimate.MaterialBreakdown) {
		t.Errorf("MaterialBreakdown length = %d, want %d", len(parsed.MaterialBreakdown), len(estimate.MaterialBreakdown))
	}
}

func TestRecipeMaterialJSON(t *testing.T) {
	material := model.RecipeMaterial{
		MaterialType:  "petg",
		WeightGrams:   150.5,
		AMSPosition:   intPtr(3),
		SequenceOrder: 1,
		Notes:         "Support material",
		ColorSpec: &model.ColorSpec{
			Mode: "exact",
			Hex:  "#000000",
			Name: "Black",
		},
	}

	data, err := json.Marshal(material)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var parsed model.RecipeMaterial
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if parsed.MaterialType != material.MaterialType {
		t.Errorf("MaterialType = %s, want %s", parsed.MaterialType, material.MaterialType)
	}
	if parsed.WeightGrams != material.WeightGrams {
		t.Errorf("WeightGrams = %f, want %f", parsed.WeightGrams, material.WeightGrams)
	}
	if parsed.AMSPosition == nil || *parsed.AMSPosition != *material.AMSPosition {
		t.Errorf("AMSPosition mismatch")
	}
	if parsed.ColorSpec == nil || parsed.ColorSpec.Mode != "exact" {
		t.Errorf("ColorSpec not properly parsed")
	}
}

func intPtr(i int) *int {
	return &i
}
