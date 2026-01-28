import { useState, useMemo } from 'react'
import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import {
  FolderKanban,
  Printer,
  Play,
  AlertCircle,
  Clock,
  TrendingUp,
  TrendingDown,
  DollarSign,
  Receipt
} from 'lucide-react'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  ArcElement,
  Filler,
  Tooltip as ChartTooltip,
  Legend as ChartLegend,
} from 'chart.js'
import { Line, Bar, Doughnut } from 'react-chartjs-2'
import { useProjects } from '../hooks/useProjects'
import { usePrinters, usePrinterStates } from '../hooks/usePrinters'
import { statsApi } from '../api/client'
import { cn, getStatusBadge } from '../lib/utils'

ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  ArcElement,
  Filler,
  ChartTooltip,
  ChartLegend,
)

const CHANNEL_COLORS: Record<string, string> = {
  marketplace: '#a855f7',
  etsy: '#f97316',
  website: '#3b82f6',
  direct: '#10b981',
  other: '#6b7280',
}

const periodOptions = [
  { label: '30 Days', value: '30d' },
  { label: '90 Days', value: '90d' },
  { label: '12 Months', value: '12m' },
]

const chartDefaults = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    tooltip: {
      backgroundColor: '#1e293b',
      borderColor: '#334155',
      borderWidth: 1,
      titleColor: '#e2e8f0',
      bodyColor: '#e2e8f0',
      callbacks: {
        label: (ctx: { dataset: { label?: string }; parsed: { y?: number | null } }) => {
          const val = ctx.parsed.y ?? 0
          return `${ctx.dataset.label}: $${(val / 100).toFixed(0)}`
        },
      },
    },
    legend: {
      labels: { color: '#94a3b8' },
    },
  },
  scales: {
    x: {
      ticks: { color: '#64748b', font: { size: 11 } },
      grid: { color: '#1e293b' },
    },
    y: {
      ticks: {
        color: '#64748b',
        callback: (value: string | number) => `$${(Number(value) / 100).toFixed(0)}`,
      },
      grid: { color: '#1e293b' },
    },
  },
} as const

export default function Dashboard() {
  const { data: projects = [], isLoading: projectsLoading } = useProjects()
  const { data: printers = [], isLoading: printersLoading } = usePrinters()
  const { data: printerStates = {} } = usePrinterStates()
  const { data: financials } = useQuery({
    queryKey: ['stats', 'financial'],
    queryFn: () => statsApi.getFinancialSummary(),
    refetchInterval: 30000,
  })

  const [chartPeriod, setChartPeriod] = useState('30d')

  const { data: timeSeries } = useQuery({
    queryKey: ['stats', 'time-series', chartPeriod],
    queryFn: () => statsApi.getTimeSeries(chartPeriod),
  })

  const { data: expensesByCategory = [] } = useQuery({
    queryKey: ['stats', 'expenses-by-category', chartPeriod],
    queryFn: () => statsApi.getExpensesByCategory(chartPeriod),
  })

  const { data: salesByChannel = [] } = useQuery({
    queryKey: ['stats', 'sales-by-channel', chartPeriod],
    queryFn: () => statsApi.getSalesByChannel(chartPeriod),
  })

  const activeProjects = projects.filter(p => p.status === 'active')
  const printingPrinters = printers.filter(p => printerStates[p.id]?.status === 'printing')
  const idlePrinters = printers.filter(p =>
    printerStates[p.id]?.status === 'idle' || !printerStates[p.id]
  )
  const errorPrinters = printers.filter(p =>
    printerStates[p.id]?.status === 'error'
  )

  const formatCents = (cents: number) => `$${(cents / 100).toFixed(2)}`

  const lineChartData = useMemo(() => {
    const points = timeSeries?.points || []
    return {
      labels: points.map(p => p.date),
      datasets: [
        {
          label: 'Revenue',
          data: points.map(p => p.revenue),
          borderColor: '#10b981',
          backgroundColor: 'rgba(16, 185, 129, 0.1)',
          fill: true,
          tension: 0.3,
        },
        {
          label: 'Expenses',
          data: points.map(p => p.expenses),
          borderColor: '#ef4444',
          backgroundColor: 'rgba(239, 68, 68, 0.1)',
          fill: true,
          tension: 0.3,
        },
        {
          label: 'Profit',
          data: points.map(p => p.profit),
          borderColor: '#3b82f6',
          backgroundColor: 'rgba(59, 130, 246, 0.1)',
          fill: true,
          tension: 0.3,
        },
      ],
    }
  }, [timeSeries])

  const barChartData = useMemo(() => ({
    labels: expensesByCategory.map(c => c.category),
    datasets: [{
      label: 'Expenses',
      data: expensesByCategory.map(c => c.total),
      backgroundColor: '#ef4444',
      borderRadius: 4,
    }],
  }), [expensesByCategory])

  const doughnutData = useMemo(() => ({
    labels: salesByChannel.map(c => c.channel),
    datasets: [{
      data: salesByChannel.map(c => c.total),
      backgroundColor: salesByChannel.map(c => CHANNEL_COLORS[c.channel] || CHANNEL_COLORS.other),
      borderWidth: 0,
    }],
  }), [salesByChannel])

  const doughnutOptions = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      tooltip: {
        backgroundColor: '#1e293b',
        borderColor: '#334155',
        borderWidth: 1,
        titleColor: '#e2e8f0',
        bodyColor: '#e2e8f0',
        callbacks: {
          label: (ctx: { label?: string; parsed?: number }) => {
            const val = ctx.parsed ?? 0
            return `${ctx.label}: $${(val / 100).toFixed(0)}`
          },
        },
      },
      legend: {
        position: 'bottom' as const,
        labels: { color: '#94a3b8', padding: 16 },
      },
    },
  }

  return (
    <div className="p-4 sm:p-6 lg:p-8">
      {/* Header */}
      <div className="mb-6 lg:mb-8">
        <h1 className="text-2xl lg:text-3xl font-display font-bold text-surface-100">
          Dashboard
        </h1>
        <p className="text-surface-400 mt-1">
          Overview of your print farm
        </p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 lg:gap-4 mb-6 lg:mb-8">
        <StatCard
          icon={FolderKanban}
          label="Active Projects"
          value={activeProjects.length}
          color="text-blue-400"
        />
        <StatCard
          icon={Play}
          label="Printing Now"
          value={printingPrinters.length}
          color="text-emerald-400"
        />
        <StatCard
          icon={Clock}
          label="Idle Printers"
          value={idlePrinters.length}
          color="text-surface-400"
        />
        <StatCard
          icon={AlertCircle}
          label="Errors"
          value={errorPrinters.length}
          color="text-red-400"
        />
      </div>

      {/* Financial Summary */}
      {financials && (
        <div className="card p-6 mb-8">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-surface-100">
              Financial Overview
            </h2>
            <Link
              to="/expenses"
              className="text-sm text-accent-400 hover:text-accent-300"
            >
              Manage expenses →
            </Link>
          </div>

          <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 lg:gap-6">
            <div>
              <div className="flex items-center gap-2 text-sm text-surface-500 mb-1">
                <DollarSign className="h-4 w-4" />
                Total Revenue
              </div>
              <div className="text-2xl font-bold text-emerald-400">
                {formatCents(financials.total_sales_gross_cents)}
              </div>
              <div className="text-xs text-surface-500 mt-1">
                {financials.sales_count} sales
              </div>
            </div>

            <div>
              <div className="flex items-center gap-2 text-sm text-surface-500 mb-1">
                <Receipt className="h-4 w-4" />
                Total Expenses
              </div>
              <div className="text-2xl font-bold text-red-400">
                {formatCents(financials.total_expenses_cents)}
              </div>
              <div className="text-xs text-surface-500 mt-1">
                {financials.confirmed_expense_count} confirmed
                {financials.pending_expense_count > 0 && (
                  <span className="text-amber-400 ml-1">
                    ({financials.pending_expense_count} pending)
                  </span>
                )}
              </div>
            </div>

            <div>
              <div className="flex items-center gap-2 text-sm text-surface-500 mb-1">
                <Play className="h-4 w-4" />
                Material Used
              </div>
              <div className="text-2xl font-bold text-blue-400">
                {(financials.total_material_used_grams / 1000).toFixed(2)} kg
              </div>
              <div className="text-xs text-surface-500 mt-1">
                {formatCents(Math.round(financials.total_material_cost * 100))} cost
              </div>
            </div>

            <div>
              <div className="flex items-center gap-2 text-sm text-surface-500 mb-1">
                {financials.net_profit_cents >= 0 ? (
                  <TrendingUp className="h-4 w-4 text-emerald-400" />
                ) : (
                  <TrendingDown className="h-4 w-4 text-red-400" />
                )}
                Net Profit
              </div>
              <div
                className={cn(
                  'text-2xl font-bold',
                  financials.net_profit_cents >= 0
                    ? 'text-emerald-400'
                    : 'text-red-400'
                )}
              >
                {formatCents(financials.net_profit_cents)}
              </div>
              <div className="text-xs text-surface-500 mt-1">
                {financials.successful_print_count}/{financials.completed_print_count} prints successful
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Period Selector */}
      <div className="flex items-center gap-2 mb-6">
        {periodOptions.map((opt) => (
          <button
            key={opt.value}
            onClick={() => setChartPeriod(opt.value)}
            className={cn(
              'px-3 py-1.5 rounded-lg text-sm font-medium transition-colors',
              chartPeriod === opt.value
                ? 'bg-accent-500/20 text-accent-400'
                : 'text-surface-400 hover:text-surface-200 hover:bg-surface-800'
            )}
          >
            {opt.label}
          </button>
        ))}
      </div>

      {/* Revenue & Profit Chart */}
      {lineChartData.labels.length > 0 && (
        <div className="card p-6 mb-6">
          <h2 className="text-lg font-semibold text-surface-100 mb-4">
            Revenue & Profit
          </h2>
          <div className="h-[300px]">
            <Line data={lineChartData} options={chartDefaults} />
          </div>
        </div>
      )}

      {/* Two-column charts row */}
      {(expensesByCategory.length > 0 || salesByChannel.length > 0) && (
        <div className="grid grid-cols-1 xl:grid-cols-2 gap-6 mb-6">
          {/* Expenses by Category */}
          {expensesByCategory.length > 0 && (
            <div className="card p-6">
              <h2 className="text-lg font-semibold text-surface-100 mb-4">
                Expenses by Category
              </h2>
              <div className="h-[250px]">
                <Bar data={barChartData} options={chartDefaults} />
              </div>
            </div>
          )}

          {/* Sales by Channel */}
          {salesByChannel.length > 0 && (
            <div className="card p-6">
              <h2 className="text-lg font-semibold text-surface-100 mb-4">
                Sales by Channel
              </h2>
              <div className="h-[250px]">
                <Doughnut data={doughnutData} options={doughnutOptions} />
              </div>
            </div>
          )}
        </div>
      )}

      <div className="grid grid-cols-1 xl:grid-cols-2 gap-6">
        {/* Printer Status */}
        <div className="card p-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-surface-100">
              Printer Fleet
            </h2>
            <Link
              to="/printers"
              className="text-sm text-accent-400 hover:text-accent-300"
            >
              View all →
            </Link>
          </div>

          {printersLoading ? (
            <div className="text-surface-500">Loading...</div>
          ) : printers.length === 0 ? (
            <div className="text-center py-8 text-surface-500">
              <Printer className="h-12 w-12 mx-auto mb-3 opacity-50" />
              <p>No printers configured</p>
              <Link
                to="/printers"
                className="text-accent-400 hover:text-accent-300 text-sm"
              >
                Add your first printer
              </Link>
            </div>
          ) : (
            <div className="space-y-3">
              {printers.slice(0, 5).map((printer) => {
                const state = printerStates[printer.id]
                return (
                  <div
                    key={printer.id}
                    className="flex items-center justify-between p-3 rounded-lg bg-surface-800/50"
                  >
                    <div className="flex items-center gap-3">
                      <div className={cn(
                        'w-2 h-2 rounded-full',
                        state?.status === 'printing' ? 'bg-emerald-400 animate-pulse' :
                        state?.status === 'idle' ? 'bg-surface-500' :
                        state?.status === 'error' ? 'bg-red-400' :
                        'bg-surface-600'
                      )} />
                      <div>
                        <div className="font-medium text-surface-100">
                          {printer.name}
                        </div>
                        <div className="text-xs text-surface-500">
                          {printer.model || printer.manufacturer || 'Unknown model'}
                        </div>
                      </div>
                    </div>
                    <div className="text-right">
                      <span className={cn(
                        'badge',
                        getStatusBadge(state?.status || 'offline')
                      )}>
                        {state?.status || 'offline'}
                      </span>
                      {state?.status === 'printing' && state.progress > 0 && (
                        <div className="text-xs text-surface-400 mt-1">
                          {state.progress.toFixed(0)}%
                        </div>
                      )}
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </div>

        {/* Active Projects */}
        <div className="card p-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-surface-100">
              Active Projects
            </h2>
            <Link
              to="/projects"
              className="text-sm text-accent-400 hover:text-accent-300"
            >
              View all →
            </Link>
          </div>

          {projectsLoading ? (
            <div className="text-surface-500">Loading...</div>
          ) : activeProjects.length === 0 ? (
            <div className="text-center py-8 text-surface-500">
              <FolderKanban className="h-12 w-12 mx-auto mb-3 opacity-50" />
              <p>No active projects</p>
              <Link
                to="/projects"
                className="text-accent-400 hover:text-accent-300 text-sm"
              >
                Create a project
              </Link>
            </div>
          ) : (
            <div className="space-y-3">
              {activeProjects.slice(0, 5).map((project) => (
                <Link
                  key={project.id}
                  to={`/projects/${project.id}`}
                  className="block p-3 rounded-lg bg-surface-800/50 hover:bg-surface-800 transition-colors"
                >
                  <div className="flex items-center justify-between">
                    <div className="font-medium text-surface-100">
                      {project.name}
                    </div>
                    <span className={cn('badge', getStatusBadge(project.status))}>
                      {project.status}
                    </span>
                  </div>
                  {project.description && (
                    <p className="text-sm text-surface-500 mt-1 truncate">
                      {project.description}
                    </p>
                  )}
                </Link>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

function StatCard({
  icon: Icon,
  label,
  value,
  color
}: {
  icon: React.ElementType
  label: string
  value: number
  color: string
}) {
  return (
    <div className="card p-4">
      <div className="flex items-center gap-3">
        <div className={cn('p-2 rounded-lg bg-surface-800', color)}>
          <Icon className="h-5 w-5" />
        </div>
        <div>
          <div className="text-2xl font-bold text-surface-100">{value}</div>
          <div className="text-sm text-surface-500">{label}</div>
        </div>
      </div>
    </div>
  )
}
