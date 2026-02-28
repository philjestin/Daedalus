import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { Users, Plus, ChevronRight, Search, Building2, Mail } from 'lucide-react'
import { customersApi } from '../api/client'
import type { Customer } from '../types'

export default function Customers() {
  const [customers, setCustomers] = useState<Customer[]>([])
  const [loading, setLoading] = useState(true)
  const [search, setSearch] = useState('')
  const [showCreateModal, setShowCreateModal] = useState(false)

  const loadCustomers = async () => {
    try {
      const data = await customersApi.list({ search: search || undefined })
      setCustomers(data)
    } catch (err) {
      console.error('Failed to load customers:', err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    loadCustomers()
  }, [search])

  const handleCreate = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    const form = e.currentTarget
    const formData = new FormData(form)
    try {
      await customersApi.create({
        name: formData.get('name') as string,
        email: formData.get('email') as string || undefined,
        company: formData.get('company') as string || undefined,
        phone: formData.get('phone') as string || undefined,
      })
      setShowCreateModal(false)
      loadCustomers()
    } catch (err) {
      console.error('Failed to create customer:', err)
    }
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
          <h1 className="text-2xl font-display font-bold text-surface-100">Customers</h1>
          <p className="text-sm text-surface-400 mt-1">{customers.length} customer{customers.length !== 1 ? 's' : ''}</p>
        </div>
        <button
          onClick={() => setShowCreateModal(true)}
          className="inline-flex items-center gap-2 px-4 py-2 bg-accent-500 text-white rounded-lg hover:bg-accent-600 transition-colors text-sm font-medium"
        >
          <Plus className="h-4 w-4" />
          New Customer
        </button>
      </div>

      {/* Search */}
      <div className="mb-4">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-surface-500" />
          <input
            type="text"
            placeholder="Search customers..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full pl-10 pr-4 py-2 bg-surface-800 border border-surface-700 rounded-lg text-surface-100 placeholder-surface-500 text-sm focus:outline-none focus:ring-1 focus:ring-accent-500"
          />
        </div>
      </div>

      {/* Table */}
      {customers.length === 0 ? (
        <div className="text-center py-16">
          <Users className="h-12 w-12 text-surface-600 mx-auto mb-4" />
          <h3 className="text-lg font-medium text-surface-300 mb-2">No customers yet</h3>
          <p className="text-surface-500 text-sm mb-4">Create your first customer to start building quotes.</p>
          <button
            onClick={() => setShowCreateModal(true)}
            className="inline-flex items-center gap-2 px-4 py-2 bg-accent-500 text-white rounded-lg hover:bg-accent-600 transition-colors text-sm"
          >
            <Plus className="h-4 w-4" />
            New Customer
          </button>
        </div>
      ) : (
        <div className="bg-surface-900 border border-surface-800 rounded-lg overflow-hidden">
          <table className="w-full">
            <thead>
              <tr className="border-b border-surface-800">
                <th className="text-left text-xs font-medium text-surface-400 uppercase tracking-wider px-4 py-3">Name</th>
                <th className="text-left text-xs font-medium text-surface-400 uppercase tracking-wider px-4 py-3">Company</th>
                <th className="text-left text-xs font-medium text-surface-400 uppercase tracking-wider px-4 py-3">Email</th>
                <th className="text-left text-xs font-medium text-surface-400 uppercase tracking-wider px-4 py-3">Phone</th>
                <th className="w-10" />
              </tr>
            </thead>
            <tbody className="divide-y divide-surface-800">
              {customers.map((customer) => (
                <tr key={customer.id} className="hover:bg-surface-800/50 transition-colors">
                  <td className="px-4 py-3">
                    <Link to={`/customers/${customer.id}`} className="text-sm font-medium text-surface-100 hover:text-accent-400">
                      {customer.name}
                    </Link>
                  </td>
                  <td className="px-4 py-3 text-sm text-surface-400">
                    {customer.company && (
                      <span className="inline-flex items-center gap-1.5">
                        <Building2 className="h-3.5 w-3.5" />
                        {customer.company}
                      </span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-sm text-surface-400">
                    {customer.email && (
                      <span className="inline-flex items-center gap-1.5">
                        <Mail className="h-3.5 w-3.5" />
                        {customer.email}
                      </span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-sm text-surface-400">{customer.phone}</td>
                  <td className="px-4 py-3">
                    <Link to={`/customers/${customer.id}`} className="text-surface-500 hover:text-surface-300">
                      <ChevronRight className="h-4 w-4" />
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Create Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-surface-900 border border-surface-700 rounded-xl p-6 w-full max-w-md">
            <h2 className="text-lg font-display font-semibold text-surface-100 mb-4">New Customer</h2>
            <form onSubmit={handleCreate} className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-surface-300 mb-1">Name *</label>
                <input name="name" required className="w-full px-3 py-2 bg-surface-800 border border-surface-700 rounded-lg text-surface-100 text-sm focus:outline-none focus:ring-1 focus:ring-accent-500" />
              </div>
              <div>
                <label className="block text-sm font-medium text-surface-300 mb-1">Email</label>
                <input name="email" type="email" className="w-full px-3 py-2 bg-surface-800 border border-surface-700 rounded-lg text-surface-100 text-sm focus:outline-none focus:ring-1 focus:ring-accent-500" />
              </div>
              <div>
                <label className="block text-sm font-medium text-surface-300 mb-1">Company</label>
                <input name="company" className="w-full px-3 py-2 bg-surface-800 border border-surface-700 rounded-lg text-surface-100 text-sm focus:outline-none focus:ring-1 focus:ring-accent-500" />
              </div>
              <div>
                <label className="block text-sm font-medium text-surface-300 mb-1">Phone</label>
                <input name="phone" className="w-full px-3 py-2 bg-surface-800 border border-surface-700 rounded-lg text-surface-100 text-sm focus:outline-none focus:ring-1 focus:ring-accent-500" />
              </div>
              <div className="flex justify-end gap-3 pt-2">
                <button type="button" onClick={() => setShowCreateModal(false)} className="px-4 py-2 text-sm text-surface-400 hover:text-surface-200">Cancel</button>
                <button type="submit" className="px-4 py-2 bg-accent-500 text-white rounded-lg hover:bg-accent-600 text-sm font-medium">Create</button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
