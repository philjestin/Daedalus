import { useMemo } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { authApi, type AuthStatus } from '../api/client'
import { AuthContext } from './auth'

export default function AuthProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient()
  const query = useQuery({
    queryKey: ['auth', 'status'],
    queryFn: () => authApi.status(),
    retry: false,
    staleTime: 30_000,
  })

  const value = useMemo(() => ({
    status: query.data,
    loading: query.isLoading,
    error: query.error instanceof Error ? query.error : query.error ? new Error('Could not reach RankOS') : null,
    setStatus: (status: AuthStatus) => {
      queryClient.setQueryData(['auth', 'status'], status)
    },
    logout: async () => {
      try {
        await authApi.logout()
      } finally {
        queryClient.setQueryData(['auth', 'status'], {
          authenticated: false,
          setup_required: false,
        } satisfies AuthStatus)
      }
    },
  }), [query.data, query.isLoading, query.error, queryClient])

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
