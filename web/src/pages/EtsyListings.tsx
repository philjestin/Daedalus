import { useState, useEffect, useCallback } from 'react'
import { RefreshCw, Store, Link, Unlink, ExternalLink, Eye, Heart } from 'lucide-react'
import { etsyApi, templatesApi } from '../api/client'
import type { EtsyListing, Template, SyncResult } from '../types'
import { cn } from '../lib/utils'

export default function EtsyListings() {
  const [listings, setListings] = useState<EtsyListing[]>([])
  const [templates, setTemplates] = useState<Template[]>([])
  const [loading, setLoading] = useState(true)
  const [syncing, setSyncing] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [syncResult, setSyncResult] = useState<SyncResult | null>(null)
  const [stateFilter, setStateFilter] = useState<string>('active')
  const [linkingId, setLinkingId] = useState<string | null>(null)
  const [linkForm, setLinkForm] = useState<{
    listingId: string
    templateId: string
    sku: string
    syncInventory: boolean
  } | null>(null)

  const loadData = useCallback(async function() {
    setLoading(true)
    setError(null)
    try {
      const [listingsData, templatesData] = await Promise.all([
        etsyApi.listListings({ state: stateFilter || undefined }),
        templatesApi.list(true),
      ])
      setListings(listingsData)
      setTemplates(templatesData)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load data')
    } finally {
      setLoading(false)
    }
  }, [stateFilter])

  useEffect(() => {
    loadData()
  }, [loadData])

  async function handleSync() {
    setSyncing(true)
    setError(null)
    setSyncResult(null)
    try {
      const result = await etsyApi.syncListings()
      setSyncResult(result)
      await loadData()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to sync listings')
    } finally {
      setSyncing(false)
    }
  }

  async function handleLink(listingId: string) {
    if (!linkForm || linkForm.listingId !== listingId) return

    setLinkingId(listingId)
    setError(null)
    try {
      await etsyApi.linkListing(listingId, {
        template_id: linkForm.templateId,
        sku: linkForm.sku || undefined,
        sync_inventory: linkForm.syncInventory,
      })
      setLinkForm(null)
      await loadData()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to link listing')
    } finally {
      setLinkingId(null)
    }
  }

  async function handleUnlink(listingId: string, templateId: string) {
    setError(null)
    try {
      await etsyApi.unlinkListing(listingId, templateId)
      await loadData()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to unlink listing')
    }
  }

  function formatCents(cents: number | undefined, currency: string = 'USD') {
    if (cents === undefined) return '-'
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency,
    }).format(cents / 100)
  }

  function getStateColor(state: string) {
    switch (state) {
      case 'active':
        return 'text-green-400 bg-green-500/20'
      case 'inactive':
        return 'text-yellow-400 bg-yellow-500/20'
      case 'draft':
        return 'text-blue-400 bg-blue-500/20'
      case 'expired':
        return 'text-red-400 bg-red-500/20'
      case 'sold_out':
        return 'text-orange-400 bg-orange-500/20'
      default:
        return 'text-surface-400 bg-surface-700'
    }
  }

  return (
    <div className="p-6">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-display font-semibold text-surface-100">
            Etsy Listings
          </h1>
          <p className="mt-1 text-sm text-surface-400">
            View listings and link them to templates
          </p>
        </div>
        <button
          onClick={handleSync}
          disabled={syncing}
          className="flex items-center gap-2 px-4 py-2 bg-accent-600 text-white rounded-lg hover:bg-accent-500 disabled:opacity-50"
        >
          <RefreshCw className={cn('h-4 w-4', syncing && 'animate-spin')} />
          {syncing ? 'Syncing...' : 'Sync Listings'}
        </button>
      </div>

      {/* Sync Result */}
      {syncResult && (
        <div className="mb-6 p-4 bg-green-500/10 border border-green-500/30 rounded-lg">
          <p className="text-green-400">
            Synced {syncResult.total_fetched} listings: {syncResult.created} new, {syncResult.updated} updated
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
        {['active', 'inactive', 'draft', 'expired', 'sold_out', ''].map((state) => (
          <button
            key={state || 'all'}
            onClick={() => setStateFilter(state)}
            className={cn(
              'px-3 py-1.5 text-sm rounded-lg transition-colors',
              stateFilter === state
                ? 'bg-accent-600 text-white'
                : 'bg-surface-800 text-surface-300 hover:bg-surface-700'
            )}
          >
            {state ? state.replace('_', ' ').charAt(0).toUpperCase() + state.slice(1).replace('_', ' ') : 'All'}
          </button>
        ))}
      </div>

      {/* Listings Grid */}
      {loading ? (
        <div className="text-center py-12 text-surface-400">Loading listings...</div>
      ) : listings.length === 0 ? (
        <div className="text-center py-12">
          <Store className="h-12 w-12 mx-auto text-surface-600 mb-3" />
          <p className="text-surface-400">No listings found</p>
          <button
            onClick={handleSync}
            className="mt-4 text-accent-400 hover:text-accent-300"
          >
            Sync listings from Etsy
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {listings.map((listing) => (
            <div
              key={listing.id}
              className="bg-surface-900 border border-surface-800 rounded-lg p-4"
            >
              <div className="flex items-start justify-between">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <h3 className="text-lg font-medium text-surface-100 truncate">
                      {listing.title}
                    </h3>
                  </div>

                  <div className="mt-1 flex items-center gap-3 text-sm">
                    <span className={cn('px-2 py-0.5 rounded text-xs', getStateColor(listing.state))}>
                      {listing.state}
                    </span>
                    <span className="font-medium text-surface-200">
                      {formatCents(listing.price_cents, listing.currency)}
                    </span>
                    <span className="text-surface-500">
                      Qty: {listing.quantity}
                    </span>
                  </div>

                  <div className="mt-2 flex items-center gap-4 text-sm text-surface-500">
                    <span className="flex items-center gap-1">
                      <Eye className="h-3.5 w-3.5" />
                      {listing.views}
                    </span>
                    <span className="flex items-center gap-1">
                      <Heart className="h-3.5 w-3.5" />
                      {listing.num_favorers}
                    </span>
                    {listing.has_variations && (
                      <span className="text-accent-400">Has Variations</span>
                    )}
                  </div>

                  {/* SKUs */}
                  {listing.skus && listing.skus.length > 0 && (
                    <div className="mt-2 flex flex-wrap gap-1">
                      {listing.skus.map((sku, i) => (
                        <span
                          key={i}
                          className="text-xs px-1.5 py-0.5 bg-surface-800 text-surface-400 rounded font-mono"
                        >
                          {sku}
                        </span>
                      ))}
                    </div>
                  )}

                  {/* Linked Template */}
                  {listing.linked_template && (
                    <div className="mt-3 flex items-center gap-2 text-sm">
                      <Link className="h-4 w-4 text-green-400" />
                      <span className="text-surface-300">
                        Linked to: {listing.linked_template.name}
                      </span>
                      <button
                        onClick={() => handleUnlink(listing.id, listing.linked_template!.id)}
                        className="text-red-400 hover:text-red-300"
                      >
                        <Unlink className="h-4 w-4" />
                      </button>
                    </div>
                  )}

                  {/* Link Form */}
                  {linkForm?.listingId === listing.id ? (
                    <div className="mt-3 p-3 bg-surface-800 rounded space-y-3">
                      <div>
                        <label className="block text-xs text-surface-500 mb-1">Template</label>
                        <select
                          value={linkForm.templateId}
                          onChange={(e) => setLinkForm({ ...linkForm, templateId: e.target.value })}
                          className="w-full px-2 py-1.5 bg-surface-700 border border-surface-600 rounded text-sm text-surface-200"
                        >
                          <option value="">Select a template...</option>
                          {templates.map((t) => (
                            <option key={t.id} value={t.id}>
                              {t.name} {t.sku && `(${t.sku})`}
                            </option>
                          ))}
                        </select>
                      </div>
                      <div>
                        <label className="block text-xs text-surface-500 mb-1">SKU (optional)</label>
                        <input
                          type="text"
                          value={linkForm.sku}
                          onChange={(e) => setLinkForm({ ...linkForm, sku: e.target.value })}
                          placeholder="SKU for matching"
                          className="w-full px-2 py-1.5 bg-surface-700 border border-surface-600 rounded text-sm text-surface-200"
                        />
                      </div>
                      <div className="flex items-center gap-2">
                        <input
                          type="checkbox"
                          id={`sync-inv-${listing.id}`}
                          checked={linkForm.syncInventory}
                          onChange={(e) => setLinkForm({ ...linkForm, syncInventory: e.target.checked })}
                          className="rounded"
                        />
                        <label htmlFor={`sync-inv-${listing.id}`} className="text-sm text-surface-300">
                          Sync inventory
                        </label>
                      </div>
                      <div className="flex gap-2">
                        <button
                          onClick={() => handleLink(listing.id)}
                          disabled={!linkForm.templateId || linkingId === listing.id}
                          className="px-3 py-1.5 text-sm bg-accent-600 text-white rounded hover:bg-accent-500 disabled:opacity-50"
                        >
                          {linkingId === listing.id ? 'Linking...' : 'Link'}
                        </button>
                        <button
                          onClick={() => setLinkForm(null)}
                          className="px-3 py-1.5 text-sm bg-surface-700 text-surface-300 rounded hover:bg-surface-600"
                        >
                          Cancel
                        </button>
                      </div>
                    </div>
                  ) : !listing.linked_template && (
                    <button
                      onClick={() => setLinkForm({
                        listingId: listing.id,
                        templateId: '',
                        sku: '',
                        syncInventory: true,
                      })}
                      className="mt-3 flex items-center gap-1 text-sm text-accent-400 hover:text-accent-300"
                    >
                      <Link className="h-4 w-4" />
                      Link to Template
                    </button>
                  )}
                </div>

                {/* External Link */}
                {listing.url && (
                  <a
                    href={listing.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="ml-2 p-2 text-surface-500 hover:text-surface-300"
                  >
                    <ExternalLink className="h-4 w-4" />
                  </a>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
