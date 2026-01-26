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

