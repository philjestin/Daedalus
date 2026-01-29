import { useState } from 'react'
import { Link } from 'react-router-dom'
import {
  Plus,
  Printer as PrinterIcon,
  Wifi,
  WifiOff,
  Trash2,
  Radar,
  Loader2,
  Check,
  Key,
  Cloud
} from 'lucide-react'
import { usePrinters, useCreatePrinter, useDeletePrinter, usePrinterStates } from '../hooks/usePrinters'
import { printersApi, bambuCloudApi, type DiscoveredPrinter } from '../api/client'
import { cn, getStatusBadge } from '../lib/utils'
import type { ConnectionType, CloudDevice } from '../types'

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
  const [bambuSetup, setBambuSetup] = useState<string | null>(null) // printer.id being set up
  const [accessCode, setAccessCode] = useState('')
  const [serialNumber, setSerialNumber] = useState('')
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null)

  // Bambu Cloud state
  const [showCloud, setShowCloud] = useState(false)
  const [cloudStep, setCloudStep] = useState<'login' | 'verify' | 'devices'>('login')
  const [cloudEmail, setCloudEmail] = useState('')
  const [cloudPassword, setCloudPassword] = useState('')
  const [cloudCode, setCloudCode] = useState('')
  const [cloudDevices, setCloudDevices] = useState<CloudDevice[]>([])
  const [cloudLoading, setCloudLoading] = useState(false)
  const [cloudError, setCloudError] = useState('')
  const [addingCloudDevice, setAddingCloudDevice] = useState<string | null>(null)
  const [addedCloudDevices, setAddedCloudDevices] = useState<Set<string>>(new Set())

  const handleCloudLogin = async () => {
    setCloudLoading(true)
    setCloudError('')
    try {
      const res = await bambuCloudApi.login(cloudEmail, cloudPassword)
      if (res.status === 'verify_code_required') {
        setCloudStep('verify')
      } else {
        // Direct login succeeded, fetch devices
        const devices = await bambuCloudApi.devices()
        setCloudDevices(devices || [])
        setCloudStep('devices')
      }
    } catch (err) {
      setCloudError(err instanceof Error ? err.message : 'Login failed')
    } finally {
      setCloudLoading(false)
    }
  }

  const handleCloudVerify = async () => {
    setCloudLoading(true)
    setCloudError('')
    try {
      await bambuCloudApi.verify(cloudEmail, cloudCode)
      const devices = await bambuCloudApi.devices()
      setCloudDevices(devices || [])
      setCloudStep('devices')
    } catch (err) {
      setCloudError(err instanceof Error ? err.message : 'Verification failed')
    } finally {
      setCloudLoading(false)
    }
  }

  const handleAddCloudDevice = async (devId: string) => {
    setAddingCloudDevice(devId)
    setCloudError('')
    try {
      await bambuCloudApi.addDevice(devId)
      setAddedCloudDevices(prev => new Set(prev).add(devId))
      refetch()
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to add device'
      setCloudError(msg)
      console.error('Failed to add cloud device:', err)
    } finally {
      setAddingCloudDevice(null)
    }
  }

  const resetCloudFlow = () => {
    setShowCloud(false)
    setCloudStep('login')
    setCloudEmail('')
    setCloudPassword('')
    setCloudCode('')
    setCloudDevices([])
    setCloudError('')
    setAddedCloudDevices(new Set())
  }

  const handleDiscover = async () => {
    setShowDiscover(true)
    setDiscovering(true)
    setDiscovered([])
    
    try {
      console.log('[Printers] Starting discovery...')
      const found = await printersApi.discover()
      console.log('[Printers] Discovery complete, found:', found)
      setDiscovered(found || [])
    } catch (err) {
      console.error('[Printers] Discovery failed:', err)
    } finally {
      console.log('[Printers] Discovery finished')
      setDiscovering(false)
    }
  }

  const handleAddDiscovered = async (printer: DiscoveredPrinter) => {
    // Bambu printers need an access code before adding
    if (printer.type === 'bambu_lan') {
      setBambuSetup(printer.id)
      setAccessCode('')
      return
    }

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

  const handleAddBambu = async (printer: DiscoveredPrinter) => {
    if (!serialNumber.trim()) return
    setAddingPrinter(printer.id)
    try {
      const req: Record<string, unknown> = {
        name: printer.name,
        model: printer.model || '',
        manufacturer: printer.manufacturer || '',
        connection_type: printer.type,
        connection_uri: printer.host,
        serial_number: serialNumber.trim(),
      }
      if (accessCode.trim()) {
        req.api_key = accessCode.trim()
      }
      await createPrinter.mutateAsync(req as Parameters<typeof createPrinter.mutateAsync>[0])
      setDiscovered(prev =>
        prev.map(p => p.id === printer.id ? { ...p, already_added: true } : p)
      )
      setBambuSetup(null)
      setAccessCode('')
      setSerialNumber('')
      refetch()
    } catch (err) {
      console.error('Failed to add Bambu printer:', err)
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
      cost_per_hour_cents: Math.round(parseFloat(formData.get('cost_per_hour') as string || '0') * 100),
    })
    
    setShowAdd(false)
  }

  const handleDelete = async (id: string) => {
    if (confirmDelete !== id) {
      setConfirmDelete(id)
      return
    }
    await deletePrinter.mutateAsync(id)
    setConfirmDelete(null)
  }

  return (
    <div className="p-4 sm:p-6 lg:p-8">
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
            onClick={() => setShowCloud(true)}
            className="btn btn-secondary"
          >
            <Cloud className="h-4 w-4 mr-2" />
            Bambu Cloud
          </button>
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
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
          {printers.map((printer) => {
            const state = printerStates[printer.id]
            return (
              <Link key={printer.id} to={`/printers/${printer.id}`} className="card p-5 block hover:border-surface-600 transition-colors">
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
                  {confirmDelete === printer.id ? (
                    <div className="flex items-center gap-1" onClick={(e) => { e.preventDefault(); e.stopPropagation() }}>
                      <button
                        onClick={() => setConfirmDelete(null)}
                        className="text-xs text-surface-500 hover:text-surface-300 px-1.5 py-0.5"
                      >
                        Cancel
                      </button>
                      <button
                        onClick={() => handleDelete(printer.id)}
                        className="text-xs bg-red-500/20 text-red-400 hover:bg-red-500/30 rounded px-2 py-0.5"
                      >
                        Delete
                      </button>
                    </div>
                  ) : (
                    <button
                      onClick={(e) => { e.preventDefault(); e.stopPropagation(); handleDelete(printer.id) }}
                      className="p-1.5 rounded hover:bg-surface-800 text-surface-500 hover:text-red-400 transition-colors"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  )}
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
              </Link>
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
                <p className="text-surface-600 text-xs mt-3">
                  This can take 15-20 seconds. Please wait...
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
                {discovered.map((printer) => {
                  const isBambu = printer.type === 'bambu_lan'
                  return (
                  <div
                    key={printer.id}
                    className="rounded-lg bg-surface-800/50 border border-surface-700"
                  >
                    <div className="flex items-center justify-between p-4">
                      <div className="flex items-center gap-4">
                        <div className={cn(
                          'p-2 rounded-lg',
                          printer.type === 'octoprint' ? 'bg-green-500/20' :
                          printer.type === 'moonraker' ? 'bg-purple-500/20' :
                          isBambu ? 'bg-blue-500/20' :
                          'bg-surface-700'
                        )}>
                          <PrinterIcon className={cn(
                            'h-5 w-5',
                            printer.type === 'octoprint' ? 'text-green-400' :
                            printer.type === 'moonraker' ? 'text-purple-400' :
                            isBambu ? 'text-blue-400' :
                            'text-surface-400'
                          )} />
                        </div>
                        <div>
                          <div className="font-medium text-surface-100">
                            {printer.name}
                          </div>
                          <div className="text-sm text-surface-500">
                            {printer.host}:{printer.port} • {printer.type.replace('_', ' ')}
                            {printer.model && ` • ${printer.model}`}
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
                        ) : !isBambu ? (
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
                        ) : null}
                      </div>
                    </div>

                    {/* Bambu access code form — always visible for Bambu printers */}
                    {isBambu && !printer.already_added && (
                      <div className="px-4 pb-4 pt-0 border-t border-surface-700">
                        <div className="mt-3 space-y-3">
                          <div>
                            <label className="block text-sm font-medium text-surface-300 mb-1">
                              <Key className="h-3.5 w-3.5 inline mr-1" />
                              LAN Access Code <span className="text-surface-500 font-normal">(optional)</span>
                            </label>
                            <input
                              type="text"
                              value={bambuSetup === printer.id ? accessCode : ''}
                              onChange={(e) => { setBambuSetup(printer.id); setAccessCode(e.target.value) }}
                              onFocus={() => {
                                if (bambuSetup !== printer.id) {
                                  setBambuSetup(printer.id)
                                  setAccessCode('')
                                  setSerialNumber(printer.serial_number || '')
                                }
                              }}
                              className="input"
                              placeholder="Enter 8-digit access code"
                            />
                            <p className="text-xs text-surface-500 mt-1">
                              Not all models require this. Check Bambu Handy app, Bambu Studio, or printer LCD.
                            </p>
                          </div>
                          <div>
                            <label className="block text-sm font-medium text-surface-300 mb-1">
                              Serial Number
                            </label>
                            <input
                              type="text"
                              value={bambuSetup === printer.id ? serialNumber : (printer.serial_number || '')}
                              onChange={(e) => { setBambuSetup(printer.id); setSerialNumber(e.target.value) }}
                              onFocus={() => {
                                if (bambuSetup !== printer.id) {
                                  setBambuSetup(printer.id)
                                  setAccessCode('')
                                  setSerialNumber(printer.serial_number || '')
                                }
                              }}
                              className="input"
                              placeholder="e.g. 01P00A000000000"
                            />
                            <p className="text-xs text-surface-500 mt-1">
                              {printer.serial_number
                                ? 'Auto-detected. Edit if incorrect.'
                                : 'Find on the printer sticker, in Bambu Handy, or Bambu Studio'}
                            </p>
                          </div>
                          <div className="flex justify-end">
                            <button
                              onClick={() => handleAddBambu(printer)}
                              disabled={
                                !(bambuSetup === printer.id && serialNumber.trim()) ||
                                addingPrinter === printer.id
                              }
                              className="btn btn-primary text-sm py-1.5 px-3"
                            >
                              {addingPrinter === printer.id ? (
                                <>
                                  <Loader2 className="h-4 w-4 mr-1 animate-spin" />
                                  Connecting...
                                </>
                              ) : (
                                <>
                                  <Check className="h-4 w-4 mr-1" />
                                  Connect
                                </>
                              )}
                            </button>
                          </div>
                        </div>
                      </div>
                    )}
                  </div>
                  )
                })}
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

      {/* Bambu Cloud Modal */}
      {showCloud && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
          <div className="card w-full max-w-lg p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-xl font-semibold text-surface-100">
                {cloudStep === 'login' && 'Connect Bambu Cloud'}
                {cloudStep === 'verify' && 'Enter Verification Code'}
                {cloudStep === 'devices' && 'Your Bambu Printers'}
              </h2>
              <button
                onClick={resetCloudFlow}
                className="text-surface-500 hover:text-surface-300"
              >
                ✕
              </button>
            </div>

            {cloudError && (
              <div className="bg-red-500/10 border border-red-500/20 rounded-lg p-3 mb-4 text-sm text-red-400">
                {cloudError}
              </div>
            )}

            {cloudStep === 'login' && (
              <div className="space-y-4">
                <p className="text-sm text-surface-400">
                  Sign in with your Bambu Lab account to connect printers via cloud MQTT.
                  This works even when printers are not in LAN-only mode.
                </p>
                <div>
                  <label className="block text-sm font-medium text-surface-300 mb-1">
                    Email
                  </label>
                  <input
                    type="email"
                    value={cloudEmail}
                    onChange={e => setCloudEmail(e.target.value)}
                    className="input"
                    placeholder="your@email.com"
                    autoFocus
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-surface-300 mb-1">
                    Password
                  </label>
                  <input
                    type="password"
                    value={cloudPassword}
                    onChange={e => setCloudPassword(e.target.value)}
                    className="input"
                    placeholder="Password"
                    onKeyDown={e => e.key === 'Enter' && cloudEmail && cloudPassword && handleCloudLogin()}
                  />
                </div>
                <div className="flex justify-end gap-3">
                  <button onClick={resetCloudFlow} className="btn btn-ghost">
                    Cancel
                  </button>
                  <button
                    onClick={handleCloudLogin}
                    disabled={cloudLoading || !cloudEmail || !cloudPassword}
                    className="btn btn-primary"
                  >
                    {cloudLoading ? (
                      <>
                        <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                        Signing in...
                      </>
                    ) : (
                      'Sign In'
                    )}
                  </button>
                </div>
              </div>
            )}

            {cloudStep === 'verify' && (
              <div className="space-y-4">
                <p className="text-sm text-surface-400">
                  A verification code has been sent to <strong className="text-surface-200">{cloudEmail}</strong>.
                  Enter the 6-digit code below.
                </p>
                <div>
                  <label className="block text-sm font-medium text-surface-300 mb-1">
                    Verification Code
                  </label>
                  <input
                    type="text"
                    value={cloudCode}
                    onChange={e => setCloudCode(e.target.value)}
                    className="input text-center text-2xl tracking-widest"
                    placeholder="000000"
                    maxLength={6}
                    autoFocus
                    onKeyDown={e => e.key === 'Enter' && cloudCode.length >= 6 && handleCloudVerify()}
                  />
                </div>
                <div className="flex justify-end gap-3">
                  <button onClick={() => setCloudStep('login')} className="btn btn-ghost">
                    Back
                  </button>
                  <button
                    onClick={handleCloudVerify}
                    disabled={cloudLoading || cloudCode.length < 6}
                    className="btn btn-primary"
                  >
                    {cloudLoading ? (
                      <>
                        <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                        Verifying...
                      </>
                    ) : (
                      'Verify'
                    )}
                  </button>
                </div>
              </div>
            )}

            {cloudStep === 'devices' && (
              <div className="space-y-3">
                {cloudDevices.length === 0 ? (
                  <p className="text-surface-400 text-center py-8">
                    No printers found on your Bambu account.
                  </p>
                ) : (
                  <>
                    <p className="text-sm text-surface-400">
                      Found {cloudDevices.length} printer{cloudDevices.length !== 1 ? 's' : ''} on your account.
                      Click Add to connect via cloud MQTT.
                    </p>
                    {cloudDevices.map(device => (
                      <div
                        key={device.dev_id}
                        className="flex items-center justify-between p-4 rounded-lg bg-surface-800/50 border border-surface-700"
                      >
                        <div className="flex items-center gap-3">
                          <div className={cn(
                            'p-2 rounded-lg',
                            device.online ? 'bg-emerald-500/20' : 'bg-surface-700'
                          )}>
                            <PrinterIcon className={cn(
                              'h-5 w-5',
                              device.online ? 'text-emerald-400' : 'text-surface-500'
                            )} />
                          </div>
                          <div>
                            <div className="font-medium text-surface-100">
                              {device.name}
                            </div>
                            <div className="text-sm text-surface-500">
                              {device.dev_product_name || device.dev_model_name}
                              {device.online ? (
                                <span className="text-emerald-400 ml-2">Online</span>
                              ) : (
                                <span className="text-surface-600 ml-2">Offline</span>
                              )}
                            </div>
                          </div>
                        </div>
                        {addedCloudDevices.has(device.dev_id) ? (
                          <span className="flex items-center gap-1 text-emerald-400 text-sm">
                            <Check className="h-4 w-4" />
                            Added
                          </span>
                        ) : (
                          <button
                            onClick={() => handleAddCloudDevice(device.dev_id)}
                            disabled={addingCloudDevice === device.dev_id}
                            className="btn btn-primary text-sm py-1.5 px-3"
                          >
                            {addingCloudDevice === device.dev_id ? (
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
                    ))}
                  </>
                )}
                <div className="flex justify-end pt-4 border-t border-surface-800">
                  <button onClick={resetCloudFlow} className="btn btn-secondary">
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
                    Cost per Hour ($)
                  </label>
                  <input
                    type="number"
                    step="0.01"
                    name="cost_per_hour"
                    className="input"
                    placeholder="0.50"
                    defaultValue="0.50"
                  />
                  <p className="text-xs text-surface-500 mt-1">
                    Suggested: A1 $0.50 / P1S $0.75 / X1 $1.00 — covers electricity, depreciation, maintenance, and utilization.
                    You can adjust this later from the printer detail page.
                  </p>
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

