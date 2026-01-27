// User types
export type UserRole = 'user' | 'admin'

export interface User {
  id: string
  email: string
  email_verified: boolean
  name: string
  role: UserRole
  is_active: boolean
  last_login_at?: string
  created_at: string
  updated_at: string
}

// Auth response from server
export interface AuthResponse {
  token: string
  expires_at: number
  user: User
}

// Request types
export interface MagicLinkRequest {
  email: string
}

export interface MagicLinkResponse {
  message: string
}

// Auth context state
export interface AuthState {
  user: User | null
  token: string | null
  isLoading: boolean
  isAuthenticated: boolean
}
