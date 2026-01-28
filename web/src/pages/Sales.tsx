import { useState, useMemo } from 'react'
import { useQuery, useQueryClient, useMutation } from '@tanstack/react-query'
import {
  DollarSign,
  Store,
  Tag,
  Globe,
  Users,
  Plus,
  Edit3,
  Trash2,
  X,
  ShoppingBag,
} from 'lucide-react'
import { salesApi, statsApi } from '../api/client'
import { cn } from '../lib/utils'
import type { Sale, SalesChannel } from '../types'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  BarElement,
  Tooltip as ChartTooltip,
} from 'chart.js'
import { Bar } from 'react-chartjs-2'

ChartJS.register(CategoryScale, LinearScale, BarElement, ChartTooltip)

const channelConfig: Record<SalesChannel, { icon: React.ElementType; color: string; bgColor: string }> = {
  marketplace: { icon: Store, color: 'text-purple-400', bgColor: 'bg-purple-500/20' },
  etsy: { icon: Tag, color: 'text-orange-400', bgColor: 'bg-orange-500/20' },
  website: { icon: Globe, color: 'text-blue-400', bgColor: 'bg-blue-500/20' },
  direct: { icon: Users, color: 'text-emerald-400', bgColor: 'bg-emerald-500/20' },
  other: { icon: DollarSign, color: 'text-surface-400', bgColor: 'bg-surface-500/20' },
}

const formatCents = (cents: number) => `$${(cents / 100).toFixed(2)}`

const toCents = (val: string) => Math.round(parseFloat(val || '0') * 100)
const fromCents = (cents: number) => (cents / 100).toFixed(2)

export default function Sales() {
  const queryClient = useQueryClient()
  const [editingSale, setEditingSale] = useState<Sale | null>(null)
  const [showModal, setShowModal] = useState(false)

  const { data: sales = [], isLoading } = useQuery({
    queryKey: ['sales'],
    queryFn: () => salesApi.list(),
  })

  const { data: projectSales = [] } = useQuery({
    queryKey: ['stats', 'sales-by-project'],
    queryFn: () => statsApi.getSalesByProject(),
  })

  const deleteSale = useMutation({
    mutationFn: (id: string) => salesApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['sales'] })
    },
  })

  const totals = useMemo(() => {
    const totalGross = sales.reduce((sum, s) => sum + s.gross_cents, 0)
    const totalNet = sales.reduce((sum, s) => sum + s.net_cents, 0)
    const count = sales.length
    const avgOrder = count > 0 ? Math.round(totalGross / count) : 0
    return { totalGross, totalNet, count, avgOrder }
  }, [sales])

  const openAdd = () => {
    setEditingSale(null)
    setShowModal(true)
  }

  const openEdit = (sale: Sale) => {
    setEditingSale(sale)
    setShowModal(true)
  }

  return (
    <div className="p-4 sm:p-6 lg:p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-display font-bold text-surface-100">
            Sales
          </h1>
          <p className="text-surface-400 mt-1">
            Track revenue across all channels
          </p>
        </div>
        <button onClick={openAdd} className="btn btn-primary">
          <Plus className="h-4 w-4 mr-2" />
          Add Sale
        </button>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
        <div className="card p-4">
          <div className="text-sm text-surface-500 mb-1">Total Revenue</div>
          <div className="text-2xl font-semibold text-emerald-400">
            {formatCents(totals.totalGross)}
          </div>
        </div>
        <div className="card p-4">
          <div className="text-sm text-surface-500 mb-1">Net Profit</div>
          <div className={cn(
            'text-2xl font-semibold',
            totals.totalNet >= 0 ? 'text-emerald-400' : 'text-red-400'
          )}>
            {formatCents(totals.totalNet)}
          </div>
        </div>
        <div className="card p-4">
          <div className="text-sm text-surface-500 mb-1">Sale Count</div>
          <div className="text-2xl font-semibold text-blue-400">
            {totals.count}
          </div>
        </div>
        <div className="card p-4">
          <div className="text-sm text-surface-500 mb-1">Avg Order Value</div>
          <div className="text-2xl font-semibold text-surface-100">
            {formatCents(totals.avgOrder)}
          </div>
        </div>
      </div>

      {/* Project Analytics */}
      {projectSales.length > 0 && (
        <div className="mb-8">
          <h2 className="text-lg font-semibold text-surface-100 mb-4">
            Top Projects
          </h2>
          <div className="grid grid-cols-1 xl:grid-cols-2 gap-6 mb-6">
            {/* Bar chart */}
            <div className="card p-6">
              <h3 className="text-sm font-medium text-surface-400 mb-3">Revenue by Project</h3>
              <div className="h-[250px]">
                <Bar
                  data={{
                    labels: projectSales.map(p => p.project_name),
                    datasets: [{
                      label: 'Revenue',
                      data: projectSales.map(p => p.gross_cents),
                      backgroundColor: '#10b981',
                      borderRadius: 4,
                    }],
                  }}
                  options={{
                    responsive: true,
                    maintainAspectRatio: false,
                    indexAxis: 'y',
                    plugins: {
                      tooltip: {
                        backgroundColor: '#1e293b',
                        borderColor: '#334155',
                        borderWidth: 1,
                        titleColor: '#e2e8f0',
                        bodyColor: '#e2e8f0',
                        callbacks: {
                          label: (ctx) => `$${((ctx.parsed.x ?? 0) / 100).toFixed(2)}`,
                        },
                      },
                      legend: { display: false },
                    },
                    scales: {
                      x: {
                        ticks: {
                          color: '#64748b',
                          callback: (value) => `$${(Number(value) / 100).toFixed(0)}`,
                        },
                        grid: { color: '#1e293b' },
                      },
                      y: {
                        ticks: { color: '#94a3b8', font: { size: 12 } },
                        grid: { display: false },
                      },
                    },
                  }}
                />
              </div>
            </div>

            {/* Table */}
            <div className="card p-6 overflow-x-auto">
              <h3 className="text-sm font-medium text-surface-400 mb-3">Breakdown</h3>
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-left text-surface-500 border-b border-surface-700">
                    <th className="pb-2 font-medium">Project</th>
                    <th className="pb-2 font-medium text-right">Units</th>
                    <th className="pb-2 font-medium text-right">Revenue</th>
                    <th className="pb-2 font-medium text-right">Avg Price</th>
                  </tr>
                </thead>
                <tbody>
                  {projectSales.map((p) => (
                    <tr key={p.project_id || p.project_name} className="border-b border-surface-800">
                      <td className="py-2.5 text-surface-100 font-medium">{p.project_name}</td>
                      <td className="py-2.5 text-right text-surface-300">{p.count}</td>
                      <td className="py-2.5 text-right text-emerald-400 font-medium">{formatCents(p.gross_cents)}</td>
                      <td className="py-2.5 text-right text-surface-300">{formatCents(p.avg_cents)}</td>
                    </tr>
                  ))}
                </tbody>
                <tfoot>
                  <tr className="text-surface-100 font-semibold">
                    <td className="pt-3">Total</td>
                    <td className="pt-3 text-right">{projectSales.reduce((s, p) => s + p.count, 0)}</td>
                    <td className="pt-3 text-right text-emerald-400">{formatCents(projectSales.reduce((s, p) => s + p.gross_cents, 0))}</td>
                    <td className="pt-3 text-right">
                      {formatCents(
                        projectSales.reduce((s, p) => s + p.gross_cents, 0) /
                        Math.max(projectSales.reduce((s, p) => s + p.count, 0), 1)
                      )}
                    </td>
                  </tr>
                </tfoot>
              </table>
            </div>
          </div>
        </div>
      )}

      {/* Sales List */}
      {isLoading ? (
        <div className="text-surface-500">Loading sales...</div>
      ) : sales.length === 0 ? (
        <div className="card p-8 text-center">
          <ShoppingBag className="h-12 w-12 mx-auto mb-3 text-surface-600" />
          <h3 className="text-lg font-medium text-surface-300 mb-2">
            No sales yet
          </h3>
          <p className="text-surface-500 mb-4">
            Record a sale to start tracking revenue
          </p>
          <button onClick={openAdd} className="btn btn-primary">
            <Plus className="h-4 w-4 mr-2" />
            Add Sale
          </button>
        </div>
      ) : (
        <div className="space-y-3">
          {sales.map((sale) => {
            const cfg = channelConfig[sale.channel] || channelConfig.other
            const Icon = cfg.icon
            return (
              <div
                key={sale.id}
                className="card p-4 flex items-center justify-between"
              >
                <div className="flex items-center gap-4">
                  <div
                    className={cn(
                      'w-10 h-10 rounded-lg flex items-center justify-center',
                      cfg.bgColor,
                      cfg.color
                    )}
                  >
                    <Icon className="h-5 w-5" />
                  </div>
                  <div>
                    <div className="font-medium text-surface-100">
                      {sale.item_description || 'Sale'}
                    </div>
                    <div className="text-sm text-surface-500 flex items-center gap-2">
                      <span>{new Date(sale.occurred_at).toLocaleDateString()}</span>
                      <span>·</span>
                      <span className="capitalize">{sale.channel}</span>
                      {sale.customer_name && (
                        <>
                          <span>·</span>
                          <span>{sale.customer_name}</span>
                        </>
                      )}
                    </div>
                  </div>
                </div>

                <div className="flex items-center gap-4">
                  <div className="text-right">
                    <div className="font-semibold text-surface-100">
                      {formatCents(sale.gross_cents)}
                    </div>
                    <div className="text-xs text-surface-500">
                      net {formatCents(sale.net_cents)}
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    <button
                      onClick={() => openEdit(sale)}
                      className="btn btn-ghost text-xs py-1 px-2 text-surface-400 hover:text-surface-200"
                    >
                      <Edit3 className="h-3 w-3" />
                    </button>
                    <button
                      onClick={() => deleteSale.mutate(sale.id)}
                      className="btn btn-ghost text-xs py-1 px-2 text-red-400 hover:text-red-300"
                    >
                      <Trash2 className="h-3 w-3" />
                    </button>
                  </div>
                </div>
              </div>
            )
          })}
        </div>
      )}

      {/* Add/Edit Modal */}
      {showModal && (
        <SaleModal
          sale={editingSale}
          onClose={() => setShowModal(false)}
          onSuccess={() => {
            queryClient.invalidateQueries({ queryKey: ['sales'] })
            setShowModal(false)
          }}
        />
      )}
    </div>
  )
}

function SaleModal({
  sale,
  onClose,
  onSuccess,
}: {
  sale: Sale | null
  onClose: () => void
  onSuccess: () => void
}) {
  const isEdit = !!sale

  const [date, setDate] = useState(
    sale ? new Date(sale.occurred_at).toISOString().split('T')[0] : new Date().toISOString().split('T')[0]
  )
  const [channel, setChannel] = useState<SalesChannel>(sale?.channel || 'marketplace')
  const [platform, setPlatform] = useState(sale?.platform || '')
  const [itemDescription, setItemDescription] = useState(sale?.item_description || '')
  const [quantity, setQuantity] = useState(String(sale?.quantity || 1))
  const [gross, setGross] = useState(sale ? fromCents(sale.gross_cents) : '')
  const [fees, setFees] = useState(sale ? fromCents(sale.fees_cents) : '')
  const [shippingCharged, setShippingCharged] = useState(sale ? fromCents(sale.shipping_charged_cents) : '')
  const [shippingCost, setShippingCost] = useState(sale ? fromCents(sale.shipping_cost_cents) : '')
  const [taxCollected, setTaxCollected] = useState(sale ? fromCents(sale.tax_collected_cents) : '')
  const [customerName, setCustomerName] = useState(sale?.customer_name || '')
  const [orderReference, setOrderReference] = useState(sale?.order_reference || '')
  const [notes, setNotes] = useState(sale?.notes || '')
  const [submitting, setSubmitting] = useState(false)

  const netCents = toCents(gross) - toCents(fees) - toCents(shippingCost)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setSubmitting(true)
    try {
      const data: Partial<Sale> = {
        occurred_at: new Date(date + 'T12:00:00Z').toISOString(),
        channel,
        platform,
        item_description: itemDescription,
        quantity: parseInt(quantity) || 1,
        gross_cents: toCents(gross),
        fees_cents: toCents(fees),
        shipping_charged_cents: toCents(shippingCharged),
        shipping_cost_cents: toCents(shippingCost),
        tax_collected_cents: toCents(taxCollected),
        net_cents: netCents,
        customer_name: customerName,
        order_reference: orderReference,
        notes,
      }

      if (isEdit) {
        await salesApi.update(sale.id, data)
      } else {
        await salesApi.create(data)
      }
      onSuccess()
    } catch (err) {
      console.error('Failed to save sale:', err)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div
      className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 overflow-y-auto"
      onClick={(e) => e.target === e.currentTarget && onClose()}
    >
      <div className="card w-full max-w-2xl p-6 my-8">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl font-semibold text-surface-100">
            {isEdit ? 'Edit Sale' : 'Add Sale'}
          </h2>
          <button onClick={onClose} className="text-surface-400 hover:text-surface-200">
            <X className="h-5 w-5" />
          </button>
        </div>

        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Row 1: Date + Channel */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm text-surface-400 mb-1">Date</label>
              <input
                type="date"
                value={date}
                onChange={(e) => setDate(e.target.value)}
                className="input w-full"
                required
              />
            </div>
            <div>
              <label className="block text-sm text-surface-400 mb-1">Channel</label>
              <select
                value={channel}
                onChange={(e) => setChannel(e.target.value as SalesChannel)}
                className="input w-full"
              >
                <option value="marketplace">Marketplace</option>
                <option value="etsy">Etsy</option>
                <option value="website">Website</option>
                <option value="direct">Direct</option>
                <option value="other">Other</option>
              </select>
            </div>
          </div>

          {/* Row 2: Platform + Item Description */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm text-surface-400 mb-1">Platform</label>
              <input
                type="text"
                value={platform}
                onChange={(e) => setPlatform(e.target.value)}
                className="input w-full"
                placeholder="e.g. Facebook Marketplace"
              />
            </div>
            <div>
              <label className="block text-sm text-surface-400 mb-1">Item Description</label>
              <input
                type="text"
                value={itemDescription}
                onChange={(e) => setItemDescription(e.target.value)}
                className="input w-full"
                placeholder="What was sold"
                required
              />
            </div>
          </div>

          {/* Row 3: Quantity + Gross Amount */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm text-surface-400 mb-1">Quantity</label>
              <input
                type="number"
                value={quantity}
                onChange={(e) => setQuantity(e.target.value)}
                className="input w-full"
                min="1"
              />
            </div>
            <div>
              <label className="block text-sm text-surface-400 mb-1">Gross Amount ($)</label>
              <input
                type="number"
                step="0.01"
                value={gross}
                onChange={(e) => setGross(e.target.value)}
                className="input w-full"
                placeholder="0.00"
                required
              />
            </div>
          </div>

          {/* Row 4: Fees + Shipping Charged */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm text-surface-400 mb-1">Fees ($)</label>
              <input
                type="number"
                step="0.01"
                value={fees}
                onChange={(e) => setFees(e.target.value)}
                className="input w-full"
                placeholder="0.00"
              />
            </div>
            <div>
              <label className="block text-sm text-surface-400 mb-1">Shipping Charged ($)</label>
              <input
                type="number"
                step="0.01"
                value={shippingCharged}
                onChange={(e) => setShippingCharged(e.target.value)}
                className="input w-full"
                placeholder="0.00"
              />
            </div>
          </div>

          {/* Row 5: Shipping Cost + Tax Collected */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm text-surface-400 mb-1">Shipping Cost ($)</label>
              <input
                type="number"
                step="0.01"
                value={shippingCost}
                onChange={(e) => setShippingCost(e.target.value)}
                className="input w-full"
                placeholder="0.00"
              />
            </div>
            <div>
              <label className="block text-sm text-surface-400 mb-1">Tax Collected ($)</label>
              <input
                type="number"
                step="0.01"
                value={taxCollected}
                onChange={(e) => setTaxCollected(e.target.value)}
                className="input w-full"
                placeholder="0.00"
              />
            </div>
          </div>

          {/* Row 6: Customer Name + Order Reference */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm text-surface-400 mb-1">Customer Name</label>
              <input
                type="text"
                value={customerName}
                onChange={(e) => setCustomerName(e.target.value)}
                className="input w-full"
                placeholder="Optional"
              />
            </div>
            <div>
              <label className="block text-sm text-surface-400 mb-1">Order Reference</label>
              <input
                type="text"
                value={orderReference}
                onChange={(e) => setOrderReference(e.target.value)}
                className="input w-full"
                placeholder="Optional"
              />
            </div>
          </div>

          {/* Row 7: Notes */}
          <div>
            <label className="block text-sm text-surface-400 mb-1">Notes</label>
            <textarea
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              className="input w-full"
              rows={2}
              placeholder="Optional notes"
            />
          </div>

          {/* Net calculation */}
          <div className="p-3 rounded-lg bg-surface-800/50 border border-surface-700">
            <div className="flex items-center justify-between">
              <span className="text-sm text-surface-400">Calculated Net</span>
              <span className={cn(
                'text-lg font-semibold',
                netCents >= 0 ? 'text-emerald-400' : 'text-red-400'
              )}>
                {formatCents(netCents)}
              </span>
            </div>
            <div className="text-xs text-surface-500 mt-1">
              Gross - Fees - Shipping Cost
            </div>
          </div>

          <div className="flex justify-end gap-3 pt-2">
            <button type="button" onClick={onClose} className="btn btn-ghost">
              Cancel
            </button>
            <button type="submit" disabled={submitting} className="btn btn-primary">
              {submitting ? 'Saving...' : isEdit ? 'Update Sale' : 'Add Sale'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
