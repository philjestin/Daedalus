import { useState, useMemo } from 'react'
import { Link } from 'react-router-dom'
import { Plus, FolderKanban, Calendar, Tag, LayoutGrid, Table, ArrowUpDown } from 'lucide-react'
import { useQueries } from '@tanstack/react-query'
import { useProjects, useCreateProject } from '../hooks/useProjects'
import { projectsApi } from '../api/client'
import { cn, formatRelativeTime } from '../lib/utils'
import type { ProjectSummary } from '../types'

type SortField = 'name' | 'updated_at' | 'revenue' | 'profit' | 'profit_per_hour' | 'success_rate' | 'print_time'
type SortDir = 'asc' | 'desc'

// SortHeader component - defined outside to avoid recreating during render
function SortHeader({
  field,
  currentField,
  onSort,
  children
}: {
  field: SortField
  currentField: SortField
  onSort: (field: SortField) => void
  children: React.ReactNode
}) {
  return (
    <button
      onClick={() => onSort(field)}
      className={cn(
        'flex items-center gap-1 text-xs font-medium uppercase tracking-wider',
        currentField === field ? 'text-accent-400' : 'text-surface-500 hover:text-surface-300'
      )}
    >
      {children}
      <ArrowUpDown className="h-3 w-3" />
    </button>
  )
}

export default function Projects() {
  const [showCreate, setShowCreate] = useState(false)
  const [viewMode, setViewMode] = useState<'grid' | 'table'>('grid')
  const [sortField, setSortField] = useState<SortField>('updated_at')
  const [sortDir, setSortDir] = useState<SortDir>('desc')

  const { data: projects = [], isLoading } = useProjects()
  const createProject = useCreateProject()

  // Fetch summaries for all projects when in table view
  const summaryQueries = useQueries({
    queries: viewMode === 'table' ? projects.map((p) => ({
      queryKey: ['project-summary', p.id],
      queryFn: () => projectsApi.getSummary(p.id),
      staleTime: 60_000,
    })) : [],
  })

  const summaryMap = useMemo(() => {
    const map: Record<string, ProjectSummary> = {}
    projects.forEach((p, i) => {
      if (viewMode === 'table' && summaryQueries[i]?.data) {
        map[p.id] = summaryQueries[i].data!
      }
    })
    return map
  }, [projects, summaryQueries, viewMode])

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortDir(sortDir === 'asc' ? 'desc' : 'asc')
    } else {
      setSortField(field)
      setSortDir('desc')
    }
  }

  const sortedProjects = useMemo(() => {
    if (viewMode !== 'table') return projects
    return [...projects].sort((a, b) => {
      const sa = summaryMap[a.id]
      const sb = summaryMap[b.id]
      let cmp = 0
      switch (sortField) {
        case 'name':
          cmp = a.name.localeCompare(b.name)
          break
        case 'updated_at':
          cmp = new Date(a.updated_at).getTime() - new Date(b.updated_at).getTime()
          break
        case 'revenue':
          cmp = (sa?.net_revenue_cents ?? 0) - (sb?.net_revenue_cents ?? 0)
          break
        case 'profit':
          cmp = (sa?.gross_profit_cents ?? 0) - (sb?.gross_profit_cents ?? 0)
          break
        case 'profit_per_hour':
          cmp = (sa?.profit_per_hour_cents ?? 0) - (sb?.profit_per_hour_cents ?? 0)
          break
        case 'success_rate':
          cmp = (sa?.success_rate ?? 0) - (sb?.success_rate ?? 0)
          break
        case 'print_time':
          cmp = (sa?.total_print_seconds ?? 0) - (sb?.total_print_seconds ?? 0)
          break
      }
      return sortDir === 'asc' ? cmp : -cmp
    })
  }, [projects, summaryMap, sortField, sortDir, viewMode])

  const handleCreate = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    const formData = new FormData(e.currentTarget)

    await createProject.mutateAsync({
      name: formData.get('name') as string,
      description: formData.get('description') as string,
      tags: [],
    })

    setShowCreate(false)
  }

  const formatCents = (cents: number) => {
    const negative = cents < 0
    const abs = Math.abs(cents)
    return `${negative ? '-' : ''}$${(abs / 100).toFixed(2)}`
  }

  const formatPrintTime = (seconds: number) => {
    if (seconds <= 0) return '-'
    const hours = Math.floor(seconds / 3600)
    const mins = Math.floor((seconds % 3600) / 60)
    if (hours > 0) return `${hours}h ${mins}m`
    return `${mins}m`
  }

  return (
    <div className="p-4 sm:p-6 lg:p-8">
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-display font-bold text-surface-100">
            Projects
          </h1>
          <p className="text-surface-400 mt-1">
            Manage your maker projects
          </p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="btn btn-primary"
        >
          <Plus className="h-4 w-4 mr-2" />
          New Project
        </button>
      </div>

      {/* View Toggle */}
      <div className="flex items-center justify-end mb-6">
        <div className="flex gap-1 bg-surface-800 rounded-lg p-1">
          <button
            onClick={() => setViewMode('grid')}
            className={cn(
              'p-1.5 rounded transition-colors',
              viewMode === 'grid' ? 'bg-surface-700 text-surface-100' : 'text-surface-500 hover:text-surface-300'
            )}
            title="Grid view"
          >
            <LayoutGrid className="h-4 w-4" />
          </button>
          <button
            onClick={() => setViewMode('table')}
            className={cn(
              'p-1.5 rounded transition-colors',
              viewMode === 'table' ? 'bg-surface-700 text-surface-100' : 'text-surface-500 hover:text-surface-300'
            )}
            title="Table view"
          >
            <Table className="h-4 w-4" />
          </button>
        </div>
      </div>

      {/* Projects */}
      {isLoading ? (
        <div className="text-surface-500">Loading...</div>
      ) : projects.length === 0 ? (
        <div className="text-center py-16">
          <FolderKanban className="h-16 w-16 mx-auto mb-4 text-surface-600" />
          <h3 className="text-xl font-semibold text-surface-300 mb-2">
            No projects found
          </h3>
          <p className="text-surface-500 mb-4">
            Create your first project to get started
          </p>
          <button
            onClick={() => setShowCreate(true)}
            className="btn btn-primary"
          >
            <Plus className="h-4 w-4 mr-2" />
            Create Project
          </button>
        </div>
      ) : viewMode === 'table' ? (
        /* Table View */
        <div className="card overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full">
              <thead>
                <tr className="border-b border-surface-800">
                  <th className="text-left p-3"><SortHeader field="name" currentField={sortField} onSort={handleSort}>Name</SortHeader></th>
                  <th className="text-right p-3"><SortHeader field="revenue" currentField={sortField} onSort={handleSort}>Revenue</SortHeader></th>
                  <th className="text-right p-3"><SortHeader field="profit" currentField={sortField} onSort={handleSort}>Profit</SortHeader></th>
                  <th className="text-right p-3"><SortHeader field="profit_per_hour" currentField={sortField} onSort={handleSort}>Profit/hr</SortHeader></th>
                  <th className="text-right p-3"><SortHeader field="success_rate" currentField={sortField} onSort={handleSort}>Success</SortHeader></th>
                  <th className="text-right p-3"><SortHeader field="print_time" currentField={sortField} onSort={handleSort}>Print Time</SortHeader></th>
                  <th className="text-right p-3"><SortHeader field="updated_at" currentField={sortField} onSort={handleSort}>Updated</SortHeader></th>
                </tr>
              </thead>
              <tbody>
                {sortedProjects.map((project) => {
                  const s = summaryMap[project.id]
                  return (
                    <tr key={project.id} className="border-b border-surface-800/50 hover:bg-surface-800/30">
                      <td className="p-3">
                        <Link
                          to={`/projects/${project.id}`}
                          className="font-medium text-surface-100 hover:text-accent-400 transition-colors"
                        >
                          {project.name}
                        </Link>
                      </td>
                      <td className="p-3 text-right text-sm text-surface-300">
                        {s ? formatCents(s.net_revenue_cents) : '-'}
                      </td>
                      <td className={cn('p-3 text-right text-sm font-medium', s && s.gross_profit_cents >= 0 ? 'text-emerald-400' : s ? 'text-red-400' : 'text-surface-500')}>
                        {s ? formatCents(s.gross_profit_cents) : '-'}
                      </td>
                      <td className={cn('p-3 text-right text-sm', s && s.profit_per_hour_cents >= 0 ? 'text-emerald-400' : s ? 'text-red-400' : 'text-surface-500')}>
                        {s ? formatCents(s.profit_per_hour_cents) : '-'}
                      </td>
                      <td className={cn('p-3 text-right text-sm', s ? (s.success_rate >= 90 ? 'text-emerald-400' : s.success_rate >= 70 ? 'text-amber-400' : 'text-red-400') : 'text-surface-500')}>
                        {s ? `${s.success_rate.toFixed(0)}%` : '-'}
                      </td>
                      <td className="p-3 text-right text-sm text-surface-300">
                        {s ? formatPrintTime(s.total_print_seconds) : '-'}
                      </td>
                      <td className="p-3 text-right text-xs text-surface-500">
                        {formatRelativeTime(project.updated_at)}
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        </div>
      ) : (
        /* Grid View */
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
          {projects.map((project) => (
            <Link
              key={project.id}
              to={`/projects/${project.id}`}
              className="card p-5 hover:border-surface-700 transition-colors group"
            >
              <h3 className="font-semibold text-surface-100 group-hover:text-accent-400 transition-colors mb-3">
                {project.name}
              </h3>

              {project.description && (
                <p className="text-sm text-surface-500 mb-4 line-clamp-2">
                  {project.description}
                </p>
              )}

              <div className="flex items-center gap-4 text-xs text-surface-500">
                <div className="flex items-center gap-1">
                  <Calendar className="h-3.5 w-3.5" />
                  {formatRelativeTime(project.updated_at)}
                </div>
                {project.tags.length > 0 && (
                  <div className="flex items-center gap-1">
                    <Tag className="h-3.5 w-3.5" />
                    {project.tags.length} tags
                  </div>
                )}
              </div>
            </Link>
          ))}
        </div>
      )}

      {/* Create Modal */}
      {showCreate && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
          <div className="card w-full max-w-md p-6">
            <h2 className="text-xl font-semibold text-surface-100 mb-4">
              Create Project
            </h2>
            <form onSubmit={handleCreate}>
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-surface-300 mb-1">
                    Project Name
                  </label>
                  <input
                    type="text"
                    name="name"
                    required
                    className="input"
                    placeholder="My Awesome Project"
                    autoFocus
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-surface-300 mb-1">
                    Description
                  </label>
                  <textarea
                    name="description"
                    rows={3}
                    className="input resize-none"
                    placeholder="What are you building?"
                  />
                </div>
              </div>
              <div className="flex justify-end gap-3 mt-6">
                <button
                  type="button"
                  onClick={() => setShowCreate(false)}
                  className="btn btn-ghost"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={createProject.isPending}
                  className="btn btn-primary"
                >
                  {createProject.isPending ? 'Creating...' : 'Create Project'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

