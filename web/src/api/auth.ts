import type { AuthResponse, MagicLinkResponse, User } from '../types/auth'

const API_URL = import.meta.env.VITE_API_URL ?? (import.meta.env.DEV ? 'http://localhost:8080' : '')

// Get stored token
export function getToken(): string | null {
  return localStorage.getItem('auth_token')
}

// Set stored token
export function setToken(token: string): void {
  localStorage.setItem('auth_token', token)
}

// Remove stored token
export function removeToken(): void {
  localStorage.removeItem('auth_token')
}

// Check if token is expired
export function isTokenExpired(): boolean {
  const expiresAt = localStorage.getItem('auth_expires_at')
  if (!expiresAt) return true
  return Date.now() > parseInt(expiresAt, 10) * 1000
}

// Set token expiry
export function setTokenExpiry(expiresAt: number): void {
  localStorage.setItem('auth_expires_at', expiresAt.toString())
}

// Generic fetch with auth header
async function fetchWithAuth<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const url = `${API_URL}/api${path}`
  const token = getToken()

  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }

  // Merge any existing headers
  if (options.headers) {
    const existingHeaders = options.headers as Record<string, string>
    Object.assign(headers, existingHeaders)
  }

  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }

  const response = await fetch(url, {
    ...options,
    headers,
  })

  if (!response.ok) {
    const error = await response.json().catch(() => ({ error: 'Unknown error' }))
    throw new Error(error.error || `HTTP ${response.status}`)
  }

  if (response.status === 204) {
    return undefined as T
  }

  return response.json() as Promise<T>
}

// Auth API
export const authApi = {
  // Request magic link email
  requestMagicLink: (email: string) =>
    fetchWithAuth<MagicLinkResponse>('/auth/request-link', {
      method: 'POST',
      body: JSON.stringify({ email }),
    }),

  // Verify magic link token
  verify: (token: string) =>
    fetchWithAuth<AuthResponse>(`/auth/verify?token=${encodeURIComponent(token)}`),

  // Get current user
  me: () => fetchWithAuth<User>('/auth/me'),

  // Logout
  logout: () =>
    fetchWithAuth<{ message: string }>('/auth/logout', {
      method: 'POST',
    }),
}
