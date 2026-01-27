import { useState } from 'react'
import type { PrintJob, Printer, FailureCategory } from '../types'
import { printJobsApi } from '../api/client'

interface FailureModalProps {
  job: PrintJob
  printers: Printer[]
  onClose: () => void
  onRetry: (job: PrintJob) => void
  onScrap: (job: PrintJob) => void
}

const FAILURE_CATEGORIES: { value: FailureCategory; label: string; description: string }[] = [
  { value: 'adhesion', label: 'Bed Adhesion', description: 'Print detached from bed' },
  { value: 'filament', label: 'Filament Issue', description: 'Jam, runout, or tangle' },
  { value: 'mechanical', label: 'Mechanical', description: 'Printer hardware issue' },
  { value: 'thermal', label: 'Thermal', description: 'Heating or cooling failure' },
  { value: 'network', label: 'Network', description: 'Connection lost during print' },
  { value: 'user_cancelled', label: 'User Cancelled', description: 'Stopped by user' },
  { value: 'unknown', label: 'Unknown', description: 'Unclassified failure' },
]

export function FailureModal({ job, printers, onClose, onRetry, onScrap }: FailureModalProps) {
  const [action, setAction] = useState<'retry' | 'reassign' | 'scrap' | null>(null)
  const [selectedPrinterId, setSelectedPrinterId] = useState<string>(job.printer_id || '')
  const [failureCategory, setFailureCategory] = useState<FailureCategory>(job.failure_category || 'unknown')
  const [notes, setNotes] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Filter to available printers (idle)
  const availablePrinters = printers.filter(p => p.status === 'idle' || p.id === job.printer_id)
  const otherPrinters = availablePrinters.filter(p => p.id !== job.printer_id)

  const handleRetry = async (reassign: boolean) => {
    setLoading(true)
    setError(null)
    try {
      const retryJob = await printJobsApi.retry(job.id, {
        printer_id: reassign ? selectedPrinterId : job.printer_id,
        material_spool_id: job.material_spool_id,
        failure_category: failureCategory,
        notes: notes || undefined,
      })
      onRetry(retryJob)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to retry job')
    } finally {
      setLoading(false)
    }
  }

  const handleScrap = async () => {
    setLoading(true)
    setError(null)
    try {
      const scrapJob = await printJobsApi.markAsScrap(job.id, {
        failure_category: failureCategory,
        notes: notes || undefined,
      })
      onScrap(scrapJob)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to mark as scrap')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
      <div className="card w-full max-w-lg p-6 m-4">
        {/* Header */}
        <div className="flex items-center gap-3 mb-4">
          <div className="w-10 h-10 rounded-full bg-red-500/20 flex items-center justify-center">
            <svg className="w-5 h-5 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
            </svg>
          </div>
          <div>
            <h2 className="text-xl font-semibold text-surface-100">Print Failed</h2>
            <p className="text-sm text-surface-400">What would you like to do?</p>
          </div>
        </div>

        {error && (
          <div className="mb-4 p-3 bg-red-500/20 border border-red-500/50 rounded text-red-300 text-sm">
            {error}
          </div>
        )}

        {/* Action Selection */}
        {!action && (
          <div className="space-y-3">
            <button
              onClick={() => setAction('retry')}
              className="w-full p-4 text-left rounded-lg border border-surface-700 hover:border-primary-500 hover:bg-surface-800/50 transition-colors"
            >
              <div className="flex items-center gap-3">
                <svg className="w-5 h-5 text-primary-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                </svg>
                <div>
                  <div className="font-medium text-surface-100">Retry Job</div>
                  <div className="text-sm text-surface-400">Try again on the same printer</div>
                </div>
              </div>
            </button>

            {otherPrinters.length > 0 && (
              <button
                onClick={() => setAction('reassign')}
                className="w-full p-4 text-left rounded-lg border border-surface-700 hover:border-primary-500 hover:bg-surface-800/50 transition-colors"
              >
                <div className="flex items-center gap-3">
                  <svg className="w-5 h-5 text-blue-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4" />
                  </svg>
                  <div>
                    <div className="font-medium text-surface-100">Assign to Another Printer</div>
                    <div className="text-sm text-surface-400">{otherPrinters.length} other printer{otherPrinters.length > 1 ? 's' : ''} available</div>
                  </div>
                </div>
              </button>
            )}

            <button
              onClick={() => setAction('scrap')}
              className="w-full p-4 text-left rounded-lg border border-surface-700 hover:border-red-500/50 hover:bg-surface-800/50 transition-colors"
            >
              <div className="flex items-center gap-3">
                <svg className="w-5 h-5 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
                <div>
                  <div className="font-medium text-surface-100">Mark as Scrap</div>
                  <div className="text-sm text-surface-400">Discard this print, no retry</div>
                </div>
              </div>
            </button>

            <button
              onClick={onClose}
              className="w-full p-3 text-center text-surface-400 hover:text-surface-200 transition-colors"
            >
              Decide Later
            </button>
          </div>
        )}

        {/* Action Form */}
        {action && (
          <div className="space-y-4">
            {/* Failure Category Selection */}
            <div>
              <label className="block text-sm font-medium text-surface-300 mb-2">
                What caused the failure?
              </label>
              <div className="grid grid-cols-2 gap-2">
                {FAILURE_CATEGORIES.map(cat => (
                  <button
                    key={cat.value}
                    onClick={() => setFailureCategory(cat.value)}
                    className={`p-2 text-left rounded border transition-colors ${
                      failureCategory === cat.value
                        ? 'border-primary-500 bg-primary-500/20 text-surface-100'
                        : 'border-surface-700 hover:border-surface-600 text-surface-300'
                    }`}
                  >
                    <div className="text-sm font-medium">{cat.label}</div>
                  </button>
                ))}
              </div>
            </div>

            {/* Printer Selection (for reassign) */}
            {action === 'reassign' && (
              <div>
                <label className="block text-sm font-medium text-surface-300 mb-2">
                  Select Printer
                </label>
                <select
                  value={selectedPrinterId}
                  onChange={(e) => setSelectedPrinterId(e.target.value)}
                  className="w-full bg-surface-800 border border-surface-700 rounded px-3 py-2 text-surface-100"
                >
                  <option value="">Select a printer...</option>
                  {availablePrinters.map(p => (
                    <option key={p.id} value={p.id}>
                      {p.name} {p.id === job.printer_id ? '(current)' : p.status === 'idle' ? '(idle)' : `(${p.status})`}
                    </option>
                  ))}
                </select>
              </div>
            )}

            {/* Notes */}
            <div>
              <label className="block text-sm font-medium text-surface-300 mb-2">
                Notes (optional)
              </label>
              <textarea
                value={notes}
                onChange={(e) => setNotes(e.target.value)}
                placeholder="What happened? Any observations..."
                className="w-full bg-surface-800 border border-surface-700 rounded px-3 py-2 text-surface-100 placeholder-surface-500 h-20 resize-none"
              />
            </div>

            {/* Action Buttons */}
            <div className="flex gap-3 pt-2">
              <button
                onClick={() => setAction(null)}
                className="flex-1 px-4 py-2 text-surface-300 hover:text-surface-100 transition-colors"
                disabled={loading}
              >
                Back
              </button>
              {action === 'scrap' ? (
                <button
                  onClick={handleScrap}
                  disabled={loading}
                  className="flex-1 px-4 py-2 bg-red-600 hover:bg-red-700 text-white rounded font-medium transition-colors disabled:opacity-50"
                >
                  {loading ? 'Processing...' : 'Mark as Scrap'}
                </button>
              ) : (
                <button
                  onClick={() => handleRetry(action === 'reassign')}
                  disabled={loading || (action === 'reassign' && !selectedPrinterId)}
                  className="flex-1 px-4 py-2 bg-primary-600 hover:bg-primary-700 text-white rounded font-medium transition-colors disabled:opacity-50"
                >
                  {loading ? 'Processing...' : action === 'reassign' ? 'Retry on Selected Printer' : 'Retry Job'}
                </button>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
