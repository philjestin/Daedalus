import { useQuery } from '@tanstack/react-query'
import { DollarSign, Clock, Package, RefreshCw, Wrench, TrendingUp } from 'lucide-react'
import { templatesApi } from '../api/client'
import { cn } from '../lib/utils'

interface RecipeCostCardProps {
  templateId: string
}

function formatCents(cents: number): string {
  return (cents / 100).toLocaleString('en-US', {
    style: 'currency',
    currency: 'USD',
  })
}

function formatDuration(seconds: number): string {
  if (seconds === 0) return 'Not set'
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  if (hours === 0) return `${minutes}m`
  if (minutes === 0) return `${hours}h`
  return `${hours}h ${minutes}m`
}

export default function RecipeCostCard({ templateId }: RecipeCostCardProps) {
  const { data: estimate, isLoading, error, refetch } = useQuery({
    queryKey: ['templates', templateId, 'cost-estimate'],
    queryFn: () => templatesApi.getCostEstimate(templateId),
  })

  if (isLoading) {
    return (
      <div className="card p-6">
        <div className="flex items-center gap-2 mb-4">
          <DollarSign className="h-5 w-5 text-surface-400" />
          <h2 className="text-lg font-semibold text-surface-100">Cost Estimate</h2>
        </div>
        <div className="text-surface-500 text-center py-4">Loading...</div>
      </div>
    )
  }

  if (error || !estimate) {
    return (
      <div className="card p-6">
        <div className="flex items-center gap-2 mb-4">
          <DollarSign className="h-5 w-5 text-surface-400" />
          <h2 className="text-lg font-semibold text-surface-100">Cost Estimate</h2>
        </div>
        <div className="text-surface-500 text-center py-4">
          Unable to calculate cost estimate
        </div>
      </div>
    )
  }

  return (
    <div className="card p-6">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <DollarSign className="h-5 w-5 text-surface-400" />
          <h2 className="text-lg font-semibold text-surface-100">Cost Estimate</h2>
        </div>
        <button
          onClick={() => refetch()}
          className="btn btn-ghost btn-sm"
          title="Refresh estimate"
        >
          <RefreshCw className="h-4 w-4" />
        </button>
      </div>

      {/* Total Cost */}
      <div className="mb-6 p-4 rounded-lg bg-accent-500/10 border border-accent-500/20">
        <div className="text-sm text-surface-400 mb-1">Total Cost</div>
        <div className="text-3xl font-bold text-accent-400">
          {formatCents(estimate.total_cost_cents)}
        </div>
      </div>

      {/* Cost Breakdown */}
      <div className="space-y-2">
        <div className="flex items-center justify-between p-3 rounded-lg bg-surface-800/50">
          <div className="flex items-center gap-3">
            <Package className="h-4 w-4 text-surface-400" />
            <span className="text-surface-200">Material Cost</span>
          </div>
          <span className="font-medium text-surface-100">
            {formatCents(estimate.material_cost_cents)}
          </span>
        </div>

        <div className="flex items-center justify-between p-3 rounded-lg bg-surface-800/50">
          <div className="flex items-center gap-3">
            <Clock className="h-4 w-4 text-surface-400" />
            <div>
              <span className="text-surface-200">Machine Time</span>
              <span className="text-sm text-surface-500 ml-2">
                ({formatDuration(estimate.estimated_print_time_seconds)})
              </span>
            </div>
          </div>
          <span className="font-medium text-surface-100">
            {formatCents(estimate.time_cost_cents)}
          </span>
        </div>

        {estimate.labor_minutes > 0 && (
          <div className="flex items-center justify-between p-3 rounded-lg bg-surface-800/50">
            <div className="flex items-center gap-3">
              <Wrench className="h-4 w-4 text-surface-400" />
              <div>
                <span className="text-surface-200">Labor</span>
                <span className="text-sm text-surface-500 ml-2">
                  ({estimate.labor_minutes} min)
                </span>
              </div>
            </div>
            <span className="font-medium text-surface-100">
              {formatCents(estimate.labor_cost_cents)}
            </span>
          </div>
        )}
      </div>

      {/* Margin Section */}
      {estimate.sale_price_cents > 0 && (
        <div className="mt-4 p-4 rounded-lg bg-surface-800/50 border border-surface-700">
          <div className="flex items-center gap-2 mb-3">
            <TrendingUp className="h-4 w-4 text-surface-400" />
            <span className="text-sm font-medium text-surface-300">Margin Analysis</span>
          </div>
          <div className="space-y-2">
            <div className="flex items-center justify-between text-sm">
              <span className="text-surface-400">Sale Price</span>
              <span className="text-surface-100">{formatCents(estimate.sale_price_cents)}</span>
            </div>
            <div className="flex items-center justify-between text-sm">
              <span className="text-surface-400">Total Cost</span>
              <span className="text-surface-100">-{formatCents(estimate.total_cost_cents)}</span>
            </div>
            <div className="border-t border-surface-700 my-2" />
            <div className="flex items-center justify-between">
              <span className="text-surface-300 font-medium">Gross Margin</span>
              <div className="text-right">
                <span className={cn(
                  'font-bold',
                  estimate.gross_margin_cents >= 0 ? 'text-emerald-400' : 'text-red-400'
                )}>
                  {formatCents(estimate.gross_margin_cents)}
                </span>
                <span className={cn(
                  'text-sm ml-2',
                  estimate.gross_margin_percent >= 0 ? 'text-emerald-500' : 'text-red-500'
                )}>
                  ({estimate.gross_margin_percent.toFixed(1)}%)
                </span>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Material Breakdown */}
      {estimate.material_breakdown && estimate.material_breakdown.length > 0 && (
        <div className="mt-6">
          <h3 className="text-sm font-medium text-surface-300 mb-3">Material Breakdown</h3>
          <div className="space-y-2">
            {estimate.material_breakdown.map((item, idx) => (
              <div
                key={idx}
                className="flex items-center justify-between text-sm p-2 rounded bg-surface-800/30"
              >
                <div className="flex items-center gap-2">
                  <span className="text-surface-300">{item.material_type.toUpperCase()}</span>
                  {item.color_name && (
                    <span className="text-surface-500">({item.color_name})</span>
                  )}
                  <span className="text-surface-500">{item.weight_grams}g</span>
                </div>
                <span className="text-surface-200">{formatCents(item.cost_cents)}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Rate Info */}
      <div className="mt-4 text-xs text-surface-500 text-center">
        Machine: {formatCents(estimate.hourly_rate_cents)}/hr
        {estimate.labor_rate_cents > 0 && (
          <span> • Labor: {formatCents(estimate.labor_rate_cents)}/hr</span>
        )}
      </div>
    </div>
  )
}
