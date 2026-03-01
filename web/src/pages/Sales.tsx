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
  TrendingUp,
  TrendingDown,
} from 'lucide-react'
import { salesApi, statsApi, projectsApi, customersApi } from '../api/client'
import { cn } from '../lib/utils'
import type { Sale, SalesChannel, ProjectSummary, WeeklyInsights, Customer } from '../types'
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

const formatTime = (seconds: number) => {
  if (!seconds || seconds <= 0) return '-'
  const hours = Math.floor(seconds / 3600)
  const mins = Math.floor((seconds % 3600) / 60)
  if (hours > 0) return `${hours}h ${mins}m`
  return `${mins}m`
}

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

  const { data: weeklyInsights } = useQuery({
    queryKey: ['sales', 'weekly-insights'],
    queryFn: () => salesApi.getWeeklyInsights(),
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
    const totalCOGS = projectSales.reduce((sum, p) => sum + (p.total_cogs_cents || 0), 0)
    const totalProfit = totalNet - totalCOGS
    const count = sales.length
    const avgOrder = count > 0 ? Math.round(totalGross / count) : 0
    return { totalGross, totalNet, totalCOGS, totalProfit, count, avgOrder }
  }, [sales, projectSales])

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
      <div className="grid grid-cols-2 lg:grid-cols-5 gap-4 mb-8">
        <div className="card p-4">
          <div className="text-sm text-surface-500 mb-1">Gross Revenue</div>
          <div className="text-2xl font-semibold text-emerald-400">
            {formatCents(totals.totalGross)}
          </div>
        </div>
        <div className="card p-4">
          <div className="text-sm text-surface-500 mb-1">Net Revenue</div>
          <div className="text-2xl font-semibold text-surface-100">
            {formatCents(totals.totalNet)}
          </div>
          <div className="text-xs text-surface-500 mt-1">after fees</div>
        </div>
        <div className="card p-4">
          <div className="text-sm text-surface-500 mb-1">Total COGS</div>
          <div className="text-2xl font-semibold text-red-400">
            {formatCents(totals.totalCOGS)}
          </div>
          <div className="text-xs text-surface-500 mt-1">materials + printing + supplies</div>
        </div>
        <div className="card p-4">
          <div className="text-sm text-surface-500 mb-1">Net Profit</div>
          <div className={cn(
            'text-2xl font-semibold',
            totals.totalProfit >= 0 ? 'text-emerald-400' : 'text-red-400'
          )}>
            {formatCents(totals.totalProfit)}
          </div>
          <div className="text-xs text-surface-500 mt-1">revenue - fees - COGS</div>
        </div>
        <div className="card p-4">
          <div className="text-sm text-surface-500 mb-1">Sales</div>
          <div className="text-2xl font-semibold text-blue-400">
            {totals.count}
          </div>
          <div className="text-xs text-surface-500 mt-1">avg {formatCents(totals.avgOrder)}</div>
        </div>
      </div>

      {/* This Week in Sales */}
      {weeklyInsights && (
        <WeeklyInsightsCard insights={weeklyInsights} />
      )}

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
                    <th className="pb-2 font-medium text-right">Unit Cost</th>
                    <th className="pb-2 font-medium text-right">COGS</th>
                    <th className="pb-2 font-medium text-right">Print Time</th>
                    <th className="pb-2 font-medium text-right">Profit</th>
                  </tr>
                </thead>
                <tbody>
                  {projectSales.map((p) => {
                    const printSeconds = (p.total_print_seconds || 0) > 0
                      ? p.total_print_seconds
                      : (p.estimated_print_seconds || 0)
                    return (
                      <tr key={p.project_id || p.project_name} className="border-b border-surface-800">
                        <td className="py-2.5 text-surface-100 font-medium">{p.project_name}</td>
                        <td className="py-2.5 text-right text-surface-300">{p.count}</td>
                        <td className="py-2.5 text-right text-emerald-400 font-medium">{formatCents(p.gross_cents)}</td>
                        <td className="py-2.5 text-right text-surface-300">{formatCents(p.unit_cost_cents || 0)}</td>
                        <td className="py-2.5 text-right text-red-400">{formatCents(p.total_cogs_cents || 0)}</td>
                        <td className="py-2.5 text-right text-surface-300">{formatTime(printSeconds)}</td>
                        <td className={cn(
                          'py-2.5 text-right font-medium',
                          (p.profit_cents || 0) >= 0 ? 'text-emerald-400' : 'text-red-400'
                        )}>
                          {formatCents(p.profit_cents || 0)}
                        </td>
                      </tr>
                    )
                  })}
                </tbody>
                <tfoot>
                  <tr className="text-surface-100 font-semibold">
                    <td className="pt-3">Total</td>
                    <td className="pt-3 text-right">{projectSales.reduce((s, p) => s + p.count, 0)}</td>
                    <td className="pt-3 text-right text-emerald-400">{formatCents(projectSales.reduce((s, p) => s + p.gross_cents, 0))}</td>
                    <td className="pt-3 text-right">-</td>
                    <td className="pt-3 text-right text-red-400">{formatCents(projectSales.reduce((s, p) => s + (p.total_cogs_cents || 0), 0))}</td>
                    <td className="pt-3 text-right">
                      {formatTime(projectSales.reduce((s, p) => {
                        const t = (p.total_print_seconds || 0) > 0 ? p.total_print_seconds : (p.estimated_print_seconds || 0)
                        return s + t
                      }, 0))}
                    </td>
                    <td className={cn(
                      'pt-3 text-right',
                      projectSales.reduce((s, p) => s + (p.profit_cents || 0), 0) >= 0 ? 'text-emerald-400' : 'text-red-400'
                    )}>
                      {formatCents(projectSales.reduce((s, p) => s + (p.profit_cents || 0), 0))}
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

function pctChange(current: number, previous: number): number | null {
  if (previous === 0) return current > 0 ? 100 : null
  return Math.round(((current - previous) / previous) * 100)
}

function ChangeBadge({ current, previous }: { current: number; previous: number }) {
  const pct = pctChange(current, previous)
  if (pct === null) return null
  const isUp = pct >= 0
  const Icon = isUp ? TrendingUp : TrendingDown
  return (
    <span className={cn(
      'inline-flex items-center gap-0.5 text-xs font-medium px-1.5 py-0.5 rounded-full',
      isUp ? 'bg-emerald-500/15 text-emerald-400' : 'bg-red-500/15 text-red-400'
    )}>
      <Icon className="h-3 w-3" />
      {Math.abs(pct)}%
    </span>
  )
}

function formatWeekRange(start: string, end: string) {
  const s = new Date(start + 'T00:00:00')
  const e = new Date(end + 'T00:00:00')
  const fmt = (d: Date) => d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
  return `${fmt(s)} – ${fmt(e)}`
}

function WeeklyInsightsCard({ insights }: { insights: WeeklyInsights }) {
  const tw = insights.this_week
  const lw = insights.last_week
  const avgOrder = tw.count > 0 ? Math.round(tw.gross_cents / tw.count) : 0
  const lastAvg = lw.count > 0 ? Math.round(lw.gross_cents / lw.count) : 0
  const pendingAvg = insights.pending_count > 0
    ? Math.round(insights.pending_revenue_cents / insights.pending_count)
    : 0

  return (
    <div className="mb-8">
      <div className="flex items-center gap-3 mb-4">
        <h2 className="text-lg font-semibold text-surface-100">This Week in Sales</h2>
        <span className="text-sm text-surface-500">{formatWeekRange(insights.week_start, insights.week_end)}</span>
      </div>

      {/* Completed sales row */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="card p-4">
          <div className="flex items-center justify-between mb-1">
            <span className="text-sm text-surface-500">Gross Revenue</span>
            <ChangeBadge current={tw.gross_cents} previous={lw.gross_cents} />
          </div>
          <div className="text-2xl font-semibold text-emerald-400">{formatCents(tw.gross_cents)}</div>
        </div>
        <div className="card p-4">
          <div className="flex items-center justify-between mb-1">
            <span className="text-sm text-surface-500">Net Revenue</span>
            <ChangeBadge current={tw.net_cents} previous={lw.net_cents} />
          </div>
          <div className="text-2xl font-semibold text-surface-100">{formatCents(tw.net_cents)}</div>
        </div>
        <div className="card p-4">
          <div className="flex items-center justify-between mb-1">
            <span className="text-sm text-surface-500"># Sales</span>
            <ChangeBadge current={tw.count} previous={lw.count} />
          </div>
          <div className="text-2xl font-semibold text-blue-400">{tw.count}</div>
        </div>
        <div className="card p-4">
          <div className="flex items-center justify-between mb-1">
            <span className="text-sm text-surface-500">Avg Order</span>
            <ChangeBadge current={avgOrder} previous={lastAvg} />
          </div>
          <div className="text-2xl font-semibold text-surface-100">{formatCents(avgOrder)}</div>
        </div>
      </div>

      {/* Pending sales row */}
      {insights.pending_count > 0 && (
        <div className="mt-4">
          <div className="text-xs font-medium text-amber-400/80 uppercase tracking-wider mb-2">Pipeline — In Production</div>
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
            <div className="card p-4 border border-amber-500/20">
              <div className="text-sm text-surface-500 mb-1">Pending Revenue</div>
              <div className="text-2xl font-semibold text-amber-400">{formatCents(insights.pending_revenue_cents)}</div>
            </div>
            <div className="card p-4 border border-amber-500/20">
              <div className="text-sm text-surface-500 mb-1">Pending Units</div>
              <div className="text-2xl font-semibold text-amber-400">{insights.pending_count}</div>
            </div>
            <div className="card p-4 border border-amber-500/20">
              <div className="text-sm text-surface-500 mb-1">Avg Unit Value</div>
              <div className="text-2xl font-semibold text-surface-100">{formatCents(pendingAvg)}</div>
            </div>
            <div className="card p-4 border border-amber-500/20">
              <div className="text-sm text-surface-500 mb-1">Total Pipeline</div>
              <div className="text-2xl font-semibold text-surface-100">
                {formatCents(tw.gross_cents + insights.pending_revenue_cents)}
              </div>
              <div className="text-xs text-surface-500 mt-1">completed + pending</div>
            </div>
          </div>
        </div>
      )}

      {/* Channel pills */}
      {insights.channels.length > 0 && (
        <div className="flex flex-wrap gap-2 mt-3">
          {insights.channels.map((ch) => {
            const cfg = channelConfig[ch.channel as keyof typeof channelConfig] || channelConfig.other
            return (
              <span key={ch.channel} className={cn('inline-flex items-center gap-1 text-xs px-2 py-1 rounded-full', cfg.bgColor, cfg.color)}>
                <span className="capitalize">{ch.channel}</span>
                <span className="opacity-60">&times; {ch.count}</span>
              </span>
            )
          })}
        </div>
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

  const [selectedProjectId, setSelectedProjectId] = useState<string>(sale?.project_id || '')
  const [selectedCustomerId, setSelectedCustomerId] = useState<string>(sale?.customer_id || '')
  const [projectSummary, setProjectSummary] = useState<ProjectSummary | null>(null)
  const [loadingSummary, setLoadingSummary] = useState(false)

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

  // Fetch projects for dropdown
  const { data: projects = [] } = useQuery({
    queryKey: ['projects'],
    queryFn: () => projectsApi.list(),
  })

  // Fetch customers for dropdown
  const { data: customers = [] } = useQuery({
    queryKey: ['customers'],
    queryFn: () => customersApi.list(),
  })

  // Fetch project summary when project is selected
  const handleProjectChange = async (projectId: string) => {
    setSelectedProjectId(projectId)
    setProjectSummary(null)

    if (projectId) {
      setLoadingSummary(true)
      try {
        const [project, summary] = await Promise.all([
          projectsApi.get(projectId),
          projectsApi.getSummary(projectId),
        ])
        setProjectSummary(summary)
        // Auto-populate item description with project name
        if (!itemDescription) {
          setItemDescription(project.name)
        }
      } catch (err) {
        console.error('Failed to fetch project summary:', err)
      } finally {
        setLoadingSummary(false)
      }
    }
  }

  const handleCustomerChange = (customerId: string) => {
    setSelectedCustomerId(customerId)
    if (customerId) {
      const customer = customers.find((c: Customer) => c.id === customerId)
      if (customer) {
        setCustomerName(customer.name)
      }
    }
  }

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
        customer_id: selectedCustomerId || undefined,
        order_reference: orderReference,
        notes,
        project_id: selectedProjectId || undefined,
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
          {/* Project Selector */}
          <div>
            <label className="block text-sm text-surface-400 mb-1">Link to Project (optional)</label>
            <select
              value={selectedProjectId}
              onChange={(e) => handleProjectChange(e.target.value)}
              className="input w-full"
            >
              <option value="">-- No project --</option>
              {projects.map((project) => (
                <option key={project.id} value={project.id}>
                  {project.name}
                </option>
              ))}
            </select>
            <p className="text-xs text-surface-500 mt-1">
              Linking to a project lets you track profit by including material and printing costs
            </p>
          </div>

          {/* Project Cost Summary */}
          {selectedProjectId && (
            <div className="p-3 rounded-lg bg-surface-800/50 border border-surface-700">
              <div className="text-sm font-medium text-surface-300 mb-2">Project Costs</div>
              {loadingSummary ? (
                <div className="text-sm text-surface-500">Loading costs...</div>
              ) : projectSummary ? (
                <div className="grid grid-cols-2 gap-2 text-sm">
                  <div className="flex justify-between">
                    <span className="text-surface-500">Material Cost:</span>
                    <span className="text-surface-300">{formatCents(projectSummary.material_cost_cents)}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-surface-500">Printer Time:</span>
                    <span className="text-surface-300">{formatCents(projectSummary.printer_time_cost_cents)}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-surface-500">Supply Cost:</span>
                    <span className="text-surface-300">{formatCents(projectSummary.supply_cost_cents)}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-surface-500">Unit Cost:</span>
                    <span className="text-surface-300">{formatCents(projectSummary.unit_cost_cents)}</span>
                  </div>
                  <div className="col-span-2 pt-2 border-t border-surface-700 flex justify-between font-medium">
                    <span className="text-surface-400">Total COGS:</span>
                    <span className="text-red-400">{formatCents(projectSummary.total_cost_cents)}</span>
                  </div>
                  {toCents(gross) > 0 && (
                    <div className="col-span-2 flex justify-between font-medium">
                      <span className="text-surface-400">Est. Profit:</span>
                      <span className={cn(
                        netCents - projectSummary.total_cost_cents >= 0 ? 'text-emerald-400' : 'text-red-400'
                      )}>
                        {formatCents(netCents - projectSummary.total_cost_cents)}
                      </span>
                    </div>
                  )}
                </div>
              ) : (
                <div className="text-sm text-surface-500">No cost data available</div>
              )}
            </div>
          )}

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

          {/* Row 6: Customer + Order Reference */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm text-surface-400 mb-1">Customer</label>
              <select
                value={selectedCustomerId}
                onChange={(e) => handleCustomerChange(e.target.value)}
                className="input w-full"
              >
                <option value="">-- No customer --</option>
                {customers.map((c: Customer) => (
                  <option key={c.id} value={c.id}>
                    {c.name}{c.company ? ` (${c.company})` : ''}
                  </option>
                ))}
              </select>
              {!selectedCustomerId && (
                <input
                  type="text"
                  value={customerName}
                  onChange={(e) => setCustomerName(e.target.value)}
                  className="input w-full mt-2"
                  placeholder="Or type a name for one-off sales"
                />
              )}
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
