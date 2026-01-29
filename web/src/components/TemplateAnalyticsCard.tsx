import { useQuery } from '@tanstack/react-query'
import { BarChart3, TrendingUp, Clock, Package, CheckCircle, AlertCircle } from 'lucide-react'
import { templatesApi } from '../api/client'
import { cn } from '../lib/utils'

interface TemplateAnalyticsCardProps {
  templateId: string
}

function formatCents(cents: number): string {
  return (cents / 100).toLocaleString('en-US', {
    style: 'currency',
    currency: 'USD',
  })
}

function formatDuration(seconds: number): string {
  if (seconds === 0) return '—'
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  if (hours === 0) return `${minutes}m`
  if (minutes === 0) return `${hours}h`
  return `${hours}h ${minutes}m`
}

function percentDiff(estimated: number, actual: number): { value: number; better: boolean } {
  if (estimated === 0) return { value: 0, better: true }
  const diff = ((actual - estimated) / estimated) * 100
  // For cost/time/material: lower is better, so negative diff is good
  return { value: Math.abs(diff), better: diff <= 0 }
}

export default function TemplateAnalyticsCard({ templateId }: TemplateAnalyticsCardProps) {
  const { data: analytics, isLoading, error } = useQuery({
    queryKey: ['templates', templateId, 'analytics'],
    queryFn: () => templatesApi.getAnalytics(templateId),
  })

  if (isLoading) {
    return (
      <div className="card p-6">
        <div className="flex items-center gap-2 mb-4">
          <BarChart3 className="h-5 w-5 text-surface-400" />
          <h2 className="text-lg font-semibold text-surface-100">Performance Analytics</h2>
        </div>
        <div className="text-surface-500 text-center py-4">Loading...</div>
      </div>
    )
  }

  if (error || !analytics) {
    return (
      <div className="card p-6">
        <div className="flex items-center gap-2 mb-4">
          <BarChart3 className="h-5 w-5 text-surface-400" />
          <h2 className="text-lg font-semibold text-surface-100">Performance Analytics</h2>
        </div>
        <div className="text-surface-500 text-center py-4">
          Unable to load analytics
        </div>
      </div>
    )
  }

  if (analytics.project_count === 0) {
    return (
      <div className="card p-6">
        <div className="flex items-center gap-2 mb-4">
          <BarChart3 className="h-5 w-5 text-surface-400" />
          <h2 className="text-lg font-semibold text-surface-100">Performance Analytics</h2>
        </div>
        <div className="text-center py-8 text-surface-500">
          <BarChart3 className="h-12 w-12 mx-auto mb-3 opacity-50" />
          <p>No projects created from this template yet</p>
          <p className="text-sm mt-1">
            Analytics will appear after you instantiate projects
          </p>
        </div>
      </div>
    )
  }

  const printTimeDiff = percentDiff(analytics.estimated_print_seconds, analytics.avg_print_seconds)
  const materialDiff = percentDiff(analytics.estimated_material_grams, analytics.avg_material_grams)
  const costDiff = percentDiff(analytics.estimated_cost_cents, analytics.avg_unit_cost_cents)

  return (
    <div className="card p-6">
      <div className="flex items-center gap-2 mb-4">
        <BarChart3 className="h-5 w-5 text-surface-400" />
        <h2 className="text-lg font-semibold text-surface-100">Performance Analytics</h2>
        <span className="text-sm text-surface-500">
          ({analytics.project_count} project{analytics.project_count !== 1 ? 's' : ''})
        </span>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-6">
        <div className="p-3 rounded-lg bg-surface-800/50">
          <div className="text-xs text-surface-500 mb-1">Total Revenue</div>
          <div className="font-semibold text-surface-100">
            {formatCents(analytics.net_revenue_cents)}
          </div>
          <div className="text-xs text-surface-500">
            {analytics.total_sales_count} sale{analytics.total_sales_count !== 1 ? 's' : ''}
          </div>
        </div>

        <div className="p-3 rounded-lg bg-surface-800/50">
          <div className="text-xs text-surface-500 mb-1">Avg Margin</div>
          <div className={cn(
            'font-semibold',
            analytics.avg_gross_margin_percent >= 0 ? 'text-emerald-400' : 'text-red-400'
          )}>
            {analytics.avg_gross_margin_percent.toFixed(1)}%
          </div>
          <div className="text-xs text-surface-500">
            {formatCents(analytics.profit_per_hour_cents)}/hr
          </div>
        </div>

        <div className="p-3 rounded-lg bg-surface-800/50">
          <div className="text-xs text-surface-500 mb-1">Success Rate</div>
          <div className={cn(
            'font-semibold',
            analytics.success_rate >= 90 ? 'text-emerald-400' :
            analytics.success_rate >= 70 ? 'text-amber-400' : 'text-red-400'
          )}>
            {analytics.success_rate.toFixed(0)}%
          </div>
          <div className="text-xs text-surface-500">
            {analytics.total_completed}/{analytics.total_job_count} jobs
          </div>
        </div>

        <div className="p-3 rounded-lg bg-surface-800/50">
          <div className="text-xs text-surface-500 mb-1">Avg Print Time</div>
          <div className="font-semibold text-surface-100">
            {formatDuration(analytics.avg_print_seconds)}
          </div>
          <div className="text-xs text-surface-500">
            {analytics.avg_material_grams.toFixed(0)}g avg
          </div>
        </div>
      </div>

      {/* Estimated vs Actual Comparison */}
      {(analytics.estimated_print_seconds > 0 || analytics.estimated_material_grams > 0 || analytics.estimated_cost_cents > 0) && (
        <div className="border-t border-surface-700 pt-4">
          <h3 className="text-sm font-medium text-surface-300 mb-3 flex items-center gap-2">
            <TrendingUp className="h-4 w-4" />
            Estimated vs Actual
          </h3>
          <div className="space-y-2">
            {analytics.estimated_print_seconds > 0 && analytics.avg_print_seconds > 0 && (
              <div className="flex items-center justify-between text-sm">
                <div className="flex items-center gap-2">
                  <Clock className="h-4 w-4 text-surface-500" />
                  <span className="text-surface-400">Print Time</span>
                </div>
                <div className="flex items-center gap-3">
                  <span className="text-surface-500">
                    Est: {formatDuration(analytics.estimated_print_seconds)}
                  </span>
                  <span className="text-surface-300">
                    Actual: {formatDuration(analytics.avg_print_seconds)}
                  </span>
                  {printTimeDiff.value > 0 && (
                    <span className={cn(
                      'text-xs px-1.5 py-0.5 rounded',
                      printTimeDiff.better ? 'bg-emerald-500/20 text-emerald-400' : 'bg-red-500/20 text-red-400'
                    )}>
                      {printTimeDiff.better ? '-' : '+'}{printTimeDiff.value.toFixed(0)}%
                    </span>
                  )}
                </div>
              </div>
            )}

            {analytics.estimated_material_grams > 0 && analytics.avg_material_grams > 0 && (
              <div className="flex items-center justify-between text-sm">
                <div className="flex items-center gap-2">
                  <Package className="h-4 w-4 text-surface-500" />
                  <span className="text-surface-400">Material</span>
                </div>
                <div className="flex items-center gap-3">
                  <span className="text-surface-500">
                    Est: {analytics.estimated_material_grams.toFixed(0)}g
                  </span>
                  <span className="text-surface-300">
                    Actual: {analytics.avg_material_grams.toFixed(0)}g
                  </span>
                  {materialDiff.value > 0 && (
                    <span className={cn(
                      'text-xs px-1.5 py-0.5 rounded',
                      materialDiff.better ? 'bg-emerald-500/20 text-emerald-400' : 'bg-red-500/20 text-red-400'
                    )}>
                      {materialDiff.better ? '-' : '+'}{materialDiff.value.toFixed(0)}%
                    </span>
                  )}
                </div>
              </div>
            )}

            {analytics.estimated_cost_cents > 0 && analytics.avg_unit_cost_cents > 0 && (
              <div className="flex items-center justify-between text-sm">
                <div className="flex items-center gap-2">
                  <TrendingUp className="h-4 w-4 text-surface-500" />
                  <span className="text-surface-400">Unit Cost</span>
                </div>
                <div className="flex items-center gap-3">
                  <span className="text-surface-500">
                    Est: {formatCents(analytics.estimated_cost_cents)}
                  </span>
                  <span className="text-surface-300">
                    Actual: {formatCents(analytics.avg_unit_cost_cents)}
                  </span>
                  {costDiff.value > 0 && (
                    <span className={cn(
                      'text-xs px-1.5 py-0.5 rounded',
                      costDiff.better ? 'bg-emerald-500/20 text-emerald-400' : 'bg-red-500/20 text-red-400'
                    )}>
                      {costDiff.better ? '-' : '+'}{costDiff.value.toFixed(0)}%
                    </span>
                  )}
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Quick Stats Footer */}
      <div className="mt-4 pt-4 border-t border-surface-700 flex items-center justify-between text-xs text-surface-500">
        <div className="flex items-center gap-4">
          <span className="flex items-center gap-1">
            <CheckCircle className="h-3 w-3 text-emerald-500" />
            {analytics.total_completed} completed
          </span>
          {analytics.total_failed > 0 && (
            <span className="flex items-center gap-1">
              <AlertCircle className="h-3 w-3 text-red-500" />
              {analytics.total_failed} failed
            </span>
          )}
        </div>
        <span>
          Total: {formatCents(analytics.total_gross_profit_cents)} profit
        </span>
      </div>
    </div>
  )
}
