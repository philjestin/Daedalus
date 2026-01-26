import { useState } from 'react'
import { 
  Plus, 
  Printer as PrinterIcon, 
  Wifi,
  WifiOff,
  Trash2,
  Radar,
  Loader2,
  Check
} from 'lucide-react'
import { usePrinters, useCreatePrinter, useDeletePrinter, usePrinterStates } from '../hooks/usePrinters'
import { printersApi, type DiscoveredPrinter } from '../api/client'
import { cn, getStatusBadge } from '../lib/utils'
import type { ConnectionType } from '../types'

export default function Printers() {
  const { data: printers = [], isLoading, refetch } = usePrinters()
  const { data: printerStates = {} } = usePrinterStates()
  const createPrinter = useCreatePrinter()
  const deletePrinter = useDeletePrinter()
  
  const [showAdd, setShowAdd] = useState(false)
  const [showDiscover, setShowDiscover] = useState(false)
  const [discovering, setDiscovering] = useState(false)
  const [discovered, setDiscovered] = useState<DiscoveredPrinter[]>([])
  const [addingPrinter, setAddingPrinter] = useState<string | null>(null)

  const handleDiscover = async () => {
    setShowDiscover(true)
    setDiscovering(true)
    setDiscovered([])
    
    try {
      const found = await printersApi.discover()
      setDiscovered(found)
    } catch (err) {
      console.error('Discovery failed:', err)
    } finally {
      setDiscovering(false)
    }
  }

  const handleAddDiscovered = async (printer: DiscoveredPrinter) => {
    setAddingPrinter(printer.id)
    try {
      await createPrinter.mutateAsync({
        name: printer.name,
        model: printer.model || '',
        manufacturer: printer.manufacturer || '',
        connection_type: printer.type,
        connection_uri: `http://${printer.host}:${printer.port}`,
      })
      // Mark as added
      setDiscovered(prev => 
        prev.map(p => p.id === printer.id ? { ...p, already_added: true } : p)
      )
      refetch()
    } catch (err) {
      console.error('Failed to add printer:', err)
    } finally {
      setAddingPrinter(null)
    }
  }

  const handleCreate = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    const formData = new FormData(e.currentTarget)
    
    await createPrinter.mutateAsync({
      name: formData.get('name') as string,
      model: formData.get('model') as string,
      manufacturer: formData.get('manufacturer') as string,
      connection_type: formData.get('connection_type') as ConnectionType,
      connection_uri: formData.get('connection_uri') as string,
      location: formData.get('location') as string,
    })
    
    setShowAdd(false)
  }

  const handleDelete = async (id: string) => {
    if (!confirm('Are you sure you want to delete this printer?')) return
    await deletePrinter.mutateAsync(id)
  }

  return (
    <div className="p-8">
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-display font-bold text-surface-100">
            Printers
          </h1>
          <p className="text-surface-400 mt-1">
            Manage your print farm
          </p>
        </div>
        <div className="flex gap-2">
          <button 
            onClick={handleDiscover}
            className="btn btn-secondary"
          >
            <Radar className="h-4 w-4 mr-2" />
            Scan Network
          </button>
          <button 
            onClick={() => setShowAdd(true)}
            className="btn btn-primary"
          >
            <Plus className="h-4 w-4 mr-2" />
            Add Manually
          </button>
        </div>
      </div>

      {/* Printers Grid */}
      {isLoading ? (
        <div className="text-surface-500">Loading...</div>
      ) : printers.length === 0 ? (
        <div className="text-center py-16">
          <PrinterIcon className="h-16 w-16 mx-auto mb-4 text-surface-600" />
          <h3 className="text-xl font-semibold text-surface-300 mb-2">
            No printers configured
          </h3>
          <p className="text-surface-500 mb-4">
            Add your first printer to start managing your farm
          </p>
          <button 
            onClick={() => setShowAdd(true)}
            className="btn btn-primary"
          >
            <Plus className="h-4 w-4 mr-2" />
            Add Printer
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-3 gap-4">
          {printers.map((printer) => {
            const state = printerStates[printer.id]
            return (
              <div key={printer.id} className="card p-5">
                <div className="flex items-start justify-between mb-4">
                  <div className="flex items-center gap-3">
                    <div className={cn(
                      'p-2 rounded-lg',
                      state?.status === 'printing' ? 'bg-emerald-500/20' :
                      state?.status === 'idle' ? 'bg-blue-500/20' :
                      state?.status === 'error' ? 'bg-red-500/20' :
                      'bg-surface-800'
                    )}>
                      <PrinterIcon className={cn(
                        'h-5 w-5',
                        state?.status === 'printing' ? 'text-emerald-400' :
                        state?.status === 'idle' ? 'text-blue-400' :
                        state?.status === 'error' ? 'text-red-400' :
                        'text-surface-500'
                      )} />
                    </div>
                    <div>
                      <h3 className="font-semibold text-surface-100">
                        {printer.name}
                      </h3>
                      <p className="text-sm text-surface-500">
                        {printer.model || printer.manufacturer || 'Unknown model'}
                      </p>
                    </div>
                  </div>
                  <button
                    onClick={() => handleDelete(printer.id)}
                    className="p-1.5 rounded hover:bg-surface-800 text-surface-500 hover:text-red-400 transition-colors"
                  >
                    <Trash2 className="h-4 w-4" />
                  </button>
                </div>

                <div className="space-y-3">
                  {/* Status */}
                  <div className="flex items-center justify-between">
                    <span className="text-sm text-surface-500">Status</span>
                    <span className={cn('badge', getStatusBadge(state?.status || 'offline'))}>
                      {state?.status || 'offline'}
                    </span>
                  </div>

                  {/* Progress (if printing) */}
                  {state?.status === 'printing' && (
                    <div>
                      <div className="flex items-center justify-between text-sm mb-1">
                        <span className="text-surface-500">Progress</span>
                        <span className="text-surface-300">{state.progress.toFixed(1)}%</span>
                      </div>
                      <div className="h-2 bg-surface-800 rounded-full overflow-hidden">
                        <div 
                          className="h-full bg-emerald-500 transition-all"
                          style={{ width: `${state.progress}%` }}
                        />
                      </div>
                      {state.current_file && (
                        <p className="text-xs text-surface-500 mt-1 truncate">
                          {state.current_file}
                        </p>
                      )}
                    </div>
                  )}

                  {/* Temperatures */}
                  {state && (state.bed_temp || state.nozzle_temp) && (
                    <div className="flex gap-4 text-sm">
                      {state.nozzle_temp !== undefined && (
                        <div>
                          <span className="text-surface-500">Nozzle: </span>
                          <span className="text-surface-300">{state.nozzle_temp.toFixed(0)}°C</span>
                        </div>
                      )}
                      {state.bed_temp !== undefined && (
                        <div>
                          <span className="text-surface-500">Bed: </span>
                          <span className="text-surface-300">{state.bed_temp.toFixed(0)}°C</span>
                        </div>
                      )}
                    </div>
                  )}

                  {/* Connection */}
                  <div className="flex items-center gap-2 text-sm">
                    {printer.connection_type === 'manual' ? (
                      <WifiOff className="h-4 w-4 text-surface-500" />
                    ) : (
                      <Wifi className={cn(
                        'h-4 w-4',
                        state?.status && state.status !== 'offline' 
                          ? 'text-emerald-400' 
                          : 'text-surface-500'
                      )} />
                    )}
                    <span className="text-surface-500">
                      {printer.connection_type === 'manual' 
                        ? 'Manual' 
                        : printer.connection_type.replace('_', ' ')}
                    </span>
                  </div>

                  {/* Location */}
                  {printer.location && (
                    <div className="text-sm text-surface-500">
                      📍 {printer.location}
                    </div>
                  )}
                </div>
              </div>
            )
          })}
        </div>
      )}

      {/* Discover Printers Modal */}
      {showDiscover && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
          <div className="card w-full max-w-2xl p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-xl font-semibold text-surface-100">
                Network Discovery
              </h2>
              <button
                onClick={() => setShowDiscover(false)}
                className="text-surface-500 hover:text-surface-300"
              >
                ✕
              </button>
            </div>
            
            {discovering ? (
              <div className="text-center py-12">
                <Loader2 className="h-12 w-12 mx-auto mb-4 text-accent-500 animate-spin" />
                <p className="text-surface-300">Scanning your network...</p>
                <p className="text-surface-500 text-sm mt-1">
                  Looking for OctoPrint, Moonraker, and Bambu printers
                </p>
              </div>
            ) : discovered.length === 0 ? (
              <div className="text-center py-12">
                <Radar className="h-12 w-12 mx-auto mb-4 text-surface-600" />
                <p className="text-surface-300">No printers found</p>
                <p className="text-surface-500 text-sm mt-1">
                  Make sure your printers are powered on and connected to the network
                </p>
                <button
                  onClick={handleDiscover}
                  className="btn btn-secondary mt-4"
                >
                  <Radar className="h-4 w-4 mr-2" />
                  Scan Again
                </button>
              </div>
            ) : (
              <div className="space-y-3">
                <p className="text-surface-400 text-sm mb-4">
                  Found {discovered.length} printer{discovered.length !== 1 ? 's' : ''} on your network
                </p>
                {discovered.map((printer) => (
                  <div 
                    key={printer.id}
                    className="flex items-center justify-between p-4 rounded-lg bg-surface-800/50 border border-surface-700"
                  >
                    <div className="flex items-center gap-4">
                      <div className={cn(
                        'p-2 rounded-lg',
                        printer.type === 'octoprint' ? 'bg-green-500/20' :
                        printer.type === 'moonraker' ? 'bg-purple-500/20' :
                        printer.type === 'bambu_lan' ? 'bg-blue-500/20' :
                        'bg-surface-700'
                      )}>
                        <PrinterIcon className={cn(
                          'h-5 w-5',
                          printer.type === 'octoprint' ? 'text-green-400' :
                          printer.type === 'moonraker' ? 'text-purple-400' :
                          printer.type === 'bambu_lan' ? 'text-blue-400' :
                          'text-surface-400'
                        )} />
                      </div>
                      <div>
                        <div className="font-medium text-surface-100">
                          {printer.name}
                        </div>
                        <div className="text-sm text-surface-500">
                          {printer.host}:{printer.port} • {printer.type.replace('_', ' ')}
                          {printer.version && ` • ${printer.version}`}
                        </div>
                      </div>
                    </div>
                    <div>
                      {printer.already_added ? (
                        <span className="flex items-center gap-1 text-emerald-400 text-sm">
                          <Check className="h-4 w-4" />
                          Added
                        </span>
                      ) : (
                        <button
                          onClick={() => handleAddDiscovered(printer)}
                          disabled={addingPrinter === printer.id}
                          className="btn btn-primary text-sm py-1.5 px-3"
                        >
                          {addingPrinter === printer.id ? (
                            <>
                              <Loader2 className="h-4 w-4 mr-1 animate-spin" />
                              Adding...
                            </>
                          ) : (
                            <>
                              <Plus className="h-4 w-4 mr-1" />
                              Add
                            </>
                          )}
                        </button>
                      )}
                    </div>
                  </div>
                ))}
                <div className="flex justify-between items-center mt-4 pt-4 border-t border-surface-800">
                  <button
                    onClick={handleDiscover}
                    className="btn btn-ghost text-sm"
                  >
                    <Radar className="h-4 w-4 mr-2" />
                    Scan Again
                  </button>
                  <button
                    onClick={() => setShowDiscover(false)}
                    className="btn btn-secondary"
                  >
                    Done
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* Add Printer Modal */}
      {showAdd && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
          <div className="card w-full max-w-lg p-6">
            <h2 className="text-xl font-semibold text-surface-100 mb-4">
              Add Printer
            </h2>
            <form onSubmit={handleCreate}>
              <div className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-surface-300 mb-1">
                      Name *
                    </label>
                    <input
                      type="text"
                      name="name"
                      required
                      className="input"
                      placeholder="Prusa MK4 #1"
                      autoFocus
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-surface-300 mb-1">
                      Model
                    </label>
                    <input
                      type="text"
                      name="model"
                      className="input"
                      placeholder="Prusa MK4"
                    />
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-surface-300 mb-1">
                      Manufacturer
                    </label>
                    <input
                      type="text"
                      name="manufacturer"
                      className="input"
                      placeholder="Prusa Research"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-surface-300 mb-1">
                      Location
                    </label>
                    <input
                      type="text"
                      name="location"
                      className="input"
                      placeholder="Workshop, Desk 3"
                    />
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-medium text-surface-300 mb-1">
                    Connection Type
                  </label>
                  <select name="connection_type" className="input">
                    <option value="manual">Manual (No Integration)</option>
                    <option value="octoprint">OctoPrint</option>
                    <option value="bambu_lan">Bambu Lab (LAN)</option>
                    <option value="moonraker">Moonraker (Klipper)</option>
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium text-surface-300 mb-1">
                    Connection URL
                  </label>
                  <input
                    type="text"
                    name="connection_uri"
                    className="input"
                    placeholder="http://192.168.1.100"
                  />
                  <p className="text-xs text-surface-500 mt-1">
                    Leave empty for manual printers
                  </p>
                </div>
              </div>
              <div className="flex justify-end gap-3 mt-6">
                <button
                  type="button"
                  onClick={() => setShowAdd(false)}
                  className="btn btn-ghost"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={createPrinter.isPending}
                  className="btn btn-primary"
                >
                  {createPrinter.isPending ? 'Adding...' : 'Add Printer'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  )
}

