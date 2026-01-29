import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Plus, Trash2, GripVertical, Palette } from 'lucide-react'
import { templatesApi } from '../api/client'
import type { RecipeMaterial, ColorSpec, MaterialType } from '../types'
import { cn } from '../lib/utils'

const materialTypes: { label: string; value: MaterialType }[] = [
  { label: 'PLA', value: 'pla' },
  { label: 'PETG', value: 'petg' },
  { label: 'ABS', value: 'abs' },
  { label: 'ASA', value: 'asa' },
  { label: 'TPU', value: 'tpu' },
]

const colorModes = [
  { label: 'Exact Match', value: 'exact' },
  { label: 'Category', value: 'category' },
  { label: 'Any Color', value: 'any' },
]

interface RecipeMaterialEditorProps {
  templateId: string
  materials: RecipeMaterial[]
}

export default function RecipeMaterialEditor({ templateId, materials }: RecipeMaterialEditorProps) {
  const queryClient = useQueryClient()
  const [showAddForm, setShowAddForm] = useState(false)
  const [newMaterial, setNewMaterial] = useState<{
    material_type: MaterialType
    weight_grams: number
    ams_position?: number
    color_spec?: ColorSpec
    notes: string
  }>({
    material_type: 'pla',
    weight_grams: 0,
    notes: '',
  })

  const addMaterial = useMutation({
    mutationFn: () =>
      templatesApi.addMaterial(templateId, {
        material_type: newMaterial.material_type,
        weight_grams: newMaterial.weight_grams,
        ams_position: newMaterial.ams_position,
        color_spec: newMaterial.color_spec,
        sequence_order: materials.length,
        notes: newMaterial.notes,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['templates', templateId] })
      setShowAddForm(false)
      setNewMaterial({ material_type: 'pla', weight_grams: 0, notes: '' })
    },
  })

  const removeMaterial = useMutation({
    mutationFn: (materialId: string) => templatesApi.removeMaterial(templateId, materialId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['templates', templateId] })
    },
  })

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

  return (
    <div className="card p-6">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold text-surface-100">
          Material Requirements ({materials.length})
        </h2>
        <button
          onClick={() => setShowAddForm(true)}
          className="btn btn-secondary"
        >
          <Plus className="h-4 w-4 mr-2" />
          Add Material
        </button>
      </div>

      {materials.length === 0 ? (
        <div className="text-center py-8 text-surface-500">
          <Palette className="h-12 w-12 mx-auto mb-3 opacity-50" />
          <p>No material requirements defined</p>
          <p className="text-sm mt-1">
            Add materials to specify what's needed to print this recipe
          </p>
        </div>
      ) : (
        <div className="space-y-2">
          {materials.map((m, idx) => (
            <div
              key={m.id}
              className="flex items-center justify-between p-4 rounded-lg bg-surface-800/50"
            >
              <div className="flex items-center gap-4">
                <GripVertical className="h-4 w-4 text-surface-600 cursor-move" />
                <div className="flex items-center gap-2">
                  <span className="text-surface-500 text-sm">#{idx + 1}</span>
                  <span className={cn('badge', getMaterialBadge(m.material_type))}>
                    {m.material_type.toUpperCase()}
                  </span>
                </div>
                <div>
                  <div className="font-medium text-surface-100">
                    {m.weight_grams}g
                    {m.ams_position && (
                      <span className="ml-2 text-sm text-surface-400">
                        AMS Slot {m.ams_position}
                      </span>
                    )}
                  </div>
                  <div className="text-sm text-surface-500">
                    {m.color_spec ? (
                      <>
                        Color: {m.color_spec.mode === 'any' ? 'Any' : m.color_spec.name || m.color_spec.hex || 'Specified'}
                        {m.color_spec.hex && (
                          <span
                            className="inline-block w-3 h-3 rounded-full ml-1 align-middle"
                            style={{ backgroundColor: m.color_spec.hex }}
                          />
                        )}
                      </>
                    ) : (
                      'No color preference'
                    )}
                    {m.notes && <> - {m.notes}</>}
                  </div>
                </div>
              </div>
              <button
                onClick={() => removeMaterial.mutate(m.id)}
                disabled={removeMaterial.isPending}
                className="btn btn-ghost text-red-400 hover:text-red-300 hover:bg-red-500/10"
              >
                <Trash2 className="h-4 w-4" />
              </button>
            </div>
          ))}
        </div>
      )}

      {/* Add Material Form */}
      {showAddForm && (
        <div className="mt-4 p-4 rounded-lg bg-surface-800 border border-surface-700">
          <h3 className="text-sm font-medium text-surface-200 mb-4">Add Material Requirement</h3>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-surface-300 mb-1">
                Material Type
              </label>
              <select
                value={newMaterial.material_type}
                onChange={(e) =>
                  setNewMaterial({ ...newMaterial, material_type: e.target.value as MaterialType })
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
                Weight (grams)
              </label>
              <input
                type="number"
                value={newMaterial.weight_grams}
                onChange={(e) =>
                  setNewMaterial({ ...newMaterial, weight_grams: parseFloat(e.target.value) || 0 })
                }
                className="input"
                min="0"
                step="0.1"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-surface-300 mb-1">
                AMS Position (optional)
              </label>
              <select
                value={newMaterial.ams_position || ''}
                onChange={(e) =>
                  setNewMaterial({
                    ...newMaterial,
                    ams_position: e.target.value ? parseInt(e.target.value) : undefined,
                  })
                }
                className="input"
              >
                <option value="">No AMS</option>
                <option value="1">Slot 1</option>
                <option value="2">Slot 2</option>
                <option value="3">Slot 3</option>
                <option value="4">Slot 4</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-surface-300 mb-1">
                Color Matching
              </label>
              <select
                value={newMaterial.color_spec?.mode || 'any'}
                onChange={(e) =>
                  setNewMaterial({
                    ...newMaterial,
                    color_spec: { mode: e.target.value as 'exact' | 'category' | 'any' },
                  })
                }
                className="input"
              >
                {colorModes.map((m) => (
                  <option key={m.value} value={m.value}>
                    {m.label}
                  </option>
                ))}
              </select>
            </div>
            {newMaterial.color_spec?.mode === 'exact' && (
              <>
                <div>
                  <label className="block text-sm font-medium text-surface-300 mb-1">
                    Color Name
                  </label>
                  <input
                    type="text"
                    value={newMaterial.color_spec?.name || ''}
                    onChange={(e) =>
                      setNewMaterial({
                        ...newMaterial,
                        color_spec: { ...newMaterial.color_spec!, name: e.target.value },
                      })
                    }
                    className="input"
                    placeholder="e.g., White, Black, Red"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium text-surface-300 mb-1">
                    Color Hex (optional)
                  </label>
                  <input
                    type="color"
                    value={newMaterial.color_spec?.hex || '#ffffff'}
                    onChange={(e) =>
                      setNewMaterial({
                        ...newMaterial,
                        color_spec: { ...newMaterial.color_spec!, hex: e.target.value },
                      })
                    }
                    className="input h-10"
                  />
                </div>
              </>
            )}
            <div className="col-span-2">
              <label className="block text-sm font-medium text-surface-300 mb-1">
                Notes (optional)
              </label>
              <input
                type="text"
                value={newMaterial.notes}
                onChange={(e) => setNewMaterial({ ...newMaterial, notes: e.target.value })}
                className="input"
                placeholder="e.g., Main body color, Support material"
              />
            </div>
          </div>
          <div className="flex justify-end gap-3 mt-4">
            <button
              onClick={() => setShowAddForm(false)}
              className="btn btn-ghost"
            >
              Cancel
            </button>
            <button
              onClick={() => addMaterial.mutate()}
              disabled={addMaterial.isPending || newMaterial.weight_grams <= 0}
              className="btn btn-primary"
            >
              {addMaterial.isPending ? 'Adding...' : 'Add Material'}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
