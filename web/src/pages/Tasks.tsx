import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { ListTodo, Plus, Clock, Play, CheckCircle, XCircle, ChevronRight, RefreshCw } from 'lucide-react'
import { tasksApi, projectsApi } from '../api/client'
import type { Task, TaskStatus, Project } from '../types'
import { cn } from '../lib/utils'

const statusConfig: Record<TaskStatus, { label: string; color: string; icon: React.ReactNode }> = {
  pending: { label: 'Pending', color: 'bg-surface-700 text-surface-300', icon: <Clock className="w-4 h-4" /> },
  in_progress: { label: 'In Progress', color: 'bg-blue-500/20 text-blue-400', icon: <Play className="w-4 h-4" /> },
  completed: { label: 'Completed', color: 'bg-emerald-500/20 text-emerald-400', icon: <CheckCircle className="w-4 h-4" /> },
  cancelled: { label: 'Cancelled', color: 'bg-red-500/20 text-red-400', icon: <XCircle className="w-4 h-4" /> },
}

export function Tasks() {
  const [tasks, setTasks] = useState<Task[]>([])
  const [projects, setProjects] = useState<Project[]>([])
  const [loading, setLoading] = useState(true)
  const [statusFilter, setStatusFilter] = useState<TaskStatus | ''>('')
  const [projectFilter, setProjectFilter] = useState('')
  const [showCreateModal, setShowCreateModal] = useState(false)

  useEffect(() => {
    loadTasks()
    loadProjects()
  }, [statusFilter, projectFilter])

  const loadTasks = async () => {
    setLoading(true)
    try {
      const filters: { status?: string; project_id?: string } = {}
      if (statusFilter) filters.status = statusFilter
      if (projectFilter) filters.project_id = projectFilter
      const data = await tasksApi.list(filters)
      setTasks(data || [])
    } catch (err) {
      console.error('Failed to load tasks:', err)
    } finally {
      setLoading(false)
    }
  }

  const loadProjects = async () => {
    try {
      const data = await projectsApi.list()
      setProjects(data || [])
    } catch (err) {
      console.error('Failed to load projects:', err)
    }
  }

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return '-'
    return new Date(dateStr).toLocaleDateString()
  }

  const getProgressColor = (progress: number) => {
    if (progress >= 100) return 'bg-emerald-500'
    if (progress >= 50) return 'bg-blue-500'
    if (progress > 0) return 'bg-amber-500'
    return 'bg-surface-600'
  }

  return (
    <div className="p-4 sm:p-6 lg:p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-display font-bold text-surface-100">Tasks</h1>
          <p className="text-surface-400 mt-1">Work instances for fulfilling orders</p>
        </div>
        <button
          onClick={() => setShowCreateModal(true)}
          className="btn btn-primary"
        >
          <Plus className="w-4 h-4 mr-2" />
          New Task
        </button>
      </div>

      {/* Filters */}
      <div className="flex gap-4 mb-6">
        <select
          value={statusFilter}
          onChange={e => setStatusFilter(e.target.value as TaskStatus | '')}
          className="input w-auto"
        >
          <option value="">All Statuses</option>
          {Object.entries(statusConfig).map(([status, config]) => (
            <option key={status} value={status}>
              {config.label}
            </option>
          ))}
        </select>

        <select
          value={projectFilter}
          onChange={e => setProjectFilter(e.target.value)}
          className="input w-auto"
        >
          <option value="">All Projects</option>
          {projects.map(project => (
            <option key={project.id} value={project.id}>
              {project.name}
            </option>
          ))}
        </select>

        <button
          onClick={loadTasks}
          className="btn btn-ghost"
        >
          <RefreshCw className="w-4 h-4" />
        </button>
      </div>

      {/* Task List */}
      {loading ? (
        <div className="text-surface-500">Loading tasks...</div>
      ) : tasks.length === 0 ? (
        <div className="text-center py-16">
          <ListTodo className="w-16 h-16 mx-auto text-surface-600 mb-4" />
          <h3 className="text-xl font-semibold text-surface-300 mb-2">No tasks found</h3>
          <p className="text-surface-500 mb-4">Tasks are created when processing order items</p>
        </div>
      ) : (
        <div className="card divide-y divide-surface-800">
          {tasks.map(task => {
            const config = statusConfig[task.status]
            const progress = task.progress ?? 0
            return (
              <Link
                key={task.id}
                to={`/tasks/${task.id}`}
                className="flex items-center justify-between p-4 hover:bg-surface-800/50 transition-colors"
              >
                <div className="flex items-center gap-4">
                  <div className={cn('p-2 rounded-lg', config.color)}>
                    {config.icon}
                  </div>
                  <div>
                    <h3 className="font-medium text-surface-100">{task.name}</h3>
                    <div className="text-sm text-surface-500">
                      Qty: {task.quantity} | Created: {formatDate(task.created_at)}
                      {task.order_id && <span> | Order linked</span>}
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-4">
                  {/* Progress bar */}
                  <div className="w-24">
                    <div className="flex items-center gap-2">
                      <div className="flex-1 h-2 bg-surface-700 rounded-full overflow-hidden">
                        <div
                          className={cn('h-full transition-all', getProgressColor(progress))}
                          style={{ width: `${progress}%` }}
                        />
                      </div>
                      <span className="text-xs text-surface-500 w-8">{Math.round(progress)}%</span>
                    </div>
                  </div>
                  <span className={cn('px-2.5 py-1 rounded-full text-xs font-medium', config.color)}>
                    {config.label}
                  </span>
                  <ChevronRight className="w-5 h-5 text-surface-600" />
                </div>
              </Link>
            )
          })}
        </div>
      )}

      {/* Create Modal */}
      {showCreateModal && (
        <CreateTaskModal
          projects={projects}
          onClose={() => setShowCreateModal(false)}
          onCreated={() => {
            setShowCreateModal(false)
            loadTasks()
          }}
        />
      )}
    </div>
  )
}

interface CreateTaskModalProps {
  projects: Project[]
  onClose: () => void
  onCreated: () => void
}

function CreateTaskModal({ projects, onClose, onCreated }: CreateTaskModalProps) {
  const [projectId, setProjectId] = useState('')
  const [name, setName] = useState('')
  const [quantity, setQuantity] = useState(1)
  const [notes, setNotes] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!projectId) {
      setError('Please select a project')
      return
    }
    if (!name.trim()) {
      setError('Please enter a task name')
      return
    }

    setSaving(true)
    setError('')
    try {
      await tasksApi.create({
        project_id: projectId,
        name: name.trim(),
        quantity,
        notes: notes.trim() || undefined,
      })
      onCreated()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create task')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
      <div className="card w-full max-w-md p-6">
        <h2 className="text-xl font-semibold text-surface-100 mb-4">Create Task</h2>
        <form onSubmit={handleSubmit}>
          {error && (
            <div className="mb-4 p-3 bg-red-500/20 text-red-400 rounded-lg text-sm">
              {error}
            </div>
          )}

          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-surface-300 mb-1">
                Project (Product)
              </label>
              <select
                value={projectId}
                onChange={e => setProjectId(e.target.value)}
                className="input"
                required
              >
                <option value="">Select a project...</option>
                {projects.map(project => (
                  <option key={project.id} value={project.id}>
                    {project.name} {project.sku ? `(${project.sku})` : ''}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-surface-300 mb-1">
                Task Name
              </label>
              <input
                type="text"
                value={name}
                onChange={e => setName(e.target.value)}
                className="input"
                placeholder="e.g., Widget production run"
                required
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-surface-300 mb-1">
                Quantity
              </label>
              <input
                type="number"
                value={quantity}
                onChange={e => setQuantity(Math.max(1, parseInt(e.target.value) || 1))}
                min={1}
                className="input"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-surface-300 mb-1">
                Notes (optional)
              </label>
              <textarea
                value={notes}
                onChange={e => setNotes(e.target.value)}
                rows={3}
                className="input resize-none"
                placeholder="Any additional notes..."
              />
            </div>
          </div>

          <div className="flex justify-end gap-3 mt-6">
            <button
              type="button"
              onClick={onClose}
              className="btn btn-ghost"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={saving}
              className="btn btn-primary"
            >
              {saving ? 'Creating...' : 'Create Task'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
