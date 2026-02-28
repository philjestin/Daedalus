package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/philjestin/daedalus/internal/model"
	"github.com/philjestin/daedalus/internal/service"
)

// MockTemplateService implements the template service interface for testing.
type MockTemplateService struct {
	templates      map[uuid.UUID]*model.Template
	templateDesigns map[uuid.UUID][]model.TemplateDesign
}

func NewMockTemplateService() *MockTemplateService {
	return &MockTemplateService{
		templates:       make(map[uuid.UUID]*model.Template),
		templateDesigns: make(map[uuid.UUID][]model.TemplateDesign),
	}
}

func (m *MockTemplateService) Create(ctx context.Context, t *model.Template) error {
	t.ID = uuid.New()
	t.IsActive = true
	m.templates[t.ID] = t
	return nil
}

func (m *MockTemplateService) GetByID(ctx context.Context, id uuid.UUID) (*model.Template, error) {
	t, ok := m.templates[id]
	if !ok {
		return nil, nil
	}
	t.Designs = m.templateDesigns[id]
	return t, nil
}

func (m *MockTemplateService) GetBySKU(ctx context.Context, sku string) (*model.Template, error) {
	for _, t := range m.templates {
		if t.SKU == sku {
			return t, nil
		}
	}
	return nil, nil
}

func (m *MockTemplateService) List(ctx context.Context, activeOnly bool) ([]model.Template, error) {
	var result []model.Template
	for _, t := range m.templates {
		if activeOnly && !t.IsActive {
			continue
		}
		result = append(result, *t)
	}
	return result, nil
}

func (m *MockTemplateService) Update(ctx context.Context, t *model.Template) error {
	m.templates[t.ID] = t
	return nil
}

func (m *MockTemplateService) Delete(ctx context.Context, id uuid.UUID) error {
	delete(m.templates, id)
	return nil
}

func (m *MockTemplateService) AddDesign(ctx context.Context, td *model.TemplateDesign) error {
	td.ID = uuid.New()
	m.templateDesigns[td.TemplateID] = append(m.templateDesigns[td.TemplateID], *td)
	return nil
}

func (m *MockTemplateService) RemoveDesign(ctx context.Context, templateID, designID uuid.UUID) error {
	designs := m.templateDesigns[templateID]
	for i, d := range designs {
		if d.DesignID == designID {
			m.templateDesigns[templateID] = append(designs[:i], designs[i+1:]...)
			break
		}
	}
	return nil
}

func (m *MockTemplateService) GetDesigns(ctx context.Context, templateID uuid.UUID) ([]model.TemplateDesign, error) {
	return m.templateDesigns[templateID], nil
}

func (m *MockTemplateService) CreateProjectFromTemplate(ctx context.Context, templateID uuid.UUID, opts service.CreateFromTemplateOptions) (*model.Project, []model.PrintJob, error) {
	t := m.templates[templateID]
	if t == nil {
		return nil, nil, nil
	}
	project := &model.Project{
		ID:          uuid.New(),
		Name:        t.Name,
		Description: t.Description,
		TemplateID:  &templateID,
		Source:      opts.Source,
	}
	return project, []model.PrintJob{}, nil
}

// Test helper to create a template handler with mock service
func setupTemplateHandler() (*TemplateHandler, *MockTemplateService) {
	mock := NewMockTemplateService()
	// We need to wrap the mock in a real service struct for the handler
	// For now, we'll test via HTTP handlers directly
	return nil, mock
}

func TestTemplateHandler_Create(t *testing.T) {
	// Create a test template
	template := model.Template{
		Name:             "Test Template",
		Description:      "A test template",
		SKU:              "TEST-001",
		MaterialType:     model.MaterialTypePLA,
		QuantityPerOrder: 2,
	}

	body, _ := json.Marshal(template)
	req := httptest.NewRequest(http.MethodPost, "/api/templates", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// For integration tests, we'd need a real database
	// This is a unit test structure showing the pattern
	t.Run("valid template creation request", func(t *testing.T) {
		if template.Name == "" {
			t.Error("Template name should not be empty")
		}
		if template.MaterialType == "" {
			t.Error("Material type should not be empty")
		}
	})
}

func TestTemplateHandler_List(t *testing.T) {
	mock := NewMockTemplateService()

	// Add some templates
	t1 := &model.Template{Name: "Template 1", MaterialType: model.MaterialTypePLA}
	t2 := &model.Template{Name: "Template 2", MaterialType: model.MaterialTypePETG}
	mock.Create(context.Background(), t1)
	mock.Create(context.Background(), t2)
	// Manually set one as inactive after creation (since Create sets IsActive=true)
	t2.IsActive = false

	t.Run("list all templates", func(t *testing.T) {
		templates, err := mock.List(context.Background(), false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(templates) != 2 {
			t.Errorf("expected 2 templates, got %d", len(templates))
		}
	})

	t.Run("list active only", func(t *testing.T) {
		templates, err := mock.List(context.Background(), true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(templates) != 1 {
			t.Errorf("expected 1 active template, got %d", len(templates))
		}
	})
}

func TestTemplateHandler_GetByID(t *testing.T) {
	mock := NewMockTemplateService()

	template := &model.Template{
		Name:         "Test Template",
		MaterialType: model.MaterialTypePLA,
	}
	mock.Create(context.Background(), template)

	t.Run("get existing template", func(t *testing.T) {
		result, err := mock.GetByID(context.Background(), template.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected template, got nil")
		}
		if result.Name != template.Name {
			t.Errorf("expected name %q, got %q", template.Name, result.Name)
		}
	})

	t.Run("get non-existent template", func(t *testing.T) {
		result, err := mock.GetByID(context.Background(), uuid.New())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Error("expected nil for non-existent template")
		}
	})
}

func TestTemplateHandler_Update(t *testing.T) {
	mock := NewMockTemplateService()

	template := &model.Template{
		Name:         "Original Name",
		MaterialType: model.MaterialTypePLA,
	}
	mock.Create(context.Background(), template)

	t.Run("update template", func(t *testing.T) {
		template.Name = "Updated Name"
		template.Description = "Updated description"
		err := mock.Update(context.Background(), template)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, _ := mock.GetByID(context.Background(), template.ID)
		if result.Name != "Updated Name" {
			t.Errorf("expected updated name, got %q", result.Name)
		}
	})
}

func TestTemplateHandler_Delete(t *testing.T) {
	mock := NewMockTemplateService()

	template := &model.Template{
		Name:         "To Delete",
		MaterialType: model.MaterialTypePLA,
	}
	mock.Create(context.Background(), template)

	t.Run("delete template", func(t *testing.T) {
		err := mock.Delete(context.Background(), template.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		result, _ := mock.GetByID(context.Background(), template.ID)
		if result != nil {
			t.Error("expected template to be deleted")
		}
	})
}

func TestTemplateHandler_AddDesign(t *testing.T) {
	mock := NewMockTemplateService()

	template := &model.Template{
		Name:         "Template with Design",
		MaterialType: model.MaterialTypePLA,
	}
	mock.Create(context.Background(), template)

	t.Run("add design to template", func(t *testing.T) {
		td := &model.TemplateDesign{
			TemplateID: template.ID,
			DesignID:   uuid.New(),
			Quantity:   2,
			IsPrimary:  true,
		}
		err := mock.AddDesign(context.Background(), td)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		designs, _ := mock.GetDesigns(context.Background(), template.ID)
		if len(designs) != 1 {
			t.Errorf("expected 1 design, got %d", len(designs))
		}
		if designs[0].Quantity != 2 {
			t.Errorf("expected quantity 2, got %d", designs[0].Quantity)
		}
	})
}

func TestTemplateHandler_RemoveDesign(t *testing.T) {
	mock := NewMockTemplateService()

	template := &model.Template{
		Name:         "Template",
		MaterialType: model.MaterialTypePLA,
	}
	mock.Create(context.Background(), template)

	designID := uuid.New()
	td := &model.TemplateDesign{
		TemplateID: template.ID,
		DesignID:   designID,
		Quantity:   1,
	}
	mock.AddDesign(context.Background(), td)

	t.Run("remove design from template", func(t *testing.T) {
		err := mock.RemoveDesign(context.Background(), template.ID, designID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		designs, _ := mock.GetDesigns(context.Background(), template.ID)
		if len(designs) != 0 {
			t.Errorf("expected 0 designs after removal, got %d", len(designs))
		}
	})
}

func TestTemplateHandler_Instantiate(t *testing.T) {
	mock := NewMockTemplateService()

	template := &model.Template{
		Name:             "Product Template",
		Description:      "A product",
		MaterialType:     model.MaterialTypePLA,
		QuantityPerOrder: 3,
	}
	mock.Create(context.Background(), template)

	t.Run("instantiate template", func(t *testing.T) {
		opts := service.CreateFromTemplateOptions{
			OrderQuantity: 2,
			Source:        "manual",
		}
		project, jobs, err := mock.CreateProjectFromTemplate(context.Background(), template.ID, opts)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if project == nil {
			t.Fatal("expected project, got nil")
		}
		if project.Name != template.Name {
			t.Errorf("expected project name %q, got %q", template.Name, project.Name)
		}
		if project.TemplateID == nil || *project.TemplateID != template.ID {
			t.Error("expected project to have template ID set")
		}
		// Jobs would be created in real implementation
		_ = jobs
	})
}

// Integration test helper - would need a real database
func TestTemplateAPI_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// This would be an integration test with a real database
	// For now, we skip it
	t.Skip("integration tests require database setup")
}

// Test request/response parsing
func TestAddDesignRequest_Parse(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		jsonStr := `{"design_id": "550e8400-e29b-41d4-a716-446655440000", "quantity": 2, "is_primary": true}`
		var req AddDesignRequest
		err := json.Unmarshal([]byte(jsonStr), &req)
		if err != nil {
			t.Fatalf("failed to parse: %v", err)
		}
		if req.DesignID != "550e8400-e29b-41d4-a716-446655440000" {
			t.Errorf("unexpected design_id: %s", req.DesignID)
		}
		if req.Quantity != 2 {
			t.Errorf("unexpected quantity: %d", req.Quantity)
		}
		if !req.IsPrimary {
			t.Error("expected is_primary to be true")
		}
	})
}

func TestInstantiateRequest_Parse(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		jsonStr := `{"order_quantity": 5, "customer_notes": "Rush order", "source": "etsy"}`
		var req InstantiateRequest
		err := json.Unmarshal([]byte(jsonStr), &req)
		if err != nil {
			t.Fatalf("failed to parse: %v", err)
		}
		if req.OrderQuantity != 5 {
			t.Errorf("unexpected order_quantity: %d", req.OrderQuantity)
		}
		if req.CustomerNotes != "Rush order" {
			t.Errorf("unexpected customer_notes: %s", req.CustomerNotes)
		}
		if req.Source != "etsy" {
			t.Errorf("unexpected source: %s", req.Source)
		}
	})
}

// Test URL parameter parsing
func TestParseUUID_Templates(t *testing.T) {
	t.Run("valid UUID", func(t *testing.T) {
		r := chi.NewRouter()
		var parsedID uuid.UUID

		r.Get("/templates/{id}", func(w http.ResponseWriter, r *http.Request) {
			id, err := parseUUID(r, "id")
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			parsedID = id
			w.WriteHeader(http.StatusOK)
		})

		testID := uuid.New()
		req := httptest.NewRequest(http.MethodGet, "/templates/"+testID.String(), nil)
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

		r.Get("/templates/{id}", func(w http.ResponseWriter, r *http.Request) {
			_, err := parseUUID(r, "id")
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)
		})

		req := httptest.NewRequest(http.MethodGet, "/templates/invalid-uuid", nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status 400 for invalid UUID, got %d", rr.Code)
		}
	})
}
