// Project types
export type ProjectStatus = 'draft' | 'active' | 'completed' | 'ready_to_ship' | 'shipped' | 'archived'

export interface Project {
  id: string
  name: string
  description: string
  status: ProjectStatus
  target_date?: string
  tags: string[]
  template_id?: string
  source: string
  external_order_id?: string
  customer_notes?: string
  shipped_at?: string
  tracking_number?: string
  created_at: string
  updated_at: string
}

// Job statistics for a project
export interface JobStats {
  total: number
  queued: number
  assigned: number
  printing: number
  completed: number
  failed: number
  cancelled: number
}

// Result of starting production
export interface StartProductionResult {
  jobs_started: number
  jobs_skipped: number
  failed_jobs?: StartJobFailure[]
}

export interface StartJobFailure {
  job_id: string
  reason: string
}

// Request for marking a project as shipped
export interface ShipRequest {
  tracking_number?: string
}

// Part types
export type PartStatus = 'design' | 'printing' | 'complete'

export interface Part {
  id: string
  project_id: string
  name: string
  description: string
  quantity: number
  status: PartStatus
  created_at: string
  updated_at: string
}

// Design types
export type FileType = 'stl' | '3mf' | 'gcode'

export interface Design {
  id: string
  part_id: string
  version: number
  file_id: string
  file_name: string
  file_hash: string
  file_size_bytes: number
  file_type: FileType
  notes: string
  slice_profile?: Record<string, unknown>
  created_at: string
}

// Printer types
export type ConnectionType = 'manual' | 'octoprint' | 'bambu_lan' | 'bambu_cloud' | 'moonraker'
export type PrinterStatus = 'idle' | 'printing' | 'paused' | 'error' | 'offline'

export interface BuildVolume {
  x: number
  y: number
  z: number
}

export interface Printer {
  id: string
  name: string
  model: string
  manufacturer: string
  connection_type: ConnectionType
  connection_uri: string
  api_key?: string
  serial_number?: string
  status: PrinterStatus
  build_volume?: BuildVolume
  nozzle_diameter: number
  location: string
  notes: string
  created_at: string
  updated_at: string
}

export interface PrinterState {
  printer_id: string
  status: PrinterStatus
  progress: number
  current_file?: string
  time_left?: number
  bed_temp?: number
  nozzle_temp?: number
  ams?: AMSState
  updated_at: string
}

// AMS (Automatic Material System) types
export interface AMSState {
  units: AMSUnit[]
  current_tray?: string
  external_spool?: AMSTray
}

export interface AMSUnit {
  id: number
  humidity: number
  temp: number
  trays: AMSTray[]
}

export interface AMSTray {
  id: number
  material_type: string
  color: string
  color_hex?: string
  remain: number
  tag_uid?: string
  brand?: string
  nozzle_temp_min?: number
  nozzle_temp_max?: number
  bed_temp?: number
  empty: boolean
}

// Material snapshot captured at job start
export interface MaterialSnapshot {
  captured_at: string
  selected_tray: number
  material_type: string
  color: string
  remain_percent: number
  brand?: string
  ams_state?: AMSState
}

// Material validation result
export interface MaterialValidation {
  valid: boolean
  warnings?: string[]
  errors?: string[]
}

// Preflight check result for job start validation
export interface PreflightCheckResult {
  ready: boolean
  validation?: MaterialValidation
  ams_state?: AMSState
  warnings?: string[]
  errors?: string[]
}

// Material types
export type MaterialType = 'pla' | 'petg' | 'abs' | 'asa' | 'tpu'

export interface TempRange {
  min: number
  max: number
}

export interface Material {
  id: string
  name: string
  type: MaterialType
  manufacturer: string
  color: string
  color_hex: string
  density: number
  cost_per_kg: number
  print_temp?: TempRange
  bed_temp?: TempRange
  notes: string
  created_at: string
  updated_at: string
}

// Spool types
export type SpoolStatus = 'new' | 'in_use' | 'low' | 'empty' | 'archived'

export interface MaterialSpool {
  id: string
  material_id: string
  initial_weight: number
  remaining_weight: number
  purchase_date?: string
  purchase_cost: number
  location: string
  status: SpoolStatus
  notes: string
  created_at: string
  updated_at: string
}

// Print job types
export type PrintJobStatus = 'queued' | 'assigned' | 'uploaded' | 'sending' | 'printing' | 'paused' | 'completed' | 'failed' | 'cancelled'

// Job event types for immutable history
export type JobEventType =
  | 'queued'
  | 'assigned'
  | 'uploaded'
  | 'started'
  | 'progress'
  | 'paused'
  | 'resumed'
  | 'completed'
  | 'failed'
  | 'cancelled'
  | 'retried'

// Actor types for event tracking
export type ActorType = 'user' | 'system' | 'printer' | 'webhook'

// Failure categories for analytics
export type FailureCategory =
  | 'mechanical'
  | 'filament'
  | 'adhesion'
  | 'thermal'
  | 'network'
  | 'user_cancelled'
  | 'unknown'

// Job event (immutable history record)
export interface JobEvent {
  id: string
  job_id: string
  event_type: JobEventType
  occurred_at: string
  status?: PrintJobStatus
  progress?: number
  printer_id?: string
  error_code?: string
  error_message?: string
  actor_type: ActorType
  actor_id?: string
  metadata?: Record<string, unknown>
  created_at: string
}

export interface PrintOutcome {
  success: boolean
  quality_rating?: number
  actual_weight?: number
  actual_time?: number
  failure_reason?: string
  photos?: string[]
  notes?: string
  material_used: number
  material_cost: number
}

export interface PrintJob {
  id: string
  design_id: string
  printer_id?: string
  material_spool_id?: string
  project_id?: string
  status: PrintJobStatus
  progress: number
  started_at?: string
  completed_at?: string
  outcome?: PrintOutcome
  notes: string
  created_at: string
  // Recipe/Retry tracking
  recipe_id?: string
  attempt_number: number
  parent_job_id?: string
  failure_category?: FailureCategory
  // Cost tracking
  estimated_seconds?: number
  actual_seconds?: number
  material_used_grams?: number
  cost_cents?: number
  // Material snapshot captured at job start
  material_snapshot?: MaterialSnapshot
  // Event history (when fetched with events)
  events?: JobEvent[]
}

// Request type for retrying a job
export interface RetryJobRequest {
  printer_id?: string
  material_spool_id?: string
  failure_category?: FailureCategory
  notes?: string
}

// Request type for recording a failure
export interface RecordFailureRequest {
  failure_category: FailureCategory
  error_code?: string
  error_message?: string
}

// Request type for marking a job as scrap
export interface ScrapRequest {
  failure_category?: FailureCategory
  notes?: string
}

// Print Profile type
export type PrintProfile = 'standard' | 'detailed' | 'fast' | 'strong' | 'custom'

// Printer constraints for recipes
export interface PrinterConstraints {
  min_bed_size?: BuildVolume
  nozzle_diameters?: number[]
  requires_enclosure: boolean
  requires_ams: boolean
  printer_tags?: string[]
}

// Color specification for material matching
export interface ColorSpec {
  mode: 'exact' | 'category' | 'any'
  hex?: string
  name?: string
}

// Recipe material requirement
export interface RecipeMaterial {
  id: string
  recipe_id: string
  material_type: MaterialType
  color_spec?: ColorSpec
  weight_grams: number
  ams_position?: number
  sequence_order: number
  notes?: string
  created_at: string
}

// Cost estimate breakdown
export interface RecipeCostEstimate {
  material_cost_cents: number
  time_cost_cents: number
  labor_cost_cents: number
  total_cost_cents: number
  estimated_print_time_seconds: number
  labor_minutes: number
  material_breakdown: RecipeMaterialCostBreakdown[]
  hourly_rate_cents: number
  labor_rate_cents: number
  // Margin calculation
  sale_price_cents: number
  gross_margin_cents: number
  gross_margin_percent: number
}

export interface RecipeMaterialCostBreakdown {
  material_type: string
  weight_grams: number
  cost_cents: number
  color_name?: string
}

// Compatible spool result
export interface CompatibleSpool {
  spool: MaterialSpool
  material: Material
  match_reason: string
}

// Printer validation result
export interface PrinterValidationResult {
  valid: boolean
  errors?: string[]
  warnings?: string[]
}

// Template types (enhanced as Recipe)
export interface Template {
  id: string
  name: string
  description: string
  sku: string
  tags: string[]
  material_type: MaterialType
  estimated_material_grams: number
  preferred_printer_id?: string
  allow_any_printer: boolean
  quantity_per_order: number
  post_process_checklist: string[]
  is_active: boolean
  printer_constraints?: PrinterConstraints
  print_profile: PrintProfile
  estimated_print_seconds: number
  // Pricing fields for margin calculation
  labor_minutes: number
  sale_price_cents: number
  material_cost_per_gram_cents: number
  version: number
  archived_at?: string
  created_at: string
  updated_at: string
  designs?: TemplateDesign[]
  materials?: RecipeMaterial[]
}

export interface TemplateDesign {
  id: string
  template_id: string
  design_id: string
  is_primary: boolean
  quantity: number
  sequence_order: number
  notes: string
  created_at: string
  design?: Design
}

// API response types
export interface ApiError {
  error: string
}

// Expense types
export type ExpenseStatus = 'pending' | 'confirmed' | 'rejected'
export type ExpenseCategory = 'filament' | 'parts' | 'tools' | 'shipping' | 'marketplace_fees' | 'subscription' | 'other'

export interface FilamentMetadata {
  brand?: string
  material_type?: string
  color?: string
  color_hex?: string
  weight_grams?: number
  diameter_mm?: number
}

export interface ExpenseItem {
  id: string
  expense_id: string
  description: string
  quantity: number
  unit_price_cents: number
  total_price_cents: number
  sku?: string
  vendor_item_id?: string
  category: ExpenseCategory
  metadata?: FilamentMetadata
  matched_spool_id?: string
  matched_material_id?: string
  confidence: number
  action_taken: 'none' | 'created_spool' | 'matched_spool' | 'skipped'
  created_at: string
}

export interface Expense {
  id: string
  occurred_at: string
  vendor: string
  subtotal_cents: number
  tax_cents: number
  shipping_cents: number
  total_cents: number
  currency: string
  category: ExpenseCategory
  notes: string
  receipt_file_id?: string
  receipt_file_path?: string
  status: ExpenseStatus
  confidence: number
  created_at: string
  updated_at: string
  items?: ExpenseItem[]
}

// Sale types
export type SalesChannel = 'marketplace' | 'etsy' | 'website' | 'direct' | 'other'

export interface Sale {
  id: string
  occurred_at: string
  channel: SalesChannel
  platform: string
  gross_cents: number
  fees_cents: number
  shipping_charged_cents: number
  shipping_cost_cents: number
  tax_collected_cents: number
  net_cents: number
  currency: string
  project_id?: string
  order_reference?: string
  customer_name?: string
  item_description: string
  quantity: number
  notes: string
  created_at: string
  updated_at: string
}

// Etsy Integration types
export interface EtsyIntegration {
  connected: boolean
  configured: boolean
  shop_id?: number
  shop_name?: string
  token_expires_at?: string
  scopes?: string[]
  is_active?: boolean
  last_sync_at?: string
  created_at?: string
  updated_at?: string
}

// Etsy Receipt (Order) types
export interface EtsyReceipt {
  id: string
  etsy_receipt_id: number
  etsy_shop_id: number
  buyer_user_id?: number
  buyer_email?: string
  name: string
  status: string
  message_from_buyer?: string
  is_shipped: boolean
  is_paid: boolean
  is_gift: boolean
  gift_message?: string
  grandtotal_cents: number
  subtotal_cents: number
  total_price_cents: number
  total_shipping_cost_cents: number
  total_tax_cost_cents: number
  discount_cents: number
  currency: string
  shipping_name?: string
  shipping_address_first_line?: string
  shipping_address_second_line?: string
  shipping_city?: string
  shipping_state?: string
  shipping_zip?: string
  shipping_country_code?: string
  create_timestamp?: string
  update_timestamp?: string
  is_processed: boolean
  project_id?: string
  synced_at: string
  created_at: string
  updated_at: string
  items?: EtsyReceiptItem[]
}

export interface EtsyReceiptItem {
  id: string
  etsy_receipt_item_id: number
  receipt_id: string
  etsy_listing_id: number
  etsy_transaction_id: number
  title: string
  description?: string
  quantity: number
  price_cents: number
  shipping_cost_cents: number
  sku?: string
  variations?: unknown[]
  is_digital: boolean
  template_id?: string
  created_at: string
}

// Etsy Listing types
export interface EtsyListing {
  id: string
  etsy_listing_id: number
  etsy_shop_id: number
  title: string
  description?: string
  state: string
  quantity: number
  url?: string
  views: number
  num_favorers: number
  is_customizable: boolean
  is_personalizable: boolean
  tags?: string[]
  has_variations: boolean
  price_cents?: number
  currency: string
  skus?: string[]
  synced_at: string
  created_at: string
  updated_at: string
  linked_template?: Template
}

// Etsy Webhook Event types
export interface EtsyWebhookEvent {
  id: string
  event_type: string
  resource_type: string
  resource_id?: number
  shop_id?: number
  payload: unknown
  signature?: string
  processed: boolean
  processed_at?: string
  error?: string
  received_at: string
  created_at: string
}

// Sync Result type
export interface SyncResult {
  total_fetched: number
  created: number
  updated: number
  skipped: number
  errors: number
}

// Bambu Cloud types
export interface CloudDevice {
  dev_id: string
  name: string
  online: boolean
  print_status: string
  dev_model_name: string
  dev_product_name: string
  dev_access_code: string
  nozzle_diameter: number
}

export interface BambuCloudStatus {
  connected: boolean
  email?: string
}

