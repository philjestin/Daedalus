import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Store, ShoppingBag, ExternalLink, RefreshCw, Unplug, AlertCircle, CheckCircle2, Key, Eye, EyeOff, Save, Database, Download, Trash2, RotateCcw, Plus } from 'lucide-react'
import { etsyApi, squarespaceApi, settingsApi, backupsApi } from '../api/client'
import type { EtsyIntegration, SquarespaceIntegration, BackupInfo } from '../types'

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

export default function Settings() {
  const [searchParams, setSearchParams] = useSearchParams()
  const [etsyStatus, setEtsyStatus] = useState<EtsyIntegration | null>(null)
  const [squarespaceStatus, setSquarespaceStatus] = useState<SquarespaceIntegration | null>(null)
  const [loading, setLoading] = useState(true)
  const [connecting, setConnecting] = useState(false)
  const [disconnecting, setDisconnecting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [successMessage, setSuccessMessage] = useState<string | null>(null)

  // API Key settings
  const [anthropicKey, setAnthropicKey] = useState('')
  const [anthropicKeyMasked, setAnthropicKeyMasked] = useState('')
  const [showAnthropicKey, setShowAnthropicKey] = useState(false)
  const [savingKey, setSavingKey] = useState(false)
  const [keyLoaded, setKeyLoaded] = useState(false)

  // Squarespace API key
  const [squarespaceKey, setSquarespaceKey] = useState('')
  const [showSquarespaceKey, setShowSquarespaceKey] = useState(false)
  const [connectingSquarespace, setConnectingSquarespace] = useState(false)
  const [disconnectingSquarespace, setDisconnectingSquarespace] = useState(false)

  // Backup settings
  const [backups, setBackups] = useState<BackupInfo[]>([])
  const [backupsLoading, setBackupsLoading] = useState(true)
  const [creatingBackup, setCreatingBackup] = useState(false)
  const [deletingBackup, setDeletingBackup] = useState<string | null>(null)
  const [restoringBackup, setRestoringBackup] = useState<string | null>(null)

  // Check for OAuth callback results
  useEffect(() => {
    const etsyParam = searchParams.get('etsy')
    if (etsyParam === 'connected') {
      setSuccessMessage('Successfully connected to Etsy!')
      searchParams.delete('etsy')
      setSearchParams(searchParams, { replace: true })
    } else if (etsyParam === 'error') {
      const message = searchParams.get('message') || 'Connection failed'
      setError(`Etsy connection error: ${message}`)
      searchParams.delete('etsy')
      searchParams.delete('message')
      setSearchParams(searchParams, { replace: true })
    }
  }, [searchParams, setSearchParams])

  // Load Etsy status, Squarespace status, and API key settings
  useEffect(() => {
    loadEtsyStatus()
    loadSquarespaceStatus()
    loadApiKeys()
    loadBackups()
  }, [])

  const loadEtsyStatus = async () => {
    try {
      setLoading(true)
      const status = await etsyApi.getStatus()
      setEtsyStatus(status)
    } catch (err) {
      console.error('Failed to load Etsy status:', err)
      setError('Failed to load Etsy integration status')
    } finally {
      setLoading(false)
    }
  }

  const loadSquarespaceStatus = async () => {
    try {
      const status = await squarespaceApi.getStatus()
      setSquarespaceStatus(status)
    } catch (err) {
      console.error('Failed to load Squarespace status:', err)
    }
  }

  const handleConnectSquarespace = async () => {
    if (!squarespaceKey.trim()) {
      setError('Please enter a Squarespace API key')
      return
    }

    try {
      setConnectingSquarespace(true)
      setError(null)
      const result = await squarespaceApi.connect(squarespaceKey.trim())
      setSquarespaceStatus(result)
      setSquarespaceKey('')
      setShowSquarespaceKey(false)
      setSuccessMessage('Successfully connected to Squarespace!')
    } catch (err) {
      console.error('Failed to connect Squarespace:', err)
      setError(err instanceof Error ? err.message : 'Failed to connect to Squarespace')
    } finally {
      setConnectingSquarespace(false)
    }
  }

  const handleDisconnectSquarespace = async () => {
    if (!confirm('Are you sure you want to disconnect your Squarespace store?')) {
      return
    }

    try {
      setDisconnectingSquarespace(true)
      setError(null)
      await squarespaceApi.disconnect()
      setSquarespaceStatus({ connected: false })
      setSuccessMessage('Squarespace store disconnected')
    } catch (err) {
      console.error('Failed to disconnect Squarespace:', err)
      setError(err instanceof Error ? err.message : 'Failed to disconnect Squarespace')
    } finally {
      setDisconnectingSquarespace(false)
    }
  }

  const loadApiKeys = async () => {
    try {
      const setting = await settingsApi.get('anthropic_api_key')
      setAnthropicKeyMasked(setting.value)
      setKeyLoaded(true)
    } catch {
      // Key not set yet — that's fine
      setKeyLoaded(true)
    }
  }

  const loadBackups = async () => {
    try {
      setBackupsLoading(true)
      const list = await backupsApi.list()
      setBackups(list || [])
    } catch (err) {
      console.error('Failed to load backups:', err)
    } finally {
      setBackupsLoading(false)
    }
  }

  const handleSaveAnthropicKey = async () => {
    if (!anthropicKey.trim()) return

    try {
      setSavingKey(true)
      setError(null)
      await settingsApi.set('anthropic_api_key', anthropicKey.trim())
      setSuccessMessage('Anthropic API key saved')
      setAnthropicKeyMasked(
        anthropicKey.trim().length > 8
          ? anthropicKey.trim().slice(0, 4) + '...' + anthropicKey.trim().slice(-4)
          : '****'
      )
      setAnthropicKey('')
      setShowAnthropicKey(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save API key')
    } finally {
      setSavingKey(false)
    }
  }

  const handleConnectEtsy = async () => {
    try {
      setConnecting(true)
      setError(null)
      const { url } = await etsyApi.getAuthUrl()
      // Redirect to Etsy OAuth page
      window.location.href = url
    } catch (err) {
      console.error('Failed to start Etsy OAuth:', err)
      setError(err instanceof Error ? err.message : 'Failed to start Etsy connection')
      setConnecting(false)
    }
  }

  const handleDisconnectEtsy = async () => {
    if (!confirm('Are you sure you want to disconnect your Etsy shop?')) {
      return
    }

    try {
      setDisconnecting(true)
      setError(null)
      await etsyApi.disconnect()
      setEtsyStatus({ connected: false, configured: etsyStatus?.configured || false })
      setSuccessMessage('Etsy shop disconnected')
    } catch (err) {
      console.error('Failed to disconnect Etsy:', err)
      setError(err instanceof Error ? err.message : 'Failed to disconnect Etsy')
    } finally {
      setDisconnecting(false)
    }
  }

  const handleCreateBackup = async () => {
    try {
      setCreatingBackup(true)
      setError(null)
      const backup = await backupsApi.create()
      setBackups(prev => [backup, ...prev])
      setSuccessMessage('Backup created successfully')
    } catch (err) {
      console.error('Failed to create backup:', err)
      setError(err instanceof Error ? err.message : 'Failed to create backup')
    } finally {
      setCreatingBackup(false)
    }
  }

  const handleDeleteBackup = async (name: string) => {
    if (!confirm(`Are you sure you want to delete backup "${name}"?`)) {
      return
    }

    try {
      setDeletingBackup(name)
      setError(null)
      await backupsApi.delete(name)
      setBackups(prev => prev.filter(b => b.name !== name))
      setSuccessMessage('Backup deleted')
    } catch (err) {
      console.error('Failed to delete backup:', err)
      setError(err instanceof Error ? err.message : 'Failed to delete backup')
    } finally {
      setDeletingBackup(null)
    }
  }

  const handleRestoreBackup = async (name: string) => {
    if (!confirm(`Are you sure you want to restore from "${name}"?\n\nThis will replace your current database. The application will need to be restarted.`)) {
      return
    }

    try {
      setRestoringBackup(name)
      setError(null)
      const result = await backupsApi.restore(name)
      setSuccessMessage(result.message + ' Please restart the application.')
    } catch (err) {
      console.error('Failed to restore backup:', err)
      setError(err instanceof Error ? err.message : 'Failed to restore backup')
    } finally {
      setRestoringBackup(null)
    }
  }

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return 'Never'
    return new Date(dateStr).toLocaleString()
  }

  const isTokenExpiringSoon = () => {
    if (!etsyStatus?.token_expires_at) return false
    const expiresAt = new Date(etsyStatus.token_expires_at)
    const hoursUntilExpiry = (expiresAt.getTime() - Date.now()) / (1000 * 60 * 60)
    return hoursUntilExpiry < 24
  }

  return (
    <div className="p-6">
      <div className="max-w-2xl">
        <h1 className="text-2xl font-display font-semibold text-surface-100 mb-6">
          Settings
        </h1>

        {/* Success Message */}
        {successMessage && (
          <div className="mb-6 p-4 bg-green-500/10 border border-green-500/30 rounded-lg flex items-center gap-3">
            <CheckCircle2 className="h-5 w-5 text-green-400 flex-shrink-0" />
            <p className="text-green-300">{successMessage}</p>
            <button
              onClick={() => setSuccessMessage(null)}
              className="ml-auto text-green-400 hover:text-green-300"
            >
              &times;
            </button>
          </div>
        )}

        {/* Error Message */}
        {error && (
          <div className="mb-6 p-4 bg-red-500/10 border border-red-500/30 rounded-lg flex items-center gap-3">
            <AlertCircle className="h-5 w-5 text-red-400 flex-shrink-0" />
            <p className="text-red-300">{error}</p>
            <button
              onClick={() => setError(null)}
              className="ml-auto text-red-400 hover:text-red-300"
            >
              &times;
            </button>
          </div>
        )}

        {/* Database Backups Card */}
        <div className="bg-surface-900/50 border border-surface-800 rounded-xl p-6 mb-6">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-3">
              <div className="p-2 bg-blue-500/10 rounded-lg">
                <Database className="h-6 w-6 text-blue-400" />
              </div>
              <div>
                <h2 className="text-lg font-semibold text-surface-100">Database Backups</h2>
                <p className="text-sm text-surface-400">
                  Create and restore database backups
                </p>
              </div>
            </div>
            <button
              onClick={handleCreateBackup}
              disabled={creatingBackup}
              className="flex items-center gap-2 px-3 py-2 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-lg transition-colors disabled:opacity-50 text-sm"
            >
              {creatingBackup ? (
                <RefreshCw className="h-4 w-4 animate-spin" />
              ) : (
                <Plus className="h-4 w-4" />
              )}
              Create Backup
            </button>
          </div>

          {backupsLoading ? (
            <div className="flex items-center justify-center py-8">
              <RefreshCw className="h-6 w-6 text-surface-400 animate-spin" />
            </div>
          ) : backups.length === 0 ? (
            <div className="bg-surface-800/50 rounded-lg p-4 text-center">
              <p className="text-surface-400">No backups yet. Create your first backup to protect your data.</p>
            </div>
          ) : (
            <div className="space-y-2">
              {backups.map((backup) => (
                <div
                  key={backup.name}
                  className="flex items-center justify-between bg-surface-800/50 rounded-lg p-3"
                >
                  <div className="flex items-center gap-3">
                    <Download className="h-4 w-4 text-surface-400" />
                    <div>
                      <p className="text-sm font-medium text-surface-200">{backup.name}</p>
                      <p className="text-xs text-surface-500">
                        {formatDate(backup.created_at)} · {formatBytes(backup.size)}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => handleRestoreBackup(backup.name)}
                      disabled={restoringBackup === backup.name}
                      className="flex items-center gap-1 px-2 py-1 text-xs font-medium text-amber-400 hover:text-amber-300 hover:bg-amber-500/10 rounded transition-colors disabled:opacity-50"
                      title="Restore this backup"
                    >
                      {restoringBackup === backup.name ? (
                        <RefreshCw className="h-3 w-3 animate-spin" />
                      ) : (
                        <RotateCcw className="h-3 w-3" />
                      )}
                      Restore
                    </button>
                    <button
                      onClick={() => handleDeleteBackup(backup.name)}
                      disabled={deletingBackup === backup.name}
                      className="flex items-center gap-1 px-2 py-1 text-xs font-medium text-red-400 hover:text-red-300 hover:bg-red-500/10 rounded transition-colors disabled:opacity-50"
                      title="Delete this backup"
                    >
                      {deletingBackup === backup.name ? (
                        <RefreshCw className="h-3 w-3 animate-spin" />
                      ) : (
                        <Trash2 className="h-3 w-3" />
                      )}
                      Delete
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}

          <p className="mt-4 text-xs text-surface-500">
            Backups are stored locally in your data directory. Consider copying important backups to external storage.
          </p>
        </div>

        {/* API Keys Card */}
        <div className="bg-surface-900/50 border border-surface-800 rounded-xl p-6 mb-6">
          <div className="flex items-center gap-3 mb-4">
            <div className="p-2 bg-purple-500/10 rounded-lg">
              <Key className="h-6 w-6 text-purple-400" />
            </div>
            <div>
              <h2 className="text-lg font-semibold text-surface-100">API Keys</h2>
              <p className="text-sm text-surface-400">
                Configure API keys for AI-powered features
              </p>
            </div>
          </div>

          <div className="space-y-4">
            {/* Anthropic API Key */}
            <div>
              <label className="block text-sm font-medium text-surface-300 mb-2">
                Anthropic API Key
              </label>
              <p className="text-xs text-surface-500 mb-2">
                Required for AI receipt parsing. Get your key at{' '}
                <a
                  href="https://console.anthropic.com/settings/keys"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-accent-400 hover:underline"
                >
                  console.anthropic.com
                </a>
              </p>

              {keyLoaded && anthropicKeyMasked && !anthropicKey && (
                <div className="flex items-center gap-2 mb-2">
                  <CheckCircle2 className="h-4 w-4 text-green-400" />
                  <span className="text-sm text-green-300">Key configured: {anthropicKeyMasked}</span>
                </div>
              )}

              <div className="flex gap-2">
                <div className="relative flex-1">
                  <input
                    type={showAnthropicKey ? 'text' : 'password'}
                    value={anthropicKey}
                    onChange={(e) => setAnthropicKey(e.target.value)}
                    placeholder={anthropicKeyMasked ? 'Enter new key to update...' : 'sk-ant-...'}
                    className="w-full bg-surface-800 border border-surface-700 rounded-lg px-3 py-2 text-sm text-surface-100 placeholder-surface-500 focus:outline-none focus:ring-2 focus:ring-accent-500 focus:border-transparent pr-10"
                  />
                  <button
                    type="button"
                    onClick={() => setShowAnthropicKey(!showAnthropicKey)}
                    className="absolute right-2 top-1/2 -translate-y-1/2 text-surface-400 hover:text-surface-300"
                  >
                    {showAnthropicKey ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                  </button>
                </div>
                <button
                  onClick={handleSaveAnthropicKey}
                  disabled={!anthropicKey.trim() || savingKey}
                  className="flex items-center gap-2 px-4 py-2 bg-accent-600 hover:bg-accent-700 text-white font-medium rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed text-sm"
                >
                  {savingKey ? (
                    <RefreshCw className="h-4 w-4 animate-spin" />
                  ) : (
                    <Save className="h-4 w-4" />
                  )}
                  Save
                </button>
              </div>
            </div>
          </div>
        </div>

        {/* Etsy Integration Card */}
        <div className="bg-surface-900/50 border border-surface-800 rounded-xl p-6">
          <div className="flex items-center gap-3 mb-4">
            <div className="p-2 bg-orange-500/10 rounded-lg">
              <Store className="h-6 w-6 text-orange-400" />
            </div>
            <div>
              <h2 className="text-lg font-semibold text-surface-100">Etsy Integration</h2>
              <p className="text-sm text-surface-400">
                Connect your Etsy shop to sync orders automatically
              </p>
            </div>
          </div>

          {loading ? (
            <div className="flex items-center justify-center py-8">
              <RefreshCw className="h-6 w-6 text-surface-400 animate-spin" />
            </div>
          ) : !etsyStatus?.configured ? (
            <div className="bg-surface-800/50 rounded-lg p-4">
              <p className="text-surface-300 mb-2">
                Etsy integration is not configured. To enable it:
              </p>
              <ol className="text-sm text-surface-400 list-decimal list-inside space-y-1">
                <li>Create an app on the <a href="https://www.etsy.com/developers/your-apps" target="_blank" rel="noopener noreferrer" className="text-accent-400 hover:underline">Etsy Developer Portal</a></li>
                <li>Set the redirect URI to: <code className="text-xs bg-surface-700 px-1 rounded">http://localhost:8080/api/integrations/etsy/callback</code></li>
                <li>Set the <code className="text-xs bg-surface-700 px-1 rounded">ETSY_CLIENT_ID</code> environment variable</li>
                <li>Restart the server</li>
              </ol>
            </div>
          ) : etsyStatus.connected ? (
            <div className="space-y-4">
              {/* Connected Shop Info */}
              <div className="bg-green-500/10 border border-green-500/30 rounded-lg p-4">
                <div className="flex items-center gap-2 mb-2">
                  <CheckCircle2 className="h-4 w-4 text-green-400" />
                  <span className="text-green-300 font-medium">Connected</span>
                </div>
                <div className="grid grid-cols-2 gap-4 text-sm">
                  <div>
                    <p className="text-surface-400">Shop Name</p>
                    <p className="text-surface-100 font-medium">{etsyStatus.shop_name}</p>
                  </div>
                  <div>
                    <p className="text-surface-400">Shop ID</p>
                    <p className="text-surface-100 font-medium">{etsyStatus.shop_id}</p>
                  </div>
                  <div>
                    <p className="text-surface-400">Last Synced</p>
                    <p className="text-surface-100">{formatDate(etsyStatus.last_sync_at)}</p>
                  </div>
                  <div>
                    <p className="text-surface-400">Connected Since</p>
                    <p className="text-surface-100">{formatDate(etsyStatus.created_at)}</p>
                  </div>
                </div>
              </div>

              {/* Token Expiry Warning */}
              {isTokenExpiringSoon() && (
                <div className="bg-yellow-500/10 border border-yellow-500/30 rounded-lg p-3 flex items-center gap-2">
                  <AlertCircle className="h-4 w-4 text-yellow-400" />
                  <span className="text-sm text-yellow-300">
                    Token expires soon. It will be automatically refreshed on next API call.
                  </span>
                </div>
              )}

              {/* Scopes */}
              <div>
                <p className="text-sm text-surface-400 mb-2">Permissions</p>
                <div className="flex flex-wrap gap-2">
                  {etsyStatus.scopes?.map((scope) => (
                    <span
                      key={scope}
                      className="text-xs bg-surface-700 text-surface-300 px-2 py-1 rounded"
                    >
                      {scope}
                    </span>
                  ))}
                </div>
              </div>

              {/* Disconnect Button */}
              <button
                onClick={handleDisconnectEtsy}
                disabled={disconnecting}
                className="flex items-center gap-2 px-4 py-2 text-sm font-medium text-red-400 hover:text-red-300 hover:bg-red-500/10 rounded-lg transition-colors disabled:opacity-50"
              >
                {disconnecting ? (
                  <RefreshCw className="h-4 w-4 animate-spin" />
                ) : (
                  <Unplug className="h-4 w-4" />
                )}
                Disconnect Shop
              </button>
            </div>
          ) : (
            <div className="space-y-4">
              <p className="text-surface-300">
                Connect your Etsy shop to automatically import orders and sync inventory.
              </p>
              <button
                onClick={handleConnectEtsy}
                disabled={connecting}
                className="flex items-center gap-2 px-4 py-2 bg-orange-500 hover:bg-orange-600 text-white font-medium rounded-lg transition-colors disabled:opacity-50"
              >
                {connecting ? (
                  <RefreshCw className="h-4 w-4 animate-spin" />
                ) : (
                  <ExternalLink className="h-4 w-4" />
                )}
                Connect to Etsy
              </button>
            </div>
          )}
        </div>

        {/* Squarespace Integration Card */}
        <div className="bg-surface-900/50 border border-surface-800 rounded-xl p-6">
          <div className="flex items-center gap-3 mb-4">
            <div className="p-2 bg-purple-500/10 rounded-lg">
              <ShoppingBag className="h-6 w-6 text-purple-400" />
            </div>
            <div>
              <h2 className="text-lg font-semibold text-surface-100">Squarespace Integration</h2>
              <p className="text-sm text-surface-400">
                Connect your Squarespace store to sync orders automatically
              </p>
            </div>
          </div>

          {squarespaceStatus?.connected ? (
            <div className="space-y-4">
              {/* Connected Store Info */}
              <div className="bg-green-500/10 border border-green-500/30 rounded-lg p-4">
                <div className="flex items-center gap-2 mb-2">
                  <CheckCircle2 className="h-4 w-4 text-green-400" />
                  <span className="text-green-300 font-medium">Connected</span>
                </div>
                <div className="grid grid-cols-2 gap-4 text-sm">
                  <div>
                    <p className="text-surface-400">Site Title</p>
                    <p className="text-surface-100 font-medium">{squarespaceStatus.site_title}</p>
                  </div>
                  <div>
                    <p className="text-surface-400">Site ID</p>
                    <p className="text-surface-100 font-medium">{squarespaceStatus.site_id}</p>
                  </div>
                  <div>
                    <p className="text-surface-400">Last Order Sync</p>
                    <p className="text-surface-100">{formatDate(squarespaceStatus.last_order_sync_at)}</p>
                  </div>
                  <div>
                    <p className="text-surface-400">Connected Since</p>
                    <p className="text-surface-100">{formatDate(squarespaceStatus.created_at)}</p>
                  </div>
                </div>
              </div>

              {/* Disconnect Button */}
              <button
                onClick={handleDisconnectSquarespace}
                disabled={disconnectingSquarespace}
                className="flex items-center gap-2 px-4 py-2 text-sm font-medium text-red-400 hover:text-red-300 hover:bg-red-500/10 rounded-lg transition-colors disabled:opacity-50"
              >
                {disconnectingSquarespace ? (
                  <RefreshCw className="h-4 w-4 animate-spin" />
                ) : (
                  <Unplug className="h-4 w-4" />
                )}
                Disconnect Store
              </button>
            </div>
          ) : (
            <div className="space-y-4">
              <p className="text-surface-300">
                Connect your Squarespace store to automatically import orders.
              </p>
              <p className="text-xs text-surface-500">
                Get your API key from{' '}
                <a
                  href="https://support.squarespace.com/hc/en-us/articles/12880553888141-Commerce-APIs"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-accent-400 hover:underline"
                >
                  Squarespace Developer Settings
                </a>
                . You'll need the <code className="bg-surface-700 px-1 rounded">Orders Read</code> and{' '}
                <code className="bg-surface-700 px-1 rounded">Products Read</code> permissions.
              </p>

              <div className="flex gap-2">
                <div className="relative flex-1">
                  <input
                    type={showSquarespaceKey ? 'text' : 'password'}
                    value={squarespaceKey}
                    onChange={(e) => setSquarespaceKey(e.target.value)}
                    placeholder="Enter your Squarespace API key..."
                    className="w-full bg-surface-800 border border-surface-700 rounded-lg px-3 py-2 text-sm text-surface-100 placeholder-surface-500 focus:outline-none focus:ring-2 focus:ring-accent-500 focus:border-transparent pr-10"
                  />
                  <button
                    type="button"
                    onClick={() => setShowSquarespaceKey(!showSquarespaceKey)}
                    className="absolute right-2 top-1/2 -translate-y-1/2 text-surface-400 hover:text-surface-300"
                  >
                    {showSquarespaceKey ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
                  </button>
                </div>
                <button
                  onClick={handleConnectSquarespace}
                  disabled={!squarespaceKey.trim() || connectingSquarespace}
                  className="flex items-center gap-2 px-4 py-2 bg-purple-500 hover:bg-purple-600 text-white font-medium rounded-lg transition-colors disabled:opacity-50"
                >
                  {connectingSquarespace ? (
                    <RefreshCw className="h-4 w-4 animate-spin" />
                  ) : (
                    <ExternalLink className="h-4 w-4" />
                  )}
                  Connect
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
