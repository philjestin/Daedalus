const API_URL = import.meta.env.VITE_API_URL ?? (import.meta.env.DEV ? 'http://localhost:8080' : '')

// Generic fetch wrapper with error handling.
async function fetchApi<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const url = `${API_URL}/api${path}`

  console.log(`[API] ${options.method || 'GET'} ${url}`)

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }

  // Merge any existing headers
  if (options.headers) {
    const existingHeaders = options.headers as Record<string, string>
    Object.assign(headers, existingHeaders)
  }

  const response = await fetch(url, {
    ...options,
    headers,
  })

  console.log(`[API] Response status: ${response.status}`)

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Unknown error' }))
    throw new Error(error.error || `HTTP ${response.status}`)
  }

  if (response.status === 204) {
    return undefined as T
  }

  const text = await response.text()
  console.log(`[API] Response body:`, text.substring(0, 500))
  
  try {
    return JSON.parse(text) as T
  } catch (e) {
    console.error('[API] JSON parse error:', e, 'Body was:', text)
    throw e
  }
}

// Projects API
export const projectsApi = {
  list: () =>
    fetchApi<import('../types').Project[]>('/projects'),

  get: (id: string) =>
    fetchApi<import('../types').Project>(`/projects/${id}`),

  create: (data: Partial<import('../types').Project>) =>
    fetchApi<import('../types').Project>('/projects', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  update: (id: string, data: Partial<import('../types').Project>) =>
    fetchApi<import('../types').Project>(`/projects/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    }),

  delete: (id: string) =>
    fetchApi<void>(`/projects/${id}`, { method: 'DELETE' }),

  // Project pipeline methods
  listJobs: (id: string) =>
    fetchApi<import('../types').PrintJob[]>(`/projects/${id}/jobs`),

  getJobStats: (id: string) =>
    fetchApi<import('../types').JobStats>(`/projects/${id}/job-stats`),

  getSummary: (id: string) =>
    fetchApi<import('../types').ProjectSummary>(`/projects/${id}/summary`),

  startProduction: (id: string) =>
    fetchApi<import('../types').StartProductionResult>(`/projects/${id}/start-production`, {
      method: 'POST',
    }),

  markReadyToShip: (id: string) =>
    fetchApi<import('../types').Project>(`/projects/${id}/ready-to-ship`, {
      method: 'POST',
    }),

  ship: (id: string, trackingNumber?: string) =>
    fetchApi<import('../types').Project>(`/projects/${id}/ship`, {
      method: 'POST',
      body: JSON.stringify({ tracking_number: trackingNumber }),
    }),

  // Tasks for this project
  listTasks: (id: string) =>
    fetchApi<import('../types').Task[]>(`/projects/${id}/tasks`),
}

// Tasks API (Work Instances)
export const tasksApi = {
  list: (filters?: { project_id?: string; order_id?: string; status?: string }) => {
    const params = new URLSearchParams()
    if (filters?.project_id) params.set('project_id', filters.project_id)
    if (filters?.order_id) params.set('order_id', filters.order_id)
    if (filters?.status) params.set('status', filters.status)
    const query = params.toString()
    return fetchApi<import('../types').Task[]>(`/tasks${query ? `?${query}` : ''}`)
  },

  get: (id: string) =>
    fetchApi<import('../types').Task>(`/tasks/${id}`),

  create: (data: {
    project_id: string
    order_id?: string
    order_item_id?: string
    name: string
    quantity?: number
    notes?: string
    pickup_date?: string
  }) =>
    fetchApi<import('../types').Task>('/tasks', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  update: (id: string, data: Partial<{ name: string; quantity: number; notes: string; pickup_date: string | null }>) =>
    fetchApi<import('../types').Task>(`/tasks/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    }),

  delete: (id: string) =>
    fetchApi<void>(`/tasks/${id}`, { method: 'DELETE' }),

  updateStatus: (id: string, status: import('../types').TaskStatus) =>
    fetchApi<import('../types').Task>(`/tasks/${id}/status`, {
      method: 'PATCH',
      body: JSON.stringify({ status }),
    }),

  getProgress: (id: string) =>
    fetchApi<{ progress: number }>(`/tasks/${id}/progress`),

  start: (id: string) =>
    fetchApi<import('../types').Task>(`/tasks/${id}/start`, { method: 'POST' }),

  complete: (id: string) =>
    fetchApi<import('../types').Task>(`/tasks/${id}/complete`, { method: 'POST' }),

  cancel: (id: string) =>
    fetchApi<import('../types').Task>(`/tasks/${id}/cancel`, { method: 'POST' }),

  getChecklist: (id: string) =>
    fetchApi<import('../types').TaskChecklistItem[]>(`/tasks/${id}/checklist`),

  regenerateChecklist: (id: string) =>
    fetchApi<import('../types').TaskChecklistItem[]>(`/tasks/${id}/checklist/regenerate`, {
      method: 'POST',
    }),

  toggleChecklistItem: (taskId: string, itemId: string, completed: boolean) =>
    fetchApi<{ ok: boolean }>(`/tasks/${taskId}/checklist/${itemId}`, {
      method: 'PATCH',
      body: JSON.stringify({ completed }),
    }),

  printFromChecklist: (taskId: string, itemId: string) =>
    fetchApi<import('../types').PrintJob>(`/tasks/${taskId}/checklist/${itemId}/print`, {
      method: 'POST',
    }),
}

// Parts API
export const partsApi = {
  listByProject: (projectId: string) => 
    fetchApi<import('../types').Part[]>(`/projects/${projectId}/parts`),
  
  get: (id: string) => 
    fetchApi<import('../types').Part>(`/parts/${id}`),
  
  create: (projectId: string, data: Partial<import('../types').Part>) => 
    fetchApi<import('../types').Part>(`/projects/${projectId}/parts`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  
  update: (id: string, data: Partial<import('../types').Part>) => 
    fetchApi<import('../types').Part>(`/parts/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    }),
  
  delete: (id: string) =>
    fetchApi<void>(`/parts/${id}`, { method: 'DELETE' }),

  createWithFile: async (
    projectId: string,
    data: Partial<import('../types').Part>,
    file?: File,
    notes?: string
  ) => {
    if (!file) {
      return partsApi.create(projectId, data)
    }

    const formData = new FormData()
    if (data.name) formData.append('name', data.name)
    if (data.description) formData.append('description', data.description)
    formData.append('quantity', String(data.quantity || 1))
    formData.append('file', file)
    if (notes) formData.append('notes', notes)

    const response = await fetch(`${API_URL}/api/projects/${projectId}/parts`, {
      method: 'POST',
      body: formData,
    })

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Upload failed' }))
      throw new Error(error.error)
    }

    return response.json()
  },
}

// Project Supplies API
export const suppliesApi = {
  listByProject: (projectId: string) =>
    fetchApi<import('../types').ProjectSupply[]>(`/projects/${projectId}/supplies`),

  create: (projectId: string, data: { name: string; unit_cost_cents: number; quantity: number; notes?: string; material_id?: string }) =>
    fetchApi<import('../types').ProjectSupply>(`/projects/${projectId}/supplies`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  delete: (id: string) =>
    fetchApi<void>(`/supplies/${id}`, { method: 'DELETE' }),
}

// Designs API
export const designsApi = {
  listByPart: (partId: string) =>
    fetchApi<import('../types').Design[]>(`/parts/${partId}/designs`),

  get: (id: string) =>
    fetchApi<import('../types').Design>(`/designs/${id}`),

  upload: async (partId: string, file: File, notes?: string) => {
    const formData = new FormData()
    formData.append('file', file)
    if (notes) formData.append('notes', notes)

    const response = await fetch(`${API_URL}/api/parts/${partId}/designs`, {
      method: 'POST',
      body: formData,
    })

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Upload failed' }))
      throw new Error(error.error)
    }

    return response.json() as Promise<import('../types').Design>
  },

  downloadUrl: (id: string) => `${API_URL}/api/designs/${id}/download`,

  listPrintJobs: (designId: string) =>
    fetchApi<import('../types').PrintJob[]>(`/designs/${designId}/print-jobs`),

  openExternal: (id: string, app?: string) =>
    fetchApi<{ status: string }>(`/designs/${id}/open-external`, {
      method: 'POST',
      body: JSON.stringify({ app: app || '' }),
    }),
}

// Discovered printer from network scan
export interface DiscoveredPrinter {
  id: string
  name: string
  host: string
  port: number
  type: import('../types').ConnectionType
  model?: string
  manufacturer?: string
  version?: string
  serial_number?: string
  already_added: boolean
}

// Printers API
export const printersApi = {
  list: () => 
    fetchApi<import('../types').Printer[]>('/printers'),
  
  get: (id: string) => 
    fetchApi<import('../types').Printer>(`/printers/${id}`),
  
  create: (data: Partial<import('../types').Printer>) => 
    fetchApi<import('../types').Printer>('/printers', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  
  update: (id: string, data: Partial<import('../types').Printer>) => 
    fetchApi<import('../types').Printer>(`/printers/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    }),
  
  delete: (id: string) => 
    fetchApi<void>(`/printers/${id}`, { method: 'DELETE' }),
  
  getState: (id: string) =>
    fetchApi<import('../types').PrinterState>(`/printers/${id}/state`),

  getAllStates: () =>
    fetchApi<Record<string, import('../types').PrinterState>>('/printers/states'),

  getJobs: (id: string) =>
    fetchApi<import('../types').PrintJob[]>(`/printers/${id}/jobs`),

  getStats: (id: string) =>
    fetchApi<import('../types').JobStats>(`/printers/${id}/stats`),

  getAnalytics: (id: string) =>
    fetchApi<import('../types').PrinterAnalytics>(`/printers/${id}/analytics`),

  // Discover printers on local network
  discover: async () => {
    console.log('Starting printer discovery...')
    try {
      const result = await fetchApi<DiscoveredPrinter[]>('/printers/discover', { method: 'POST' })
      console.log('Discovery result:', result)
      return result
    } catch (err) {
      console.error('Discovery error:', err)
      throw err
    }
  },
}

// Materials API
export const materialsApi = {
  list: () =>
    fetchApi<import('../types').Material[]>('/materials'),

  listByType: (type: string) =>
    fetchApi<import('../types').Material[]>(`/materials?type=${encodeURIComponent(type)}`),

  get: (id: string) =>
    fetchApi<import('../types').Material>(`/materials/${id}`),

  create: (data: Partial<import('../types').Material>) =>
    fetchApi<import('../types').Material>('/materials', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  delete: (id: string) =>
    fetchApi<void>(`/materials/${id}`, { method: 'DELETE' }),
}

// Spools API
export const spoolsApi = {
  list: () => 
    fetchApi<import('../types').MaterialSpool[]>('/spools'),
  
  get: (id: string) => 
    fetchApi<import('../types').MaterialSpool>(`/spools/${id}`),
  
  create: (data: Partial<import('../types').MaterialSpool>) => 
    fetchApi<import('../types').MaterialSpool>('/spools', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
}

// Print Jobs API
export const printJobsApi = {
  list: (params?: { printer_id?: string; status?: string }) => {
    const searchParams = new URLSearchParams()
    if (params?.printer_id) searchParams.set('printer_id', params.printer_id)
    if (params?.status) searchParams.set('status', params.status)
    const query = searchParams.toString()
    return fetchApi<import('../types').PrintJob[]>(`/print-jobs${query ? `?${query}` : ''}`)
  },

  get: (id: string) =>
    fetchApi<import('../types').PrintJob>(`/print-jobs/${id}`),

  create: (data: Partial<import('../types').PrintJob>) =>
    fetchApi<import('../types').PrintJob>('/print-jobs', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  update: (id: string, data: Partial<import('../types').PrintJob>) =>
    fetchApi<import('../types').PrintJob>(`/print-jobs/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    }),

  start: (id: string) =>
    fetchApi<void>(`/print-jobs/${id}/start`, { method: 'POST' }),

  pause: (id: string) =>
    fetchApi<void>(`/print-jobs/${id}/pause`, { method: 'POST' }),

  resume: (id: string) =>
    fetchApi<void>(`/print-jobs/${id}/resume`, { method: 'POST' }),

  cancel: (id: string) =>
    fetchApi<void>(`/print-jobs/${id}/cancel`, { method: 'POST' }),

  recordOutcome: (id: string, outcome: import('../types').PrintOutcome) =>
    fetchApi<import('../types').PrintJob>(`/print-jobs/${id}/outcome`, {
      method: 'POST',
      body: JSON.stringify(outcome),
    }),

  // Job history methods
  getWithEvents: (id: string) =>
    fetchApi<import('../types').PrintJob>(`/print-jobs/${id}/with-events`),

  getEvents: (id: string) =>
    fetchApi<import('../types').JobEvent[]>(`/print-jobs/${id}/events`),

  getRetryChain: (id: string) =>
    fetchApi<import('../types').PrintJob[]>(`/print-jobs/${id}/retry-chain`),

  retry: (id: string, request?: import('../types').RetryJobRequest) =>
    fetchApi<import('../types').PrintJob>(`/print-jobs/${id}/retry`, {
      method: 'POST',
      body: JSON.stringify(request || {}),
    }),

  recordFailure: (id: string, request: import('../types').RecordFailureRequest) =>
    fetchApi<import('../types').PrintJob>(`/print-jobs/${id}/failure`, {
      method: 'POST',
      body: JSON.stringify(request),
    }),

  // Pre-flight check for material validation before starting
  preflightCheck: (id: string) =>
    fetchApi<import('../types').PreflightCheckResult>(`/print-jobs/${id}/preflight`),

  // Mark a failed job as scrap (no retry intended)
  markAsScrap: (id: string, request?: import('../types').ScrapRequest) =>
    fetchApi<import('../types').PrintJob>(`/print-jobs/${id}/scrap`, {
      method: 'POST',
      body: JSON.stringify(request || {}),
    }),

  // Jobs by recipe
  listByRecipe: (recipeId: string) =>
    fetchApi<import('../types').PrintJob[]>(`/templates/${recipeId}/jobs`),
}

// Expenses API
export const expensesApi = {
  list: (status?: string) =>
    fetchApi<import('../types').Expense[]>(
      `/expenses${status ? `?status=${status}` : ''}`
    ),

  get: (id: string) =>
    fetchApi<import('../types').Expense>(`/expenses/${id}`),

  uploadReceipt: async (file: File) => {
    const formData = new FormData()
    formData.append('file', file)

    const response = await fetch(`${API_URL}/api/expenses/receipt`, {
      method: 'POST',
      body: formData,
    })

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Upload failed' }))
      throw new Error(error.error)
    }

    return response.json() as Promise<import('../types').Expense>
  },

  confirm: (
    id: string,
    items: Array<{
      item_id: string
      create_spool: boolean
      material_id?: string
      new_material?: Partial<import('../types').Material>
      weight_grams?: number
    }>
  ) =>
    fetchApi<import('../types').Expense>(`/expenses/${id}/confirm`, {
      method: 'POST',
      body: JSON.stringify({ items }),
    }),

  retry: (id: string) =>
    fetchApi<import('../types').Expense>(`/expenses/${id}/retry`, {
      method: 'POST',
    }),

  delete: (id: string) =>
    fetchApi<void>(`/expenses/${id}`, { method: 'DELETE' }),
}

// Settings API
export interface AppSetting {
  key: string
  value: string
  updated_at: string
}

export const settingsApi = {
  list: () =>
    fetchApi<AppSetting[]>('/settings'),

  get: (key: string) =>
    fetchApi<AppSetting>(`/settings/${key}`),

  set: (key: string, value: string) =>
    fetchApi<{ status: string }>(`/settings/${key}`, {
      method: 'PUT',
      body: JSON.stringify({ value }),
    }),

  delete: (key: string) =>
    fetchApi<void>(`/settings/${key}`, { method: 'DELETE' }),
}

// Backups API
export const backupsApi = {
  list: () =>
    fetchApi<import('../types').BackupInfo[]>('/backups'),

  create: () =>
    fetchApi<import('../types').BackupInfo>('/backups', { method: 'POST' }),

  delete: (name: string) =>
    fetchApi<void>(`/backups/${encodeURIComponent(name)}`, { method: 'DELETE' }),

  restore: (name: string) =>
    fetchApi<{ message: string }>(`/backups/${encodeURIComponent(name)}/restore`, { method: 'POST' }),
}

// Sales API
export const salesApi = {
  list: (projectId?: string) =>
    fetchApi<import('../types').Sale[]>(
      `/sales${projectId ? `?project_id=${projectId}` : ''}`
    ),

  get: (id: string) =>
    fetchApi<import('../types').Sale>(`/sales/${id}`),

  create: (data: Partial<import('../types').Sale>) =>
    fetchApi<import('../types').Sale>('/sales', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  update: (id: string, data: Partial<import('../types').Sale>) =>
    fetchApi<import('../types').Sale>(`/sales/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    }),

  delete: (id: string) =>
    fetchApi<void>(`/sales/${id}`, { method: 'DELETE' }),

  getWeeklyInsights: () =>
    fetchApi<import('../types').WeeklyInsights>('/sales/weekly-insights'),
}

// Financial Summary
export interface FinancialSummary {
  total_expenses_cents: number
  total_sales_gross_cents: number
  total_sales_net_cents: number
  total_fees_cents: number
  total_material_cost: number
  total_material_used_grams: number
  total_cogs_cents: number
  net_profit_cents: number
  confirmed_expense_count: number
  pending_expense_count: number
  sales_count: number
  completed_print_count: number
  successful_print_count: number
}

// Stats API
export const statsApi = {
  getFinancialSummary: (period?: string) =>
    fetchApi<FinancialSummary>(`/stats/financial${period ? `?period=${period}` : ''}`),

  getTimeSeries: (period: string) =>
    fetchApi<import('../types').TimeSeriesData>(`/stats/time-series?period=${period}`),

  getExpensesByCategory: (period: string) =>
    fetchApi<import('../types').CategoryBreakdown[]>(`/stats/expenses-by-category?period=${period}`),

  getSalesByChannel: (period: string) =>
    fetchApi<import('../types').ChannelBreakdown[]>(`/stats/sales-by-channel?period=${period}`),

  getSalesByProject: () =>
    fetchApi<import('../types').ProjectSales[]>('/stats/sales-by-project'),
}

// Templates (Recipes) API
export const templatesApi = {
  list: (activeOnly?: boolean) =>
    fetchApi<import('../types').Template[]>(
      `/templates${activeOnly ? '?active=true' : ''}`
    ),

  get: (id: string) =>
    fetchApi<import('../types').Template>(`/templates/${id}`),

  create: (data: Partial<import('../types').Template>) =>
    fetchApi<import('../types').Template>('/templates', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  update: (id: string, data: Partial<import('../types').Template>) =>
    fetchApi<import('../types').Template>(`/templates/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    }),

  delete: (id: string) =>
    fetchApi<void>(`/templates/${id}`, { method: 'DELETE' }),

  addDesign: (
    id: string,
    data: { design_id: string; quantity: number; is_primary: boolean; notes?: string }
  ) =>
    fetchApi<import('../types').TemplateDesign>(`/templates/${id}/designs`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  removeDesign: (id: string, designId: string) =>
    fetchApi<void>(`/templates/${id}/designs/${designId}`, { method: 'DELETE' }),

  instantiate: (
    id: string,
    opts: {
      order_quantity?: number
      customer_notes?: string
      external_order_id?: string
      source?: string
      material_spool_id?: string
    }
  ) =>
    fetchApi<{ project: import('../types').Project; jobs: import('../types').PrintJob[] }>(
      `/templates/${id}/instantiate`,
      {
        method: 'POST',
        body: JSON.stringify(opts),
      }
    ),

  // Recipe material methods
  listMaterials: (id: string) =>
    fetchApi<import('../types').RecipeMaterial[]>(`/templates/${id}/materials`),

  addMaterial: (
    id: string,
    data: {
      material_type: string
      color_spec?: import('../types').ColorSpec
      weight_grams: number
      ams_position?: number
      sequence_order: number
      notes?: string
    }
  ) =>
    fetchApi<import('../types').RecipeMaterial>(`/templates/${id}/materials`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  updateMaterial: (id: string, materialId: string, data: Partial<import('../types').RecipeMaterial>) =>
    fetchApi<import('../types').RecipeMaterial>(`/templates/${id}/materials/${materialId}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    }),

  removeMaterial: (id: string, materialId: string) =>
    fetchApi<void>(`/templates/${id}/materials/${materialId}`, { method: 'DELETE' }),

  // Recipe compatibility methods
  getCompatiblePrinters: (id: string) =>
    fetchApi<import('../types').Printer[]>(`/templates/${id}/compatible-printers`),

  getCompatibleSpools: (id: string) =>
    fetchApi<import('../types').CompatibleSpool[]>(`/templates/${id}/compatible-spools`),

  getCostEstimate: (id: string) =>
    fetchApi<import('../types').RecipeCostEstimate>(`/templates/${id}/cost-estimate`),

  validatePrinter: (id: string, printerId: string) =>
    fetchApi<import('../types').PrinterValidationResult>(`/templates/${id}/validate-printer/${printerId}`, {
      method: 'POST',
    }),

  // Recipe supply methods
  listSupplies: (id: string) =>
    fetchApi<import('../types').RecipeSupply[]>(`/templates/${id}/supplies`),

  addSupply: (
    id: string,
    data: {
      name: string
      unit_cost_cents: number
      quantity: number
      notes?: string
      material_id?: string
    }
  ) =>
    fetchApi<import('../types').RecipeSupply>(`/templates/${id}/supplies`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  updateSupply: (id: string, supplyId: string, data: Partial<import('../types').RecipeSupply>) =>
    fetchApi<import('../types').RecipeSupply>(`/templates/${id}/supplies/${supplyId}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    }),

  removeSupply: (id: string, supplyId: string) =>
    fetchApi<void>(`/templates/${id}/supplies/${supplyId}`, { method: 'DELETE' }),

  // Analytics
  getAnalytics: (id: string) =>
    fetchApi<import('../types').TemplateAnalytics>(`/templates/${id}/analytics`),
}

// Etsy API
export const etsyApi = {
  getAuthUrl: () => fetchApi<{ url: string }>('/integrations/etsy/auth'),

  getStatus: () => fetchApi<import('../types').EtsyIntegration>('/integrations/etsy/status'),

  disconnect: () =>
    fetchApi<{ status: string }>('/integrations/etsy/disconnect', { method: 'POST' }),

  // Receipts/Orders
  syncReceipts: () =>
    fetchApi<import('../types').SyncResult>('/integrations/etsy/receipts/sync', { method: 'POST' }),

  listReceipts: (params?: { processed?: boolean; limit?: number; offset?: number }) => {
    const searchParams = new URLSearchParams()
    if (params?.processed !== undefined) searchParams.set('processed', String(params.processed))
    if (params?.limit) searchParams.set('limit', String(params.limit))
    if (params?.offset) searchParams.set('offset', String(params.offset))
    const query = searchParams.toString()
    return fetchApi<import('../types').EtsyReceipt[]>(`/integrations/etsy/receipts${query ? `?${query}` : ''}`)
  },

  getReceipt: (id: string) =>
    fetchApi<import('../types').EtsyReceipt>(`/integrations/etsy/receipts/${id}`),

  processReceipt: (id: string) =>
    fetchApi<{ project: import('../types').Project }>(`/integrations/etsy/receipts/${id}/process`, {
      method: 'POST',
    }),

  // Listings
  syncListings: () =>
    fetchApi<import('../types').SyncResult>('/integrations/etsy/listings/sync', { method: 'POST' }),

  listListings: (params?: { state?: string; limit?: number; offset?: number }) => {
    const searchParams = new URLSearchParams()
    if (params?.state) searchParams.set('state', params.state)
    if (params?.limit) searchParams.set('limit', String(params.limit))
    if (params?.offset) searchParams.set('offset', String(params.offset))
    const query = searchParams.toString()
    return fetchApi<import('../types').EtsyListing[]>(`/integrations/etsy/listings${query ? `?${query}` : ''}`)
  },

  getListing: (id: string) =>
    fetchApi<import('../types').EtsyListing>(`/integrations/etsy/listings/${id}`),

  linkListing: (id: string, data: { template_id: string; sku?: string; sync_inventory?: boolean }) =>
    fetchApi<{ status: string }>(`/integrations/etsy/listings/${id}/link`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  unlinkListing: (id: string, templateId: string) =>
    fetchApi<void>(`/integrations/etsy/listings/${id}/link?template_id=${templateId}`, {
      method: 'DELETE',
    }),

  syncInventory: (id: string) =>
    fetchApi<{ status: string }>(`/integrations/etsy/listings/${id}/sync-inventory`, {
      method: 'POST',
    }),

  // Webhook Events
  listWebhookEvents: (params?: { type?: string; limit?: number; offset?: number }) => {
    const searchParams = new URLSearchParams()
    if (params?.type) searchParams.set('type', params.type)
    if (params?.limit) searchParams.set('limit', String(params.limit))
    if (params?.offset) searchParams.set('offset', String(params.offset))
    const query = searchParams.toString()
    return fetchApi<import('../types').EtsyWebhookEvent[]>(`/integrations/etsy/webhook/events${query ? `?${query}` : ''}`)
  },

  reprocessWebhookEvent: (id: string) =>
    fetchApi<{ status: string }>(`/integrations/etsy/webhook/events/${id}/reprocess`, {
      method: 'POST',
    }),
}

// Bambu Cloud API
export const bambuCloudApi = {
  login: (email: string, password: string) =>
    fetchApi<{ status: string }>('/bambu-cloud/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    }),

  verify: (email: string, code: string) =>
    fetchApi<{ status: string }>('/bambu-cloud/verify', {
      method: 'POST',
      body: JSON.stringify({ email, code }),
    }),

  status: () =>
    fetchApi<import('../types').BambuCloudStatus>('/bambu-cloud/status'),

  devices: () =>
    fetchApi<import('../types').CloudDevice[]>('/bambu-cloud/devices'),

  addDevice: (devId: string) =>
    fetchApi<import('../types').Printer>('/bambu-cloud/devices/add', {
      method: 'POST',
      body: JSON.stringify({ dev_id: devId }),
    }),

  logout: () =>
    fetchApi<void>('/bambu-cloud/logout', { method: 'DELETE' }),
}

// Squarespace API
export const squarespaceApi = {
  // Connection
  connect: (apiKey: string) =>
    fetchApi<import('../types').SquarespaceIntegration>('/integrations/squarespace/connect', {
      method: 'POST',
      body: JSON.stringify({ api_key: apiKey })
    }),

  getStatus: () =>
    fetchApi<import('../types').SquarespaceIntegration>('/integrations/squarespace/status'),

  disconnect: () =>
    fetchApi<{ status: string }>('/integrations/squarespace/disconnect', { method: 'POST' }),

  // Orders
  syncOrders: () =>
    fetchApi<import('../types').SyncResult>('/integrations/squarespace/orders/sync', { method: 'POST' }),

  listOrders: (params?: { processed?: boolean; limit?: number; offset?: number }) => {
    const searchParams = new URLSearchParams()
    if (params?.processed !== undefined) searchParams.set('processed', String(params.processed))
    if (params?.limit) searchParams.set('limit', String(params.limit))
    if (params?.offset) searchParams.set('offset', String(params.offset))
    const query = searchParams.toString()
    return fetchApi<import('../types').SquarespaceOrder[]>(`/integrations/squarespace/orders${query ? `?${query}` : ''}`)
  },

  getOrder: (id: string) =>
    fetchApi<import('../types').SquarespaceOrder>(`/integrations/squarespace/orders/${id}`),

  processOrder: (id: string) =>
    fetchApi<{ project_id: string; project: import('../types').Project }>(`/integrations/squarespace/orders/${id}/process`, {
      method: 'POST',
    }),

  // Products
  syncProducts: () =>
    fetchApi<import('../types').SyncResult>('/integrations/squarespace/products/sync', { method: 'POST' }),

  listProducts: (params?: { limit?: number; offset?: number }) => {
    const searchParams = new URLSearchParams()
    if (params?.limit) searchParams.set('limit', String(params.limit))
    if (params?.offset) searchParams.set('offset', String(params.offset))
    const query = searchParams.toString()
    return fetchApi<import('../types').SquarespaceProduct[]>(`/integrations/squarespace/products${query ? `?${query}` : ''}`)
  },

  getProduct: (id: string) =>
    fetchApi<import('../types').SquarespaceProduct>(`/integrations/squarespace/products/${id}`),

  linkProduct: (id: string, templateId: string, sku?: string) =>
    fetchApi<{ status: string }>(`/integrations/squarespace/products/${id}/link`, {
      method: 'POST',
      body: JSON.stringify({ template_id: templateId, sku })
    }),

  unlinkProduct: (id: string, templateId: string) =>
    fetchApi<void>(`/integrations/squarespace/products/${id}/link?template_id=${templateId}`, {
      method: 'DELETE'
    }),
}

// Dispatch API (auto-dispatch queue management)
export const dispatchApi = {
  listPending: () =>
    fetchApi<import('../types').DispatchRequest[]>('/dispatch/requests'),

  confirm: (id: string) =>
    fetchApi<{ status: string }>(`/dispatch/requests/${id}/confirm`, { method: 'POST' }),

  reject: (id: string, reason?: string) =>
    fetchApi<{ status: string }>(`/dispatch/requests/${id}/reject`, {
      method: 'POST',
      body: JSON.stringify({ reason }),
    }),

  skip: (id: string) =>
    fetchApi<{ status: string }>(`/dispatch/requests/${id}/skip`, { method: 'POST' }),

  getGlobalSettings: () =>
    fetchApi<{ enabled: boolean }>('/dispatch/settings'),

  updateGlobalSettings: (enabled: boolean) =>
    fetchApi<{ status: string }>('/dispatch/settings', {
      method: 'PUT',
      body: JSON.stringify({ enabled }),
    }),

  getPrinterSettings: (printerId: string) =>
    fetchApi<import('../types').AutoDispatchSettings>(`/printers/${printerId}/dispatch-settings`),

  updatePrinterSettings: (printerId: string, settings: Partial<import('../types').AutoDispatchSettings>) =>
    fetchApi<import('../types').AutoDispatchSettings>(`/printers/${printerId}/dispatch-settings`, {
      method: 'PUT',
      body: JSON.stringify(settings),
    }),
}

// Print Jobs API extension for priority
export const printJobPriorityApi = {
  updatePriority: (id: string, priority: number) =>
    fetchApi<{ status: string }>(`/print-jobs/${id}/priority`, {
      method: 'PATCH',
      body: JSON.stringify({ priority }),
    }),
}

// ============================================
// New Feature Gap APIs
// ============================================

// Alerts API
export const alertsApi = {
  list: () =>
    fetchApi<import('../types').Alert[]>('/alerts'),

  getCounts: () =>
    fetchApi<import('../types').AlertCounts>('/alerts/counts'),

  dismiss: (type: string, entityId: string, duration?: string) =>
    fetchApi<{ status: string }>(`/alerts/${type}/${entityId}/dismiss`, {
      method: 'POST',
      body: JSON.stringify({ duration: duration || '1h' }),
    }),

  undismiss: (type: string, entityId: string) =>
    fetchApi<{ status: string }>(`/alerts/${type}/${entityId}/dismiss`, {
      method: 'DELETE',
    }),

  updateMaterialThreshold: (materialId: string, thresholdGrams: number) =>
    fetchApi<{ status: string }>(`/materials/${materialId}/threshold`, {
      method: 'PATCH',
      body: JSON.stringify({ threshold_grams: thresholdGrams }),
    }),
}

// Orders API (Unified)
export const ordersApi = {
  list: (params?: { status?: string; source?: string; limit?: number; offset?: number }) => {
    const searchParams = new URLSearchParams()
    if (params?.status) searchParams.set('status', params.status)
    if (params?.source) searchParams.set('source', params.source)
    if (params?.limit) searchParams.set('limit', String(params.limit))
    if (params?.offset) searchParams.set('offset', String(params.offset))
    const query = searchParams.toString()
    return fetchApi<import('../types').Order[]>(`/orders${query ? `?${query}` : ''}`)
  },

  get: (id: string) =>
    fetchApi<import('../types').Order>(`/orders/${id}`),

  create: (data: { customer_name: string; customer_email?: string; due_date?: string; priority?: number; notes?: string }) =>
    fetchApi<import('../types').Order>('/orders', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  update: (id: string, data: Partial<import('../types').Order>) =>
    fetchApi<import('../types').Order>(`/orders/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    }),

  delete: (id: string) =>
    fetchApi<void>(`/orders/${id}`, { method: 'DELETE' }),

  updateStatus: (id: string, status: import('../types').OrderStatus) =>
    fetchApi<import('../types').Order>(`/orders/${id}/status`, {
      method: 'PATCH',
      body: JSON.stringify({ status }),
    }),

  getProgress: (id: string) =>
    fetchApi<import('../types').OrderProgress>(`/orders/${id}/progress`),

  getCounts: () =>
    fetchApi<import('../types').OrderCounts>('/orders/counts'),

  markShipped: (id: string, trackingNumber?: string) =>
    fetchApi<import('../types').Order>(`/orders/${id}/ship`, {
      method: 'POST',
      body: JSON.stringify({ tracking_number: trackingNumber }),
    }),

  // Order items
  addItem: (orderId: string, data: { template_id?: string; sku?: string; quantity: number; notes?: string }) =>
    fetchApi<import('../types').OrderItem>(`/orders/${orderId}/items`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  removeItem: (orderId: string, itemId: string) =>
    fetchApi<void>(`/orders/${orderId}/items/${itemId}`, { method: 'DELETE' }),

  processItem: (orderId: string, itemId: string) =>
    fetchApi<import('../types').Project>(`/orders/${orderId}/items/${itemId}/process`, {
      method: 'POST',
    }),
}

// Tags API
export const tagsApi = {
  list: () =>
    fetchApi<import('../types').Tag[]>('/tags'),

  get: (id: string) =>
    fetchApi<import('../types').Tag>(`/tags/${id}`),

  create: (data: { name: string; color?: string }) =>
    fetchApi<import('../types').Tag>('/tags', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  update: (id: string, data: { name?: string; color?: string }) =>
    fetchApi<import('../types').Tag>(`/tags/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    }),

  delete: (id: string) =>
    fetchApi<void>(`/tags/${id}`, { method: 'DELETE' }),

  // Part tags
  getPartTags: (partId: string) =>
    fetchApi<import('../types').Tag[]>(`/parts/${partId}/tags`),

  addTagToPart: (partId: string, tagId: string) =>
    fetchApi<void>(`/parts/${partId}/tags/${tagId}`, { method: 'POST' }),

  removeTagFromPart: (partId: string, tagId: string) =>
    fetchApi<void>(`/parts/${partId}/tags/${tagId}`, { method: 'DELETE' }),

  // Design tags
  getDesignTags: (designId: string) =>
    fetchApi<import('../types').Tag[]>(`/designs/${designId}/tags`),

  addTagToDesign: (designId: string, tagId: string) =>
    fetchApi<void>(`/designs/${designId}/tags/${tagId}`, { method: 'POST' }),

  removeTagFromDesign: (designId: string, tagId: string) =>
    fetchApi<void>(`/designs/${designId}/tags/${tagId}`, { method: 'DELETE' }),

  // Search by tag
  listPartsByTag: (tagId: string) =>
    fetchApi<import('../types').Part[]>(`/tags/${tagId}/parts`),

  listDesignsByTag: (tagId: string) =>
    fetchApi<import('../types').Design[]>(`/tags/${tagId}/designs`),
}

// Shopify API
export const shopifyApi = {
  getAuthUrl: (shopDomain: string) =>
    fetchApi<{ auth_url: string }>(`/integrations/shopify/auth-url?shop=${encodeURIComponent(shopDomain)}`),

  getStatus: () =>
    fetchApi<import('../types').ShopifyIntegrationStatus>('/integrations/shopify/status'),

  disconnect: () =>
    fetchApi<{ status: string }>('/integrations/shopify', { method: 'DELETE' }),

  syncOrders: () =>
    fetchApi<import('../types').SyncResult>('/integrations/shopify/sync', { method: 'POST' }),

  listOrders: (params?: { processed?: boolean; limit?: number; offset?: number }) => {
    const searchParams = new URLSearchParams()
    if (params?.processed !== undefined) searchParams.set('processed', String(params.processed))
    if (params?.limit) searchParams.set('limit', String(params.limit))
    if (params?.offset) searchParams.set('offset', String(params.offset))
    const query = searchParams.toString()
    return fetchApi<import('../types').ShopifyOrder[]>(`/integrations/shopify/orders${query ? `?${query}` : ''}`)
  },

  getOrder: (id: string) =>
    fetchApi<import('../types').ShopifyOrder>(`/integrations/shopify/orders/${id}`),

  processOrder: (id: string) =>
    fetchApi<import('../types').Order>(`/integrations/shopify/orders/${id}/process`, {
      method: 'POST',
    }),

  linkProduct: (productId: string, templateId: string, sku?: string) =>
    fetchApi<{ status: string }>(`/integrations/shopify/products/${productId}/link`, {
      method: 'POST',
      body: JSON.stringify({ template_id: templateId, sku }),
    }),

  unlinkProduct: (productId: string, templateId: string) =>
    fetchApi<void>(`/integrations/shopify/products/${productId}/link?template_id=${templateId}`, {
      method: 'DELETE',
    }),
}

// Timeline API (Gantt View)
export const timelineApi = {
  getTimeline: (params?: { start?: string; end?: string }) => {
    const searchParams = new URLSearchParams()
    if (params?.start) searchParams.set('start', params.start)
    if (params?.end) searchParams.set('end', params.end)
    const query = searchParams.toString()
    return fetchApi<import('../types').TimelineItem[]>(`/timeline${query ? `?${query}` : ''}`)
  },

  getOrderTimeline: (orderId: string) =>
    fetchApi<import('../types').TimelineItem>(`/timeline/orders/${orderId}`),

  getProjectTimeline: (projectId: string) =>
    fetchApi<import('../types').TimelineItem>(`/timeline/projects/${projectId}`),
}

// WebSocket connection for real-time updates
export function createWebSocket(onMessage: (event: { type: string; data: unknown }) => void) {
  const wsUrl = API_URL.replace('http', 'ws') + '/ws'
  const ws = new WebSocket(wsUrl)

  ws.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data)
      onMessage(data)
    } catch (e) {
      console.error('Failed to parse WebSocket message:', e)
    }
  }

  ws.onerror = (error) => {
    console.error('WebSocket error:', error)
  }

  return ws
}

