import { useState, useEffect } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { PlayCircle, SkipForward, XCircle, Clock, Printer, FileText } from 'lucide-react'
import { dispatchApi } from '../api/client'
import type { DispatchRequest } from '../types'

interface DispatchNotificationProps {
  request: DispatchRequest
  onDismiss: () => void
}

export default function DispatchNotification({ request, onDismiss }: DispatchNotificationProps) {
  const queryClient = useQueryClient()
  const [timeLeft, setTimeLeft] = useState('')

  // Calculate time left until expiration
  useEffect(() => {
    const updateTimeLeft = () => {
      const expiresAt = new Date(request.expires_at).getTime()
      const now = Date.now()
      const diff = expiresAt - now

      if (diff <= 0) {
        setTimeLeft('Expired')
        return
      }

      const minutes = Math.floor(diff / 60000)
      const seconds = Math.floor((diff % 60000) / 1000)
      setTimeLeft(`${minutes}:${seconds.toString().padStart(2, '0')}`)
    }

    updateTimeLeft()
    const interval = setInterval(updateTimeLeft, 1000)
    return () => clearInterval(interval)
  }, [request.expires_at])

  const confirmMutation = useMutation({
    mutationFn: () => dispatchApi.confirm(request.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dispatch-requests'] })
      queryClient.invalidateQueries({ queryKey: ['print-jobs'] })
      onDismiss()
    },
  })

  const rejectMutation = useMutation({
    mutationFn: () => dispatchApi.reject(request.id, 'User declined'),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dispatch-requests'] })
      onDismiss()
    },
  })

  const skipMutation = useMutation({
    mutationFn: () => dispatchApi.skip(request.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['dispatch-requests'] })
      queryClient.invalidateQueries({ queryKey: ['print-jobs'] })
    },
  })

  const isLoading = confirmMutation.isPending || rejectMutation.isPending || skipMutation.isPending

  return (
    <div className="fixed bottom-4 right-4 z-50 w-96 animate-in slide-in-from-bottom-4">
      <div className="card border-accent-500/50 shadow-lg shadow-accent-500/10">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-surface-700">
          <div className="flex items-center gap-2">
            <div className="h-2 w-2 rounded-full bg-accent-500 animate-pulse" />
            <span className="font-semibold text-surface-100">Ready to Print</span>
          </div>
          <div className="flex items-center gap-1 text-sm text-surface-400">
            <Clock className="h-4 w-4" />
            <span>{timeLeft}</span>
          </div>
        </div>

        {/* Content */}
        <div className="p-4 space-y-3">
          {/* Printer Info */}
          <div className="flex items-center gap-3 p-3 rounded-lg bg-surface-800/50">
            <Printer className="h-5 w-5 text-surface-400" />
            <div>
              <div className="text-sm text-surface-300">Printer</div>
              <div className="font-medium text-surface-100">
                {request.printer?.name || 'Unknown Printer'}
              </div>
            </div>
          </div>

          {/* Job Info */}
          <div className="flex items-center gap-3 p-3 rounded-lg bg-surface-800/50">
            <FileText className="h-5 w-5 text-surface-400" />
            <div>
              <div className="text-sm text-surface-300">Job</div>
              <div className="font-medium text-surface-100">
                {request.job?.notes || `Job #${request.job_id.slice(0, 8)}`}
              </div>
              {request.job?.priority !== undefined && request.job.priority > 0 && (
                <div className="text-xs text-accent-400 mt-1">
                  Priority: {request.job.priority}
                </div>
              )}
            </div>
          </div>

          {/* Confirmation prompt */}
          <div className="text-sm text-surface-400 text-center">
            Is the bed clear and ready for the next print?
          </div>
        </div>

        {/* Actions */}
        <div className="p-4 border-t border-surface-700 space-y-2">
          <button
            onClick={() => confirmMutation.mutate()}
            disabled={isLoading}
            className="btn btn-primary w-full flex items-center justify-center gap-2"
          >
            <PlayCircle className="h-4 w-4" />
            Bed Clear - Start Print
          </button>

          <div className="flex gap-2">
            <button
              onClick={() => skipMutation.mutate()}
              disabled={isLoading}
              className="btn btn-secondary flex-1 flex items-center justify-center gap-2"
            >
              <SkipForward className="h-4 w-4" />
              Skip Job
            </button>
            <button
              onClick={() => rejectMutation.mutate()}
              disabled={isLoading}
              className="btn btn-ghost flex-1 flex items-center justify-center gap-2"
            >
              <XCircle className="h-4 w-4" />
              Not Now
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
