package threemf

import (
	"archive/zip"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"strconv"
)

// SliceProfile holds extracted slicer metadata from a 3MF file.
type SliceProfile struct {
	PrintTimeSeconds int             `json:"print_time_seconds"`
	WeightGrams      float64         `json:"weight_grams"`
	PrinterModel     string          `json:"printer_model,omitempty"`
	NozzleDiameter   float64         `json:"nozzle_diameter,omitempty"`
	Filaments        []FilamentUsage `json:"filaments"`
}

// FilamentUsage describes per-filament usage from slicer output.
type FilamentUsage struct {
	Type      string  `json:"type"`
	Color     string  `json:"color"`
	UsedGrams float64 `json:"used_grams"`
	UsedMeters float64 `json:"used_meters"`
}

// XML structures for Metadata/slice_info.config

type sliceInfoConfig struct {
	XMLName xml.Name     `xml:"config"`
	Plates  []slicePlate `xml:"plate"`
}

type slicePlate struct {
	Metadata  []sliceMetadata  `xml:"metadata"`
	Filaments []sliceFilament  `xml:"filament"`
}

type sliceMetadata struct {
	Key   string `xml:"key,attr"`
	Value string `xml:"value,attr"`
}

type sliceFilament struct {
	ID             string `xml:"id,attr"`
	Type           string `xml:"type,attr"`
	Color          string `xml:"color,attr"`
	UsedMeters     string `xml:"used_m,attr"`
	UsedGrams      string `xml:"used_g,attr"`
	NozzleDiameter string `xml:"nozzle_diameter,attr"`
}

// Parse opens a 3MF file (ZIP archive) and extracts slicer metadata
// from Metadata/slice_info.config. Returns nil, nil if the file is not
// a sliced export (no slice_info.config or no plate data).
func Parse(filePath string) (json.RawMessage, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}
	defer r.Close()

	// Find slice_info.config
	var configFile *zip.File
	for _, f := range r.File {
		if f.Name == "Metadata/slice_info.config" {
			configFile = f
			break
		}
	}
	if configFile == nil {
		return nil, nil // not a sliced file
	}

	rc, err := configFile.Open()
	if err != nil {
		return nil, fmt.Errorf("open slice_info.config: %w", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("read slice_info.config: %w", err)
	}

	var config sliceInfoConfig
	if err := xml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse slice_info.config: %w", err)
	}

	// No plate data means unsliced project file
	if len(config.Plates) == 0 {
		return nil, nil
	}

	// Use first plate (most 3MF exports have a single plate)
	plate := config.Plates[0]

	// Check if plate has meaningful slicer data
	metaMap := make(map[string]string, len(plate.Metadata))
	for _, m := range plate.Metadata {
		metaMap[m.Key] = m.Value
	}

	if metaMap["prediction"] == "" && metaMap["weight"] == "" && len(plate.Filaments) == 0 {
		return nil, nil // no slicer data
	}

	profile := SliceProfile{
		Filaments: make([]FilamentUsage, 0, len(plate.Filaments)),
	}

	if v, ok := metaMap["prediction"]; ok {
		profile.PrintTimeSeconds, _ = strconv.Atoi(v)
	}
	if v, ok := metaMap["weight"]; ok {
		profile.WeightGrams, _ = strconv.ParseFloat(v, 64)
	}
	if v, ok := metaMap["printer_model_id"]; ok {
		profile.PrinterModel = v
	}
	if v, ok := metaMap["nozzle_diameters"]; ok {
		profile.NozzleDiameter, _ = strconv.ParseFloat(v, 64)
	}

	for _, f := range plate.Filaments {
		fu := FilamentUsage{
			Type:  f.Type,
			Color: f.Color,
		}
		fu.UsedGrams, _ = strconv.ParseFloat(f.UsedGrams, 64)
		fu.UsedMeters, _ = strconv.ParseFloat(f.UsedMeters, 64)
		profile.Filaments = append(profile.Filaments, fu)
	}

	raw, err := json.Marshal(profile)
	if err != nil {
		return nil, fmt.Errorf("marshal slice profile: %w", err)
	}

	return json.RawMessage(raw), nil
}
