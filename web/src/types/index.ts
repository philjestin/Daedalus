// Project types
export type ProjectStatus = 'draft' | 'active' | 'completed' | 'archived'

export interface Project {
  id: string
  name: string
  description: string
  status: ProjectStatus
  target_date?: string
  tags: string[]
  created_at: string
  updated_at: string
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
export type ConnectionType = 'manual' | 'octoprint' | 'bambu_lan' | 'moonraker'
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
  updated_at: string
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
export type PrintJobStatus = 'queued' | 'sending' | 'printing' | 'completed' | 'failed' | 'cancelled'

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
  printer_id: string
  material_spool_id: string
  status: PrintJobStatus
  progress: number
  started_at?: string
  completed_at?: string
  outcome?: PrintOutcome
  notes: string
  created_at: string
}

// API response types
export interface ApiError {
  error: string
}

