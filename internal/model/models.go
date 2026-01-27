package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ProjectStatus represents the status of a project.
type ProjectStatus string

const (
	ProjectStatusDraft       ProjectStatus = "draft"
	ProjectStatusActive      ProjectStatus = "active"
	ProjectStatusCompleted   ProjectStatus = "completed"
	ProjectStatusReadyToShip ProjectStatus = "ready_to_ship"
	ProjectStatusShipped     ProjectStatus = "shipped"
	ProjectStatusArchived    ProjectStatus = "archived"
)

// Project represents a maker project containing parts to be printed.
type Project struct {
	ID              uuid.UUID     `json:"id"`
	Name            string        `json:"name"`
	Description     string        `json:"description"`
	Status          ProjectStatus `json:"status"`
	TargetDate      *time.Time    `json:"target_date,omitempty"`
	Tags            []string      `json:"tags"`
	TemplateID      *uuid.UUID    `json:"template_id,omitempty"`
	Source          string        `json:"source"`
	ExternalOrderID string        `json:"external_order_id,omitempty"`
	CustomerNotes   string        `json:"customer_notes,omitempty"`
	ShippedAt       *time.Time    `json:"shipped_at,omitempty"`
	TrackingNumber  string        `json:"tracking_number,omitempty"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
}

// PrintProfile represents a symbolic slicer profile.
type PrintProfile string

const (
	PrintProfileStandard PrintProfile = "standard"
	PrintProfileDetailed PrintProfile = "detailed"
	PrintProfileFast     PrintProfile = "fast"
	PrintProfileStrong   PrintProfile = "strong"
	PrintProfileCustom   PrintProfile = "custom"
)

// PrinterConstraints defines hardware requirements for a recipe.
type PrinterConstraints struct {
	MinBedSize        *BuildVolume `json:"min_bed_size,omitempty"`
	NozzleDiameters   []float64    `json:"nozzle_diameters,omitempty"`
	RequiresEnclosure bool         `json:"requires_enclosure"`
	RequiresAMS       bool         `json:"requires_ams"`
	PrinterTags       []string     `json:"printer_tags,omitempty"`
}

// ColorSpec defines how to match material color.
type ColorSpec struct {
	Mode string `json:"mode"` // "exact", "category", "any"
	Hex  string `json:"hex,omitempty"`
	Name string `json:"name,omitempty"`
}

// RecipeMaterial represents a material requirement for a recipe.
type RecipeMaterial struct {
	ID            uuid.UUID    `json:"id"`
	RecipeID      uuid.UUID    `json:"recipe_id"`
	MaterialType  MaterialType `json:"material_type"`
	ColorSpec     *ColorSpec   `json:"color_spec,omitempty"`
	WeightGrams   float64      `json:"weight_grams"`
	AMSPosition   *int         `json:"ams_position,omitempty"`
	SequenceOrder int          `json:"sequence_order"`
	Notes         string       `json:"notes,omitempty"`
	CreatedAt     time.Time    `json:"created_at"`
}

// RecipeCostEstimate represents the cost breakdown for a recipe.
type RecipeCostEstimate struct {
	MaterialCostCents   int                           `json:"material_cost_cents"`
	TimeCostCents       int                           `json:"time_cost_cents"`
	LaborCostCents      int                           `json:"labor_cost_cents"`
	TotalCostCents      int                           `json:"total_cost_cents"`
	EstimatedPrintTime  int                           `json:"estimated_print_time_seconds"`
	LaborMinutes        int                           `json:"labor_minutes"`
	MaterialBreakdown   []RecipeMaterialCostBreakdown `json:"material_breakdown"`
	HourlyRateCents     int                           `json:"hourly_rate_cents"`
	LaborRateCents      int                           `json:"labor_rate_cents"`
	// Margin calculation
	SalePriceCents     int     `json:"sale_price_cents"`
	GrossMarginCents   int     `json:"gross_margin_cents"`
	GrossMarginPercent float64 `json:"gross_margin_percent"`
}

// RecipeMaterialCostBreakdown shows cost for each material in a recipe.
type RecipeMaterialCostBreakdown struct {
	MaterialType string  `json:"material_type"`
	WeightGrams  float64 `json:"weight_grams"`
	CostCents    int     `json:"cost_cents"`
	ColorName    string  `json:"color_name,omitempty"`
}

// Template represents a reusable project blueprint for order fulfillment (aka Recipe).
type Template struct {
	ID                     uuid.UUID           `json:"id"`
	Name                   string              `json:"name"`
	Description            string              `json:"description"`
	SKU                    string              `json:"sku"`
	Tags                   []string            `json:"tags"`
	MaterialType           MaterialType        `json:"material_type"`
	EstimatedMaterialGrams float64             `json:"estimated_material_grams"`
	PreferredPrinterID     *uuid.UUID          `json:"preferred_printer_id,omitempty"`
	AllowAnyPrinter        bool                `json:"allow_any_printer"`
	QuantityPerOrder       int                 `json:"quantity_per_order"`
	PostProcessChecklist   []string            `json:"post_process_checklist"`
	IsActive               bool                `json:"is_active"`
	PrinterConstraints     *PrinterConstraints `json:"printer_constraints,omitempty"`
	PrintProfile           PrintProfile        `json:"print_profile"`
	EstimatedPrintSeconds  int                 `json:"estimated_print_seconds"`
	// Pricing fields for margin calculation
	LaborMinutes             int `json:"labor_minutes"`
	SalePriceCents           int `json:"sale_price_cents"`
	MaterialCostPerGramCents int `json:"material_cost_per_gram_cents"`
	Version                int                 `json:"version"`
	ArchivedAt             *time.Time          `json:"archived_at,omitempty"`
	CreatedAt              time.Time           `json:"created_at"`
	UpdatedAt              time.Time           `json:"updated_at"`
	Designs                []TemplateDesign    `json:"designs,omitempty"`
	Materials              []RecipeMaterial    `json:"materials,omitempty"`
}

// CostBreakdown contains the cost and margin breakdown for a recipe/order.
type CostBreakdown struct {
	MaterialCostCents  int     `json:"material_cost_cents"`
	LaborCostCents     int     `json:"labor_cost_cents"`
	TotalCostCents     int     `json:"total_cost_cents"`
	SalePriceCents     int     `json:"sale_price_cents"`
	GrossMarginCents   int     `json:"gross_margin_cents"`
	GrossMarginPercent float64 `json:"gross_margin_percent"`
	// Breakdown details
	MaterialGrams    float64 `json:"material_grams"`
	PrintTimeMinutes int     `json:"print_time_minutes"`
	LaborMinutes     int     `json:"labor_minutes"`
}

// TemplateDesign represents a design file linked to a template.
type TemplateDesign struct {
	ID            uuid.UUID `json:"id"`
	TemplateID    uuid.UUID `json:"template_id"`
	DesignID      uuid.UUID `json:"design_id"`
	IsPrimary     bool      `json:"is_primary"`
	Quantity      int       `json:"quantity"`
	SequenceOrder int       `json:"sequence_order"`
	Notes         string    `json:"notes"`
	CreatedAt     time.Time `json:"created_at"`
	Design        *Design   `json:"design,omitempty"`
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
	ConnectionTypeBambuLAN   ConnectionType = "bambu_lan"
	ConnectionTypeBambuCloud ConnectionType = "bambu_cloud"
	ConnectionTypeMoonraker  ConnectionType = "moonraker"
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
	ID                uuid.UUID      `json:"id"`
	Name              string         `json:"name"`
	Model             string         `json:"model"`
	Manufacturer      string         `json:"manufacturer"`
	ConnectionType    ConnectionType `json:"connection_type"`
	ConnectionURI     string         `json:"connection_uri"`
	APIKey            string         `json:"api_key,omitempty"`
	SerialNumber      string         `json:"serial_number,omitempty"`
	Status            PrinterStatus  `json:"status"`
	BuildVolume       *BuildVolume   `json:"build_volume,omitempty"`
	NozzleDiameter    float64        `json:"nozzle_diameter"`
	Location          string         `json:"location"`
	Notes             string         `json:"notes"`
	MinMaterialPercent int           `json:"min_material_percent"` // Minimum % before warning (default 10)
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
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
	AMS         *AMSState     `json:"ams,omitempty"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// AMSState represents the complete AMS (Automatic Material System) state.
type AMSState struct {
	Units         []AMSUnit `json:"units"`
	CurrentTray   string    `json:"current_tray,omitempty"`
	ExternalSpool *AMSTray  `json:"external_spool,omitempty"`
}

// AMSUnit represents a single AMS unit (typically holds 4 trays).
type AMSUnit struct {
	ID       int       `json:"id"`
	Humidity int       `json:"humidity"`
	Temp     float64   `json:"temp"`
	Trays    []AMSTray `json:"trays"`
}

// AMSTray represents a single tray/slot in an AMS unit.
type AMSTray struct {
	ID            int     `json:"id"`
	MaterialType  string  `json:"material_type"`
	Color         string  `json:"color"`
	ColorHex      string  `json:"color_hex,omitempty"`
	Remain        int     `json:"remain"`        // Remaining percentage (0-100)
	TagUID        string  `json:"tag_uid,omitempty"`
	Brand         string  `json:"brand,omitempty"`
	NozzleTempMin int     `json:"nozzle_temp_min,omitempty"`
	NozzleTempMax int     `json:"nozzle_temp_max,omitempty"`
	BedTemp       int     `json:"bed_temp,omitempty"`
	Empty         bool    `json:"empty"`
}

// MaterialSnapshot captures the AMS state at job start for auditing.
type MaterialSnapshot struct {
	CapturedAt    time.Time `json:"captured_at"`
	SelectedTray  int       `json:"selected_tray"`
	MaterialType  string    `json:"material_type"`
	Color         string    `json:"color"`
	RemainPercent int       `json:"remain_percent"`
	Brand         string    `json:"brand,omitempty"`
	AMSState      *AMSState `json:"ams_state,omitempty"`
}

// MaterialValidation contains the result of pre-start material checks.
type MaterialValidation struct {
	Valid    bool     `json:"valid"`
	Warnings []string `json:"warnings,omitempty"`
	Errors   []string `json:"errors,omitempty"`
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
	PrintJobStatusAssigned  PrintJobStatus = "assigned"  // assigned to printer
	PrintJobStatusUploaded  PrintJobStatus = "uploaded"  // file sent to printer
	PrintJobStatusPrinting  PrintJobStatus = "printing"
	PrintJobStatusPaused    PrintJobStatus = "paused"
	PrintJobStatusCompleted PrintJobStatus = "completed"
	PrintJobStatusFailed    PrintJobStatus = "failed"
	PrintJobStatusCancelled PrintJobStatus = "cancelled"
)

// IsTerminal returns true if this status is a final state.
func (s PrintJobStatus) IsTerminal() bool {
	return s == PrintJobStatusCompleted || s == PrintJobStatusFailed || s == PrintJobStatusCancelled
}

// FailureCategory categorizes why a print job failed.
type FailureCategory string

const (
	FailureMechanical    FailureCategory = "mechanical"     // printer hardware issue
	FailureFilament      FailureCategory = "filament"       // filament jam, runout, tangle
	FailureAdhesion      FailureCategory = "adhesion"       // bed adhesion failure
	FailureThermal       FailureCategory = "thermal"        // thermal runaway, heating failure
	FailureNetwork       FailureCategory = "network"        // connection lost during print
	FailureUserCancelled FailureCategory = "user_cancelled" // user stopped the print
	FailureUnknown       FailureCategory = "unknown"        // unclassified failure
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

// PrintJob represents an immutable print job instance.
// Once created, core fields should not change. State changes are recorded as JobEvents.
type PrintJob struct {
	ID              uuid.UUID  `json:"id"`
	DesignID        uuid.UUID  `json:"design_id"`
	PrinterID       *uuid.UUID `json:"printer_id,omitempty"`
	MaterialSpoolID *uuid.UUID `json:"material_spool_id,omitempty"`
	ProjectID       *uuid.UUID `json:"project_id,omitempty"`
	Notes           string     `json:"notes"`
	CreatedAt       time.Time  `json:"created_at"`

	// Recipe context (optional, for SKU-based orders)
	RecipeID *uuid.UUID `json:"recipe_id,omitempty"`

	// Retry tracking
	AttemptNumber int        `json:"attempt_number"` // 1 = first attempt, 2+ = retries
	ParentJobID   *uuid.UUID `json:"parent_job_id,omitempty"`

	// Failure classification (set when job fails)
	FailureCategory *FailureCategory `json:"failure_category,omitempty"`

	// Time tracking
	EstimatedSeconds *int `json:"estimated_seconds,omitempty"`
	ActualSeconds    *int `json:"actual_seconds,omitempty"`

	// Cost tracking
	MaterialUsedGrams *float64 `json:"material_used_grams,omitempty"`
	CostCents         *int     `json:"cost_cents,omitempty"`

	// Material snapshot (AMS state captured at job start)
	MaterialSnapshot *MaterialSnapshot `json:"material_snapshot,omitempty"`

	// Computed fields (derived from events, not stored in print_jobs table)
	Status      PrintJobStatus `json:"status"`                   // computed from latest event
	Progress    float64        `json:"progress"`                 // computed from latest event
	StartedAt   *time.Time     `json:"started_at,omitempty"`     // computed from first printing event
	CompletedAt *time.Time     `json:"completed_at,omitempty"`   // computed from terminal event
	Outcome     *PrintOutcome  `json:"outcome,omitempty"`        // kept for backward compat
	Events      []JobEvent     `json:"events,omitempty"`         // full event timeline when requested
}

// NeedsAssignment returns true if the job is missing printer or spool assignment.
func (j *PrintJob) NeedsAssignment() bool {
	return j.PrinterID == nil || j.MaterialSpoolID == nil
}

// IsAssigned returns true if the job has both printer and spool assigned.
func (j *PrintJob) IsAssigned() bool {
	return j.PrinterID != nil && j.MaterialSpoolID != nil
}

// JobEventType represents types of job events.
type JobEventType string

const (
	JobEventQueued    JobEventType = "queued"     // job created and waiting
	JobEventAssigned  JobEventType = "assigned"   // assigned to a printer
	JobEventUploaded  JobEventType = "uploaded"   // file sent to printer
	JobEventStarted   JobEventType = "started"    // print started
	JobEventProgress  JobEventType = "progress"   // progress update (no status change)
	JobEventPaused    JobEventType = "paused"     // print paused
	JobEventResumed   JobEventType = "resumed"    // print resumed
	JobEventCompleted JobEventType = "completed"  // print finished successfully
	JobEventFailed    JobEventType = "failed"     // print failed
	JobEventCancelled JobEventType = "cancelled"  // print cancelled by user/system
	JobEventRetried   JobEventType = "retried"    // new job created as retry
)

// ActorType identifies what triggered an event.
type ActorType string

const (
	ActorUser    ActorType = "user"    // human user action
	ActorSystem  ActorType = "system"  // automated system action
	ActorPrinter ActorType = "printer" // printer-reported event
	ActorWebhook ActorType = "webhook" // external webhook trigger
)

// JobEvent represents an immutable event in the print job lifecycle.
// These are append-only records that form the complete history of a job.
type JobEvent struct {
	ID         uuid.UUID    `json:"id"`
	JobID      uuid.UUID    `json:"job_id"`
	EventType  JobEventType `json:"event_type"`
	OccurredAt time.Time    `json:"occurred_at"`

	// Status context
	Status   *PrintJobStatus `json:"status,omitempty"`   // resulting status (nil for progress events)
	Progress *float64        `json:"progress,omitempty"` // progress at time of event

	// Printer context (for assignment/transfer)
	PrinterID *uuid.UUID `json:"printer_id,omitempty"`

	// Error context (for failures)
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`

	// Actor tracking
	ActorType ActorType `json:"actor_type"`
	ActorID   string    `json:"actor_id,omitempty"`

	// Flexible metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

// NewJobEvent creates a new job event with sensible defaults.
func NewJobEvent(jobID uuid.UUID, eventType JobEventType, status *PrintJobStatus) *JobEvent {
	now := time.Now()
	return &JobEvent{
		ID:         uuid.New(),
		JobID:      jobID,
		EventType:  eventType,
		OccurredAt: now,
		Status:     status,
		ActorType:  ActorSystem,
		CreatedAt:  now,
	}
}

// WithActor sets the actor information on the event.
func (e *JobEvent) WithActor(actorType ActorType, actorID string) *JobEvent {
	e.ActorType = actorType
	e.ActorID = actorID
	return e
}

// WithError sets error information on the event.
func (e *JobEvent) WithError(code, message string) *JobEvent {
	e.ErrorCode = code
	e.ErrorMessage = message
	return e
}

// WithProgress sets the progress on the event.
func (e *JobEvent) WithProgress(progress float64) *JobEvent {
	e.Progress = &progress
	return e
}

// WithPrinter sets the printer context on the event.
func (e *JobEvent) WithPrinter(printerID uuid.UUID) *JobEvent {
	e.PrinterID = &printerID
	return e
}

// WithMetadata sets metadata on the event.
func (e *JobEvent) WithMetadata(metadata map[string]interface{}) *JobEvent {
	e.Metadata = metadata
	return e
}

// Legacy type aliases for backward compatibility
type PrintEventType = JobEventType
type PrintEvent = JobEvent

// Legacy constants for backward compatibility
const (
	EventJobCreated   = JobEventQueued
	EventJobStarted   = JobEventStarted
	EventJobProgress  = JobEventProgress
	EventJobPaused    = JobEventPaused
	EventJobResumed   = JobEventResumed
	EventJobCompleted = JobEventCompleted
	EventJobFailed    = JobEventFailed
	EventJobCancelled = JobEventCancelled
)

// WebSocket event types for real-time updates.
const (
	EventPrinterStateUpdated = "printer_state_updated"
	EventPrinterConnected    = "printer_connected"
	EventPrinterDisconnected = "printer_disconnected"
)

// Broadcaster defines the interface for broadcasting real-time events.
// This interface allows the printer package to broadcast without importing realtime.
type Broadcaster interface {
	Broadcast(event BroadcastEvent)
}

// BroadcastEvent represents an event to be sent to WebSocket clients.
type BroadcastEvent struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// File represents a stored file.
type File struct {
	ID           uuid.UUID `json:"id"`
	Hash         string    `json:"hash"`
	OriginalName string    `json:"original_name"`
	ContentType  string    `json:"content_type"`
	SizeBytes    int64     `json:"size_bytes"`
	StoragePath  string    `json:"storage_path"`
	CreatedAt    time.Time `json:"created_at"`
}

// ExpenseStatus represents the status of an expense record.
type ExpenseStatus string

const (
	ExpenseStatusPending   ExpenseStatus = "pending"
	ExpenseStatusConfirmed ExpenseStatus = "confirmed"
	ExpenseStatusRejected  ExpenseStatus = "rejected"
)

// ExpenseCategory represents categories for expenses.
type ExpenseCategory string

const (
	ExpenseCategoryFilament        ExpenseCategory = "filament"
	ExpenseCategoryParts           ExpenseCategory = "parts"
	ExpenseCategoryTools           ExpenseCategory = "tools"
	ExpenseCategoryShipping        ExpenseCategory = "shipping"
	ExpenseCategoryMarketplaceFees ExpenseCategory = "marketplace_fees"
	ExpenseCategorySubscription    ExpenseCategory = "subscription"
	ExpenseCategoryOther           ExpenseCategory = "other"
)

// Expense represents an expense record.
type Expense struct {
	ID               uuid.UUID       `json:"id"`
	OccurredAt       time.Time       `json:"occurred_at"`
	Vendor           string          `json:"vendor"`
	SubtotalCents    int             `json:"subtotal_cents"`
	TaxCents         int             `json:"tax_cents"`
	ShippingCents    int             `json:"shipping_cents"`
	TotalCents       int             `json:"total_cents"`
	Currency         string          `json:"currency"`
	Category         ExpenseCategory `json:"category"`
	Notes            string          `json:"notes"`
	ReceiptFileID    *uuid.UUID      `json:"receipt_file_id,omitempty"`
	ReceiptFilePath  string          `json:"receipt_file_path,omitempty"`
	Status           ExpenseStatus   `json:"status"`
	RawOCRText       string          `json:"raw_ocr_text,omitempty"`
	RawAIResponse    json.RawMessage `json:"raw_ai_response,omitempty"`
	Confidence       int             `json:"confidence"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`

	// Joined data
	Items []ExpenseItem `json:"items,omitempty"`
}

// ExpenseItemAction represents what action was taken for an expense item.
type ExpenseItemAction string

const (
	ExpenseItemActionNone         ExpenseItemAction = "none"
	ExpenseItemActionCreatedSpool ExpenseItemAction = "created_spool"
	ExpenseItemActionMatchedSpool ExpenseItemAction = "matched_spool"
	ExpenseItemActionSkipped      ExpenseItemAction = "skipped"
)

// FilamentMetadata holds parsed attributes for filament items.
type FilamentMetadata struct {
	Brand        string  `json:"brand,omitempty"`
	MaterialType string  `json:"material_type,omitempty"` // PLA, PETG, ABS, etc.
	Color        string  `json:"color,omitempty"`
	ColorHex     string  `json:"color_hex,omitempty"`
	WeightGrams  float64 `json:"weight_grams,omitempty"`
	DiameterMM   float64 `json:"diameter_mm,omitempty"`
}

// ExpenseItem represents a line item in an expense.
type ExpenseItem struct {
	ID                uuid.UUID         `json:"id"`
	ExpenseID         uuid.UUID         `json:"expense_id"`
	Description       string            `json:"description"`
	Quantity          float64           `json:"quantity"`
	UnitPriceCents    int               `json:"unit_price_cents"`
	TotalPriceCents   int               `json:"total_price_cents"`
	SKU               string            `json:"sku,omitempty"`
	VendorItemID      string            `json:"vendor_item_id,omitempty"`
	Category          ExpenseCategory   `json:"category"`
	Metadata          *FilamentMetadata `json:"metadata,omitempty"`
	MatchedSpoolID    *uuid.UUID        `json:"matched_spool_id,omitempty"`
	MatchedMaterialID *uuid.UUID        `json:"matched_material_id,omitempty"`
	Confidence        int               `json:"confidence"`
	ActionTaken       ExpenseItemAction `json:"action_taken"`
	CreatedAt         time.Time         `json:"created_at"`
}

// SalesChannel represents where a sale was made.
type SalesChannel string

const (
	SalesChannelMarketplace SalesChannel = "marketplace"
	SalesChannelEtsy        SalesChannel = "etsy"
	SalesChannelWebsite     SalesChannel = "website"
	SalesChannelDirect      SalesChannel = "direct"
	SalesChannelOther       SalesChannel = "other"
)

// Sale represents a revenue record.
type Sale struct {
	ID                   uuid.UUID    `json:"id"`
	OccurredAt           time.Time    `json:"occurred_at"`
	Channel              SalesChannel `json:"channel"`
	Platform             string       `json:"platform"`
	GrossCents           int          `json:"gross_cents"`
	FeesCents            int          `json:"fees_cents"`
	ShippingChargedCents int          `json:"shipping_charged_cents"`
	ShippingCostCents    int          `json:"shipping_cost_cents"`
	TaxCollectedCents    int          `json:"tax_collected_cents"`
	NetCents             int          `json:"net_cents"`
	Currency             string       `json:"currency"`
	ProjectID            *uuid.UUID   `json:"project_id,omitempty"`
	OrderReference       string       `json:"order_reference,omitempty"`
	CustomerName         string       `json:"customer_name,omitempty"`
	ItemDescription      string       `json:"item_description"`
	Quantity             int          `json:"quantity"`
	Notes                string       `json:"notes"`
	CreatedAt            time.Time    `json:"created_at"`
	UpdatedAt            time.Time    `json:"updated_at"`
}

// ParsedReceipt represents the AI-extracted data from a receipt.
type ParsedReceipt struct {
	Vendor       string             `json:"vendor"`
	Date         string             `json:"date"`
	SubtotalCents int               `json:"subtotal_cents"`
	TaxCents     int                `json:"tax_cents"`
	ShippingCents int               `json:"shipping_cents"`
	TotalCents   int                `json:"total_cents"`
	Currency     string             `json:"currency"`
	Items        []ParsedReceiptItem `json:"items"`
	Confidence   int                `json:"confidence"`
	RawText      string             `json:"raw_text,omitempty"`
}

// ParsedReceiptItem represents a parsed line item from a receipt.
type ParsedReceiptItem struct {
	Description    string           `json:"description"`
	Quantity       float64          `json:"quantity"`
	UnitPriceCents int              `json:"unit_price_cents"`
	TotalPriceCents int             `json:"total_price_cents"`
	Category       ExpenseCategory  `json:"category"`
	IsFilament     bool             `json:"is_filament"`
	Filament       *FilamentMetadata `json:"filament,omitempty"`
	Confidence     int              `json:"confidence"`
}

// BambuCloudAuth stores authentication credentials for Bambu Cloud API.
type BambuCloudAuth struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	AccessToken  string     `json:"access_token"`
	MQTTUsername string     `json:"mqtt_username"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

