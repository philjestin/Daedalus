// Project types (Product Catalog)
export interface Project {
  id: string
  name: string
  description: string
  target_date?: string
  tags: string[]
  template_id?: string  // Legacy: kept for migration
  source: string
  external_order_id?: string
  customer_notes?: string
  // Template-like fields for product catalog
  sku?: string
  price_cents?: number
  printer_type?: string
  allowed_printer_ids?: string[]
  default_settings?: Record<string, unknown>
  notes?: string
  created_at: string
  updated_at: string
  // Aggregated stats from tasks (computed)
  total_tasks?: number
  completed_tasks?: number
}

// Task status types
export type TaskStatus = 'pending' | 'in_progress' | 'completed' | 'cancelled'

// Task types (Work Instances)
export interface Task {
  id: string
  project_id: string
  order_id?: string
  order_item_id?: string
  name: string
  status: TaskStatus
  quantity: number
  notes?: string
  pickup_date?: string
  created_at: string
  updated_at: string
  started_at?: string
  completed_at?: string
  // Loaded relations
  project?: Project
  jobs?: PrintJob[]
  checklist_items?: TaskChecklistItem[]
  progress?: number
}

// Task checklist item
export interface TaskChecklistItem {
  id: string
  task_id: string
  name: string
  part_id?: string
  sort_order: number
  completed: boolean
  completed_at?: string
  created_at: string
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

// Derived analytics for a project — computed from jobs and sales
export interface ProjectSummary {
  total_revenue_cents: number
  total_fees_cents: number
  net_revenue_cents: number
  sales_count: number
  unit_cost_cents: number
  total_cost_cents: number
  printer_time_cost_cents: number
  material_cost_cents: number
  gross_profit_cents: number
  gross_margin_percent: number
  total_print_seconds: number
  avg_print_seconds: number
  profit_per_hour_cents: number
  job_count: number
  completed_count: number
  failed_count: number
  success_rate: number
  total_material_grams: number
  estimated_material_cost_cents: number
  estimated_material_grams: number
  estimated_print_seconds: number
  supply_cost_cents: number
}

// Project supply (non-printed BOM item)
export interface ProjectSupply {
  id: string
  project_id: string
  name: string
  unit_cost_cents: number
  quantity: number
  notes: string
  material_id?: string
  created_at: string
  updated_at: string
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

export interface FilamentUsage {
  type: string
  color: string
  used_grams: number
  used_meters: number
}

export interface SliceProfile {
  print_time_seconds: number
  weight_grams: number
  printer_model?: string
  nozzle_diameter?: number
  filaments: FilamentUsage[]
}

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
  slice_profile?: SliceProfile
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
  cost_per_hour_cents: number // Hourly cost in cents (e.g. 150 = $1.50/hr)
  purchase_price_cents: number // Purchase price in cents for ROI tracking
  created_at: string
  updated_at: string
}

export interface HMSError {
  attr: number
  code: number
  module: number
  severity: number
}

export interface LightState {
  node: string
  mode: string
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

  // Target temperatures
  bed_target_temp?: number
  nozzle_target_temp?: number
  chamber_temp?: number

  // Layers
  layer_num?: number
  total_layer_num?: number

  // Fan speeds (0-100%)
  cooling_fan_speed?: number
  aux_fan_speed?: number
  chamber_fan_speed?: number
  heatbreak_fan_speed?: number

  // Speed
  speed_percent?: number
  speed_level?: number       // 1=silent, 2=standard, 3=sport, 4=ludicrous
  print_real_speed?: number

  // Network
  wifi_signal?: string

  // Nozzle info
  nozzle_diameter?: string
  nozzle_type?: string

  // Diagnostics
  hms_errors?: HMSError[]
  lights?: LightState[]

  // Timing / Job
  gcode_start_time?: string
  subtask_id?: string
  task_id?: string
  print_type?: string
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
export type MaterialType = 'pla' | 'petg' | 'abs' | 'asa' | 'tpu' | 'supply'

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
  cost_cents?: number                  // Total cost (printer time + material)
  printer_time_cost_cents?: number     // Snapshot: printer hourly rate * actual hours
  material_cost_cents?: number         // Snapshot: material cost at completion
  // Material snapshot captured at job start
  material_snapshot?: MaterialSnapshot
  // Event history (when fetched with events)
  events?: JobEvent[]
  // Queue management
  priority?: number
  auto_dispatch_enabled?: boolean
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
  supply_cost_cents: number
  total_cost_cents: number
  estimated_print_time_seconds: number
  labor_minutes: number
  material_breakdown: RecipeMaterialCostBreakdown[]
  supply_breakdown?: RecipeSupplyCostBreakdown[]
  hourly_rate_cents: number
  labor_rate_cents: number
  printer_name?: string
  // Margin calculation
  sale_price_cents: number
  gross_margin_cents: number
  gross_margin_percent: number
  profit_per_hour_cents: number
}

// Supply cost breakdown
export interface RecipeSupplyCostBreakdown {
  name: string
  unit_cost_cents: number
  quantity: number
  total_cents: number
}

// Recipe supply item (non-printed BOM)
export interface RecipeSupply {
  id: string
  recipe_id: string
  name: string
  unit_cost_cents: number
  quantity: number
  material_id?: string
  notes?: string
  created_at: string
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
  supplies?: RecipeSupply[]
}

// Aggregated performance analytics from projects created from a template
export interface TemplateAnalytics {
  template_id: string
  project_count: number
  // Revenue
  total_revenue_cents: number
  total_fees_cents: number
  net_revenue_cents: number
  total_sales_count: number
  // Costs
  total_cost_cents: number
  avg_unit_cost_cents: number
  total_printer_time_cost: number
  total_material_cost: number
  total_supply_cost: number
  // Profit
  total_gross_profit_cents: number
  avg_gross_margin_percent: number
  profit_per_hour_cents: number
  // Performance
  total_job_count: number
  total_completed: number
  total_failed: number
  success_rate: number
  // Print time
  total_print_seconds: number
  avg_print_seconds: number
  // Material
  total_material_grams: number
  avg_material_grams: number
  // Estimated vs Actual comparison
  estimated_print_seconds: number
  estimated_material_grams: number
  estimated_cost_cents: number
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
export type ExpenseCategory = 'filament' | 'parts' | 'tools' | 'shipping' | 'marketplace_fees' | 'subscription' | 'advertising' | 'other'

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

// Analytics types
export interface TimeSeriesPoint {
  date: string
  revenue: number
  expenses: number
  profit: number
}

export interface TimeSeriesData {
  points: TimeSeriesPoint[]
  period: string
}

export interface CategoryBreakdown {
  category: string
  total: number
  count: number
}

export interface ChannelBreakdown {
  channel: string
  total: number
  count: number
}

export interface WeekSummary {
  gross_cents: number
  net_cents: number
  fees_cents: number
  count: number
}

export interface WeeklyInsights {
  this_week: WeekSummary
  last_week: WeekSummary
  channels: ChannelBreakdown[]
  week_start: string
  week_end: string
  pending_count: number
  pending_revenue_cents: number
}

export interface ProjectSales {
  project_id: string
  project_name: string
  gross_cents: number
  net_cents: number
  count: number
  avg_cents: number
  unit_cost_cents: number
  total_cogs_cents: number
  profit_cents: number
  estimated_print_seconds: number
  total_print_seconds: number
  first_sale: string
  last_sale: string
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

// Printer Analytics types
export interface PrinterUtilization {
  period: string
  total_hours: number
  printing_hours: number
  failed_hours: number
  idle_hours: number
  utilization_percent: number
  configured_cost_per_hour_cents: number
  actual_revenue_per_hour_cents: number
}

export interface PrinterROI {
  purchase_price_cents: number
  total_revenue_cents: number
  total_cost_cents: number
  lifetime_profit_cents: number
  total_printing_hours: number
  revenue_per_hour_cents: number
  cost_per_hour_cents: number
  net_per_hour_cents: number
  hours_to_break_even: number
  printer_age_hours: number
  break_even_reached: boolean
}

export interface PrinterHealth {
  total_jobs: number
  completed_jobs: number
  failed_jobs: number
  failure_rate: number
  avg_job_duration_sec: number
  avg_cost_cents: number
  total_material_grams: number
  total_cost_cents: number
  total_revenue_cents: number
  failure_breakdown: Record<string, number>
}

export interface PrinterAnalytics {
  utilization: PrinterUtilization[]
  roi: PrinterROI
  health: PrinterHealth
}

// Backup types
export interface BackupInfo {
  name: string
  path: string
  size: number
  created_at: string
}

export interface BackupConfig {
  auto_on_startup: boolean
  schedule_enabled: boolean
  schedule_interval: 'daily' | 'weekly'
  retention_count: number
}

// Squarespace Integration types
export interface SquarespaceIntegration {
  connected: boolean
  id?: string
  site_id?: string
  site_title?: string
  is_active?: boolean
  last_order_sync_at?: string
  last_product_sync_at?: string
  created_at?: string
  updated_at?: string
}

export interface SquarespaceAddress {
  first_name: string
  last_name: string
  address1: string
  address2: string
  city: string
  state: string
  postal_code: string
  country_code: string
  phone: string
}

export interface SquarespaceOrder {
  id: string
  squarespace_order_id: string
  order_number: string
  customer_email: string
  customer_name: string
  channel: string
  subtotal_cents: number
  shipping_cents: number
  tax_cents: number
  discount_cents: number
  refunded_cents: number
  grand_total_cents: number
  currency: string
  fulfillment_status: string
  billing_address?: SquarespaceAddress
  shipping_address?: SquarespaceAddress
  created_on?: string
  modified_on?: string
  is_processed: boolean
  project_id?: string
  items?: SquarespaceOrderItem[]
  synced_at: string
  created_at: string
  updated_at: string
}

export interface SquarespaceOrderItem {
  id: string
  order_id: string
  squarespace_item_id: string
  product_id: string
  variant_id: string
  product_name: string
  sku: string
  quantity: number
  unit_price_cents: number
  currency: string
  image_url: string
  variant_options?: string[]
  template_id?: string
  created_at: string
}

export interface SquarespaceProduct {
  id: string
  squarespace_product_id: string
  name: string
  description: string
  url: string
  type: string
  is_visible: boolean
  tags?: string[]
  variants?: SquarespaceProductVariant[]
  created_on?: string
  modified_on?: string
  synced_at: string
  created_at: string
  updated_at: string
}

export interface SquarespaceProductVariant {
  id: string
  product_id: string
  squarespace_variant_id: string
  sku: string
  price_cents: number
  sale_price_cents: number
  on_sale: boolean
  stock_quantity: number
  stock_unlimited: boolean
  attributes?: Record<string, string>
  created_at: string
}

// Auto-dispatch types
export type DispatchRequestStatus = 'pending' | 'confirmed' | 'rejected' | 'expired'

export interface DispatchRequest {
  id: string
  job_id: string
  printer_id: string
  status: DispatchRequestStatus
  created_at: string
  expires_at: string
  responded_at?: string
  reason?: string
  job?: PrintJob
  printer?: Printer
}

export interface AutoDispatchSettings {
  printer_id: string
  enabled: boolean
  require_confirmation: boolean
  auto_start: boolean
  timeout_minutes: number
  updated_at?: string
}

// ============================================
// Alerts (Phase 1)
// ============================================

export type AlertType = 'low_spool' | 'empty_spool' | 'order_due' | 'job_failed'
export type AlertSeverity = 'info' | 'warning' | 'critical'

export interface Alert {
  id: string
  type: AlertType
  severity: AlertSeverity
  entity_id: string
  entity_type: string
  message: string
  created_at: string
  dismissed_until?: string
}

export interface AlertCounts {
  info: number
  warning: number
  critical: number
}

// ============================================
// Unified Orders (Phase 2)
// ============================================

export type OrderStatus = 'pending' | 'in_progress' | 'completed' | 'shipped' | 'cancelled'
export type OrderSource = 'manual' | 'etsy' | 'squarespace' | 'shopify'

export interface Order {
  id: string
  source: OrderSource
  source_order_id?: string
  customer_name: string
  customer_email?: string
  status: OrderStatus
  priority: number
  due_date?: string
  notes?: string
  created_at: string
  updated_at: string
  completed_at?: string
  shipped_at?: string
  items?: OrderItem[]
  tasks?: Task[]
  events?: OrderEvent[]
}

export interface OrderItem {
  id: string
  order_id: string
  project_id?: string   // Link to project (product catalog)
  template_id?: string  // Legacy: kept for migration
  sku?: string
  quantity: number
  notes?: string
  created_at: string
}

export interface OrderEvent {
  id: string
  order_id: string
  event_type: string
  message?: string
  created_at: string
}

export interface OrderProgress {
  order_id: string
  total_items: number
  completed_items: number
  total_jobs: number
  completed_jobs: number
  progress_percent: number
}

export interface OrderCounts {
  pending: number
  in_progress: number
  completed: number
  shipped: number
  cancelled: number
}

// ============================================
// Shopify Integration (Phase 3)
// ============================================

export interface ShopifyIntegrationStatus {
  connected: boolean
  shop_domain?: string
  last_sync_at?: string
  order_count?: number
}

export interface ShopifyOrder {
  id: string
  shopify_order_id: string
  order_id?: string
  shop_domain: string
  order_number: string
  customer_name: string
  customer_email: string
  total_cents: number
  status: string
  synced_at: string
  created_at: string
  updated_at: string
  items?: ShopifyOrderItem[]
}

export interface ShopifyOrderItem {
  id: string
  shopify_order_id: string
  shopify_line_item_id: string
  sku: string
  title: string
  quantity: number
  price_cents: number
  created_at: string
}

// ============================================
// Timeline / Gantt View (Phase 4)
// ============================================

export interface TimelineItem {
  id: string
  type: 'order' | 'project' | 'job'
  name: string
  status: string
  start_date?: string
  due_date?: string
  end_date?: string
  progress: number
  parent_id?: string
  children?: TimelineItem[]
}

// ============================================
// Tags (Phase 5)
// ============================================

export interface Tag {
  id: string
  name: string
  color: string
  created_at: string
}

// ============================================
// Beta Feedback
// ============================================

export interface Feedback {
  id: string
  type: 'bug' | 'feature' | 'general'
  message: string
  contact?: string
  page?: string
  app_version?: string
  created_at: string
}

