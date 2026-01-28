package database

import (
	"database/sql"
	"strings"
	"testing"
)

// TestSchemaHasPrinterColumns verifies the printers table contains every
// column that the repository layer references. This is a regression guard
// against adding a column to queries without updating the schema.
func TestSchemaHasPrinterColumns(t *testing.T) {
	db := openTestDB(t)

	expected := []string{
		"id", "name", "model", "manufacturer",
		"connection_type", "connection_uri", "api_key", "serial_number",
		"status", "build_volume", "nozzle_diameter",
		"location", "notes", "min_material_percent",
		"cost_per_hour_cents",
		"created_at", "updated_at",
	}

	cols := tableColumns(t, db, "printers")
	for _, want := range expected {
		if !cols[want] {
			t.Errorf("printers table missing column %q", want)
		}
	}
}

// TestSchemaHasDesignColumns verifies designs table columns.
func TestSchemaHasDesignColumns(t *testing.T) {
	db := openTestDB(t)

	expected := []string{
		"id", "part_id", "version", "file_name", "file_type",
		"file_id", "notes", "created_at",
	}

	cols := tableColumns(t, db, "designs")
	for _, want := range expected {
		if !cols[want] {
			t.Errorf("designs table missing column %q", want)
		}
	}
}

// TestSchemaHasPartColumns verifies parts table columns.
func TestSchemaHasPartColumns(t *testing.T) {
	db := openTestDB(t)

	expected := []string{
		"id", "project_id", "name", "description", "quantity",
		"created_at", "updated_at",
	}

	cols := tableColumns(t, db, "parts")
	for _, want := range expected {
		if !cols[want] {
			t.Errorf("parts table missing column %q", want)
		}
	}
}

// TestSchemaHasPrintJobColumns verifies print_jobs table columns.
func TestSchemaHasPrintJobColumns(t *testing.T) {
	db := openTestDB(t)

	expected := []string{
		"id", "design_id", "printer_id", "status",
		"created_at",
	}

	cols := tableColumns(t, db, "print_jobs")
	for _, want := range expected {
		if !cols[want] {
			t.Errorf("print_jobs table missing column %q", want)
		}
	}
}

// TestSchemaHasBambuCloudAuthColumns verifies bambu_cloud_auth table columns.
func TestSchemaHasBambuCloudAuthColumns(t *testing.T) {
	db := openTestDB(t)

	expected := []string{
		"id", "email", "access_token", "mqtt_username",
	}

	cols := tableColumns(t, db, "bambu_cloud_auth")
	for _, want := range expected {
		if !cols[want] {
			t.Errorf("bambu_cloud_auth table missing column %q", want)
		}
	}
}

// TestSchemaHasSettingsColumns verifies the settings table columns.
func TestSchemaHasSettingsColumns(t *testing.T) {
	db := openTestDB(t)

	expected := []string{
		"key", "value", "updated_at",
	}

	cols := tableColumns(t, db, "settings")
	for _, want := range expected {
		if !cols[want] {
			t.Errorf("settings table missing column %q", want)
		}
	}
}

// TestSettingsUpsert verifies insert-on-conflict works for settings.
func TestSettingsUpsert(t *testing.T) {
	db := openTestDB(t)

	// Insert
	_, err := db.Exec(`INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?)`, "test_key", "value1", "2024-01-01T00:00:00Z")
	if err != nil {
		t.Fatalf("initial insert failed: %v", err)
	}

	// Upsert with ON CONFLICT
	_, err = db.Exec(`
		INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at
	`, "test_key", "value2", "2024-01-02T00:00:00Z")
	if err != nil {
		t.Fatalf("upsert failed: %v", err)
	}

	var value string
	err = db.QueryRow(`SELECT value FROM settings WHERE key = ?`, "test_key").Scan(&value)
	if err != nil {
		t.Fatalf("select failed: %v", err)
	}
	if value != "value2" {
		t.Errorf("expected value2, got %q", value)
	}
}

// TestPrinterInsertSucceeds verifies a full INSERT with all columns works.
// This catches column count mismatches between INSERT queries and schema.
func TestPrinterInsertSucceeds(t *testing.T) {
	db := openTestDB(t)

	_, err := db.Exec(`
		INSERT INTO printers (id, name, model, manufacturer, connection_type, connection_uri, api_key, serial_number, status, build_volume, nozzle_diameter, location, notes, min_material_percent, cost_per_hour_cents, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "test-id-1", "Test Printer", "A1 Mini", "Bambu Lab", "bambu_cloud", "u_123", "token", "SN001", "offline", "null", 0.4, "Shelf 1", "", 10, 150, "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z")

	if err != nil {
		t.Fatalf("printer INSERT failed: %v", err)
	}

	// Verify SELECT with same columns also works
	var name, connType, serial string
	var cost int
	err = db.QueryRow(`
		SELECT name, connection_type, serial_number, cost_per_hour_cents
		FROM printers WHERE id = ?
	`, "test-id-1").Scan(&name, &connType, &serial, &cost)

	if err != nil {
		t.Fatalf("printer SELECT failed: %v", err)
	}
	if name != "Test Printer" {
		t.Errorf("name: got %q", name)
	}
	if connType != "bambu_cloud" {
		t.Errorf("connection_type: got %q", connType)
	}
	if serial != "SN001" {
		t.Errorf("serial_number: got %q", serial)
	}
	if cost != 150 {
		t.Errorf("cost_per_hour_cents: got %d", cost)
	}
}

// TestPrinterUpdateSucceeds verifies UPDATE with all columns works.
func TestPrinterUpdateSucceeds(t *testing.T) {
	db := openTestDB(t)

	// Insert first
	db.Exec(`
		INSERT INTO printers (id, name, model, manufacturer, connection_type, connection_uri, api_key, serial_number, status, build_volume, nozzle_diameter, location, notes, min_material_percent, cost_per_hour_cents, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "test-id-2", "Original", "", "", "manual", "", "", "", "offline", "null", 0.4, "", "", 10, 0, "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z")

	// Update
	_, err := db.Exec(`
		UPDATE printers SET name = ?, model = ?, manufacturer = ?, connection_type = ?, connection_uri = ?, api_key = ?, serial_number = ?, status = ?, build_volume = ?, nozzle_diameter = ?, location = ?, notes = ?, min_material_percent = ?, cost_per_hour_cents = ?, updated_at = ?
		WHERE id = ?
	`, "Updated", "P1S", "Bambu Lab", "bambu_cloud", "u_999", "new_token", "SN002", "offline", "null", 0.4, "Rack 1", "Updated notes", 10, 200, "2024-01-02T00:00:00Z", "test-id-2")

	if err != nil {
		t.Fatalf("printer UPDATE failed: %v", err)
	}

	var name string
	var cost int
	db.QueryRow("SELECT name, cost_per_hour_cents FROM printers WHERE id = ?", "test-id-2").Scan(&name, &cost)
	if name != "Updated" {
		t.Errorf("name after update: got %q", name)
	}
	if cost != 200 {
		t.Errorf("cost after update: got %d", cost)
	}
}

// TestMigrationAddsColumns verifies that the ALTER TABLE migrations
// successfully add columns to an existing table.
func TestMigrationAddsColumns(t *testing.T) {
	db := openTestDB(t)

	// The migration should have already run via openTestDB.
	// Verify the columns added by ALTER TABLE exist.
	cols := tableColumns(t, db, "printers")

	if !cols["serial_number"] {
		t.Error("migration did not add serial_number column")
	}
	if !cols["cost_per_hour_cents"] {
		t.Error("migration did not add cost_per_hour_cents column")
	}
}

// TestMaterialFindByTypeManufacturerColor verifies the material lookup query works.
func TestMaterialFindByTypeManufacturerColor(t *testing.T) {
	db := openTestDB(t)

	// Insert a material
	_, err := db.Exec(`
		INSERT INTO materials (id, name, type, manufacturer, color, color_hex, density, cost_per_kg, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "mat-1", "Bambu Lab PLA - Beige", "pla", "Bambu Lab", "Beige", "#F5DEB3", 1.24, 19.99, "", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z")
	if err != nil {
		t.Fatalf("insert material: %v", err)
	}

	// Exact match
	var id string
	err = db.QueryRow(`
		SELECT id FROM materials
		WHERE LOWER(type) = LOWER(?) AND LOWER(manufacturer) = LOWER(?) AND LOWER(color) = LOWER(?)
		LIMIT 1
	`, "pla", "Bambu Lab", "Beige").Scan(&id)
	if err != nil {
		t.Fatalf("exact match failed: %v", err)
	}
	if id != "mat-1" {
		t.Errorf("expected mat-1, got %q", id)
	}

	// Case-insensitive match
	err = db.QueryRow(`
		SELECT id FROM materials
		WHERE LOWER(type) = LOWER(?) AND LOWER(manufacturer) = LOWER(?) AND LOWER(color) = LOWER(?)
		LIMIT 1
	`, "PLA", "bambu lab", "beige").Scan(&id)
	if err != nil {
		t.Fatalf("case-insensitive match failed: %v", err)
	}
	if id != "mat-1" {
		t.Errorf("expected mat-1, got %q", id)
	}

	// No match
	err = db.QueryRow(`
		SELECT id FROM materials
		WHERE LOWER(type) = LOWER(?) AND LOWER(manufacturer) = LOWER(?) AND LOWER(color) = LOWER(?)
		LIMIT 1
	`, "petg", "Bambu Lab", "Beige").Scan(&id)
	if err == nil {
		t.Error("expected no match for PETG + Beige")
	}
}

// TestMaterialInsertAndExpenseItemLink verifies the full material → spool → expense_item link.
func TestMaterialInsertAndExpenseItemLink(t *testing.T) {
	db := openTestDB(t)

	// Insert material
	_, err := db.Exec(`
		INSERT INTO materials (id, name, type, manufacturer, color, color_hex, density, cost_per_kg, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "mat-pla-black", "Bambu Lab PLA - Black", "pla", "Bambu Lab", "Black", "#000000", 1.24, 19.99, "", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z")
	if err != nil {
		t.Fatalf("insert material: %v", err)
	}

	// Insert spool referencing material
	_, err = db.Exec(`
		INSERT INTO material_spools (id, material_id, initial_weight, remaining_weight, status, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, "spool-1", "mat-pla-black", 1000, 1000, "new", "From receipt: Bambu Lab US", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z")
	if err != nil {
		t.Fatalf("insert spool: %v", err)
	}

	// Insert expense
	_, err = db.Exec(`
		INSERT INTO expenses (id, status, created_at, updated_at)
		VALUES (?, ?, ?, ?)
	`, "exp-1", "confirmed", "2024-01-01T00:00:00Z", "2024-01-01T00:00:00Z")
	if err != nil {
		t.Fatalf("insert expense: %v", err)
	}

	// Insert expense item linked to spool and material
	_, err = db.Exec(`
		INSERT INTO expense_items (id, expense_id, description, quantity, unit_price_cents, total_price_cents, category, matched_spool_id, matched_material_id, action_taken, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, "ei-1", "exp-1", "PLA Basic - Black", 1, 1999, 1299, "filament", "spool-1", "mat-pla-black", "created_spool", "2024-01-01T00:00:00Z")
	if err != nil {
		t.Fatalf("insert expense item: %v", err)
	}

	// Verify the join works
	var description, spoolID, materialID string
	err = db.QueryRow(`
		SELECT ei.description, ei.matched_spool_id, ei.matched_material_id
		FROM expense_items ei
		JOIN material_spools ms ON ms.id = ei.matched_spool_id
		JOIN materials m ON m.id = ei.matched_material_id
		WHERE ei.id = ?
	`, "ei-1").Scan(&description, &spoolID, &materialID)
	if err != nil {
		t.Fatalf("join query failed: %v", err)
	}
	if description != "PLA Basic - Black" {
		t.Errorf("description: got %q", description)
	}
	if spoolID != "spool-1" {
		t.Errorf("spool_id: got %q", spoolID)
	}
	if materialID != "mat-pla-black" {
		t.Errorf("material_id: got %q", materialID)
	}
}

// --- helpers ---

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := Open(":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

// tableColumns returns a set of column names for the given table.
func tableColumns(t *testing.T, db *sql.DB, table string) map[string]bool {
	t.Helper()
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		t.Fatalf("PRAGMA table_info(%s): %v", table, err)
	}
	defer rows.Close()

	cols := make(map[string]bool)
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull int
		var dfltValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			t.Fatalf("scan column info: %v", err)
		}
		cols[strings.ToLower(name)] = true
	}
	return cols
}
