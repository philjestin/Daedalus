import { useEffect, useState } from 'react'
import { useParams, Link, useNavigate } from 'react-router-dom'
import {
  ArrowLeft, Send, CheckCircle, XCircle, Plus, Trash2, Clock, Edit3,
  Printer, Package, Truck, Wrench, Paintbrush, MessageSquare, Pencil,
  MoreHorizontal, Copy, MapPin, DollarSign, Link2,
} from 'lucide-react'
import { quotesApi, projectsApi } from '../api/client'
import type { Quote, QuoteStatus, QuoteOption, QuoteLineItem, QuoteLineItemType, Project, Address, DiscountType } from '../types'

// ── Constants ──────────────────────────────────────────────

const statusConfig: Record<QuoteStatus, { label: string; color: string; icon: typeof Clock }> = {
  draft: { label: 'Draft', color: 'bg-surface-700 text-surface-300', icon: Edit3 },
  sent: { label: 'Sent', color: 'bg-blue-500/20 text-blue-400', icon: Send },
  accepted: { label: 'Accepted', color: 'bg-green-500/20 text-green-400', icon: CheckCircle },
  rejected: { label: 'Rejected', color: 'bg-red-500/20 text-red-400', icon: XCircle },
  expired: { label: 'Expired', color: 'bg-surface-700 text-surface-400', icon: Clock },
}

const typeConfig: Record<QuoteLineItemType, { label: string; icon: typeof Clock; defaultUnit: string }> = {
  printing: { label: 'Printing', icon: Printer, defaultUnit: 'each' },
  post_processing: { label: 'Post Processing', icon: Wrench, defaultUnit: 'each' },
  finishing: { label: 'Finishing', icon: Paintbrush, defaultUnit: 'each' },
  labor: { label: 'Labor', icon: Clock, defaultUnit: 'hours' },
  consumables: { label: 'Consumables', icon: Package, defaultUnit: 'each' },
  design: { label: 'Design', icon: Pencil, defaultUnit: 'hours' },
  consulting: { label: 'Consulting', icon: MessageSquare, defaultUnit: 'hours' },
  shipping: { label: 'Shipping', icon: Truck, defaultUnit: 'each' },
  other: { label: 'Other', icon: MoreHorizontal, defaultUnit: 'each' },
}

const typeOrder: QuoteLineItemType[] = [
  'printing', 'post_processing', 'finishing', 'labor', 'consumables', 'design', 'consulting', 'shipping', 'other',
]

const lineItemTypes = typeOrder.map(t => ({ value: t, label: typeConfig[t].label }))

// ── Helpers ────────────────────────────────────────────────

function formatCents(cents: number): string {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(cents / 100)
}

function calculateFinancials(subtotalCents: number, quote: Quote) {
  const discountAmount = quote.discount_type === 'flat'
    ? quote.discount_value
    : quote.discount_type === 'percent'
      ? Math.round(subtotalCents * quote.discount_value / 10000)
      : 0
  const afterDiscount = subtotalCents - discountAmount
  const withRush = afterDiscount + quote.rush_fee_cents
  const taxAmount = Math.round(withRush * quote.tax_rate / 10000)
  const grandTotal = withRush + taxAmount
  return { discountAmount, afterDiscount, withRush, taxAmount, grandTotal }
}

function formatAddressLines(addr?: Address): string[] {
  if (!addr) return []
  const lines: string[] = []
  if (addr.line1) lines.push(addr.line1)
  if (addr.line2) lines.push(addr.line2)
  const cityState = [addr.city, addr.state].filter(Boolean).join(', ')
  if (cityState || addr.zip) lines.push([cityState, addr.zip].filter(Boolean).join(' '))
  if (addr.country) lines.push(addr.country)
  return lines
}

interface PendingLineItem {
  key: number
  project_id: string
  type: string
  description: string
  quantity: string
  unit: string
  unit_price: string
}

function emptyPendingItem(key: number): PendingLineItem {
  return { key, project_id: '', type: 'printing', description: '', quantity: '1', unit: 'each', unit_price: '' }
}

// ── Main Component ─────────────────────────────────────────

export default function QuoteDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [quote, setQuote] = useState<Quote | null>(null)
  const [loading, setLoading] = useState(true)
  const [actionLoading, setActionLoading] = useState(false)
  const [showAddOption, setShowAddOption] = useState(false)
  const [addingItemToOption, setAddingItemToOption] = useState<string | null>(null)
  const [projects, setProjects] = useState<Project[]>([])
  const [pendingItems, setPendingItems] = useState<PendingLineItem[]>([])
  const [editingBillingAddr, setEditingBillingAddr] = useState(false)
  const [editingShippingAddr, setEditingShippingAddr] = useState(false)
  const [copied, setCopied] = useState(false)
  const [showTerms, setShowTerms] = useState(false)

  useEffect(() => {
    projectsApi.list().then(setProjects).catch(() => {})
  }, [])

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

  const handleUpdateQuote = async (updates: Record<string, unknown>) => {
    if (!id) return
    try {
      const updated = await quotesApi.update(id, updates as Partial<Quote>)
      setQuote(updated)
    } catch (err) {
      console.error('Failed to update quote:', err)
    }
  }

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

  const openAddOptionModal = () => {
    setPendingItems([emptyPendingItem(0)])
    setShowAddOption(true)
  }

  const handleAddOption = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    if (!id) return
    const form = e.currentTarget
    const formData = new FormData(form)
    try {
      const option = await quotesApi.createOption(id, {
        name: formData.get('name') as string,
        description: formData.get('description') as string || undefined,
      })
      for (const item of pendingItems) {
        if (!item.description) continue
        const quantity = parseFloat(item.quantity) || 1
        const unitPriceCents = Math.round(parseFloat(item.unit_price) * 100) || 0
        await quotesApi.createLineItem(id, option.id, {
          type: item.type,
          description: item.description,
          quantity,
          unit: item.unit,
          unit_price_cents: unitPriceCents,
          total_cents: Math.round(quantity * unitPriceCents),
          project_id: item.project_id || undefined,
        })
      }
      setShowAddOption(false)
      setPendingItems([])
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
    try {
      await quotesApi.createLineItem(id, optionId, {
        type: formData.get('type') as string,
        description: formData.get('description') as string,
        quantity,
        unit: formData.get('unit') as string,
        unit_price_cents: unitPriceCents,
        total_cents: Math.round(quantity * unitPriceCents),
        project_id: formData.get('project_id') as string || undefined,
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

  const handleSaveAddress = async (type: 'billing' | 'shipping', addr: Address) => {
    const key = type === 'billing' ? 'billing_address' : 'shipping_address'
    await handleUpdateQuote({ [key]: addr })
    if (type === 'billing') setEditingBillingAddr(false)
    else setEditingShippingAddr(false)
  }

  const handleCopyShareLink = () => {
    if (!quote?.share_token) return
    const url = `${window.location.origin}/quote/${quote.share_token}`
    navigator.clipboard.writeText(url).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    })
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

  // Grand total from the accepted option or first option
  const primaryOption = quote.options?.find(o => o.id === quote.accepted_option_id) || quote.options?.[0]
  const primaryFinancials = primaryOption ? calculateFinancials(primaryOption.total_cents, quote) : null

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
            {primaryFinancials && (
              <span className="text-lg font-mono font-semibold text-surface-200">
                {formatCents(primaryFinancials.grandTotal)}
              </span>
            )}
          </div>
        </div>
        {quote.customer && (
          <p className="text-sm text-surface-400 mt-1">
            Customer: <Link to={`/customers/${quote.customer.id}`} className="text-accent-400 hover:text-accent-300">{quote.customer.name}</Link>
            {quote.customer.company && <span className="text-surface-500"> ({quote.customer.company})</span>}
          </p>
        )}

        {/* Option comparison bar */}
        {quote.options && quote.options.length >= 2 && (
          <div className="flex items-center gap-3 mt-3">
            {quote.options.map(opt => {
              const fin = calculateFinancials(opt.total_cents, quote)
              const isAccepted = quote.accepted_option_id === opt.id
              return (
                <div key={opt.id} className={`px-3 py-1.5 rounded-lg text-xs font-medium border ${isAccepted ? 'border-green-500/50 bg-green-500/10 text-green-400' : 'border-surface-700 bg-surface-800/50 text-surface-300'}`}>
                  {opt.name}: <span className="font-mono">{formatCents(fin.grandTotal)}</span>
                </div>
              )
            })}
          </div>
        )}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Main Content */}
        <div className="lg:col-span-2 space-y-6">

          {/* Addresses */}
          {(quote.billing_address || quote.shipping_address || isDraft) && (
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <AddressCard
                label="Bill To"
                address={quote.billing_address}
                editing={editingBillingAddr}
                editable={isDraft}
                onEdit={() => setEditingBillingAddr(true)}
                onCancel={() => setEditingBillingAddr(false)}
                onSave={(addr) => handleSaveAddress('billing', addr)}
                customerAddress={quote.customer?.billing_address}
                onCopyFromCustomer={() => {
                  if (quote.customer?.billing_address) handleSaveAddress('billing', quote.customer.billing_address)
                }}
              />
              <AddressCard
                label="Ship To"
                address={quote.shipping_address}
                editing={editingShippingAddr}
                editable={isDraft}
                onEdit={() => setEditingShippingAddr(true)}
                onCancel={() => setEditingShippingAddr(false)}
                onSave={(addr) => handleSaveAddress('shipping', addr)}
                customerAddress={quote.customer?.shipping_address}
                onCopyFromCustomer={() => {
                  if (quote.customer?.shipping_address) handleSaveAddress('shipping', quote.customer.shipping_address)
                }}
              />
            </div>
          )}

          {/* Options */}
          <div>
            <div className="flex items-center justify-between mb-3">
              <h2 className="text-lg font-display font-semibold text-surface-100">Options</h2>
              {isDraft && (
                <button onClick={openAddOptionModal} className="inline-flex items-center gap-1.5 text-sm text-accent-400 hover:text-accent-300">
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
                    quote={quote}
                    isDraft={isDraft}
                    isSent={isSent}
                    isAccepted={quote.accepted_option_id === option.id}
                    addingItem={addingItemToOption === option.id}
                    projects={projects}
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
              <button onClick={handleSend} disabled={actionLoading} className="w-full inline-flex items-center justify-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 text-sm font-medium disabled:opacity-50">
                <Send className="h-4 w-4" />
                Mark as Sent
              </button>
            )}
            {isSent && (
              <button onClick={handleReject} disabled={actionLoading} className="w-full inline-flex items-center justify-center gap-2 px-4 py-2 border border-red-500/50 text-red-400 rounded-lg hover:bg-red-500/10 text-sm font-medium disabled:opacity-50">
                <XCircle className="h-4 w-4" />
                Mark Rejected
              </button>
            )}
            {quote.order_id && (
              <Link to={`/orders/${quote.order_id}`} className="block w-full text-center px-4 py-2 border border-surface-700 text-surface-300 rounded-lg hover:bg-surface-800 text-sm">
                View Order
              </Link>
            )}
            {isDraft && (
              <button onClick={handleDelete} className="w-full inline-flex items-center justify-center gap-2 px-4 py-2 text-red-400 hover:bg-red-500/10 rounded-lg text-sm">
                <Trash2 className="h-4 w-4" />
                Delete Quote
              </button>
            )}
          </div>

          {/* Financial Adjustments */}
          <div className="bg-surface-900 border border-surface-800 rounded-lg p-4 space-y-3">
            <h3 className="text-sm font-medium text-surface-300 flex items-center gap-1.5">
              <DollarSign className="h-3.5 w-3.5" />
              Financial Adjustments
            </h3>

            {/* Discount */}
            <div>
              <label className="block text-xs text-surface-500 mb-1">Discount</label>
              <div className="flex gap-2">
                <select
                  value={quote.discount_type}
                  onChange={(e) => handleUpdateQuote({
                    discount_type: e.target.value as DiscountType,
                    discount_value: e.target.value === 'none' ? 0 : quote.discount_value,
                  })}
                  disabled={!isDraft}
                  className="px-2 py-1.5 bg-surface-800 border border-surface-700 rounded text-xs text-surface-200 focus:outline-none focus:ring-1 focus:ring-accent-500 disabled:opacity-50"
                >
                  <option value="none">None</option>
                  <option value="flat">Flat ($)</option>
                  <option value="percent">Percent (%)</option>
                </select>
                {quote.discount_type !== 'none' && (
                  <input
                    type="number"
                    step="0.01"
                    defaultValue={(quote.discount_value / 100).toFixed(2)}
                    key={`discount-${quote.discount_type}-${quote.discount_value}`}
                    onBlur={(e) => {
                      const val = parseFloat(e.target.value) || 0
                      handleUpdateQuote({ discount_value: Math.round(val * 100) })
                    }}
                    disabled={!isDraft}
                    className="flex-1 px-2 py-1.5 bg-surface-800 border border-surface-700 rounded text-xs text-surface-200 text-right focus:outline-none focus:ring-1 focus:ring-accent-500 disabled:opacity-50"
                    placeholder={quote.discount_type === 'flat' ? '$0.00' : '0.00%'}
                  />
                )}
              </div>
            </div>

            {/* Rush Fee */}
            <div>
              <label className="block text-xs text-surface-500 mb-1">Rush Fee</label>
              <div className="relative">
                <span className="absolute left-2 top-1/2 -translate-y-1/2 text-xs text-surface-500">$</span>
                <input
                  type="number"
                  step="0.01"
                  defaultValue={(quote.rush_fee_cents / 100).toFixed(2)}
                  key={`rush-${quote.rush_fee_cents}`}
                  onBlur={(e) => handleUpdateQuote({ rush_fee_cents: Math.round((parseFloat(e.target.value) || 0) * 100) })}
                  disabled={!isDraft}
                  className="w-full pl-5 pr-2 py-1.5 bg-surface-800 border border-surface-700 rounded text-xs text-surface-200 text-right focus:outline-none focus:ring-1 focus:ring-accent-500 disabled:opacity-50"
                />
              </div>
            </div>

            {/* Tax Rate */}
            <div>
              <label className="block text-xs text-surface-500 mb-1">Tax Rate</label>
              <div className="relative">
                <input
                  type="number"
                  step="0.01"
                  defaultValue={(quote.tax_rate / 100).toFixed(2)}
                  key={`tax-${quote.tax_rate}`}
                  onBlur={(e) => handleUpdateQuote({ tax_rate: Math.round((parseFloat(e.target.value) || 0) * 100) })}
                  disabled={!isDraft}
                  className="w-full pl-2 pr-5 py-1.5 bg-surface-800 border border-surface-700 rounded text-xs text-surface-200 text-right focus:outline-none focus:ring-1 focus:ring-accent-500 disabled:opacity-50"
                />
                <span className="absolute right-2 top-1/2 -translate-y-1/2 text-xs text-surface-500">%</span>
              </div>
            </div>
          </div>

          {/* Details */}
          <div className="bg-surface-900 border border-surface-800 rounded-lg p-4 space-y-2">
            <h3 className="text-sm font-medium text-surface-300 mb-2">Details</h3>
            <div className="text-xs text-surface-500">Created: {new Date(quote.created_at).toLocaleString()}</div>
            {quote.sent_at && <div className="text-xs text-surface-500">Sent: {new Date(quote.sent_at).toLocaleString()}</div>}
            {quote.accepted_at && <div className="text-xs text-surface-500">Accepted: {new Date(quote.accepted_at).toLocaleString()}</div>}

            {/* Valid Until */}
            <div>
              <label className="block text-xs text-surface-500 mb-1 mt-2">Valid Until</label>
              <input
                type="date"
                defaultValue={quote.valid_until ? quote.valid_until.split('T')[0] : ''}
                key={`valid-${quote.valid_until}`}
                onBlur={(e) => {
                  if (e.target.value) handleUpdateQuote({ valid_until: e.target.value + 'T00:00:00Z' })
                }}
                disabled={!isDraft}
                className="w-full px-2 py-1.5 bg-surface-800 border border-surface-700 rounded text-xs text-surface-200 focus:outline-none focus:ring-1 focus:ring-accent-500 disabled:opacity-50"
              />
            </div>

            {/* Requested Due Date */}
            <div>
              <label className="block text-xs text-surface-500 mb-1">Requested Due Date</label>
              <input
                type="date"
                defaultValue={quote.requested_due_date ? quote.requested_due_date.split('T')[0] : ''}
                key={`due-${quote.requested_due_date}`}
                onBlur={(e) => {
                  if (e.target.value) handleUpdateQuote({ requested_due_date: e.target.value + 'T00:00:00Z' })
                }}
                disabled={!isDraft}
                className="w-full px-2 py-1.5 bg-surface-800 border border-surface-700 rounded text-xs text-surface-200 focus:outline-none focus:ring-1 focus:ring-accent-500 disabled:opacity-50"
              />
            </div>

            {/* Terms & Conditions */}
            <div className="pt-2 border-t border-surface-800">
              <button
                onClick={() => setShowTerms(!showTerms)}
                className="text-xs text-surface-400 hover:text-surface-200 w-full text-left"
              >
                Terms & Conditions {showTerms ? '\u25BE' : '\u25B8'}
              </button>
              {showTerms && (
                <textarea
                  defaultValue={quote.terms || ''}
                  key={`terms-${quote.id}`}
                  onBlur={(e) => handleUpdateQuote({ terms: e.target.value || undefined })}
                  disabled={!isDraft}
                  rows={4}
                  placeholder="Enter terms and conditions..."
                  className="w-full mt-1 px-2 py-1.5 bg-surface-800 border border-surface-700 rounded text-xs text-surface-200 focus:outline-none focus:ring-1 focus:ring-accent-500 disabled:opacity-50"
                />
              )}
            </div>

            {/* Share Link */}
            {quote.share_token && quote.status !== 'draft' && (
              <div className="pt-2 border-t border-surface-800">
                <label className="block text-xs text-surface-500 mb-1">
                  <Link2 className="h-3 w-3 inline mr-1" />
                  Share Link
                </label>
                <div className="flex gap-1">
                  <input
                    readOnly
                    value={`${window.location.origin}/quote/${quote.share_token}`}
                    className="flex-1 px-2 py-1 bg-surface-800 border border-surface-700 rounded text-xs text-surface-400 truncate"
                  />
                  <button
                    onClick={handleCopyShareLink}
                    className="px-2 py-1 bg-surface-800 border border-surface-700 rounded text-xs text-surface-400 hover:text-surface-200 shrink-0"
                    title="Copy link"
                  >
                    {copied ? <CheckCircle className="h-3.5 w-3.5 text-green-400" /> : <Copy className="h-3.5 w-3.5" />}
                  </button>
                </div>
                {copied && <p className="text-xs text-green-400 mt-0.5">Copied!</p>}
              </div>
            )}

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
          <div className="bg-surface-900 border border-surface-700 rounded-xl p-6 w-full max-w-2xl max-h-[90vh] overflow-y-auto">
            <h2 className="text-lg font-display font-semibold text-surface-100 mb-4">Add Option</h2>
            <form onSubmit={handleAddOption} className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-sm font-medium text-surface-300 mb-1">Name *</label>
                  <input name="name" required className="w-full px-3 py-2 bg-surface-800 border border-surface-700 rounded-lg text-surface-100 text-sm focus:outline-none focus:ring-1 focus:ring-accent-500" placeholder="e.g. Standard Package" />
                </div>
                <div>
                  <label className="block text-sm font-medium text-surface-300 mb-1">Description</label>
                  <input name="description" className="w-full px-3 py-2 bg-surface-800 border border-surface-700 rounded-lg text-surface-100 text-sm focus:outline-none focus:ring-1 focus:ring-accent-500" />
                </div>
              </div>

              {/* Line Items */}
              <div>
                <div className="flex items-center justify-between mb-2">
                  <label className="block text-sm font-medium text-surface-300">Line Items</label>
                  <button type="button" onClick={() => setPendingItems(prev => [...prev, emptyPendingItem(Date.now())])} className="inline-flex items-center gap-1 text-xs text-accent-400 hover:text-accent-300">
                    <Plus className="h-3.5 w-3.5" />
                    Add Item
                  </button>
                </div>
                <div className="space-y-3">
                  {pendingItems.map((item, idx) => (
                    <div key={item.key} className="bg-surface-800/50 border border-surface-700 rounded-lg p-3 space-y-2">
                      <div className="flex items-center gap-2">
                        <select value={item.project_id} onChange={(e) => {
                          const project = projects.find(p => p.id === e.target.value)
                          setPendingItems(prev => prev.map((it, i) => i === idx ? {
                            ...it,
                            project_id: e.target.value,
                            description: it.description || (project?.name ?? ''),
                            unit_price: it.unit_price || (project?.price_cents ? (project.price_cents / 100).toFixed(2) : ''),
                          } : it))
                        }} className="flex-1 px-2 py-1.5 bg-surface-800 border border-surface-700 rounded text-sm text-surface-200 focus:outline-none focus:ring-1 focus:ring-accent-500">
                          <option value="">No project (free text)</option>
                          {projects.map((p) => (
                            <option key={p.id} value={p.id}>{p.name}{p.sku ? ` (${p.sku})` : ''}</option>
                          ))}
                        </select>
                        {pendingItems.length > 1 && (
                          <button type="button" onClick={() => setPendingItems(prev => prev.filter((_, i) => i !== idx))} className="text-surface-500 hover:text-red-400">
                            <Trash2 className="h-3.5 w-3.5" />
                          </button>
                        )}
                      </div>
                      <div className="grid grid-cols-6 gap-2">
                        <select value={item.type} onChange={(e) => {
                          const newType = e.target.value as QuoteLineItemType
                          const defaultUnit = typeConfig[newType]?.defaultUnit || 'each'
                          setPendingItems(prev => prev.map((it, i) => i === idx ? { ...it, type: newType, unit: defaultUnit } : it))
                        }} className="col-span-1 px-2 py-1.5 bg-surface-800 border border-surface-700 rounded text-sm text-surface-200 focus:outline-none focus:ring-1 focus:ring-accent-500">
                          {lineItemTypes.map((t) => (
                            <option key={t.value} value={t.value}>{t.label}</option>
                          ))}
                        </select>
                        <input value={item.description} onChange={(e) => setPendingItems(prev => prev.map((it, i) => i === idx ? { ...it, description: e.target.value } : it))} placeholder="Description *" className="col-span-2 px-2 py-1.5 bg-surface-800 border border-surface-700 rounded text-sm text-surface-200 focus:outline-none focus:ring-1 focus:ring-accent-500" />
                        <input value={item.quantity} onChange={(e) => setPendingItems(prev => prev.map((it, i) => i === idx ? { ...it, quantity: e.target.value } : it))} type="number" step="any" placeholder="Qty" className="col-span-1 px-2 py-1.5 bg-surface-800 border border-surface-700 rounded text-sm text-surface-200 text-right focus:outline-none focus:ring-1 focus:ring-accent-500" />
                        <select value={item.unit} onChange={(e) => setPendingItems(prev => prev.map((it, i) => i === idx ? { ...it, unit: e.target.value } : it))} className="col-span-1 px-2 py-1.5 bg-surface-800 border border-surface-700 rounded text-sm text-surface-200 focus:outline-none focus:ring-1 focus:ring-accent-500">
                          <option value="each">each</option>
                          <option value="hours">hours</option>
                          <option value="units">units</option>
                          <option value="grams">grams</option>
                        </select>
                        <input value={item.unit_price} onChange={(e) => setPendingItems(prev => prev.map((it, i) => i === idx ? { ...it, unit_price: e.target.value } : it))} type="number" step="0.01" placeholder="Rate $" className="col-span-1 px-2 py-1.5 bg-surface-800 border border-surface-700 rounded text-sm text-surface-200 text-right focus:outline-none focus:ring-1 focus:ring-accent-500" />
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              <div className="flex justify-end gap-3 pt-2 border-t border-surface-800">
                <button type="button" onClick={() => { setShowAddOption(false); setPendingItems([]) }} className="px-4 py-2 text-sm text-surface-400 hover:text-surface-200">Cancel</button>
                <button type="submit" className="px-4 py-2 bg-accent-500 text-white rounded-lg hover:bg-accent-600 text-sm font-medium">Add Option</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

// ── AddressCard Component ──────────────────────────────────

interface AddressCardProps {
  label: string
  address?: Address
  editing: boolean
  editable: boolean
  onEdit: () => void
  onCancel: () => void
  onSave: (addr: Address) => void
  customerAddress?: Address
  onCopyFromCustomer: () => void
}

function AddressCard({ label, address, editing, editable, onEdit, onCancel, onSave, customerAddress, onCopyFromCustomer }: AddressCardProps) {
  const [form, setForm] = useState<Address>(address || {})

  useEffect(() => {
    setForm(address || {})
  }, [address, editing])

  if (editing) {
    return (
      <div className="bg-surface-900 border border-surface-800 rounded-lg p-4">
        <div className="flex items-center justify-between mb-2">
          <h4 className="text-xs font-medium text-surface-400 uppercase">{label}</h4>
          {customerAddress && (
            <button onClick={onCopyFromCustomer} className="text-xs text-accent-400 hover:text-accent-300">Copy from customer</button>
          )}
        </div>
        <div className="space-y-2">
          <input value={form.line1 || ''} onChange={e => setForm(f => ({ ...f, line1: e.target.value }))} placeholder="Address line 1" className="w-full px-2 py-1.5 bg-surface-800 border border-surface-700 rounded text-xs text-surface-200 focus:outline-none focus:ring-1 focus:ring-accent-500" />
          <input value={form.line2 || ''} onChange={e => setForm(f => ({ ...f, line2: e.target.value }))} placeholder="Address line 2" className="w-full px-2 py-1.5 bg-surface-800 border border-surface-700 rounded text-xs text-surface-200 focus:outline-none focus:ring-1 focus:ring-accent-500" />
          <div className="grid grid-cols-3 gap-2">
            <input value={form.city || ''} onChange={e => setForm(f => ({ ...f, city: e.target.value }))} placeholder="City" className="px-2 py-1.5 bg-surface-800 border border-surface-700 rounded text-xs text-surface-200 focus:outline-none focus:ring-1 focus:ring-accent-500" />
            <input value={form.state || ''} onChange={e => setForm(f => ({ ...f, state: e.target.value }))} placeholder="State" className="px-2 py-1.5 bg-surface-800 border border-surface-700 rounded text-xs text-surface-200 focus:outline-none focus:ring-1 focus:ring-accent-500" />
            <input value={form.zip || ''} onChange={e => setForm(f => ({ ...f, zip: e.target.value }))} placeholder="Zip" className="px-2 py-1.5 bg-surface-800 border border-surface-700 rounded text-xs text-surface-200 focus:outline-none focus:ring-1 focus:ring-accent-500" />
          </div>
          <input value={form.country || ''} onChange={e => setForm(f => ({ ...f, country: e.target.value }))} placeholder="Country" className="w-full px-2 py-1.5 bg-surface-800 border border-surface-700 rounded text-xs text-surface-200 focus:outline-none focus:ring-1 focus:ring-accent-500" />
        </div>
        <div className="flex justify-end gap-2 mt-2">
          <button onClick={onCancel} className="px-3 py-1 text-xs text-surface-400 hover:text-surface-200">Cancel</button>
          <button onClick={() => onSave(form)} className="px-3 py-1 bg-accent-500 text-white rounded text-xs hover:bg-accent-600">Save</button>
        </div>
      </div>
    )
  }

  const lines = formatAddressLines(address)
  return (
    <div className="bg-surface-900 border border-surface-800 rounded-lg p-4">
      <div className="flex items-center justify-between mb-1">
        <h4 className="text-xs font-medium text-surface-400 uppercase flex items-center gap-1">
          <MapPin className="h-3 w-3" />
          {label}
        </h4>
        {editable && (
          <button onClick={onEdit} className="text-xs text-accent-400 hover:text-accent-300">Edit</button>
        )}
      </div>
      {lines.length > 0 ? (
        <div className="text-sm text-surface-300 space-y-0.5">
          {lines.map((line, i) => <div key={i}>{line}</div>)}
        </div>
      ) : (
        <p className="text-xs text-surface-500 italic">No address set</p>
      )}
    </div>
  )
}

// ── OptionCard Component ───────────────────────────────────

interface OptionCardProps {
  option: QuoteOption
  quote: Quote
  isDraft: boolean
  isSent: boolean
  isAccepted: boolean
  addingItem: boolean
  projects: Project[]
  actionLoading: boolean
  onAddItem: () => void
  onCancelAddItem: () => void
  onSubmitItem: (e: React.FormEvent<HTMLFormElement>) => void
  onDeleteItem: (itemId: string) => void
  onDelete: () => void
  onAccept: () => void
}

function OptionCard({ option, quote, isDraft, isSent, isAccepted, addingItem, projects, actionLoading, onAddItem, onCancelAddItem, onSubmitItem, onDeleteItem, onDelete, onAccept }: OptionCardProps) {
  const projectMap = new Map(projects.map(p => [p.id, p]))
  const financials = calculateFinancials(option.total_cents, quote)
  const hasAdjustments = quote.discount_type !== 'none' || quote.rush_fee_cents > 0 || quote.tax_rate > 0

  // Group items by type
  const groupedItems = new Map<QuoteLineItemType, QuoteLineItem[]>()
  if (option.items) {
    for (const item of option.items) {
      const type = item.type as QuoteLineItemType
      if (!groupedItems.has(type)) groupedItems.set(type, [])
      groupedItems.get(type)!.push(item)
    }
  }
  const orderedGroups = typeOrder
    .filter(t => groupedItems.has(t))
    .map(t => ({ type: t, items: groupedItems.get(t)! }))

  return (
    <div className={`bg-surface-900 border rounded-lg overflow-hidden ${isAccepted ? 'border-green-500/50' : 'border-surface-800'}`}>
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-surface-800">
        <div>
          <h3 className="text-sm font-medium text-surface-100">
            {option.name}
            {isAccepted && <span className="ml-2 text-xs text-green-400">(Accepted)</span>}
          </h3>
          {option.description && <p className="text-xs text-surface-500 mt-0.5">{option.description}</p>}
        </div>
        <div className="flex items-center gap-3">
          <span className="text-sm font-mono font-medium text-surface-200">
            {hasAdjustments ? formatCents(financials.grandTotal) : formatCents(option.total_cents)}
          </span>
          {isDraft && (
            <button onClick={onDelete} className="text-surface-500 hover:text-red-400" title="Delete option">
              <Trash2 className="h-4 w-4" />
            </button>
          )}
        </div>
      </div>

      {/* Grouped Line Items */}
      {orderedGroups.length > 0 && (
        <div className="divide-y divide-surface-800/30">
          {orderedGroups.map(({ type, items }) => {
            const cfg = typeConfig[type]
            const TypeIcon = cfg.icon
            const groupTotal = items.reduce((sum, it) => sum + it.total_cents, 0)
            return (
              <div key={type}>
                <div className="flex items-center justify-between px-4 py-2 bg-surface-800/20">
                  <div className="flex items-center gap-1.5 text-xs font-medium text-surface-400 uppercase tracking-wide">
                    <TypeIcon className="h-3.5 w-3.5" />
                    {cfg.label}
                    <span className="text-surface-600 font-normal">({items.length})</span>
                  </div>
                  <span className="text-xs font-mono text-surface-400">{formatCents(groupTotal)}</span>
                </div>
                <table className="w-full">
                  <tbody>
                    {items.map((item) => (
                      <tr key={item.id} className="hover:bg-surface-800/30">
                        <td className="pl-8 pr-4 py-1.5 text-sm text-surface-300">
                          {item.description}
                          {item.project_id && projectMap.get(item.project_id) && (
                            <Link to={`/projects/${item.project_id}`} className="ml-1.5 text-xs text-accent-400 hover:text-accent-300">
                              ({projectMap.get(item.project_id)!.name})
                            </Link>
                          )}
                        </td>
                        <td className="px-4 py-1.5 text-xs text-surface-400 text-right whitespace-nowrap">{item.quantity} {item.unit}</td>
                        <td className="px-4 py-1.5 text-xs text-surface-400 text-right font-mono whitespace-nowrap">@{formatCents(item.unit_price_cents)}</td>
                        <td className="px-4 py-1.5 text-sm text-surface-200 text-right font-mono whitespace-nowrap">{formatCents(item.total_cents)}</td>
                        {isDraft && (
                          <td className="px-2 py-1.5 w-8">
                            <button onClick={() => onDeleteItem(item.id)} className="text-surface-600 hover:text-red-400">
                              <Trash2 className="h-3.5 w-3.5" />
                            </button>
                          </td>
                        )}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )
          })}
        </div>
      )}

      {/* Financial Summary */}
      {option.items && option.items.length > 0 && (
        <div className="border-t border-surface-700 px-4 py-3 space-y-1">
          <div className="flex justify-between text-xs text-surface-400">
            <span>Subtotal</span>
            <span className="font-mono">{formatCents(option.total_cents)}</span>
          </div>
          {quote.discount_type !== 'none' && financials.discountAmount > 0 && (
            <div className="flex justify-between text-xs text-surface-400">
              <span>Discount {quote.discount_type === 'percent' ? `(${(quote.discount_value / 100).toFixed(2)}%)` : '(flat)'}</span>
              <span className="font-mono text-red-400">-{formatCents(financials.discountAmount)}</span>
            </div>
          )}
          {quote.rush_fee_cents > 0 && (
            <div className="flex justify-between text-xs text-surface-400">
              <span>Rush Fee</span>
              <span className="font-mono">{formatCents(quote.rush_fee_cents)}</span>
            </div>
          )}
          {quote.tax_rate > 0 && (
            <div className="flex justify-between text-xs text-surface-400">
              <span>Tax ({(quote.tax_rate / 100).toFixed(2)}%)</span>
              <span className="font-mono">{formatCents(financials.taxAmount)}</span>
            </div>
          )}
          {hasAdjustments && (
            <div className="flex justify-between text-sm font-medium text-surface-100 pt-1 border-t border-surface-800">
              <span>Total</span>
              <span className="font-mono">{formatCents(financials.grandTotal)}</span>
            </div>
          )}
        </div>
      )}

      {/* Add Line Item Form */}
      {addingItem && (
        <form onSubmit={onSubmitItem} className="px-4 py-3 border-t border-surface-800 bg-surface-800/30">
          <div className="mb-2">
            <label className="block text-xs font-medium text-surface-400 mb-1">Project (optional)</label>
            <select name="project_id" className="w-full px-2 py-1.5 bg-surface-800 border border-surface-700 rounded text-sm text-surface-200 focus:outline-none focus:ring-1 focus:ring-accent-500" onChange={(e) => {
              const form = e.target.form
              if (!form) return
              const project = projects.find(p => p.id === e.target.value)
              if (project) {
                const descInput = form.elements.namedItem('description') as HTMLInputElement
                if (descInput && !descInput.value) descInput.value = project.name
                if (project.price_cents) {
                  const priceInput = form.elements.namedItem('unit_price') as HTMLInputElement
                  if (priceInput && !priceInput.value) priceInput.value = (project.price_cents / 100).toFixed(2)
                }
              }
            }}>
              <option value="">No project (free text)</option>
              {projects.map((p) => (
                <option key={p.id} value={p.id}>{p.name}{p.sku ? ` (${p.sku})` : ''}</option>
              ))}
            </select>
          </div>
          <div className="grid grid-cols-6 gap-2">
            <select name="type" required className="col-span-1 px-2 py-1.5 bg-surface-800 border border-surface-700 rounded text-sm text-surface-200 focus:outline-none focus:ring-1 focus:ring-accent-500" onChange={(e) => {
              const form = e.target.form
              if (!form) return
              const newType = e.target.value as QuoteLineItemType
              const unitSelect = form.elements.namedItem('unit') as HTMLSelectElement
              if (unitSelect) unitSelect.value = typeConfig[newType]?.defaultUnit || 'each'
            }}>
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

      {/* Footer */}
      <div className="flex items-center justify-between px-4 py-2 border-t border-surface-800">
        {isDraft && !addingItem && (
          <button onClick={onAddItem} className="inline-flex items-center gap-1 text-xs text-accent-400 hover:text-accent-300">
            <Plus className="h-3.5 w-3.5" />
            Add Item
          </button>
        )}
        {!isDraft && <div />}
        {isSent && (
          <button onClick={onAccept} disabled={actionLoading} className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-green-600 text-white rounded-lg text-xs font-medium hover:bg-green-700 disabled:opacity-50">
            <CheckCircle className="h-3.5 w-3.5" />
            Accept This Option
          </button>
        )}
      </div>
    </div>
  )
}
