import { useEffect, useState } from 'react'
import { useParams, Link, useNavigate } from 'react-router-dom'
import { ArrowLeft, Send, CheckCircle, XCircle, Plus, Trash2, Clock, Edit3 } from 'lucide-react'
import { quotesApi } from '../api/client'
import type { Quote, QuoteStatus, QuoteOption, QuoteLineItemType } from '../types'

const statusConfig: Record<QuoteStatus, { label: string; color: string; icon: typeof Clock }> = {
  draft: { label: 'Draft', color: 'bg-surface-700 text-surface-300', icon: Edit3 },
  sent: { label: 'Sent', color: 'bg-blue-500/20 text-blue-400', icon: Send },
  accepted: { label: 'Accepted', color: 'bg-green-500/20 text-green-400', icon: CheckCircle },
  rejected: { label: 'Rejected', color: 'bg-red-500/20 text-red-400', icon: XCircle },
  expired: { label: 'Expired', color: 'bg-surface-700 text-surface-400', icon: Clock },
}

const lineItemTypes: { value: QuoteLineItemType; label: string }[] = [
  { value: 'printing', label: 'Printing' },
  { value: 'post_processing', label: 'Post Processing' },
  { value: 'consulting', label: 'Consulting' },
  { value: 'design', label: 'Design' },
  { value: 'other', label: 'Other' },
]

function formatCents(cents: number): string {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(cents / 100)
}

export default function QuoteDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [quote, setQuote] = useState<Quote | null>(null)
  const [loading, setLoading] = useState(true)
  const [actionLoading, setActionLoading] = useState(false)
  const [showAddOption, setShowAddOption] = useState(false)
  const [addingItemToOption, setAddingItemToOption] = useState<string | null>(null)

  const loadQuote = async () => {
    if (!id) return
    try {
      const data = await quotesApi.get(id)
      setQuote(data)
    } catch (err) {
      console.error('Failed to load quote:', err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadQuote()
  }, [id])

  const handleSend = async () => {
    if (!id) return
    setActionLoading(true)
    try {
      const updated = await quotesApi.send(id)
      setQuote(updated)
    } catch (err) {
      console.error('Failed to send quote:', err)
    } finally {
      setActionLoading(false)
    }
  }

  const handleAccept = async (optionId: string) => {
    if (!id) return
    setActionLoading(true)
    try {
      const updated = await quotesApi.accept(id, optionId)
      setQuote(updated)
    } catch (err) {
      console.error('Failed to accept quote:', err)
    } finally {
      setActionLoading(false)
    }
  }

  const handleReject = async () => {
    if (!id) return
    setActionLoading(true)
    try {
      const updated = await quotesApi.reject(id)
      setQuote(updated)
    } catch (err) {
      console.error('Failed to reject quote:', err)
    } finally {
      setActionLoading(false)
    }
  }

  const handleAddOption = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    if (!id) return
    const form = e.currentTarget
    const formData = new FormData(form)
    try {
      await quotesApi.createOption(id, {
        name: formData.get('name') as string,
        description: formData.get('description') as string || undefined,
      })
      setShowAddOption(false)
      loadQuote()
    } catch (err) {
      console.error('Failed to add option:', err)
    }
  }

  const handleDeleteOption = async (optionId: string) => {
    if (!id) return
    try {
      await quotesApi.deleteOption(id, optionId)
      loadQuote()
    } catch (err) {
      console.error('Failed to delete option:', err)
    }
  }

  const handleAddLineItem = async (e: React.FormEvent<HTMLFormElement>, optionId: string) => {
    e.preventDefault()
    if (!id) return
    const form = e.currentTarget
    const formData = new FormData(form)
    const quantity = parseFloat(formData.get('quantity') as string) || 1
    const unitPriceCents = Math.round(parseFloat(formData.get('unit_price') as string) * 100) || 0
    const totalCents = Math.round(quantity * unitPriceCents)
    try {
      await quotesApi.createLineItem(id, optionId, {
        type: formData.get('type') as string,
        description: formData.get('description') as string,
        quantity,
        unit: formData.get('unit') as string,
        unit_price_cents: unitPriceCents,
        total_cents: totalCents,
      })
      setAddingItemToOption(null)
      loadQuote()
    } catch (err) {
      console.error('Failed to add line item:', err)
    }
  }

  const handleDeleteLineItem = async (optionId: string, itemId: string) => {
    if (!id) return
    try {
      await quotesApi.deleteLineItem(id, optionId, itemId)
      loadQuote()
    } catch (err) {
      console.error('Failed to delete line item:', err)
    }
  }

  const handleDelete = async () => {
    if (!id || !confirm('Delete this quote?')) return
    try {
      await quotesApi.delete(id)
      navigate('/quotes')
    } catch (err) {
      console.error('Failed to delete quote:', err)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-accent-500" />
      </div>
    )
  }

  if (!quote) {
    return <div className="p-6 text-center text-surface-400">Quote not found</div>
  }

  const config = statusConfig[quote.status]
  const StatusIcon = config.icon
  const isDraft = quote.status === 'draft'
  const isSent = quote.status === 'sent'

  return (
    <div className="p-6 max-w-7xl mx-auto">
      {/* Header */}
      <div className="mb-6">
        <Link to="/quotes" className="inline-flex items-center gap-1.5 text-sm text-surface-400 hover:text-surface-200 mb-4">
          <ArrowLeft className="h-4 w-4" />
          Back to Quotes
        </Link>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <h1 className="text-2xl font-display font-bold text-surface-100">
              <span className="text-surface-500 font-mono mr-2">{quote.quote_number}</span>
              {quote.title}
            </h1>
            <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-xs font-medium ${config.color}`}>
              <StatusIcon className="h-3.5 w-3.5" />
              {config.label}
            </span>
          </div>
        </div>
        {quote.customer && (
          <p className="text-sm text-surface-400 mt-1">
            Customer: <Link to={`/customers/${quote.customer.id}`} className="text-accent-400 hover:text-accent-300">{quote.customer.name}</Link>
            {quote.customer.company && <span className="text-surface-500"> ({quote.customer.company})</span>}
          </p>
        )}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Main Content */}
        <div className="lg:col-span-2 space-y-6">
          {/* Options */}
          <div>
            <div className="flex items-center justify-between mb-3">
              <h2 className="text-lg font-display font-semibold text-surface-100">Options</h2>
              {isDraft && (
                <button
                  onClick={() => setShowAddOption(true)}
                  className="inline-flex items-center gap-1.5 text-sm text-accent-400 hover:text-accent-300"
                >
                  <Plus className="h-4 w-4" />
                  Add Option
                </button>
              )}
            </div>

            {(!quote.options || quote.options.length === 0) ? (
              <div className="bg-surface-900 border border-surface-800 rounded-lg p-8 text-center">
                <p className="text-sm text-surface-500">No options yet. Add an option to start building this quote.</p>
              </div>
            ) : (
              <div className="space-y-4">
                {quote.options.map((option) => (
                  <OptionCard
                    key={option.id}
                    option={option}
                    quoteId={quote.id}
                    isDraft={isDraft}
                    isSent={isSent}
                    isAccepted={quote.accepted_option_id === option.id}
                    addingItem={addingItemToOption === option.id}
                    onAddItem={() => setAddingItemToOption(option.id)}
                    onCancelAddItem={() => setAddingItemToOption(null)}
                    onSubmitItem={(e) => handleAddLineItem(e, option.id)}
                    onDeleteItem={(itemId) => handleDeleteLineItem(option.id, itemId)}
                    onDelete={() => handleDeleteOption(option.id)}
                    onAccept={() => handleAccept(option.id)}
                    actionLoading={actionLoading}
                  />
                ))}
              </div>
            )}
          </div>

          {/* Events Timeline */}
          {quote.events && quote.events.length > 0 && (
            <div>
              <h2 className="text-lg font-display font-semibold text-surface-100 mb-3">Activity</h2>
              <div className="space-y-3">
                {quote.events.map((event) => (
                  <div key={event.id} className="flex gap-3">
                    <div className="w-2 h-2 rounded-full bg-surface-600 mt-1.5 shrink-0" />
                    <div>
                      <p className="text-sm text-surface-300">{event.message || event.event_type}</p>
                      <p className="text-xs text-surface-500">{new Date(event.created_at).toLocaleString()}</p>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* Sidebar */}
        <div className="space-y-4">
          {/* Actions */}
          <div className="bg-surface-900 border border-surface-800 rounded-lg p-4 space-y-3">
            <h3 className="text-sm font-medium text-surface-300">Actions</h3>
            {isDraft && (
              <button
                onClick={handleSend}
                disabled={actionLoading}
                className="w-full inline-flex items-center justify-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 text-sm font-medium disabled:opacity-50"
              >
                <Send className="h-4 w-4" />
                Send to Customer
              </button>
            )}
            {isSent && (
              <button
                onClick={handleReject}
                disabled={actionLoading}
                className="w-full inline-flex items-center justify-center gap-2 px-4 py-2 border border-red-500/50 text-red-400 rounded-lg hover:bg-red-500/10 text-sm font-medium disabled:opacity-50"
              >
                <XCircle className="h-4 w-4" />
                Mark Rejected
              </button>
            )}
            {quote.order_id && (
              <Link
                to={`/orders/${quote.order_id}`}
                className="block w-full text-center px-4 py-2 border border-surface-700 text-surface-300 rounded-lg hover:bg-surface-800 text-sm"
              >
                View Order
              </Link>
            )}
            {isDraft && (
              <button
                onClick={handleDelete}
                className="w-full inline-flex items-center justify-center gap-2 px-4 py-2 text-red-400 hover:bg-red-500/10 rounded-lg text-sm"
              >
                <Trash2 className="h-4 w-4" />
                Delete Quote
              </button>
            )}
          </div>

          {/* Details */}
          <div className="bg-surface-900 border border-surface-800 rounded-lg p-4 space-y-2">
            <h3 className="text-sm font-medium text-surface-300 mb-2">Details</h3>
            <div className="text-xs text-surface-500">Created: {new Date(quote.created_at).toLocaleString()}</div>
            {quote.sent_at && <div className="text-xs text-surface-500">Sent: {new Date(quote.sent_at).toLocaleString()}</div>}
            {quote.accepted_at && <div className="text-xs text-surface-500">Accepted: {new Date(quote.accepted_at).toLocaleString()}</div>}
            {quote.valid_until && <div className="text-xs text-surface-500">Valid until: {new Date(quote.valid_until).toLocaleDateString()}</div>}
            {quote.notes && (
              <div className="pt-2 border-t border-surface-800">
                <p className="text-sm text-surface-400">{quote.notes}</p>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Add Option Modal */}
      {showAddOption && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-surface-900 border border-surface-700 rounded-xl p-6 w-full max-w-md">
            <h2 className="text-lg font-display font-semibold text-surface-100 mb-4">Add Option</h2>
            <form onSubmit={handleAddOption} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-surface-300 mb-1">Name *</label>
                <input name="name" required className="w-full px-3 py-2 bg-surface-800 border border-surface-700 rounded-lg text-surface-100 text-sm focus:outline-none focus:ring-1 focus:ring-accent-500" placeholder="e.g. Standard Package" />
              </div>
              <div>
                <label className="block text-sm font-medium text-surface-300 mb-1">Description</label>
                <textarea name="description" rows={2} className="w-full px-3 py-2 bg-surface-800 border border-surface-700 rounded-lg text-surface-100 text-sm focus:outline-none focus:ring-1 focus:ring-accent-500" />
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button type="button" onClick={() => setShowAddOption(false)} className="px-4 py-2 text-sm text-surface-400 hover:text-surface-200">Cancel</button>
                <button type="submit" className="px-4 py-2 bg-accent-500 text-white rounded-lg hover:bg-accent-600 text-sm font-medium">Add Option</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

interface OptionCardProps {
  option: QuoteOption
  quoteId: string
  isDraft: boolean
  isSent: boolean
  isAccepted: boolean
  addingItem: boolean
  actionLoading: boolean
  onAddItem: () => void
  onCancelAddItem: () => void
  onSubmitItem: (e: React.FormEvent<HTMLFormElement>) => void
  onDeleteItem: (itemId: string) => void
  onDelete: () => void
  onAccept: () => void
}

function OptionCard({ option, isDraft, isSent, isAccepted, addingItem, actionLoading, onAddItem, onCancelAddItem, onSubmitItem, onDeleteItem, onDelete, onAccept }: OptionCardProps) {
  return (
    <div className={`bg-surface-900 border rounded-lg overflow-hidden ${isAccepted ? 'border-green-500/50' : 'border-surface-800'}`}>
      <div className="flex items-center justify-between px-4 py-3 border-b border-surface-800">
        <div>
          <h3 className="text-sm font-medium text-surface-100">
            {option.name}
            {isAccepted && <span className="ml-2 text-xs text-green-400">(Accepted)</span>}
          </h3>
          {option.description && <p className="text-xs text-surface-500 mt-0.5">{option.description}</p>}
        </div>
        <div className="flex items-center gap-3">
          <span className="text-sm font-mono font-medium text-surface-200">{formatCents(option.total_cents)}</span>
          {isDraft && (
            <button onClick={onDelete} className="text-surface-500 hover:text-red-400" title="Delete option">
              <Trash2 className="h-4 w-4" />
            </button>
          )}
        </div>
      </div>

      {/* Line Items */}
      {option.items && option.items.length > 0 && (
        <table className="w-full">
          <thead>
            <tr className="border-b border-surface-800/50">
              <th className="text-left text-xs text-surface-500 px-4 py-2">Type</th>
              <th className="text-left text-xs text-surface-500 px-4 py-2">Description</th>
              <th className="text-right text-xs text-surface-500 px-4 py-2">Qty</th>
              <th className="text-right text-xs text-surface-500 px-4 py-2">Rate</th>
              <th className="text-right text-xs text-surface-500 px-4 py-2">Total</th>
              {isDraft && <th className="w-8" />}
            </tr>
          </thead>
          <tbody className="divide-y divide-surface-800/30">
            {option.items.map((item) => (
              <tr key={item.id} className="hover:bg-surface-800/30">
                <td className="px-4 py-2">
                  <span className="text-xs px-1.5 py-0.5 rounded bg-surface-800 text-surface-400">{item.type}</span>
                </td>
                <td className="px-4 py-2 text-sm text-surface-300">{item.description}</td>
                <td className="px-4 py-2 text-sm text-surface-400 text-right">{item.quantity} {item.unit}</td>
                <td className="px-4 py-2 text-sm text-surface-400 text-right font-mono">{formatCents(item.unit_price_cents)}</td>
                <td className="px-4 py-2 text-sm text-surface-200 text-right font-mono">{formatCents(item.total_cents)}</td>
                {isDraft && (
                  <td className="px-2 py-2">
                    <button onClick={() => onDeleteItem(item.id)} className="text-surface-600 hover:text-red-400">
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </td>
                )}
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {/* Add Line Item Form */}
      {addingItem && (
        <form onSubmit={onSubmitItem} className="px-4 py-3 border-t border-surface-800 bg-surface-800/30">
          <div className="grid grid-cols-6 gap-2">
            <select name="type" required className="col-span-1 px-2 py-1.5 bg-surface-800 border border-surface-700 rounded text-sm text-surface-200 focus:outline-none focus:ring-1 focus:ring-accent-500">
              {lineItemTypes.map((t) => (
                <option key={t.value} value={t.value}>{t.label}</option>
              ))}
            </select>
            <input name="description" required placeholder="Description" className="col-span-2 px-2 py-1.5 bg-surface-800 border border-surface-700 rounded text-sm text-surface-200 focus:outline-none focus:ring-1 focus:ring-accent-500" />
            <input name="quantity" type="number" step="any" defaultValue="1" required className="col-span-1 px-2 py-1.5 bg-surface-800 border border-surface-700 rounded text-sm text-surface-200 text-right focus:outline-none focus:ring-1 focus:ring-accent-500" />
            <select name="unit" className="col-span-1 px-2 py-1.5 bg-surface-800 border border-surface-700 rounded text-sm text-surface-200 focus:outline-none focus:ring-1 focus:ring-accent-500">
              <option value="each">each</option>
              <option value="hours">hours</option>
              <option value="units">units</option>
              <option value="grams">grams</option>
            </select>
            <input name="unit_price" type="number" step="0.01" required placeholder="Rate $" className="col-span-1 px-2 py-1.5 bg-surface-800 border border-surface-700 rounded text-sm text-surface-200 text-right focus:outline-none focus:ring-1 focus:ring-accent-500" />
          </div>
          <div className="flex justify-end gap-2 mt-2">
            <button type="button" onClick={onCancelAddItem} className="px-3 py-1 text-xs text-surface-400 hover:text-surface-200">Cancel</button>
            <button type="submit" className="px-3 py-1 bg-accent-500 text-white rounded text-xs hover:bg-accent-600">Add</button>
          </div>
        </form>
      )}

      {/* Footer Actions */}
      <div className="flex items-center justify-between px-4 py-2 border-t border-surface-800">
        {isDraft && !addingItem && (
          <button onClick={onAddItem} className="inline-flex items-center gap-1 text-xs text-accent-400 hover:text-accent-300">
            <Plus className="h-3.5 w-3.5" />
            Add Item
          </button>
        )}
        {!isDraft && <div />}
        {isSent && (
          <button
            onClick={onAccept}
            disabled={actionLoading}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-green-600 text-white rounded-lg text-xs font-medium hover:bg-green-700 disabled:opacity-50"
          >
            <CheckCircle className="h-3.5 w-3.5" />
            Accept This Option
          </button>
        )}
      </div>
    </div>
  )
}
