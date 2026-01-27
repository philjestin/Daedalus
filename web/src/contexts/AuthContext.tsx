import { createContext, useContext, useEffect, useState, useCallback, type ReactNode } from 'react'
import type { User, AuthState } from '../types/auth'
import { authApi, getToken, setToken, removeToken, setTokenExpiry, isTokenExpired } from '../api/auth'

interface AuthContextType extends AuthState {
  login: (email: string) => Promise<void>
  verifyToken: (token: string) => Promise<void>
  logout: () => Promise<void>
  refreshUser: () => Promise<void>
}

const AuthContext = createContext<AuthContextType | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [token, setTokenState] = useState<string | null>(getToken())
  const [isLoading, setIsLoading] = useState(true)

  const isAuthenticated = !!user && !!token && !isTokenExpired()

  // Fetch current user on mount if token exists
  useEffect(() => {
    const initAuth = async () => {
      const storedToken = getToken()
      if (storedToken && !isTokenExpired()) {
        try {
          const userData = await authApi.me()
          setUser(userData)
          setTokenState(storedToken)
        } catch (err) {
          // Token is invalid, clear it
          console.error('Failed to fetch user:', err)
          removeToken()
          setTokenState(null)
          setUser(null)
        }
      }
      setIsLoading(false)
    }

    initAuth()
  }, [])

  // Request magic link
  const login = useCallback(async (email: string) => {
    await authApi.requestMagicLink(email)
  }, [])

  // Verify magic link token
  const verifyToken = useCallback(async (tokenParam: string) => {
    setIsLoading(true)
    try {
      const response = await authApi.verify(tokenParam)
      setToken(response.token)
      setTokenExpiry(response.expires_at)
      setTokenState(response.token)
      setUser(response.user)
    } finally {
      setIsLoading(false)
    }
  }, [])

  // Logout
  const logout = useCallback(async () => {
    try {
      await authApi.logout()
    } catch (err) {
      // Ignore logout errors
      console.error('Logout error:', err)
    } finally {
      removeToken()
      setTokenState(null)
      setUser(null)
    }
  }, [])

  // Refresh user data
  const refreshUser = useCallback(async () => {
    if (!token) return
    try {
      const userData = await authApi.me()
      setUser(userData)
    } catch (err) {
      console.error('Failed to refresh user:', err)
    }
  }, [token])

  return (
    <AuthContext.Provider
      value={{
        user,
        token,
        isLoading,
        isAuthenticated,
        login,
        verifyToken,
        logout,
        refreshUser,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}
