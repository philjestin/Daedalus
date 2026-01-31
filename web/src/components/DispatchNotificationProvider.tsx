import { useState, useEffect } from 'react'
import { onDispatchRequest } from '../hooks/useWebSocket'
import DispatchNotification from './DispatchNotification'
import type { DispatchRequest } from '../types'

export default function DispatchNotificationProvider() {
  const [request, setRequest] = useState<DispatchRequest | null>(null)

  useEffect(() => {
    // Subscribe to dispatch requests from WebSocket
    const unsubscribe = onDispatchRequest((req) => {
      setRequest(req)
    })

    return () => {
      unsubscribe()
    }
  }, [])

  if (!request) {
    return null
  }

  return (
    <DispatchNotification
      request={request}
      onDismiss={() => setRequest(null)}
    />
  )
}
