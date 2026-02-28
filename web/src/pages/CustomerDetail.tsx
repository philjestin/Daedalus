import { useEffect, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { ArrowLeft, Mail, Building2, Phone, FileText, Package } from 'lucide-react'
import { customersApi, quotesApi, ordersApi } from '../api/client'
import type { Customer, Quote, Order } from '../types'

export default function CustomerDetail() {
  const { id } = useParams<{ id: string }>()
  const [customer, setCustomer] = useState<Customer | null>(null)
  const [quotes, setQuotes] = useState<Quote[]>([])
  const [orders, setOrders] = useState<Order[]>([])
  const [loading, setLoading] = useState(true)
  const [activeTab, setActiveTab] = useState<'quotes' | 'orders'>('quotes')
  const [editing, setEditing] = useState(false)

  const loadData = async () => {
    if (!id) return
    try {
      const customerData = await customersApi.get(id)
      setCustomer(customerData)

      // Load related data separately so failures don't block the page
      quotesApi.list({ customer_id: id }).then(setQuotes).catch(() => {})
      ordersApi.list().then(data => setOrders(data.filter(o => o.customer_id === id))).catch(() => {})
    } catch (err) {
      console.error('Failed to load customer:', err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadData()
  }, [id])

  const handleUpdate = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    if (!id) return
    const form = e.currentTarget
    const formData = new FormData(form)
    try {
      const updated = await customersApi.update(id, {
        name: formData.get('name') as string,
        email: formData.get('email') as string || undefined,
        company: formData.get('company') as string || undefined,
        phone: formData.get('phone') as string || undefined,
        notes: formData.get('notes') as string || undefined,
      })
      setCustomer(updated)
      setEditing(false)
    } catch (err) {
      console.error('Failed to update customer:', err)
    }
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-accent-500" />
      </div>
    )
  }

  if (!customer) {
    return (
      <div className="p-6 text-center text-surface-400">Customer not found</div>
    )
  }

  const statusColors: Record<string, string> = {
    draft: 'bg-surface-700 text-surface-300',
    sent: 'bg-blue-500/20 text-blue-400',
    accepted: 'bg-green-500/20 text-green-400',
    rejected: 'bg-red-500/20 text-red-400',
    expired: 'bg-surface-700 text-surface-400',
  }

  return (
    <div className="p-6 max-w-7xl mx-auto">
      {/* Header */}
      <div className="mb-6">
        <Link to="/customers" className="inline-flex items-center gap-1.5 text-sm text-surface-400 hover:text-surface-200 mb-4">
          <ArrowLeft className="h-4 w-4" />
          Back to Customers
        </Link>
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-display font-bold text-surface-100">{customer.name}</h1>
          <button
            onClick={() => setEditing(!editing)}
            className="px-4 py-2 text-sm text-surface-400 hover:text-surface-200 border border-surface-700 rounded-lg hover:bg-surface-800"
          >
            {editing ? 'Cancel' : 'Edit'}
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Info / Edit */}
        <div className="lg:col-span-1">
          {editing ? (
            <div className="bg-surface-900 border border-surface-800 rounded-lg p-4">
              <form onSubmit={handleUpdate} className="space-y-3">
                <div>
                  <label className="block text-xs font-medium text-surface-400 mb-1">Name</label>
                  <input name="name" defaultValue={customer.name} required className="w-full px-3 py-2 bg-surface-800 border border-surface-700 rounded-lg text-surface-100 text-sm focus:outline-none focus:ring-1 focus:ring-accent-500" />
                </div>
                <div>
                  <label className="block text-xs font-medium text-surface-400 mb-1">Email</label>
                  <input name="email" type="email" defaultValue={customer.email} className="w-full px-3 py-2 bg-surface-800 border border-surface-700 rounded-lg text-surface-100 text-sm focus:outline-none focus:ring-1 focus:ring-accent-500" />
                </div>
                <div>
                  <label className="block text-xs font-medium text-surface-400 mb-1">Company</label>
                  <input name="company" defaultValue={customer.company} className="w-full px-3 py-2 bg-surface-800 border border-surface-700 rounded-lg text-surface-100 text-sm focus:outline-none focus:ring-1 focus:ring-accent-500" />
                </div>
                <div>
                  <label className="block text-xs font-medium text-surface-400 mb-1">Phone</label>
                  <input name="phone" defaultValue={customer.phone} className="w-full px-3 py-2 bg-surface-800 border border-surface-700 rounded-lg text-surface-100 text-sm focus:outline-none focus:ring-1 focus:ring-accent-500" />
                </div>
                <div>
                  <label className="block text-xs font-medium text-surface-400 mb-1">Notes</label>
                  <textarea name="notes" defaultValue={customer.notes} rows={3} className="w-full px-3 py-2 bg-surface-800 border border-surface-700 rounded-lg text-surface-100 text-sm focus:outline-none focus:ring-1 focus:ring-accent-500" />
                </div>
                <button type="submit" className="w-full px-4 py-2 bg-accent-500 text-white rounded-lg hover:bg-accent-600 text-sm font-medium">Save</button>
              </form>
            </div>
          ) : (
            <div className="bg-surface-900 border border-surface-800 rounded-lg p-4 space-y-3">
              {customer.email && (
                <div className="flex items-center gap-2 text-sm text-surface-300">
                  <Mail className="h-4 w-4 text-surface-500" />
                  {customer.email}
                </div>
              )}
              {customer.company && (
                <div className="flex items-center gap-2 text-sm text-surface-300">
                  <Building2 className="h-4 w-4 text-surface-500" />
                  {customer.company}
                </div>
              )}
              {customer.phone && (
                <div className="flex items-center gap-2 text-sm text-surface-300">
                  <Phone className="h-4 w-4 text-surface-500" />
                  {customer.phone}
                </div>
              )}
              {customer.notes && (
                <div className="pt-2 border-t border-surface-800">
                  <p className="text-sm text-surface-400">{customer.notes}</p>
                </div>
              )}
              <div className="pt-2 border-t border-surface-800 text-xs text-surface-500">
                Created {new Date(customer.created_at).toLocaleDateString()}
              </div>
            </div>
          )}
        </div>

        {/* Quotes & Orders Tabs */}
        <div className="lg:col-span-2">
          <div className="flex gap-4 mb-4 border-b border-surface-800">
            <button
              onClick={() => setActiveTab('quotes')}
              className={`pb-2 text-sm font-medium transition-colors border-b-2 ${activeTab === 'quotes' ? 'border-accent-500 text-accent-400' : 'border-transparent text-surface-400 hover:text-surface-200'}`}
            >
              <FileText className="h-4 w-4 inline mr-1.5" />
              Quotes ({quotes.length})
            </button>
            <button
              onClick={() => setActiveTab('orders')}
              className={`pb-2 text-sm font-medium transition-colors border-b-2 ${activeTab === 'orders' ? 'border-accent-500 text-accent-400' : 'border-transparent text-surface-400 hover:text-surface-200'}`}
            >
              <Package className="h-4 w-4 inline mr-1.5" />
              Orders ({orders.length})
            </button>
          </div>

          {activeTab === 'quotes' && (
            <div className="space-y-2">
              {quotes.length === 0 ? (
                <p className="text-sm text-surface-500 py-8 text-center">No quotes for this customer</p>
              ) : (
                quotes.map((quote) => (
                  <Link
                    key={quote.id}
                    to={`/quotes/${quote.id}`}
                    className="block bg-surface-900 border border-surface-800 rounded-lg p-4 hover:bg-surface-800/50 transition-colors"
                  >
                    <div className="flex items-center justify-between">
                      <div>
                        <span className="text-sm font-mono text-surface-500 mr-2">{quote.quote_number}</span>
                        <span className="text-sm font-medium text-surface-100">{quote.title}</span>
                      </div>
                      <span className={`px-2 py-0.5 rounded-full text-xs font-medium ${statusColors[quote.status] || ''}`}>
                        {quote.status}
                      </span>
                    </div>
                    <div className="text-xs text-surface-500 mt-1">
                      {new Date(quote.created_at).toLocaleDateString()}
                      {quote.options && quote.options.length > 0 && (
                        <> &middot; {quote.options.length} option{quote.options.length !== 1 ? 's' : ''}</>
                      )}
                    </div>
                  </Link>
                ))
              )}
            </div>
          )}

          {activeTab === 'orders' && (
            <div className="space-y-2">
              {orders.length === 0 ? (
                <p className="text-sm text-surface-500 py-8 text-center">No orders for this customer</p>
              ) : (
                orders.map((order) => (
                  <Link
                    key={order.id}
                    to={`/orders/${order.id}`}
                    className="block bg-surface-900 border border-surface-800 rounded-lg p-4 hover:bg-surface-800/50 transition-colors"
                  >
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-medium text-surface-100">{order.customer_name}</span>
                      <span className="text-xs text-surface-500">{order.status}</span>
                    </div>
                    <div className="text-xs text-surface-500 mt-1">
                      {order.source} &middot; {new Date(order.created_at).toLocaleDateString()}
                    </div>
                  </Link>
                ))
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
