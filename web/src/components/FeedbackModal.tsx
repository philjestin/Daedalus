import { useState } from 'react'
import { useLocation } from 'react-router-dom'
import { X } from 'lucide-react'
import { cn } from '../lib/utils'
import { feedbackApi } from '../api/client'

interface FeedbackModalProps {
  onClose: () => void
}

const FEEDBACK_TYPES = [
  { value: 'bug', label: 'Bug Report' },
  { value: 'feature', label: 'Feature Request' },
  { value: 'general', label: 'General' },
] as const

export default function FeedbackModal({ onClose }: FeedbackModalProps) {
  const location = useLocation()
  const [type, setType] = useState<string>('general')
  const [message, setMessage] = useState('')
  const [contact, setContact] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [submitted, setSubmitted] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!message.trim()) return

    setLoading(true)
    setError(null)

    try {
      await feedbackApi.submit({
        type,
        message: message.trim(),
        contact: contact.trim() || undefined,
        page: location.pathname,
      })
      setSubmitted(true)
      setTimeout(onClose, 1500)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to submit feedback')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
      <div className="card w-full max-w-lg p-6 m-4">
        {submitted ? (
          <div className="text-center py-8">
            <div className="w-12 h-12 rounded-full bg-green-500/20 flex items-center justify-center mx-auto mb-4">
              <svg className="w-6 h-6 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
            </div>
            <h3 className="text-lg font-semibold text-surface-100">Thanks for your feedback!</h3>
            <p className="text-sm text-surface-400 mt-1">We appreciate you helping improve Daedalus.</p>
          </div>
        ) : (
          <>
            {/* Header */}
            <div className="flex items-center justify-between mb-5">
              <h2 className="text-xl font-semibold text-surface-100">Send Feedback</h2>
              <button
                onClick={onClose}
                className="w-8 h-8 flex items-center justify-center rounded-lg text-surface-400 hover:text-surface-100 hover:bg-surface-800 transition-colors"
              >
                <X className="h-5 w-5" />
              </button>
            </div>

            {error && (
              <div className="mb-4 p-3 bg-red-500/20 border border-red-500/50 rounded text-red-300 text-sm">
                {error}
              </div>
            )}

            <form onSubmit={handleSubmit} className="space-y-4">
              {/* Type selector */}
              <div>
                <label className="block text-sm font-medium text-surface-300 mb-2">Type</label>
                <div className="flex gap-2">
                  {FEEDBACK_TYPES.map((ft) => (
                    <button
                      key={ft.value}
                      type="button"
                      onClick={() => setType(ft.value)}
                      className={cn(
                        'flex-1 px-3 py-2 text-sm font-medium rounded-lg border transition-colors',
                        type === ft.value
                          ? 'border-accent-500 bg-accent-500/20 text-accent-300'
                          : 'border-surface-700 text-surface-400 hover:border-surface-600 hover:text-surface-200'
                      )}
                    >
                      {ft.label}
                    </button>
                  ))}
                </div>
              </div>

              {/* Message */}
              <div>
                <label className="block text-sm font-medium text-surface-300 mb-2">Message</label>
                <textarea
                  value={message}
                  onChange={(e) => setMessage(e.target.value)}
                  placeholder="Tell us what's on your mind..."
                  required
                  rows={4}
                  className="w-full bg-surface-800 border border-surface-700 rounded-lg px-3 py-2 text-surface-100 placeholder-surface-500 resize-none focus:outline-none focus:border-accent-500 transition-colors"
                />
              </div>

              {/* Contact */}
              <div>
                <label className="block text-sm font-medium text-surface-300 mb-2">
                  Contact <span className="text-surface-500">(optional)</span>
                </label>
                <input
                  type="text"
                  value={contact}
                  onChange={(e) => setContact(e.target.value)}
                  placeholder="Email if you'd like a response"
                  className="w-full bg-surface-800 border border-surface-700 rounded-lg px-3 py-2 text-surface-100 placeholder-surface-500 focus:outline-none focus:border-accent-500 transition-colors"
                />
              </div>

              {/* Submit */}
              <div className="flex justify-end gap-3 pt-2">
                <button
                  type="button"
                  onClick={onClose}
                  className="px-4 py-2 text-surface-400 hover:text-surface-200 transition-colors"
                  disabled={loading}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={loading || !message.trim()}
                  className="px-5 py-2 bg-accent-600 hover:bg-accent-500 text-white rounded-lg font-medium transition-colors disabled:opacity-50"
                >
                  {loading ? 'Sending...' : 'Send Feedback'}
                </button>
              </div>
            </form>
          </>
        )}
      </div>
    </div>
  )
}
