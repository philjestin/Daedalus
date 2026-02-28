import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { FileText, Plus, ChevronRight, Send, CheckCircle, XCircle, Clock, Edit3 } from 'lucide-react'
import { quotesApi, customersApi } from '../api/client'
import type { Quote, QuoteStatus, Customer } from '../types'

const statusConfig: Record<QuoteStatus, { label: string; color: string; icon: typeof Clock }> = {
  draft: { label: 'Draft', color: 'bg-surface-700 text-surface-300', icon: Edit3 },
  sent: { label: 'Sent', color: 'bg-blue-500/20 text-blue-400', icon: Send },
  accepted: { label: 'Accepted', color: 'bg-green-500/20 text-green-400', icon: CheckCircle },
  rejected: { label: 'Rejected', color: 'bg-red-500/20 text-red-400', icon: XCircle },
  expired: { label: 'Expired', color: 'bg-surface-700 text-surface-400', icon: Clock },
}

function formatCents(cents: number): string {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(cents / 100)
}

export default function Quotes() {
  const [quotes, setQuotes] = useState<Quote[]>([])
  const [customers, setCustomers] = useState<Customer[]>([])
  const [loading, setLoading] = useState(true)
  const [statusFilter, setStatusFilter] = useState<string>('')
  const [showCreateModal, setShowCreateModal] = useState(false)

  const loadData = async () => {
    try {
      const [quotesData, customersData] = await Promise.all([
        quotesApi.list({ status: statusFilter || undefined }),
        customersApi.list(),
      ])
      setQuotes(quotesData)
      setCustomers(customersData)
    } catch (err) {
      console.error('Failed to load quotes:', err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadData()
  }, [statusFilter])

  const handleCreate = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    const form = e.currentTarget
    const formData = new FormData(form)
    try {
      await quotesApi.create({
        customer_id: formData.get('customer_id') as string,
        title: formData.get('title') as string,
        notes: formData.get('notes') as string || undefined,
      })
      setShowCreateModal(false)
      loadData()
    } catch (err) {
      console.error('Failed to create quote:', err)
    }
  }

  const getMaxOptionTotal = (quote: Quote): number => {
    if (!quote.options || quote.options.length === 0) return 0
    return Math.max(...quote.options.map(o => o.total_cents))
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-accent-500" />
      </div>
    )
  }

  return (
    <div className="p-6 max-w-7xl mx-auto">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-display font-bold text-surface-100">Quotes</h1>
          <p className="text-sm text-surface-400 mt-1">{quotes.length} quote{quotes.length !== 1 ? 's' : ''}</p>
        </div>
        <button
          onClick={() => setShowCreateModal(true)}
          className="inline-flex items-center gap-2 px-4 py-2 bg-accent-500 text-white rounded-lg hover:bg-accent-600 transition-colors text-sm font-medium"
        >
          <Plus className="h-4 w-4" />
          New Quote
        </button>
      </div>

      {/* Filters */}
      <div className="mb-4">
        <select
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="px-3 py-2 bg-surface-800 border border-surface-700 rounded-lg text-surface-200 text-sm focus:outline-none focus:ring-1 focus:ring-accent-500"
        >
          <option value="">All Statuses</option>
          <option value="draft">Draft</option>
          <option value="sent">Sent</option>
          <option value="accepted">Accepted</option>
          <option value="rejected">Rejected</option>
          <option value="expired">Expired</option>
        </select>
      </div>

      {/* Table */}
      {quotes.length === 0 ? (
        <div className="text-center py-16">
          <FileText className="h-12 w-12 text-surface-600 mx-auto mb-4" />
          <h3 className="text-lg font-medium text-surface-300 mb-2">No quotes yet</h3>
          <p className="text-surface-500 text-sm mb-4">Create your first quote to start tracking custom jobs.</p>
          <button
            onClick={() => setShowCreateModal(true)}
            className="inline-flex items-center gap-2 px-4 py-2 bg-accent-500 text-white rounded-lg hover:bg-accent-600 transition-colors text-sm"
          >
            <Plus className="h-4 w-4" />
            New Quote
          </button>
        </div>
      ) : (
        <div className="bg-surface-900 border border-surface-800 rounded-lg overflow-hidden">
          <table className="w-full">
            <thead>
              <tr className="border-b border-surface-800">
                <th className="text-left text-xs font-medium text-surface-400 uppercase tracking-wider px-4 py-3">Quote #</th>
                <th className="text-left text-xs font-medium text-surface-400 uppercase tracking-wider px-4 py-3">Title</th>
                <th className="text-left text-xs font-medium text-surface-400 uppercase tracking-wider px-4 py-3">Customer</th>
                <th className="text-left text-xs font-medium text-surface-400 uppercase tracking-wider px-4 py-3">Status</th>
                <th className="text-right text-xs font-medium text-surface-400 uppercase tracking-wider px-4 py-3">Total</th>
                <th className="text-left text-xs font-medium text-surface-400 uppercase tracking-wider px-4 py-3">Date</th>
                <th className="w-10" />
              </tr>
            </thead>
            <tbody className="divide-y divide-surface-800">
              {quotes.map((quote) => {
                const config = statusConfig[quote.status]
                const StatusIcon = config.icon
                return (
                  <tr key={quote.id} className="hover:bg-surface-800/50 transition-colors">
                    <td className="px-4 py-3">
                      <Link to={`/quotes/${quote.id}`} className="text-sm font-mono text-accent-400 hover:text-accent-300">
                        {quote.quote_number}
                      </Link>
                    </td>
                    <td className="px-4 py-3 text-sm text-surface-100">{quote.title}</td>
                    <td className="px-4 py-3 text-sm text-surface-400">
                      {quote.customer ? (
                        <Link to={`/customers/${quote.customer.id}`} className="hover:text-surface-200">
                          {quote.customer.name}
                        </Link>
                      ) : '—'}
                    </td>
                    <td className="px-4 py-3">
                      <span className={`inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium ${config.color}`}>
                        <StatusIcon className="h-3 w-3" />
                        {config.label}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-sm text-surface-300 text-right font-mono">
                      {getMaxOptionTotal(quote) > 0 ? formatCents(getMaxOptionTotal(quote)) : '—'}
                    </td>
                    <td className="px-4 py-3 text-sm text-surface-500">
                      {new Date(quote.created_at).toLocaleDateString()}
                    </td>
                    <td className="px-4 py-3">
                      <Link to={`/quotes/${quote.id}`} className="text-surface-500 hover:text-surface-300">
                        <ChevronRight className="h-4 w-4" />
                      </Link>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}

      {/* Create Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-surface-900 border border-surface-700 rounded-xl p-6 w-full max-w-md">
            <h2 className="text-lg font-display font-semibold text-surface-100 mb-4">New Quote</h2>
            <form onSubmit={handleCreate} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-surface-300 mb-1">Customer *</label>
                <select name="customer_id" required className="w-full px-3 py-2 bg-surface-800 border border-surface-700 rounded-lg text-surface-100 text-sm focus:outline-none focus:ring-1 focus:ring-accent-500">
                  <option value="">Select a customer...</option>
                  {customers.map((c) => (
                    <option key={c.id} value={c.id}>{c.name}{c.company ? ` (${c.company})` : ''}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-surface-300 mb-1">Title *</label>
                <input name="title" required className="w-full px-3 py-2 bg-surface-800 border border-surface-700 rounded-lg text-surface-100 text-sm focus:outline-none focus:ring-1 focus:ring-accent-500" placeholder="e.g. Custom enclosure build" />
              </div>
              <div>
                <label className="block text-sm font-medium text-surface-300 mb-1">Notes</label>
                <textarea name="notes" rows={3} className="w-full px-3 py-2 bg-surface-800 border border-surface-700 rounded-lg text-surface-100 text-sm focus:outline-none focus:ring-1 focus:ring-accent-500" />
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button type="button" onClick={() => setShowCreateModal(false)} className="px-4 py-2 text-sm text-surface-400 hover:text-surface-200">Cancel</button>
                <button type="submit" className="px-4 py-2 bg-accent-500 text-white rounded-lg hover:bg-accent-600 text-sm font-medium">Create Quote</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
