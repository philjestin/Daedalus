const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'

// Generic fetch wrapper with error handling.
async function fetchApi<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const url = `${API_URL}/api${path}`
  
  console.log(`[API] ${options.method || 'GET'} ${url}`)
  
  const response = await fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...options.headers,
    },
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
  list: (status?: string) => 
    fetchApi<import('../types').Project[]>(`/projects${status ? `?status=${status}` : ''}`),
  
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
  
  get: (id: string) => 
    fetchApi<import('../types').Material>(`/materials/${id}`),
  
  create: (data: Partial<import('../types').Material>) => 
    fetchApi<import('../types').Material>('/materials', {
      method: 'POST',
      body: JSON.stringify(data),
    }),
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

  delete: (id: string) =>
    fetchApi<void>(`/expenses/${id}`, { method: 'DELETE' }),
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
}

// Financial Summary
export interface FinancialSummary {
  total_expenses_cents: number
  total_sales_gross_cents: number
  total_sales_net_cents: number
  total_fees_cents: number
  total_material_cost: number
  total_material_used_grams: number
  net_profit_cents: number
  confirmed_expense_count: number
  pending_expense_count: number
  sales_count: number
  completed_print_count: number
  successful_print_count: number
}

// Stats API
export const statsApi = {
  getFinancialSummary: () =>
    fetchApi<FinancialSummary>('/stats/financial'),
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

