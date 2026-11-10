import { useEffect, useState } from 'react'
import { AlertTriangle, X, Clock, Bell } from 'lucide-react'
import { alertsApi } from '../api/client'
import type { Alert, AlertSeverity } from '../types'

interface AlertBannerProps {
  className?: string
}

export function AlertBanner({ className = '' }: AlertBannerProps) {
  const [alerts, setAlerts] = useState<Alert[]>([])
  const [loading, setLoading] = useState(true)
  const [dismissingId, setDismissingId] = useState<string | null>(null)

  useEffect(() => {
    loadAlerts()
    // Poll for updates every 30 seconds
    const interval = setInterval(loadAlerts, 30000)
    return () => clearInterval(interval)
  }, [])

  const loadAlerts = async () => {
    try {
      const data = await alertsApi.list()
      setAlerts(data)
    } catch (err) {
      console.error('Failed to load alerts:', err)
    } finally {
      setLoading(false)
    }
  }

  const handleDismiss = async (alert: Alert, duration: string) => {
    setDismissingId(alert.id)
    try {
      await alertsApi.dismiss(alert.type, alert.entity_id, duration)
      setAlerts(prev => prev.filter(a => a.id !== alert.id))
    } catch (err) {
      console.error('Failed to dismiss alert:', err)
    } finally {
      setDismissingId(null)
    }
  }

  if (loading || alerts.length === 0) {
    return null
  }

  const severityStyles: Record<AlertSeverity, string> = {
    info: 'bg-info/15 border-info/35 text-info',
    warning: 'bg-warning/15 border-warning/35 text-warning',
    critical: 'bg-error/15 border-error/35 text-error',
  }

  const severityIcons: Record<AlertSeverity, React.ReactNode> = {
    info: <Bell className="w-4 h-4" />,
    warning: <AlertTriangle className="w-4 h-4" />,
    critical: <AlertTriangle className="w-4 h-4" />,
  }

  // Group alerts by severity, showing critical first
  const criticalAlerts = alerts.filter(a => a.severity === 'critical')
  const warningAlerts = alerts.filter(a => a.severity === 'warning')
  const infoAlerts = alerts.filter(a => a.severity === 'info')
  const sortedAlerts = [...criticalAlerts, ...warningAlerts, ...infoAlerts]

  return (
    <div className={`space-y-2 ${className}`}>
      {sortedAlerts.slice(0, 3).map(alert => (
        <div
          key={alert.id}
          className={`flex items-center justify-between rounded-none border px-4 py-2 ${severityStyles[alert.severity]}`}
        >
          <div className="flex items-center gap-3">
            {severityIcons[alert.severity]}
            <span className="text-sm font-medium">{alert.message}</span>
          </div>

          <div className="flex items-center gap-2">
            {/* Snooze options */}
            <div className="relative group">
              <button
                onClick={() => handleDismiss(alert, '1h')}
                disabled={dismissingId === alert.id}
                className="p-1.5 rounded-none hover:bg-ink/10 transition-colors"
                title="Snooze for 1 hour"
              >
                <Clock className="w-4 h-4" />
              </button>
              <div className="absolute right-0 top-full mt-1 hidden group-hover:block z-10">
                <div className="min-w-[120px] rounded-none border border-line bg-canvas py-1 text-ink shadow-card">
                  <button
                    onClick={() => handleDismiss(alert, '1h')}
                    className="w-full px-3 py-1.5 text-left text-sm hover:bg-surface-raised"
                  >
                    1 hour
                  </button>
                  <button
                    onClick={() => handleDismiss(alert, '4h')}
                    className="w-full px-3 py-1.5 text-left text-sm hover:bg-surface-raised"
                  >
                    4 hours
                  </button>
                  <button
                    onClick={() => handleDismiss(alert, '24h')}
                    className="w-full px-3 py-1.5 text-left text-sm hover:bg-surface-raised"
                  >
                    24 hours
                  </button>
                  <button
                    onClick={() => handleDismiss(alert, 'permanent')}
                    className="w-full px-3 py-1.5 text-left text-sm hover:bg-surface-raised"
                  >
                    Dismiss permanently
                  </button>
                </div>
              </div>
            </div>

            {/* Quick dismiss */}
            <button
              onClick={() => handleDismiss(alert, '1h')}
              disabled={dismissingId === alert.id}
              className="p-1.5 rounded-none hover:bg-ink/10 transition-colors"
              title="Dismiss"
            >
              <X className="w-4 h-4" />
            </button>
          </div>
        </div>
      ))}

      {alerts.length > 3 && (
        <div className="text-center">
          <button className="text-sm text-body hover:text-ink">
            +{alerts.length - 3} more alerts
          </button>
        </div>
      )}
    </div>
  )
}

export default AlertBanner
