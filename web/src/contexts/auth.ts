import { createContext, useContext } from 'react'
import type { AuthStatus } from '../api/client'

export interface AuthValue {
  status?: AuthStatus
  loading: boolean
  error: Error | null
  setStatus: (status: AuthStatus) => void
  logout: () => Promise<void>
}

export const AuthContext = createContext<AuthValue | null>(null)

export function useAuth(): AuthValue {
  const value = useContext(AuthContext)
  if (!value) {
    throw new Error('useAuth must be used within AuthProvider')
  }
  return value
}
