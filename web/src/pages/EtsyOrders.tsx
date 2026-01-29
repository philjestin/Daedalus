import { useState, useEffect, useCallback } from 'react'
import { RefreshCw, Package, CheckCircle, Clock, ExternalLink } from 'lucide-react'
import { etsyApi } from '../api/client'
import type { EtsyReceipt, SyncResult } from '../types'
import { cn } from '../lib/utils'

export default function EtsyOrders() {
  const [receipts, setReceipts] = useState<EtsyReceipt[]>([])
  const [loading, setLoading] = useState(true)
  const [syncing, setSyncing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [syncResult, setSyncResult] = useState<SyncResult | null>(null)
  const [filter, setFilter] = useState<'all' | 'unprocessed' | 'processed'>('all')
  const [processingId, setProcessingId] = useState<string | null>(null)

  const loadReceipts = useCallback(async function() {
    setLoading(true)
    setError(null)
    try {
      const processed = filter === 'all' ? undefined : filter === 'processed'
      const data = await etsyApi.listReceipts({ processed })
      setReceipts(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load orders')
    } finally {
      setLoading(false)
    }
  }, [filter])

  useEffect(() => {
    loadReceipts()
  }, [loadReceipts])

  async function handleSync() {
    setSyncing(true)
    setError(null)
    setSyncResult(null)
    try {
      const result = await etsyApi.syncReceipts()
      setSyncResult(result)
      await loadReceipts()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to sync orders')
    } finally {
      setSyncing(false)
    }
  }

  async function handleProcess(id: string) {
    setProcessingId(id)
    setError(null)
    try {
      await etsyApi.processReceipt(id)
      await loadReceipts()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to process order')
    } finally {
      setProcessingId(null)
    }
  }

  function formatCents(cents: number, currency: string = 'USD') {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency,
    }).format(cents / 100)
  }

  function formatDate(dateStr: string) {
    return new Date(dateStr).toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
    })
  }

  return (
    <div className="p-6">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-display font-semibold text-surface-100">
            Etsy Orders
          </h1>
          <p className="mt-1 text-sm text-surface-400">
            View and process orders from your Etsy shop
          </p>
        </div>
        <button
          onClick={handleSync}
          disabled={syncing}
          className="flex items-center gap-2 px-4 py-2 bg-accent-600 text-white rounded-lg hover:bg-accent-500 disabled:opacity-50"
        >
          <RefreshCw className={cn('h-4 w-4', syncing && 'animate-spin')} />
          {syncing ? 'Syncing...' : 'Sync Orders'}
        </button>
      </div>

      {/* Sync Result */}
      {syncResult && (
        <div className="mb-6 p-4 bg-green-500/10 border border-green-500/30 rounded-lg">
          <p className="text-green-400">
            Synced {syncResult.total_fetched} orders: {syncResult.created} new, {syncResult.updated} updated
            {syncResult.errors > 0 && `, ${syncResult.errors} errors`}
          </p>
        </div>
      )}

      {/* Error */}
      {error && (
        <div className="mb-6 p-4 bg-red-500/10 border border-red-500/30 rounded-lg">
          <p className="text-red-400">{error}</p>
        </div>
      )}

      {/* Filters */}
      <div className="mb-6 flex gap-2">
        {(['all', 'unprocessed', 'processed'] as const).map((f) => (
          <button
            key={f}
            onClick={() => setFilter(f)}
            className={cn(
              'px-3 py-1.5 text-sm rounded-lg transition-colors',
              filter === f
                ? 'bg-accent-600 text-white'
                : 'bg-surface-800 text-surface-300 hover:bg-surface-700'
            )}
          >
            {f.charAt(0).toUpperCase() + f.slice(1)}
          </button>
        ))}
      </div>

      {/* Orders List */}
      {loading ? (
        <div className="text-center py-12 text-surface-400">Loading orders...</div>
      ) : receipts.length === 0 ? (
        <div className="text-center py-12">
          <Package className="h-12 w-12 mx-auto text-surface-600 mb-3" />
          <p className="text-surface-400">No orders found</p>
          <button
            onClick={handleSync}
            className="mt-4 text-accent-400 hover:text-accent-300"
          >
            Sync orders from Etsy
          </button>
        </div>
      ) : (
        <div className="space-y-4">
          {receipts.map((receipt) => (
            <div
              key={receipt.id}
              className="bg-surface-900 border border-surface-800 rounded-lg p-4"
            >
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <div className="flex items-center gap-3">
                    <h3 className="text-lg font-medium text-surface-100">
                      {receipt.name}
                    </h3>
                    <span className="text-sm text-surface-500">
                      #{receipt.etsy_receipt_id}
                    </span>
                    {receipt.is_processed ? (
                      <span className="flex items-center gap-1 text-xs px-2 py-0.5 bg-green-500/20 text-green-400 rounded">
                        <CheckCircle className="h-3 w-3" />
                        Processed
                      </span>
                    ) : (
                      <span className="flex items-center gap-1 text-xs px-2 py-0.5 bg-yellow-500/20 text-yellow-400 rounded">
                        <Clock className="h-3 w-3" />
                        Pending
                      </span>
                    )}
                  </div>

                  <div className="mt-2 flex items-center gap-4 text-sm text-surface-400">
                    <span>{formatDate(receipt.create_timestamp || receipt.created_at)}</span>
                    <span className="font-medium text-surface-200">
                      {formatCents(receipt.grandtotal_cents, receipt.currency)}
                    </span>
                    {receipt.is_paid && (
                      <span className="text-green-400">Paid</span>
                    )}
                    {receipt.is_shipped && (
                      <span className="text-blue-400">Shipped</span>
                    )}
                    {receipt.is_gift && (
                      <span className="text-purple-400">Gift</span>
                    )}
                  </div>

                  {/* Shipping Address */}
                  {receipt.shipping_city && (
                    <div className="mt-2 text-sm text-surface-500">
                      {receipt.shipping_city}, {receipt.shipping_state} {receipt.shipping_zip} {receipt.shipping_country_code}
                    </div>
                  )}

                  {/* Message from Buyer */}
                  {receipt.message_from_buyer && (
                    <div className="mt-3 p-2 bg-surface-800/50 rounded text-sm text-surface-300">
                      <span className="text-surface-500">Note: </span>
                      {receipt.message_from_buyer}
                    </div>
                  )}

                  {/* Items */}
                  {receipt.items && receipt.items.length > 0 && (
                    <div className="mt-3 space-y-2">
                      {receipt.items.map((item) => (
                        <div
                          key={item.id}
                          className="flex items-center justify-between text-sm"
                        >
                          <div className="flex items-center gap-2">
                            <span className="text-surface-300">{item.title}</span>
                            {item.quantity > 1 && (
                              <span className="text-surface-500">x{item.quantity}</span>
                            )}
                            {item.sku && (
                              <span className="text-xs text-surface-600 font-mono">
                                SKU: {item.sku}
                              </span>
                            )}
                          </div>
                          <span className="text-surface-400">
                            {formatCents(item.price_cents * item.quantity, receipt.currency)}
                          </span>
                        </div>
                      ))}
                    </div>
                  )}
                </div>

                {/* Actions */}
                <div className="flex items-center gap-2 ml-4">
                  {!receipt.is_processed && (
                    <button
                      onClick={() => handleProcess(receipt.id)}
                      disabled={processingId === receipt.id}
                      className="px-3 py-1.5 text-sm bg-accent-600 text-white rounded hover:bg-accent-500 disabled:opacity-50"
                    >
                      {processingId === receipt.id ? 'Processing...' : 'Create Project'}
                    </button>
                  )}
                  {receipt.project_id && (
                    <a
                      href={`/projects/${receipt.project_id}`}
                      className="px-3 py-1.5 text-sm bg-surface-700 text-surface-200 rounded hover:bg-surface-600 flex items-center gap-1"
                    >
                      View Project
                      <ExternalLink className="h-3 w-3" />
                    </a>
                  )}
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
