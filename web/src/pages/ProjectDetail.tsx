import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import {
  ArrowLeft,
  Plus,
  Upload,
  Printer,
  Play,
  FileCode,
  Box,
  AlertTriangle,
  CheckCircle,
  XCircle,
  Star,
  History,
  Clock,
  RefreshCw,
  BarChart3,
  DollarSign,
  TrendingUp,
  Timer,
  ExternalLink,
  X,
  Scale,
  Trash2,
  ShoppingCart,
} from 'lucide-react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useProject, useParts, useCreatePart } from '../hooks/useProjects'
import { usePrinters, usePrinterStates } from '../hooks/usePrinters'
import { useSpoolsWithMaterials } from '../hooks/useMaterials'
import { designsApi, printJobsApi, projectsApi, partsApi, suppliesApi, materialsApi } from '../api/client'
import { cn, getStatusBadge, formatBytes, formatRelativeTime } from '../lib/utils'
import { FailureModal } from '../components/FailureModal'
import { ExpandableJobEvents } from '../components/JobEventTimeline'
import { Tooltip } from '../components/Tooltip'
import type { Design, Part, Material, PrintJob, ProjectSummary, ProjectSupply } from '../types'

export default function ProjectDetail() {
  const { id } = useParams<{ id: string }>()
  const queryClient = useQueryClient()
  
  const { data: project, isLoading: projectLoading } = useProject(id!)
  const { data: parts = [], isLoading: partsLoading } = useParts(id!)
  const { data: printers = [] } = usePrinters()
  const { data: printerStates = {} } = usePrinterStates()
  
  const createPart = useCreatePart()
  
  const [showAddPart, setShowAddPart] = useState(false)
  const [selectedPart, setSelectedPart] = useState<Part | null>(null)
  const [showUpload, setShowUpload] = useState(false)
  const [showSendToPrinter, setShowSendToPrinter] = useState<Design | null>(null)
  const [showOutcomeCapture, setShowOutcomeCapture] = useState<PrintJob | null>(null)
  const [showFailureModal, setShowFailureModal] = useState<PrintJob | null>(null)
  const [activeTab, setActiveTab] = useState<'parts' | 'history' | 'analytics'>('parts')
  const [partFile, setPartFile] = useState<File | null>(null)
  const [partFileNotes, setPartFileNotes] = useState('')

  const { data: projectSummary } = useQuery({
    queryKey: ['project-summary', id],
    queryFn: () => projectsApi.getSummary(id!),
    enabled: !!id,
  })

  const handleAddPart = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    const formData = new FormData(e.currentTarget)

    await createPart.mutateAsync({
      projectId: id!,
      data: {
        name: formData.get('name') as string,
        description: formData.get('description') as string,
        quantity: parseInt(formData.get('quantity') as string) || 1,
      },
      file: partFile || undefined,
      notes: partFileNotes || undefined,
    })

    setShowAddPart(false)
    setPartFile(null)
    setPartFileNotes('')
  }

  if (projectLoading) {
    return (
      <div className="p-4 sm:p-6 lg:p-8">
        <div className="text-surface-500">Loading...</div>
      </div>
    )
  }

  if (!project) {
    return (
      <div className="p-4 sm:p-6 lg:p-8">
        <div className="text-surface-500">Project not found</div>
      </div>
    )
  }

  return (
    <div className="p-4 sm:p-6 lg:p-8">
      {/* Header */}
      <div className="mb-8">
        <Link 
          to="/projects" 
          className="inline-flex items-center text-sm text-surface-500 hover:text-surface-300 mb-4"
        >
          <ArrowLeft className="h-4 w-4 mr-1" />
          Back to Projects
        </Link>
        
        <div>
          <h1 className="text-3xl font-display font-bold text-surface-100">
            {project.name}
          </h1>
          {project.description && (
            <p className="text-surface-400 mt-1">{project.description}</p>
          )}
        </div>
      </div>

      {/* Printer Control Panel */}
      <div className="card p-4 mb-6">
        <div className="flex items-center justify-between mb-3">
          <h2 className="font-semibold text-surface-100 flex items-center gap-2">
            <Printer className="h-5 w-5 text-accent-500" />
            Printer Fleet
          </h2>
        </div>
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
          {printers.length === 0 ? (
            <div className="col-span-4 text-center py-4 text-surface-500">
              <Link to="/printers" className="text-accent-400 hover:text-accent-300">
                Add printers to get started
              </Link>
            </div>
          ) : (
            printers.map((printer) => {
              const state = printerStates[printer.id]
              return (
                <div 
                  key={printer.id}
                  className="p-3 rounded-lg bg-surface-800/50 border border-surface-700"
                >
                  <div className="flex items-center gap-2 mb-2">
                    <div className={cn(
                      'w-2 h-2 rounded-full',
                      state?.status === 'printing' ? 'bg-emerald-400 animate-pulse' :
                      state?.status === 'idle' ? 'bg-blue-400' :
                      state?.status === 'error' ? 'bg-red-400' :
                      'bg-surface-600'
                    )} />
                    <span className="font-medium text-surface-100 text-sm truncate">
                      {printer.name}
                    </span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className={cn('badge text-xs', getStatusBadge(state?.status || 'offline'))}>
                      {state?.status || 'offline'}
                    </span>
                    {state?.status === 'printing' && (
                      <span className="text-xs text-surface-400">
                        {state.progress.toFixed(0)}%
                      </span>
                    )}
                  </div>
                  {state?.status === 'printing' && (
                    <div className="mt-2 h-1 bg-surface-700 rounded-full overflow-hidden">
                      <div 
                        className="h-full bg-emerald-500 transition-all"
                        style={{ width: `${state.progress}%` }}
                      />
                    </div>
                  )}
                </div>
              )
            })
          )}
        </div>
      </div>

      {/* Project Quick Stats */}
      {projectSummary && (
        <ProjectQuickStats summary={projectSummary} />
      )}

      {/* Tab Navigation */}
      <div className="flex items-center gap-4 border-b border-surface-800 mb-4">
        <button
          onClick={() => setActiveTab('parts')}
          className={cn(
            'flex items-center gap-2 px-4 py-2 border-b-2 -mb-px transition-colors',
            activeTab === 'parts'
              ? 'border-accent-500 text-accent-400'
              : 'border-transparent text-surface-400 hover:text-surface-200'
          )}
        >
          <Box className="h-4 w-4" />
          Parts
        </button>
        <button
          onClick={() => setActiveTab('history')}
          className={cn(
            'flex items-center gap-2 px-4 py-2 border-b-2 -mb-px transition-colors',
            activeTab === 'history'
              ? 'border-accent-500 text-accent-400'
              : 'border-transparent text-surface-400 hover:text-surface-200'
          )}
        >
          <History className="h-4 w-4" />
          Print History
        </button>
        <button
          onClick={() => setActiveTab('analytics')}
          className={cn(
            'flex items-center gap-2 px-4 py-2 border-b-2 -mb-px transition-colors',
            activeTab === 'analytics'
              ? 'border-accent-500 text-accent-400'
              : 'border-transparent text-surface-400 hover:text-surface-200'
          )}
        >
          <BarChart3 className="h-4 w-4" />
          Analytics
        </button>
      </div>

      {/* Parts Tab */}
      {activeTab === 'parts' && (
        <>
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-xl font-semibold text-surface-100">Parts</h2>
            <button
              onClick={() => setShowAddPart(true)}
              className="btn btn-secondary"
            >
              <Plus className="h-4 w-4 mr-2" />
              Add Part
            </button>
          </div>

          {partsLoading ? (
            <div className="text-surface-500">Loading parts...</div>
          ) : parts.length === 0 ? (
            <div className="card p-8 text-center">
              <Box className="h-12 w-12 mx-auto mb-3 text-surface-600" />
              <h3 className="text-lg font-medium text-surface-300 mb-2">
                No parts yet
              </h3>
              <p className="text-surface-500 mb-4">
                Add parts to start organizing your project
              </p>
              <button
                onClick={() => setShowAddPart(true)}
                className="btn btn-primary"
              >
                <Plus className="h-4 w-4 mr-2" />
                Add First Part
              </button>
            </div>
          ) : (
            <div className="space-y-4">
              {parts.map((part) => (
                <PartCard
                  key={part.id}
                  part={part}
                  onUpload={() => {
                    setSelectedPart(part)
                    setShowUpload(true)
                  }}
                  onSendToPrinter={(design) => setShowSendToPrinter(design)}
                  onDelete={async () => {
                    if (!confirm(`Delete part "${part.name}"? This cannot be undone.`)) return
                    try {
                      await partsApi.delete(part.id)
                      queryClient.invalidateQueries({ queryKey: ['parts', id] })
                    } catch (err) {
                      console.error('Failed to delete part:', err)
                    }
                  }}
                />
              ))}
            </div>
          )}

          {/* Supplies Section */}
          <SuppliesSection projectId={id!} />
        </>
      )}

      {/* History Tab */}
      {activeTab === 'history' && (
        <PrintHistoryTab
          projectId={id!}
          parts={parts}
          printers={printers}
          onRecordOutcome={(job) => setShowOutcomeCapture(job)}
          onHandleFailure={(job) => setShowFailureModal(job)}
        />
      )}

      {/* Analytics Tab */}
      {activeTab === 'analytics' && (
        <ProjectAnalyticsTab summary={projectSummary} />
      )}

      {/* Add Part Modal */}
      {showAddPart && (
        <Modal title="Add Part" onClose={() => setShowAddPart(false)}>
          <form onSubmit={handleAddPart}>
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-surface-300 mb-1">
                  Part Name
                </label>
                <input
                  type="text"
                  name="name"
                  required
                  className="input"
                  placeholder="e.g., Top Case, Motor Mount"
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
                  placeholder="Optional description"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-surface-300 mb-1">
                  Quantity
                </label>
                <input
                  type="number"
                  name="quantity"
                  min="1"
                  defaultValue="1"
                  className="input w-24"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-surface-300 mb-2">
                  Design File (optional)
                </label>
                <div
                  className={cn(
                    'border-2 border-dashed rounded-lg p-6 text-center transition-colors',
                    partFile ? 'border-accent-500 bg-accent-500/5' : 'border-surface-700 hover:border-surface-600'
                  )}
                >
                  <input
                    type="file"
                    accept=".stl,.3mf,.gcode"
                    onChange={(e) => setPartFile(e.target.files?.[0] || null)}
                    className="hidden"
                    id="part-file-upload"
                  />
                  <label htmlFor="part-file-upload" className="cursor-pointer">
                    {partFile ? (
                      <div>
                        <FileCode className="h-8 w-8 mx-auto mb-1 text-accent-500" />
                        <p className="text-surface-100 font-medium text-sm">{partFile.name}</p>
                        <p className="text-surface-500 text-xs">{formatBytes(partFile.size)}</p>
                      </div>
                    ) : (
                      <div>
                        <Upload className="h-8 w-8 mx-auto mb-1 text-surface-500" />
                        <p className="text-surface-400 text-sm">Attach a design file</p>
                        <p className="text-surface-500 text-xs">STL, 3MF, or GCODE</p>
                      </div>
                    )}
                  </label>
                  {partFile && (
                    <button
                      type="button"
                      onClick={(e) => { e.stopPropagation(); setPartFile(null); setPartFileNotes('') }}
                      className="mt-2 text-xs text-surface-400 hover:text-surface-200 flex items-center gap-1 mx-auto"
                    >
                      <X className="h-3 w-3" />
                      Remove
                    </button>
                  )}
                </div>
              </div>
              {partFile && (
                <div>
                  <label className="block text-sm font-medium text-surface-300 mb-1">
                    File Notes
                  </label>
                  <textarea
                    value={partFileNotes}
                    onChange={(e) => setPartFileNotes(e.target.value)}
                    rows={2}
                    className="input resize-none"
                    placeholder="Optional notes about this design"
                  />
                </div>
              )}
            </div>
            <div className="flex justify-end gap-3 mt-6">
              <button
                type="button"
                onClick={() => { setShowAddPart(false); setPartFile(null); setPartFileNotes('') }}
                className="btn btn-ghost"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={createPart.isPending}
                className="btn btn-primary"
              >
                {createPart.isPending ? 'Adding...' : 'Add Part'}
              </button>
            </div>
          </form>
        </Modal>
      )}

      {/* Upload Design Modal */}
      {showUpload && selectedPart && (
        <UploadDesignModal
          part={selectedPart}
          onClose={() => {
            setShowUpload(false)
            setSelectedPart(null)
          }}
          onSuccess={() => {
            queryClient.invalidateQueries({ queryKey: ['designs', selectedPart.id] })
            setShowUpload(false)
            setSelectedPart(null)
          }}
        />
      )}

      {/* Send to Printer Modal */}
      {showSendToPrinter && (
        <SendToPrinterModal
          design={showSendToPrinter}
          printers={printers}
          printerStates={printerStates}
          onClose={() => setShowSendToPrinter(null)}
        />
      )}

      {/* Outcome Capture Modal */}
      {showOutcomeCapture && (
        <OutcomeCaptureModal
          job={showOutcomeCapture}
          onClose={() => setShowOutcomeCapture(null)}
          onSuccess={() => {
            queryClient.invalidateQueries({ queryKey: ['print-jobs'] })
            queryClient.invalidateQueries({ queryKey: ['spools'] })
            setShowOutcomeCapture(null)
          }}
        />
      )}

      {/* Failure Modal */}
      {showFailureModal && (
        <FailureModal
          job={showFailureModal}
          printers={printers}
          onClose={() => setShowFailureModal(null)}
          onRetry={() => {
            queryClient.invalidateQueries({ queryKey: ['print-jobs'] })
            setShowFailureModal(null)
          }}
          onScrap={() => {
            queryClient.invalidateQueries({ queryKey: ['print-jobs'] })
            setShowFailureModal(null)
          }}
        />
      )}
    </div>
  )
}

function formatPrintTime(seconds: number): string {
  const h = Math.floor(seconds / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (h > 0) return `${h}h ${m}m`
  return `${m}m`
}

// Part Card Component
function PartCard({
  part,
  onUpload,
  onSendToPrinter,
  onDelete,
}: {
  part: Part
  onUpload: () => void
  onSendToPrinter: (design: Design) => void
  onDelete: () => void
}) {
  const { data: designs = [] } = useQuery({
    queryKey: ['designs', part.id],
    queryFn: () => designsApi.listByPart(part.id),
  })

  return (
    <div className="card p-5">
      <div className="flex items-start justify-between mb-4">
        <div>
          <div className="flex items-center gap-2">
            <h3 className="font-semibold text-surface-100">{part.name}</h3>
            <span className="text-sm text-surface-500">×{part.quantity}</span>
          </div>
          {part.description && (
            <p className="text-sm text-surface-500 mt-1">{part.description}</p>
          )}
        </div>
        <div className="flex items-center gap-2">
          <span className={cn('badge', getStatusBadge(part.status))}>
            {part.status}
          </span>
          <button
            onClick={onDelete}
            className="p-1.5 rounded-lg text-surface-500 hover:text-red-400 hover:bg-red-500/10 transition-colors"
            title="Delete part"
          >
            <Trash2 className="h-4 w-4" />
          </button>
        </div>
      </div>

      {/* Designs */}
      <div className="border-t border-surface-800 pt-4">
        <div className="flex items-center justify-between mb-3">
          <span className="text-sm font-medium text-surface-400">
            Designs ({designs.length} version{designs.length !== 1 ? 's' : ''})
          </span>
          <button 
            onClick={onUpload}
            className="btn btn-ghost text-xs py-1 px-2"
          >
            <Upload className="h-3.5 w-3.5 mr-1" />
            Upload
          </button>
        </div>

        {designs.length === 0 ? (
          <div className="text-center py-4 text-surface-500 text-sm">
            No designs uploaded yet
          </div>
        ) : (
          <div className="space-y-2">
            {designs.slice(0, 3).map((design) => (
              <div 
                key={design.id}
                className="flex items-center justify-between p-3 rounded-lg bg-surface-800/50"
              >
                <div className="flex items-center gap-3">
                  <FileCode className="h-5 w-5 text-surface-500" />
                  <div>
                    <div className="text-sm font-medium text-surface-200">
                      v{design.version} — {design.file_name}
                    </div>
                    <div className="text-xs text-surface-500">
                      {formatBytes(design.file_size_bytes)} • {formatRelativeTime(design.created_at)}
                    </div>
                    {design.slice_profile && (
                      <div className="flex items-center gap-2 mt-1 flex-wrap">
                        <span className="inline-flex items-center gap-1 text-xs bg-surface-700/60 text-surface-300 rounded px-1.5 py-0.5">
                          <Scale className="h-3 w-3" />
                          {Math.round(design.slice_profile.weight_grams)}g
                        </span>
                        <span className="inline-flex items-center gap-1 text-xs bg-surface-700/60 text-surface-300 rounded px-1.5 py-0.5">
                          <Timer className="h-3 w-3" />
                          {formatPrintTime(design.slice_profile.print_time_seconds)}
                        </span>
                        {design.slice_profile.filaments.map((f, i) => (
                          <span
                            key={i}
                            className="inline-flex items-center gap-1.5 text-xs bg-surface-700/60 text-surface-300 rounded px-1.5 py-0.5"
                          >
                            <span
                              className="inline-block h-2.5 w-2.5 rounded-full border border-surface-600"
                              style={{ backgroundColor: f.color }}
                            />
                            {f.type} · {Math.round(f.used_meters)}m · {Math.round(f.used_grams)}g
                          </span>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  {design.file_type === '3mf' && (
                    <button
                      onClick={() => {
                        designsApi.openExternal(design.id, 'BambuStudio').catch((err) => {
                          alert('Failed to open Bambu Studio: ' + err.message)
                        })
                      }}
                      className="btn btn-ghost text-xs py-1.5 px-3"
                      title="Open in Bambu Studio"
                    >
                      <ExternalLink className="h-3.5 w-3.5 mr-1" />
                      Bambu Studio
                    </button>
                  )}
                  <button
                    onClick={() => onSendToPrinter(design)}
                    className="btn btn-primary text-xs py-1.5 px-3"
                  >
                    <Play className="h-3.5 w-3.5 mr-1" />
                    Print
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

// Supplies Section Component
function SuppliesSection({ projectId }: { projectId: string }) {
  const queryClient = useQueryClient()
  const [showAddForm, setShowAddForm] = useState(false)
  const [addMode, setAddMode] = useState<'catalog' | 'manual'>('catalog')
  const [selectedMaterialId, setSelectedMaterialId] = useState('')
  const [newName, setNewName] = useState('')
  const [newCost, setNewCost] = useState('')
  const [newQuantity, setNewQuantity] = useState('1')

  const { data: supplies = [] } = useQuery({
    queryKey: ['project-supplies', projectId],
    queryFn: () => suppliesApi.listByProject(projectId),
  })

  const { data: supplyMaterials = [] } = useQuery({
    queryKey: ['materials', 'supply'],
    queryFn: () => materialsApi.listByType('supply'),
  })

  const handleCatalogAdd = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!selectedMaterialId) return
    try {
      await suppliesApi.create(projectId, {
        name: '', // auto-populated from material by backend
        unit_cost_cents: 0, // auto-populated from material by backend
        quantity: parseInt(newQuantity) || 1,
        material_id: selectedMaterialId,
      })
      queryClient.invalidateQueries({ queryKey: ['project-supplies', projectId] })
      queryClient.invalidateQueries({ queryKey: ['project-summary', projectId] })
      setSelectedMaterialId('')
      setNewQuantity('1')
      setShowAddForm(false)
    } catch (err) {
      console.error('Failed to add supply from catalog:', err)
    }
  }

  const handleManualAdd = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newName.trim()) return
    try {
      await suppliesApi.create(projectId, {
        name: newName.trim(),
        unit_cost_cents: Math.round(parseFloat(newCost || '0') * 100),
        quantity: parseInt(newQuantity) || 1,
      })
      queryClient.invalidateQueries({ queryKey: ['project-supplies', projectId] })
      queryClient.invalidateQueries({ queryKey: ['project-summary', projectId] })
      setNewName('')
      setNewCost('')
      setNewQuantity('1')
      setShowAddForm(false)
    } catch (err) {
      console.error('Failed to add supply:', err)
    }
  }

  const [confirmDeleteSupplyId, setConfirmDeleteSupplyId] = useState<string | null>(null)

  const handleDelete = async (supply: ProjectSupply) => {
    if (confirmDeleteSupplyId !== supply.id) {
      setConfirmDeleteSupplyId(supply.id)
      return
    }
    setConfirmDeleteSupplyId(null)
    try {
      await suppliesApi.delete(supply.id)
      queryClient.invalidateQueries({ queryKey: ['project-supplies', projectId] })
      queryClient.invalidateQueries({ queryKey: ['project-summary', projectId] })
    } catch (err) {
      console.error('Failed to delete supply:', err)
    }
  }

  // Auto-fill cost when selecting a catalog material
  const handleMaterialSelect = (materialId: string) => {
    setSelectedMaterialId(materialId)
    const mat = supplyMaterials.find((m) => m.id === materialId)
    if (mat) {
      setNewCost((mat.cost_per_kg).toFixed(2)) // cost_per_kg is repurposed as per-unit $ for supplies
    }
  }

  const totalCents = supplies.reduce((sum, s) => sum + s.unit_cost_cents * s.quantity, 0)

  return (
    <div className="mt-8">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-xl font-semibold text-surface-100 flex items-center gap-2">
          <ShoppingCart className="h-5 w-5 text-surface-400" />
          Supplies
        </h2>
        <button
          onClick={() => setShowAddForm(!showAddForm)}
          className="btn btn-secondary"
        >
          <Plus className="h-4 w-4 mr-2" />
          Add Supply
        </button>
      </div>

      {showAddForm && (
        <div className="card p-4 mb-4">
          {supplyMaterials.length > 0 && (
            <div className="flex gap-2 mb-3">
              <button
                type="button"
                onClick={() => setAddMode('catalog')}
                className={cn(
                  'text-xs px-3 py-1 rounded-full transition-colors',
                  addMode === 'catalog'
                    ? 'bg-accent-500/20 text-accent-400 border border-accent-500'
                    : 'bg-surface-800 text-surface-400 border border-surface-700 hover:text-surface-200'
                )}
              >
                From Catalog
              </button>
              <button
                type="button"
                onClick={() => setAddMode('manual')}
                className={cn(
                  'text-xs px-3 py-1 rounded-full transition-colors',
                  addMode === 'manual'
                    ? 'bg-accent-500/20 text-accent-400 border border-accent-500'
                    : 'bg-surface-800 text-surface-400 border border-surface-700 hover:text-surface-200'
                )}
              >
                Manual Entry
              </button>
            </div>
          )}

          {addMode === 'catalog' && supplyMaterials.length > 0 ? (
            <form onSubmit={handleCatalogAdd}>
              <div className="grid grid-cols-[1fr_auto_auto_auto] gap-3 items-end">
                <div>
                  <label className="block text-xs font-medium text-surface-400 mb-1">Supply Material</label>
                  <select
                    value={selectedMaterialId}
                    onChange={(e) => handleMaterialSelect(e.target.value)}
                    className="input"
                    required
                  >
                    <option value="">Select a supply...</option>
                    {supplyMaterials.map((mat) => (
                      <option key={mat.id} value={mat.id}>
                        {mat.name} {mat.manufacturer ? `(${mat.manufacturer})` : ''} — ${mat.cost_per_kg.toFixed(2)}
                      </option>
                    ))}
                  </select>
                </div>
                <div>
                  <label className="block text-xs font-medium text-surface-400 mb-1">Unit Cost ($)</label>
                  <input
                    type="number"
                    value={newCost}
                    onChange={(e) => setNewCost(e.target.value)}
                    className="input w-28"
                    placeholder="0.00"
                    min="0"
                    step="0.01"
                    readOnly
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-surface-400 mb-1">Qty</label>
                  <input
                    type="number"
                    value={newQuantity}
                    onChange={(e) => setNewQuantity(e.target.value)}
                    className="input w-20"
                    min="1"
                  />
                </div>
                <div className="flex gap-2">
                  <button type="submit" className="btn btn-primary" disabled={!selectedMaterialId}>Add</button>
                  <button type="button" onClick={() => setShowAddForm(false)} className="btn btn-ghost">
                    Cancel
                  </button>
                </div>
              </div>
            </form>
          ) : (
            <form onSubmit={handleManualAdd}>
              <div className="grid grid-cols-[1fr_auto_auto_auto] gap-3 items-end">
                <div>
                  <label className="block text-xs font-medium text-surface-400 mb-1">Name</label>
                  <input
                    type="text"
                    value={newName}
                    onChange={(e) => setNewName(e.target.value)}
                    className="input"
                    placeholder="e.g., Lamp cord, Lightbulb"
                    autoFocus
                    required
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-surface-400 mb-1">Unit Cost ($)</label>
                  <input
                    type="number"
                    value={newCost}
                    onChange={(e) => setNewCost(e.target.value)}
                    className="input w-28"
                    placeholder="0.00"
                    min="0"
                    step="0.01"
                  />
                </div>
                <div>
                  <label className="block text-xs font-medium text-surface-400 mb-1">Qty</label>
                  <input
                    type="number"
                    value={newQuantity}
                    onChange={(e) => setNewQuantity(e.target.value)}
                    className="input w-20"
                    min="1"
                  />
                </div>
                <div className="flex gap-2">
                  <button type="submit" className="btn btn-primary">Add</button>
                  <button type="button" onClick={() => setShowAddForm(false)} className="btn btn-ghost">
                    Cancel
                  </button>
                </div>
              </div>
            </form>
          )}
        </div>
      )}

      {supplies.length === 0 ? (
        <div className="card p-6 text-center text-surface-500 text-sm">
          No supplies added yet. Add non-printed items like lamp cords, lightbulbs, etc.
        </div>
      ) : (
        <div className="card overflow-hidden">
          <table className="w-full">
            <thead>
              <tr className="border-b border-surface-800">
                <th className="text-left text-xs font-medium text-surface-500 uppercase px-4 py-2">Item</th>
                <th className="text-right text-xs font-medium text-surface-500 uppercase px-4 py-2">Unit Cost</th>
                <th className="text-right text-xs font-medium text-surface-500 uppercase px-4 py-2">Qty</th>
                <th className="text-right text-xs font-medium text-surface-500 uppercase px-4 py-2">Total</th>
                <th className="w-10 px-4 py-2"></th>
              </tr>
            </thead>
            <tbody>
              {supplies.map((supply) => (
                <tr key={supply.id} className="border-b border-surface-800/50">
                  <td className="px-4 py-3 text-surface-200">{supply.name}</td>
                  <td className="px-4 py-3 text-right text-surface-300">
                    ${(supply.unit_cost_cents / 100).toFixed(2)}
                  </td>
                  <td className="px-4 py-3 text-right text-surface-300">{supply.quantity}</td>
                  <td className="px-4 py-3 text-right text-surface-100 font-medium">
                    ${((supply.unit_cost_cents * supply.quantity) / 100).toFixed(2)}
                  </td>
                  <td className="px-4 py-3">
                    {confirmDeleteSupplyId === supply.id ? (
                      <button
                        onClick={() => handleDelete(supply)}
                        onBlur={() => setConfirmDeleteSupplyId(null)}
                        className="text-xs px-2 py-1 rounded bg-red-500/20 text-red-400 border border-red-500/50 hover:bg-red-500/30 transition-colors"
                        autoFocus
                      >
                        Delete?
                      </button>
                    ) : (
                      <button
                        onClick={() => handleDelete(supply)}
                        className="p-1 rounded text-surface-500 hover:text-red-400 hover:bg-red-500/10 transition-colors"
                      >
                        <Trash2 className="h-3.5 w-3.5" />
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
            <tfoot>
              <tr className="border-t border-surface-700">
                <td colSpan={3} className="px-4 py-3 text-right text-sm font-medium text-surface-400">
                  Total Supplies
                </td>
                <td className="px-4 py-3 text-right font-semibold text-surface-100">
                  ${(totalCents / 100).toFixed(2)}
                </td>
                <td></td>
              </tr>
            </tfoot>
          </table>
        </div>
      )}
    </div>
  )
}

// Print History Tab Component
function PrintHistoryTab({
  projectId,
  parts,
  printers,
  onRecordOutcome,
  onHandleFailure,
}: {
  projectId: string
  parts: Part[]
  printers: { id: string; name: string }[]
  onRecordOutcome: (job: PrintJob) => void
  onHandleFailure: (job: PrintJob) => void
}) {
  // Server-side job stats
  const { data: jobStats } = useQuery({
    queryKey: ['project-job-stats', projectId],
    queryFn: () => projectsApi.getJobStats(projectId),
    enabled: !!projectId,
  })

  // Get all designs from all parts
  const designQueries = parts.map((part) =>
    // eslint-disable-next-line react-hooks/rules-of-hooks
    useQuery({
      queryKey: ['designs', part.id],
      queryFn: () => designsApi.listByPart(part.id),
    })
  )

  const allDesigns = designQueries.flatMap((q) => q.data || [])
  const isLoadingDesigns = designQueries.some((q) => q.isLoading)

  // Get print jobs for all designs
  const jobQueries = allDesigns.map((design) =>
    // eslint-disable-next-line react-hooks/rules-of-hooks
    useQuery({
      queryKey: ['design-jobs', design.id],
      queryFn: () => designsApi.listPrintJobs(design.id),
      enabled: !!design.id,
    })
  )

  const allJobs = jobQueries.flatMap((q) => q.data || [])
  const isLoadingJobs = jobQueries.some((q) => q.isLoading)

  // Create lookup maps
  const designMap = Object.fromEntries(allDesigns.map((d) => [d.id, d]))
  const printerMap = Object.fromEntries(printers.map((p) => [p.id, p]))

  // Sort jobs by created_at descending
  const sortedJobs = [...allJobs].sort(
    (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
  )

  const formatDuration = (startedAt?: string, completedAt?: string) => {
    if (!startedAt) return '-'
    const start = new Date(startedAt)
    const end = completedAt ? new Date(completedAt) : new Date()
    const diffMs = end.getTime() - start.getTime()
    const hours = Math.floor(diffMs / (1000 * 60 * 60))
    const mins = Math.floor((diffMs % (1000 * 60 * 60)) / (1000 * 60))
    if (hours > 0) return `${hours}h ${mins}m`
    return `${mins}m`
  }

  if (isLoadingDesigns || isLoadingJobs) {
    return <div className="text-surface-500">Loading print history...</div>
  }

  if (allJobs.length === 0) {
    return (
      <div className="card p-8 text-center">
        <History className="h-12 w-12 mx-auto mb-3 text-surface-600" />
        <h3 className="text-lg font-medium text-surface-300 mb-2">
          No print history yet
        </h3>
        <p className="text-surface-500">
          Print jobs will appear here once you start printing
        </p>
      </div>
    )
  }

  // Use server-side stats when available, fallback to client-side
  const totalMaterialUsed = allJobs.reduce(
    (sum, job) => sum + (job.outcome?.material_used || 0),
    0
  )
  const totalCost = allJobs.reduce(
    (sum, job) => sum + (job.outcome?.material_cost || 0),
    0
  )

  const statsTotal = jobStats?.total ?? allJobs.length
  const statsCompleted = jobStats?.completed ?? allJobs.filter((j) => j.outcome?.success === true).length
  const statsFailed = jobStats?.failed ?? allJobs.filter((j) => j.outcome?.success === false || j.status === 'failed').length
  const statsPrinting = jobStats?.printing ?? allJobs.filter((j) => j.status === 'printing').length
  const statsQueued = jobStats?.queued ?? 0

  return (
    <div className="space-y-4">
      {/* Stats Summary */}
      <div className="grid grid-cols-2 lg:grid-cols-5 gap-4">
        <div className="card p-4">
          <div className="text-sm text-surface-500 mb-1">Total Prints</div>
          <div className="text-2xl font-semibold text-surface-100">
            {statsTotal}
          </div>
        </div>
        <div className="card p-4">
          <div className="text-sm text-surface-500 mb-1">Success Rate</div>
          <div className="text-2xl font-semibold text-emerald-400">
            {(statsCompleted + statsFailed) > 0
              ? Math.round((statsCompleted / (statsCompleted + statsFailed)) * 100)
              : 0}%
          </div>
        </div>
        <div className="card p-4">
          <div className="text-sm text-surface-500 mb-1">Active</div>
          <div className="text-2xl font-semibold text-blue-400">
            {statsPrinting}{statsQueued > 0 && <span className="text-sm text-surface-400 ml-1">+{statsQueued} queued</span>}
          </div>
        </div>
        <div className="card p-4">
          <div className="text-sm text-surface-500 mb-1">Material Used</div>
          <div className="text-2xl font-semibold text-surface-100">
            {totalMaterialUsed.toFixed(0)}g
          </div>
        </div>
        <div className="card p-4">
          <div className="text-sm text-surface-500 mb-1">Total Cost</div>
          <div className="text-2xl font-semibold text-emerald-400">
            ${totalCost.toFixed(2)}
          </div>
        </div>
      </div>

      {/* Job List */}
      <div className="space-y-3">
      {sortedJobs.map((job) => {
        const design = designMap[job.design_id]
        const printer = job.printer_id ? printerMap[job.printer_id] : undefined

        return (
          <div key={job.id} className="card p-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-4">
                <div
                  className={cn(
                    'w-10 h-10 rounded-lg flex items-center justify-center',
                    job.status === 'completed' && job.outcome?.success
                      ? 'bg-emerald-500/20 text-emerald-400'
                      : job.status === 'failed' || (job.outcome && !job.outcome.success)
                      ? 'bg-red-500/20 text-red-400'
                      : job.status === 'printing'
                      ? 'bg-blue-500/20 text-blue-400'
                      : 'bg-surface-700 text-surface-400'
                  )}
                >
                  {job.status === 'completed' && job.outcome?.success ? (
                    <CheckCircle className="h-5 w-5" />
                  ) : job.status === 'failed' ||
                    (job.outcome && !job.outcome.success) ? (
                    <XCircle className="h-5 w-5" />
                  ) : job.status === 'printing' ? (
                    <Printer className="h-5 w-5" />
                  ) : (
                    <Clock className="h-5 w-5" />
                  )}
                </div>

                <div>
                  <div className="font-medium text-surface-100">
                    {design?.file_name || 'Unknown design'}
                  </div>
                  <div className="text-sm text-surface-500 flex items-center gap-2">
                    <span>{printer?.name || 'Unknown printer'}</span>
                    <span>·</span>
                    <span>{formatRelativeTime(job.created_at)}</span>
                    {job.started_at && (
                      <>
                        <span>·</span>
                        <span>{formatDuration(job.started_at, job.completed_at)}</span>
                      </>
                    )}
                  </div>
                </div>
              </div>

              <div className="flex items-center gap-3">
                {job.outcome && (
                  <div className="text-right text-sm">
                    <div className="text-surface-300">
                      {job.outcome.material_used.toFixed(1)}g used
                      {job.outcome.material_cost > 0 && (
                        <span className="text-emerald-400 ml-2">
                          ${job.outcome.material_cost.toFixed(2)}
                        </span>
                      )}
                    </div>
                    {job.outcome.quality_rating && (
                      <div className="flex items-center gap-0.5 justify-end">
                        {[1, 2, 3, 4, 5].map((star) => (
                          <Star
                            key={star}
                            className={cn(
                              'h-3 w-3',
                              star <= job.outcome!.quality_rating!
                                ? 'fill-amber-400 text-amber-400'
                                : 'text-surface-600'
                            )}
                          />
                        ))}
                      </div>
                    )}
                  </div>
                )}

                <span
                  className={cn(
                    'badge',
                    getStatusBadge(
                      job.outcome?.success === false ? 'failed' : job.status
                    )
                  )}
                >
                  {job.outcome?.success === false
                    ? 'failed'
                    : job.status === 'completed' && job.outcome?.success
                    ? 'success'
                    : job.status}
                </span>

                {/* Handle Failure button for failed jobs without outcome */}
                {job.status === 'failed' && !job.outcome && (
                  <button
                    onClick={() => onHandleFailure(job)}
                    className="btn btn-primary text-xs py-1 px-2 flex items-center gap-1"
                  >
                    <RefreshCw className="h-3 w-3" />
                    Handle Failure
                  </button>
                )}

                {/* Record Outcome button for completed jobs without outcome */}
                {job.status === 'completed' && !job.outcome && (
                  <button
                    onClick={() => onRecordOutcome(job)}
                    className="btn btn-secondary text-xs py-1 px-2"
                  >
                    Record Outcome
                  </button>
                )}
              </div>
            </div>

            {/* Expandable event timeline */}
            <div className="mt-3 pt-3 border-t border-surface-800">
              <ExpandableJobEvents jobId={job.id} />
            </div>
          </div>
        )
      })}
      </div>
    </div>
  )
}

// Quick Stats Bar (above tabs)
function ProjectQuickStats({ summary }: { summary: ProjectSummary }) {
  const formatCents = (cents: number) => {
    const negative = cents < 0
    const abs = Math.abs(cents)
    return `${negative ? '-' : ''}$${(abs / 100).toFixed(2)}`
  }

  const formatTime = (seconds: number) => {
    if (seconds <= 0) return '-'
    const hours = Math.floor(seconds / 3600)
    const mins = Math.floor((seconds % 3600) / 60)
    if (hours > 0) return `${hours}h ${mins}m`
    return `${mins}m`
  }

  const printSeconds = summary.total_print_seconds > 0
    ? summary.total_print_seconds
    : summary.estimated_print_seconds
  const materialGrams = summary.total_material_grams > 0
    ? summary.total_material_grams
    : summary.estimated_material_grams
  const avgProfit = summary.sales_count > 0
    ? Math.round(summary.gross_profit_cents / summary.sales_count)
    : 0

  return (
    <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 mb-6">
      <div className="flex items-center gap-3 px-4 py-3 rounded-lg bg-surface-800/50 border border-surface-700">
        <Timer className="h-5 w-5 text-blue-400 shrink-0" />
        <div className="min-w-0">
          <div className="text-xs text-surface-500">Print Time</div>
          <div className="text-sm font-semibold text-surface-100 truncate">
            {formatTime(printSeconds)}
          </div>
        </div>
      </div>
      <div className="flex items-center gap-3 px-4 py-3 rounded-lg bg-surface-800/50 border border-surface-700">
        <DollarSign className="h-5 w-5 text-amber-400 shrink-0" />
        <div className="min-w-0">
          <div className="text-xs text-surface-500">Unit Cost</div>
          <div className="text-sm font-semibold text-surface-100 truncate">
            {formatCents(summary.unit_cost_cents)}
          </div>
        </div>
      </div>
      <div className="flex items-center gap-3 px-4 py-3 rounded-lg bg-surface-800/50 border border-surface-700">
        <Scale className="h-5 w-5 text-purple-400 shrink-0" />
        <div className="min-w-0">
          <div className="text-xs text-surface-500">Material</div>
          <div className="text-sm font-semibold text-surface-100 truncate">
            {materialGrams > 0 ? `${materialGrams.toFixed(0)}g` : '-'}
          </div>
        </div>
      </div>
      <div className="flex items-center gap-3 px-4 py-3 rounded-lg bg-surface-800/50 border border-surface-700">
        <TrendingUp className="h-5 w-5 text-emerald-400 shrink-0" />
        <div className="min-w-0">
          <div className="text-xs text-surface-500">Avg Profit / Sale</div>
          <div className={cn(
            'text-sm font-semibold truncate',
            summary.sales_count > 0
              ? avgProfit >= 0 ? 'text-emerald-400' : 'text-red-400'
              : 'text-surface-500'
          )}>
            {summary.sales_count > 0 ? formatCents(avgProfit) : '-'}
          </div>
        </div>
      </div>
    </div>
  )
}

// Project Analytics Tab
function ProjectAnalyticsTab({ summary }: { summary?: ProjectSummary }) {
  if (!summary) {
    return (
      <div className="card p-8 text-center">
        <BarChart3 className="h-12 w-12 mx-auto mb-3 text-surface-600" />
        <h3 className="text-lg font-medium text-surface-300 mb-2">
          No analytics yet
        </h3>
        <p className="text-surface-500">
          Complete some print jobs and record sales to see project analytics
        </p>
      </div>
    )
  }

  const formatCents = (cents: number) => {
    const negative = cents < 0
    const abs = Math.abs(cents)
    return `${negative ? '-' : ''}$${(abs / 100).toFixed(2)}`
  }

  const formatSeconds = (seconds: number) => {
    if (seconds <= 0) return '-'
    const hours = Math.floor(seconds / 3600)
    const mins = Math.floor((seconds % 3600) / 60)
    if (hours > 0) return `${hours}h ${mins}m`
    return `${mins}m`
  }

  return (
    <div className="space-y-6">
      {/* Revenue Section */}
      <div>
        <h3 className="text-sm font-medium text-surface-400 uppercase tracking-wider mb-3 flex items-center gap-2">
          <DollarSign className="h-4 w-4" />
          Revenue
        </h3>
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          <div className="card p-4">
            <div className="flex items-center gap-1.5 text-sm text-surface-500 mb-1">
              Gross Revenue
              <Tooltip text="Total amount collected from all sales of this project before any deductions." />
            </div>
            <div className="text-2xl font-semibold text-surface-100">
              {formatCents(summary.total_revenue_cents)}
            </div>
            {summary.sales_count > 0 && (
              <div className="text-xs text-surface-500 mt-1">
                {summary.sales_count} sale{summary.sales_count !== 1 ? 's' : ''}
              </div>
            )}
          </div>
          <div className="card p-4">
            <div className="flex items-center gap-1.5 text-sm text-surface-500 mb-1">
              Fees
              <Tooltip text="Marketplace and payment processing fees deducted from sales (e.g. Etsy fees, PayPal fees)." />
            </div>
            <div className="text-2xl font-semibold text-red-400">
              {formatCents(summary.total_fees_cents)}
            </div>
          </div>
          <div className="card p-4">
            <div className="flex items-center gap-1.5 text-sm text-surface-500 mb-1">
              Net Revenue
              <Tooltip text="Gross revenue minus fees. The actual amount received after marketplace and payment deductions." />
            </div>
            <div className="text-2xl font-semibold text-surface-100">
              {formatCents(summary.net_revenue_cents)}
            </div>
          </div>
          <div className="card p-4">
            <div className="flex items-center gap-1.5 text-sm text-surface-500 mb-1">
              Gross Profit
              <Tooltip text="Net revenue minus total cost of goods sold (COGS). This is the profit after accounting for all production costs and fees." />
            </div>
            <div className={cn(
              'text-2xl font-semibold',
              summary.gross_profit_cents >= 0 ? 'text-emerald-400' : 'text-red-400'
            )}>
              {formatCents(summary.gross_profit_cents)}
            </div>
            {summary.gross_margin_percent > 0 && (
              <div className="text-xs text-surface-500 mt-1">
                {summary.gross_margin_percent.toFixed(1)}% margin
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Cost Breakdown */}
      <div>
        <h3 className="text-sm font-medium text-surface-400 uppercase tracking-wider mb-3 flex items-center gap-2">
          <TrendingUp className="h-4 w-4" />
          Cost Breakdown
        </h3>
        <div className="grid grid-cols-2 lg:grid-cols-3 gap-4">
          <div className="card p-4">
            <div className="flex items-center gap-1.5 text-sm text-surface-500 mb-1">
              Unit Cost
              <Tooltip text="Total cost to produce one unit of this project, including printer time, material, and supplies." />
            </div>
            <div className="text-2xl font-semibold text-surface-100">
              {formatCents(summary.unit_cost_cents)}
            </div>
            <div className="text-xs text-surface-500 mt-1">per unit produced</div>
          </div>
          {summary.sales_count > 1 && (
            <div className="card p-4">
              <div className="flex items-center gap-1.5 text-sm text-surface-500 mb-1">
                Total COGS
                <Tooltip text="Cost of Goods Sold. Unit cost multiplied by the number of units sold. Represents total production cost for all sales." />
              </div>
              <div className="text-2xl font-semibold text-red-400">
                {formatCents(summary.total_cost_cents)}
              </div>
              <div className="text-xs text-surface-500 mt-1">
                {summary.sales_count} units sold
              </div>
            </div>
          )}
          <div className="card p-4">
            <div className="flex items-center gap-1.5 text-sm text-surface-500 mb-1">
              Printer Time
              <Tooltip text="Cost of printer usage based on each printer's hourly rate multiplied by actual print time from completed jobs." />
            </div>
            <div className="text-2xl font-semibold text-surface-100">
              {formatCents(summary.printer_time_cost_cents)}
            </div>
          </div>
          <div className="card p-4">
            <div className="flex items-center gap-1.5 text-sm text-surface-500 mb-1">
              Material (Actual)
              <Tooltip text="Actual filament cost recorded when print jobs complete, calculated from material used and spool cost per kg." />
            </div>
            <div className="text-2xl font-semibold text-surface-100">
              {formatCents(summary.material_cost_cents)}
            </div>
            {summary.total_material_grams > 0 && (
              <div className="text-xs text-surface-500 mt-1">
                {summary.total_material_grams.toFixed(0)}g used
              </div>
            )}
          </div>
          {summary.estimated_material_cost_cents > 0 && (
            <div className="card p-4">
              <div className="flex items-center gap-1.5 text-sm text-surface-500 mb-1">
                Est. Material
                <Tooltip text="Estimated material cost calculated from slice profile weight data at a default rate of $19.99/kg. Used when actual costs aren't available." />
              </div>
              <div className="text-2xl font-semibold text-amber-400">
                {formatCents(summary.estimated_material_cost_cents)}
              </div>
              <div className="text-xs text-surface-500 mt-1">from slice profiles</div>
            </div>
          )}
          {summary.supply_cost_cents > 0 && (
            <div className="card p-4">
              <div className="flex items-center gap-1.5 text-sm text-surface-500 mb-1">
                Supplies
                <Tooltip text="Total cost of non-printed items added to this project's bill of materials (e.g. hardware, wiring, packaging)." />
              </div>
              <div className="text-2xl font-semibold text-surface-100">
                {formatCents(summary.supply_cost_cents)}
              </div>
              <div className="text-xs text-surface-500 mt-1">non-printed items</div>
            </div>
          )}
        </div>
      </div>

      {/* Performance */}
      <div>
        <h3 className="text-sm font-medium text-surface-400 uppercase tracking-wider mb-3 flex items-center gap-2">
          <Timer className="h-4 w-4" />
          Performance
        </h3>
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          <div className="card p-4">
            <div className="flex items-center gap-1.5 text-sm text-surface-500 mb-1">
              Total Print Time
              <Tooltip text="Combined print time across all jobs. Uses actual time from completed jobs, or estimated time from slice profiles if no jobs have run." />
            </div>
            <div className="text-2xl font-semibold text-surface-100">
              {formatSeconds(summary.total_print_seconds || summary.estimated_print_seconds)}
            </div>
            {summary.total_print_seconds <= 0 && summary.estimated_print_seconds > 0 && (
              <div className="text-xs text-surface-500 mt-1">estimated from slices</div>
            )}
          </div>
          <div className="card p-4">
            <div className="flex items-center gap-1.5 text-sm text-surface-500 mb-1">
              Avg Print Time
              <Tooltip text="Average print time per completed job. Calculated from actual completed print job durations." />
            </div>
            <div className="text-2xl font-semibold text-surface-100">
              {formatSeconds(summary.avg_print_seconds)}
            </div>
          </div>
          <div className="card p-4">
            <div className="flex items-center gap-1.5 text-sm text-surface-500 mb-1">
              Success Rate
              <Tooltip text="Percentage of print jobs that completed successfully out of all completed and failed jobs." />
            </div>
            <div className={cn(
              'text-2xl font-semibold',
              summary.success_rate >= 90 ? 'text-emerald-400' :
              summary.success_rate >= 70 ? 'text-amber-400' : 'text-red-400'
            )}>
              {summary.success_rate.toFixed(0)}%
            </div>
            <div className="text-xs text-surface-500 mt-1">
              {summary.completed_count}/{summary.job_count} jobs
            </div>
          </div>
          <div className="card p-4">
            <div className="flex items-center gap-1.5 text-sm text-surface-500 mb-1">
              Profit / Hour
              <Tooltip text="Gross profit divided by total print hours. Measures how efficiently this project converts printer time into profit." />
            </div>
            <div className={cn(
              'text-2xl font-semibold',
              summary.profit_per_hour_cents >= 0 ? 'text-emerald-400' : 'text-red-400'
            )}>
              {formatCents(summary.profit_per_hour_cents)}
            </div>
            <div className="text-xs text-surface-500 mt-1">per print hour</div>
          </div>
        </div>
      </div>
    </div>
  )
}

// Upload Design Modal
function UploadDesignModal({
  part,
  onClose,
  onSuccess,
}: {
  part: Part
  onClose: () => void
  onSuccess: () => void
}) {
  const [file, setFile] = useState<File | null>(null)
  const [notes, setNotes] = useState('')
  const [uploading, setUploading] = useState(false)

  const handleUpload = async () => {
    if (!file) return
    
    setUploading(true)
    try {
      await designsApi.upload(part.id, file, notes)
      onSuccess()
    } catch (err) {
      console.error('Upload failed:', err)
    } finally {
      setUploading(false)
    }
  }

  return (
    <Modal title={`Upload Design for ${part.name}`} onClose={onClose}>
      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-surface-300 mb-2">
            Design File
          </label>
          <div 
            className={cn(
              'border-2 border-dashed rounded-lg p-8 text-center transition-colors',
              file ? 'border-accent-500 bg-accent-500/5' : 'border-surface-700 hover:border-surface-600'
            )}
          >
            <input
              type="file"
              accept=".stl,.3mf,.gcode"
              onChange={(e) => setFile(e.target.files?.[0] || null)}
              className="hidden"
              id="file-upload"
            />
            <label htmlFor="file-upload" className="cursor-pointer">
              {file ? (
                <div>
                  <FileCode className="h-10 w-10 mx-auto mb-2 text-accent-500" />
                  <p className="text-surface-100 font-medium">{file.name}</p>
                  <p className="text-surface-500 text-sm">{formatBytes(file.size)}</p>
                </div>
              ) : (
                <div>
                  <Upload className="h-10 w-10 mx-auto mb-2 text-surface-500" />
                  <p className="text-surface-300">Click to upload or drag & drop</p>
                  <p className="text-surface-500 text-sm">STL, 3MF, or GCODE</p>
                </div>
              )}
            </label>
          </div>
        </div>
        <div>
          <label className="block text-sm font-medium text-surface-300 mb-1">
            Notes
          </label>
          <textarea
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            rows={2}
            className="input resize-none"
            placeholder="Optional notes about this version"
          />
        </div>
      </div>
      <div className="flex justify-end gap-3 mt-6">
        <button onClick={onClose} className="btn btn-ghost">
          Cancel
        </button>
        <button
          onClick={handleUpload}
          disabled={!file || uploading}
          className="btn btn-primary"
        >
          {uploading ? 'Uploading...' : 'Upload Design'}
        </button>
      </div>
    </Modal>
  )
}

// Spool with material info type
interface SpoolWithMaterial {
  id: string
  material_id: string
  initial_weight: number
  remaining_weight: number
  status: string
  material?: Material
}

// Send to Printer Modal
function SendToPrinterModal({
  design,
  printers,
  printerStates,
  onClose,
}: {
  design: Design
  printers: { id: string; name: string; model: string }[]
  printerStates: Record<string, { status: string; progress: number }>
  onClose: () => void
}) {
  const [selectedPrinter, setSelectedPrinter] = useState('')
  const [selectedSpool, setSelectedSpool] = useState('')
  const [sending, setSending] = useState(false)
  const queryClient = useQueryClient()

  const { data: spoolsWithMaterials = [] } = useSpoolsWithMaterials()

  // Filter spools to show only available ones (not empty or archived)
  const availableSpools = spoolsWithMaterials.filter(
    (spool: SpoolWithMaterial) =>
      spool.status !== 'empty' && spool.status !== 'archived'
  )

  // Show warning if spool has low remaining weight
  const LOW_WEIGHT_THRESHOLD = 100 // grams

  const handleSend = async () => {
    if (!selectedPrinter || !selectedSpool) return

    setSending(true)
    try {
      // Create print job
      const job = await printJobsApi.create({
        design_id: design.id,
        printer_id: selectedPrinter,
        material_spool_id: selectedSpool,
      })

      // Start the job
      await printJobsApi.start(job.id)

      queryClient.invalidateQueries({ queryKey: ['print-jobs'] })
      onClose()
    } catch (err) {
      console.error('Failed to send to printer:', err)
    } finally {
      setSending(false)
    }
  }

  const availablePrinters = printers.filter(
    (p) => printerStates[p.id]?.status === 'idle' || !printerStates[p.id]
  )

  return (
    <Modal title="Send to Printer" onClose={onClose}>
      <div className="space-y-4">
        <div>
          <label className="block text-sm font-medium text-surface-300 mb-2">
            Design
          </label>
          <div className="p-3 rounded-lg bg-surface-800/50">
            <div className="font-medium text-surface-100">
              v{design.version} — {design.file_name}
            </div>
            <div className="text-sm text-surface-500">
              {formatBytes(design.file_size_bytes)}
            </div>
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-surface-300 mb-2">
            Select Printer
          </label>
          {availablePrinters.length === 0 ? (
            <div className="text-surface-500 text-sm p-4 text-center bg-surface-800/50 rounded-lg">
              No printers available. All printers are busy or offline.
            </div>
          ) : (
            <div className="space-y-2">
              {availablePrinters.map((printer) => (
                <label
                  key={printer.id}
                  className={cn(
                    'flex items-center gap-3 p-3 rounded-lg cursor-pointer transition-colors',
                    selectedPrinter === printer.id
                      ? 'bg-accent-500/10 border border-accent-500'
                      : 'bg-surface-800/50 border border-transparent hover:bg-surface-800'
                  )}
                >
                  <input
                    type="radio"
                    name="printer"
                    value={printer.id}
                    checked={selectedPrinter === printer.id}
                    onChange={(e) => setSelectedPrinter(e.target.value)}
                    className="sr-only"
                  />
                  <div className="w-3 h-3 rounded-full border-2 border-current flex items-center justify-center">
                    {selectedPrinter === printer.id && (
                      <div className="w-1.5 h-1.5 rounded-full bg-current" />
                    )}
                  </div>
                  <div>
                    <div className="font-medium text-surface-100">
                      {printer.name}
                    </div>
                    <div className="text-xs text-surface-500">
                      {printer.model || 'Unknown model'}
                    </div>
                  </div>
                </label>
              ))}
            </div>
          )}
        </div>

        <div>
          <label className="block text-sm font-medium text-surface-300 mb-2">
            Select Material Spool
          </label>
          {availableSpools.length === 0 ? (
            <div className="text-surface-500 text-sm p-4 text-center bg-surface-800/50 rounded-lg">
              No spools available. Add material spools in the Materials page.
            </div>
          ) : (
            <div className="space-y-2 max-h-48 overflow-y-auto">
              {availableSpools.map((spool: SpoolWithMaterial) => {
                const isLow = spool.remaining_weight < LOW_WEIGHT_THRESHOLD
                return (
                  <label
                    key={spool.id}
                    className={cn(
                      'flex items-center gap-3 p-3 rounded-lg cursor-pointer transition-colors',
                      selectedSpool === spool.id
                        ? 'bg-accent-500/10 border border-accent-500'
                        : 'bg-surface-800/50 border border-transparent hover:bg-surface-800'
                    )}
                  >
                    <input
                      type="radio"
                      name="spool"
                      value={spool.id}
                      checked={selectedSpool === spool.id}
                      onChange={(e) => setSelectedSpool(e.target.value)}
                      className="sr-only"
                    />
                    <div className="w-3 h-3 rounded-full border-2 border-current flex items-center justify-center">
                      {selectedSpool === spool.id && (
                        <div className="w-1.5 h-1.5 rounded-full bg-current" />
                      )}
                    </div>
                    {spool.material?.color_hex && (
                      <div
                        className="w-4 h-4 rounded-full border border-surface-600"
                        style={{ backgroundColor: spool.material.color_hex }}
                      />
                    )}
                    <div className="flex-1 min-w-0">
                      <div className="font-medium text-surface-100 truncate">
                        {spool.material?.name || 'Unknown material'}
                      </div>
                      <div className="text-xs text-surface-500 flex items-center gap-2">
                        <span className="uppercase">
                          {spool.material?.type || '?'}
                        </span>
                        <span>•</span>
                        <span
                          className={cn(
                            isLow && 'text-amber-400 font-medium'
                          )}
                        >
                          {spool.remaining_weight.toFixed(0)}g remaining
                        </span>
                        {isLow && (
                          <AlertTriangle className="h-3 w-3 text-amber-400" />
                        )}
                      </div>
                    </div>
                  </label>
                )
              })}
            </div>
          )}
        </div>
      </div>

      <div className="flex justify-end gap-3 mt-6">
        <button onClick={onClose} className="btn btn-ghost">
          Cancel
        </button>
        <button
          onClick={handleSend}
          disabled={!selectedPrinter || !selectedSpool || sending}
          className="btn btn-primary"
        >
          <Play className="h-4 w-4 mr-2" />
          {sending ? 'Sending...' : 'Start Print'}
        </button>
      </div>
    </Modal>
  )
}

// Generic Modal Component
function Modal({
  title,
  children,
  onClose,
}: {
  title: string
  children: React.ReactNode
  onClose: () => void
}) {
  return (
    <div
      className="fixed inset-0 bg-black/60 flex items-center justify-center z-50"
      onClick={(e) => e.target === e.currentTarget && onClose()}
    >
      <div className="card w-full max-w-md p-6">
        <h2 className="text-xl font-semibold text-surface-100 mb-4">{title}</h2>
        {children}
      </div>
    </div>
  )
}

// Outcome Capture Modal
function OutcomeCaptureModal({
  job,
  onClose,
  onSuccess,
}: {
  job: PrintJob
  onClose: () => void
  onSuccess: () => void
}) {
  const [success, setSuccess] = useState(true)
  const [qualityRating, setQualityRating] = useState(4)
  const [materialUsed, setMaterialUsed] = useState('')
  const [notes, setNotes] = useState('')
  const [failureReason, setFailureReason] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const handleSubmit = async () => {
    setSubmitting(true)
    try {
      const materialGrams = parseFloat(materialUsed) || 0

      await printJobsApi.recordOutcome(job.id, {
        success,
        quality_rating: success ? qualityRating : undefined,
        failure_reason: !success ? failureReason : undefined,
        notes: notes || undefined,
        material_used: materialGrams,
        material_cost: 0, // Will be calculated by backend
      })
      onSuccess()
    } catch (err) {
      console.error('Failed to record outcome:', err)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Modal title="Record Print Outcome" onClose={onClose}>
      <div className="space-y-4">
        {/* Success/Failure Toggle */}
        <div>
          <label className="block text-sm font-medium text-surface-300 mb-2">
            Print Result
          </label>
          <div className="grid grid-cols-2 gap-2">
            <button
              type="button"
              onClick={() => setSuccess(true)}
              className={cn(
                'flex items-center justify-center gap-2 p-3 rounded-lg border transition-colors',
                success
                  ? 'bg-emerald-500/20 border-emerald-500 text-emerald-400'
                  : 'bg-surface-800/50 border-surface-700 text-surface-400 hover:bg-surface-800'
              )}
            >
              <CheckCircle className="h-5 w-5" />
              Success
            </button>
            <button
              type="button"
              onClick={() => setSuccess(false)}
              className={cn(
                'flex items-center justify-center gap-2 p-3 rounded-lg border transition-colors',
                !success
                  ? 'bg-red-500/20 border-red-500 text-red-400'
                  : 'bg-surface-800/50 border-surface-700 text-surface-400 hover:bg-surface-800'
              )}
            >
              <XCircle className="h-5 w-5" />
              Failed
            </button>
          </div>
        </div>

        {/* Quality Rating (only for success) */}
        {success && (
          <div>
            <label className="block text-sm font-medium text-surface-300 mb-2">
              Quality Rating
            </label>
            <div className="flex gap-1">
              {[1, 2, 3, 4, 5].map((rating) => (
                <button
                  key={rating}
                  type="button"
                  onClick={() => setQualityRating(rating)}
                  className="p-1"
                >
                  <Star
                    className={cn(
                      'h-6 w-6 transition-colors',
                      rating <= qualityRating
                        ? 'fill-amber-400 text-amber-400'
                        : 'text-surface-600'
                    )}
                  />
                </button>
              ))}
            </div>
          </div>
        )}

        {/* Failure Reason (only for failure) */}
        {!success && (
          <div>
            <label className="block text-sm font-medium text-surface-300 mb-1">
              Failure Reason
            </label>
            <input
              type="text"
              value={failureReason}
              onChange={(e) => setFailureReason(e.target.value)}
              className="input"
              placeholder="e.g., Bed adhesion, spaghetti, layer shift"
            />
          </div>
        )}

        {/* Material Used */}
        <div>
          <label className="block text-sm font-medium text-surface-300 mb-1">
            Material Used (grams)
          </label>
          <input
            type="number"
            value={materialUsed}
            onChange={(e) => setMaterialUsed(e.target.value)}
            className="input"
            placeholder="e.g., 25"
            min="0"
            step="0.1"
          />
          <p className="text-xs text-surface-500 mt-1">
            Enter the amount of material consumed during this print
          </p>
        </div>

        {/* Notes */}
        <div>
          <label className="block text-sm font-medium text-surface-300 mb-1">
            Notes
          </label>
          <textarea
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            rows={2}
            className="input resize-none"
            placeholder="Optional notes about this print"
          />
        </div>
      </div>

      <div className="flex justify-end gap-3 mt-6">
        <button onClick={onClose} className="btn btn-ghost">
          Cancel
        </button>
        <button
          onClick={handleSubmit}
          disabled={submitting}
          className="btn btn-primary"
        >
          {submitting ? 'Saving...' : 'Save Outcome'}
        </button>
      </div>
    </Modal>
  )
}

