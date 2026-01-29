package threemf

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// createTestZip creates a temporary 3MF (zip) file with the given files.
// files is a map of filename to content.
func createTestZip(t *testing.T, files map[string]string) string {
	t.Helper()

	tmpFile, err := os.CreateTemp("", "test-*.3mf")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	tmpFile.Close()

	f, err := os.Create(tmpFile.Name())
	if err != nil {
		t.Fatalf("open temp file: %v", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	for name, content := range files {
		fw, err := w.Create(name)
		if err != nil {
			t.Fatalf("create zip entry %s: %v", name, err)
		}
		if _, err := fw.Write([]byte(content)); err != nil {
			t.Fatalf("write zip entry %s: %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}

	t.Cleanup(func() {
		os.Remove(tmpFile.Name())
	})

	return tmpFile.Name()
}

func TestParse_ValidSlicedFile(t *testing.T) {
	sliceConfig := `<?xml version="1.0" encoding="UTF-8"?>
<config>
  <plate>
    <metadata key="prediction" value="3600"/>
    <metadata key="weight" value="25.5"/>
    <metadata key="printer_model_id" value="Bambu Lab P1S"/>
    <metadata key="nozzle_diameters" value="0.4"/>
    <filament id="1" type="PLA" color="#FF0000" used_m="8.5" used_g="25.5"/>
  </plate>
</config>`

	path := createTestZip(t, map[string]string{
		"Metadata/slice_info.config": sliceConfig,
		"3D/3dmodel.model":           "<model/>", // dummy model file
	})

	result, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if result == nil {
		t.Fatal("Parse returned nil result for valid sliced file")
	}

	var profile SliceProfile
	if err := json.Unmarshal(result, &profile); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if profile.PrintTimeSeconds != 3600 {
		t.Errorf("PrintTimeSeconds = %d, want 3600", profile.PrintTimeSeconds)
	}
	if profile.WeightGrams != 25.5 {
		t.Errorf("WeightGrams = %f, want 25.5", profile.WeightGrams)
	}
	if profile.PrinterModel != "Bambu Lab P1S" {
		t.Errorf("PrinterModel = %q, want %q", profile.PrinterModel, "Bambu Lab P1S")
	}
	if profile.NozzleDiameter != 0.4 {
		t.Errorf("NozzleDiameter = %f, want 0.4", profile.NozzleDiameter)
	}
	if len(profile.Filaments) != 1 {
		t.Fatalf("len(Filaments) = %d, want 1", len(profile.Filaments))
	}

	f := profile.Filaments[0]
	if f.Type != "PLA" {
		t.Errorf("Filament.Type = %q, want %q", f.Type, "PLA")
	}
	if f.Color != "#FF0000" {
		t.Errorf("Filament.Color = %q, want %q", f.Color, "#FF0000")
	}
	if f.UsedGrams != 25.5 {
		t.Errorf("Filament.UsedGrams = %f, want 25.5", f.UsedGrams)
	}
	if f.UsedMeters != 8.5 {
		t.Errorf("Filament.UsedMeters = %f, want 8.5", f.UsedMeters)
	}
}

func TestParse_MultipleFilaments(t *testing.T) {
	sliceConfig := `<?xml version="1.0" encoding="UTF-8"?>
<config>
  <plate>
    <metadata key="prediction" value="7200"/>
    <metadata key="weight" value="50.0"/>
    <filament id="1" type="PLA" color="#FF0000" used_m="8.5" used_g="25.0"/>
    <filament id="2" type="PETG" color="#00FF00" used_m="8.5" used_g="25.0"/>
  </plate>
</config>`

	path := createTestZip(t, map[string]string{
		"Metadata/slice_info.config": sliceConfig,
	})

	result, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	var profile SliceProfile
	if err := json.Unmarshal(result, &profile); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if len(profile.Filaments) != 2 {
		t.Fatalf("len(Filaments) = %d, want 2", len(profile.Filaments))
	}
	if profile.Filaments[0].Type != "PLA" {
		t.Errorf("Filaments[0].Type = %q, want %q", profile.Filaments[0].Type, "PLA")
	}
	if profile.Filaments[1].Type != "PETG" {
		t.Errorf("Filaments[1].Type = %q, want %q", profile.Filaments[1].Type, "PETG")
	}
}

func TestParse_NoSliceInfoConfig(t *testing.T) {
	// A 3MF file exported without slicing (no Metadata/slice_info.config)
	path := createTestZip(t, map[string]string{
		"3D/3dmodel.model": "<model/>",
		"[Content_Types].xml": `<?xml version="1.0" encoding="UTF-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"/>`,
	})

	result, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if result != nil {
		t.Errorf("Parse returned non-nil result for file without slice_info.config: %s", string(result))
	}
}

func TestParse_EmptyPlates(t *testing.T) {
	// slice_info.config exists but has no plates
	sliceConfig := `<?xml version="1.0" encoding="UTF-8"?>
<config>
</config>`

	path := createTestZip(t, map[string]string{
		"Metadata/slice_info.config": sliceConfig,
	})

	result, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if result != nil {
		t.Errorf("Parse returned non-nil result for empty plates: %s", string(result))
	}
}

func TestParse_PlateWithNoSlicerData(t *testing.T) {
	// Plate exists but has no prediction, weight, or filaments
	sliceConfig := `<?xml version="1.0" encoding="UTF-8"?>
<config>
  <plate>
    <metadata key="some_other_key" value="some_value"/>
  </plate>
</config>`

	path := createTestZip(t, map[string]string{
		"Metadata/slice_info.config": sliceConfig,
	})

	result, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if result != nil {
		t.Errorf("Parse returned non-nil result for plate with no slicer data: %s", string(result))
	}
}

func TestParse_PartialMetadata(t *testing.T) {
	// Has filaments but missing prediction/weight metadata
	sliceConfig := `<?xml version="1.0" encoding="UTF-8"?>
<config>
  <plate>
    <filament id="1" type="PLA" color="#FFFFFF" used_m="10.0" used_g="30.0"/>
  </plate>
</config>`

	path := createTestZip(t, map[string]string{
		"Metadata/slice_info.config": sliceConfig,
	})

	result, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if result == nil {
		t.Fatal("Parse returned nil for plate with filaments")
	}

	var profile SliceProfile
	if err := json.Unmarshal(result, &profile); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	// Should have filament data even if metadata is missing
	if len(profile.Filaments) != 1 {
		t.Fatalf("len(Filaments) = %d, want 1", len(profile.Filaments))
	}
	// PrintTimeSeconds and WeightGrams should be zero (default)
	if profile.PrintTimeSeconds != 0 {
		t.Errorf("PrintTimeSeconds = %d, want 0", profile.PrintTimeSeconds)
	}
	if profile.WeightGrams != 0 {
		t.Errorf("WeightGrams = %f, want 0", profile.WeightGrams)
	}
}

func TestParse_OnlyPrediction(t *testing.T) {
	// Has prediction but no weight or filaments
	sliceConfig := `<?xml version="1.0" encoding="UTF-8"?>
<config>
  <plate>
    <metadata key="prediction" value="1800"/>
  </plate>
</config>`

	path := createTestZip(t, map[string]string{
		"Metadata/slice_info.config": sliceConfig,
	})

	result, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if result == nil {
		t.Fatal("Parse returned nil for plate with prediction")
	}

	var profile SliceProfile
	if err := json.Unmarshal(result, &profile); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if profile.PrintTimeSeconds != 1800 {
		t.Errorf("PrintTimeSeconds = %d, want 1800", profile.PrintTimeSeconds)
	}
}

func TestParse_MalformedXML(t *testing.T) {
	sliceConfig := `<?xml version="1.0" encoding="UTF-8"?>
<config>
  <plate>
    <metadata key="prediction" value="1800"
  </plate>
</config>`

	path := createTestZip(t, map[string]string{
		"Metadata/slice_info.config": sliceConfig,
	})

	result, err := Parse(path)
	if err == nil {
		t.Errorf("Parse should return error for malformed XML, got result: %s", string(result))
	}
}

func TestParse_InvalidZipFile(t *testing.T) {
	// Create a file that's not a valid zip
	tmpFile, err := os.CreateTemp("", "test-*.3mf")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	tmpFile.WriteString("this is not a zip file")
	tmpFile.Close()
	t.Cleanup(func() {
		os.Remove(tmpFile.Name())
	})

	result, err := Parse(tmpFile.Name())
	if err == nil {
		t.Errorf("Parse should return error for invalid zip, got result: %s", string(result))
	}
}

func TestParse_NonExistentFile(t *testing.T) {
	result, err := Parse("/nonexistent/path/to/file.3mf")
	if err == nil {
		t.Errorf("Parse should return error for non-existent file, got result: %s", string(result))
	}
}

func TestParse_MultiplePlates_UsesFirst(t *testing.T) {
	// Multiple plates - should use first one
	sliceConfig := `<?xml version="1.0" encoding="UTF-8"?>
<config>
  <plate>
    <metadata key="prediction" value="1000"/>
    <metadata key="weight" value="10.0"/>
    <filament id="1" type="PLA" color="#FF0000" used_m="5.0" used_g="10.0"/>
  </plate>
  <plate>
    <metadata key="prediction" value="2000"/>
    <metadata key="weight" value="20.0"/>
    <filament id="1" type="PETG" color="#00FF00" used_m="10.0" used_g="20.0"/>
  </plate>
</config>`

	path := createTestZip(t, map[string]string{
		"Metadata/slice_info.config": sliceConfig,
	})

	result, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	var profile SliceProfile
	if err := json.Unmarshal(result, &profile); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	// Should get first plate's data
	if profile.PrintTimeSeconds != 1000 {
		t.Errorf("PrintTimeSeconds = %d, want 1000 (from first plate)", profile.PrintTimeSeconds)
	}
	if profile.WeightGrams != 10.0 {
		t.Errorf("WeightGrams = %f, want 10.0 (from first plate)", profile.WeightGrams)
	}
	if len(profile.Filaments) != 1 || profile.Filaments[0].Type != "PLA" {
		t.Errorf("Filaments should be from first plate (PLA), got: %+v", profile.Filaments)
	}
}

func TestParse_FloatParsing(t *testing.T) {
	// Test various float formats
	sliceConfig := `<?xml version="1.0" encoding="UTF-8"?>
<config>
  <plate>
    <metadata key="prediction" value="3661"/>
    <metadata key="weight" value="25.123456"/>
    <metadata key="nozzle_diameters" value="0.4"/>
    <filament id="1" type="PLA" color="#FF0000" used_m="8.123" used_g="25.456"/>
  </plate>
</config>`

	path := createTestZip(t, map[string]string{
		"Metadata/slice_info.config": sliceConfig,
	})

	result, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	var profile SliceProfile
	if err := json.Unmarshal(result, &profile); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	if profile.WeightGrams != 25.123456 {
		t.Errorf("WeightGrams = %f, want 25.123456", profile.WeightGrams)
	}
	if profile.Filaments[0].UsedMeters != 8.123 {
		t.Errorf("Filament.UsedMeters = %f, want 8.123", profile.Filaments[0].UsedMeters)
	}
	if profile.Filaments[0].UsedGrams != 25.456 {
		t.Errorf("Filament.UsedGrams = %f, want 25.456", profile.Filaments[0].UsedGrams)
	}
}

func TestParse_InvalidNumericValues(t *testing.T) {
	// Invalid numeric values should default to 0
	sliceConfig := `<?xml version="1.0" encoding="UTF-8"?>
<config>
  <plate>
    <metadata key="prediction" value="not_a_number"/>
    <metadata key="weight" value="also_not_a_number"/>
    <filament id="1" type="PLA" color="#FF0000" used_m="invalid" used_g="invalid"/>
  </plate>
</config>`

	path := createTestZip(t, map[string]string{
		"Metadata/slice_info.config": sliceConfig,
	})

	result, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	var profile SliceProfile
	if err := json.Unmarshal(result, &profile); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}

	// Values should be 0 (default) when parsing fails
	if profile.PrintTimeSeconds != 0 {
		t.Errorf("PrintTimeSeconds = %d, want 0 for invalid value", profile.PrintTimeSeconds)
	}
	if profile.WeightGrams != 0 {
		t.Errorf("WeightGrams = %f, want 0 for invalid value", profile.WeightGrams)
	}
	// Filament exists because it has type/color (even with invalid numeric values)
	if len(profile.Filaments) != 1 {
		t.Fatalf("len(Filaments) = %d, want 1", len(profile.Filaments))
	}
	if profile.Filaments[0].UsedGrams != 0 {
		t.Errorf("Filament.UsedGrams = %f, want 0 for invalid value", profile.Filaments[0].UsedGrams)
	}
}

func TestParse_RealWorldPath(t *testing.T) {
	// Test that Parse handles paths with special characters
	tmpDir := t.TempDir()
	specialPath := filepath.Join(tmpDir, "test file (1).3mf")

	sliceConfig := `<?xml version="1.0" encoding="UTF-8"?>
<config>
  <plate>
    <metadata key="prediction" value="100"/>
  </plate>
</config>`

	f, err := os.Create(specialPath)
	if err != nil {
		t.Fatalf("create file: %v", err)
	}
	w := zip.NewWriter(f)
	fw, _ := w.Create("Metadata/slice_info.config")
	fw.Write([]byte(sliceConfig))
	w.Close()
	f.Close()

	result, err := Parse(specialPath)
	if err != nil {
		t.Fatalf("Parse returned error for path with special chars: %v", err)
	}
	if result == nil {
		t.Error("Parse returned nil for valid file with special path")
	}
}
