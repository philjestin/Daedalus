import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { Store, ExternalLink, RefreshCw, Unplug, AlertCircle, CheckCircle2, Key, Eye, EyeOff, Save } from 'lucide-react'
import { etsyApi, settingsApi } from '../api/client'
import type { EtsyIntegration } from '../types'

export default function Settings() {
  const [searchParams, setSearchParams] = useSearchParams()
  const [etsyStatus, setEtsyStatus] = useState<EtsyIntegration | null>(null)
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

  // Load Etsy status and API key settings
  useEffect(() => {
    loadEtsyStatus()
    loadApiKeys()
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
      </div>
    </div>
  )
}
