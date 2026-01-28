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
  X
} from 'lucide-react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useProject, useParts, useCreatePart, useUpdateProject } from '../hooks/useProjects'
import { usePrinters, usePrinterStates } from '../hooks/usePrinters'
import { useSpoolsWithMaterials } from '../hooks/useMaterials'
import { designsApi, printJobsApi, projectsApi } from '../api/client'
import { cn, getStatusBadge, formatBytes, formatRelativeTime } from '../lib/utils'
import { FailureModal } from '../components/FailureModal'
import { ExpandableJobEvents } from '../components/JobEventTimeline'
import type { Design, Part, ProjectStatus, Material, PrintJob, ProjectSummary } from '../types'

export default function ProjectDetail() {
  const { id } = useParams<{ id: string }>()
  const queryClient = useQueryClient()
  
  const { data: project, isLoading: projectLoading } = useProject(id!)
  const { data: parts = [], isLoading: partsLoading } = useParts(id!)
  const { data: printers = [] } = usePrinters()
  const { data: printerStates = {} } = usePrinterStates()
  
  const createPart = useCreatePart()
  const updateProject = useUpdateProject()
  
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

  const handleStatusChange = async (status: ProjectStatus) => {
    if (!project) return
    await updateProject.mutateAsync({
      id: project.id,
      data: { status },
    })
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
        
        <div className="flex items-start justify-between">
          <div>
            <h1 className="text-3xl font-display font-bold text-surface-100">
              {project.name}
            </h1>
            {project.description && (
              <p className="text-surface-400 mt-1">{project.description}</p>
            )}
          </div>
          
          <div className="flex items-center gap-3">
            <select
              value={project.status}
              onChange={(e) => handleStatusChange(e.target.value as ProjectStatus)}
              className="input w-auto"
            >
              <option value="draft">Draft</option>
              <option value="active">Active</option>
              <option value="completed">Completed</option>
              <option value="archived">Archived</option>
            </select>
          </div>
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
                />
              ))}
            </div>
          )}
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

// Part Card Component
function PartCard({ 
  part, 
  onUpload,
  onSendToPrinter,
}: { 
  part: Part
  onUpload: () => void
  onSendToPrinter: (design: Design) => void
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
        <span className={cn('badge', getStatusBadge(part.status))}>
          {part.status}
        </span>
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
            <div className="text-sm text-surface-500 mb-1">Gross Revenue</div>
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
            <div className="text-sm text-surface-500 mb-1">Fees</div>
            <div className="text-2xl font-semibold text-red-400">
              {formatCents(summary.total_fees_cents)}
            </div>
          </div>
          <div className="card p-4">
            <div className="text-sm text-surface-500 mb-1">Net Revenue</div>
            <div className="text-2xl font-semibold text-surface-100">
              {formatCents(summary.net_revenue_cents)}
            </div>
          </div>
          <div className="card p-4">
            <div className="text-sm text-surface-500 mb-1">Gross Profit</div>
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
            <div className="text-sm text-surface-500 mb-1">Total Cost</div>
            <div className="text-2xl font-semibold text-surface-100">
              {formatCents(summary.total_cost_cents)}
            </div>
          </div>
          <div className="card p-4">
            <div className="text-sm text-surface-500 mb-1">Printer Time</div>
            <div className="text-2xl font-semibold text-surface-100">
              {formatCents(summary.printer_time_cost_cents)}
            </div>
          </div>
          <div className="card p-4">
            <div className="text-sm text-surface-500 mb-1">Material</div>
            <div className="text-2xl font-semibold text-surface-100">
              {formatCents(summary.material_cost_cents)}
            </div>
            {summary.total_material_grams > 0 && (
              <div className="text-xs text-surface-500 mt-1">
                {summary.total_material_grams.toFixed(0)}g used
              </div>
            )}
          </div>
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
            <div className="text-sm text-surface-500 mb-1">Total Print Time</div>
            <div className="text-2xl font-semibold text-surface-100">
              {formatSeconds(summary.total_print_seconds)}
            </div>
          </div>
          <div className="card p-4">
            <div className="text-sm text-surface-500 mb-1">Avg Print Time</div>
            <div className="text-2xl font-semibold text-surface-100">
              {formatSeconds(summary.avg_print_seconds)}
            </div>
          </div>
          <div className="card p-4">
            <div className="text-sm text-surface-500 mb-1">Success Rate</div>
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
            <div className="text-sm text-surface-500 mb-1">Profit / Hour</div>
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

