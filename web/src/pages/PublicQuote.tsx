import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { publicApi } from '../api/client'
import type { Quote, QuoteOption, QuoteLineItem, QuoteLineItemType, Address } from '../types'

// ── Helpers ────────────────────────────────────────────────

function formatCents(cents: number): string {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD' }).format(cents / 100)
}

function calculateFinancials(subtotalCents: number, quote: Quote) {
  const discountAmount = quote.discount_type === 'flat'
    ? quote.discount_value
    : quote.discount_type === 'percent'
      ? Math.round(subtotalCents * quote.discount_value / 10000)
      : 0
  const afterDiscount = subtotalCents - discountAmount
  const withRush = afterDiscount + quote.rush_fee_cents
  const taxAmount = Math.round(withRush * quote.tax_rate / 10000)
  const grandTotal = withRush + taxAmount
  return { discountAmount, afterDiscount, withRush, taxAmount, grandTotal }
}

function formatDate(dateStr?: string): string {
  if (!dateStr) return ''
  return new Date(dateStr).toLocaleDateString('en-US', { year: 'numeric', month: 'long', day: 'numeric' })
}

function formatAddressLines(addr?: Address): string[] {
  if (!addr) return []
  const lines: string[] = []
  if (addr.line1) lines.push(addr.line1)
  if (addr.line2) lines.push(addr.line2)
  const cityState = [addr.city, addr.state].filter(Boolean).join(', ')
  if (cityState || addr.zip) lines.push([cityState, addr.zip].filter(Boolean).join(' '))
  if (addr.country) lines.push(addr.country)
  return lines
}

const typeOrder: QuoteLineItemType[] = [
  'printing', 'post_processing', 'finishing', 'labor', 'consumables', 'design', 'consulting', 'shipping', 'other',
]

const typeLabels: Record<QuoteLineItemType, string> = {
  printing: 'Printing',
  post_processing: 'Post Processing',
  finishing: 'Finishing',
  labor: 'Labor',
  consumables: 'Consumables',
  design: 'Design',
  consulting: 'Consulting',
  shipping: 'Shipping',
  other: 'Other',
}

interface BusinessInfo {
  business_name?: string
  business_phone?: string
  business_email?: string
  business_website?: string
  business_address?: Address
}

// ── Component ──────────────────────────────────────────────

export default function PublicQuote() {
  const { token } = useParams<{ token: string }>()
  const [quote, setQuote] = useState<Quote | null>(null)
  const [business, setBusiness] = useState<BusinessInfo>({})
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!token) return
    Promise.all([
      publicApi.getQuote(token),
      publicApi.getBusinessInfo().catch(() => ({})),
    ]).then(([q, b]) => {
      setQuote(q)
      // Parse address JSON if present
      const info: BusinessInfo = {
        business_name: (b as Record<string, string>).business_name,
        business_phone: (b as Record<string, string>).business_phone,
        business_email: (b as Record<string, string>).business_email,
        business_website: (b as Record<string, string>).business_website,
      }
      const addrJson = (b as Record<string, string>).business_address_json
      if (addrJson) {
        try { info.business_address = JSON.parse(addrJson) } catch {}
      }
      setBusiness(info)
    }).catch((err) => {
      setError(err.message || 'Quote not found')
    }).finally(() => {
      setLoading(false)
    })
  }, [token])

  if (loading) {
    return (
      <div className="min-h-screen bg-white flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-gray-800" />
      </div>
    )
  }

  if (error || !quote) {
    return (
      <div className="min-h-screen bg-white flex items-center justify-center">
        <div className="text-center">
          <h1 className="text-2xl font-bold text-gray-800 mb-2">Quote Not Found</h1>
          <p className="text-gray-500">{error || 'This quote link may be invalid or expired.'}</p>
        </div>
      </div>
    )
  }

  const hasAdjustments = quote.discount_type !== 'none' || quote.rush_fee_cents > 0 || quote.tax_rate > 0
  const businessAddrLines = formatAddressLines(business.business_address)
  const billingLines = formatAddressLines(quote.billing_address)
  const shippingLines = formatAddressLines(quote.shipping_address)

  return (
    <div className="min-h-screen bg-gray-50 print:bg-white">
      <div className="max-w-4xl mx-auto bg-white shadow-sm print:shadow-none">
        {/* Print Button */}
        <div className="no-print px-8 py-4 border-b border-gray-100 flex justify-end">
          <button
            onClick={() => window.print()}
            className="px-4 py-2 bg-gray-800 text-white rounded-lg text-sm font-medium hover:bg-gray-700"
          >
            Print Quote
          </button>
        </div>

        <div className="px-8 py-8">
          {/* Business Header */}
          <div className="flex justify-between items-start mb-8 pb-6 border-b border-gray-200">
            <div>
              <h1 className="text-2xl font-bold text-gray-900">
                {business.business_name || 'Quote'}
              </h1>
              {businessAddrLines.length > 0 && (
                <div className="text-sm text-gray-500 mt-1">
                  {businessAddrLines.map((line, i) => <div key={i}>{line}</div>)}
                </div>
              )}
            </div>
            <div className="text-right text-sm text-gray-500">
              {business.business_phone && <div>{business.business_phone}</div>}
              {business.business_email && <div>{business.business_email}</div>}
              {business.business_website && <div>{business.business_website}</div>}
            </div>
          </div>

          {/* Quote Info */}
          <div className="flex justify-between items-start mb-6">
            <div>
              <h2 className="text-xl font-semibold text-gray-900">
                Quote <span className="font-mono">{quote.quote_number}</span>
              </h2>
              <div className="text-sm text-gray-500 mt-1 space-y-0.5">
                <div>Date: {formatDate(quote.created_at)}</div>
                {quote.valid_until && <div>Valid Until: {formatDate(quote.valid_until)}</div>}
                {quote.requested_due_date && <div>Requested Due Date: {formatDate(quote.requested_due_date)}</div>}
              </div>
            </div>
            <div className="px-3 py-1 bg-gray-100 rounded-full text-xs font-medium text-gray-600 uppercase">
              {quote.status}
            </div>
          </div>

          {/* Addresses */}
          {(billingLines.length > 0 || shippingLines.length > 0) && (
            <div className="grid grid-cols-2 gap-8 mb-8 pb-6 border-b border-gray-200">
              <div>
                <h3 className="text-xs font-semibold text-gray-400 uppercase mb-1">Bill To</h3>
                {quote.customer && (
                  <div className="text-sm font-medium text-gray-900">
                    {quote.customer.name}
                    {quote.customer.company && <span className="text-gray-500"> / {quote.customer.company}</span>}
                  </div>
                )}
                <div className="text-sm text-gray-600">
                  {billingLines.map((line, i) => <div key={i}>{line}</div>)}
                </div>
              </div>
              <div>
                <h3 className="text-xs font-semibold text-gray-400 uppercase mb-1">Ship To</h3>
                {quote.customer && (
                  <div className="text-sm font-medium text-gray-900">
                    {quote.customer.name}
                    {quote.customer.company && <span className="text-gray-500"> / {quote.customer.company}</span>}
                  </div>
                )}
                <div className="text-sm text-gray-600">
                  {shippingLines.map((line, i) => <div key={i}>{line}</div>)}
                </div>
              </div>
            </div>
          )}

          {/* Title & Notes */}
          {(quote.title || quote.notes) && (
            <div className="mb-6 pb-6 border-b border-gray-200">
              {quote.title && <h3 className="text-lg font-semibold text-gray-900 mb-1">{quote.title}</h3>}
              {quote.notes && <p className="text-sm text-gray-600">{quote.notes}</p>}
            </div>
          )}

          {/* Options */}
          {quote.options && quote.options.map((option) => (
            <OptionSection key={option.id} option={option} quote={quote} hasAdjustments={hasAdjustments} />
          ))}

          {/* Terms */}
          {quote.terms && (
            <div className="mt-8 pt-6 border-t border-gray-200">
              <h3 className="text-sm font-semibold text-gray-900 uppercase mb-2">Terms & Conditions</h3>
              <p className="text-sm text-gray-600 whitespace-pre-wrap">{quote.terms}</p>
            </div>
          )}
        </div>
      </div>

      {/* Print styles */}
      <style>{`
        @media print {
          @page { size: letter; margin: 0.75in; }
          .no-print { display: none !important; }
          body { background: white !important; }
        }
      `}</style>
    </div>
  )
}

// ── OptionSection ──────────────────────────────────────────

function OptionSection({ option, quote, hasAdjustments }: { option: QuoteOption; quote: Quote; hasAdjustments: boolean }) {
  const financials = calculateFinancials(option.total_cents, quote)

  // Group items by type
  const groupedItems = new Map<QuoteLineItemType, QuoteLineItem[]>()
  if (option.items) {
    for (const item of option.items) {
      const type = item.type as QuoteLineItemType
      if (!groupedItems.has(type)) groupedItems.set(type, [])
      groupedItems.get(type)!.push(item)
    }
  }
  const orderedGroups = typeOrder
    .filter(t => groupedItems.has(t))
    .map(t => ({ type: t, items: groupedItems.get(t)! }))

  const isAccepted = quote.accepted_option_id === option.id

  return (
    <div className={`mb-6 border rounded-lg overflow-hidden quote-option ${isAccepted ? 'border-green-300' : 'border-gray-200'}`}>
      {/* Header */}
      <div className={`px-4 py-3 ${isAccepted ? 'bg-green-50' : 'bg-gray-50'} border-b border-gray-200`}>
        <div className="flex items-center justify-between">
          <h3 className="text-sm font-semibold text-gray-900">
            {option.name}
            {isAccepted && <span className="ml-2 text-xs text-green-600">(Accepted)</span>}
          </h3>
          <span className="text-sm font-mono font-semibold text-gray-900">
            {hasAdjustments ? formatCents(financials.grandTotal) : formatCents(option.total_cents)}
          </span>
        </div>
        {option.description && <p className="text-xs text-gray-500 mt-0.5">{option.description}</p>}
      </div>

      {/* Grouped Items */}
      {orderedGroups.map(({ type, items }) => {
        const groupTotal = items.reduce((sum, it) => sum + it.total_cents, 0)
        return (
          <div key={type}>
            <div className="flex items-center justify-between px-4 py-1.5 bg-gray-50/50 border-b border-gray-100">
              <span className="text-xs font-semibold text-gray-400 uppercase tracking-wide">
                {typeLabels[type] || type} ({items.length})
              </span>
              <span className="text-xs font-mono text-gray-400">{formatCents(groupTotal)}</span>
            </div>
            <table className="w-full">
              <tbody>
                {items.map((item) => (
                  <tr key={item.id} className="border-b border-gray-50">
                    <td className="pl-6 pr-4 py-1.5 text-sm text-gray-700">{item.description}</td>
                    <td className="px-4 py-1.5 text-xs text-gray-500 text-right whitespace-nowrap">{item.quantity} {item.unit}</td>
                    <td className="px-4 py-1.5 text-xs text-gray-500 text-right font-mono whitespace-nowrap">@{formatCents(item.unit_price_cents)}</td>
                    <td className="px-4 py-1.5 text-sm text-gray-800 text-right font-mono whitespace-nowrap">{formatCents(item.total_cents)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )
      })}

      {/* Financial Summary */}
      {option.items && option.items.length > 0 && (
        <div className="border-t border-gray-200 px-4 py-3 space-y-1">
          <div className="flex justify-between text-xs text-gray-500">
            <span>Subtotal</span>
            <span className="font-mono">{formatCents(option.total_cents)}</span>
          </div>
          {quote.discount_type !== 'none' && financials.discountAmount > 0 && (
            <div className="flex justify-between text-xs text-gray-500">
              <span>Discount {quote.discount_type === 'percent' ? `(${(quote.discount_value / 100).toFixed(2)}%)` : '(flat)'}</span>
              <span className="font-mono text-red-600">-{formatCents(financials.discountAmount)}</span>
            </div>
          )}
          {quote.rush_fee_cents > 0 && (
            <div className="flex justify-between text-xs text-gray-500">
              <span>Rush Fee</span>
              <span className="font-mono">{formatCents(quote.rush_fee_cents)}</span>
            </div>
          )}
          {quote.tax_rate > 0 && (
            <div className="flex justify-between text-xs text-gray-500">
              <span>Tax ({(quote.tax_rate / 100).toFixed(2)}%)</span>
              <span className="font-mono">{formatCents(financials.taxAmount)}</span>
            </div>
          )}
          {hasAdjustments && (
            <div className="flex justify-between text-sm font-semibold text-gray-900 pt-1 border-t border-gray-200">
              <span>Total</span>
              <span className="font-mono">{formatCents(financials.grandTotal)}</span>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
