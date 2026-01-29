import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Plus, Trash2, ShoppingCart } from 'lucide-react'
import { templatesApi, materialsApi } from '../api/client'
import type { RecipeSupply } from '../types'
import { cn } from '../lib/utils'

interface RecipeSupplyEditorProps {
  templateId: string
  supplies: RecipeSupply[]
}

function formatCents(cents: number): string {
  return (cents / 100).toLocaleString('en-US', {
    style: 'currency',
    currency: 'USD',
  })
}

export default function RecipeSupplyEditor({ templateId, supplies }: RecipeSupplyEditorProps) {
  const queryClient = useQueryClient()
  const [showAddForm, setShowAddForm] = useState(false)
  const [addMode, setAddMode] = useState<'catalog' | 'manual'>('catalog')

  // Catalog mode state
  const [selectedMaterialId, setSelectedMaterialId] = useState('')
  const [catalogQuantity, setCatalogQuantity] = useState('1')

  // Manual mode state
  const [newSupply, setNewSupply] = useState<{
    name: string
    unit_cost_cents: number
    quantity: number
    notes: string
  }>({
    name: '',
    unit_cost_cents: 0,
    quantity: 1,
    notes: '',
  })

  // Fetch supply materials from catalog
  const { data: supplyMaterials = [] } = useQuery({
    queryKey: ['materials', 'supply'],
    queryFn: () => materialsApi.listByType('supply'),
  })

  const addSupply = useMutation({
    mutationFn: (data: {
      name: string
      unit_cost_cents: number
      quantity: number
      notes?: string
      material_id?: string
    }) => templatesApi.addSupply(templateId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['templates', templateId] })
      queryClient.invalidateQueries({ queryKey: ['templates', templateId, 'cost-estimate'] })
      setShowAddForm(false)
      resetForm()
    },
  })

  const removeSupply = useMutation({
    mutationFn: (supplyId: string) => templatesApi.removeSupply(templateId, supplyId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['templates', templateId] })
      queryClient.invalidateQueries({ queryKey: ['templates', templateId, 'cost-estimate'] })
    },
  })

  const resetForm = () => {
    setSelectedMaterialId('')
    setCatalogQuantity('1')
    setNewSupply({ name: '', unit_cost_cents: 0, quantity: 1, notes: '' })
  }

  const handleCatalogAdd = () => {
    if (!selectedMaterialId) return
    addSupply.mutate({
      name: '', // auto-populated from material by backend
      unit_cost_cents: 0, // auto-populated from material by backend
      quantity: parseInt(catalogQuantity) || 1,
      material_id: selectedMaterialId,
    })
  }

  const handleManualAdd = () => {
    if (!newSupply.name.trim()) return
    addSupply.mutate({
      name: newSupply.name.trim(),
      unit_cost_cents: newSupply.unit_cost_cents,
      quantity: newSupply.quantity,
      notes: newSupply.notes,
    })
  }

  const totalCost = supplies.reduce((sum, s) => sum + s.unit_cost_cents * s.quantity, 0)

  // Auto-select catalog mode if we have supply materials, otherwise manual
  const effectiveAddMode = supplyMaterials.length > 0 ? addMode : 'manual'

  return (
    <div className="card p-6">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <ShoppingCart className="h-5 w-5 text-surface-400" />
          <h2 className="text-lg font-semibold text-surface-100">
            Supplies ({supplies.length})
          </h2>
          {totalCost > 0 && (
            <span className="text-sm text-surface-500">
              • {formatCents(totalCost)} total
            </span>
          )}
        </div>
        <button
          onClick={() => setShowAddForm(true)}
          className="btn btn-secondary"
        >
          <Plus className="h-4 w-4 mr-2" />
          Add Supply
        </button>
      </div>

      {supplies.length === 0 ? (
        <div className="text-center py-8 text-surface-500">
          <ShoppingCart className="h-12 w-12 mx-auto mb-3 opacity-50" />
          <p>No supply items defined</p>
          <p className="text-sm mt-1">
            Add non-printed items like magnets, packaging, or hardware
          </p>
        </div>
      ) : (
        <div className="space-y-2">
          {supplies.map((s) => (
            <div
              key={s.id}
              className="flex items-center justify-between p-4 rounded-lg bg-surface-800/50"
            >
              <div className="flex items-center gap-4">
                <div>
                  <div className="font-medium text-surface-100">
                    {s.name}
                    <span className="ml-2 text-sm text-surface-400">
                      ×{s.quantity}
                    </span>
                  </div>
                  <div className="text-sm text-surface-500">
                    {formatCents(s.unit_cost_cents)} each
                    {s.quantity > 1 && (
                      <span className="ml-2">
                        = {formatCents(s.unit_cost_cents * s.quantity)}
                      </span>
                    )}
                    {s.notes && <span className="ml-2">• {s.notes}</span>}
                  </div>
                </div>
              </div>
              <button
                onClick={() => removeSupply.mutate(s.id)}
                disabled={removeSupply.isPending}
                className="btn btn-ghost text-red-400 hover:text-red-300 hover:bg-red-500/10"
              >
                <Trash2 className="h-4 w-4" />
              </button>
            </div>
          ))}
        </div>
      )}

      {/* Add Supply Form */}
      {showAddForm && (
        <div className="mt-4 p-4 rounded-lg bg-surface-800 border border-surface-700">
          <h3 className="text-sm font-medium text-surface-200 mb-4">Add Supply Item</h3>

          {/* Mode Toggle */}
          {supplyMaterials.length > 0 && (
            <div className="flex gap-2 mb-4">
              <button
                type="button"
                onClick={() => setAddMode('catalog')}
                className={cn(
                  'text-xs px-3 py-1 rounded-full transition-colors',
                  addMode === 'catalog'
                    ? 'bg-accent-500/20 text-accent-400 border border-accent-500'
                    : 'bg-surface-700 text-surface-400 border border-surface-600 hover:text-surface-200'
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
                    : 'bg-surface-700 text-surface-400 border border-surface-600 hover:text-surface-200'
                )}
              >
                Manual Entry
              </button>
            </div>
          )}

          {effectiveAddMode === 'catalog' && supplyMaterials.length > 0 ? (
            /* Catalog Mode */
            <div className="grid grid-cols-[1fr_auto] gap-4 items-end">
              <div>
                <label className="block text-sm font-medium text-surface-300 mb-1">
                  Supply Material
                </label>
                <select
                  value={selectedMaterialId}
                  onChange={(e) => setSelectedMaterialId(e.target.value)}
                  className="input"
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
                <label className="block text-sm font-medium text-surface-300 mb-1">
                  Quantity
                </label>
                <input
                  type="number"
                  value={catalogQuantity}
                  onChange={(e) => setCatalogQuantity(e.target.value)}
                  className="input w-24"
                  min="1"
                />
              </div>
            </div>
          ) : (
            /* Manual Mode */
            <div className="grid grid-cols-2 gap-4">
              <div className="col-span-2">
                <label className="block text-sm font-medium text-surface-300 mb-1">
                  Name
                </label>
                <input
                  type="text"
                  value={newSupply.name}
                  onChange={(e) => setNewSupply({ ...newSupply, name: e.target.value })}
                  className="input"
                  placeholder="e.g., Magnets, Box, Labels"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-surface-300 mb-1">
                  Unit Cost ($)
                </label>
                <input
                  type="number"
                  value={newSupply.unit_cost_cents / 100}
                  onChange={(e) =>
                    setNewSupply({
                      ...newSupply,
                      unit_cost_cents: Math.round(parseFloat(e.target.value) * 100) || 0,
                    })
                  }
                  className="input"
                  min="0"
                  step="0.01"
                  placeholder="0.00"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-surface-300 mb-1">
                  Quantity
                </label>
                <input
                  type="number"
                  value={newSupply.quantity}
                  onChange={(e) =>
                    setNewSupply({ ...newSupply, quantity: parseInt(e.target.value) || 1 })
                  }
                  className="input"
                  min="1"
                />
              </div>
              <div className="col-span-2">
                <label className="block text-sm font-medium text-surface-300 mb-1">
                  Notes (optional)
                </label>
                <input
                  type="text"
                  value={newSupply.notes}
                  onChange={(e) => setNewSupply({ ...newSupply, notes: e.target.value })}
                  className="input"
                  placeholder="e.g., 10mm × 2mm N52 neodymium"
                />
              </div>
            </div>
          )}

          <div className="flex justify-end gap-3 mt-4">
            <button
              onClick={() => {
                setShowAddForm(false)
                resetForm()
              }}
              className="btn btn-ghost"
            >
              Cancel
            </button>
            <button
              onClick={effectiveAddMode === 'catalog' ? handleCatalogAdd : handleManualAdd}
              disabled={
                addSupply.isPending ||
                (effectiveAddMode === 'catalog' ? !selectedMaterialId : !newSupply.name.trim())
              }
              className="btn btn-primary"
            >
              {addSupply.isPending ? 'Adding...' : 'Add Supply'}
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
