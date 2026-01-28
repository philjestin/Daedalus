import { useState, useCallback } from 'react'
import { useQuery, useQueryClient, useMutation } from '@tanstack/react-query'
import {
  Receipt,
  Upload,
  CheckCircle,
  Clock,
  AlertTriangle,
  Trash2,
  Eye,
  DollarSign,
  Package,
  X,
  RotateCcw,
  Loader2
} from 'lucide-react'
import { expensesApi } from '../api/client'
import { useMaterials } from '../hooks/useMaterials'
import { cn, formatRelativeTime } from '../lib/utils'
import type { Expense, ExpenseCategory } from '../types'

export default function Expenses() {
  const queryClient = useQueryClient()
  const [showUpload, setShowUpload] = useState(false)
  const [reviewExpense, setReviewExpense] = useState<Expense | null>(null)

  const { data: expenses = [], isLoading } = useQuery({
    queryKey: ['expenses'],
    queryFn: () => expensesApi.list(),
    refetchInterval: 5000, // Poll for parsing status
  })

  const deleteExpense = useMutation({
    mutationFn: (id: string) => expensesApi.delete(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['expenses'] })
    },
  })

  const retryExpense = useMutation({
    mutationFn: (id: string) => expensesApi.retry(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['expenses'] })
    },
  })

  // Calculate totals
  const totalExpenses = expenses
    .filter((e) => e.status === 'confirmed')
    .reduce((sum, e) => sum + e.total_cents, 0)

  const pendingCount = expenses.filter((e) => e.status === 'pending').length

  const formatCents = (cents: number) => {
    return `$${(cents / 100).toFixed(2)}`
  }

  const getCategoryIcon = (category: ExpenseCategory) => {
    switch (category) {
      case 'filament':
        return <Package className="h-4 w-4" />
      default:
        return <DollarSign className="h-4 w-4" />
    }
  }

  return (
    <div className="p-4 sm:p-6 lg:p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-display font-bold text-surface-100">
            Expenses
          </h1>
          <p className="text-surface-400 mt-1">
            Track purchases and scan receipts
          </p>
        </div>
        <button
          onClick={() => setShowUpload(true)}
          className="btn btn-primary"
        >
          <Upload className="h-4 w-4 mr-2" />
          Upload Receipt
        </button>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
        <div className="card p-4">
          <div className="text-sm text-surface-500 mb-1">Total Expenses</div>
          <div className="text-2xl font-semibold text-surface-100">
            {formatCents(totalExpenses)}
          </div>
        </div>
        <div className="card p-4">
          <div className="text-sm text-surface-500 mb-1">Confirmed</div>
          <div className="text-2xl font-semibold text-emerald-400">
            {expenses.filter((e) => e.status === 'confirmed').length}
          </div>
        </div>
        <div className="card p-4">
          <div className="text-sm text-surface-500 mb-1">Pending Review</div>
          <div className="text-2xl font-semibold text-amber-400">
            {pendingCount}
          </div>
        </div>
      </div>

      {/* Expenses List */}
      {isLoading ? (
        <div className="text-surface-500">Loading expenses...</div>
      ) : expenses.length === 0 ? (
        <div className="card p-8 text-center">
          <Receipt className="h-12 w-12 mx-auto mb-3 text-surface-600" />
          <h3 className="text-lg font-medium text-surface-300 mb-2">
            No expenses yet
          </h3>
          <p className="text-surface-500 mb-4">
            Upload a receipt to start tracking expenses
          </p>
          <button
            onClick={() => setShowUpload(true)}
            className="btn btn-primary"
          >
            <Upload className="h-4 w-4 mr-2" />
            Upload Receipt
          </button>
        </div>
      ) : (
        <div className="space-y-3">
          {expenses.map((expense) => (
            <div
              key={expense.id}
              className="card p-4 flex items-center justify-between"
            >
              <div className="flex items-center gap-4">
                <div
                  className={cn(
                    'w-10 h-10 rounded-lg flex items-center justify-center',
                    expense.status === 'confirmed'
                      ? 'bg-emerald-500/20 text-emerald-400'
                      : expense.status === 'rejected'
                      ? 'bg-red-500/20 text-red-400'
                      : expense.vendor
                      ? 'bg-amber-500/20 text-amber-400'
                      : 'bg-blue-500/20 text-blue-400'
                  )}
                >
                  {expense.status === 'confirmed' ? (
                    <CheckCircle className="h-5 w-5" />
                  ) : expense.status === 'rejected' ? (
                    <AlertTriangle className="h-5 w-5" />
                  ) : expense.vendor ? (
                    <Clock className="h-5 w-5" />
                  ) : (
                    <Loader2 className="h-5 w-5 animate-spin" />
                  )}
                </div>

                <div>
                  <div className="font-medium text-surface-100">
                    {expense.vendor || (expense.status === 'rejected' ? 'Parse Failed' : 'Processing...')}
                  </div>
                  <div className="text-sm text-surface-500 flex items-center gap-2">
                    <span>{formatRelativeTime(expense.occurred_at)}</span>
                    <span>•</span>
                    <span className="flex items-center gap-1">
                      {getCategoryIcon(expense.category)}
                      {expense.category}
                    </span>
                    {expense.confidence > 0 && (
                      <>
                        <span>•</span>
                        <span
                          className={cn(
                            expense.confidence >= 80
                              ? 'text-emerald-400'
                              : expense.confidence >= 50
                              ? 'text-amber-400'
                              : 'text-red-400'
                          )}
                        >
                          {expense.confidence}% confidence
                        </span>
                      </>
                    )}
                  </div>
                  {expense.status === 'rejected' && expense.notes && (
                    <div className="text-xs text-red-400 mt-1 max-w-md truncate">
                      {expense.notes}
                    </div>
                  )}
                </div>
              </div>

              <div className="flex items-center gap-4">
                <div className="text-right">
                  <div className="font-semibold text-surface-100">
                    {formatCents(expense.total_cents)}
                  </div>
                  <div className="text-xs text-surface-500">
                    {expense.status}
                  </div>
                </div>

                <div className="flex items-center gap-2">
                  {expense.status === 'pending' && expense.vendor && (
                    <button
                      onClick={() => setReviewExpense(expense)}
                      className="btn btn-secondary text-xs py-1 px-2"
                    >
                      <Eye className="h-3 w-3 mr-1" />
                      Review
                    </button>
                  )}
                  {(expense.status === 'rejected' || (expense.status === 'pending' && !expense.vendor)) && (
                    <button
                      onClick={() => retryExpense.mutate(expense.id)}
                      disabled={retryExpense.isPending}
                      className="btn btn-secondary text-xs py-1 px-2"
                    >
                      {retryExpense.isPending ? (
                        <Loader2 className="h-3 w-3 mr-1 animate-spin" />
                      ) : (
                        <RotateCcw className="h-3 w-3 mr-1" />
                      )}
                      Retry
                    </button>
                  )}
                  <button
                    onClick={() => deleteExpense.mutate(expense.id)}
                    className="btn btn-ghost text-xs py-1 px-2 text-red-400 hover:text-red-300"
                  >
                    <Trash2 className="h-3 w-3" />
                  </button>
                </div>
              </div>
            </div>
          ))}
        </div>
      )}

      {/* Upload Modal */}
      {showUpload && (
        <ReceiptUploadModal
          onClose={() => setShowUpload(false)}
          onSuccess={() => {
            queryClient.invalidateQueries({ queryKey: ['expenses'] })
            setShowUpload(false)
          }}
        />
      )}

      {/* Review Modal */}
      {reviewExpense && (
        <ExpenseReviewModal
          expenseId={reviewExpense.id}
          onClose={() => setReviewExpense(null)}
          onSuccess={() => {
            queryClient.invalidateQueries({ queryKey: ['expenses'] })
            queryClient.invalidateQueries({ queryKey: ['spools'] })
            queryClient.invalidateQueries({ queryKey: ['materials'] })
            setReviewExpense(null)
          }}
        />
      )}
    </div>
  )
}

// Receipt Upload Modal
function ReceiptUploadModal({
  onClose,
  onSuccess,
}: {
  onClose: () => void
  onSuccess: () => void
}) {
  const [file, setFile] = useState<File | null>(null)
  const [uploading, setUploading] = useState(false)
  const [dragOver, setDragOver] = useState(false)
  const [uploadError, setUploadError] = useState<string | null>(null)

  const handleUpload = async () => {
    if (!file) return

    setUploading(true)
    setUploadError(null)
    try {
      await expensesApi.uploadReceipt(file)
      onSuccess()
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Upload failed'
      setUploadError(msg)
    } finally {
      setUploading(false)
    }
  }

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    setDragOver(false)
    const droppedFile = e.dataTransfer.files[0]
    if (droppedFile) {
      setFile(droppedFile)
    }
  }, [])

  return (
    <div
      className="fixed inset-0 bg-black/60 flex items-center justify-center z-50"
      onClick={(e) => e.target === e.currentTarget && onClose()}
    >
      <div className="card w-full max-w-md p-6">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl font-semibold text-surface-100">
            Upload Receipt
          </h2>
          <button onClick={onClose} className="text-surface-400 hover:text-surface-200">
            <X className="h-5 w-5" />
          </button>
        </div>

        <div
          onDragOver={(e) => {
            e.preventDefault()
            setDragOver(true)
          }}
          onDragLeave={() => setDragOver(false)}
          onDrop={handleDrop}
          className={cn(
            'border-2 border-dashed rounded-lg p-8 text-center transition-colors',
            dragOver
              ? 'border-accent-500 bg-accent-500/10'
              : file
              ? 'border-emerald-500 bg-emerald-500/5'
              : 'border-surface-700 hover:border-surface-600'
          )}
        >
          <input
            type="file"
            accept="image/*,.pdf"
            onChange={(e) => setFile(e.target.files?.[0] || null)}
            className="hidden"
            id="receipt-upload"
          />
          <label htmlFor="receipt-upload" className="cursor-pointer">
            {file ? (
              <div>
                <Receipt className="h-10 w-10 mx-auto mb-2 text-emerald-500" />
                <p className="text-surface-100 font-medium">{file.name}</p>
                <p className="text-surface-500 text-sm">
                  {(file.size / 1024).toFixed(1)} KB
                </p>
              </div>
            ) : (
              <div>
                <Upload className="h-10 w-10 mx-auto mb-2 text-surface-500" />
                <p className="text-surface-300">
                  Drop receipt here or click to upload
                </p>
                <p className="text-surface-500 text-sm">
                  Supports images and PDFs
                </p>
              </div>
            )}
          </label>
        </div>

        <p className="text-sm text-surface-500 mt-4">
          AI will automatically extract vendor, items, and totals from your receipt.
          Filament purchases can be added to your spool inventory.
        </p>

        {uploadError && (
          <div className="mt-4 p-3 rounded-lg bg-red-500/10 border border-red-500/30 text-red-400 text-sm">
            {uploadError}
          </div>
        )}

        <div className="flex justify-end gap-3 mt-6">
          <button onClick={onClose} className="btn btn-ghost">
            Cancel
          </button>
          <button
            onClick={handleUpload}
            disabled={!file || uploading}
            className="btn btn-primary"
          >
            {uploading ? 'Uploading...' : 'Upload & Parse'}
          </button>
        </div>
      </div>
    </div>
  )
}

// Expense Review Modal
function ExpenseReviewModal({
  expenseId,
  onClose,
  onSuccess,
}: {
  expenseId: string
  onClose: () => void
  onSuccess: () => void
}) {
  const { data: expense, isLoading } = useQuery({
    queryKey: ['expenses', expenseId],
    queryFn: () => expensesApi.get(expenseId),
  })

  const { data: materials = [] } = useMaterials()

  const [itemDecisions, setItemDecisions] = useState<
    Record<string, { createSpool: boolean; materialId?: string; weightGrams?: number }>
  >({})

  const [submitting, setSubmitting] = useState(false)

  const handleConfirm = async () => {
    if (!expense?.items) return

    setSubmitting(true)
    try {
      const items = expense.items.map((item) => {
        const decision = itemDecisions[item.id] || { createSpool: false }
        return {
          item_id: item.id,
          create_spool: decision.createSpool,
          material_id: decision.materialId,
          weight_grams: decision.weightGrams || item.metadata?.weight_grams,
        }
      })

      await expensesApi.confirm(expenseId, items)
      onSuccess()
    } catch (err) {
      console.error('Failed to confirm expense:', err)
    } finally {
      setSubmitting(false)
    }
  }

  const formatCents = (cents: number) => `$${(cents / 100).toFixed(2)}`

  const toggleCreateSpool = (itemId: string) => {
    setItemDecisions((prev) => ({
      ...prev,
      [itemId]: {
        ...prev[itemId],
        createSpool: !prev[itemId]?.createSpool,
      },
    }))
  }

  const setMaterialId = (itemId: string, materialId: string) => {
    setItemDecisions((prev) => ({
      ...prev,
      [itemId]: {
        ...prev[itemId],
        materialId,
      },
    }))
  }

  if (isLoading || !expense) {
    return (
      <div className="fixed inset-0 bg-black/60 flex items-center justify-center z-50">
        <div className="card p-6">
          <div className="text-surface-400">Loading...</div>
        </div>
      </div>
    )
  }

  return (
    <div
      className="fixed inset-0 bg-black/60 flex items-center justify-center z-50 overflow-y-auto"
      onClick={(e) => e.target === e.currentTarget && onClose()}
    >
      <div className="card w-full max-w-2xl p-6 my-8">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-xl font-semibold text-surface-100">
            Review Expense
          </h2>
          <button onClick={onClose} className="text-surface-400 hover:text-surface-200">
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Expense Summary */}
        <div className="grid grid-cols-2 gap-4 mb-6">
          <div>
            <div className="text-sm text-surface-500">Vendor</div>
            <div className="font-medium text-surface-100">
              {expense.vendor || 'Unknown'}
            </div>
          </div>
          <div>
            <div className="text-sm text-surface-500">Date</div>
            <div className="font-medium text-surface-100">
              {new Date(expense.occurred_at).toLocaleDateString()}
            </div>
          </div>
          <div>
            <div className="text-sm text-surface-500">Subtotal</div>
            <div className="font-medium text-surface-100">
              {formatCents(expense.subtotal_cents)}
            </div>
          </div>
          <div>
            <div className="text-sm text-surface-500">Tax</div>
            <div className="font-medium text-surface-100">
              {formatCents(expense.tax_cents)}
            </div>
          </div>
          <div>
            <div className="text-sm text-surface-500">Shipping</div>
            <div className="font-medium text-surface-100">
              {formatCents(expense.shipping_cents)}
            </div>
          </div>
          <div>
            <div className="text-sm text-surface-500">Total</div>
            <div className="font-semibold text-emerald-400 text-lg">
              {formatCents(expense.total_cents)}
            </div>
          </div>
        </div>

        {/* Line Items */}
        <div className="mb-6">
          <h3 className="font-medium text-surface-100 mb-3">Line Items</h3>
          <div className="space-y-3">
            {expense.items?.map((item) => {
              const decision = itemDecisions[item.id] || { createSpool: false }
              const isFilament = item.category === 'filament' || item.metadata

              return (
                <div
                  key={item.id}
                  className="p-4 rounded-lg bg-surface-800/50 border border-surface-700"
                >
                  <div className="flex items-start justify-between mb-2">
                    <div>
                      <div className="font-medium text-surface-100">
                        {item.description}
                      </div>
                      <div className="text-sm text-surface-500">
                        Qty: {item.quantity} @ {formatCents(item.unit_price_cents)} each
                      </div>
                    </div>
                    <div className="text-right">
                      <div className="font-medium text-surface-100">
                        {formatCents(item.total_price_cents)}
                      </div>
                      <div className="text-xs text-surface-500">
                        {item.confidence}% confidence
                      </div>
                    </div>
                  </div>

                  {/* Filament metadata */}
                  {item.metadata && (
                    <div className="flex flex-wrap gap-2 mb-3">
                      {item.metadata.brand && (
                        <span className="badge bg-surface-700 text-surface-300">
                          {item.metadata.brand}
                        </span>
                      )}
                      {item.metadata.material_type && (
                        <span className="badge bg-blue-500/20 text-blue-400">
                          {item.metadata.material_type}
                        </span>
                      )}
                      {item.metadata.color && (
                        <span className="badge bg-surface-700 text-surface-300 flex items-center gap-1">
                          {item.metadata.color_hex && (
                            <span
                              className="w-3 h-3 rounded-full"
                              style={{ backgroundColor: item.metadata.color_hex }}
                            />
                          )}
                          {item.metadata.color}
                        </span>
                      )}
                      {item.metadata.weight_grams && (
                        <span className="badge bg-surface-700 text-surface-300">
                          {item.metadata.weight_grams}g
                        </span>
                      )}
                      {item.metadata.diameter_mm && (
                        <span className="badge bg-surface-700 text-surface-300">
                          {item.metadata.diameter_mm}mm
                        </span>
                      )}
                    </div>
                  )}

                  {/* Add to inventory option for filament */}
                  {isFilament && (
                    <div className="pt-3 border-t border-surface-700">
                      <label className="flex items-center gap-3 cursor-pointer">
                        <input
                          type="checkbox"
                          checked={decision.createSpool}
                          onChange={() => toggleCreateSpool(item.id)}
                          className="w-4 h-4 rounded border-surface-600 bg-surface-800 text-accent-500 focus:ring-accent-500"
                        />
                        <span className="text-sm text-surface-300">
                          Add to spool inventory
                        </span>
                      </label>

                      {decision.createSpool && (
                        <div className="mt-3 ml-7">
                          <label className="block text-sm text-surface-400 mb-1">
                            Match to material
                          </label>
                          <select
                            value={decision.materialId || ''}
                            onChange={(e) => setMaterialId(item.id, e.target.value)}
                            className="input text-sm"
                          >
                            <option value="">Select or create new...</option>
                            {materials.map((mat) => (
                              <option key={mat.id} value={mat.id}>
                                {mat.name} - {mat.type.toUpperCase()} ({mat.color})
                              </option>
                            ))}
                          </select>
                        </div>
                      )}
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        </div>

        <div className="flex justify-end gap-3">
          <button onClick={onClose} className="btn btn-ghost">
            Cancel
          </button>
          <button
            onClick={handleConfirm}
            disabled={submitting}
            className="btn btn-primary"
          >
            <CheckCircle className="h-4 w-4 mr-2" />
            {submitting ? 'Confirming...' : 'Confirm & Save'}
          </button>
        </div>
      </div>
    </div>
  )
}
