import { useEffect, useState, useMemo } from 'react'
import { Link } from 'react-router-dom'
import { Calendar, ChevronLeft, ChevronRight, Package, ListTodo, Printer } from 'lucide-react'
import { timelineApi } from '../api/client'
import type { TimelineItem } from '../types'
import { cn } from '../lib/utils'

const statusColors: Record<string, { bar: string; text: string }> = {
  pending:     { bar: 'bg-surface-600',   text: 'text-surface-400' },
  in_progress: { bar: 'bg-blue-500',      text: 'text-blue-400' },
  printing:    { bar: 'bg-blue-500',      text: 'text-blue-400' },
  completed:   { bar: 'bg-emerald-500',   text: 'text-emerald-400' },
  shipped:     { bar: 'bg-purple-500',    text: 'text-purple-400' },
  cancelled:   { bar: 'bg-red-500/60',    text: 'text-red-400' },
  failed:      { bar: 'bg-red-500',       text: 'text-red-400' },
  queued:      { bar: 'bg-surface-500',   text: 'text-surface-400' },
  paused:      { bar: 'bg-amber-500',     text: 'text-amber-400' },
}

const typeIcons: Record<string, React.ReactNode> = {
  order: <Package className="w-3.5 h-3.5" />,
  task: <ListTodo className="w-3.5 h-3.5" />,
  job: <Printer className="w-3.5 h-3.5" />,
}

const DAY_MS = 24 * 60 * 60 * 1000

export function Timeline() {
  const [items, setItems] = useState<TimelineItem[]>([])
  const [loading, setLoading] = useState(true)
  const [startDate, setStartDate] = useState(() => {
    const date = new Date()
    date.setDate(date.getDate() - 7)
    date.setHours(0, 0, 0, 0)
    return date
  })
  const [daysToShow, setDaysToShow] = useState(14)

  useEffect(() => {
    loadTimeline()
  }, [startDate, daysToShow])

  const loadTimeline = async () => {
    setLoading(true)
    try {
      const endDate = new Date(startDate)
      endDate.setDate(endDate.getDate() + daysToShow)
      const data = await timelineApi.getTimeline({
        start: startDate.toISOString(),
        end: endDate.toISOString(),
      })
      setItems(data)
    } catch (err) {
      console.error('Failed to load timeline:', err)
    } finally {
      setLoading(false)
    }
  }

  const dateRange = useMemo(() => {
    const dates: Date[] = []
    const current = new Date(startDate)
    for (let i = 0; i < daysToShow; i++) {
      dates.push(new Date(current))
      current.setDate(current.getDate() + 1)
    }
    return dates
  }, [startDate, daysToShow])

  const rangeStartMs = startDate.getTime()
  const rangeEndMs = rangeStartMs + daysToShow * DAY_MS
  const rangeDuration = rangeEndMs - rangeStartMs

  const navigateDays = (days: number) => {
    const newDate = new Date(startDate)
    newDate.setDate(newDate.getDate() + days)
    setStartDate(newDate)
  }

  const goToToday = () => {
    const today = new Date()
    today.setHours(0, 0, 0, 0)
    today.setDate(today.getDate() - 3)
    setStartDate(today)
  }

  const isToday = (date: Date) => {
    const today = new Date()
    return date.toDateString() === today.toDateString()
  }

  const todayPct = useMemo(() => {
    const now = new Date()
    now.setHours(12, 0, 0, 0)
    const pct = ((now.getTime() - rangeStartMs) / rangeDuration) * 100
    return pct >= 0 && pct <= 100 ? pct : null
  }, [rangeStartMs, rangeDuration])

  const getItemPosition = (item: TimelineItem) => {
    if (!item.start_date) return null

    const itemStart = new Date(item.start_date)
    const itemEnd = item.end_date ? new Date(item.end_date) : item.due_date ? new Date(item.due_date) : new Date()
    const minDuration = DAY_MS // at least 1 day wide
    const effectiveEnd = Math.max(itemEnd.getTime(), itemStart.getTime() + minDuration)

    const startPct = Math.max(0, ((itemStart.getTime() - rangeStartMs) / rangeDuration) * 100)
    const endPct = Math.min(100, ((effectiveEnd - rangeStartMs) / rangeDuration) * 100)
    const width = Math.max(100 / daysToShow * 0.6, endPct - startPct)

    return { left: `${startPct}%`, width: `${width}%` }
  }

  return (
    <div className="p-6">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-display font-bold text-surface-100">Timeline</h1>
          <p className="text-surface-400 text-sm mt-1">Gantt view of orders and print jobs</p>
        </div>
        <div className="flex items-center gap-3">
          <select
            value={daysToShow}
            onChange={e => setDaysToShow(Number(e.target.value))}
            className="input h-auto py-1.5 w-auto text-sm"
          >
            <option value={7}>1 week</option>
            <option value={14}>2 weeks</option>
            <option value={30}>1 month</option>
          </select>
          <div className="flex items-center gap-0.5 bg-surface-800 rounded-lg p-0.5">
            <button
              onClick={() => navigateDays(-daysToShow)}
              className="p-1.5 hover:bg-surface-700 rounded-md transition-colors text-surface-400 hover:text-surface-100"
            >
              <ChevronLeft className="w-4 h-4" />
            </button>
            <button
              onClick={goToToday}
              className="px-3 py-1.5 text-xs font-medium text-accent-400 hover:bg-surface-700 rounded-md transition-colors"
            >
              Today
            </button>
            <button
              onClick={() => navigateDays(daysToShow)}
              className="p-1.5 hover:bg-surface-700 rounded-md transition-colors text-surface-400 hover:text-surface-100"
            >
              <ChevronRight className="w-4 h-4" />
            </button>
          </div>
        </div>
      </div>

      {/* Gantt chart */}
      <div className="rounded-xl border border-surface-800 bg-surface-900/50 overflow-hidden">
        {/* Date header */}
        <div className="flex border-b border-surface-800">
          <div className="w-56 flex-shrink-0 px-4 py-2.5 bg-surface-900 border-r border-surface-800 text-xs font-medium text-surface-400 uppercase tracking-wider">
            Item
          </div>
          <div className="flex-1 flex relative">
            {dateRange.map((date, i) => (
              <div
                key={i}
                className={cn(
                  'flex-1 py-2.5 text-center text-xs font-medium border-r border-surface-800/50 last:border-r-0',
                  isToday(date)
                    ? 'bg-accent-500/10 text-accent-400'
                    : 'text-surface-500'
                )}
              >
                <div>{date.toLocaleDateString('en-US', { weekday: 'short' })}</div>
                <div className={cn('text-[11px]', isToday(date) ? 'text-accent-300' : 'text-surface-600')}>
                  {date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })}
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Rows */}
        {loading ? (
          <div className="text-center py-16 text-surface-500">Loading timeline...</div>
        ) : items.length === 0 ? (
          <div className="text-center py-16">
            <Calendar className="w-10 h-10 mx-auto text-surface-600 mb-3" />
            <h3 className="text-sm font-medium text-surface-300 mb-1">No items in this period</h3>
            <p className="text-xs text-surface-500">Adjust the date range or create new orders</p>
          </div>
        ) : (
          <div>
            {items.map(item => (
              <TimelineRow
                key={item.id}
                item={item}
                dateRange={dateRange}
                getPosition={getItemPosition}
                todayPct={todayPct}
              />
            ))}
          </div>
        )}
      </div>

      {/* Legend */}
      <div className="mt-4 flex items-center gap-5 text-xs text-surface-500">
        <span className="font-medium text-surface-400">Status:</span>
        {Object.entries(statusColors).slice(0, 6).map(([status, colors]) => (
          <div key={status} className="flex items-center gap-1.5">
            <div className={cn('w-2.5 h-2.5 rounded-sm', colors.bar)} />
            <span className="capitalize">{status.replace('_', ' ')}</span>
          </div>
        ))}
      </div>
    </div>
  )
}

interface TimelineRowProps {
  item: TimelineItem
  dateRange: Date[]
  getPosition: (item: TimelineItem) => { left: string; width: string } | null
  todayPct: number | null
  depth?: number
}

function TimelineRow({ item, dateRange, getPosition, todayPct, depth = 0 }: TimelineRowProps) {
  const [expanded, setExpanded] = useState(depth === 0)
  const position = getPosition(item)
  const hasChildren = item.children && item.children.length > 0
  const colors = statusColors[item.status] || statusColors.pending

  const linkTo = item.type === 'order'
    ? `/orders/${item.id}`
    : item.type === 'job'
    ? `/jobs/${item.id}`
    : `/projects/${item.id}`

  return (
    <>
      <div className={cn(
        'flex group',
        depth === 0 ? 'border-b border-surface-800/50' : 'border-b border-surface-800/20',
        depth > 0 && 'bg-surface-900/30'
      )}>
        {/* Label column */}
        <div
          className="w-56 flex-shrink-0 px-3 py-2.5 border-r border-surface-800 flex items-center gap-1.5 min-w-0"
          style={{ paddingLeft: `${12 + depth * 16}px` }}
        >
          {hasChildren ? (
            <button
              onClick={() => setExpanded(!expanded)}
              className="p-0.5 hover:bg-surface-700 rounded transition-colors flex-shrink-0"
            >
              <ChevronRight
                className={cn('w-3.5 h-3.5 text-surface-500 transition-transform', expanded && 'rotate-90')}
              />
            </button>
          ) : (
            <span className="w-4.5" />
          )}
          <span className={cn('flex-shrink-0', colors.text)}>{typeIcons[item.type]}</span>
          <Link
            to={linkTo}
            className="text-sm text-surface-200 hover:text-accent-400 truncate transition-colors"
          >
            {item.name || `${item.type} ${item.id.slice(0, 8)}`}
          </Link>
        </div>

        {/* Bar area */}
        <div className="flex-1 relative" style={{ height: '40px' }}>
          {/* Day grid lines */}
          <div className="absolute inset-0 flex pointer-events-none">
            {dateRange.map((_, i) => (
              <div key={i} className="flex-1 border-r border-surface-800/30" />
            ))}
          </div>

          {/* Today line */}
          {todayPct !== null && (
            <div
              className="absolute top-0 bottom-0 w-px bg-accent-500/40 z-10 pointer-events-none"
              style={{ left: `${todayPct}%` }}
            />
          )}

          {/* Gantt bar */}
          {position && (
            <div
              className={cn(
                'absolute top-1/2 -translate-y-1/2 rounded-sm transition-opacity',
                depth === 0 ? 'h-5' : 'h-3.5',
                colors.bar
              )}
              style={{ left: position.left, width: position.width, minWidth: '8px' }}
              title={`${item.name}: ${item.status}${item.progress > 0 ? ` (${Math.round(item.progress)}%)` : ''}`}
            >
              {/* Progress fill */}
              {item.progress > 0 && item.progress < 100 && (
                <div
                  className="absolute inset-y-0 left-0 bg-white/20 rounded-l-sm"
                  style={{ width: `${item.progress}%` }}
                />
              )}
            </div>
          )}

          {/* Due date marker */}
          {item.due_date && (() => {
            const dueMs = new Date(item.due_date).getTime()
            const rangeStart = dateRange[0].getTime()
            const rangeEnd = dateRange[dateRange.length - 1].getTime() + DAY_MS
            const pct = ((dueMs - rangeStart) / (rangeEnd - rangeStart)) * 100
            if (pct >= 0 && pct <= 100) {
              return (
                <div
                  className="absolute top-1 bottom-1 w-0.5 bg-red-500/70 z-10 pointer-events-none"
                  style={{ left: `${pct}%` }}
                  title={`Due: ${new Date(item.due_date).toLocaleDateString()}`}
                />
              )
            }
            return null
          })()}
        </div>
      </div>

      {/* Children rows */}
      {expanded && hasChildren && item.children!.map(child => (
        <TimelineRow
          key={child.id}
          item={child}
          dateRange={dateRange}
          getPosition={getPosition}
          todayPct={todayPct}
          depth={depth + 1}
        />
      ))}
    </>
  )
}

export default Timeline
