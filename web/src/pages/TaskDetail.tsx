import { useEffect, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { ArrowLeft, Clock, Play, CheckCircle, XCircle, Package, Printer, AlertCircle, CalendarDays, Square, CheckSquare } from 'lucide-react'
import { tasksApi } from '../api/client'
import type { Task, TaskStatus, PrintJob, TaskChecklistItem } from '../types'
import { cn } from '../lib/utils'

const statusConfig: Record<TaskStatus, { label: string; color: string; bgColor: string; icon: React.ReactNode }> = {
  pending: { label: 'Pending', color: 'text-surface-300', bgColor: 'bg-surface-700', icon: <Clock className="w-5 h-5" /> },
  in_progress: { label: 'In Progress', color: 'text-blue-400', bgColor: 'bg-blue-500/20', icon: <Play className="w-5 h-5" /> },
  completed: { label: 'Completed', color: 'text-emerald-400', bgColor: 'bg-emerald-500/20', icon: <CheckCircle className="w-5 h-5" /> },
  cancelled: { label: 'Cancelled', color: 'text-red-400', bgColor: 'bg-red-500/20', icon: <XCircle className="w-5 h-5" /> },
}

const jobStatusColors: Record<string, string> = {
  queued: 'bg-surface-700 text-surface-300',
  assigned: 'bg-amber-500/20 text-amber-400',
  uploaded: 'bg-blue-500/20 text-blue-400',
  sending: 'bg-blue-500/20 text-blue-400',
  printing: 'bg-indigo-500/20 text-indigo-400',
  paused: 'bg-amber-500/20 text-amber-400',
  completed: 'bg-emerald-500/20 text-emerald-400',
  failed: 'bg-red-500/20 text-red-400',
  cancelled: 'bg-surface-700 text-surface-400',
}

export function TaskDetail() {
  const { id } = useParams<{ id: string }>()
  const [task, setTask] = useState<Task | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [actionLoading, setActionLoading] = useState(false)

  useEffect(() => {
    if (id) {
      loadTask()
    }
  }, [id])

  const loadTask = async () => {
    if (!id) return
    setLoading(true)
    try {
      const data = await tasksApi.get(id)
      setTask(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load task')
    } finally {
      setLoading(false)
    }
  }

  const handleStart = async () => {
    if (!id) return
    setActionLoading(true)
    try {
      await tasksApi.start(id)
      loadTask()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start task')
    } finally {
      setActionLoading(false)
    }
  }

  const handleComplete = async () => {
    if (!id) return
    setActionLoading(true)
    try {
      await tasksApi.complete(id)
      loadTask()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to complete task')
    } finally {
      setActionLoading(false)
    }
  }

  const handleCancel = async () => {
    if (!id || !confirm('Are you sure you want to cancel this task?')) return
    setActionLoading(true)
    try {
      await tasksApi.cancel(id)
      loadTask()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to cancel task')
    } finally {
      setActionLoading(false)
    }
  }

  const handleToggleChecklist = async (item: TaskChecklistItem) => {
    if (!id) return
    try {
      await tasksApi.toggleChecklistItem(id, item.id, !item.completed)
      loadTask()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update checklist')
    }
  }

  const handlePrintFromChecklist = async (itemId: string) => {
    if (!id) return
    setActionLoading(true)
    try {
      await tasksApi.printFromChecklist(id, itemId)
      loadTask()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create print job')
    } finally {
      setActionLoading(false)
    }
  }

  const handleRegenerateChecklist = async () => {
    if (!id) return
    setActionLoading(true)
    try {
      await tasksApi.regenerateChecklist(id)
      loadTask()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to generate checklist')
    } finally {
      setActionLoading(false)
    }
  }

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return '-'
    return new Date(dateStr).toLocaleString()
  }

  const getProgressColor = (progress: number) => {
    if (progress >= 100) return 'bg-emerald-500'
    if (progress >= 50) return 'bg-blue-500'
    if (progress > 0) return 'bg-amber-500'
    return 'bg-surface-600'
  }

  if (loading) {
    return (
      <div className="p-4 sm:p-6 lg:p-8">
        <div className="text-surface-500">Loading task...</div>
      </div>
    )
  }

  if (error || !task) {
    return (
      <div className="p-4 sm:p-6 lg:p-8">
        <div className="text-center py-16">
          <AlertCircle className="w-16 h-16 mx-auto text-red-400 mb-4" />
          <h3 className="text-xl font-semibold text-surface-100 mb-2">Error</h3>
          <p className="text-surface-500">{error || 'Task not found'}</p>
          <Link to="/tasks" className="mt-4 inline-block text-accent-400 hover:text-accent-300">
            Back to Tasks
          </Link>
        </div>
      </div>
    )
  }

  const config = statusConfig[task.status]
  const progress = task.progress ?? 0
  const jobs = task.jobs ?? []
  const checklist = task.checklist_items ?? []
  const completedJobs = jobs.filter(j => j.status === 'completed').length
  const failedJobs = jobs.filter(j => j.status === 'failed').length
  const completedChecklist = checklist.filter(i => i.completed).length

  return (
    <div className="p-4 sm:p-6 lg:p-8">
      {/* Header */}
      <div className="mb-6">
        <Link to="/tasks" className="inline-flex items-center gap-2 text-surface-400 hover:text-surface-100 mb-4 transition-colors">
          <ArrowLeft className="w-4 h-4" />
          Back to Tasks
        </Link>
        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-2xl font-bold text-surface-100">{task.name}</h1>
            <p className="text-surface-400">
              Quantity: {task.quantity} | Created: {formatDate(task.created_at)}
            </p>
          </div>
          <div className={cn('inline-flex items-center gap-2 px-3 py-1.5 rounded-full', config.bgColor, config.color)}>
            {config.icon}
            <span className="font-medium">{config.label}</span>
          </div>
        </div>
      </div>

      {/* Progress */}
      <div className="card p-6 mb-6">
        <h2 className="text-lg font-semibold text-surface-100 mb-4">Progress</h2>
        <div className="mb-4">
          <div className="flex justify-between text-sm mb-1">
            <span className="text-surface-400">Overall Progress</span>
            <span className="font-medium text-surface-100">{Math.round(progress)}%</span>
          </div>
          <div className="h-3 bg-surface-700 rounded-full overflow-hidden">
            <div
              className={cn('h-full transition-all', getProgressColor(progress))}
              style={{ width: `${progress}%` }}
            />
          </div>
        </div>
        <div className="grid grid-cols-3 gap-4 text-center">
          <div>
            <div className="text-2xl font-bold text-surface-100">{jobs.length}</div>
            <div className="text-sm text-surface-500">Total Jobs</div>
          </div>
          <div>
            <div className="text-2xl font-bold text-emerald-400">{completedJobs}</div>
            <div className="text-sm text-surface-500">Completed</div>
          </div>
          <div>
            <div className="text-2xl font-bold text-red-400">{failedJobs}</div>
            <div className="text-sm text-surface-500">Failed</div>
          </div>
        </div>
      </div>

      {/* Checklist */}
      <div className="card p-6 mb-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-surface-100">Checklist</h2>
          {checklist.length > 0 ? (
            <span className="text-sm text-surface-400">
              {completedChecklist}/{checklist.length} completed
            </span>
          ) : (
            <button
              onClick={handleRegenerateChecklist}
              disabled={actionLoading}
              className="text-sm text-accent-400 hover:text-accent-300 transition-colors disabled:opacity-50"
            >
              Generate from parts
            </button>
          )}
        </div>
        {checklist.length > 0 ? (
          <>
            <div className="mb-4">
              <div className="h-2 bg-surface-700 rounded-full overflow-hidden">
                <div
                  className={cn(
                    'h-full transition-all',
                    completedChecklist === checklist.length ? 'bg-emerald-500' : 'bg-accent-500'
                  )}
                  style={{ width: `${(completedChecklist / checklist.length) * 100}%` }}
                />
              </div>
            </div>
            <div className="space-y-2">
              {checklist.map((item) => (
                <div
                  key={item.id}
                  className="flex items-center gap-3 p-3 bg-surface-800 rounded-lg hover:bg-surface-700 transition-colors"
                >
                  <button
                    onClick={() => handleToggleChecklist(item)}
                    className="flex items-center gap-3 flex-1 text-left"
                  >
                    {item.completed ? (
                      <CheckSquare className="w-5 h-5 text-emerald-400 flex-shrink-0" />
                    ) : (
                      <Square className="w-5 h-5 text-surface-500 flex-shrink-0" />
                    )}
                    <span className={cn(
                      'font-medium',
                      item.completed ? 'text-surface-500 line-through' : 'text-surface-100'
                    )}>
                      {item.name}
                    </span>
                    {item.completed_at && (
                      <span className="ml-auto text-xs text-surface-600">
                        {formatDate(item.completed_at)}
                      </span>
                    )}
                  </button>
                  {item.part_id && !item.completed && task.status !== 'completed' && task.status !== 'cancelled' && (
                    <button
                      onClick={() => handlePrintFromChecklist(item.id)}
                      disabled={actionLoading}
                      className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-accent-400 border border-accent-500/50 rounded-lg hover:bg-accent-500/10 transition-colors disabled:opacity-50 flex-shrink-0"
                    >
                      <Printer className="w-3.5 h-3.5" />
                      Print
                    </button>
                  )}
                </div>
              ))}
            </div>
          </>
        ) : (
          <p className="text-sm text-surface-500">
            No checklist items. Click "Generate from parts" to create from project parts.
          </p>
        )}
      </div>

      {/* Actions */}
      {task.status !== 'completed' && task.status !== 'cancelled' && (
        <div className="card p-6 mb-6">
          <h2 className="text-lg font-semibold text-surface-100 mb-4">Actions</h2>
          <div className="flex gap-3">
            {task.status === 'pending' && (
              <button
                onClick={handleStart}
                disabled={actionLoading}
                className="btn btn-primary"
              >
                <Play className="w-4 h-4 mr-2" />
                Start Task
              </button>
            )}
            {task.status === 'in_progress' && (
              <button
                onClick={handleComplete}
                disabled={actionLoading}
                className="inline-flex items-center gap-2 px-4 py-2 bg-emerald-600 text-white rounded-lg hover:bg-emerald-700 transition-colors disabled:opacity-50"
              >
                <CheckCircle className="w-4 h-4" />
                Mark Complete
              </button>
            )}
            <button
              onClick={handleCancel}
              disabled={actionLoading}
              className="inline-flex items-center gap-2 px-4 py-2 text-red-400 border border-red-500/50 rounded-lg hover:bg-red-500/10 transition-colors disabled:opacity-50"
            >
              <XCircle className="w-4 h-4" />
              Cancel
            </button>
          </div>
        </div>
      )}

      {/* Project Info */}
      {task.project && (
        <div className="card p-6 mb-6">
          <h2 className="text-lg font-semibold text-surface-100 mb-4">Project</h2>
          <Link
            to={`/projects/${task.project.id}`}
            className="flex items-center gap-3 p-3 bg-surface-800 rounded-lg hover:bg-surface-700 transition-colors"
          >
            <Package className="w-8 h-8 text-accent-400" />
            <div>
              <div className="font-medium text-surface-100">{task.project.name}</div>
              {task.project.sku && (
                <div className="text-sm text-surface-500">SKU: {task.project.sku}</div>
              )}
            </div>
          </Link>
        </div>
      )}

      {/* Jobs */}
      <div className="card p-6 mb-6">
        <h2 className="text-lg font-semibold text-surface-100 mb-4">Print Jobs</h2>
        {jobs.length === 0 ? (
          <div className="text-center py-8">
            <Printer className="w-10 h-10 mx-auto mb-3 text-surface-600" />
            <p className="text-surface-500">No print jobs assigned to this task yet</p>
          </div>
        ) : (
          <div className="space-y-3">
            {jobs.map((job: PrintJob) => (
              <Link
                key={job.id}
                to={`/print-jobs/${job.id}`}
                className="flex items-center justify-between p-4 bg-surface-800 rounded-lg hover:bg-surface-700 transition-colors"
              >
                <div className="flex items-center gap-3">
                  <Printer className="w-5 h-5 text-surface-500" />
                  <div>
                    <div className="font-medium text-surface-100">Job {job.id.substring(0, 8)}</div>
                    <div className="text-sm text-surface-500">
                      Progress: {Math.round(job.progress)}%
                      {job.started_at && ` | Started: ${formatDate(job.started_at)}`}
                    </div>
                  </div>
                </div>
                <span className={cn('px-2.5 py-1 rounded-full text-xs font-medium', jobStatusColors[job.status] || 'bg-surface-700 text-surface-400')}>
                  {job.status}
                </span>
              </Link>
            ))}
          </div>
        )}
      </div>

      {/* Details */}
      <div className="card p-6">
        <h2 className="text-lg font-semibold text-surface-100 mb-4">Details</h2>
        <dl className="grid grid-cols-2 gap-4">
          <div>
            <dt className="text-sm text-surface-500">Task ID</dt>
            <dd className="font-mono text-sm text-surface-300">{task.id}</dd>
          </div>
          <div>
            <dt className="text-sm text-surface-500">Status</dt>
            <dd className="text-surface-100">{config.label}</dd>
          </div>
          <div>
            <dt className="text-sm text-surface-500">Created</dt>
            <dd className="text-surface-300">{formatDate(task.created_at)}</dd>
          </div>
          <div>
            <dt className="text-sm text-surface-500">Updated</dt>
            <dd className="text-surface-300">{formatDate(task.updated_at)}</dd>
          </div>
          {task.pickup_date && (
            <div>
              <dt className="text-sm text-surface-500">Pickup / Ship Date</dt>
              <dd className="flex items-center gap-2 text-surface-300">
                <CalendarDays className="w-4 h-4 text-accent-400" />
                {new Date(task.pickup_date).toLocaleDateString()}
              </dd>
            </div>
          )}
          {task.started_at && (
            <div>
              <dt className="text-sm text-surface-500">Started</dt>
              <dd className="text-surface-300">{formatDate(task.started_at)}</dd>
            </div>
          )}
          {task.completed_at && (
            <div>
              <dt className="text-sm text-surface-500">Completed</dt>
              <dd className="text-surface-300">{formatDate(task.completed_at)}</dd>
            </div>
          )}
          {task.order_id && (
            <div className="col-span-2">
              <dt className="text-sm text-surface-500">Order</dt>
              <dd>
                <Link to={`/orders/${task.order_id}`} className="text-accent-400 hover:text-accent-300">
                  View Order
                </Link>
              </dd>
            </div>
          )}
          {task.notes && (
            <div className="col-span-2">
              <dt className="text-sm text-surface-500">Notes</dt>
              <dd className="whitespace-pre-wrap text-surface-300">{task.notes}</dd>
            </div>
          )}
        </dl>
      </div>
    </div>
  )
}
