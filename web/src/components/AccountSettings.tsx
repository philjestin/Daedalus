import { type FormEvent, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { KeyRound, UserPlus, Trash2 } from 'lucide-react'
import { authApi } from '../api/client'
import { useAuth } from '../contexts/auth'

export default function AccountSettings() {
  const { status } = useAuth()
  const queryClient = useQueryClient()
  const isAdmin = status?.user?.is_admin ?? false
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')
  const [passwordError, setPasswordError] = useState<string | null>(null)
  const [passwordSaved, setPasswordSaved] = useState(false)
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [makeAdmin, setMakeAdmin] = useState(false)
  const [userError, setUserError] = useState<string | null>(null)

  const users = useQuery({
    queryKey: ['auth', 'users'],
    queryFn: () => authApi.listUsers(),
    enabled: isAdmin,
  })

  const changePassword = useMutation({
    mutationFn: () => authApi.changePassword(currentPassword, newPassword),
    onSuccess: () => {
      setCurrentPassword('')
      setNewPassword('')
      setConfirmPassword('')
      setPasswordError(null)
      setPasswordSaved(true)
    },
    onError: (err: Error) => {
      setPasswordSaved(false)
      setPasswordError(err.message)
    },
  })

  const createUser = useMutation({
    mutationFn: () => authApi.createUser(username.trim(), password, makeAdmin),
    onSuccess: () => {
      setUsername('')
      setPassword('')
      setMakeAdmin(false)
      setUserError(null)
      void queryClient.invalidateQueries({ queryKey: ['auth', 'users'] })
    },
    onError: (err: Error) => setUserError(err.message),
  })

  const deleteUser = useMutation({
    mutationFn: (id: string) => authApi.deleteUser(id),
    onSuccess: () => {
      setUserError(null)
      void queryClient.invalidateQueries({ queryKey: ['auth', 'users'] })
    },
    onError: (err: Error) => setUserError(err.message),
  })

  function onPasswordSubmit(event: FormEvent) {
    event.preventDefault()
    setPasswordSaved(false)
    if (newPassword !== confirmPassword) {
      setPasswordError('Passwords do not match')
      return
    }
    changePassword.mutate()
  }

  function onCreateUser(event: FormEvent) {
    event.preventDefault()
    createUser.mutate()
  }

  return (
    <div className="bg-surface-900/50 border border-surface-800 rounded-none p-6 mb-6">
      <div className="flex items-center gap-3 mb-4">
        <div className="p-2 bg-primary/10 rounded-none">
          <KeyRound className="h-6 w-6 text-primary" />
        </div>
        <div>
          <h2 className="text-heading-xs font-display font-semibold text-surface-100">Account</h2>
          <p className="text-sm text-surface-400">
            Signed in as {status?.user?.username ?? '…'}
            {isAdmin ? ' (admin)' : ''}
          </p>
        </div>
      </div>

      <form onSubmit={onPasswordSubmit} className="space-y-3 mb-6">
        <h3 className="text-sm font-medium text-surface-200">Change password</h3>
        {passwordError && <p className="text-error text-sm" role="alert">{passwordError}</p>}
        {passwordSaved && <p className="text-success text-sm">Password updated.</p>}
        <input
          type="password"
          autoComplete="current-password"
          required
          placeholder="Current password"
          value={currentPassword}
          onChange={(e) => setCurrentPassword(e.target.value)}
          className="input w-full"
        />
        <input
          type="password"
          autoComplete="new-password"
          required
          minLength={8}
          placeholder="New password"
          value={newPassword}
          onChange={(e) => setNewPassword(e.target.value)}
          className="input w-full"
        />
        <input
          type="password"
          autoComplete="new-password"
          required
          minLength={8}
          placeholder="Confirm new password"
          value={confirmPassword}
          onChange={(e) => setConfirmPassword(e.target.value)}
          className="input w-full"
        />
        <button type="submit" className="btn btn-primary" disabled={changePassword.isPending}>
          {changePassword.isPending ? 'Saving…' : 'Update password'}
        </button>
      </form>

      {isAdmin && (
        <div className="border-t border-surface-800 pt-6 space-y-4">
          <div className="flex items-center gap-2">
            <UserPlus className="h-5 w-5 text-surface-300" />
            <h3 className="text-sm font-medium text-surface-200">Users</h3>
          </div>
          {userError && <p className="text-error text-sm" role="alert">{userError}</p>}
          {users.isError && <p className="text-error text-sm">Could not load users.</p>}
          <ul className="space-y-2">
            {(users.data ?? []).map((user) => (
              <li key={user.id} className="flex items-center justify-between gap-3 text-sm">
                <span className="text-surface-100">
                  {user.username}
                  {user.is_admin ? <span className="text-surface-400"> · admin</span> : null}
                  {user.id === status?.user?.id ? <span className="text-surface-400"> · you</span> : null}
                </span>
                {user.id !== status?.user?.id && (
                  <button
                    type="button"
                    className="btn btn-destructive btn-sm"
                    disabled={deleteUser.isPending}
                    onClick={() => {
                      if (confirm(`Remove ${user.username}?`)) {
                        deleteUser.mutate(user.id)
                      }
                    }}
                  >
                    <Trash2 className="h-4 w-4" />
                    Remove
                  </button>
                )}
              </li>
            ))}
          </ul>
          <form onSubmit={onCreateUser} className="space-y-3">
            <input
              required
              autoComplete="off"
              placeholder="Username"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              className="input w-full"
            />
            <input
              type="password"
              required
              minLength={8}
              autoComplete="new-password"
              placeholder="Password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              className="input w-full"
            />
            <label className="flex items-center gap-2 text-sm text-surface-300">
              <input
                type="checkbox"
                checked={makeAdmin}
                onChange={(e) => setMakeAdmin(e.target.checked)}
              />
              Admin
            </label>
            <button type="submit" className="btn btn-secondary" disabled={createUser.isPending}>
              {createUser.isPending ? 'Adding…' : 'Add user'}
            </button>
          </form>
        </div>
      )}
    </div>
  )
}
