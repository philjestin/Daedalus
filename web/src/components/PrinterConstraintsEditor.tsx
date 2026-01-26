import { useState, useEffect } from 'react'
import { Settings2, Box, CircleDot, Thermometer, Layers } from 'lucide-react'
import type { PrinterConstraints, BuildVolume } from '../types'

interface PrinterConstraintsEditorProps {
  constraints?: PrinterConstraints
  onChange: (constraints: PrinterConstraints) => void
  disabled?: boolean
}

const commonNozzleSizes = [0.2, 0.4, 0.6, 0.8]

export default function PrinterConstraintsEditor({
  constraints,
  onChange,
  disabled = false,
}: PrinterConstraintsEditorProps) {
  const [localConstraints, setLocalConstraints] = useState<PrinterConstraints>(
    constraints || {
      requires_enclosure: false,
      requires_ams: false,
    }
  )

  useEffect(() => {
    if (constraints) {
      setLocalConstraints(constraints)
    }
  }, [constraints])

  const updateConstraints = (updates: Partial<PrinterConstraints>) => {
    const updated = { ...localConstraints, ...updates }
    setLocalConstraints(updated)
    onChange(updated)
  }

  const updateBedSize = (axis: keyof BuildVolume, value: number) => {
    const newBedSize: BuildVolume = {
      x: localConstraints.min_bed_size?.x || 0,
      y: localConstraints.min_bed_size?.y || 0,
      z: localConstraints.min_bed_size?.z || 0,
      [axis]: value,
    }
    // Only set if at least one dimension is non-zero
    if (newBedSize.x > 0 || newBedSize.y > 0 || newBedSize.z > 0) {
      updateConstraints({ min_bed_size: newBedSize })
    } else {
      updateConstraints({ min_bed_size: undefined })
    }
  }

  const toggleNozzle = (size: number) => {
    const current = localConstraints.nozzle_diameters || []
    let updated: number[]
    if (current.includes(size)) {
      updated = current.filter((n) => n !== size)
    } else {
      updated = [...current, size].sort()
    }
    updateConstraints({ nozzle_diameters: updated.length > 0 ? updated : undefined })
  }

  return (
    <div className="card p-6">
      <div className="flex items-center gap-2 mb-4">
        <Settings2 className="h-5 w-5 text-surface-400" />
        <h2 className="text-lg font-semibold text-surface-100">Printer Constraints</h2>
      </div>

      <div className="space-y-6">
        {/* Minimum Bed Size */}
        <div>
          <div className="flex items-center gap-2 mb-3">
            <Box className="h-4 w-4 text-surface-400" />
            <label className="text-sm font-medium text-surface-300">
              Minimum Build Volume (mm)
            </label>
          </div>
          <div className="grid grid-cols-3 gap-3">
            <div>
              <label className="block text-xs text-surface-500 mb-1">X (Width)</label>
              <input
                type="number"
                value={localConstraints.min_bed_size?.x || ''}
                onChange={(e) => updateBedSize('x', parseFloat(e.target.value) || 0)}
                className="input"
                placeholder="0"
                min="0"
                disabled={disabled}
              />
            </div>
            <div>
              <label className="block text-xs text-surface-500 mb-1">Y (Depth)</label>
              <input
                type="number"
                value={localConstraints.min_bed_size?.y || ''}
                onChange={(e) => updateBedSize('y', parseFloat(e.target.value) || 0)}
                className="input"
                placeholder="0"
                min="0"
                disabled={disabled}
              />
            </div>
            <div>
              <label className="block text-xs text-surface-500 mb-1">Z (Height)</label>
              <input
                type="number"
                value={localConstraints.min_bed_size?.z || ''}
                onChange={(e) => updateBedSize('z', parseFloat(e.target.value) || 0)}
                className="input"
                placeholder="0"
                min="0"
                disabled={disabled}
              />
            </div>
          </div>
        </div>

        {/* Nozzle Diameter */}
        <div>
          <div className="flex items-center gap-2 mb-3">
            <CircleDot className="h-4 w-4 text-surface-400" />
            <label className="text-sm font-medium text-surface-300">
              Compatible Nozzle Sizes
            </label>
          </div>
          <div className="flex flex-wrap gap-2">
            {commonNozzleSizes.map((size) => (
              <button
                key={size}
                onClick={() => toggleNozzle(size)}
                disabled={disabled}
                className={`px-3 py-1.5 rounded-lg text-sm font-medium transition-colors ${
                  localConstraints.nozzle_diameters?.includes(size)
                    ? 'bg-accent-500/20 text-accent-400 border border-accent-500/30'
                    : 'bg-surface-800 text-surface-400 border border-surface-700 hover:bg-surface-700'
                }`}
              >
                {size}mm
              </button>
            ))}
          </div>
          <p className="text-xs text-surface-500 mt-2">
            Leave all unselected to allow any nozzle size
          </p>
        </div>

        {/* Feature Requirements */}
        <div>
          <div className="flex items-center gap-2 mb-3">
            <Layers className="h-4 w-4 text-surface-400" />
            <label className="text-sm font-medium text-surface-300">
              Required Features
            </label>
          </div>
          <div className="space-y-3">
            <label className="flex items-center gap-3 cursor-pointer">
              <input
                type="checkbox"
                checked={localConstraints.requires_enclosure}
                onChange={(e) => updateConstraints({ requires_enclosure: e.target.checked })}
                className="w-4 h-4 rounded border-surface-600"
                disabled={disabled}
              />
              <div className="flex items-center gap-2">
                <Thermometer className="h-4 w-4 text-surface-400" />
                <span className="text-surface-200">Requires Enclosure</span>
              </div>
              <span className="text-xs text-surface-500 ml-auto">
                For ABS, ASA, or temperature-sensitive prints
              </span>
            </label>
            <label className="flex items-center gap-3 cursor-pointer">
              <input
                type="checkbox"
                checked={localConstraints.requires_ams}
                onChange={(e) => updateConstraints({ requires_ams: e.target.checked })}
                className="w-4 h-4 rounded border-surface-600"
                disabled={disabled}
              />
              <div className="flex items-center gap-2">
                <Layers className="h-4 w-4 text-surface-400" />
                <span className="text-surface-200">Requires AMS</span>
              </div>
              <span className="text-xs text-surface-500 ml-auto">
                For multi-color or multi-material prints
              </span>
            </label>
          </div>
        </div>

        {/* Printer Tags */}
        <div>
          <div className="flex items-center gap-2 mb-3">
            <label className="text-sm font-medium text-surface-300">
              Printer Tags (optional)
            </label>
          </div>
          <input
            type="text"
            value={localConstraints.printer_tags?.join(', ') || ''}
            onChange={(e) =>
              updateConstraints({
                printer_tags: e.target.value
                  ? e.target.value.split(',').map((s) => s.trim()).filter(Boolean)
                  : undefined,
              })
            }
            className="input"
            placeholder="e.g., production, workshop-a, enclosed"
            disabled={disabled}
          />
          <p className="text-xs text-surface-500 mt-2">
            Comma-separated tags to filter compatible printers
          </p>
        </div>
      </div>
    </div>
  )
}
