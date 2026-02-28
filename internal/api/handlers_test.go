package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/philjestin/daedalus/internal/model"
)

// --- Part creation (JSON path) ---

func TestPartCreate_JSON(t *testing.T) {
	env := newTestEnv(t)

	// Create a project first
	project := &model.Project{Name: "Test Project"}
	if err := env.services.Projects.Create(context.Background(), project); err != nil {
		t.Fatalf("create project: %v", err)
	}

	body, _ := json.Marshal(map[string]interface{}{
		"name":        "Bracket",
		"description": "A mounting bracket",
		"quantity":    2,
	})

	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/projects/%s/parts", project.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var part model.Part
	if err := json.NewDecoder(rr.Body).Decode(&part); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if part.Name != "Bracket" {
		t.Errorf("expected name 'Bracket', got %q", part.Name)
	}
	if part.Quantity != 2 {
		t.Errorf("expected quantity 2, got %d", part.Quantity)
	}
	if part.ProjectID != project.ID {
		t.Errorf("expected project ID %s, got %s", project.ID, part.ProjectID)
	}
}

func TestPartCreate_JSON_InvalidProjectID(t *testing.T) {
	env := newTestEnv(t)

	body, _ := json.Marshal(map[string]interface{}{"name": "Test"})
	req := httptest.NewRequest(http.MethodPost, "/api/projects/not-a-uuid/parts", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid UUID, got %d", rr.Code)
	}
}

func TestPartCreate_JSON_MissingName(t *testing.T) {
	env := newTestEnv(t)

	project := &model.Project{Name: "Test Project"}
	env.services.Projects.Create(context.Background(), project)

	body, _ := json.Marshal(map[string]interface{}{"description": "no name"})
	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/projects/%s/parts", project.ID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing name, got %d: %s", rr.Code, rr.Body.String())
	}
}

// --- Part creation (multipart path, no file) ---

func TestPartCreate_Multipart_NoFile(t *testing.T) {
	env := newTestEnv(t)

	project := &model.Project{Name: "Test Project"}
	env.services.Projects.Create(context.Background(), project)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	w.WriteField("name", "Multipart Part")
	w.WriteField("description", "Created via multipart")
	w.WriteField("quantity", "3")
	w.Close()

	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/projects/%s/parts", project.ID), &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	// Without a file, the response is just a Part
	var part model.Part
	if err := json.NewDecoder(rr.Body).Decode(&part); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if part.Name != "Multipart Part" {
		t.Errorf("expected name 'Multipart Part', got %q", part.Name)
	}
	if part.Quantity != 3 {
		t.Errorf("expected quantity 3, got %d", part.Quantity)
	}
}

// --- Part creation (multipart path, with file) ---

func TestPartCreate_Multipart_WithFile(t *testing.T) {
	env := newTestEnv(t)

	project := &model.Project{Name: "Test Project"}
	env.services.Projects.Create(context.Background(), project)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	w.WriteField("name", "Part With Design")
	w.WriteField("description", "Has 3MF attached")
	w.WriteField("quantity", "1")
	w.WriteField("notes", "Initial version")

	fw, _ := w.CreateFormFile("file", "bracket.3mf")
	fw.Write([]byte("fake 3mf content"))
	w.Close()

	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/projects/%s/parts", project.ID), &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	// Response should have both "part" and "design" keys
	var resp map[string]json.RawMessage
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if _, ok := resp["part"]; !ok {
		t.Error("response missing 'part' key")
	}
	if _, ok := resp["design"]; !ok {
		t.Error("response missing 'design' key")
	}

	// Parse the design to verify it was created correctly
	var design model.Design
	if err := json.Unmarshal(resp["design"], &design); err != nil {
		t.Fatalf("decode design: %v", err)
	}
	if design.FileName != "bracket.3mf" {
		t.Errorf("expected filename 'bracket.3mf', got %q", design.FileName)
	}
	if design.FileType != model.FileType3MF {
		t.Errorf("expected file type '3mf', got %q", design.FileType)
	}
	if design.Notes != "Initial version" {
		t.Errorf("expected notes 'Initial version', got %q", design.Notes)
	}

	// Verify the part was also created correctly
	var part model.Part
	json.Unmarshal(resp["part"], &part)
	if part.Name != "Part With Design" {
		t.Errorf("expected part name 'Part With Design', got %q", part.Name)
	}
}

func TestPartCreate_Multipart_WithSTLFile(t *testing.T) {
	env := newTestEnv(t)

	project := &model.Project{Name: "Test Project"}
	env.services.Projects.Create(context.Background(), project)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	w.WriteField("name", "STL Part")
	w.WriteField("quantity", "1")

	fw, _ := w.CreateFormFile("file", "model.stl")
	fw.Write([]byte("fake stl content"))
	w.Close()

	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/projects/%s/parts", project.ID), &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]json.RawMessage
	json.NewDecoder(rr.Body).Decode(&resp)

	var design model.Design
	json.Unmarshal(resp["design"], &design)
	if design.FileType != model.FileTypeSTL {
		t.Errorf("expected file type 'stl', got %q", design.FileType)
	}
}

func TestPartCreate_Multipart_DefaultQuantity(t *testing.T) {
	env := newTestEnv(t)

	project := &model.Project{Name: "Test Project"}
	env.services.Projects.Create(context.Background(), project)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	w.WriteField("name", "Default Qty Part")
	// No quantity field — should default to 1
	w.Close()

	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/projects/%s/parts", project.ID), &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var part model.Part
	json.NewDecoder(rr.Body).Decode(&part)
	if part.Quantity != 1 {
		t.Errorf("expected default quantity 1, got %d", part.Quantity)
	}
}

// --- OpenExternal handler ---

func TestDesignOpenExternal_InvalidID(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/api/designs/bad-uuid/open-external", nil)
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid UUID, got %d", rr.Code)
	}
}

func TestDesignOpenExternal_DesignNotFound(t *testing.T) {
	env := newTestEnv(t)

	fakeID := uuid.New()
	body, _ := json.Marshal(map[string]string{"app": "BambuStudio"})
	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/designs/%s/open-external", fakeID), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	// Should return 404 or 500 (design doesn't exist)
	if rr.Code == http.StatusOK {
		t.Errorf("expected non-200 for missing design, got %d", rr.Code)
	}
}

func TestDesignOpenExternal_EmptyBody(t *testing.T) {
	env := newTestEnv(t)

	fakeID := uuid.New()
	req := httptest.NewRequest(http.MethodPost,
		fmt.Sprintf("/api/designs/%s/open-external", fakeID), nil)
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	// Should not panic — empty body is allowed
	if rr.Code == http.StatusInternalServerError {
		t.Errorf("should handle empty body gracefully, got 500: %s", rr.Body.String())
	}
}

// --- Bambu Cloud handler ---

func TestBambuCloudLogin_BadJSON(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/api/bambu-cloud/login",
		bytes.NewReader([]byte(`not json`)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad JSON, got %d", rr.Code)
	}
}

func TestBambuCloudLogin_MissingFields(t *testing.T) {
	env := newTestEnv(t)

	body, _ := json.Marshal(map[string]string{"email": "test@example.com"})
	req := httptest.NewRequest(http.MethodPost, "/api/bambu-cloud/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing password, got %d", rr.Code)
	}
}

func TestBambuCloudAddDevice_BadJSON(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/api/bambu-cloud/devices/add",
		bytes.NewReader([]byte(`not json`)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad JSON, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestBambuCloudAddDevice_EmptyDevID(t *testing.T) {
	env := newTestEnv(t)

	body, _ := json.Marshal(map[string]string{"dev_id": ""})
	req := httptest.NewRequest(http.MethodPost, "/api/bambu-cloud/devices/add", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty dev_id, got %d: %s", rr.Code, rr.Body.String())
	}
}

// TestPrinterCreate_ViaAPI verifies that printer creation works end-to-end
// through the database. This is a regression test for the cost_per_hour_cents
// column that was missing from the schema.
func TestPrinterCreate_ViaAPI(t *testing.T) {
	env := newTestEnv(t)

	body, _ := json.Marshal(map[string]interface{}{
		"name":            "Test Printer",
		"model":           "A1 Mini",
		"manufacturer":    "Bambu Lab",
		"connection_type": "bambu_cloud",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/printers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var printer model.Printer
	if err := json.NewDecoder(rr.Body).Decode(&printer); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if printer.Name != "Test Printer" {
		t.Errorf("expected name 'Test Printer', got %q", printer.Name)
	}
	if printer.ID == (uuid.UUID{}) {
		t.Error("expected non-zero printer ID")
	}
}

// --- Printer CRUD (full lifecycle) ---

// TestPrinterCreate_AllFields verifies every field round-trips through
// create → get correctly, catching any schema / scan mismatch.
func TestPrinterCreate_AllFields(t *testing.T) {
	env := newTestEnv(t)

	body, _ := json.Marshal(map[string]interface{}{
		"name":                "Bambu A1 Mini",
		"model":               "A1 Mini",
		"manufacturer":        "Bambu Lab",
		"connection_type":     "bambu_cloud",
		"connection_uri":      "u_12345",
		"api_key":             "token_abc",
		"serial_number":       "00M09A350100123",
		"nozzle_diameter":     0.4,
		"location":            "Shelf 2",
		"notes":               "Production printer",
		"cost_per_hour_cents": 150,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/printers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var created model.Printer
	json.NewDecoder(rr.Body).Decode(&created)

	// Now GET the same printer and verify all fields survive the round-trip.
	req2 := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/printers/%s", created.ID), nil)
	rr2 := httptest.NewRecorder()
	env.handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d: %s", rr2.Code, rr2.Body.String())
	}

	var fetched model.Printer
	json.NewDecoder(rr2.Body).Decode(&fetched)

	if fetched.Name != "Bambu A1 Mini" {
		t.Errorf("name: got %q", fetched.Name)
	}
	if fetched.Model != "A1 Mini" {
		t.Errorf("model: got %q", fetched.Model)
	}
	if fetched.Manufacturer != "Bambu Lab" {
		t.Errorf("manufacturer: got %q", fetched.Manufacturer)
	}
	if fetched.ConnectionType != "bambu_cloud" {
		t.Errorf("connection_type: got %q", fetched.ConnectionType)
	}
	if fetched.ConnectionURI != "u_12345" {
		t.Errorf("connection_uri: got %q", fetched.ConnectionURI)
	}
	if fetched.APIKey != "token_abc" {
		t.Errorf("api_key: got %q", fetched.APIKey)
	}
	if fetched.SerialNumber != "00M09A350100123" {
		t.Errorf("serial_number: got %q", fetched.SerialNumber)
	}
	if fetched.NozzleDiameter != 0.4 {
		t.Errorf("nozzle_diameter: got %f", fetched.NozzleDiameter)
	}
	if fetched.Location != "Shelf 2" {
		t.Errorf("location: got %q", fetched.Location)
	}
	if fetched.Notes != "Production printer" {
		t.Errorf("notes: got %q", fetched.Notes)
	}
	if fetched.CostPerHourCents != 150 {
		t.Errorf("cost_per_hour_cents: got %d", fetched.CostPerHourCents)
	}
	if fetched.Status != model.PrinterStatusOffline {
		t.Errorf("expected default status 'offline', got %q", fetched.Status)
	}
}

// TestPrinterList verifies listing returns all created printers.
func TestPrinterList(t *testing.T) {
	env := newTestEnv(t)

	// Create two printers
	for _, name := range []string{"Printer A", "Printer B"} {
		body, _ := json.Marshal(map[string]interface{}{
			"name":            name,
			"connection_type": "manual",
		})
		req := httptest.NewRequest(http.MethodPost, "/api/printers", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		env.handler.ServeHTTP(rr, req)
		if rr.Code != http.StatusCreated {
			t.Fatalf("create %s: got %d: %s", name, rr.Code, rr.Body.String())
		}
	}

	// List
	req := httptest.NewRequest(http.MethodGet, "/api/printers", nil)
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var printers []model.Printer
	json.NewDecoder(rr.Body).Decode(&printers)
	if len(printers) != 2 {
		t.Errorf("expected 2 printers, got %d", len(printers))
	}
}

// TestPrinterUpdate verifies PATCH updates fields and round-trips correctly.
func TestPrinterUpdate(t *testing.T) {
	env := newTestEnv(t)

	// Create
	body, _ := json.Marshal(map[string]interface{}{
		"name":            "Original",
		"connection_type": "manual",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/printers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)
	var created model.Printer
	json.NewDecoder(rr.Body).Decode(&created)

	// Update
	patchBody, _ := json.Marshal(map[string]interface{}{
		"name":                "Updated",
		"cost_per_hour_cents": 200,
		"location":            "Rack 3",
	})
	req2 := httptest.NewRequest(http.MethodPatch,
		fmt.Sprintf("/api/printers/%s", created.ID), bytes.NewReader(patchBody))
	req2.Header.Set("Content-Type", "application/json")
	rr2 := httptest.NewRecorder()
	env.handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d: %s", rr2.Code, rr2.Body.String())
	}

	// GET to verify
	req3 := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/printers/%s", created.ID), nil)
	rr3 := httptest.NewRecorder()
	env.handler.ServeHTTP(rr3, req3)
	var updated model.Printer
	json.NewDecoder(rr3.Body).Decode(&updated)

	if updated.Name != "Updated" {
		t.Errorf("expected name 'Updated', got %q", updated.Name)
	}
	if updated.CostPerHourCents != 200 {
		t.Errorf("expected cost 200, got %d", updated.CostPerHourCents)
	}
	if updated.Location != "Rack 3" {
		t.Errorf("expected location 'Rack 3', got %q", updated.Location)
	}
}

// TestPrinterDelete verifies deletion and confirms the printer is gone.
func TestPrinterDelete(t *testing.T) {
	env := newTestEnv(t)

	// Create
	body, _ := json.Marshal(map[string]interface{}{
		"name":            "To Delete",
		"connection_type": "manual",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/printers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)
	var created model.Printer
	json.NewDecoder(rr.Body).Decode(&created)

	// Delete
	req2 := httptest.NewRequest(http.MethodDelete,
		fmt.Sprintf("/api/printers/%s", created.ID), nil)
	rr2 := httptest.NewRecorder()
	env.handler.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusNoContent && rr2.Code != http.StatusOK {
		t.Fatalf("delete: expected 204 or 200, got %d: %s", rr2.Code, rr2.Body.String())
	}

	// Confirm it's gone
	req3 := httptest.NewRequest(http.MethodGet,
		fmt.Sprintf("/api/printers/%s", created.ID), nil)
	rr3 := httptest.NewRecorder()
	env.handler.ServeHTTP(rr3, req3)

	if rr3.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", rr3.Code)
	}
}

// TestPrinterCreate_BambuCloudType specifically tests creating a printer
// with bambu_cloud connection type and all the fields AddDevice would set.
// This is the exact code path that was broken by the missing schema column.
func TestPrinterCreate_BambuCloudType(t *testing.T) {
	env := newTestEnv(t)

	body, _ := json.Marshal(map[string]interface{}{
		"name":            "Bambu Lab P1S",
		"model":           "P1S",
		"manufacturer":    "Bambu Lab",
		"connection_type": "bambu_cloud",
		"connection_uri":  "u_998877",
		"api_key":         "eyJhbGciOiJSUzI1NiJ9.fake",
		"serial_number":   "00W00A350700234",
		"nozzle_diameter": 0.4,
	})

	req := httptest.NewRequest(http.MethodPost, "/api/printers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var printer model.Printer
	json.NewDecoder(rr.Body).Decode(&printer)

	if printer.ConnectionType != "bambu_cloud" {
		t.Errorf("connection_type: got %q", printer.ConnectionType)
	}
	if printer.SerialNumber != "00W00A350700234" {
		t.Errorf("serial_number: got %q", printer.SerialNumber)
	}
	if printer.APIKey != "eyJhbGciOiJSUzI1NiJ9.fake" {
		t.Errorf("api_key not persisted correctly")
	}

	// Verify it appears in the list
	req2 := httptest.NewRequest(http.MethodGet, "/api/printers", nil)
	rr2 := httptest.NewRecorder()
	env.handler.ServeHTTP(rr2, req2)

	var printers []model.Printer
	json.NewDecoder(rr2.Body).Decode(&printers)

	found := false
	for _, p := range printers {
		if p.ID == printer.ID {
			found = true
			break
		}
	}
	if !found {
		t.Error("created printer not found in list")
	}
}

// TestPrinterCreate_MultipleBambuCloud creates multiple bambu_cloud printers
// to verify no unique constraint issues (since dev_id is stored as serial_number
// and there's no unique constraint).
func TestPrinterCreate_MultipleBambuCloud(t *testing.T) {
	env := newTestEnv(t)

	for i, serial := range []string{"DEV001", "DEV002", "DEV003"} {
		body, _ := json.Marshal(map[string]interface{}{
			"name":            fmt.Sprintf("Printer %d", i+1),
			"connection_type": "bambu_cloud",
			"serial_number":   serial,
		})
		req := httptest.NewRequest(http.MethodPost, "/api/printers", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		env.handler.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Fatalf("printer %d: expected 201, got %d: %s", i+1, rr.Code, rr.Body.String())
		}
	}

	// Verify all 3 exist
	req := httptest.NewRequest(http.MethodGet, "/api/printers", nil)
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	var printers []model.Printer
	json.NewDecoder(rr.Body).Decode(&printers)
	if len(printers) != 3 {
		t.Errorf("expected 3 printers, got %d", len(printers))
	}
}

// --- Bambu Cloud handler (continued) ---

func TestBambuCloudStatus_NotAuthenticated(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/bambu-cloud/status", nil)
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&resp)
	if connected, ok := resp["connected"].(bool); !ok || connected {
		t.Errorf("expected connected=false for fresh db, got %v", resp["connected"])
	}
}

func TestBambuCloudDevices_NotAuthenticated(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/bambu-cloud/devices", nil)
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	// Should return an error since we're not authenticated
	if rr.Code == http.StatusOK {
		t.Error("expected non-200 when not authenticated")
	}
}

// --- Receipt upload ---

func TestExpenseUploadReceipt_Success(t *testing.T) {
	env := newTestEnv(t)

	// Create a multipart form with a fake receipt file
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("file", "receipt.pdf")
	if err != nil {
		t.Fatal(err)
	}
	part.Write([]byte("%PDF-1.4 fake pdf content"))
	w.Close()

	req := httptest.NewRequest("POST", "/api/expenses/receipt", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var expense map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&expense)

	if expense["id"] == nil || expense["id"] == "" {
		t.Error("expense should have an ID")
	}
	if expense["status"] != "pending" {
		t.Errorf("expected status pending, got %v", expense["status"])
	}
	if expense["receipt_file_path"] == nil || expense["receipt_file_path"] == "" {
		t.Error("expense should have receipt_file_path")
	}
}

func TestExpenseUploadReceipt_NoFile(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest("POST", "/api/expenses/receipt", nil)
	req.Header.Set("Content-Type", "multipart/form-data; boundary=xxx")
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code == http.StatusCreated {
		t.Error("expected error when no file is provided")
	}
}

func TestExpenseList_Empty(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest("GET", "/api/expenses", nil)
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var expenses []interface{}
	json.NewDecoder(rr.Body).Decode(&expenses)

	if len(expenses) != 0 {
		t.Errorf("expected empty list, got %d", len(expenses))
	}
}

func TestExpenseGet_NotFound(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest("GET", "/api/expenses/"+uuid.New().String(), nil)
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rr.Code)
	}
}

func TestExpenseDelete(t *testing.T) {
	env := newTestEnv(t)

	// Upload first
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("file", "test.jpg")
	part.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0})
	w.Close()

	req := httptest.NewRequest("POST", "/api/expenses/receipt", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("upload: expected 201, got %d", rr.Code)
	}

	var expense map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&expense)
	id := expense["id"].(string)

	// Delete it
	req = httptest.NewRequest("DELETE", "/api/expenses/"+id, nil)
	rr = httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("delete: expected 204, got %d: %s", rr.Code, rr.Body.String())
	}

	// Verify it's gone
	req = httptest.NewRequest("GET", "/api/expenses/"+id, nil)
	rr = httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("after delete: expected 404, got %d", rr.Code)
	}
}

func TestExpenseListAfterUpload(t *testing.T) {
	env := newTestEnv(t)

	// Upload a receipt
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("file", "receipt.png")
	part.Write([]byte{0x89, 'P', 'N', 'G'})
	w.Close()

	req := httptest.NewRequest("POST", "/api/expenses/receipt", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("upload: expected 201, got %d", rr.Code)
	}

	// List should return 1
	req = httptest.NewRequest("GET", "/api/expenses", nil)
	rr = httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", rr.Code)
	}

	var expenses []interface{}
	json.NewDecoder(rr.Body).Decode(&expenses)

	if len(expenses) != 1 {
		t.Errorf("expected 1 expense, got %d", len(expenses))
	}
}

func TestExpenseListFilterByStatus(t *testing.T) {
	env := newTestEnv(t)

	// Upload a receipt (creates a pending expense)
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("file", "receipt.jpg")
	part.Write([]byte{0xFF, 0xD8, 0xFF, 0xE0})
	w.Close()

	req := httptest.NewRequest("POST", "/api/expenses/receipt", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("upload: expected 201, got %d", rr.Code)
	}

	// Filter by pending — should return 1
	req = httptest.NewRequest("GET", "/api/expenses?status=pending", nil)
	rr = httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	var pending []interface{}
	json.NewDecoder(rr.Body).Decode(&pending)
	if len(pending) != 1 {
		t.Errorf("expected 1 pending, got %d", len(pending))
	}

	// Filter by confirmed — should return 0
	req = httptest.NewRequest("GET", "/api/expenses?status=confirmed", nil)
	rr = httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	var confirmed []interface{}
	json.NewDecoder(rr.Body).Decode(&confirmed)
	if len(confirmed) != 0 {
		t.Errorf("expected 0 confirmed, got %d", len(confirmed))
	}
}

func TestExpenseRetry_NotFound(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest("POST", "/api/expenses/"+uuid.New().String()+"/retry", nil)
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestExpenseRetry_ResetsToProcessing(t *testing.T) {
	env := newTestEnv(t)

	// Upload a receipt first
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, _ := w.CreateFormFile("file", "receipt.pdf")
	part.Write([]byte("%PDF-1.4 test pdf"))
	w.Close()

	req := httptest.NewRequest("POST", "/api/expenses/receipt", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("upload: expected 201, got %d: %s", rr.Code, rr.Body.String())
	}

	var uploaded map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&uploaded)
	id := uploaded["id"].(string)

	// Give the async parse goroutine a moment to finish (it will fail since
	// no ANTHROPIC_API_KEY is set — that's fine, we just need the expense to exist)
	// Wait briefly to ensure the goroutine has started and failed
	<-time.After(200 * time.Millisecond)

	// Retry
	req = httptest.NewRequest("POST", "/api/expenses/"+id+"/retry", nil)
	rr = httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("retry: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var retried map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&retried)

	// After retry, expense should be reset to pending with empty vendor
	if retried["status"] != "pending" {
		t.Errorf("expected status pending after retry, got %v", retried["status"])
	}
	if retried["vendor"] != "" {
		t.Errorf("expected empty vendor after retry, got %v", retried["vendor"])
	}
	if retried["notes"] != "" {
		t.Errorf("expected empty notes after retry, got %v", retried["notes"])
	}
}

func TestExpenseRetry_InvalidID(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest("POST", "/api/expenses/not-a-uuid/retry", nil)
	rr := httptest.NewRecorder()
	env.handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rr.Code)
	}
}

