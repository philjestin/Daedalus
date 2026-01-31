import { useEffect, useRef, useCallback, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import type { PrinterState, DispatchRequest } from '../types'

// Build WebSocket URL - use relative path in production
function getWsUrl(): string {
  if (import.meta.env.DEV) {
    return 'ws://localhost:8080/ws'
  }
  // In production, use current host with appropriate protocol
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${protocol}//${window.location.host}/ws`
}
const WS_URL = getWsUrl()

// Reconnection config
const RECONNECT_INITIAL_DELAY = 1000
const RECONNECT_MAX_DELAY = 30000
const RECONNECT_MULTIPLIER = 2

interface WebSocketEvent {
  type: string
  data: unknown
}

type ConnectionStatus = 'connecting' | 'connected' | 'disconnected'

// Dispatch request callback type
type DispatchRequestCallback = (request: DispatchRequest | null) => void

// Global dispatch request callbacks
const dispatchRequestCallbacks = new Set<DispatchRequestCallback>()

// Subscribe to dispatch requests
export function onDispatchRequest(callback: DispatchRequestCallback) {
  dispatchRequestCallbacks.add(callback)
  return () => dispatchRequestCallbacks.delete(callback)
}

// Notify all subscribers of a dispatch request
function notifyDispatchRequest(request: DispatchRequest | null) {
  dispatchRequestCallbacks.forEach(cb => cb(request))
}

export function useWebSocket() {
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<number | null>(null)
  const reconnectDelayRef = useRef(RECONNECT_INITIAL_DELAY)
  const [status, setStatus] = useState<ConnectionStatus>('disconnected')
  const queryClient = useQueryClient()

  // Use a ref to hold the connect function to avoid circular dependency
  const connectRef = useRef<() => void>(() => {})

  const handleMessage = useCallback((event: WebSocketEvent) => {
    switch (event.type) {
      case 'printer_state_updated': {
        const state = event.data as PrinterState

        // Update the printer states cache
        queryClient.setQueryData<Record<string, PrinterState>>(
          ['printer-states'],
          (old) => {
            if (!old) return { [state.printer_id]: state }
            return { ...old, [state.printer_id]: state }
          }
        )

        // Also update individual printer state cache
        queryClient.setQueryData(
          ['printer-states', state.printer_id],
          state
        )
        break
      }

      case 'printer_connected':
      case 'printer_disconnected': {
        // Invalidate printer states to trigger refetch
        queryClient.invalidateQueries({ queryKey: ['printer-states'] })
        break
      }

      case 'job_started':
      case 'job_completed':
      case 'job_failed':
      case 'job_cancelled': {
        // Invalidate print jobs to trigger refetch
        queryClient.invalidateQueries({ queryKey: ['print-jobs'] })
        break
      }

      case 'dispatch_request': {
        // New dispatch request - notify subscribers
        const request = event.data as DispatchRequest
        notifyDispatchRequest(request)
        queryClient.invalidateQueries({ queryKey: ['dispatch-requests'] })
        break
      }

      case 'dispatch_confirmed': {
        // Dispatch was confirmed - clear notification and refresh jobs
        notifyDispatchRequest(null)
        queryClient.invalidateQueries({ queryKey: ['dispatch-requests'] })
        queryClient.invalidateQueries({ queryKey: ['print-jobs'] })
        break
      }

      case 'dispatch_rejected': {
        // Dispatch was rejected - clear notification
        notifyDispatchRequest(null)
        queryClient.invalidateQueries({ queryKey: ['dispatch-requests'] })
        break
      }

      case 'dispatch_expired': {
        // Dispatch expired - clear notification and refresh
        notifyDispatchRequest(null)
        queryClient.invalidateQueries({ queryKey: ['dispatch-requests'] })
        break
      }

      default:
        console.log('[WebSocket] Unknown event type:', event.type)
    }
  }, [queryClient])

  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return
    }

    setStatus('connecting')
    console.log('[WebSocket] Connecting to', WS_URL)

    const ws = new WebSocket(WS_URL)

    ws.onopen = () => {
      console.log('[WebSocket] Connected')
      setStatus('connected')
      reconnectDelayRef.current = RECONNECT_INITIAL_DELAY
    }

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data) as WebSocketEvent
        handleMessage(data)
      } catch (e) {
        console.error('[WebSocket] Failed to parse message:', e)
      }
    }

    ws.onerror = (error) => {
      console.error('[WebSocket] Error:', error)
    }

    ws.onclose = (event) => {
      console.log('[WebSocket] Disconnected:', event.code, event.reason)
      setStatus('disconnected')
      wsRef.current = null

      // Schedule reconnection with exponential backoff
      const delay = reconnectDelayRef.current
      console.log(`[WebSocket] Reconnecting in ${delay}ms...`)

      reconnectTimeoutRef.current = window.setTimeout(() => {
        reconnectDelayRef.current = Math.min(
          reconnectDelayRef.current * RECONNECT_MULTIPLIER,
          RECONNECT_MAX_DELAY
        )
        // Use ref to call connect to avoid stale closure
        connectRef.current()
      }, delay)
    }

    wsRef.current = ws
  }, [handleMessage])

  // Keep connectRef in sync with connect (must be in effect, not during render)
  useEffect(() => {
    connectRef.current = connect
  }, [connect])

  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
      reconnectTimeoutRef.current = null
    }
    if (wsRef.current) {
      wsRef.current.close(1000, 'User disconnect')
      wsRef.current = null
    }
    setStatus('disconnected')
  }, [])

  useEffect(() => {
    connect()
    return () => disconnect()
  }, [connect, disconnect])

  return { status, reconnect: connect }
}
