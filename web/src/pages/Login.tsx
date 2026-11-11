import { type FormEvent, useState } from 'react'
import { Navigate, useLocation, useNavigate } from 'react-router-dom'
import { authApi } from '../api/client'
import { useAuth } from '../contexts/auth'
import AuthScreen from '../components/AuthScreen'

export default function Login() {
  const { status, loading, error, setStatus } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [formError, setFormError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  if (loading) {
    return <AuthScreen title="Sign in" subtitle="Checking your session…">{null}</AuthScreen>
  }
  if (error) {
    return <AuthScreen title="Sign in" subtitle="Could not reach RankOS. Check that the server is running.">{null}</AuthScreen>
  }
  if (status?.setup_required) {
    return <Navigate to="/setup" replace />
  }
  if (status?.authenticated) {
    return <Navigate to="/dashboard" replace />
  }

  async function onSubmit(event: FormEvent) {
    event.preventDefault()
    setSubmitting(true)
    setFormError(null)
    try {
      const next = await authApi.login(username.trim(), password)
      setStatus(next)
      const from = (location.state as { from?: string } | null)?.from
      const dest = from && from.startsWith('/') && !from.startsWith('/login') ? from : '/dashboard'
      navigate(dest, { replace: true })
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Sign in failed')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <AuthScreen title="Sign in" subtitle="Local account. Nothing leaves this machine.">
      <form onSubmit={onSubmit} className="space-y-4">
        {formError && <p className="text-error text-sm" role="alert">{formError}</p>}
        <div>
          <label htmlFor="username" className="block text-sm text-body mb-1">Username</label>
          <input
            id="username"
            name="username"
            autoComplete="username"
            required
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            className="input w-full"
          />
        </div>
        <div>
          <label htmlFor="password" className="block text-sm text-body mb-1">Password</label>
          <input
            id="password"
            name="password"
            type="password"
            autoComplete="current-password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="input w-full"
          />
        </div>
        <button type="submit" className="btn btn-primary w-full" disabled={submitting}>
          {submitting ? 'Signing in…' : 'Sign in'}
        </button>
      </form>
    </AuthScreen>
  )
}
