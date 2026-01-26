package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ProjectStatus represents the status of a project.
type ProjectStatus string

const (
	ProjectStatusDraft     ProjectStatus = "draft"
	ProjectStatusActive    ProjectStatus = "active"
	ProjectStatusCompleted ProjectStatus = "completed"
	ProjectStatusArchived  ProjectStatus = "archived"
)

// Project represents a maker project containing parts to be printed.
type Project struct {
	ID          uuid.UUID     `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Status      ProjectStatus `json:"status"`
	TargetDate  *time.Time    `json:"target_date,omitempty"`
	Tags        []string      `json:"tags"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// PartStatus represents the status of a part.
type PartStatus string

const (
	PartStatusDesign   PartStatus = "design"
	PartStatusPrinting PartStatus = "printing"
	PartStatusComplete PartStatus = "complete"
)

// Part represents a discrete printable component of a project.
type Part struct {
	ID          uuid.UUID  `json:"id"`
	ProjectID   uuid.UUID  `json:"project_id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Quantity    int        `json:"quantity"`
	Status      PartStatus `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// FileType represents supported design file types.
type FileType string

const (
	FileTypeSTL  FileType = "stl"
	FileType3MF  FileType = "3mf"
	FileTypeGCODE FileType = "gcode"
)

// Design represents a versioned design file for a part.
type Design struct {
	ID            uuid.UUID       `json:"id"`
	PartID        uuid.UUID       `json:"part_id"`
	Version       int             `json:"version"`
	FileID        uuid.UUID       `json:"file_id"`
	FileName      string          `json:"file_name"`
	FileHash      string          `json:"file_hash"`
	FileSizeBytes int64           `json:"file_size_bytes"`
	FileType      FileType        `json:"file_type"`
	Notes         string          `json:"notes"`
	SliceProfile  json.RawMessage `json:"slice_profile,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

// ConnectionType represents how a printer is connected.
type ConnectionType string

const (
	ConnectionTypeManual    ConnectionType = "manual"
	ConnectionTypeOctoPrint ConnectionType = "octoprint"
	ConnectionTypeBambuLAN  ConnectionType = "bambu_lan"
	ConnectionTypeMoonraker ConnectionType = "moonraker"
	ConnectionTypeChiTu     ConnectionType = "chitu"      // Elegoo, Anycubic, Phrozen resin printers
)

// PrinterStatus represents the current state of a printer.
type PrinterStatus string

const (
	PrinterStatusIdle     PrinterStatus = "idle"
	PrinterStatusPrinting PrinterStatus = "printing"
	PrinterStatusPaused   PrinterStatus = "paused"
	PrinterStatusError    PrinterStatus = "error"
	PrinterStatusOffline  PrinterStatus = "offline"
)

// BuildVolume represents the print area dimensions.
type BuildVolume struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// Printer represents a 3D printer in the farm.
type Printer struct {
	ID             uuid.UUID      `json:"id"`
	Name           string         `json:"name"`
	Model          string         `json:"model"`
	Manufacturer   string         `json:"manufacturer"`
	ConnectionType ConnectionType `json:"connection_type"`
	ConnectionURI  string         `json:"connection_uri"`
	APIKey         string         `json:"api_key,omitempty"`
	Status         PrinterStatus  `json:"status"`
	BuildVolume    *BuildVolume   `json:"build_volume,omitempty"`
	NozzleDiameter float64        `json:"nozzle_diameter"`
	Location       string         `json:"location"`
	Notes          string         `json:"notes"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// PrinterState represents real-time state from a connected printer.
type PrinterState struct {
	PrinterID   uuid.UUID     `json:"printer_id"`
	Status      PrinterStatus `json:"status"`
	Progress    float64       `json:"progress"`
	CurrentFile string        `json:"current_file,omitempty"`
	TimeLeft    int           `json:"time_left,omitempty"`
	BedTemp     float64       `json:"bed_temp,omitempty"`
	NozzleTemp  float64       `json:"nozzle_temp,omitempty"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// MaterialType represents the type of printing material.
type MaterialType string

const (
	MaterialTypePLA  MaterialType = "pla"
	MaterialTypePETG MaterialType = "petg"
	MaterialTypeABS  MaterialType = "abs"
	MaterialTypeASA  MaterialType = "asa"
	MaterialTypeTPU  MaterialType = "tpu"
)

// TempRange represents a temperature range.
type TempRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

// Material represents a type of printing material in the catalog.
type Material struct {
	ID           uuid.UUID    `json:"id"`
	Name         string       `json:"name"`
	Type         MaterialType `json:"type"`
	Manufacturer string       `json:"manufacturer"`
	Color        string       `json:"color"`
	ColorHex     string       `json:"color_hex"`
	Density      float64      `json:"density"`
	CostPerKg    float64      `json:"cost_per_kg"`
	PrintTemp    *TempRange   `json:"print_temp,omitempty"`
	BedTemp      *TempRange   `json:"bed_temp,omitempty"`
	Notes        string       `json:"notes"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// SpoolStatus represents the status of a material spool.
type SpoolStatus string

const (
	SpoolStatusNew      SpoolStatus = "new"
	SpoolStatusInUse    SpoolStatus = "in_use"
	SpoolStatusLow      SpoolStatus = "low"
	SpoolStatusEmpty    SpoolStatus = "empty"
	SpoolStatusArchived SpoolStatus = "archived"
)

// MaterialSpool represents a physical spool of material (inventory).
type MaterialSpool struct {
	ID              uuid.UUID   `json:"id"`
	MaterialID      uuid.UUID   `json:"material_id"`
	InitialWeight   float64     `json:"initial_weight"`
	RemainingWeight float64     `json:"remaining_weight"`
	PurchaseDate    *time.Time  `json:"purchase_date,omitempty"`
	PurchaseCost    float64     `json:"purchase_cost"`
	Location        string      `json:"location"`
	Status          SpoolStatus `json:"status"`
	Notes           string      `json:"notes"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
}

// PrintJobStatus represents the status of a print job.
type PrintJobStatus string

const (
	PrintJobStatusQueued    PrintJobStatus = "queued"
	PrintJobStatusSending   PrintJobStatus = "sending"
	PrintJobStatusPrinting  PrintJobStatus = "printing"
	PrintJobStatusCompleted PrintJobStatus = "completed"
	PrintJobStatusFailed    PrintJobStatus = "failed"
	PrintJobStatusCancelled PrintJobStatus = "cancelled"
)

// PrintOutcome captures the result of a print job.
type PrintOutcome struct {
	Success       bool     `json:"success"`
	QualityRating *int     `json:"quality_rating,omitempty"`
	ActualWeight  *float64 `json:"actual_weight,omitempty"`
	ActualTime    *int     `json:"actual_time,omitempty"`
	FailureReason string   `json:"failure_reason,omitempty"`
	Photos        []string `json:"photos,omitempty"`
	Notes         string   `json:"notes,omitempty"`
	MaterialUsed  float64  `json:"material_used"`
	MaterialCost  float64  `json:"material_cost"`
}

// PrintJob represents a print job assignment.
type PrintJob struct {
	ID              uuid.UUID       `json:"id"`
	DesignID        uuid.UUID       `json:"design_id"`
	PrinterID       uuid.UUID       `json:"printer_id"`
	MaterialSpoolID uuid.UUID       `json:"material_spool_id"`
	Status          PrintJobStatus  `json:"status"`
	Progress        float64         `json:"progress"`
	StartedAt       *time.Time      `json:"started_at,omitempty"`
	CompletedAt     *time.Time      `json:"completed_at,omitempty"`
	Outcome         *PrintOutcome   `json:"outcome,omitempty"`
	Notes           string          `json:"notes"`
	CreatedAt       time.Time       `json:"created_at"`
}

// PrintEventType represents types of print events.
type PrintEventType string

const (
	EventJobCreated   PrintEventType = "job_created"
	EventJobStarted   PrintEventType = "job_started"
	EventJobProgress  PrintEventType = "job_progress"
	EventJobPaused    PrintEventType = "job_paused"
	EventJobResumed   PrintEventType = "job_resumed"
	EventJobCompleted PrintEventType = "job_completed"
	EventJobFailed    PrintEventType = "job_failed"
	EventJobCancelled PrintEventType = "job_cancelled"
)

// PrintEvent represents an immutable event in the print job lifecycle.
type PrintEvent struct {
	ID         uuid.UUID       `json:"id"`
	PrintJobID uuid.UUID       `json:"print_job_id"`
	EventType  PrintEventType  `json:"event_type"`
	Timestamp  time.Time       `json:"timestamp"`
	Data       json.RawMessage `json:"data,omitempty"`
}

// File represents a stored file.
type File struct {
	ID          uuid.UUID `json:"id"`
	Hash        string    `json:"hash"`
	OriginalName string   `json:"original_name"`
	ContentType string    `json:"content_type"`
	SizeBytes   int64     `json:"size_bytes"`
	StoragePath string    `json:"storage_path"`
	CreatedAt   time.Time `json:"created_at"`
}

