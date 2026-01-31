import { useEffect, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import {
  ArrowLeft,
  Package,
  Clock,
  CheckCircle,
  Truck,
  XCircle,
  RefreshCw,
  Plus,
  Play,
  Mail,
  Calendar,
} from 'lucide-react'
import { ordersApi, templatesApi } from '../api/client'
import type { Order, OrderProgress, OrderStatus, Template } from '../types'

const statusConfig: Record<OrderStatus, { label: string; color: string; icon: React.ReactNode }> = {
  pending: { label: 'Pending', color: 'bg-gray-100 text-gray-700', icon: <Clock className="w-4 h-4" /> },
  in_progress: { label: 'In Progress', color: 'bg-blue-100 text-blue-700', icon: <RefreshCw className="w-4 h-4" /> },
  completed: { label: 'Completed', color: 'bg-green-100 text-green-700', icon: <CheckCircle className="w-4 h-4" /> },
  shipped: { label: 'Shipped', color: 'bg-purple-100 text-purple-700', icon: <Truck className="w-4 h-4" /> },
  cancelled: { label: 'Cancelled', color: 'bg-red-100 text-red-700', icon: <XCircle className="w-4 h-4" /> },
}

export function OrderDetail() {
  const { id } = useParams<{ id: string }>()
  const [order, setOrder] = useState<Order | null>(null)
  const [progress, setProgress] = useState<OrderProgress | null>(null)
  const [loading, setLoading] = useState(true)
  const [showAddItem, setShowAddItem] = useState(false)
  const [processingItem, setProcessingItem] = useState<string | null>(null)
  const [updatingStatus, setUpdatingStatus] = useState(false)

  useEffect(() => {
    if (id) {
      loadOrder()
    }
  }, [id])

  const loadOrder = async () => {
    if (!id) return
    setLoading(true)
    try {
      const [orderData, progressData] = await Promise.all([
        ordersApi.get(id),
        ordersApi.getProgress(id),
      ])
      setOrder(orderData)
      setProgress(progressData)
    } catch (err) {
      console.error('Failed to load order:', err)
    } finally {
      setLoading(false)
    }
  }

  const handleStatusChange = async (newStatus: OrderStatus) => {
    if (!id || !order) return
    setUpdatingStatus(true)
    try {
      const updated = await ordersApi.updateStatus(id, newStatus)
      setOrder(updated)
    } catch (err) {
      console.error('Failed to update status:', err)
    } finally {
      setUpdatingStatus(false)
    }
  }

  const handleProcessItem = async (itemId: string) => {
    if (!id) return
    setProcessingItem(itemId)
    try {
      await ordersApi.processItem(id, itemId)
      loadOrder()
    } catch (err) {
      console.error('Failed to process item:', err)
    } finally {
      setProcessingItem(null)
    }
  }

  const handleShip = async () => {
    if (!id) return
    const trackingNumber = prompt('Enter tracking number (optional):')
    try {
      const updated = await ordersApi.markShipped(id, trackingNumber || undefined)
      setOrder(updated)
    } catch (err) {
      console.error('Failed to mark as shipped:', err)
    }
  }

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return '-'
    return new Date(dateStr).toLocaleString()
  }

  if (loading) {
    return (
      <div className="p-6">
        <div className="text-center py-12 text-gray-500">Loading order...</div>
      </div>
    )
  }

  if (!order) {
    return (
      <div className="p-6">
        <div className="text-center py-12">
          <Package className="w-12 h-12 mx-auto text-gray-400 mb-4" />
          <h3 className="text-lg font-medium text-gray-900 mb-2">Order not found</h3>
          <Link to="/orders" className="text-blue-600 hover:text-blue-700">
            Back to orders
          </Link>
        </div>
      </div>
    )
  }

  return (
    <div className="p-6">
      {/* Header */}
      <div className="flex items-center gap-4 mb-6">
        <Link to="/orders" className="p-2 hover:bg-gray-100 rounded-lg transition-colors">
          <ArrowLeft className="w-5 h-5" />
        </Link>
        <div className="flex-1">
          <h1 className="text-2xl font-bold">{order.customer_name}</h1>
          <p className="text-gray-600">
            {order.source.charAt(0).toUpperCase() + order.source.slice(1)} Order
            {order.source_order_id && ` #${order.source_order_id}`}
          </p>
        </div>
        <span className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-sm font-medium ${statusConfig[order.status].color}`}>
          {statusConfig[order.status].icon}
          {statusConfig[order.status].label}
        </span>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Main content */}
        <div className="lg:col-span-2 space-y-6">
          {/* Progress */}
          {progress && (
            <div className="bg-white rounded-lg border p-6">
              <h2 className="text-lg font-semibold mb-4">Progress</h2>
              <div className="mb-4">
                <div className="flex justify-between text-sm text-gray-600 mb-1">
                  <span>{progress.completed_jobs} of {progress.total_jobs} jobs complete</span>
                  <span>{Math.round(progress.progress_percent)}%</span>
                </div>
                <div className="h-2 bg-gray-200 rounded-full overflow-hidden">
                  <div
                    className="h-full bg-blue-600 rounded-full transition-all duration-300"
                    style={{ width: `${progress.progress_percent}%` }}
                  />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4 text-center">
                <div className="p-3 bg-gray-50 rounded-lg">
                  <div className="text-2xl font-bold text-gray-900">{progress.completed_items}</div>
                  <div className="text-sm text-gray-500">Items Processed</div>
                </div>
                <div className="p-3 bg-gray-50 rounded-lg">
                  <div className="text-2xl font-bold text-gray-900">{progress.total_items}</div>
                  <div className="text-sm text-gray-500">Total Items</div>
                </div>
              </div>
            </div>
          )}

          {/* Items */}
          <div className="bg-white rounded-lg border p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold">Order Items</h2>
              <button
                onClick={() => setShowAddItem(true)}
                className="inline-flex items-center gap-1 text-sm text-blue-600 hover:text-blue-700"
              >
                <Plus className="w-4 h-4" />
                Add Item
              </button>
            </div>

            {order.items && order.items.length > 0 ? (
              <div className="space-y-3">
                {order.items.map(item => (
                  <div
                    key={item.id}
                    className="flex items-center justify-between p-4 border rounded-lg"
                  >
                    <div>
                      <div className="font-medium">{item.sku || 'No SKU'}</div>
                      <div className="text-sm text-gray-500">Qty: {item.quantity}</div>
                      {item.notes && (
                        <div className="text-sm text-gray-500 mt-1">{item.notes}</div>
                      )}
                    </div>
                    <div className="flex items-center gap-2">
                      {item.template_id ? (
                        <button
                          onClick={() => handleProcessItem(item.id)}
                          disabled={processingItem === item.id}
                          className="inline-flex items-center gap-1 px-3 py-1.5 text-sm bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
                        >
                          <Play className="w-4 h-4" />
                          {processingItem === item.id ? 'Processing...' : 'Process'}
                        </button>
                      ) : (
                        <span className="text-sm text-amber-600">No template linked</span>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="text-center py-8 text-gray-500">
                No items yet. Add items to this order.
              </div>
            )}
          </div>

          {/* Tasks */}
          {order.tasks && order.tasks.length > 0 && (
            <div className="bg-white rounded-lg border p-6">
              <h2 className="text-lg font-semibold mb-4">Tasks</h2>
              <div className="space-y-2">
                {order.tasks.map(task => (
                  <Link
                    key={task.id}
                    to={`/tasks/${task.id}`}
                    className="block p-3 border rounded-lg hover:bg-gray-50 transition-colors"
                  >
                    <div className="font-medium">{task.name}</div>
                    <div className="text-sm text-gray-500">
                      Status: {task.status} | Progress: {Math.round(task.progress || 0)}%
                    </div>
                  </Link>
                ))}
              </div>
            </div>
          )}

          {/* Events */}
          {order.events && order.events.length > 0 && (
            <div className="bg-white rounded-lg border p-6">
              <h2 className="text-lg font-semibold mb-4">Activity</h2>
              <div className="space-y-3">
                {order.events.map(event => (
                  <div key={event.id} className="flex gap-3">
                    <div className="w-2 h-2 rounded-full bg-gray-400 mt-2" />
                    <div>
                      <div className="text-sm">{event.message || event.event_type}</div>
                      <div className="text-xs text-gray-500">{formatDate(event.created_at)}</div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          {/* Actions */}
          <div className="bg-white rounded-lg border p-6">
            <h2 className="text-lg font-semibold mb-4">Actions</h2>
            <div className="space-y-2">
              {order.status === 'pending' && (
                <button
                  onClick={() => handleStatusChange('in_progress')}
                  disabled={updatingStatus}
                  className="w-full px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
                >
                  Start Processing
                </button>
              )}
              {order.status === 'in_progress' && (
                <button
                  onClick={() => handleStatusChange('completed')}
                  disabled={updatingStatus}
                  className="w-full px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 disabled:opacity-50"
                >
                  Mark Completed
                </button>
              )}
              {order.status === 'completed' && (
                <button
                  onClick={handleShip}
                  disabled={updatingStatus}
                  className="w-full px-4 py-2 bg-purple-600 text-white rounded-lg hover:bg-purple-700 disabled:opacity-50"
                >
                  Mark Shipped
                </button>
              )}
              {(order.status === 'pending' || order.status === 'in_progress') && (
                <button
                  onClick={() => handleStatusChange('cancelled')}
                  disabled={updatingStatus}
                  className="w-full px-4 py-2 text-red-600 border border-red-200 rounded-lg hover:bg-red-50"
                >
                  Cancel Order
                </button>
              )}
            </div>
          </div>

          {/* Details */}
          <div className="bg-white rounded-lg border p-6">
            <h2 className="text-lg font-semibold mb-4">Details</h2>
            <dl className="space-y-3">
              {order.customer_email && (
                <div>
                  <dt className="text-sm text-gray-500 flex items-center gap-1">
                    <Mail className="w-4 h-4" /> Email
                  </dt>
                  <dd className="text-sm font-medium">{order.customer_email}</dd>
                </div>
              )}
              {order.due_date && (
                <div>
                  <dt className="text-sm text-gray-500 flex items-center gap-1">
                    <Calendar className="w-4 h-4" /> Due Date
                  </dt>
                  <dd className="text-sm font-medium">{formatDate(order.due_date)}</dd>
                </div>
              )}
              <div>
                <dt className="text-sm text-gray-500">Created</dt>
                <dd className="text-sm font-medium">{formatDate(order.created_at)}</dd>
              </div>
              <div>
                <dt className="text-sm text-gray-500">Priority</dt>
                <dd className="text-sm font-medium">
                  {order.priority === 0 ? 'Normal' : order.priority === 1 ? 'High' : 'Urgent'}
                </dd>
              </div>
              {order.notes && (
                <div>
                  <dt className="text-sm text-gray-500">Notes</dt>
                  <dd className="text-sm">{order.notes}</dd>
                </div>
              )}
            </dl>
          </div>
        </div>
      </div>

      {/* Add Item Modal */}
      {showAddItem && (
        <AddItemModal
          orderId={order.id}
          onClose={() => setShowAddItem(false)}
          onAdded={() => {
            setShowAddItem(false)
            loadOrder()
          }}
        />
      )}
    </div>
  )
}

interface AddItemModalProps {
  orderId: string
  onClose: () => void
  onAdded: () => void
}

function AddItemModal({ orderId, onClose, onAdded }: AddItemModalProps) {
  const [templates, setTemplates] = useState<Template[]>([])
  const [templateId, setTemplateId] = useState('')
  const [sku, setSku] = useState('')
  const [quantity, setQuantity] = useState(1)
  const [notes, setNotes] = useState('')
  const [adding, setAdding] = useState(false)

  useEffect(() => {
    loadTemplates()
  }, [])

  const loadTemplates = async () => {
    try {
      const data = await templatesApi.list()
      setTemplates(data)
    } catch (err) {
      console.error('Failed to load templates:', err)
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setAdding(true)
    try {
      await ordersApi.addItem(orderId, {
        template_id: templateId || undefined,
        sku: sku.trim() || undefined,
        quantity,
        notes: notes.trim() || undefined,
      })
      onAdded()
    } catch (err) {
      console.error('Failed to add item:', err)
    } finally {
      setAdding(false)
    }
  }

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-white rounded-lg shadow-xl w-full max-w-md p-6">
        <h2 className="text-xl font-bold mb-4">Add Order Item</h2>

        <form onSubmit={handleSubmit}>
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Template (Recipe)
              </label>
              <select
                value={templateId}
                onChange={e => setTemplateId(e.target.value)}
                className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              >
                <option value="">No template</option>
                {templates.map(t => (
                  <option key={t.id} value={t.id}>
                    {t.name} {t.sku && `(${t.sku})`}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                SKU
              </label>
              <input
                type="text"
                value={sku}
                onChange={e => setSku(e.target.value)}
                className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder="Optional SKU"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Quantity
              </label>
              <input
                type="number"
                min={1}
                value={quantity}
                onChange={e => setQuantity(Number(e.target.value))}
                className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Notes
              </label>
              <textarea
                value={notes}
                onChange={e => setNotes(e.target.value)}
                className="w-full px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
                rows={2}
              />
            </div>
          </div>

          <div className="flex justify-end gap-3 mt-6">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-gray-600 hover:text-gray-800"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={adding}
              className="px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
            >
              {adding ? 'Adding...' : 'Add Item'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

export default OrderDetail
