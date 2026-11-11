import { type FormEvent, useState } from 'react'
import { Navigate, useNavigate } from 'react-router-dom'
import { authApi } from '../api/client'
import { useAuth } from '../contexts/auth'
import AuthScreen from '../components/AuthScreen'

export default function Setup() {
  const { status, loading, error, setStatus } = useAuth()
  const navigate = useNavigate()
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [confirm, setConfirm] = useState('')
  const [formError, setFormError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  if (loading) {
    return <AuthScreen title="Create admin account" subtitle="Checking this install…">{null}</AuthScreen>
  }
  if (error) {
    return <AuthScreen title="Create admin account" subtitle="Could not reach RankOS. Check that the server is running.">{null}</AuthScreen>
  }
  if (status?.authenticated) {
    return <Navigate to="/dashboard" replace />
  }
  if (status && !status.setup_required) {
    return <Navigate to="/login" replace />
  }

  async function onSubmit(event: FormEvent) {
    event.preventDefault()
    if (password !== confirm) {
      setFormError('Passwords do not match')
      return
    }
    setSubmitting(true)
    setFormError(null)
    try {
      const next = await authApi.setup(username.trim(), password)
      setStatus(next)
      navigate('/dashboard', { replace: true })
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Could not create the account')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <AuthScreen
      title="Create admin account"
      subtitle="This is the first account on this RankOS install. The page closes after it exists. There is no default password."
    >
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
            autoComplete="new-password"
            required
            minLength={8}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="input w-full"
          />
        </div>
        <div>
          <label htmlFor="confirm" className="block text-sm text-body mb-1">Confirm password</label>
          <input
            id="confirm"
            name="confirm"
            type="password"
            autoComplete="new-password"
            required
            minLength={8}
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
            className="input w-full"
          />
        </div>
        <button type="submit" className="btn btn-primary w-full" disabled={submitting}>
          {submitting ? 'Creating…' : 'Create admin account'}
        </button>
      </form>
    </AuthScreen>
  )
}
