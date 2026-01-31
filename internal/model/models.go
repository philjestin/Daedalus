package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Project represents a maker project/product that tracks lifetime performance.
type Project struct {
	ID              uuid.UUID  `json:"id"`
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	TargetDate      *time.Time `json:"target_date,omitempty"`
	Tags            []string   `json:"tags"`
	TemplateID      *uuid.UUID `json:"template_id,omitempty"`
	Source          string     `json:"source"`
	ExternalOrderID string     `json:"external_order_id,omitempty"`
	CustomerNotes   string     `json:"customer_notes,omitempty"`
	OrderID         *uuid.UUID `json:"order_id,omitempty"`
	OrderItemID     *uuid.UUID `json:"order_item_id,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// ProjectSummary is a derived analytics object for a project.
// All fields are computed from jobs and sales — never stored.
type ProjectSummary struct {
	// Revenue (from Sales linked to this project)
	TotalRevenueCents int `json:"total_revenue_cents"`
	TotalFeesCents    int `json:"total_fees_cents"`
	NetRevenueCents   int `json:"net_revenue_cents"`
	SalesCount        int `json:"sales_count"`

	// Cost breakdown
	UnitCostCents         int `json:"unit_cost_cents"`          // cost to produce one unit
	TotalCostCents        int `json:"total_cost_cents"`         // unit_cost × max(sales_count, 1)
	PrinterTimeCostCents  int `json:"printer_time_cost_cents"`
	MaterialCostCents     int `json:"material_cost_cents"`

	// Profit
	GrossProfitCents   int     `json:"gross_profit_cents"`
	GrossMarginPercent float64 `json:"gross_margin_percent"`

	// Print time
	TotalPrintSeconds    int     `json:"total_print_seconds"`
	AvgPrintSeconds      int     `json:"avg_print_seconds"`
	ProfitPerHourCents   int     `json:"profit_per_hour_cents"`

	// Performance
	JobCount     int     `json:"job_count"`
	CompletedCount int   `json:"completed_count"`
	FailedCount  int     `json:"failed_count"`
	SuccessRate  float64 `json:"success_rate"`

	// Material
	TotalMaterialGrams float64 `json:"total_material_grams"`

	// Estimated values (from slice profiles and supplies)
	EstimatedMaterialCostCents int     `json:"estimated_material_cost_cents"`
	EstimatedMaterialGrams     float64 `json:"estimated_material_grams"`
	EstimatedPrintSeconds      int     `json:"estimated_print_seconds"`
	SupplyCostCents            int     `json:"supply_cost_cents"`
}

// ProjectSupply represents a non-printed purchased item in a project's bill of materials.
type ProjectSupply struct {
	ID            uuid.UUID  `json:"id"`
	ProjectID     uuid.UUID  `json:"project_id"`
	Name          string     `json:"name"`
	UnitCostCents int        `json:"unit_cost_cents"`
	Quantity      int        `json:"quantity"`
	Notes         string     `json:"notes"`
	MaterialID    *uuid.UUID `json:"material_id,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// SliceProfileData represents parsed data from a 3MF slice profile.
type SliceProfileData struct {
	PrintTimeSeconds int              `json:"print_time_seconds"`
	WeightGrams      float64          `json:"weight_grams"`
	PrinterModel     string           `json:"printer_model,omitempty"`
	NozzleDiameter   float64          `json:"nozzle_diameter,omitempty"`
	Filaments        []FilamentUsage  `json:"filaments"`
}

// FilamentUsage represents filament usage data from a slice profile.
type FilamentUsage struct {
	Type      string  `json:"type"`
	Color     string  `json:"color"`
	UsedGrams float64 `json:"used_grams"`
	UsedMeters float64 `json:"used_meters"`
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
	SupplyCostCents     int                           `json:"supply_cost_cents"`
	TotalCostCents      int                           `json:"total_cost_cents"`
	EstimatedPrintTime  int                           `json:"estimated_print_time_seconds"`
	LaborMinutes        int                           `json:"labor_minutes"`
	MaterialBreakdown   []RecipeMaterialCostBreakdown `json:"material_breakdown"`
	SupplyBreakdown     []RecipeSupplyCostBreakdown   `json:"supply_breakdown,omitempty"`
	HourlyRateCents     int                           `json:"hourly_rate_cents"`
	LaborRateCents      int                           `json:"labor_rate_cents"`
	PrinterName         string                        `json:"printer_name,omitempty"`
	// Margin calculation
	SalePriceCents     int     `json:"sale_price_cents"`
	GrossMarginCents   int     `json:"gross_margin_cents"`
	GrossMarginPercent float64 `json:"gross_margin_percent"`
	ProfitPerHourCents int     `json:"profit_per_hour_cents"`
}

// RecipeSupplyCostBreakdown shows cost for each supply item in a recipe.
type RecipeSupplyCostBreakdown struct {
	Name          string `json:"name"`
	UnitCostCents int    `json:"unit_cost_cents"`
	Quantity      int    `json:"quantity"`
	TotalCents    int    `json:"total_cents"`
}

// RecipeSupply represents a non-printed purchased item in a recipe's bill of materials.
type RecipeSupply struct {
	ID            uuid.UUID  `json:"id"`
	RecipeID      uuid.UUID  `json:"recipe_id"`
	Name          string     `json:"name"`
	UnitCostCents int        `json:"unit_cost_cents"`
	Quantity      int        `json:"quantity"`
	MaterialID    *uuid.UUID `json:"material_id,omitempty"`
	Notes         string     `json:"notes,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
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
	Supplies               []RecipeSupply      `json:"supplies,omitempty"`
}

// TemplateAnalytics contains aggregated performance metrics from projects created from a template.
type TemplateAnalytics struct {
	TemplateID   uuid.UUID `json:"template_id"`
	ProjectCount int       `json:"project_count"`

	// Revenue
	TotalRevenueCents int `json:"total_revenue_cents"`
	TotalFeesCents    int `json:"total_fees_cents"`
	NetRevenueCents   int `json:"net_revenue_cents"`
	TotalSalesCount   int `json:"total_sales_count"`

	// Costs
	TotalCostCents       int `json:"total_cost_cents"`
	AvgUnitCostCents     int `json:"avg_unit_cost_cents"`
	TotalPrinterTimeCost int `json:"total_printer_time_cost"`
	TotalMaterialCost    int `json:"total_material_cost"`
	TotalSupplyCost      int `json:"total_supply_cost"`

	// Profit
	TotalGrossProfitCents  int     `json:"total_gross_profit_cents"`
	AvgGrossMarginPercent  float64 `json:"avg_gross_margin_percent"`
	ProfitPerHourCents     int     `json:"profit_per_hour_cents"`

	// Performance
	TotalJobCount  int     `json:"total_job_count"`
	TotalCompleted int     `json:"total_completed"`
	TotalFailed    int     `json:"total_failed"`
	SuccessRate    float64 `json:"success_rate"`

	// Print time
	TotalPrintSeconds int `json:"total_print_seconds"`
	AvgPrintSeconds   int `json:"avg_print_seconds"`

	// Material
	TotalMaterialGrams float64 `json:"total_material_grams"`
	AvgMaterialGrams   float64 `json:"avg_material_grams"`

	// Estimated vs Actual comparison (from template)
	EstimatedPrintSeconds  int     `json:"estimated_print_seconds"`
	EstimatedMaterialGrams float64 `json:"estimated_material_grams"`
	EstimatedCostCents     int     `json:"estimated_cost_cents"`
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
	Tags        []Tag      `json:"tags,omitempty"`
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
	Tags          []Tag           `json:"tags,omitempty"`
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
	CostPerHourCents  int            `json:"cost_per_hour_cents"`  // Hourly cost in cents (e.g. 150 = $1.50/hr)
	PurchasePriceCents int           `json:"purchase_price_cents"` // Purchase price in cents for ROI tracking
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

	// Target temperatures
	BedTargetTemp    float64 `json:"bed_target_temp,omitempty"`
	NozzleTargetTemp float64 `json:"nozzle_target_temp,omitempty"`
	ChamberTemp      float64 `json:"chamber_temp,omitempty"`

	// Layers
	LayerNum      int `json:"layer_num,omitempty"`
	TotalLayerNum int `json:"total_layer_num,omitempty"`

	// Fan speeds (0-100%)
	CoolingFanSpeed  int `json:"cooling_fan_speed,omitempty"`
	AuxFanSpeed      int `json:"aux_fan_speed,omitempty"`
	ChamberFanSpeed  int `json:"chamber_fan_speed,omitempty"`
	HeatbreakFanSpeed int `json:"heatbreak_fan_speed,omitempty"`

	// Speed
	SpeedPercent   int `json:"speed_percent,omitempty"`
	SpeedLevel     int `json:"speed_level,omitempty"`     // 1=silent, 2=standard, 3=sport, 4=ludicrous
	PrintRealSpeed int `json:"print_real_speed,omitempty"`

	// Network
	WiFiSignal string `json:"wifi_signal,omitempty"`

	// Nozzle info
	NozzleDiameter string `json:"nozzle_diameter,omitempty"`
	NozzleType     string `json:"nozzle_type,omitempty"`

	// Diagnostics
	HMSErrors []HMSError  `json:"hms_errors,omitempty"`
	Lights    []LightState `json:"lights,omitempty"`

	// Timing / Job
	GcodeStartTime string `json:"gcode_start_time,omitempty"`
	SubtaskID      string `json:"subtask_id,omitempty"`
	TaskID         string `json:"task_id,omitempty"`
	PrintType      string `json:"print_type,omitempty"`
}

// HMSError represents a Health Management System error from the printer.
type HMSError struct {
	Attr     int `json:"attr"`
	Code     int `json:"code"`
	Module   int `json:"module"`
	Severity int `json:"severity"`
}

// LightState represents the state of a printer light.
type LightState struct {
	Node string `json:"node"`
	Mode string `json:"mode"`
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
	MaterialTypeTPU    MaterialType = "tpu"
	MaterialTypeSupply MaterialType = "supply"
)

// TempRange represents a temperature range.
type TempRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

// Material represents a type of printing material in the catalog.
type Material struct {
	ID                uuid.UUID    `json:"id"`
	Name              string       `json:"name"`
	Type              MaterialType `json:"type"`
	Manufacturer      string       `json:"manufacturer"`
	Color             string       `json:"color"`
	ColorHex          string       `json:"color_hex"`
	Density           float64      `json:"density"`
	CostPerKg         float64      `json:"cost_per_kg"`
	PrintTemp         *TempRange   `json:"print_temp,omitempty"`
	BedTemp           *TempRange   `json:"bed_temp,omitempty"`
	Notes             string       `json:"notes"`
	LowThresholdGrams int          `json:"low_threshold_grams"` // Alert when spool falls below this
	CreatedAt         time.Time    `json:"created_at"`
	UpdatedAt         time.Time    `json:"updated_at"`
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
	MaterialUsedGrams    *float64 `json:"material_used_grams,omitempty"`
	CostCents            *int     `json:"cost_cents,omitempty"`             // Total cost (printer time + material)
	PrinterTimeCostCents *int     `json:"printer_time_cost_cents,omitempty"` // Snapshot: printer hourly rate * actual hours
	MaterialCostCents    *int     `json:"material_cost_cents,omitempty"`     // Snapshot: material cost at completion

	// Material snapshot (AMS state captured at job start)
	MaterialSnapshot *MaterialSnapshot `json:"material_snapshot,omitempty"`

	// Queue management
	Priority            int  `json:"priority"`              // Higher values = higher priority
	AutoDispatchEnabled bool `json:"auto_dispatch_enabled"` // Whether this job participates in auto-dispatch

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
	ExpenseItemActionSkipped       ExpenseItemAction = "skipped"
	ExpenseItemActionCreatedSupply ExpenseItemAction = "created_supply"
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
	RefreshToken string     `json:"refresh_token,omitempty"`
	MQTTUsername string     `json:"mqtt_username"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// IsExpired returns true if the auth token has expired or will expire within the given buffer.
func (a *BambuCloudAuth) IsExpired(buffer time.Duration) bool {
	if a.ExpiresAt == nil {
		return false // No expiration set, assume valid
	}
	return time.Now().Add(buffer).After(*a.ExpiresAt)
}

// CanRefresh returns true if a refresh token is available.
func (a *BambuCloudAuth) CanRefresh() bool {
	return a.RefreshToken != ""
}

// PrinterUtilization represents utilization metrics for a time period.
type PrinterUtilization struct {
	Period                     string  `json:"period"`
	TotalHours                 float64 `json:"total_hours"`
	PrintingHours              float64 `json:"printing_hours"`
	FailedHours                float64 `json:"failed_hours"`
	IdleHours                  float64 `json:"idle_hours"`
	UtilizationPercent         float64 `json:"utilization_percent"`
	ConfiguredCostPerHourCents int     `json:"configured_cost_per_hour_cents"`
	ActualRevenuePerHourCents  int     `json:"actual_revenue_per_hour_cents"`
}

// PrinterROI represents ROI and break-even metrics for a printer.
type PrinterROI struct {
	PurchasePriceCents  int     `json:"purchase_price_cents"`
	TotalRevenueCents   int     `json:"total_revenue_cents"`
	TotalCostCents      int     `json:"total_cost_cents"`
	LifetimeProfitCents int     `json:"lifetime_profit_cents"`
	TotalPrintingHours  float64 `json:"total_printing_hours"`
	RevenuePerHourCents int     `json:"revenue_per_hour_cents"`
	CostPerHourCents    int     `json:"cost_per_hour_cents"`
	NetPerHourCents     int     `json:"net_per_hour_cents"`
	HoursToBreakEven    float64 `json:"hours_to_break_even"`
	PrinterAgeHours     float64 `json:"printer_age_hours"`
	BreakEvenReached    bool    `json:"break_even_reached"`
}

// PrinterHealth represents health and performance metrics for a printer.
type PrinterHealth struct {
	TotalJobs          int            `json:"total_jobs"`
	CompletedJobs      int            `json:"completed_jobs"`
	FailedJobs         int            `json:"failed_jobs"`
	FailureRate        float64        `json:"failure_rate"`
	AvgJobDurationSec  int            `json:"avg_job_duration_sec"`
	AvgCostCents       int            `json:"avg_cost_cents"`
	TotalMaterialGrams float64        `json:"total_material_grams"`
	TotalCostCents     int            `json:"total_cost_cents"`
	TotalRevenueCents  int            `json:"total_revenue_cents"`
	FailureBreakdown   map[string]int `json:"failure_breakdown"`
}

// PrinterAnalytics combines utilization, ROI, and health metrics.
type PrinterAnalytics struct {
	Utilization []PrinterUtilization `json:"utilization"`
	ROI         *PrinterROI          `json:"roi"`
	Health      *PrinterHealth       `json:"health"`
}

// DispatchRequestStatus represents the status of a dispatch request.
type DispatchRequestStatus string

const (
	DispatchPending   DispatchRequestStatus = "pending"
	DispatchConfirmed DispatchRequestStatus = "confirmed"
	DispatchRejected  DispatchRequestStatus = "rejected"
	DispatchExpired   DispatchRequestStatus = "expired"
)

// DispatchRequest represents a pending dispatch request awaiting operator confirmation.
type DispatchRequest struct {
	ID          uuid.UUID             `json:"id"`
	JobID       uuid.UUID             `json:"job_id"`
	PrinterID   uuid.UUID             `json:"printer_id"`
	Status      DispatchRequestStatus `json:"status"`
	CreatedAt   time.Time             `json:"created_at"`
	ExpiresAt   time.Time             `json:"expires_at"`
	RespondedAt *time.Time            `json:"responded_at,omitempty"`
	Reason      string                `json:"reason,omitempty"`
	Job         *PrintJob             `json:"job,omitempty"`
	Printer     *Printer              `json:"printer,omitempty"`
}

// AutoDispatchSettings holds per-printer auto-dispatch configuration.
type AutoDispatchSettings struct {
	PrinterID           uuid.UUID `json:"printer_id"`
	Enabled             bool      `json:"enabled"`
	RequireConfirmation bool      `json:"require_confirmation"`
	AutoStart           bool      `json:"auto_start"`
	TimeoutMinutes      int       `json:"timeout_minutes"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// ============================================
// Alerts (Phase 1)
// ============================================

// AlertType identifies the type of alert.
type AlertType string

const (
	AlertTypeLowSpool   AlertType = "low_spool"
	AlertTypeEmptySpool AlertType = "empty_spool"
	AlertTypeOrderDue   AlertType = "order_due"
	AlertTypeJobFailed  AlertType = "job_failed"
)

// AlertSeverity indicates how urgent an alert is.
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityCritical AlertSeverity = "critical"
)

// Alert represents a system alert for user attention.
type Alert struct {
	ID             string        `json:"id"`
	Type           AlertType     `json:"type"`
	Severity       AlertSeverity `json:"severity"`
	EntityID       uuid.UUID     `json:"entity_id"`
	EntityType     string        `json:"entity_type"` // "spool", "order", "job"
	Message        string        `json:"message"`
	CreatedAt      time.Time     `json:"created_at"`
	DismissedUntil *time.Time    `json:"dismissed_until,omitempty"`
}

// AlertDismissal records when a user dismissed an alert.
type AlertDismissal struct {
	ID             uuid.UUID  `json:"id"`
	AlertType      AlertType  `json:"alert_type"`
	EntityID       string     `json:"entity_id"`
	DismissedAt    time.Time  `json:"dismissed_at"`
	DismissedUntil *time.Time `json:"dismissed_until,omitempty"`
}

// ============================================
// Unified Orders (Phase 2)
// ============================================

// OrderStatus represents the status of an order.
type OrderStatus string

const (
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusInProgress OrderStatus = "in_progress"
	OrderStatusCompleted  OrderStatus = "completed"
	OrderStatusShipped    OrderStatus = "shipped"
	OrderStatusCancelled  OrderStatus = "cancelled"
)

// OrderSource identifies where an order originated.
type OrderSource string

const (
	OrderSourceManual      OrderSource = "manual"
	OrderSourceEtsy        OrderSource = "etsy"
	OrderSourceSquarespace OrderSource = "squarespace"
	OrderSourceShopify     OrderSource = "shopify"
)

// Order represents a unified order from any source.
type Order struct {
	ID            uuid.UUID   `json:"id"`
	Source        OrderSource `json:"source"`
	SourceOrderID string      `json:"source_order_id,omitempty"`
	CustomerName  string      `json:"customer_name"`
	CustomerEmail string      `json:"customer_email,omitempty"`
	Status        OrderStatus `json:"status"`
	Priority      int         `json:"priority"`
	DueDate       *time.Time  `json:"due_date,omitempty"`
	Notes         string      `json:"notes,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
	CompletedAt   *time.Time  `json:"completed_at,omitempty"`
	ShippedAt     *time.Time  `json:"shipped_at,omitempty"`
	Items         []OrderItem `json:"items,omitempty"`
	Projects      []Project   `json:"projects,omitempty"`
	Events        []OrderEvent `json:"events,omitempty"`
}

// OrderItem represents a line item in an order.
type OrderItem struct {
	ID         uuid.UUID  `json:"id"`
	OrderID    uuid.UUID  `json:"order_id"`
	TemplateID *uuid.UUID `json:"template_id,omitempty"`
	SKU        string     `json:"sku,omitempty"`
	Quantity   int        `json:"quantity"`
	Notes      string     `json:"notes,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// OrderEvent records an event in the order lifecycle.
type OrderEvent struct {
	ID        uuid.UUID `json:"id"`
	OrderID   uuid.UUID `json:"order_id"`
	EventType string    `json:"event_type"`
	Message   string    `json:"message,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// OrderProgress represents the completion progress of an order.
type OrderProgress struct {
	OrderID         uuid.UUID `json:"order_id"`
	TotalItems      int       `json:"total_items"`
	CompletedItems  int       `json:"completed_items"`
	TotalJobs       int       `json:"total_jobs"`
	CompletedJobs   int       `json:"completed_jobs"`
	ProgressPercent float64   `json:"progress_percent"`
}

// OrderFilters defines filter options for listing orders.
type OrderFilters struct {
	Status    *OrderStatus `json:"status,omitempty"`
	Source    *OrderSource `json:"source,omitempty"`
	StartDate *time.Time   `json:"start_date,omitempty"`
	EndDate   *time.Time   `json:"end_date,omitempty"`
	Limit     int          `json:"limit,omitempty"`
	Offset    int          `json:"offset,omitempty"`
}

// ============================================
// Shopify Integration (Phase 3)
// ============================================

// ShopifyCredentials stores OAuth credentials for a Shopify store.
type ShopifyCredentials struct {
	ID          uuid.UUID `json:"id"`
	ShopDomain  string    `json:"shop_domain"`
	AccessToken string    `json:"access_token"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ShopifyOrder represents a synced Shopify order.
type ShopifyOrder struct {
	ID              uuid.UUID          `json:"id"`
	ShopifyOrderID  string             `json:"shopify_order_id"`
	OrderID         *uuid.UUID         `json:"order_id,omitempty"` // Link to unified Order
	ShopDomain      string             `json:"shop_domain"`
	OrderNumber     string             `json:"order_number"`
	CustomerName    string             `json:"customer_name"`
	CustomerEmail   string             `json:"customer_email"`
	TotalCents      int                `json:"total_cents"`
	Status          string             `json:"status"`
	SyncedAt        time.Time          `json:"synced_at"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
	Items           []ShopifyOrderItem `json:"items,omitempty"`
}

// ShopifyOrderItem represents a line item in a Shopify order.
type ShopifyOrderItem struct {
	ID                 uuid.UUID `json:"id"`
	ShopifyOrderID     uuid.UUID `json:"shopify_order_id"`
	ShopifyLineItemID  string    `json:"shopify_line_item_id"`
	SKU                string    `json:"sku"`
	Title              string    `json:"title"`
	Quantity           int       `json:"quantity"`
	PriceCents         int       `json:"price_cents"`
	CreatedAt          time.Time `json:"created_at"`
}

// ShopifyProductTemplate links a Shopify product to a template.
type ShopifyProductTemplate struct {
	ID               uuid.UUID `json:"id"`
	ShopifyProductID string    `json:"shopify_product_id"`
	TemplateID       uuid.UUID `json:"template_id"`
	SKU              string    `json:"sku,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

// ShopifyIntegrationStatus represents the connection status.
type ShopifyIntegrationStatus struct {
	Connected   bool       `json:"connected"`
	ShopDomain  string     `json:"shop_domain,omitempty"`
	LastSyncAt  *time.Time `json:"last_sync_at,omitempty"`
	OrderCount  int        `json:"order_count,omitempty"`
}

// ============================================
// Timeline / Gantt View (Phase 4)
// ============================================

// TimelineItem represents an item on the timeline/Gantt view.
type TimelineItem struct {
	ID        uuid.UUID      `json:"id"`
	Type      string         `json:"type"` // "order", "project", "job"
	Name      string         `json:"name"`
	Status    string         `json:"status"`
	StartDate *time.Time     `json:"start_date,omitempty"`
	DueDate   *time.Time     `json:"due_date,omitempty"`
	EndDate   *time.Time     `json:"end_date,omitempty"` // Actual or estimated
	Progress  float64        `json:"progress"`           // 0-100
	ParentID  *uuid.UUID     `json:"parent_id,omitempty"`
	Children  []TimelineItem `json:"children,omitempty"`
}

// ============================================
// Tags (Phase 5)
// ============================================

// Tag represents a user-defined tag for organization.
type Tag struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"created_at"`
}

