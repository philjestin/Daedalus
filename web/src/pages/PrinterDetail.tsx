import { useParams, Link } from 'react-router-dom'
import { ArrowLeft, Wifi, Thermometer, Fan, Gauge, Info, AlertTriangle, Lightbulb, Box, History, CheckCircle, XCircle, Printer as PrinterIcon, Clock } from 'lucide-react'
import { usePrinter, usePrinterState, usePrinterJobs, usePrinterStats } from '../hooks/usePrinters'
import { cn, getStatusBadge, formatDuration, formatRelativeTime } from '../lib/utils'
import { ExpandableJobEvents } from '../components/JobEventTimeline'
import type { PrintJob } from '../types'

const SPEED_LABELS: Record<number, string> = {
  1: 'Silent',
  2: 'Standard',
  3: 'Sport',
  4: 'Ludicrous',
}

function wifiStrength(signal: string): string {
  const dbm = parseInt(signal, 10)
  if (isNaN(dbm)) return signal
  if (dbm >= -50) return `${signal} dBm (Excellent)`
  if (dbm >= -60) return `${signal} dBm (Good)`
  if (dbm >= -70) return `${signal} dBm (Fair)`
  return `${signal} dBm (Weak)`
}

function FanBar({ label, percent }: { label: string; percent: number }) {
  return (
    <div>
      <div className="flex items-center justify-between text-sm mb-1">
        <span className="text-surface-400">{label}</span>
        <span className="text-surface-200">{percent}%</span>
      </div>
      <div className="h-2 bg-surface-800 rounded-full overflow-hidden">
        <div
          className="h-full bg-blue-500 transition-all"
          style={{ width: `${percent}%` }}
        />
      </div>
    </div>
  )
}

function TempRow({ label, current, target }: { label: string; current?: number; target?: number }) {
  if (current === undefined && target === undefined) return null
  const cur = current ?? 0
  const tgt = target ?? 0
  const max = Math.max(tgt, 300)
  const pct = max > 0 ? Math.min((cur / max) * 100, 100) : 0

  return (
    <div>
      <div className="flex items-center justify-between text-sm mb-1">
        <span className="text-surface-400">{label}</span>
        <span className="text-surface-200">
          {cur.toFixed(0)}{target !== undefined && target > 0 ? ` / ${target.toFixed(0)}` : ''}&deg;C
        </span>
      </div>
      <div className="h-2 bg-surface-800 rounded-full overflow-hidden">
        <div
          className={cn(
            'h-full transition-all',
            cur > 0 && tgt > 0 && cur >= tgt - 2 ? 'bg-emerald-500' :
            cur > 50 ? 'bg-amber-500' :
            'bg-surface-600'
          )}
          style={{ width: `${pct}%` }}
        />
      </div>
    </div>
  )
}

export default function PrinterDetail() {
  const { id } = useParams<{ id: string }>()
  const { data: printer, isLoading: printerLoading } = usePrinter(id!)
  const { data: state } = usePrinterState(id!)

  if (printerLoading) {
    return (
      <div className="p-4 sm:p-6 lg:p-8">
        <div className="text-surface-500">Loading...</div>
      </div>
    )
  }

  if (!printer) {
    return (
      <div className="p-4 sm:p-6 lg:p-8">
        <Link to="/printers" className="text-accent-400 hover:text-accent-300 flex items-center gap-1 mb-4">
          <ArrowLeft className="h-4 w-4" />
          Printers
        </Link>
        <div className="text-surface-400">Printer not found.</div>
      </div>
    )
  }

  const status = state?.status || 'offline'
  const isPrinting = status === 'printing'
  const hasTemps = state && (state.bed_temp || state.nozzle_temp || state.chamber_temp)
  const hasFans = state && (state.cooling_fan_speed || state.aux_fan_speed || state.chamber_fan_speed || state.heatbreak_fan_speed)
  const hasAMS = state?.ams && state.ams.units.length > 0
  const hasHMS = state?.hms_errors && state.hms_errors.length > 0
  const hasLights = state?.lights && state.lights.length > 0

  return (
    <div className="p-4 sm:p-6 lg:p-8">
      {/* Header */}
      <div className="mb-6">
        <Link to="/printers" className="text-accent-400 hover:text-accent-300 flex items-center gap-1 mb-3 text-sm">
          <ArrowLeft className="h-4 w-4" />
          Printers
        </Link>
        <div className="flex items-start justify-between">
          <div>
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-display font-bold text-surface-100">
                {printer.name}
              </h1>
              <span className={cn('badge', getStatusBadge(status))}>
                {status}
              </span>
              <span className="badge bg-surface-800 text-surface-400">
                {printer.connection_type.replace('_', ' ')}
              </span>
            </div>
            <p className="text-surface-500 mt-1">
              {[printer.model, printer.serial_number].filter(Boolean).join(' / ')}
            </p>
          </div>
        </div>
      </div>

      {/* 2-column grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* PRINT JOB */}
        <div className="card p-5">
          <h2 className="text-sm font-semibold text-surface-400 uppercase tracking-wider mb-4 flex items-center gap-2">
            <Gauge className="h-4 w-4" />
            Print Job
          </h2>
          {isPrinting || (state && state.progress > 0 && status !== 'idle') ? (
            <div className="space-y-3">
              {/* Progress bar */}
              <div>
                <div className="flex items-center justify-between text-sm mb-1">
                  <span className="text-surface-400">Progress</span>
                  <span className="text-surface-100 font-medium">{state!.progress.toFixed(1)}%</span>
                </div>
                <div className="h-3 bg-surface-800 rounded-full overflow-hidden">
                  <div
                    className="h-full bg-emerald-500 transition-all"
                    style={{ width: `${state!.progress}%` }}
                  />
                </div>
              </div>

              {/* Layers */}
              {state!.total_layer_num ? (
                <div className="flex items-center justify-between text-sm">
                  <span className="text-surface-400">Layer</span>
                  <span className="text-surface-200">{state!.layer_num || 0} / {state!.total_layer_num}</span>
                </div>
              ) : null}

              {/* File */}
              {state!.current_file && (
                <div className="flex items-center justify-between text-sm">
                  <span className="text-surface-400">File</span>
                  <span className="text-surface-200 truncate ml-4 text-right">{state!.current_file}</span>
                </div>
              )}

              {/* Time left */}
              {state!.time_left ? (
                <div className="flex items-center justify-between text-sm">
                  <span className="text-surface-400">Time left</span>
                  <span className="text-surface-200">{formatDuration(state!.time_left)}</span>
                </div>
              ) : null}

              {/* Speed */}
              {(state!.speed_level || state!.speed_percent) ? (
                <div className="flex items-center justify-between text-sm">
                  <span className="text-surface-400">Speed</span>
                  <span className="text-surface-200">
                    {state!.speed_level ? SPEED_LABELS[state!.speed_level] || `Level ${state!.speed_level}` : ''}
                    {state!.speed_percent ? ` (${state!.speed_percent}%)` : ''}
                  </span>
                </div>
              ) : null}
            </div>
          ) : (
            <p className="text-surface-500 text-sm">No active print</p>
          )}
        </div>

        {/* TEMPERATURES */}
        <div className="card p-5">
          <h2 className="text-sm font-semibold text-surface-400 uppercase tracking-wider mb-4 flex items-center gap-2">
            <Thermometer className="h-4 w-4" />
            Temperatures
          </h2>
          {hasTemps ? (
            <div className="space-y-3">
              <TempRow label="Nozzle" current={state!.nozzle_temp} target={state!.nozzle_target_temp} />
              <TempRow label="Bed" current={state!.bed_temp} target={state!.bed_target_temp} />
              {state!.chamber_temp !== undefined && state!.chamber_temp > 0 && (
                <TempRow label="Chamber" current={state!.chamber_temp} />
              )}
            </div>
          ) : (
            <p className="text-surface-500 text-sm">No temperature data</p>
          )}
        </div>

        {/* FAN SPEEDS */}
        {hasFans && (
          <div className="card p-5">
            <h2 className="text-sm font-semibold text-surface-400 uppercase tracking-wider mb-4 flex items-center gap-2">
              <Fan className="h-4 w-4" />
              Fan Speeds
            </h2>
            <div className="space-y-3">
              {state!.cooling_fan_speed !== undefined && <FanBar label="Part Cooling" percent={state!.cooling_fan_speed} />}
              {state!.aux_fan_speed !== undefined && <FanBar label="Aux Fan" percent={state!.aux_fan_speed} />}
              {state!.chamber_fan_speed !== undefined && <FanBar label="Chamber Fan" percent={state!.chamber_fan_speed} />}
              {state!.heatbreak_fan_speed !== undefined && <FanBar label="Heatbreak" percent={state!.heatbreak_fan_speed} />}
            </div>
          </div>
        )}

        {/* DEVICE INFO */}
        <div className="card p-5">
          <h2 className="text-sm font-semibold text-surface-400 uppercase tracking-wider mb-4 flex items-center gap-2">
            <Info className="h-4 w-4" />
            Device Info
          </h2>
          <div className="space-y-2 text-sm">
            {state?.wifi_signal && (
              <div className="flex items-center justify-between">
                <span className="text-surface-400 flex items-center gap-1.5">
                  <Wifi className="h-3.5 w-3.5" /> WiFi
                </span>
                <span className="text-surface-200">{wifiStrength(state.wifi_signal)}</span>
              </div>
            )}
            {state?.nozzle_diameter && (
              <div className="flex items-center justify-between">
                <span className="text-surface-400">Nozzle</span>
                <span className="text-surface-200">
                  {state.nozzle_diameter}mm{state.nozzle_type ? ` ${state.nozzle_type}` : ''}
                </span>
              </div>
            )}
            {printer.serial_number && (
              <div className="flex items-center justify-between">
                <span className="text-surface-400">Serial</span>
                <span className="text-surface-200 font-mono text-xs">{printer.serial_number}</span>
              </div>
            )}
            {printer.location && (
              <div className="flex items-center justify-between">
                <span className="text-surface-400">Location</span>
                <span className="text-surface-200">{printer.location}</span>
              </div>
            )}
            {printer.model && (
              <div className="flex items-center justify-between">
                <span className="text-surface-400">Model</span>
                <span className="text-surface-200">{printer.model}</span>
              </div>
            )}
            {!state?.wifi_signal && !state?.nozzle_diameter && !printer.serial_number && !printer.location && !printer.model && (
              <p className="text-surface-500">No additional info available</p>
            )}
          </div>
        </div>

        {/* AMS STATUS */}
        {hasAMS && (
          <div className="card p-5">
            <h2 className="text-sm font-semibold text-surface-400 uppercase tracking-wider mb-4 flex items-center gap-2">
              <Box className="h-4 w-4" />
              AMS Status
            </h2>
            <div className="space-y-4">
              {state!.ams!.units.map((unit) => (
                <div key={unit.id}>
                  <div className="text-xs text-surface-500 mb-2">
                    Unit {unit.id}
                    {unit.humidity > 0 && ` (${unit.humidity}% RH`}
                    {unit.temp > 0 && `, ${unit.temp.toFixed(0)}\u00B0C`}
                    {(unit.humidity > 0 || unit.temp > 0) && ')'}
                  </div>
                  <div className="space-y-1.5">
                    {unit.trays.map((tray) => (
                      <div key={tray.id} className="flex items-center gap-2 text-sm">
                        <span className="text-surface-500 w-5 text-right">[{tray.id}]</span>
                        {tray.color_hex && (
                          <span
                            className="w-3 h-3 rounded-full border border-surface-600 flex-shrink-0"
                            style={{ backgroundColor: `#${tray.color_hex.slice(0, 6)}` }}
                          />
                        )}
                        <span className="text-surface-300 flex-1 truncate">
                          {tray.empty ? (
                            <span className="text-surface-600">Empty</span>
                          ) : (
                            <>
                              {tray.material_type}
                              {tray.brand && ` ${tray.brand}`}
                            </>
                          )}
                        </span>
                        {!tray.empty && (
                          <div className="flex items-center gap-1.5 flex-shrink-0">
                            <div className="w-16 h-1.5 bg-surface-800 rounded-full overflow-hidden">
                              <div
                                className={cn(
                                  'h-full rounded-full',
                                  tray.remain > 50 ? 'bg-emerald-500' :
                                  tray.remain > 20 ? 'bg-amber-500' :
                                  'bg-red-500'
                                )}
                                style={{ width: `${tray.remain}%` }}
                              />
                            </div>
                            <span className="text-surface-400 text-xs w-8 text-right">{tray.remain}%</span>
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                </div>
              ))}
              {state!.ams!.external_spool && !state!.ams!.external_spool.empty && (
                <div>
                  <div className="text-xs text-surface-500 mb-2">External Spool</div>
                  <div className="flex items-center gap-2 text-sm">
                    {state!.ams!.external_spool.color_hex && (
                      <span
                        className="w-3 h-3 rounded-full border border-surface-600 flex-shrink-0"
                        style={{ backgroundColor: `#${state!.ams!.external_spool.color_hex.slice(0, 6)}` }}
                      />
                    )}
                    <span className="text-surface-300 flex-1">
                      {state!.ams!.external_spool.material_type}
                      {state!.ams!.external_spool.brand && ` ${state!.ams!.external_spool.brand}`}
                    </span>
                    <span className="text-surface-400 text-xs">{state!.ams!.external_spool.remain}%</span>
                  </div>
                </div>
              )}
            </div>
          </div>
        )}

        {/* ALERTS */}
        {(hasHMS || hasLights) && (
          <div className="card p-5">
            <h2 className="text-sm font-semibold text-surface-400 uppercase tracking-wider mb-4 flex items-center gap-2">
              <AlertTriangle className="h-4 w-4" />
              Alerts
            </h2>
            <div className="space-y-3 text-sm">
              {hasHMS ? (
                <div>
                  <div className="text-surface-400 mb-1">HMS Errors</div>
                  {state!.hms_errors!.map((err, i) => (
                    <div key={i} className={cn(
                      'px-2 py-1 rounded text-xs mb-1',
                      err.severity >= 3 ? 'bg-red-500/10 text-red-400' :
                      err.severity >= 2 ? 'bg-amber-500/10 text-amber-400' :
                      'bg-surface-800 text-surface-300'
                    )}>
                      Module {err.module} / Code {err.code} (severity {err.severity})
                    </div>
                  ))}
                </div>
              ) : (
                <div className="flex items-center gap-2 text-surface-400">
                  <span className="text-emerald-400">&#10003;</span> No active HMS errors
                </div>
              )}
              {hasLights && (
                <div>
                  <div className="text-surface-400 mb-1 flex items-center gap-1.5">
                    <Lightbulb className="h-3.5 w-3.5" /> Lights
                  </div>
                  {state!.lights!.map((light, i) => (
                    <div key={i} className="text-surface-300 text-xs">
                      {light.node}: {light.mode}
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        )}

        {/* Show Alerts card with "no errors" when no HMS and no lights but printer is connected */}
        {!hasHMS && !hasLights && status !== 'offline' && (
          <div className="card p-5">
            <h2 className="text-sm font-semibold text-surface-400 uppercase tracking-wider mb-4 flex items-center gap-2">
              <AlertTriangle className="h-4 w-4" />
              Alerts
            </h2>
            <div className="flex items-center gap-2 text-surface-400 text-sm">
              <span className="text-emerald-400">&#10003;</span> No active HMS errors
            </div>
          </div>
        )}
      </div>

      {/* Print History Section */}
      <PrinterHistory printerId={id!} />
    </div>
  )
}

function PrinterHistory({ printerId }: { printerId: string }) {
  const { data: stats } = usePrinterStats(printerId)
  const { data: jobs = [], isLoading } = usePrinterJobs(printerId)

  const sortedJobs = [...jobs].sort(
    (a, b) => new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
  )

  const totalMaterialUsed = jobs.reduce(
    (sum, job) => sum + (job.material_used_grams || 0),
    0
  )
  const totalCost = jobs.reduce(
    (sum, job) => sum + (job.cost_cents || 0),
    0
  )

  const fmtJobDuration = (startedAt?: string, completedAt?: string) => {
    if (!startedAt) return '-'
    const start = new Date(startedAt)
    const end = completedAt ? new Date(completedAt) : new Date()
    const diffMs = end.getTime() - start.getTime()
    const hours = Math.floor(diffMs / (1000 * 60 * 60))
    const mins = Math.floor((diffMs % (1000 * 60 * 60)) / (1000 * 60))
    if (hours > 0) return `${hours}h ${mins}m`
    return `${mins}m`
  }

  return (
    <div className="mt-6">
      <h2 className="text-lg font-semibold text-surface-100 flex items-center gap-2 mb-4">
        <History className="h-5 w-5 text-surface-400" />
        Print History
      </h2>

      {/* Stats Cards */}
      {stats && stats.total > 0 && (
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-4">
          <div className="card p-4">
            <div className="text-sm text-surface-500 mb-1">Total Prints</div>
            <div className="text-2xl font-semibold text-surface-100">
              {stats.total}
            </div>
          </div>
          <div className="card p-4">
            <div className="text-sm text-surface-500 mb-1">Success Rate</div>
            <div className="text-2xl font-semibold text-emerald-400">
              {(stats.completed + stats.failed) > 0
                ? Math.round((stats.completed / (stats.completed + stats.failed)) * 100)
                : 0}%
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
              ${(totalCost / 100).toFixed(2)}
            </div>
          </div>
        </div>
      )}

      {/* Job List */}
      {isLoading ? (
        <div className="text-surface-500 text-sm">Loading print history...</div>
      ) : sortedJobs.length === 0 ? (
        <div className="card p-8 text-center">
          <History className="h-12 w-12 mx-auto mb-3 text-surface-600" />
          <h3 className="text-lg font-medium text-surface-300 mb-2">
            No print history
          </h3>
          <p className="text-surface-500">
            Print jobs for this printer will appear here
          </p>
        </div>
      ) : (
        <div className="space-y-3">
          {sortedJobs.map((job) => (
            <PrinterJobRow key={job.id} job={job} fmtJobDuration={fmtJobDuration} />
          ))}
        </div>
      )}
    </div>
  )
}

function PrinterJobRow({
  job,
  fmtJobDuration,
}: {
  job: PrintJob
  fmtJobDuration: (startedAt?: string, completedAt?: string) => string
}) {
  return (
    <div className="card p-4">
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
            ) : job.status === 'failed' || (job.outcome && !job.outcome.success) ? (
              <XCircle className="h-5 w-5" />
            ) : job.status === 'printing' ? (
              <PrinterIcon className="h-5 w-5" />
            ) : (
              <Clock className="h-5 w-5" />
            )}
          </div>

          <div>
            <div className="font-medium text-surface-100">
              {job.design_id ? `Job #${job.id.slice(0, 8)}` : 'Unknown'}
            </div>
            <div className="text-sm text-surface-500 flex items-center gap-2">
              <span>{formatRelativeTime(job.created_at)}</span>
              {job.started_at && (
                <>
                  <span>·</span>
                  <span>{fmtJobDuration(job.started_at, job.completed_at)}</span>
                </>
              )}
              {job.attempt_number > 1 && (
                <>
                  <span>·</span>
                  <span className="text-amber-400">Attempt #{job.attempt_number}</span>
                </>
              )}
            </div>
          </div>
        </div>

        <div className="flex items-center gap-3">
          {job.material_used_grams != null && job.material_used_grams > 0 && (
            <span className="text-sm text-surface-400">
              {job.material_used_grams.toFixed(1)}g
            </span>
          )}
          {job.cost_cents != null && job.cost_cents > 0 && (
            <span className="text-sm text-emerald-400">
              ${(job.cost_cents / 100).toFixed(2)}
            </span>
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
        </div>
      </div>

      {/* Expandable event timeline */}
      <div className="mt-3 pt-3 border-t border-surface-800">
        <ExpandableJobEvents jobId={job.id} />
      </div>
    </div>
  )
}
