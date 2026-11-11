import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { useAuth } from '../contexts/auth'
import AuthScreen from './AuthScreen'

export default function RequireAuth() {
  const { status, loading, error } = useAuth()
  const location = useLocation()

  if (loading) {
    return <AuthScreen title="RankOS" subtitle="Loading…">{null}</AuthScreen>
  }
  if (error || !status) {
    return <AuthScreen title="RankOS" subtitle="Could not reach RankOS. Check that the server is running.">{null}</AuthScreen>
  }
  if (status.setup_required) {
    return <Navigate to="/setup" replace />
  }
  if (!status.authenticated) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />
  }
  return <Outlet />
}
