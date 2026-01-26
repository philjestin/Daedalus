import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

// Merge Tailwind classes with clsx.
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

// Format bytes to human readable string.
export function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(1))} ${sizes[i]}`
}

// Format duration in seconds to human readable string.
export function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`
  const hours = Math.floor(seconds / 3600)
  const mins = Math.floor((seconds % 3600) / 60)
  return `${hours}h ${mins}m`
}

// Format date to relative time.
export function formatRelativeTime(dateString: string): string {
  const date = new Date(dateString)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / (1000 * 60))
  
  if (diffMins < 1) return 'just now'
  if (diffMins < 60) return `${diffMins}m ago`
  
  const diffHours = Math.floor(diffMins / 60)
  if (diffHours < 24) return `${diffHours}h ago`
  
  const diffDays = Math.floor(diffHours / 24)
  if (diffDays < 7) return `${diffDays}d ago`
  
  return date.toLocaleDateString()
}

// Get status color class.
export function getStatusColor(status: string): string {
  switch (status) {
    case 'idle':
    case 'draft':
    case 'new':
      return 'text-surface-400'
    case 'active':
    case 'printing':
    case 'in_use':
      return 'text-blue-400'
    case 'completed':
    case 'complete':
      return 'text-emerald-400'
    case 'paused':
    case 'low':
      return 'text-amber-400'
    case 'error':
    case 'failed':
    case 'cancelled':
      return 'text-red-400'
    case 'offline':
    case 'archived':
    case 'empty':
      return 'text-surface-500'
    default:
      return 'text-surface-400'
  }
}

// Get status badge classes.
export function getStatusBadge(status: string): string {
  switch (status) {
    case 'idle':
    case 'draft':
    case 'new':
      return 'bg-surface-700/50 text-surface-300'
    case 'active':
    case 'printing':
    case 'in_use':
    case 'sending':
      return 'bg-blue-500/20 text-blue-400'
    case 'completed':
    case 'complete':
      return 'bg-emerald-500/20 text-emerald-400'
    case 'paused':
    case 'low':
    case 'queued':
      return 'bg-amber-500/20 text-amber-400'
    case 'error':
    case 'failed':
    case 'cancelled':
      return 'bg-red-500/20 text-red-400'
    case 'offline':
    case 'archived':
    case 'empty':
      return 'bg-surface-800 text-surface-500'
    default:
      return 'bg-surface-700/50 text-surface-300'
  }
}

