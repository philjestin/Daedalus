import { useState } from 'react'
import { ChevronDown, ChevronRight } from 'lucide-react'
import { useJobEvents } from '../hooks/usePrinters'
import { cn } from '../lib/utils'
import type { JobEvent } from '../types'

const EVENT_ICONS: Record<string, { icon: string; color: string }> = {
  queued:    { icon: '\u25CB', color: 'text-surface-400' },
  assigned:  { icon: '\u25CF', color: 'text-blue-400' },
  uploaded:  { icon: '\u2191', color: 'text-blue-400' },
  started:   { icon: '\u25B6', color: 'text-emerald-400' },
  progress:  { icon: '\u2026', color: 'text-blue-300' },
  paused:    { icon: '\u23F8', color: 'text-amber-400' },
  resumed:   { icon: '\u25B6', color: 'text-emerald-400' },
  completed: { icon: '\u2713', color: 'text-emerald-400' },
  failed:    { icon: '\u2717', color: 'text-red-400' },
  cancelled: { icon: '\u2715', color: 'text-surface-400' },
  retried:   { icon: '\u21BB', color: 'text-amber-400' },
}

function formatEventTime(dateString: string): string {
  const d = new Date(dateString)
  return d.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

function EventRow({ event }: { event: JobEvent }) {
  const config = EVENT_ICONS[event.event_type] || { icon: '?', color: 'text-surface-400' }

  return (
    <div className="flex items-start gap-3 py-1.5">
      <span className={cn('text-sm w-4 text-center flex-shrink-0', config.color)}>
        {config.icon}
      </span>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="text-sm text-surface-200 capitalize">{event.event_type}</span>
          <span className="text-xs text-surface-500">{event.actor_type}</span>
        </div>
        {event.error_message && (
          <div className="text-xs text-red-400 mt-0.5">{event.error_message}</div>
        )}
        {event.progress !== undefined && event.event_type === 'progress' && (
          <div className="text-xs text-surface-400 mt-0.5">{event.progress}%</div>
        )}
      </div>
      <span className="text-xs text-surface-500 flex-shrink-0">
        {formatEventTime(event.occurred_at)}
      </span>
    </div>
  )
}

export function JobEventTimeline({ jobId }: { jobId: string }) {
  const { data: events, isLoading } = useJobEvents(jobId)

  if (isLoading) {
    return <div className="text-xs text-surface-500 py-2 pl-7">Loading events...</div>
  }

  if (!events || events.length === 0) {
    return <div className="text-xs text-surface-500 py-2 pl-7">No events recorded</div>
  }

  // Filter out frequent progress events, keep first, last, and every ~25% milestone
  const filtered = events.filter((e, i) => {
    if (e.event_type !== 'progress') return true
    if (i === 0 || i === events.length - 1) return true
    const prev = events[i - 1]
    if (prev.event_type !== 'progress') return true
    return (e.progress ?? 0) - (prev.progress ?? 0) >= 25
  })

  return (
    <div className="border-l-2 border-surface-700 ml-2 pl-3 py-1">
      {filtered.map((event) => (
        <EventRow key={event.id} event={event} />
      ))}
    </div>
  )
}

export function ExpandableJobEvents({ jobId }: { jobId: string }) {
  const [expanded, setExpanded] = useState(false)

  return (
    <div>
      <button
        onClick={() => setExpanded(!expanded)}
        className="flex items-center gap-1 text-xs text-surface-400 hover:text-surface-300 transition-colors"
      >
        {expanded ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
        Event Timeline
      </button>
      {expanded && <JobEventTimeline jobId={jobId} />}
    </div>
  )
}
