import { useEffect, useState, useMemo } from 'react'
import { Link } from 'react-router-dom'
import { Calendar, ChevronLeft, ChevronRight, Package, Layers, Printer } from 'lucide-react'
import { timelineApi } from '../api/client'
import type { TimelineItem } from '../types'

const statusColors: Record<string, string> = {
  pending: 'bg-gray-200',
  in_progress: 'bg-blue-400',
  printing: 'bg-blue-500',
  completed: 'bg-green-500',
  shipped: 'bg-purple-500',
  cancelled: 'bg-red-400',
  failed: 'bg-red-500',
  queued: 'bg-gray-300',
  paused: 'bg-amber-400',
}

const typeIcons: Record<string, React.ReactNode> = {
  order: <Package className="w-4 h-4" />,
  project: <Layers className="w-4 h-4" />,
  job: <Printer className="w-4 h-4" />,
}

export function Timeline() {
  const [items, setItems] = useState<TimelineItem[]>([])
  const [loading, setLoading] = useState(true)
  const [startDate, setStartDate] = useState(() => {
    const date = new Date()
    date.setDate(date.getDate() - 7)
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

  const navigateDays = (days: number) => {
    const newDate = new Date(startDate)
    newDate.setDate(newDate.getDate() + days)
    setStartDate(newDate)
  }

  const goToToday = () => {
    const today = new Date()
    today.setDate(today.getDate() - 3) // Show a few days before today
    setStartDate(today)
  }

  const isToday = (date: Date) => {
    const today = new Date()
    return date.toDateString() === today.toDateString()
  }

  const formatDate = (date: Date) => {
    return date.toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric' })
  }

  const getItemPosition = (item: TimelineItem) => {
    if (!item.start_date) return null

    const itemStart = new Date(item.start_date)
    const itemEnd = item.end_date ? new Date(item.end_date) : new Date()
    const rangeStart = startDate.getTime()
    const rangeEnd = dateRange[dateRange.length - 1].getTime() + 24 * 60 * 60 * 1000

    // Calculate position as percentage
    const dayWidth = 100 / daysToShow
    const startOffset = Math.max(0, (itemStart.getTime() - rangeStart) / (rangeEnd - rangeStart) * 100)
    const endOffset = Math.min(100, (itemEnd.getTime() - rangeStart) / (rangeEnd - rangeStart) * 100)
    const width = Math.max(dayWidth * 0.5, endOffset - startOffset)

    return { left: `${startOffset}%`, width: `${width}%` }
  }

  return (
    <div className="p-6">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold">Timeline</h1>
          <p className="text-gray-600">Visual overview of orders and jobs</p>
        </div>
        <div className="flex items-center gap-4">
          <select
            value={daysToShow}
            onChange={e => setDaysToShow(Number(e.target.value))}
            className="px-3 py-2 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value={7}>1 week</option>
            <option value={14}>2 weeks</option>
            <option value={30}>1 month</option>
          </select>
          <div className="flex items-center gap-1">
            <button
              onClick={() => navigateDays(-daysToShow)}
              className="p-2 hover:bg-gray-100 rounded-lg transition-colors"
            >
              <ChevronLeft className="w-5 h-5" />
            </button>
            <button
              onClick={goToToday}
              className="px-3 py-1.5 text-sm text-blue-600 hover:bg-blue-50 rounded-lg transition-colors"
            >
              Today
            </button>
            <button
              onClick={() => navigateDays(daysToShow)}
              className="p-2 hover:bg-gray-100 rounded-lg transition-colors"
            >
              <ChevronRight className="w-5 h-5" />
            </button>
          </div>
        </div>
      </div>

      {/* Timeline */}
      <div className="bg-white rounded-lg border overflow-hidden">
        {/* Date header */}
        <div className="flex border-b">
          <div className="w-48 flex-shrink-0 px-4 py-3 bg-gray-50 border-r font-medium text-sm text-gray-700">
            Item
          </div>
          <div className="flex-1 flex">
            {dateRange.map((date, i) => (
              <div
                key={i}
                className={`flex-1 px-2 py-3 text-center text-xs font-medium border-r last:border-r-0 ${
                  isToday(date) ? 'bg-blue-50 text-blue-700' : 'text-gray-500'
                }`}
              >
                {formatDate(date)}
              </div>
            ))}
          </div>
        </div>

        {/* Content */}
        {loading ? (
          <div className="text-center py-12 text-gray-500">Loading timeline...</div>
        ) : items.length === 0 ? (
          <div className="text-center py-12">
            <Calendar className="w-12 h-12 mx-auto text-gray-400 mb-4" />
            <h3 className="text-lg font-medium text-gray-900 mb-2">No items in this period</h3>
            <p className="text-gray-500">Adjust the date range or create new orders</p>
          </div>
        ) : (
          <div className="divide-y">
            {items.map(item => (
              <TimelineRow
                key={item.id}
                item={item}
                dateRange={dateRange}
                getPosition={getItemPosition}
              />
            ))}
          </div>
        )}
      </div>

      {/* Legend */}
      <div className="mt-4 flex items-center gap-4 text-sm text-gray-600">
        <span className="font-medium">Status:</span>
        {Object.entries(statusColors).slice(0, 6).map(([status, color]) => (
          <div key={status} className="flex items-center gap-1">
            <div className={`w-3 h-3 rounded ${color}`} />
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
  depth?: number
}

function TimelineRow({ item, dateRange, getPosition, depth = 0 }: TimelineRowProps) {
  const [expanded, setExpanded] = useState(depth === 0)
  const position = getPosition(item)
  const hasChildren = item.children && item.children.length > 0

  const linkTo = item.type === 'order' ? `/orders/${item.id}` : `/projects/${item.id}`

  return (
    <>
      <div className="flex hover:bg-gray-50">
        {/* Label */}
        <div
          className="w-48 flex-shrink-0 px-4 py-3 border-r flex items-center gap-2"
          style={{ paddingLeft: `${16 + depth * 16}px` }}
        >
          {hasChildren && (
            <button
              onClick={() => setExpanded(!expanded)}
              className="p-0.5 hover:bg-gray-200 rounded"
            >
              <ChevronRight
                className={`w-4 h-4 transition-transform ${expanded ? 'rotate-90' : ''}`}
              />
            </button>
          )}
          <span className="text-gray-400">{typeIcons[item.type]}</span>
          <Link
            to={linkTo}
            className="text-sm font-medium text-gray-900 hover:text-blue-600 truncate"
          >
            {item.name || `${item.type} ${item.id.slice(0, 8)}`}
          </Link>
        </div>

        {/* Bar */}
        <div className="flex-1 relative py-2">
          {/* Grid lines */}
          <div className="absolute inset-0 flex">
            {dateRange.map((_, i) => (
              <div key={i} className="flex-1 border-r border-gray-100" />
            ))}
          </div>

          {/* Item bar */}
          {position && (
            <div
              className={`absolute top-1/2 -translate-y-1/2 h-6 rounded ${statusColors[item.status] || 'bg-gray-300'}`}
              style={{ left: position.left, width: position.width, minWidth: '20px' }}
              title={`${item.name}: ${item.status} (${Math.round(item.progress)}%)`}
            >
              {/* Progress indicator */}
              {item.progress > 0 && item.progress < 100 && (
                <div
                  className="absolute inset-y-0 left-0 bg-white/30 rounded-l"
                  style={{ width: `${item.progress}%` }}
                />
              )}
            </div>
          )}

          {/* Due date marker */}
          {item.due_date && (() => {
            const dueDate = new Date(item.due_date)
            const rangeStart = dateRange[0].getTime()
            const rangeEnd = dateRange[dateRange.length - 1].getTime() + 24 * 60 * 60 * 1000
            const duePct = ((dueDate.getTime() - rangeStart) / (rangeEnd - rangeStart)) * 100
            if (duePct >= 0 && duePct <= 100) {
              return (
                <div
                  className="absolute top-0 bottom-0 w-0.5 bg-red-500"
                  style={{ left: `${duePct}%` }}
                  title={`Due: ${dueDate.toLocaleDateString()}`}
                />
              )
            }
            return null
          })()}
        </div>
      </div>

      {/* Children */}
      {expanded && hasChildren && (
        <>
          {item.children!.map(child => (
            <TimelineRow
              key={child.id}
              item={child}
              dateRange={dateRange}
              getPosition={getPosition}
              depth={depth + 1}
            />
          ))}
        </>
      )}
    </>
  )
}

export default Timeline
