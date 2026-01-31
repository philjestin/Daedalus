import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Settings, Zap, Clock, Play, Bell } from 'lucide-react'
import { dispatchApi } from '../api/client'
import { cn } from '../lib/utils'

interface AutoDispatchSettingsProps {
  printerId: string
}

export default function AutoDispatchSettings({ printerId }: AutoDispatchSettingsProps) {
  const queryClient = useQueryClient()

  const { data: settings, isLoading } = useQuery({
    queryKey: ['printer-dispatch-settings', printerId],
    queryFn: () => dispatchApi.getPrinterSettings(printerId),
  })

  const updateMutation = useMutation({
    mutationFn: (updates: Partial<import('../types').AutoDispatchSettings>) =>
      dispatchApi.updatePrinterSettings(printerId, updates),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['printer-dispatch-settings', printerId] })
    },
  })

  if (isLoading) {
    return (
      <div className="card p-6">
        <div className="flex items-center gap-2 mb-4">
          <Zap className="h-5 w-5 text-surface-400" />
          <h2 className="text-lg font-semibold text-surface-100">Auto-Dispatch</h2>
        </div>
        <div className="text-surface-500 text-center py-4">Loading...</div>
      </div>
    )
  }

  if (!settings) {
    return null
  }

  const handleToggle = (field: keyof typeof settings, value: boolean) => {
    updateMutation.mutate({ [field]: value })
  }

  const handleTimeoutChange = (minutes: number) => {
    updateMutation.mutate({ timeout_minutes: minutes })
  }

  return (
    <div className="card p-6">
      <div className="flex items-center gap-2 mb-4">
        <Zap className="h-5 w-5 text-surface-400" />
        <h2 className="text-lg font-semibold text-surface-100">Auto-Dispatch</h2>
      </div>

      <div className="space-y-4">
        {/* Enable toggle */}
        <div className="flex items-center justify-between p-3 rounded-lg bg-surface-800/50">
          <div className="flex items-center gap-3">
            <Settings className="h-4 w-4 text-surface-400" />
            <div>
              <div className="text-surface-200">Enable Auto-Dispatch</div>
              <div className="text-sm text-surface-500">
                Automatically queue jobs when printer becomes idle
              </div>
            </div>
          </div>
          <button
            onClick={() => handleToggle('enabled', !settings.enabled)}
            disabled={updateMutation.isPending}
            className={cn(
              'relative inline-flex h-6 w-11 items-center rounded-full transition-colors',
              settings.enabled ? 'bg-accent-500' : 'bg-surface-600'
            )}
          >
            <span
              className={cn(
                'inline-block h-4 w-4 transform rounded-full bg-white transition-transform',
                settings.enabled ? 'translate-x-6' : 'translate-x-1'
              )}
            />
          </button>
        </div>

        {/* Require confirmation toggle */}
        <div className="flex items-center justify-between p-3 rounded-lg bg-surface-800/50">
          <div className="flex items-center gap-3">
            <Bell className="h-4 w-4 text-surface-400" />
            <div>
              <div className="text-surface-200">Require Confirmation</div>
              <div className="text-sm text-surface-500">
                Ask for "bed clear" confirmation before starting
              </div>
            </div>
          </div>
          <button
            onClick={() => handleToggle('require_confirmation', !settings.require_confirmation)}
            disabled={updateMutation.isPending || !settings.enabled}
            className={cn(
              'relative inline-flex h-6 w-11 items-center rounded-full transition-colors',
              !settings.enabled && 'opacity-50 cursor-not-allowed',
              settings.require_confirmation ? 'bg-accent-500' : 'bg-surface-600'
            )}
          >
            <span
              className={cn(
                'inline-block h-4 w-4 transform rounded-full bg-white transition-transform',
                settings.require_confirmation ? 'translate-x-6' : 'translate-x-1'
              )}
            />
          </button>
        </div>

        {/* Auto-start toggle */}
        <div className="flex items-center justify-between p-3 rounded-lg bg-surface-800/50">
          <div className="flex items-center gap-3">
            <Play className="h-4 w-4 text-surface-400" />
            <div>
              <div className="text-surface-200">Auto-Start After Confirm</div>
              <div className="text-sm text-surface-500">
                Immediately start printing after confirmation
              </div>
            </div>
          </div>
          <button
            onClick={() => handleToggle('auto_start', !settings.auto_start)}
            disabled={updateMutation.isPending || !settings.enabled}
            className={cn(
              'relative inline-flex h-6 w-11 items-center rounded-full transition-colors',
              !settings.enabled && 'opacity-50 cursor-not-allowed',
              settings.auto_start ? 'bg-accent-500' : 'bg-surface-600'
            )}
          >
            <span
              className={cn(
                'inline-block h-4 w-4 transform rounded-full bg-white transition-transform',
                settings.auto_start ? 'translate-x-6' : 'translate-x-1'
              )}
            />
          </button>
        </div>

        {/* Timeout setting */}
        <div className="p-3 rounded-lg bg-surface-800/50">
          <div className="flex items-center gap-3 mb-3">
            <Clock className="h-4 w-4 text-surface-400" />
            <div>
              <div className="text-surface-200">Confirmation Timeout</div>
              <div className="text-sm text-surface-500">
                How long to wait for confirmation
              </div>
            </div>
          </div>
          <select
            value={settings.timeout_minutes}
            onChange={(e) => handleTimeoutChange(parseInt(e.target.value))}
            disabled={updateMutation.isPending || !settings.enabled}
            className={cn(
              'w-full bg-surface-700 border border-surface-600 rounded-lg px-3 py-2 text-surface-100',
              !settings.enabled && 'opacity-50 cursor-not-allowed'
            )}
          >
            <option value={5}>5 minutes</option>
            <option value={10}>10 minutes</option>
            <option value={15}>15 minutes</option>
            <option value={30}>30 minutes</option>
            <option value={60}>1 hour</option>
            <option value={120}>2 hours</option>
          </select>
        </div>
      </div>

      {/* Status indicator */}
      {settings.enabled && (
        <div className="mt-4 p-3 rounded-lg bg-accent-500/10 border border-accent-500/20">
          <div className="flex items-center gap-2 text-accent-400 text-sm">
            <div className="h-2 w-2 rounded-full bg-accent-500 animate-pulse" />
            Auto-dispatch is active for this printer
          </div>
        </div>
      )}
    </div>
  )
}
