package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/philjestin/daedalus/internal/database"
	"github.com/philjestin/daedalus/internal/model"
)

// openTestDB creates an in-memory SQLite database with schema applied.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.Open(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// --- Project Repository Tests ---

func TestProjectRepository_CreateAndGetByID(t *testing.T) {
	db := openTestDB(t)
	repo := &ProjectRepository{db: db}
	ctx := context.Background()

	project := &model.Project{
		Name:        "Test Project",
		Description: "A test project",
		Tags:        []string{"test", "project"},
	}

	if err := repo.Create(ctx, project); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if project.ID == uuid.Nil {
		t.Error("Create should set project ID")
	}
	if project.CreatedAt.IsZero() {
		t.Error("Create should set CreatedAt")
	}
	if project.Source != "manual" {
		t.Errorf("Create should default Source to 'manual', got %q", project.Source)
	}

	// Retrieve the project
	got, err := repo.GetByID(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got == nil {
		t.Fatal("GetByID returned nil")
	}

	if got.Name != "Test Project" {
		t.Errorf("Name = %q, want %q", got.Name, "Test Project")
	}
	if got.Description != "A test project" {
		t.Errorf("Description = %q, want %q", got.Description, "A test project")
	}
	if len(got.Tags) != 2 || got.Tags[0] != "test" || got.Tags[1] != "project" {
		t.Errorf("Tags = %v, want [test, project]", got.Tags)
	}
}

func TestProjectRepository_GetByID_NotFound(t *testing.T) {
	db := openTestDB(t)
	repo := &ProjectRepository{db: db}
	ctx := context.Background()

	got, err := repo.GetByID(ctx, uuid.New())
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}
	if got != nil {
		t.Error("GetByID should return nil for non-existent project")
	}
}

func TestProjectRepository_List(t *testing.T) {
	db := openTestDB(t)
	repo := &ProjectRepository{db: db}
	ctx := context.Background()

	// Create multiple projects
	for i := 0; i < 3; i++ {
		p := &model.Project{Name: "Project " + string(rune('A'+i))}
		if err := repo.Create(ctx, p); err != nil {
			t.Fatalf("Create %d failed: %v", i, err)
		}
		// Small delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)
	}

	projects, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(projects) != 3 {
		t.Errorf("len(projects) = %d, want 3", len(projects))
	}

	// Should be ordered by updated_at DESC (most recent first)
	if projects[0].Name != "Project C" {
		t.Errorf("First project should be 'Project C', got %q", projects[0].Name)
	}
}

func TestProjectRepository_Update(t *testing.T) {
	db := openTestDB(t)
	repo := &ProjectRepository{db: db}
	ctx := context.Background()

	project := &model.Project{Name: "Original"}
	if err := repo.Create(ctx, project); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	project.Name = "Updated"
	project.Description = "New description"
	project.Tags = []string{"updated"}

	if err := repo.Update(ctx, project); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := repo.GetByID(ctx, project.ID)
	if got.Name != "Updated" {
		t.Errorf("Name = %q, want %q", got.Name, "Updated")
	}
	if got.Description != "New description" {
		t.Errorf("Description = %q, want %q", got.Description, "New description")
	}
	if len(got.Tags) != 1 || got.Tags[0] != "updated" {
		t.Errorf("Tags = %v, want [updated]", got.Tags)
	}
}

func TestProjectRepository_Delete(t *testing.T) {
	db := openTestDB(t)
	repo := &ProjectRepository{db: db}
	ctx := context.Background()

	project := &model.Project{Name: "To Delete"}
	if err := repo.Create(ctx, project); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := repo.Delete(ctx, project.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	got, _ := repo.GetByID(ctx, project.ID)
	if got != nil {
		t.Error("Project should be deleted")
	}
}

func TestProjectRepository_ListByTemplateID(t *testing.T) {
	db := openTestDB(t)
	projectRepo := &ProjectRepository{db: db}
	templateRepo := &TemplateRepository{db: db}
	ctx := context.Background()

	// Create a template first
	template := &model.Template{Name: "Test Template"}
	if err := templateRepo.Create(ctx, template); err != nil {
		t.Fatalf("Create template failed: %v", err)
	}

	// Create projects linked to template
	for i := 0; i < 2; i++ {
		p := &model.Project{
			Name:       "Linked Project " + string(rune('A'+i)),
			TemplateID: &template.ID,
		}
		if err := projectRepo.Create(ctx, p); err != nil {
			t.Fatalf("Create linked project %d failed: %v", i, err)
		}
	}

	// Create unlinked project
	unlinked := &model.Project{Name: "Unlinked Project"}
	if err := projectRepo.Create(ctx, unlinked); err != nil {
		t.Fatalf("Create unlinked project failed: %v", err)
	}

	// List by template ID
	projects, err := projectRepo.ListByTemplateID(ctx, template.ID)
	if err != nil {
		t.Fatalf("ListByTemplateID failed: %v", err)
	}

	if len(projects) != 2 {
		t.Errorf("len(projects) = %d, want 2", len(projects))
	}
}

// --- Part Repository Tests ---

func TestPartRepository_CreateAndGetByID(t *testing.T) {
	db := openTestDB(t)
	projectRepo := &ProjectRepository{db: db}
	partRepo := &PartRepository{db: db}
	ctx := context.Background()

	// Create a project first
	project := &model.Project{Name: "Test Project"}
	if err := projectRepo.Create(ctx, project); err != nil {
		t.Fatalf("Create project failed: %v", err)
	}

	part := &model.Part{
		ProjectID:   project.ID,
		Name:        "Test Part",
		Description: "A test part",
		Quantity:    2,
	}

	if err := partRepo.Create(ctx, part); err != nil {
		t.Fatalf("Create part failed: %v", err)
	}

	if part.ID == uuid.Nil {
		t.Error("Create should set part ID")
	}
	if part.Status != model.PartStatusDesign {
		t.Errorf("Create should default Status to 'design', got %q", part.Status)
	}

	got, err := partRepo.GetByID(ctx, part.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.Name != "Test Part" {
		t.Errorf("Name = %q, want %q", got.Name, "Test Part")
	}
	if got.Quantity != 2 {
		t.Errorf("Quantity = %d, want 2", got.Quantity)
	}
}

func TestPartRepository_ListByProject(t *testing.T) {
	db := openTestDB(t)
	projectRepo := &ProjectRepository{db: db}
	partRepo := &PartRepository{db: db}
	ctx := context.Background()

	project := &model.Project{Name: "Project with Parts"}
	if err := projectRepo.Create(ctx, project); err != nil {
		t.Fatalf("Create project failed: %v", err)
	}

	// Create parts
	for i := 0; i < 3; i++ {
		p := &model.Part{
			ProjectID: project.ID,
			Name:      "Part " + string(rune('A'+i)),
		}
		if err := partRepo.Create(ctx, p); err != nil {
			t.Fatalf("Create part %d failed: %v", i, err)
		}
	}

	parts, err := partRepo.ListByProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("ListByProject failed: %v", err)
	}

	if len(parts) != 3 {
		t.Errorf("len(parts) = %d, want 3", len(parts))
	}
}

// --- Printer Repository Tests ---

func TestPrinterRepository_CreateAndGetByID(t *testing.T) {
	db := openTestDB(t)
	repo := &PrinterRepository{db: db}
	ctx := context.Background()

	printer := &model.Printer{
		Name:            "Test Printer",
		Model:           "P1S",
		Manufacturer:    "Bambu Lab",
		ConnectionType:  model.ConnectionTypeBambuLAN,
		ConnectionURI:   "192.168.1.100",
		NozzleDiameter:  0.4,
		CostPerHourCents: 150,
	}

	if err := repo.Create(ctx, printer); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if printer.ID == uuid.Nil {
		t.Error("Create should set printer ID")
	}
	if printer.Status != model.PrinterStatusOffline {
		t.Errorf("Create should default Status to 'offline', got %q", printer.Status)
	}

	got, err := repo.GetByID(ctx, printer.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.Name != "Test Printer" {
		t.Errorf("Name = %q, want %q", got.Name, "Test Printer")
	}
	if got.CostPerHourCents != 150 {
		t.Errorf("CostPerHourCents = %d, want 150", got.CostPerHourCents)
	}
	if got.NozzleDiameter != 0.4 {
		t.Errorf("NozzleDiameter = %f, want 0.4", got.NozzleDiameter)
	}
}

func TestPrinterRepository_List(t *testing.T) {
	db := openTestDB(t)
	repo := &PrinterRepository{db: db}
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		p := &model.Printer{Name: "Printer " + string(rune('A'+i))}
		if err := repo.Create(ctx, p); err != nil {
			t.Fatalf("Create %d failed: %v", i, err)
		}
	}

	printers, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(printers) != 3 {
		t.Errorf("len(printers) = %d, want 3", len(printers))
	}
}

func TestPrinterRepository_Update(t *testing.T) {
	db := openTestDB(t)
	repo := &PrinterRepository{db: db}
	ctx := context.Background()

	printer := &model.Printer{Name: "Original", CostPerHourCents: 100}
	if err := repo.Create(ctx, printer); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	printer.Name = "Updated"
	printer.CostPerHourCents = 200
	printer.Location = "Shelf 1"

	if err := repo.Update(ctx, printer); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := repo.GetByID(ctx, printer.ID)
	if got.Name != "Updated" {
		t.Errorf("Name = %q, want %q", got.Name, "Updated")
	}
	if got.CostPerHourCents != 200 {
		t.Errorf("CostPerHourCents = %d, want 200", got.CostPerHourCents)
	}
	if got.Location != "Shelf 1" {
		t.Errorf("Location = %q, want %q", got.Location, "Shelf 1")
	}
}

// --- Material Repository Tests ---

func TestMaterialRepository_CreateAndGetByID(t *testing.T) {
	db := openTestDB(t)
	repo := &MaterialRepository{db: db}
	ctx := context.Background()

	material := &model.Material{
		Name:         "Test PLA",
		Type:         "pla",
		Manufacturer: "Bambu Lab",
		Color:        "Black",
		ColorHex:     "#000000",
		Density:      1.24,
		CostPerKg:    19.99,
	}

	if err := repo.Create(ctx, material); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := repo.GetByID(ctx, material.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.Name != "Test PLA" {
		t.Errorf("Name = %q, want %q", got.Name, "Test PLA")
	}
	if got.CostPerKg != 19.99 {
		t.Errorf("CostPerKg = %f, want 19.99", got.CostPerKg)
	}
}

func TestMaterialRepository_ListByType(t *testing.T) {
	db := openTestDB(t)
	repo := &MaterialRepository{db: db}
	ctx := context.Background()

	// Create PLA materials
	for i := 0; i < 2; i++ {
		m := &model.Material{Name: "PLA " + string(rune('A'+i)), Type: "pla"}
		if err := repo.Create(ctx, m); err != nil {
			t.Fatalf("Create PLA %d failed: %v", i, err)
		}
	}

	// Create PETG material
	petg := &model.Material{Name: "PETG", Type: "petg"}
	if err := repo.Create(ctx, petg); err != nil {
		t.Fatalf("Create PETG failed: %v", err)
	}

	// List only PLA
	materials, err := repo.ListByType(ctx, "pla")
	if err != nil {
		t.Fatalf("ListByType failed: %v", err)
	}

	if len(materials) != 2 {
		t.Errorf("len(materials) = %d, want 2", len(materials))
	}
	for _, m := range materials {
		if m.Type != "pla" {
			t.Errorf("Expected type 'pla', got %q", m.Type)
		}
	}
}

// --- Spool Repository Tests ---

func TestSpoolRepository_CreateAndGetByID(t *testing.T) {
	db := openTestDB(t)
	materialRepo := &MaterialRepository{db: db}
	spoolRepo := &SpoolRepository{db: db}
	ctx := context.Background()

	// Create material first
	material := &model.Material{Name: "PLA", Type: "pla"}
	if err := materialRepo.Create(ctx, material); err != nil {
		t.Fatalf("Create material failed: %v", err)
	}

	spool := &model.MaterialSpool{
		MaterialID:      material.ID,
		InitialWeight:   1000,
		RemainingWeight: 800,
		PurchaseCost:    25.00,
		Status:          model.SpoolStatusInUse,
	}

	if err := spoolRepo.Create(ctx, spool); err != nil {
		t.Fatalf("Create spool failed: %v", err)
	}

	got, err := spoolRepo.GetByID(ctx, spool.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.InitialWeight != 1000 {
		t.Errorf("InitialWeight = %f, want 1000", got.InitialWeight)
	}
	if got.RemainingWeight != 800 {
		t.Errorf("RemainingWeight = %f, want 800", got.RemainingWeight)
	}
	if got.Status != model.SpoolStatusInUse {
		t.Errorf("Status = %q, want %q", got.Status, model.SpoolStatusInUse)
	}
}

// --- Print Job Repository Tests ---

// setupPrintJobTestData creates the necessary project, part, file, and design for print job tests.
func setupPrintJobTestData(t *testing.T, db *sql.DB) (*model.Project, *model.Design) {
	t.Helper()
	ctx := context.Background()

	projectRepo := &ProjectRepository{db: db}
	partRepo := &PartRepository{db: db}
	fileRepo := &FileRepository{db: db}
	designRepo := &DesignRepository{db: db}

	// Create project
	project := &model.Project{Name: "Print Job Test Project"}
	if err := projectRepo.Create(ctx, project); err != nil {
		t.Fatalf("Create project failed: %v", err)
	}

	// Create part
	part := &model.Part{ProjectID: project.ID, Name: "Print Job Test Part"}
	if err := partRepo.Create(ctx, part); err != nil {
		t.Fatalf("Create part failed: %v", err)
	}

	// Create file (required by design FK)
	file := &model.File{
		Hash:         "abc123hash",
		OriginalName: "test.3mf",
		ContentType:  "application/3mf",
		SizeBytes:    1024,
		StoragePath:  "ab/c1/abc123hash/test.3mf",
	}
	if err := fileRepo.Create(ctx, file); err != nil {
		t.Fatalf("Create file failed: %v", err)
	}

	// Create design
	design := &model.Design{
		PartID:        part.ID,
		FileName:      "test.3mf",
		FileID:        file.ID,
		FileHash:      file.Hash,
		FileSizeBytes: file.SizeBytes,
		FileType:      "3mf",
	}
	if err := designRepo.Create(ctx, design); err != nil {
		t.Fatalf("Create design failed: %v", err)
	}

	return project, design
}

func TestPrintJobRepository_CreateAndGetByID(t *testing.T) {
	db := openTestDB(t)
	printJobRepo := &PrintJobRepository{db: db}
	ctx := context.Background()

	project, design := setupPrintJobTestData(t, db)

	job := &model.PrintJob{
		DesignID:  design.ID,
		ProjectID: &project.ID,
		Status:    model.PrintJobStatusQueued,
	}

	if err := printJobRepo.Create(ctx, job); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if job.ID == uuid.Nil {
		t.Error("Create should set job ID")
	}

	got, err := printJobRepo.GetByID(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.Status != model.PrintJobStatusQueued {
		t.Errorf("Status = %q, want %q", got.Status, model.PrintJobStatusQueued)
	}
	if got.AttemptNumber != 1 {
		t.Errorf("AttemptNumber = %d, want 1", got.AttemptNumber)
	}
}

func TestPrintJobRepository_Update(t *testing.T) {
	db := openTestDB(t)
	printJobRepo := &PrintJobRepository{db: db}
	ctx := context.Background()

	_, design := setupPrintJobTestData(t, db)

	job := &model.PrintJob{DesignID: design.ID, Status: model.PrintJobStatusQueued}
	if err := printJobRepo.Create(ctx, job); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Update job fields (note: status is computed from events in GetByID, so we test other fields)
	job.Notes = "Test notes"
	actualSeconds := 3600
	materialUsedGrams := 25.5
	job.ActualSeconds = &actualSeconds
	job.MaterialUsedGrams = &materialUsedGrams
	if err := printJobRepo.Update(ctx, job); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	got, _ := printJobRepo.GetByID(ctx, job.ID)
	if got.Notes != "Test notes" {
		t.Errorf("Notes = %q, want %q", got.Notes, "Test notes")
	}
	if got.ActualSeconds == nil || *got.ActualSeconds != 3600 {
		t.Errorf("ActualSeconds = %v, want 3600", got.ActualSeconds)
	}
	if got.MaterialUsedGrams == nil || *got.MaterialUsedGrams != 25.5 {
		t.Errorf("MaterialUsedGrams = %v, want 25.5", got.MaterialUsedGrams)
	}
}

func TestPrintJobRepository_AppendEvent(t *testing.T) {
	db := openTestDB(t)
	printJobRepo := &PrintJobRepository{db: db}
	ctx := context.Background()

	_, design := setupPrintJobTestData(t, db)

	job := &model.PrintJob{DesignID: design.ID, Status: model.PrintJobStatusQueued}
	if err := printJobRepo.Create(ctx, job); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Append event to change status
	printingStatus := model.PrintJobStatusPrinting
	event := model.NewJobEvent(job.ID, model.JobEventStarted, &printingStatus)
	progress := 0.0
	event.Progress = &progress

	if err := printJobRepo.AppendEvent(ctx, event); err != nil {
		t.Fatalf("AppendEvent failed: %v", err)
	}

	// Now GetByID should return the status from the event
	got, _ := printJobRepo.GetByID(ctx, job.ID)
	if got.Status != model.PrintJobStatusPrinting {
		t.Errorf("Status after event = %q, want %q", got.Status, model.PrintJobStatusPrinting)
	}
}

func TestPrintJobRepository_GetEvents(t *testing.T) {
	db := openTestDB(t)
	printJobRepo := &PrintJobRepository{db: db}
	ctx := context.Background()

	_, design := setupPrintJobTestData(t, db)

	job := &model.PrintJob{DesignID: design.ID, Status: model.PrintJobStatusQueued}
	printJobRepo.Create(ctx, job)

	// Add multiple events (note: Create also adds a "created" event, so we start with 1)
	statuses := []model.PrintJobStatus{
		model.PrintJobStatusAssigned,
		model.PrintJobStatusPrinting,
		model.PrintJobStatusCompleted,
	}
	eventTypes := []model.JobEventType{
		model.JobEventAssigned,
		model.JobEventStarted,
		model.JobEventCompleted,
	}

	for i, status := range statuses {
		s := status // capture
		event := model.NewJobEvent(job.ID, eventTypes[i], &s)
		if err := printJobRepo.AppendEvent(ctx, event); err != nil {
			t.Fatalf("AppendEvent %d failed: %v", i, err)
		}
	}

	events, err := printJobRepo.GetEvents(ctx, job.ID)
	if err != nil {
		t.Fatalf("GetEvents failed: %v", err)
	}

	// Expect 4 events: 1 created (from Create) + 3 appended
	if len(events) != 4 {
		t.Errorf("len(events) = %d, want 4", len(events))
	}
}

// --- Settings Repository Tests ---

func TestSettingsRepository_SetAndGet(t *testing.T) {
	db := openTestDB(t)
	repo := &SettingsRepository{db: db}
	ctx := context.Background()

	// Set a value
	if err := repo.Set(ctx, "test_key", "test_value"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Get the value
	setting, err := repo.Get(ctx, "test_key")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if setting == nil {
		t.Fatal("Get returned nil for existing key")
	}
	if setting.Value != "test_value" {
		t.Errorf("Get.Value = %q, want %q", setting.Value, "test_value")
	}

	// Upsert (update existing)
	if err := repo.Set(ctx, "test_key", "updated_value"); err != nil {
		t.Fatalf("Set (upsert) failed: %v", err)
	}

	setting, _ = repo.Get(ctx, "test_key")
	if setting.Value != "updated_value" {
		t.Errorf("Get after upsert = %q, want %q", setting.Value, "updated_value")
	}
}

func TestSettingsRepository_Get_NotFound(t *testing.T) {
	db := openTestDB(t)
	repo := &SettingsRepository{db: db}
	ctx := context.Background()

	setting, err := repo.Get(ctx, "nonexistent_key")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if setting != nil {
		t.Errorf("Get for nonexistent key should return nil, got %+v", setting)
	}
}

func TestSettingsRepository_Delete(t *testing.T) {
	db := openTestDB(t)
	repo := &SettingsRepository{db: db}
	ctx := context.Background()

	repo.Set(ctx, "to_delete", "value")

	if err := repo.Delete(ctx, "to_delete"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	setting, _ := repo.Get(ctx, "to_delete")
	if setting != nil {
		t.Errorf("Setting should be nil after delete, got %+v", setting)
	}
}

// --- Template Repository Tests ---

func TestTemplateRepository_CreateAndGetByID(t *testing.T) {
	db := openTestDB(t)
	repo := &TemplateRepository{db: db}
	ctx := context.Background()

	template := &model.Template{
		Name:                   "Test Template",
		Description:            "A test template",
		SKU:                    "TEST-001",
		MaterialType:           "pla",
		EstimatedMaterialGrams: 50,
		QuantityPerOrder:       1,
		LaborMinutes:           5,
		SalePriceCents:         1500,
		IsActive:               true,
	}

	if err := repo.Create(ctx, template); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := repo.GetByID(ctx, template.ID)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if got.Name != "Test Template" {
		t.Errorf("Name = %q, want %q", got.Name, "Test Template")
	}
	if got.SKU != "TEST-001" {
		t.Errorf("SKU = %q, want %q", got.SKU, "TEST-001")
	}
	if got.SalePriceCents != 1500 {
		t.Errorf("SalePriceCents = %d, want 1500", got.SalePriceCents)
	}
}

// --- Helper Tests ---

func TestMarshalUnmarshalStringArray(t *testing.T) {
	testCases := []struct {
		name  string
		input []string
	}{
		{"nil", nil},
		{"empty", []string{}},
		{"single", []string{"one"}},
		{"multiple", []string{"a", "b", "c"}},
		{"special chars", []string{"hello world", "with,comma", "with\"quote"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			marshaled := marshalStringArray(tc.input)
			unmarshaled := unmarshalStringArray([]byte(marshaled))

			// nil should become empty slice
			expected := tc.input
			if expected == nil {
				expected = []string{}
			}

			if len(unmarshaled) != len(expected) {
				t.Errorf("length mismatch: got %d, want %d", len(unmarshaled), len(expected))
				return
			}
			for i := range expected {
				if unmarshaled[i] != expected[i] {
					t.Errorf("element %d: got %q, want %q", i, unmarshaled[i], expected[i])
				}
			}
		})
	}
}

func TestUnmarshalStringArray_InvalidJSON(t *testing.T) {
	// Invalid JSON should return empty slice
	result := unmarshalStringArray([]byte("not valid json"))
	if len(result) != 0 {
		t.Errorf("invalid JSON should return empty slice, got %v", result)
	}
}
