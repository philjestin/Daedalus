import { useState, useEffect, useCallback } from 'react'
import { Link as RouterLink } from 'react-router-dom'
import { RefreshCw, Package, CheckCircle, Clock, ExternalLink, ShoppingBag, Store, Link, Filter } from 'lucide-react'
import { etsyApi, squarespaceApi, templatesApi } from '../api/client'
import type { EtsyReceipt, EtsyListing, SquarespaceOrder, SquarespaceProduct, SyncResult, Template, EtsyIntegration, SquarespaceIntegration } from '../types'
import { cn } from '../lib/utils'

type Tab = 'orders' | 'products'
type Channel = 'all' | 'etsy' | 'squarespace'
type OrderFilter = 'all' | 'unprocessed' | 'processed'

// Unified order type for display
interface UnifiedOrder {
  id: string
  channel: 'etsy' | 'squarespace'
  orderNumber: string
  customerName: string
  customerEmail?: string
  totalCents: number
  currency: string
  isProcessed: boolean
  projectId?: string
  createdAt: string
  status?: string
  items: Array<{
    id: string
    name: string
    quantity: number
    priceCents: number
    sku?: string
  }>
  raw: EtsyReceipt | SquarespaceOrder
}

// Unified product type for display
interface UnifiedProduct {
  id: string
  channel: 'etsy' | 'squarespace'
  name: string
  description?: string
  type?: string
  isVisible: boolean
  skus: string[]
  priceCents?: number
  linkedTemplateId?: string
  raw: EtsyListing | SquarespaceProduct
}

export default function Channels() {
  const [tab, setTab] = useState<Tab>('orders')
  const [channel, setChannel] = useState<Channel>('all')
  const [orderFilter, setOrderFilter] = useState<OrderFilter>('all')

  // Connection status
  const [etsyStatus, setEtsyStatus] = useState<EtsyIntegration | null>(null)
  const [squarespaceStatus, setSquarespaceStatus] = useState<SquarespaceIntegration | null>(null)

  // Data
  const [orders, setOrders] = useState<UnifiedOrder[]>([])
  const [products, setProducts] = useState<UnifiedProduct[]>([])
  const [templates, setTemplates] = useState<Template[]>([])

  // UI state
  const [loading, setLoading] = useState(true)
  const [syncing, setSyncing] = useState<Channel | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [syncResult, setSyncResult] = useState<{ channel: string; result: SyncResult } | null>(null)
  const [processingId, setProcessingId] = useState<string | null>(null)
  const [linkingId, setLinkingId] = useState<string | null>(null)
  const [selectedTemplate, setSelectedTemplate] = useState<Record<string, string>>({})

  // Load connection status
  useEffect(() => {
    async function loadStatus() {
      try {
        const [etsy, squarespace] = await Promise.all([
          etsyApi.getStatus().catch(() => null),
          squarespaceApi.getStatus().catch(() => null),
        ])
        setEtsyStatus(etsy)
        setSquarespaceStatus(squarespace)
      } catch (err) {
        console.error('Failed to load channel status:', err)
      }
    }
    loadStatus()
  }, [])

  // Load orders
  const loadOrders = useCallback(async () => {
    setLoading(true)
    setError(null)
    const unified: UnifiedOrder[] = []

    try {
      const processed = orderFilter === 'all' ? undefined : orderFilter === 'processed'

      // Load Etsy orders if connected and channel matches
      if ((channel === 'all' || channel === 'etsy') && etsyStatus?.connected) {
        try {
          const etsyOrders = await etsyApi.listReceipts({ processed })
          for (const r of etsyOrders) {
            unified.push({
              id: r.id,
              channel: 'etsy',
              orderNumber: String(r.etsy_receipt_id),
              customerName: r.name,
              customerEmail: r.buyer_email,
              totalCents: r.grandtotal_cents,
              currency: r.currency,
              isProcessed: r.is_processed,
              projectId: r.project_id,
              createdAt: r.create_timestamp || r.created_at,
              status: r.status,
              items: (r.items || []).map(i => ({
                id: i.id,
                name: i.title,
                quantity: i.quantity,
                priceCents: i.price_cents,
                sku: i.sku,
              })),
              raw: r,
            })
          }
        } catch (err) {
          console.error('Failed to load Etsy orders:', err)
        }
      }

      // Load Squarespace orders if connected and channel matches
      if ((channel === 'all' || channel === 'squarespace') && squarespaceStatus?.connected) {
        try {
          const sqOrders = await squarespaceApi.listOrders({ processed })
          for (const o of sqOrders) {
            unified.push({
              id: o.id,
              channel: 'squarespace',
              orderNumber: o.order_number,
              customerName: o.customer_name || o.customer_email,
              customerEmail: o.customer_email,
              totalCents: o.grand_total_cents,
              currency: o.currency,
              isProcessed: o.is_processed,
              projectId: o.project_id,
              createdAt: o.created_on || o.created_at,
              status: o.fulfillment_status,
              items: (o.items || []).map(i => ({
                id: i.id,
                name: i.product_name,
                quantity: i.quantity,
                priceCents: i.unit_price_cents,
                sku: i.sku,
              })),
              raw: o,
            })
          }
        } catch (err) {
          console.error('Failed to load Squarespace orders:', err)
        }
      }

      // Sort by date descending
      unified.sort((a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime())
      setOrders(unified)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load orders')
    } finally {
      setLoading(false)
    }
  }, [channel, orderFilter, etsyStatus?.connected, squarespaceStatus?.connected])

  // Load products
  const loadProducts = useCallback(async () => {
    setLoading(true)
    setError(null)
    const unified: UnifiedProduct[] = []

    try {
      // Load templates for linking
      const tpls = await templatesApi.list(true)
      setTemplates(tpls)

      // Load Etsy listings if connected and channel matches
      if ((channel === 'all' || channel === 'etsy') && etsyStatus?.connected) {
        try {
          const listings = await etsyApi.listListings()
          for (const l of listings) {
            unified.push({
              id: l.id,
              channel: 'etsy',
              name: l.title,
              description: l.description,
              type: l.state,
              isVisible: l.state === 'active',
              skus: l.skus || [],
              priceCents: l.price_cents,
              linkedTemplateId: l.linked_template?.id,
              raw: l,
            })
          }
        } catch (err) {
          console.error('Failed to load Etsy listings:', err)
        }
      }

      // Load Squarespace products if connected and channel matches
      if ((channel === 'all' || channel === 'squarespace') && squarespaceStatus?.connected) {
        try {
          const prods = await squarespaceApi.listProducts()
          for (const p of prods) {
            unified.push({
              id: p.id,
              channel: 'squarespace',
              name: p.name,
              description: p.description,
              type: p.type,
              isVisible: p.is_visible,
              skus: (p.variants || []).map(v => v.sku).filter(Boolean),
              priceCents: p.variants?.[0]?.price_cents,
              raw: p,
            })
          }
        } catch (err) {
          console.error('Failed to load Squarespace products:', err)
        }
      }

      // Sort by name
      unified.sort((a, b) => a.name.localeCompare(b.name))
      setProducts(unified)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load products')
    } finally {
      setLoading(false)
    }
  }, [channel, etsyStatus?.connected, squarespaceStatus?.connected])

  // Load data when tab or filters change
  useEffect(() => {
    if (tab === 'orders') {
      loadOrders()
    } else {
      loadProducts()
    }
  }, [tab, loadOrders, loadProducts])

  // Sync handlers
  async function handleSync(syncChannel: Channel) {
    if (syncChannel === 'all') return
    setSyncing(syncChannel)
    setError(null)
    setSyncResult(null)

    try {
      let result: SyncResult
      if (tab === 'orders') {
        if (syncChannel === 'etsy') {
          result = await etsyApi.syncReceipts()
        } else {
          result = await squarespaceApi.syncOrders()
        }
      } else {
        if (syncChannel === 'etsy') {
          result = await etsyApi.syncListings()
        } else {
          result = await squarespaceApi.syncProducts()
        }
      }
      setSyncResult({ channel: syncChannel, result })
      if (tab === 'orders') {
        await loadOrders()
      } else {
        await loadProducts()
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : `Failed to sync ${syncChannel}`)
    } finally {
      setSyncing(null)
    }
  }

  async function handleSyncAll() {
    setSyncing('all')
    setError(null)
    setSyncResult(null)

    try {
      const results: SyncResult = { total_fetched: 0, created: 0, updated: 0, skipped: 0, errors: 0 }

      if (tab === 'orders') {
        if (etsyStatus?.connected) {
          const r = await etsyApi.syncReceipts()
          results.total_fetched += r.total_fetched
          results.created += r.created
          results.updated += r.updated
          results.errors += r.errors
        }
        if (squarespaceStatus?.connected) {
          const r = await squarespaceApi.syncOrders()
          results.total_fetched += r.total_fetched
          results.created += r.created
          results.updated += r.updated
          results.errors += r.errors
        }
      } else {
        if (etsyStatus?.connected) {
          const r = await etsyApi.syncListings()
          results.total_fetched += r.total_fetched
          results.created += r.created
          results.updated += r.updated
          results.errors += r.errors
        }
        if (squarespaceStatus?.connected) {
          const r = await squarespaceApi.syncProducts()
          results.total_fetched += r.total_fetched
          results.created += r.created
          results.updated += r.updated
          results.errors += r.errors
        }
      }

      setSyncResult({ channel: 'all', result: results })
      if (tab === 'orders') {
        await loadOrders()
      } else {
        await loadProducts()
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to sync')
    } finally {
      setSyncing(null)
    }
  }

  // Process order
  async function handleProcess(order: UnifiedOrder) {
    setProcessingId(order.id)
    setError(null)
    try {
      if (order.channel === 'etsy') {
        await etsyApi.processReceipt(order.id)
      } else {
        await squarespaceApi.processOrder(order.id)
      }
      await loadOrders()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to process order')
    } finally {
      setProcessingId(null)
    }
  }

  // Link product to template
  async function handleLink(product: UnifiedProduct) {
    const templateId = selectedTemplate[product.id]
    if (!templateId) {
      setError('Please select a template')
      return
    }

    setLinkingId(product.id)
    setError(null)
    try {
      const sku = product.skus[0] || ''
      if (product.channel === 'etsy') {
        await etsyApi.linkListing(product.id, { template_id: templateId, sku })
      } else {
        await squarespaceApi.linkProduct(product.id, templateId, sku)
      }
      await loadProducts()
      setSelectedTemplate(prev => ({ ...prev, [product.id]: '' }))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to link product')
    } finally {
      setLinkingId(null)
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

  const hasConnectedChannel = etsyStatus?.connected || squarespaceStatus?.connected

  return (
    <div className="p-6">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-display font-semibold text-surface-100">
            Sales Channels
          </h1>
          <p className="mt-1 text-sm text-surface-400">
            Manage orders and products from connected marketplaces
          </p>
        </div>

        {/* Connection Status Pills */}
        <div className="flex items-center gap-2">
          {etsyStatus?.connected && (
            <div className="flex items-center gap-1.5 px-2 py-1 bg-orange-500/10 border border-orange-500/30 rounded-lg">
              <Store className="h-3.5 w-3.5 text-orange-400" />
              <span className="text-xs text-orange-300">{etsyStatus.shop_name}</span>
            </div>
          )}
          {squarespaceStatus?.connected && (
            <div className="flex items-center gap-1.5 px-2 py-1 bg-purple-500/10 border border-purple-500/30 rounded-lg">
              <ShoppingBag className="h-3.5 w-3.5 text-purple-400" />
              <span className="text-xs text-purple-300">{squarespaceStatus.site_title}</span>
            </div>
          )}
          {!hasConnectedChannel && (
            <RouterLink
              to="/settings"
              className="text-sm text-accent-400 hover:text-accent-300"
            >
              Connect a channel in Settings
            </RouterLink>
          )}
        </div>
      </div>

      {/* Tabs */}
      <div className="flex items-center gap-4 mb-6 border-b border-surface-800">
        <button
          onClick={() => setTab('orders')}
          className={cn(
            'pb-3 text-sm font-medium border-b-2 transition-colors',
            tab === 'orders'
              ? 'border-accent-500 text-accent-400'
              : 'border-transparent text-surface-400 hover:text-surface-200'
          )}
        >
          Orders
        </button>
        <button
          onClick={() => setTab('products')}
          className={cn(
            'pb-3 text-sm font-medium border-b-2 transition-colors',
            tab === 'products'
              ? 'border-accent-500 text-accent-400'
              : 'border-transparent text-surface-400 hover:text-surface-200'
          )}
        >
          Products / Listings
        </button>
      </div>

      {/* Filters and Actions */}
      <div className="flex items-center justify-between mb-6">
        <div className="flex items-center gap-3">
          {/* Channel Filter */}
          <div className="flex items-center gap-2">
            <Filter className="h-4 w-4 text-surface-500" />
            <select
              value={channel}
              onChange={(e) => setChannel(e.target.value as Channel)}
              className="bg-surface-800 border border-surface-700 rounded-lg px-3 py-1.5 text-sm text-surface-200"
            >
              <option value="all">All Channels</option>
              {etsyStatus?.connected && <option value="etsy">Etsy</option>}
              {squarespaceStatus?.connected && <option value="squarespace">Squarespace</option>}
            </select>
          </div>

          {/* Order Status Filter (orders tab only) */}
          {tab === 'orders' && (
            <div className="flex gap-1">
              {(['all', 'unprocessed', 'processed'] as const).map((f) => (
                <button
                  key={f}
                  onClick={() => setOrderFilter(f)}
                  className={cn(
                    'px-3 py-1.5 text-xs rounded-lg transition-colors',
                    orderFilter === f
                      ? 'bg-accent-600 text-white'
                      : 'bg-surface-800 text-surface-300 hover:bg-surface-700'
                  )}
                >
                  {f.charAt(0).toUpperCase() + f.slice(1)}
                </button>
              ))}
            </div>
          )}
        </div>

        {/* Sync Buttons */}
        <div className="flex items-center gap-2">
          {channel === 'all' ? (
            <button
              onClick={handleSyncAll}
              disabled={syncing !== null || !hasConnectedChannel}
              className="flex items-center gap-2 px-4 py-2 bg-accent-600 text-white rounded-lg hover:bg-accent-500 disabled:opacity-50 text-sm"
            >
              <RefreshCw className={cn('h-4 w-4', syncing && 'animate-spin')} />
              {syncing ? 'Syncing...' : 'Sync All'}
            </button>
          ) : (
            <button
              onClick={() => handleSync(channel)}
              disabled={syncing !== null}
              className={cn(
                'flex items-center gap-2 px-4 py-2 text-white rounded-lg disabled:opacity-50 text-sm',
                channel === 'etsy' ? 'bg-orange-500 hover:bg-orange-600' : 'bg-purple-500 hover:bg-purple-600'
              )}
            >
              <RefreshCw className={cn('h-4 w-4', syncing === channel && 'animate-spin')} />
              {syncing === channel ? 'Syncing...' : `Sync ${channel === 'etsy' ? 'Etsy' : 'Squarespace'}`}
            </button>
          )}
        </div>
      </div>

      {/* Sync Result */}
      {syncResult && (
        <div className="mb-6 p-4 bg-green-500/10 border border-green-500/30 rounded-lg">
          <p className="text-green-400">
            Synced {syncResult.result.total_fetched} {tab}: {syncResult.result.created} new, {syncResult.result.updated} updated
            {syncResult.result.errors > 0 && `, ${syncResult.result.errors} errors`}
          </p>
        </div>
      )}

      {/* Error */}
      {error && (
        <div className="mb-6 p-4 bg-red-500/10 border border-red-500/30 rounded-lg">
          <p className="text-red-400">{error}</p>
        </div>
      )}

      {/* Content */}
      {!hasConnectedChannel ? (
        <div className="text-center py-12">
          <Package className="h-12 w-12 mx-auto text-surface-600 mb-3" />
          <p className="text-surface-400">No sales channels connected</p>
          <RouterLink
            to="/settings"
            className="mt-4 inline-block text-accent-400 hover:text-accent-300"
          >
            Connect Etsy or Squarespace in Settings
          </RouterLink>
        </div>
      ) : loading ? (
        <div className="text-center py-12 text-surface-400">Loading...</div>
      ) : tab === 'orders' ? (
        /* Orders List */
        orders.length === 0 ? (
          <div className="text-center py-12">
            <Package className="h-12 w-12 mx-auto text-surface-600 mb-3" />
            <p className="text-surface-400">No orders found</p>
          </div>
        ) : (
          <div className="space-y-4">
            {orders.map((order) => (
              <div
                key={`${order.channel}-${order.id}`}
                className="bg-surface-900 border border-surface-800 rounded-lg p-4"
              >
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center gap-3">
                      {/* Channel Icon */}
                      {order.channel === 'etsy' ? (
                        <Store className="h-4 w-4 text-orange-400" />
                      ) : (
                        <ShoppingBag className="h-4 w-4 text-purple-400" />
                      )}
                      <h3 className="text-lg font-medium text-surface-100">
                        {order.customerName}
                      </h3>
                      <span className="text-sm text-surface-500">
                        #{order.orderNumber}
                      </span>
                      {order.isProcessed ? (
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
                      {order.status && (
                        <span className="text-xs px-2 py-0.5 bg-surface-700 text-surface-300 rounded">
                          {order.status}
                        </span>
                      )}
                    </div>

                    <div className="mt-2 flex items-center gap-4 text-sm text-surface-400">
                      <span>{formatDate(order.createdAt)}</span>
                      <span className="font-medium text-surface-200">
                        {formatCents(order.totalCents, order.currency)}
                      </span>
                    </div>

                    {/* Items */}
                    {order.items.length > 0 && (
                      <div className="mt-3 space-y-1">
                        {order.items.map((item) => (
                          <div key={item.id} className="flex items-center justify-between text-sm">
                            <div className="flex items-center gap-2">
                              <span className="text-surface-300">{item.name}</span>
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
                              {formatCents(item.priceCents * item.quantity, order.currency)}
                            </span>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>

                  {/* Actions */}
                  <div className="flex items-center gap-2 ml-4">
                    {!order.isProcessed && (
                      <button
                        onClick={() => handleProcess(order)}
                        disabled={processingId === order.id}
                        className="px-3 py-1.5 text-sm bg-accent-600 text-white rounded hover:bg-accent-500 disabled:opacity-50"
                      >
                        {processingId === order.id ? 'Processing...' : 'Create Project'}
                      </button>
                    )}
                    {order.projectId && (
                      <a
                        href={`/projects/${order.projectId}`}
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
        )
      ) : (
        /* Products List */
        products.length === 0 ? (
          <div className="text-center py-12">
            <Package className="h-12 w-12 mx-auto text-surface-600 mb-3" />
            <p className="text-surface-400">No products found</p>
          </div>
        ) : (
          <div className="space-y-4">
            {products.map((product) => (
              <div
                key={`${product.channel}-${product.id}`}
                className="bg-surface-900 border border-surface-800 rounded-lg p-4"
              >
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <div className="flex items-center gap-3">
                      {/* Channel Icon */}
                      {product.channel === 'etsy' ? (
                        <Store className="h-4 w-4 text-orange-400" />
                      ) : (
                        <ShoppingBag className="h-4 w-4 text-purple-400" />
                      )}
                      <h3 className="text-lg font-medium text-surface-100">
                        {product.name}
                      </h3>
                      {product.type && (
                        <span className="text-xs px-2 py-0.5 bg-surface-700 text-surface-300 rounded">
                          {product.type}
                        </span>
                      )}
                      {!product.isVisible && (
                        <span className="text-xs px-2 py-0.5 bg-yellow-500/20 text-yellow-400 rounded">
                          Hidden
                        </span>
                      )}
                      {product.linkedTemplateId && (
                        <span className="flex items-center gap-1 text-xs px-2 py-0.5 bg-green-500/20 text-green-400 rounded">
                          <Link className="h-3 w-3" />
                          Linked
                        </span>
                      )}
                    </div>

                    {product.description && (
                      <p className="mt-1 text-sm text-surface-500 line-clamp-1">
                        {product.description}
                      </p>
                    )}

                    <div className="mt-2 flex items-center gap-4 text-sm text-surface-400">
                      {product.priceCents !== undefined && (
                        <span className="font-medium text-surface-200">
                          {formatCents(product.priceCents)}
                        </span>
                      )}
                      {product.skus.length > 0 && (
                        <span className="font-mono text-xs">
                          SKU: {product.skus.slice(0, 3).join(', ')}{product.skus.length > 3 && '...'}
                        </span>
                      )}
                    </div>
                  </div>

                  {/* Link to Template */}
                  <div className="flex items-center gap-2 ml-4">
                    <select
                      value={selectedTemplate[product.id] || ''}
                      onChange={(e) => setSelectedTemplate(prev => ({ ...prev, [product.id]: e.target.value }))}
                      className="bg-surface-800 border border-surface-700 rounded px-2 py-1 text-sm text-surface-200"
                    >
                      <option value="">Select template...</option>
                      {templates.map((template) => (
                        <option key={template.id} value={template.id}>
                          {template.name} {template.sku && `(${template.sku})`}
                        </option>
                      ))}
                    </select>
                    <button
                      onClick={() => handleLink(product)}
                      disabled={linkingId === product.id || !selectedTemplate[product.id]}
                      className="flex items-center gap-1 px-3 py-1.5 text-sm bg-accent-600 text-white rounded hover:bg-accent-500 disabled:opacity-50"
                    >
                      {linkingId === product.id ? (
                        <RefreshCw className="h-3 w-3 animate-spin" />
                      ) : (
                        <Link className="h-3 w-3" />
                      )}
                      Link
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )
      )}
    </div>
  )
}
