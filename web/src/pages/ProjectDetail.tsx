import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { 
  ArrowLeft, 
  Plus, 
  Upload,
  Printer,
  Play,
  FileCode,
  Box
} from 'lucide-react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useProject, useParts, useCreatePart, useUpdateProject } from '../hooks/useProjects'
import { usePrinters, usePrinterStates } from '../hooks/usePrinters'
import { designsApi, printJobsApi } from '../api/client'
import { cn, getStatusBadge, formatBytes, formatRelativeTime } from '../lib/utils'
import type { Design, Part, ProjectStatus } from '../types'

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
    })
    
    setShowAddPart(false)
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
      <div className="p-8">
        <div className="text-surface-500">Loading...</div>
      </div>
    )
  }

  if (!project) {
    return (
      <div className="p-8">
        <div className="text-surface-500">Project not found</div>
      </div>
    )
  }

  return (
    <div className="p-8">
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
        <div className="grid grid-cols-4 gap-3">
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

      {/* Parts */}
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
          <h3 className="text-lg font-medium text-surface-300 mb-2">No parts yet</h3>
          <p className="text-surface-500 mb-4">Add parts to start organizing your project</p>
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
            </div>
            <div className="flex justify-end gap-3 mt-6">
              <button
                type="button"
                onClick={() => setShowAddPart(false)}
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
                <button
                  onClick={() => onSendToPrinter(design)}
                  className="btn btn-primary text-xs py-1.5 px-3"
                >
                  <Play className="h-3.5 w-3.5 mr-1" />
                  Print
                </button>
              </div>
            ))}
          </div>
        )}
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
  const [sending, setSending] = useState(false)
  const queryClient = useQueryClient()

  const handleSend = async () => {
    if (!selectedPrinter) return
    
    setSending(true)
    try {
      // Create print job
      const job = await printJobsApi.create({
        design_id: design.id,
        printer_id: selectedPrinter,
        material_spool_id: '00000000-0000-0000-0000-000000000000', // TODO: Add spool selection
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

  const availablePrinters = printers.filter(p => 
    printerStates[p.id]?.status === 'idle' || !printerStates[p.id]
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
                    <div className="font-medium text-surface-100">{printer.name}</div>
                    <div className="text-xs text-surface-500">{printer.model || 'Unknown model'}</div>
                  </div>
                </label>
              ))}
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
          disabled={!selectedPrinter || sending}
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
  onClose 
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

