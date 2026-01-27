import { useState } from 'react'
import { Link } from 'react-router-dom'
import { Plus, FolderKanban, Calendar, Tag } from 'lucide-react'
import { useProjects, useCreateProject } from '../hooks/useProjects'
import { cn, getStatusBadge, formatRelativeTime } from '../lib/utils'
import type { ProjectStatus } from '../types'

export default function Projects() {
  const [filter, setFilter] = useState<ProjectStatus | ''>('')
  const [showCreate, setShowCreate] = useState(false)
  
  const { data: projects = [], isLoading } = useProjects(filter || undefined)
  const createProject = useCreateProject()

  const handleCreate = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    const formData = new FormData(e.currentTarget)
    
    await createProject.mutateAsync({
      name: formData.get('name') as string,
      description: formData.get('description') as string,
      status: 'draft',
      tags: [],
    })
    
    setShowCreate(false)
  }

  const statusFilters: { label: string; value: ProjectStatus | '' }[] = [
    { label: 'All', value: '' },
    { label: 'Draft', value: 'draft' },
    { label: 'Active', value: 'active' },
    { label: 'Completed', value: 'completed' },
    { label: 'Archived', value: 'archived' },
  ]

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

      {/* Filters */}
      <div className="flex gap-2 mb-6">
        {statusFilters.map((f) => (
          <button
            key={f.value}
            onClick={() => setFilter(f.value)}
            className={cn(
              'px-3 py-1.5 rounded-lg text-sm font-medium transition-colors',
              filter === f.value
                ? 'bg-accent-500/20 text-accent-400'
                : 'text-surface-400 hover:text-surface-100 hover:bg-surface-800'
            )}
          >
            {f.label}
          </button>
        ))}
      </div>

      {/* Projects Grid */}
      {isLoading ? (
        <div className="text-surface-500">Loading...</div>
      ) : projects.length === 0 ? (
        <div className="text-center py-16">
          <FolderKanban className="h-16 w-16 mx-auto mb-4 text-surface-600" />
          <h3 className="text-xl font-semibold text-surface-300 mb-2">
            No projects found
          </h3>
          <p className="text-surface-500 mb-4">
            {filter ? 'Try a different filter' : 'Create your first project to get started'}
          </p>
          {!filter && (
            <button 
              onClick={() => setShowCreate(true)}
              className="btn btn-primary"
            >
              <Plus className="h-4 w-4 mr-2" />
              Create Project
            </button>
          )}
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
          {projects.map((project) => (
            <Link
              key={project.id}
              to={`/projects/${project.id}`}
              className="card p-5 hover:border-surface-700 transition-colors group"
            >
              <div className="flex items-start justify-between mb-3">
                <h3 className="font-semibold text-surface-100 group-hover:text-accent-400 transition-colors">
                  {project.name}
                </h3>
                <span className={cn('badge', getStatusBadge(project.status))}>
                  {project.status}
                </span>
              </div>
              
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

