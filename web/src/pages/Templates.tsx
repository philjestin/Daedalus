import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Plus, FileStack, Package, Tag } from 'lucide-react'
import { templatesApi } from '../api/client'
import { cn } from '../lib/utils'
import type { Template, MaterialType } from '../types'

const materialTypes: { label: string; value: MaterialType }[] = [
  { label: 'PLA', value: 'pla' },
  { label: 'PETG', value: 'petg' },
  { label: 'ABS', value: 'abs' },
  { label: 'ASA', value: 'asa' },
  { label: 'TPU', value: 'tpu' },
]

export default function Templates() {
  const [showActive, setShowActive] = useState(false)
  const [showCreate, setShowCreate] = useState(false)
  const queryClient = useQueryClient()

  const { data: templates = [], isLoading } = useQuery({
    queryKey: ['templates', showActive],
    queryFn: () => templatesApi.list(showActive),
  })

  const createTemplate = useMutation({
    mutationFn: (data: Partial<Template>) => templatesApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['templates'] })
      setShowCreate(false)
    },
  })

  const handleCreate = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    const formData = new FormData(e.currentTarget)

    await createTemplate.mutateAsync({
      name: formData.get('name') as string,
      description: formData.get('description') as string,
      sku: formData.get('sku') as string || undefined,
      material_type: formData.get('material_type') as MaterialType,
      quantity_per_order: parseInt(formData.get('quantity_per_order') as string) || 1,
      estimated_material_grams: parseFloat(formData.get('estimated_material_grams') as string) || 0,
      tags: [],
      post_process_checklist: [],
    })
  }

  const getMaterialBadge = (type: MaterialType) => {
    const colors: Record<MaterialType, string> = {
      pla: 'bg-emerald-500/20 text-emerald-400',
      petg: 'bg-blue-500/20 text-blue-400',
      abs: 'bg-amber-500/20 text-amber-400',
      asa: 'bg-orange-500/20 text-orange-400',
      tpu: 'bg-purple-500/20 text-purple-400',
    }
    return colors[type] || 'bg-surface-700 text-surface-300'
  }

  return (
    <div className="p-8">
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-display font-bold text-surface-100">
            Templates
          </h1>
          <p className="text-surface-400 mt-1">
            Reusable project blueprints for order fulfillment
          </p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="btn btn-primary"
        >
          <Plus className="h-4 w-4 mr-2" />
          New Template
        </button>
      </div>

      {/* Filters */}
      <div className="flex gap-2 mb-6">
        <button
          onClick={() => setShowActive(false)}
          className={cn(
            'px-3 py-1.5 rounded-lg text-sm font-medium transition-colors',
            !showActive
              ? 'bg-accent-500/20 text-accent-400'
              : 'text-surface-400 hover:text-surface-100 hover:bg-surface-800'
          )}
        >
          All
        </button>
        <button
          onClick={() => setShowActive(true)}
          className={cn(
            'px-3 py-1.5 rounded-lg text-sm font-medium transition-colors',
            showActive
              ? 'bg-accent-500/20 text-accent-400'
              : 'text-surface-400 hover:text-surface-100 hover:bg-surface-800'
          )}
        >
          Active Only
        </button>
      </div>

      {/* Templates Grid */}
      {isLoading ? (
        <div className="text-surface-500">Loading...</div>
      ) : templates.length === 0 ? (
        <div className="text-center py-16">
          <FileStack className="h-16 w-16 mx-auto mb-4 text-surface-600" />
          <h3 className="text-xl font-semibold text-surface-300 mb-2">
            No templates found
          </h3>
          <p className="text-surface-500 mb-4">
            Create your first template to streamline order fulfillment
          </p>
          <button
            onClick={() => setShowCreate(true)}
            className="btn btn-primary"
          >
            <Plus className="h-4 w-4 mr-2" />
            Create Template
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-3 gap-4">
          {templates.map((template) => (
            <Link
              key={template.id}
              to={`/templates/${template.id}`}
              className="card p-5 hover:border-surface-700 transition-colors group"
            >
              <div className="flex items-start justify-between mb-3">
                <h3 className="font-semibold text-surface-100 group-hover:text-accent-400 transition-colors">
                  {template.name}
                </h3>
                <span className={cn(
                  'badge',
                  template.is_active
                    ? 'bg-emerald-500/20 text-emerald-400'
                    : 'bg-surface-700 text-surface-400'
                )}>
                  {template.is_active ? 'Active' : 'Inactive'}
                </span>
              </div>

              {template.description && (
                <p className="text-sm text-surface-500 mb-4 line-clamp-2">
                  {template.description}
                </p>
              )}

              <div className="flex flex-wrap gap-2 mb-4">
                <span className={cn('badge', getMaterialBadge(template.material_type))}>
                  {template.material_type.toUpperCase()}
                </span>
                {template.sku && (
                  <span className="badge bg-surface-700 text-surface-300">
                    SKU: {template.sku}
                  </span>
                )}
              </div>

              <div className="flex items-center gap-4 text-xs text-surface-500">
                <div className="flex items-center gap-1">
                  <Package className="h-3.5 w-3.5" />
                  {template.quantity_per_order} per order
                </div>
                {template.estimated_material_grams > 0 && (
                  <div className="flex items-center gap-1">
                    ~{template.estimated_material_grams}g
                  </div>
                )}
                {template.tags.length > 0 && (
                  <div className="flex items-center gap-1">
                    <Tag className="h-3.5 w-3.5" />
                    {template.tags.length}
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
          <div className="card w-full max-w-lg p-6">
            <h2 className="text-xl font-semibold text-surface-100 mb-4">
              Create Template
            </h2>
            <form onSubmit={handleCreate}>
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-surface-300 mb-1">
                    Template Name
                  </label>
                  <input
                    type="text"
                    name="name"
                    required
                    className="input"
                    placeholder="Product Name"
                    autoFocus
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-surface-300 mb-1">
                    Description
                  </label>
                  <textarea
                    name="description"
                    rows={2}
                    className="input resize-none"
                    placeholder="What does this template produce?"
                  />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-surface-300 mb-1">
                      SKU (Optional)
                    </label>
                    <input
                      type="text"
                      name="sku"
                      className="input"
                      placeholder="PROD-001"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-surface-300 mb-1">
                      Material Type
                    </label>
                    <select name="material_type" className="input" required>
                      {materialTypes.map((m) => (
                        <option key={m.value} value={m.value}>
                          {m.label}
                        </option>
                      ))}
                    </select>
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-surface-300 mb-1">
                      Quantity per Order
                    </label>
                    <input
                      type="number"
                      name="quantity_per_order"
                      min="1"
                      defaultValue="1"
                      className="input"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-surface-300 mb-1">
                      Est. Material (grams)
                    </label>
                    <input
                      type="number"
                      name="estimated_material_grams"
                      min="0"
                      step="0.01"
                      defaultValue="0"
                      className="input"
                    />
                  </div>
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
                  disabled={createTemplate.isPending}
                  className="btn btn-primary"
                >
                  {createTemplate.isPending ? 'Creating...' : 'Create Template'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}
