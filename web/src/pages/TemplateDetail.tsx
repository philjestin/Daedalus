import { useState } from 'react'
import { useParams, Link, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  ArrowLeft,
  Plus,
  PlayCircle,
  FileCode,
  Trash2,
  Package,
  CheckSquare,
  Edit2,
  Save,
  X,
  Clock,
  Printer,
  Gauge
} from 'lucide-react'
import { templatesApi, designsApi, projectsApi, partsApi } from '../api/client'
import { cn, formatBytes } from '../lib/utils'
import type { Template, Design, Part, MaterialType, PrintProfile, PrinterConstraints } from '../types'
import RecipeMaterialEditor from '../components/RecipeMaterialEditor'
import RecipeSupplyEditor from '../components/RecipeSupplyEditor'
import PrinterConstraintsEditor from '../components/PrinterConstraintsEditor'
import RecipeCostCard from '../components/RecipeCostCard'
import TemplateAnalyticsCard from '../components/TemplateAnalyticsCard'

const materialTypes: { label: string; value: MaterialType }[] = [
  { label: 'PLA', value: 'pla' },
  { label: 'PETG', value: 'petg' },
  { label: 'ABS', value: 'abs' },
  { label: 'ASA', value: 'asa' },
  { label: 'TPU', value: 'tpu' },
]

const printProfiles: { label: string; value: PrintProfile; description: string }[] = [
  { label: 'Standard', value: 'standard', description: 'Balanced quality and speed' },
  { label: 'Detailed', value: 'detailed', description: 'Higher quality, slower' },
  { label: 'Fast', value: 'fast', description: 'Quick prints, lower quality' },
  { label: 'Strong', value: 'strong', description: 'Maximum strength' },
  { label: 'Custom', value: 'custom', description: 'Custom slicer settings' },
]

function formatDuration(seconds: number): string {
  if (!seconds || seconds === 0) return 'Not set'
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  if (hours === 0) return `${minutes}m`
  if (minutes === 0) return `${hours}h`
  return `${hours}h ${minutes}m`
}

function parseDuration(input: string): number {
  // Parse formats like "1h 30m", "1.5h", "90m", "5400"
  const hoursMatch = input.match(/(\d+(?:\.\d+)?)\s*h/i)
  const minutesMatch = input.match(/(\d+)\s*m/i)

  let seconds = 0
  if (hoursMatch) {
    seconds += parseFloat(hoursMatch[1]) * 3600
  }
  if (minutesMatch) {
    seconds += parseInt(minutesMatch[1]) * 60
  }

  // If no h or m, treat as raw seconds or minutes
  if (!hoursMatch && !minutesMatch) {
    const num = parseFloat(input)
    if (!isNaN(num)) {
      seconds = num > 1000 ? num : num * 60 // Assume minutes if < 1000
    }
  }

  return Math.round(seconds)
}

export default function TemplateDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const [isEditing, setIsEditing] = useState(false)
  const [showAddDesign, setShowAddDesign] = useState(false)
  const [showInstantiate, setShowInstantiate] = useState(false)

  const { data: template, isLoading } = useQuery({
    queryKey: ['templates', id],
    queryFn: () => templatesApi.get(id!),
  })

  const updateTemplate = useMutation({
    mutationFn: (data: Partial<Template>) => templatesApi.update(id!, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['templates', id] })
      setIsEditing(false)
    },
  })

  const deleteTemplate = useMutation({
    mutationFn: () => templatesApi.delete(id!),
    onSuccess: () => {
      navigate('/templates')
    },
  })

  const removeDesign = useMutation({
    mutationFn: (designId: string) => templatesApi.removeDesign(id!, designId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['templates', id] })
    },
  })

  const instantiate = useMutation({
    mutationFn: (opts: { order_quantity: number; customer_notes: string }) =>
      templatesApi.instantiate(id!, opts),
    onSuccess: (data) => {
      navigate(`/projects/${data.project.id}`)
    },
  })

  const [editForm, setEditForm] = useState<Partial<Template>>({})

  const startEditing = () => {
    setEditForm({
      name: template?.name,
      description: template?.description,
      sku: template?.sku,
      material_type: template?.material_type,
      quantity_per_order: template?.quantity_per_order,
      estimated_material_grams: template?.estimated_material_grams,
      is_active: template?.is_active,
      print_profile: template?.print_profile || 'standard',
      estimated_print_seconds: template?.estimated_print_seconds || 0,
      printer_constraints: template?.printer_constraints,
      labor_minutes: template?.labor_minutes || 0,
      sale_price_cents: template?.sale_price_cents || 0,
    })
    setIsEditing(true)
  }

  // Query for compatible printers
  const { data: compatiblePrinters = [] } = useQuery({
    queryKey: ['templates', id, 'compatible-printers'],
    queryFn: () => templatesApi.getCompatiblePrinters(id!),
    enabled: !!id,
  })

  const handleSave = async () => {
    await updateTemplate.mutateAsync(editForm)
  }

  const getMaterialBadge = (type: MaterialType) => {
    const colors: Record<MaterialType, string> = {
      pla: 'bg-emerald-500/20 text-emerald-400',
      petg: 'bg-blue-500/20 text-blue-400',
      abs: 'bg-amber-500/20 text-amber-400',
      asa: 'bg-orange-500/20 text-orange-400',
      tpu: 'bg-purple-500/20 text-purple-400',
      supply: 'bg-surface-500/20 text-surface-400',
    }
    return colors[type] || 'bg-surface-700 text-surface-300'
  }

  if (isLoading) {
    return (
      <div className="p-4 sm:p-6 lg:p-8">
        <div className="text-surface-500">Loading...</div>
      </div>
    )
  }

  if (!template) {
    return (
      <div className="p-4 sm:p-6 lg:p-8">
        <div className="text-surface-500">Template not found</div>
      </div>
    )
  }

  return (
    <div className="p-4 sm:p-6 lg:p-8">
      {/* Header */}
      <div className="mb-8">
        <Link
          to="/templates"
          className="inline-flex items-center text-sm text-surface-500 hover:text-surface-300 mb-4"
        >
          <ArrowLeft className="h-4 w-4 mr-1" />
          Back to Templates
        </Link>

        <div className="flex items-start justify-between">
          <div>
            {isEditing ? (
              <input
                type="text"
                value={editForm.name || ''}
                onChange={(e) => setEditForm({ ...editForm, name: e.target.value })}
                className="input text-2xl font-bold mb-2"
              />
            ) : (
              <h1 className="text-3xl font-display font-bold text-surface-100">
                {template.name}
              </h1>
            )}
            {isEditing ? (
              <textarea
                value={editForm.description || ''}
                onChange={(e) => setEditForm({ ...editForm, description: e.target.value })}
                className="input resize-none w-full"
                rows={2}
              />
            ) : (
              template.description && (
                <p className="text-surface-400 mt-1">{template.description}</p>
              )
            )}
          </div>

          <div className="flex items-center gap-2">
            {isEditing ? (
              <>
                <button
                  onClick={() => setIsEditing(false)}
                  className="btn btn-ghost"
                >
                  <X className="h-4 w-4 mr-2" />
                  Cancel
                </button>
                <button
                  onClick={handleSave}
                  disabled={updateTemplate.isPending}
                  className="btn btn-primary"
                >
                  <Save className="h-4 w-4 mr-2" />
                  {updateTemplate.isPending ? 'Saving...' : 'Save'}
                </button>
              </>
            ) : (
              <>
                <button onClick={startEditing} className="btn btn-secondary">
                  <Edit2 className="h-4 w-4 mr-2" />
                  Edit
                </button>
                <button
                  onClick={() => setShowInstantiate(true)}
                  className="btn btn-primary"
                  disabled={!template.designs || template.designs.length === 0}
                >
                  <PlayCircle className="h-4 w-4 mr-2" />
                  Instantiate
                </button>
              </>
            )}
          </div>
        </div>
      </div>

      {/* Properties Card */}
      <div className="card p-6 mb-6">
        <h2 className="text-lg font-semibold text-surface-100 mb-4">
          Template Properties
        </h2>

        {isEditing ? (
          <div className="grid grid-cols-3 gap-4">
            <div>
              <label className="block text-sm font-medium text-surface-300 mb-1">
                SKU
              </label>
              <input
                type="text"
                value={editForm.sku || ''}
                onChange={(e) => setEditForm({ ...editForm, sku: e.target.value })}
                className="input"
                placeholder="PROD-001"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-surface-300 mb-1">
                Material Type
              </label>
              <select
                value={editForm.material_type || ''}
                onChange={(e) =>
                  setEditForm({ ...editForm, material_type: e.target.value as MaterialType })
                }
                className="input"
              >
                {materialTypes.map((m) => (
                  <option key={m.value} value={m.value}>
                    {m.label}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-surface-300 mb-1">
                Quantity per Order
              </label>
              <input
                type="number"
                value={editForm.quantity_per_order || 1}
                onChange={(e) =>
                  setEditForm({ ...editForm, quantity_per_order: parseInt(e.target.value) })
                }
                className="input"
                min="1"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-surface-300 mb-1">
                Est. Material (grams)
              </label>
              <input
                type="number"
                value={editForm.estimated_material_grams || 0}
                onChange={(e) =>
                  setEditForm({
                    ...editForm,
                    estimated_material_grams: parseFloat(e.target.value),
                  })
                }
                className="input"
                min="0"
                step="0.01"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-surface-300 mb-1">
                Print Profile
              </label>
              <select
                value={editForm.print_profile || 'standard'}
                onChange={(e) =>
                  setEditForm({ ...editForm, print_profile: e.target.value as PrintProfile })
                }
                className="input"
              >
                {printProfiles.map((p) => (
                  <option key={p.value} value={p.value}>
                    {p.label}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-surface-300 mb-1">
                Est. Print Time
              </label>
              <input
                type="text"
                value={formatDuration(editForm.estimated_print_seconds || 0)}
                onChange={(e) =>
                  setEditForm({
                    ...editForm,
                    estimated_print_seconds: parseDuration(e.target.value),
                  })
                }
                className="input"
                placeholder="e.g., 1h 30m"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-surface-300 mb-1">
                Labor Time (minutes)
              </label>
              <input
                type="number"
                value={editForm.labor_minutes || 0}
                onChange={(e) =>
                  setEditForm({
                    ...editForm,
                    labor_minutes: parseInt(e.target.value) || 0,
                  })
                }
                className="input"
                min="0"
                placeholder="Manual labor time"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-surface-300 mb-1">
                Sale Price ($)
              </label>
              <input
                type="number"
                value={(editForm.sale_price_cents || 0) / 100}
                onChange={(e) =>
                  setEditForm({
                    ...editForm,
                    sale_price_cents: Math.round(parseFloat(e.target.value) * 100) || 0,
                  })
                }
                className="input"
                min="0"
                step="0.01"
                placeholder="0.00"
              />
            </div>
            <div className="flex items-center gap-2">
              <input
                type="checkbox"
                id="is_active"
                checked={editForm.is_active}
                onChange={(e) => setEditForm({ ...editForm, is_active: e.target.checked })}
                className="w-4 h-4"
              />
              <label htmlFor="is_active" className="text-sm text-surface-300">
                Template is active
              </label>
            </div>
          </div>
        ) : (
          <div className="grid grid-cols-6 gap-6">
            <div>
              <div className="text-sm text-surface-500 mb-1">SKU</div>
              <div className="font-medium text-surface-100">
                {template.sku || '—'}
              </div>
            </div>
            <div>
              <div className="text-sm text-surface-500 mb-1">Material</div>
              <span className={cn('badge', getMaterialBadge(template.material_type))}>
                {template.material_type.toUpperCase()}
              </span>
            </div>
            <div>
              <div className="text-sm text-surface-500 mb-1">Quantity/Order</div>
              <div className="flex items-center gap-2">
                <Package className="h-4 w-4 text-surface-400" />
                <span className="font-medium text-surface-100">
                  {template.quantity_per_order}
                </span>
              </div>
            </div>
            <div>
              <div className="text-sm text-surface-500 mb-1">Est. Material</div>
              <div className="font-medium text-surface-100">
                {template.estimated_material_grams > 0
                  ? `${template.estimated_material_grams}g`
                  : '—'}
              </div>
            </div>
            <div>
              <div className="text-sm text-surface-500 mb-1">Print Profile</div>
              <div className="flex items-center gap-2">
                <Gauge className="h-4 w-4 text-surface-400" />
                <span className="font-medium text-surface-100 capitalize">
                  {template.print_profile || 'standard'}
                </span>
              </div>
            </div>
            <div>
              <div className="text-sm text-surface-500 mb-1">Est. Print Time</div>
              <div className="flex items-center gap-2">
                <Clock className="h-4 w-4 text-surface-400" />
                <span className="font-medium text-surface-100">
                  {formatDuration(template.estimated_print_seconds)}
                </span>
              </div>
            </div>
            <div>
              <div className="text-sm text-surface-500 mb-1">Labor Time</div>
              <div className="font-medium text-surface-100">
                {template.labor_minutes > 0 ? `${template.labor_minutes} min` : '—'}
              </div>
            </div>
            <div>
              <div className="text-sm text-surface-500 mb-1">Sale Price</div>
              <div className="font-medium text-surface-100">
                {template.sale_price_cents > 0
                  ? `$${(template.sale_price_cents / 100).toFixed(2)}`
                  : '—'}
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Designs */}
      <div className="card p-6 mb-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-surface-100">
            Design Files ({template.designs?.length || 0})
          </h2>
          <button
            onClick={() => setShowAddDesign(true)}
            className="btn btn-secondary"
          >
            <Plus className="h-4 w-4 mr-2" />
            Add Design
          </button>
        </div>

        {!template.designs || template.designs.length === 0 ? (
          <div className="text-center py-8 text-surface-500">
            <FileCode className="h-12 w-12 mx-auto mb-3 opacity-50" />
            <p>No designs linked to this template</p>
            <p className="text-sm mt-1">
              Add designs from existing parts to use them in this template
            </p>
          </div>
        ) : (
          <div className="space-y-2">
            {template.designs.map((td) => (
              <div
                key={td.id}
                className="flex items-center justify-between p-4 rounded-lg bg-surface-800/50"
              >
                <div className="flex items-center gap-4">
                  <FileCode className="h-6 w-6 text-surface-500" />
                  <div>
                    <div className="font-medium text-surface-100">
                      {td.design?.file_name || 'Unknown design'}
                    </div>
                    <div className="text-sm text-surface-500 flex items-center gap-2">
                      <span>v{td.design?.version || '?'}</span>
                      {td.design && (
                        <>
                          <span>•</span>
                          <span>{formatBytes(td.design.file_size_bytes)}</span>
                        </>
                      )}
                      <span>•</span>
                      <span>Qty: {td.quantity}</span>
                      {td.is_primary && (
                        <>
                          <span>•</span>
                          <span className="text-accent-400">Primary</span>
                        </>
                      )}
                    </div>
                    {td.notes && (
                      <div className="text-xs text-surface-400 mt-1">{td.notes}</div>
                    )}
                  </div>
                </div>
                <button
                  onClick={() => removeDesign.mutate(td.design_id)}
                  disabled={removeDesign.isPending}
                  className="btn btn-ghost text-red-400 hover:text-red-300 hover:bg-red-500/10"
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Recipe Materials */}
      <div className="mb-6">
        <RecipeMaterialEditor
          templateId={id!}
          materials={template.materials || []}
        />
      </div>

      {/* Recipe Supplies */}
      <div className="mb-6">
        <RecipeSupplyEditor
          templateId={id!}
          supplies={template.supplies || []}
        />
      </div>

      {/* Two-column layout for constraints and cost */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
        {/* Printer Constraints */}
        <PrinterConstraintsEditor
          constraints={template.printer_constraints}
          onChange={(constraints: PrinterConstraints) => {
            updateTemplate.mutate({ printer_constraints: constraints })
          }}
          disabled={false}
        />

        {/* Cost Estimate */}
        <RecipeCostCard templateId={id!} />
      </div>

      {/* Performance Analytics */}
      <div className="mb-6">
        <TemplateAnalyticsCard templateId={id!} />
      </div>

      {/* Compatible Printers */}
      <div className="card p-6 mb-6">
        <div className="flex items-center gap-2 mb-4">
          <Printer className="h-5 w-5 text-surface-400" />
          <h2 className="text-lg font-semibold text-surface-100">
            Compatible Printers ({compatiblePrinters.length})
          </h2>
        </div>

        {compatiblePrinters.length === 0 ? (
          <div className="text-center py-4 text-surface-500">
            <Printer className="h-8 w-8 mx-auto mb-2 opacity-50" />
            <p className="text-sm">No printers match current constraints</p>
          </div>
        ) : (
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
            {compatiblePrinters.map((printer) => (
              <div
                key={printer.id}
                className="p-3 rounded-lg bg-surface-800/50 border border-surface-700"
              >
                <div className="font-medium text-surface-100">{printer.name}</div>
                <div className="text-sm text-surface-500">{printer.model}</div>
                {printer.build_volume && (
                  <div className="text-xs text-surface-600 mt-1">
                    {printer.build_volume.x} x {printer.build_volume.y} x {printer.build_volume.z}mm
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Post-Process Checklist */}
      <div className="card p-6 mb-6">
        <h2 className="text-lg font-semibold text-surface-100 mb-4">
          Post-Processing Checklist
        </h2>

        {!template.post_process_checklist || template.post_process_checklist.length === 0 ? (
          <div className="text-center py-4 text-surface-500">
            <CheckSquare className="h-8 w-8 mx-auto mb-2 opacity-50" />
            <p className="text-sm">No post-processing steps defined</p>
          </div>
        ) : (
          <div className="space-y-2">
            {template.post_process_checklist.map((item, idx) => (
              <div
                key={idx}
                className="flex items-center gap-3 p-3 rounded-lg bg-surface-800/50"
              >
                <CheckSquare className="h-4 w-4 text-surface-500" />
                <span className="text-surface-200">{item}</span>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Danger Zone */}
      <div className="card p-6 border-red-500/20">
        <h2 className="text-lg font-semibold text-red-400 mb-4">Danger Zone</h2>
        <p className="text-surface-400 mb-4">
          Deleting this template will not affect any projects already created from it.
        </p>
        <button
          onClick={() => {
            if (confirm('Are you sure you want to delete this template?')) {
              deleteTemplate.mutate()
            }
          }}
          disabled={deleteTemplate.isPending}
          className="btn bg-red-500/20 text-red-400 hover:bg-red-500/30"
        >
          <Trash2 className="h-4 w-4 mr-2" />
          {deleteTemplate.isPending ? 'Deleting...' : 'Delete Template'}
        </button>
      </div>

      {/* Add Design Modal */}
      {showAddDesign && (
        <AddDesignModal
          templateId={id!}
          onClose={() => setShowAddDesign(false)}
          onSuccess={() => {
            queryClient.invalidateQueries({ queryKey: ['templates', id] })
            setShowAddDesign(false)
          }}
        />
      )}

      {/* Instantiate Modal */}
      {showInstantiate && (
        <InstantiateModal
          template={template}
          onClose={() => setShowInstantiate(false)}
          onInstantiate={async (opts) => {
            await instantiate.mutateAsync(opts)
          }}
          isPending={instantiate.isPending}
        />
      )}
    </div>
  )
}

// Add Design Modal
function AddDesignModal({
  templateId,
  onClose,
  onSuccess,
}: {
  templateId: string
  onClose: () => void
  onSuccess: () => void
}) {
  const [selectedDesign, setSelectedDesign] = useState('')
  const [quantity, setQuantity] = useState(1)
  const [isPrimary, setIsPrimary] = useState(false)
  const [notes, setNotes] = useState('')

  // Get all projects and their parts/designs
  const { data: projects = [] } = useQuery({
    queryKey: ['projects'],
    queryFn: () => projectsApi.list(),
  })

  const [selectedProject, setSelectedProject] = useState('')
  const [selectedPart, setSelectedPart] = useState('')

  const { data: parts = [] } = useQuery({
    queryKey: ['parts', selectedProject],
    queryFn: () => partsApi.listByProject(selectedProject),
    enabled: !!selectedProject,
  })

  const { data: designs = [] } = useQuery({
    queryKey: ['designs', selectedPart],
    queryFn: () => designsApi.listByPart(selectedPart),
    enabled: !!selectedPart,
  })

  const addDesign = useMutation({
    mutationFn: () =>
      templatesApi.addDesign(templateId, {
        design_id: selectedDesign,
        quantity,
        is_primary: isPrimary,
        notes,
      }),
    onSuccess,
  })

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    await addDesign.mutateAsync()
  }

  return (
    <div
      className="fixed inset-0 bg-black/60 flex items-center justify-center z-50"
      onClick={(e) => e.target === e.currentTarget && onClose()}
    >
      <div className="card w-full max-w-lg p-6">
        <h2 className="text-xl font-semibold text-surface-100 mb-4">
          Add Design to Template
        </h2>
        <form onSubmit={handleSubmit}>
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-surface-300 mb-1">
                Select Project
              </label>
              <select
                value={selectedProject}
                onChange={(e) => {
                  setSelectedProject(e.target.value)
                  setSelectedPart('')
                  setSelectedDesign('')
                }}
                className="input"
              >
                <option value="">Choose a project...</option>
                {projects.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.name}
                  </option>
                ))}
              </select>
            </div>

            {selectedProject && (
              <div>
                <label className="block text-sm font-medium text-surface-300 mb-1">
                  Select Part
                </label>
                <select
                  value={selectedPart}
                  onChange={(e) => {
                    setSelectedPart(e.target.value)
                    setSelectedDesign('')
                  }}
                  className="input"
                >
                  <option value="">Choose a part...</option>
                  {parts.map((p: Part) => (
                    <option key={p.id} value={p.id}>
                      {p.name}
                    </option>
                  ))}
                </select>
              </div>
            )}

            {selectedPart && (
              <div>
                <label className="block text-sm font-medium text-surface-300 mb-1">
                  Select Design Version
                </label>
                <select
                  value={selectedDesign}
                  onChange={(e) => setSelectedDesign(e.target.value)}
                  className="input"
                >
                  <option value="">Choose a design...</option>
                  {designs.map((d: Design) => (
                    <option key={d.id} value={d.id}>
                      v{d.version} — {d.file_name} ({formatBytes(d.file_size_bytes)})
                    </option>
                  ))}
                </select>
              </div>
            )}

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-surface-300 mb-1">
                  Quantity per Instance
                </label>
                <input
                  type="number"
                  value={quantity}
                  onChange={(e) => setQuantity(parseInt(e.target.value) || 1)}
                  min="1"
                  className="input"
                />
              </div>
              <div className="flex items-center gap-2 pt-6">
                <input
                  type="checkbox"
                  id="is_primary"
                  checked={isPrimary}
                  onChange={(e) => setIsPrimary(e.target.checked)}
                  className="w-4 h-4"
                />
                <label htmlFor="is_primary" className="text-sm text-surface-300">
                  Primary design
                </label>
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-surface-300 mb-1">
                Notes (optional)
              </label>
              <input
                type="text"
                value={notes}
                onChange={(e) => setNotes(e.target.value)}
                className="input"
                placeholder="e.g., Main body, Support piece"
              />
            </div>
          </div>

          <div className="flex justify-end gap-3 mt-6">
            <button type="button" onClick={onClose} className="btn btn-ghost">
              Cancel
            </button>
            <button
              type="submit"
              disabled={!selectedDesign || addDesign.isPending}
              className="btn btn-primary"
            >
              {addDesign.isPending ? 'Adding...' : 'Add Design'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

// Instantiate Modal
function InstantiateModal({
  template,
  onClose,
  onInstantiate,
  isPending,
}: {
  template: Template
  onClose: () => void
  onInstantiate: (opts: { order_quantity: number; customer_notes: string }) => Promise<void>
  isPending: boolean
}) {
  const [orderQuantity, setOrderQuantity] = useState(1)
  const [customerNotes, setCustomerNotes] = useState('')

  const totalParts = template.quantity_per_order * orderQuantity
  const totalMaterial = template.estimated_material_grams * orderQuantity

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    await onInstantiate({
      order_quantity: orderQuantity,
      customer_notes: customerNotes,
    })
  }

  return (
    <div
      className="fixed inset-0 bg-black/60 flex items-center justify-center z-50"
      onClick={(e) => e.target === e.currentTarget && onClose()}
    >
      <div className="card w-full max-w-md p-6">
        <h2 className="text-xl font-semibold text-surface-100 mb-4">
          Create Project from Template
        </h2>
        <form onSubmit={handleSubmit}>
          <div className="space-y-4">
            <div className="p-4 rounded-lg bg-surface-800/50">
              <div className="text-sm text-surface-500 mb-1">Template</div>
              <div className="font-medium text-surface-100">{template.name}</div>
              <div className="text-sm text-surface-400 mt-1">
                {template.designs?.length || 0} design(s), {template.quantity_per_order} part(s) per order
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-surface-300 mb-1">
                Order Quantity
              </label>
              <input
                type="number"
                value={orderQuantity}
                onChange={(e) => setOrderQuantity(parseInt(e.target.value) || 1)}
                min="1"
                className="input"
              />
              <p className="text-xs text-surface-500 mt-1">
                Number of times to fulfill this template
              </p>
            </div>

            <div className="p-4 rounded-lg bg-accent-500/10 border border-accent-500/20">
              <div className="flex items-center justify-between text-sm">
                <span className="text-surface-300">Total Parts:</span>
                <span className="font-medium text-accent-400">{totalParts}</span>
              </div>
              {template.estimated_material_grams > 0 && (
                <div className="flex items-center justify-between text-sm mt-1">
                  <span className="text-surface-300">Est. Material:</span>
                  <span className="font-medium text-accent-400">{totalMaterial}g</span>
                </div>
              )}
            </div>

            <div>
              <label className="block text-sm font-medium text-surface-300 mb-1">
                Customer Notes (optional)
              </label>
              <textarea
                value={customerNotes}
                onChange={(e) => setCustomerNotes(e.target.value)}
                rows={2}
                className="input resize-none"
                placeholder="Special instructions, customizations, etc."
              />
            </div>
          </div>

          <div className="flex justify-end gap-3 mt-6">
            <button type="button" onClick={onClose} className="btn btn-ghost">
              Cancel
            </button>
            <button
              type="submit"
              disabled={isPending}
              className="btn btn-primary"
            >
              <PlayCircle className="h-4 w-4 mr-2" />
              {isPending ? 'Creating...' : 'Create Project'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}
