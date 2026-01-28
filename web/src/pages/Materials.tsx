import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Plus, Package, Droplet, ShoppingCart, Trash2 } from 'lucide-react'
import { materialsApi, spoolsApi } from '../api/client'
import { cn, getStatusBadge } from '../lib/utils'
import type { Material, MaterialSpool, MaterialType } from '../types'

export default function Materials() {
  const queryClient = useQueryClient()

  const { data: materials = [], isLoading: materialsLoading } = useQuery({
    queryKey: ['materials'],
    queryFn: () => materialsApi.list(),
    refetchInterval: 5000,
  })

  const { data: spools = [], isLoading: spoolsLoading } = useQuery({
    queryKey: ['spools'],
    queryFn: () => spoolsApi.list(),
  })

  const filamentMaterials = materials.filter(m => m.type !== 'supply')
  const supplyMaterials = materials.filter(m => m.type === 'supply')

  const createMaterial = useMutation({
    mutationFn: (data: Partial<Material>) => materialsApi.create(data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['materials'] }),
  })

  const createSpool = useMutation({
    mutationFn: (data: Partial<MaterialSpool>) => spoolsApi.create(data),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['spools'] }),
  })

  const deleteMaterial = useMutation({
    mutationFn: (id: string) => materialsApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['materials'] })
      queryClient.invalidateQueries({ queryKey: ['materials', 'supply'] })
    },
  })

  const [confirmDeleteId, setConfirmDeleteId] = useState<string | null>(null)

  const handleDeleteMaterial = (material: Material) => {
    if (confirmDeleteId === material.id) {
      deleteMaterial.mutate(material.id)
      setConfirmDeleteId(null)
    } else {
      setConfirmDeleteId(material.id)
    }
  }

  const [showAddMaterial, setShowAddMaterial] = useState(false)
  const [showAddSpool, setShowAddSpool] = useState(false)
  const [tab, setTab] = useState<'spools' | 'catalog' | 'supplies'>('spools')

  const handleCreateMaterial = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    const formData = new FormData(e.currentTarget)
    
    await createMaterial.mutateAsync({
      name: formData.get('name') as string,
      type: formData.get('type') as MaterialType,
      manufacturer: formData.get('manufacturer') as string,
      color: formData.get('color') as string,
      color_hex: formData.get('color_hex') as string,
      cost_per_kg: parseFloat(formData.get('cost_per_kg') as string) || 0,
    })
    
    setShowAddMaterial(false)
  }

  const handleCreateSpool = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault()
    const formData = new FormData(e.currentTarget)
    
    await createSpool.mutateAsync({
      material_id: formData.get('material_id') as string,
      initial_weight: parseFloat(formData.get('initial_weight') as string) || 1000,
      remaining_weight: parseFloat(formData.get('initial_weight') as string) || 1000,
      purchase_cost: parseFloat(formData.get('purchase_cost') as string) || 0,
      location: formData.get('location') as string,
    })
    
    setShowAddSpool(false)
  }

  const getMaterialById = (id: string) => materials.find(m => m.id === id)

  return (
    <div className="p-4 sm:p-6 lg:p-8">
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-display font-bold text-surface-100">
            Materials
          </h1>
          <p className="text-surface-400 mt-1">
            Manage your filament inventory and supplies
          </p>
        </div>
        <div className="flex gap-2">
          <button 
            onClick={() => setShowAddMaterial(true)}
            className="btn btn-secondary"
          >
            <Plus className="h-4 w-4 mr-2" />
            Add Material
          </button>
          <button 
            onClick={() => setShowAddSpool(true)}
            className="btn btn-primary"
          >
            <Plus className="h-4 w-4 mr-2" />
            Add Spool
          </button>
        </div>
      </div>

      {/* Tabs */}
      <div className="flex gap-2 mb-6">
        <button
          onClick={() => setTab('spools')}
          className={cn(
            'px-4 py-2 rounded-lg text-sm font-medium transition-colors',
            tab === 'spools'
              ? 'bg-accent-500/20 text-accent-400'
              : 'text-surface-400 hover:text-surface-100 hover:bg-surface-800'
          )}
        >
          <Droplet className="h-4 w-4 inline mr-2" />
          Inventory ({spools.length})
        </button>
        <button
          onClick={() => setTab('catalog')}
          className={cn(
            'px-4 py-2 rounded-lg text-sm font-medium transition-colors',
            tab === 'catalog'
              ? 'bg-accent-500/20 text-accent-400'
              : 'text-surface-400 hover:text-surface-100 hover:bg-surface-800'
          )}
        >
          <Package className="h-4 w-4 inline mr-2" />
          Catalog ({filamentMaterials.length})
        </button>
        <button
          onClick={() => setTab('supplies')}
          className={cn(
            'px-4 py-2 rounded-lg text-sm font-medium transition-colors',
            tab === 'supplies'
              ? 'bg-accent-500/20 text-accent-400'
              : 'text-surface-400 hover:text-surface-100 hover:bg-surface-800'
          )}
        >
          <ShoppingCart className="h-4 w-4 inline mr-2" />
          Supplies ({supplyMaterials.length})
        </button>
      </div>

      {/* Spools Tab */}
      {tab === 'spools' && (
        spoolsLoading ? (
          <div className="text-surface-500">Loading...</div>
        ) : spools.length === 0 ? (
          <div className="text-center py-16">
            <Droplet className="h-16 w-16 mx-auto mb-4 text-surface-600" />
            <h3 className="text-xl font-semibold text-surface-300 mb-2">
              No spools in inventory
            </h3>
            <p className="text-surface-500 mb-4">
              Add your first spool to start tracking material usage
            </p>
            <button 
              onClick={() => setShowAddSpool(true)}
              className="btn btn-primary"
            >
              <Plus className="h-4 w-4 mr-2" />
              Add Spool
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
            {spools.map((spool) => {
              const material = getMaterialById(spool.material_id)
              const percentRemaining = (spool.remaining_weight / spool.initial_weight) * 100
              
              return (
                <div key={spool.id} className="card p-4">
                  <div className="flex items-center gap-3 mb-3">
                    <div 
                      className="w-8 h-8 rounded-full border-2 border-surface-700"
                      style={{ backgroundColor: material?.color_hex || '#666' }}
                    />
                    <div className="flex-1 min-w-0">
                      <h3 className="font-medium text-surface-100 truncate">
                        {material?.name || 'Unknown'}
                      </h3>
                      <p className="text-xs text-surface-500">
                        {material?.type?.toUpperCase()}
                      </p>
                    </div>
                  </div>
                  
                  <div className="space-y-2">
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-surface-500">Remaining</span>
                      <span className="text-surface-300">
                        {spool.remaining_weight.toFixed(0)}g / {spool.initial_weight.toFixed(0)}g
                      </span>
                    </div>
                    <div className="h-2 bg-surface-800 rounded-full overflow-hidden">
                      <div 
                        className={cn(
                          'h-full transition-all',
                          percentRemaining > 30 ? 'bg-emerald-500' :
                          percentRemaining > 10 ? 'bg-amber-500' :
                          'bg-red-500'
                        )}
                        style={{ width: `${percentRemaining}%` }}
                      />
                    </div>
                    <div className="flex items-center justify-between">
                      <span className={cn('badge', getStatusBadge(spool.status))}>
                        {spool.status}
                      </span>
                      {spool.location && (
                        <span className="text-xs text-surface-500">
                          📍 {spool.location}
                        </span>
                      )}
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        )
      )}

      {/* Catalog Tab */}
      {tab === 'catalog' && (
        materialsLoading ? (
          <div className="text-surface-500">Loading...</div>
        ) : filamentMaterials.length === 0 ? (
          <div className="text-center py-16">
            <Package className="h-16 w-16 mx-auto mb-4 text-surface-600" />
            <h3 className="text-xl font-semibold text-surface-300 mb-2">
              No materials in catalog
            </h3>
            <p className="text-surface-500 mb-4">
              Add materials to your catalog before creating spools
            </p>
            <button
              onClick={() => setShowAddMaterial(true)}
              className="btn btn-primary"
            >
              <Plus className="h-4 w-4 mr-2" />
              Add Material
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
            {filamentMaterials.map((material) => (
              <div key={material.id} className="card p-4">
                <div className="flex items-center gap-3 mb-3">
                  <div
                    className="w-10 h-10 rounded-full border-2 border-surface-700"
                    style={{ backgroundColor: material.color_hex || '#666' }}
                  />
                  <div className="flex-1 min-w-0">
                    <h3 className="font-medium text-surface-100 truncate">
                      {material.name}
                    </h3>
                    <p className="text-xs text-surface-500">
                      {material.manufacturer || material.type.toUpperCase()}
                    </p>
                  </div>
                  {confirmDeleteId === material.id ? (
                    <button
                      onClick={() => handleDeleteMaterial(material)}
                      onBlur={() => setConfirmDeleteId(null)}
                      className="text-xs px-2 py-1 rounded bg-red-500/20 text-red-400 border border-red-500/50 hover:bg-red-500/30 transition-colors"
                      autoFocus
                    >
                      Delete?
                    </button>
                  ) : (
                    <button
                      onClick={() => handleDeleteMaterial(material)}
                      className="p-1.5 rounded-lg text-surface-500 hover:text-red-400 hover:bg-red-500/10 transition-colors"
                      title="Delete material"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  )}
                </div>
                <div className="space-y-1 text-sm">
                  <div className="flex justify-between">
                    <span className="text-surface-500">Type</span>
                    <span className="text-surface-300">{material.type.toUpperCase()}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-surface-500">Color</span>
                    <span className="text-surface-300">{material.color || '—'}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-surface-500">Cost</span>
                    <span className="text-surface-300">${material.cost_per_kg.toFixed(2)}/kg</span>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )
      )}

      {/* Supplies Tab */}
      {tab === 'supplies' && (
        materialsLoading ? (
          <div className="text-surface-500">Loading...</div>
        ) : supplyMaterials.length === 0 ? (
          <div className="text-center py-16">
            <ShoppingCart className="h-16 w-16 mx-auto mb-4 text-surface-600" />
            <h3 className="text-xl font-semibold text-surface-300 mb-2">
              No supplies yet
            </h3>
            <p className="text-surface-500 mb-4">
              Upload Amazon or other receipts to auto-create supply materials
            </p>
          </div>
        ) : (
          <div className="card overflow-hidden">
            <table className="w-full">
              <thead>
                <tr className="border-b border-surface-800">
                  <th className="text-left text-xs font-medium text-surface-500 uppercase px-4 py-2">Item</th>
                  <th className="text-left text-xs font-medium text-surface-500 uppercase px-4 py-2">Vendor</th>
                  <th className="text-right text-xs font-medium text-surface-500 uppercase px-4 py-2">Unit Cost</th>
                  <th className="w-10 px-4 py-2"></th>
                </tr>
              </thead>
              <tbody>
                {supplyMaterials.map((material) => (
                  <tr key={material.id} className="border-b border-surface-800/50">
                    <td className="px-4 py-3 text-surface-200">{material.name}</td>
                    <td className="px-4 py-3 text-surface-400">{material.manufacturer || '—'}</td>
                    <td className="px-4 py-3 text-right text-surface-300">
                      ${material.cost_per_kg.toFixed(2)}
                    </td>
                    <td className="px-4 py-3">
                      {confirmDeleteId === material.id ? (
                        <button
                          onClick={() => handleDeleteMaterial(material)}
                          onBlur={() => setConfirmDeleteId(null)}
                          className="text-xs px-2 py-1 rounded bg-red-500/20 text-red-400 border border-red-500/50 hover:bg-red-500/30 transition-colors"
                          autoFocus
                        >
                          Delete?
                        </button>
                      ) : (
                        <button
                          onClick={() => handleDeleteMaterial(material)}
                          className="p-1 rounded text-surface-500 hover:text-red-400 hover:bg-red-500/10 transition-colors"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )
      )}

      {/* Add Material Modal */}
      {showAddMaterial && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
          <div className="card w-full max-w-md p-6">
            <h2 className="text-xl font-semibold text-surface-100 mb-4">
              Add Material
            </h2>
            <form onSubmit={handleCreateMaterial}>
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-surface-300 mb-1">
                    Name *
                  </label>
                  <input
                    type="text"
                    name="name"
                    required
                    className="input"
                    placeholder="Prusament PLA Galaxy Black"
                    autoFocus
                  />
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-surface-300 mb-1">
                      Type *
                    </label>
                    <select name="type" required className="input">
                      <option value="pla">PLA</option>
                      <option value="petg">PETG</option>
                      <option value="abs">ABS</option>
                      <option value="asa">ASA</option>
                      <option value="tpu">TPU</option>
                    </select>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-surface-300 mb-1">
                      Manufacturer
                    </label>
                    <input
                      type="text"
                      name="manufacturer"
                      className="input"
                      placeholder="Prusament"
                    />
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label className="block text-sm font-medium text-surface-300 mb-1">
                      Color
                    </label>
                    <input
                      type="text"
                      name="color"
                      className="input"
                      placeholder="Galaxy Black"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-surface-300 mb-1">
                      Color Hex
                    </label>
                    <input
                      type="color"
                      name="color_hex"
                      className="input h-10 p-1"
                      defaultValue="#1a1a2e"
                    />
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-medium text-surface-300 mb-1">
                    Cost per kg ($)
                  </label>
                  <input
                    type="number"
                    name="cost_per_kg"
                    step="0.01"
                    className="input"
                    placeholder="25.00"
                  />
                </div>
              </div>
              <div className="flex justify-end gap-3 mt-6">
                <button
                  type="button"
                  onClick={() => setShowAddMaterial(false)}
                  className="btn btn-ghost"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={createMaterial.isPending}
                  className="btn btn-primary"
                >
                  {createMaterial.isPending ? 'Adding...' : 'Add Material'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Add Spool Modal */}
      {showAddSpool && (
        <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
          <div className="card w-full max-w-md p-6">
            <h2 className="text-xl font-semibold text-surface-100 mb-4">
              Add Spool
            </h2>
            {materials.length === 0 ? (
              <div className="text-center py-4">
                <p className="text-surface-500 mb-4">
                  You need to add a material first
                </p>
                <button
                  onClick={() => {
                    setShowAddSpool(false)
                    setShowAddMaterial(true)
                  }}
                  className="btn btn-primary"
                >
                  Add Material
                </button>
              </div>
            ) : (
              <form onSubmit={handleCreateSpool}>
                <div className="space-y-4">
                  <div>
                    <label className="block text-sm font-medium text-surface-300 mb-1">
                      Material *
                    </label>
                    <select name="material_id" required className="input">
                      <option value="">Select material...</option>
                      {materials.map((m) => (
                        <option key={m.id} value={m.id}>
                          {m.name} ({m.type.toUpperCase()})
                        </option>
                      ))}
                    </select>
                  </div>
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <label className="block text-sm font-medium text-surface-300 mb-1">
                        Initial Weight (g)
                      </label>
                      <input
                        type="number"
                        name="initial_weight"
                        defaultValue="1000"
                        className="input"
                      />
                    </div>
                    <div>
                      <label className="block text-sm font-medium text-surface-300 mb-1">
                        Purchase Cost ($)
                      </label>
                      <input
                        type="number"
                        name="purchase_cost"
                        step="0.01"
                        className="input"
                        placeholder="25.00"
                      />
                    </div>
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-surface-300 mb-1">
                      Location
                    </label>
                    <input
                      type="text"
                      name="location"
                      className="input"
                      placeholder="Dry Box A, Shelf 3"
                    />
                  </div>
                </div>
                <div className="flex justify-end gap-3 mt-6">
                  <button
                    type="button"
                    onClick={() => setShowAddSpool(false)}
                    className="btn btn-ghost"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={createSpool.isPending}
                    className="btn btn-primary"
                  >
                    {createSpool.isPending ? 'Adding...' : 'Add Spool'}
                  </button>
                </div>
              </form>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

