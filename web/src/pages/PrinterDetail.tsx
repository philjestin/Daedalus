import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { ArrowLeft, Wifi, Thermometer, Fan, Gauge, Info, AlertTriangle, Lightbulb, Box, History, CheckCircle, XCircle, Printer as PrinterIcon, Clock, DollarSign, Check, X, Activity, TrendingUp, Target, Heart } from 'lucide-react'
import { usePrinter, usePrinterState, usePrinterJobs, usePrinterStats, useUpdatePrinter, usePrinterAnalytics } from '../hooks/usePrinters'
import { cn, getStatusBadge, formatDuration, formatRelativeTime } from '../lib/utils'
import { ExpandableJobEvents } from '../components/JobEventTimeline'
import AutoDispatchSettings from '../components/AutoDispatchSettings'
import type { PrintJob, PrinterUtilization, PrinterROI, PrinterHealth } from '../types'

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
  const { data: analytics } = usePrinterAnalytics(id!)

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

        {/* COST SETTINGS */}
        <PrinterCostSetting printerId={printer.id} costPerHourCents={printer.cost_per_hour_cents} printerModel={printer.model} />

        {/* AUTO-DISPATCH SETTINGS */}
        <AutoDispatchSettings printerId={printer.id} />

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

      {/* Analytics Section */}
      {analytics && (
        <div className="mt-6">
          <h2 className="text-lg font-semibold text-surface-100 flex items-center gap-2 mb-4">
            <TrendingUp className="h-5 w-5 text-surface-400" />
            Analytics
          </h2>
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
            <PrinterUtilizationCard utilization={analytics.utilization} />
            <PrinterROICard roi={analytics.roi} printerId={id!} purchasePriceCents={printer.purchase_price_cents} />
            <PrinterHealthCard health={analytics.health} />
          </div>
        </div>
      )}

      {/* Print History Section */}
      <PrinterHistory printerId={id!} />
    </div>
  )
}

function PrinterCostSetting({ printerId, costPerHourCents, printerModel }: { printerId: string; costPerHourCents: number; printerModel: string }) {
  const [editing, setEditing] = useState(false)
  const [value, setValue] = useState((costPerHourCents / 100).toFixed(2))
  const [showBreakdown, setShowBreakdown] = useState(false)
  const updatePrinter = useUpdatePrinter()

  const handleSave = async () => {
    const cents = Math.round(parseFloat(value || '0') * 100)
    await updatePrinter.mutateAsync({ id: printerId, data: { cost_per_hour_cents: cents } })
    setEditing(false)
  }

  const handleCancel = () => {
    setValue((costPerHourCents / 100).toFixed(2))
    setEditing(false)
  }

  const handleApplyDefault = async (cents: number) => {
    await updatePrinter.mutateAsync({ id: printerId, data: { cost_per_hour_cents: cents } })
    setValue((cents / 100).toFixed(2))
  }

  // Detect model tier for suggested defaults
  const modelLower = printerModel.toLowerCase()
  const isP1Series = modelLower.includes('p1s') || modelLower.includes('p1p')
  const isX1Series = modelLower.includes('x1')
  const suggestedLabel = isX1Series ? 'X1' : isP1Series ? 'P1S' : 'A1'

  return (
    <div className="card p-5">
      <h2 className="text-sm font-semibold text-surface-400 uppercase tracking-wider mb-4 flex items-center gap-2">
        <DollarSign className="h-4 w-4" />
        Cost Settings
      </h2>
      <div className="space-y-4">
        <div>
          <div className="flex items-center justify-between mb-1">
            <span className="text-sm text-surface-400">Cost per Hour</span>
            {!editing && (
              <button
                onClick={() => setEditing(true)}
                className="text-xs text-accent-400 hover:text-accent-300"
              >
                Edit
              </button>
            )}
          </div>
          {editing ? (
            <div className="flex items-center gap-2">
              <span className="text-surface-400">$</span>
              <input
                type="number"
                step="0.01"
                min="0"
                value={value}
                onChange={(e) => setValue(e.target.value)}
                className="input w-28 text-sm"
                autoFocus
                onKeyDown={(e) => {
                  if (e.key === 'Enter') handleSave()
                  if (e.key === 'Escape') handleCancel()
                }}
              />
              <span className="text-surface-500 text-sm">/ hr</span>
              <button
                onClick={handleSave}
                disabled={updatePrinter.isPending}
                className="p-1 rounded text-emerald-400 hover:bg-emerald-500/10"
              >
                <Check className="h-4 w-4" />
              </button>
              <button
                onClick={handleCancel}
                className="p-1 rounded text-surface-400 hover:bg-surface-700"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
          ) : (
            <div className="text-2xl font-semibold text-surface-100">
              ${(costPerHourCents / 100).toFixed(2)}
              <span className="text-sm font-normal text-surface-500 ml-1">/ hr</span>
            </div>
          )}
        </div>

        {/* Suggested defaults */}
        {costPerHourCents === 0 && (
          <div className="p-3 rounded-lg bg-amber-500/10 border border-amber-500/20">
            <div className="text-sm text-amber-300 font-medium mb-2">No cost set</div>
            <p className="text-xs text-surface-400 mb-3">
              Set an hourly rate so project analytics can calculate printer time costs accurately.
            </p>
            <div className="flex flex-wrap gap-2">
              <button
                onClick={() => handleApplyDefault(50)}
                className={cn('btn btn-ghost text-xs py-1 px-3', suggestedLabel === 'A1' && 'ring-1 ring-accent-500')}
              >
                A1 tier — $0.50/hr
              </button>
              <button
                onClick={() => handleApplyDefault(75)}
                className={cn('btn btn-ghost text-xs py-1 px-3', suggestedLabel === 'P1S' && 'ring-1 ring-accent-500')}
              >
                P1S tier — $0.75/hr
              </button>
              <button
                onClick={() => handleApplyDefault(100)}
                className={cn('btn btn-ghost text-xs py-1 px-3', suggestedLabel === 'X1' && 'ring-1 ring-accent-500')}
              >
                X1 tier — $1.00/hr
              </button>
            </div>
            {printerModel && (
              <p className="text-xs text-surface-500 mt-2">
                Based on your model ({printerModel}), the {suggestedLabel} tier is highlighted.
              </p>
            )}
          </div>
        )}

        {/* Cost breakdown explainer */}
        <div>
          <button
            onClick={() => setShowBreakdown(!showBreakdown)}
            className="text-xs text-surface-500 hover:text-surface-300 transition-colors"
          >
            {showBreakdown ? 'Hide' : 'How is this cost calculated?'}
          </button>
          {showBreakdown && (
            <div className="mt-3 p-3 rounded-lg bg-surface-800/50 border border-surface-700 text-xs text-surface-400 space-y-3">
              <p>
                Hourly cost covers everything except filament (which is tracked separately per job). It includes:
              </p>
              <table className="w-full">
                <thead>
                  <tr className="text-left text-surface-500">
                    <th className="pb-1 font-medium">Component</th>
                    <th className="pb-1 font-medium text-right">A1 (~$400)</th>
                    <th className="pb-1 font-medium text-right">P1S (~$700)</th>
                  </tr>
                </thead>
                <tbody className="text-surface-300">
                  <tr>
                    <td className="py-0.5">Electricity</td>
                    <td className="py-0.5 text-right">$0.02</td>
                    <td className="py-0.5 text-right">$0.04</td>
                  </tr>
                  <tr>
                    <td className="py-0.5">Depreciation</td>
                    <td className="py-0.5 text-right">$0.03</td>
                    <td className="py-0.5 text-right">$0.05</td>
                  </tr>
                  <tr>
                    <td className="py-0.5">Maintenance</td>
                    <td className="py-0.5 text-right">$0.15</td>
                    <td className="py-0.5 text-right">$0.20</td>
                  </tr>
                  <tr>
                    <td className="py-0.5">Utilization buffer</td>
                    <td className="py-0.5 text-right">$0.25</td>
                    <td className="py-0.5 text-right">$0.35</td>
                  </tr>
                  <tr className="border-t border-surface-700 font-medium text-surface-200">
                    <td className="pt-1">Total</td>
                    <td className="pt-1 text-right">$0.50/hr</td>
                    <td className="pt-1 text-right">$0.75/hr</td>
                  </tr>
                </tbody>
              </table>
              <p className="text-surface-500">
                Depreciation assumes ~13,000 hr lifespan (3 yr @ 12 hr/day). Maintenance covers nozzles, hotends, fans, belts, and failed prints. Utilization buffer accounts for downtime, calibration, and demand spikes.
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

function PrinterUtilizationCard({ utilization }: { utilization: PrinterUtilization[] }) {
  const [selectedPeriod, setSelectedPeriod] = useState<string>('7d')
  const data = utilization.find(u => u.period === selectedPeriod) || utilization[0]

  if (!data) {
    return (
      <div className="card p-5">
        <h3 className="text-sm font-semibold text-surface-400 uppercase tracking-wider mb-4 flex items-center gap-2">
          <Activity className="h-4 w-4" />
          Utilization
        </h3>
        <p className="text-surface-500 text-sm">No utilization data available</p>
      </div>
    )
  }

  const printingPercent = data.total_hours > 0 ? (data.printing_hours / data.total_hours) * 100 : 0
  const failedPercent = data.total_hours > 0 ? (data.failed_hours / data.total_hours) * 100 : 0
  const idlePercent = 100 - printingPercent - failedPercent

  return (
    <div className="card p-5">
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-sm font-semibold text-surface-400 uppercase tracking-wider flex items-center gap-2">
          <Activity className="h-4 w-4" />
          Utilization
        </h3>
        <div className="flex gap-1">
          {['7d', '30d', '90d'].map(period => (
            <button
              key={period}
              onClick={() => setSelectedPeriod(period)}
              className={cn(
                'px-2 py-0.5 text-xs rounded transition-colors',
                selectedPeriod === period
                  ? 'bg-accent-500 text-white'
                  : 'bg-surface-700 text-surface-400 hover:bg-surface-600'
              )}
            >
              {period}
            </button>
          ))}
        </div>
      </div>

      {/* Donut-style breakdown bar */}
      <div className="mb-4">
        <div className="h-4 flex rounded-full overflow-hidden">
          <div
            className="bg-emerald-500 transition-all"
            style={{ width: `${printingPercent}%` }}
            title={`Printing: ${data.printing_hours.toFixed(1)}h`}
          />
          <div
            className="bg-red-500 transition-all"
            style={{ width: `${failedPercent}%` }}
            title={`Failed: ${data.failed_hours.toFixed(1)}h`}
          />
          <div
            className="bg-surface-700 transition-all"
            style={{ width: `${idlePercent}%` }}
            title={`Idle: ${data.idle_hours.toFixed(1)}h`}
          />
        </div>
        <div className="flex justify-between mt-2 text-xs">
          <div className="flex items-center gap-1">
            <span className="w-2 h-2 rounded-full bg-emerald-500" />
            <span className="text-surface-400">Printing</span>
          </div>
          <div className="flex items-center gap-1">
            <span className="w-2 h-2 rounded-full bg-red-500" />
            <span className="text-surface-400">Failed</span>
          </div>
          <div className="flex items-center gap-1">
            <span className="w-2 h-2 rounded-full bg-surface-700" />
            <span className="text-surface-400">Idle</span>
          </div>
        </div>
      </div>

      {/* Stats */}
      <div className="space-y-2 text-sm">
        <div className="flex justify-between">
          <span className="text-surface-400">Utilization</span>
          <span className={cn(
            'font-medium',
            data.utilization_percent >= 50 ? 'text-emerald-400' :
            data.utilization_percent >= 25 ? 'text-amber-400' :
            'text-surface-300'
          )}>
            {data.utilization_percent.toFixed(1)}%
          </span>
        </div>
        <div className="flex justify-between">
          <span className="text-surface-400">Printing Hours</span>
          <span className="text-surface-200">{data.printing_hours.toFixed(1)}h</span>
        </div>
        <div className="flex justify-between">
          <span className="text-surface-400">Failed Hours</span>
          <span className="text-red-400">{data.failed_hours.toFixed(1)}h</span>
        </div>
        <div className="flex justify-between">
          <span className="text-surface-400">Idle Hours</span>
          <span className="text-surface-500">{data.idle_hours.toFixed(1)}h</span>
        </div>
        <div className="border-t border-surface-700 pt-2 mt-2">
          <div className="flex justify-between">
            <span className="text-surface-400">Configured Rate</span>
            <span className="text-surface-200">${(data.configured_cost_per_hour_cents / 100).toFixed(2)}/hr</span>
          </div>
          <div className="flex justify-between">
            <span className="text-surface-400">Actual Revenue/hr</span>
            <span className={cn(
              data.actual_revenue_per_hour_cents > data.configured_cost_per_hour_cents
                ? 'text-emerald-400'
                : 'text-surface-300'
            )}>
              ${(data.actual_revenue_per_hour_cents / 100).toFixed(2)}/hr
            </span>
          </div>
        </div>
      </div>
    </div>
  )
}

function PrinterROICard({ roi, printerId, purchasePriceCents }: { roi: PrinterROI; printerId: string; purchasePriceCents: number }) {
  const [editing, setEditing] = useState(false)
  const [value, setValue] = useState((purchasePriceCents / 100).toFixed(2))
  const updatePrinter = useUpdatePrinter()

  const handleSave = async () => {
    const cents = Math.round(parseFloat(value || '0') * 100)
    await updatePrinter.mutateAsync({ id: printerId, data: { purchase_price_cents: cents } })
    setEditing(false)
  }

  const handleCancel = () => {
    setValue((purchasePriceCents / 100).toFixed(2))
    setEditing(false)
  }

  // Break-even progress (capped at 100%)
  const breakEvenProgress = roi.purchase_price_cents > 0
    ? Math.min((roi.lifetime_profit_cents + roi.purchase_price_cents) / roi.purchase_price_cents * 100, 100)
    : 0

  return (
    <div className="card p-5">
      <h3 className="text-sm font-semibold text-surface-400 uppercase tracking-wider mb-4 flex items-center gap-2">
        <Target className="h-4 w-4" />
        ROI & Break-Even
      </h3>

      {purchasePriceCents === 0 ? (
        <div className="p-3 rounded-lg bg-amber-500/10 border border-amber-500/20">
          <div className="text-sm text-amber-300 font-medium mb-2">Set Purchase Price</div>
          <p className="text-xs text-surface-400 mb-3">
            Enter the purchase price to track ROI and break-even.
          </p>
          <div className="flex items-center gap-2">
            <span className="text-surface-400">$</span>
            <input
              type="number"
              step="0.01"
              min="0"
              value={value}
              onChange={(e) => setValue(e.target.value)}
              className="input w-28 text-sm"
              placeholder="0.00"
              onKeyDown={(e) => {
                if (e.key === 'Enter') handleSave()
                if (e.key === 'Escape') handleCancel()
              }}
            />
            <button
              onClick={handleSave}
              disabled={updatePrinter.isPending}
              className="btn btn-primary text-xs py-1 px-3"
            >
              Save
            </button>
          </div>
        </div>
      ) : (
        <>
          {/* Break-even progress bar */}
          <div className="mb-4">
            <div className="flex items-center justify-between text-sm mb-1">
              <span className="text-surface-400">Break-even Progress</span>
              <span className={cn(
                'font-medium',
                roi.break_even_reached ? 'text-emerald-400' : 'text-surface-200'
              )}>
                {roi.break_even_reached ? 'Reached!' : `${breakEvenProgress.toFixed(0)}%`}
              </span>
            </div>
            <div className="h-3 bg-surface-800 rounded-full overflow-hidden">
              <div
                className={cn(
                  'h-full transition-all',
                  roi.break_even_reached ? 'bg-emerald-500' : 'bg-accent-500'
                )}
                style={{ width: `${breakEvenProgress}%` }}
              />
            </div>
            {!roi.break_even_reached && roi.hours_to_break_even > 0 && (
              <p className="text-xs text-surface-500 mt-1">
                ~{roi.hours_to_break_even.toFixed(0)}h remaining to break even
              </p>
            )}
          </div>

          {/* Stats grid */}
          <div className="space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="text-surface-400">Purchase Price</span>
              <div className="flex items-center gap-2">
                {editing ? (
                  <>
                    <span className="text-surface-400">$</span>
                    <input
                      type="number"
                      step="0.01"
                      min="0"
                      value={value}
                      onChange={(e) => setValue(e.target.value)}
                      className="input w-20 text-xs py-0.5"
                      autoFocus
                      onKeyDown={(e) => {
                        if (e.key === 'Enter') handleSave()
                        if (e.key === 'Escape') handleCancel()
                      }}
                    />
                    <button onClick={handleSave} className="p-0.5 rounded text-emerald-400 hover:bg-emerald-500/10">
                      <Check className="h-3 w-3" />
                    </button>
                    <button onClick={handleCancel} className="p-0.5 rounded text-surface-400 hover:bg-surface-700">
                      <X className="h-3 w-3" />
                    </button>
                  </>
                ) : (
                  <>
                    <span className="text-surface-200">${(purchasePriceCents / 100).toFixed(2)}</span>
                    <button
                      onClick={() => setEditing(true)}
                      className="text-xs text-accent-400 hover:text-accent-300"
                    >
                      Edit
                    </button>
                  </>
                )}
              </div>
            </div>
            <div className="flex justify-between">
              <span className="text-surface-400">Lifetime Profit</span>
              <span className={cn(
                'font-medium',
                roi.lifetime_profit_cents >= 0 ? 'text-emerald-400' : 'text-red-400'
              )}>
                {roi.lifetime_profit_cents >= 0 ? '' : '-'}${Math.abs(roi.lifetime_profit_cents / 100).toFixed(2)}
              </span>
            </div>
            <div className="flex justify-between">
              <span className="text-surface-400">Revenue/Hour</span>
              <span className="text-surface-200">${(roi.revenue_per_hour_cents / 100).toFixed(2)}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-surface-400">Net/Hour</span>
              <span className={cn(
                roi.net_per_hour_cents >= 0 ? 'text-emerald-400' : 'text-red-400'
              )}>
                ${(roi.net_per_hour_cents / 100).toFixed(2)}
              </span>
            </div>
            <div className="border-t border-surface-700 pt-2 mt-2">
              <div className="flex justify-between">
                <span className="text-surface-400">Total Print Hours</span>
                <span className="text-surface-200">{roi.total_printing_hours.toFixed(1)}h</span>
              </div>
              <div className="flex justify-between">
                <span className="text-surface-400">Printer Age</span>
                <span className="text-surface-500">{(roi.printer_age_hours / 24).toFixed(0)} days</span>
              </div>
            </div>
          </div>
        </>
      )}
    </div>
  )
}

function PrinterHealthCard({ health }: { health: PrinterHealth }) {
  const failureCategories = Object.entries(health.failure_breakdown || {})
    .sort((a, b) => b[1] - a[1])
    .slice(0, 5)
  const maxFailures = failureCategories.length > 0 ? Math.max(...failureCategories.map(f => f[1])) : 0

  const categoryLabels: Record<string, string> = {
    mechanical: 'Mechanical',
    filament: 'Filament',
    adhesion: 'Adhesion',
    thermal: 'Thermal',
    network: 'Network',
    user_cancelled: 'Cancelled',
    unknown: 'Unknown'
  }

  return (
    <div className="card p-5">
      <h3 className="text-sm font-semibold text-surface-400 uppercase tracking-wider mb-4 flex items-center gap-2">
        <Heart className="h-4 w-4" />
        Health
      </h3>

      {/* Stats grid */}
      <div className="grid grid-cols-2 gap-3 mb-4">
        <div className="p-2 rounded bg-surface-800/50">
          <div className="text-xs text-surface-500">Failure Rate</div>
          <div className={cn(
            'text-lg font-semibold',
            health.failure_rate < 10 ? 'text-emerald-400' :
            health.failure_rate < 25 ? 'text-amber-400' :
            'text-red-400'
          )}>
            {health.failure_rate.toFixed(1)}%
          </div>
        </div>
        <div className="p-2 rounded bg-surface-800/50">
          <div className="text-xs text-surface-500">Avg Duration</div>
          <div className="text-lg font-semibold text-surface-200">
            {formatDuration(health.avg_job_duration_sec)}
          </div>
        </div>
        <div className="p-2 rounded bg-surface-800/50">
          <div className="text-xs text-surface-500">Avg Cost/Job</div>
          <div className="text-lg font-semibold text-surface-200">
            ${(health.avg_cost_cents / 100).toFixed(2)}
          </div>
        </div>
        <div className="p-2 rounded bg-surface-800/50">
          <div className="text-xs text-surface-500">Revenue</div>
          <div className="text-lg font-semibold text-emerald-400">
            ${(health.total_revenue_cents / 100).toFixed(2)}
          </div>
        </div>
      </div>

      {/* Failure breakdown */}
      {failureCategories.length > 0 ? (
        <div>
          <div className="text-xs text-surface-500 mb-2">Failure Breakdown</div>
          <div className="space-y-1.5">
            {failureCategories.map(([category, count]) => (
              <div key={category} className="flex items-center gap-2">
                <span className="text-xs text-surface-400 w-20 truncate">
                  {categoryLabels[category] || category}
                </span>
                <div className="flex-1 h-2 bg-surface-800 rounded-full overflow-hidden">
                  <div
                    className="h-full bg-red-500/70 transition-all"
                    style={{ width: `${maxFailures > 0 ? (count / maxFailures) * 100 : 0}%` }}
                  />
                </div>
                <span className="text-xs text-surface-500 w-6 text-right">{count}</span>
              </div>
            ))}
          </div>
        </div>
      ) : (
        <div className="flex items-center gap-2 text-surface-400 text-sm">
          <span className="text-emerald-400">&#10003;</span> No failures recorded
        </div>
      )}

      {/* Job counts */}
      <div className="mt-3 pt-3 border-t border-surface-700 flex justify-between text-xs text-surface-500">
        <span>{health.completed_jobs} completed</span>
        <span>{health.failed_jobs} failed</span>
        <span>{health.total_jobs} total</span>
      </div>
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
