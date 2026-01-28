import { useEffect, useState, useRef, useCallback } from 'react'
import { toast } from '@/hooks/useToast'

export interface SSEHandlers {
  onQuestion?: (data: { goal_id: string; question: string }) => void
  onAnswered?: (data: { id: string }) => void
  onExecutorStarted?: (data: { goal_id: string; session_id: string }) => void
  onExecutorStopped?: (data: { goal_id: string; session_id: string; reason?: string; output?: string }) => void
  onGoalUpdated?: (data: { goal_id: string }) => void
  onRegistryUpdated?: () => void
  onGoalIced?: (data: { goal_id: string }) => void
  onGoalCompleted?: (data: { goal_id: string }) => void
  onUserMessage?: (data: { goal_id: string; content: string; user: string }) => void
  onPlanningFileReceived?: (data: { goal_id: string; project: string; filename: string }) => void
  onPhaseUpdated?: (data: { goal_id: string; phase: number; status: string; project?: string }) => void
}

const RECONNECT_DELAY = 3000 // 3 seconds

export function useSSE(handlers: SSEHandlers) {
  const [connected, setConnected] = useState(false)
  const handlersRef = useRef(handlers)
  const eventSourceRef = useRef<EventSource | null>(null)
  const reconnectTimeoutRef = useRef<number | null>(null)
  const wasConnectedRef = useRef(false)

  // Keep handlers ref updated
  useEffect(() => {
    handlersRef.current = handlers
  }, [handlers])

  const connect = useCallback(() => {
    // Clean up existing connection
    if (eventSourceRef.current) {
      eventSourceRef.current.close()
    }

    const eventSource = new EventSource('/api/events')
    eventSourceRef.current = eventSource

    eventSource.addEventListener('connected', () => {
      setConnected(true)
      // Show reconnected toast if we were previously connected
      if (wasConnectedRef.current) {
        toast({
          title: 'Connected',
          description: 'Reconnected to vega-hub',
          variant: 'success',
        })
      }
      wasConnectedRef.current = true
    })

    eventSource.addEventListener('question', (e) => {
      const data = JSON.parse(e.data)
      handlersRef.current.onQuestion?.({ goal_id: data.goal_id, question: data.question })
    })

    eventSource.addEventListener('answered', (e) => {
      const data = JSON.parse(e.data)
      handlersRef.current.onAnswered?.(data)
    })

    eventSource.addEventListener('executor_started', (e) => {
      const data = JSON.parse(e.data)
      handlersRef.current.onExecutorStarted?.(data)
    })

    eventSource.addEventListener('executor_stopped', (e) => {
      const data = JSON.parse(e.data)
      handlersRef.current.onExecutorStopped?.(data)
    })

    eventSource.addEventListener('goal_updated', (e) => {
      const data = JSON.parse(e.data)
      handlersRef.current.onGoalUpdated?.(data)
    })

    eventSource.addEventListener('registry_updated', () => {
      handlersRef.current.onRegistryUpdated?.()
    })

    eventSource.addEventListener('goal_iced', (e) => {
      const data = JSON.parse(e.data)
      handlersRef.current.onGoalIced?.(data)
    })

    eventSource.addEventListener('goal_completed', (e) => {
      const data = JSON.parse(e.data)
      handlersRef.current.onGoalCompleted?.(data)
    })

    eventSource.addEventListener('user_message', (e) => {
      const data = JSON.parse(e.data)
      handlersRef.current.onUserMessage?.(data)
    })

    eventSource.addEventListener('planning_file_received', (e) => {
      const data = JSON.parse(e.data)
      handlersRef.current.onPlanningFileReceived?.(data)
    })

    eventSource.addEventListener('phase_updated', (e) => {
      const data = JSON.parse(e.data)
      handlersRef.current.onPhaseUpdated?.(data)
    })

    eventSource.onerror = () => {
      setConnected(false)
      eventSource.close()

      // Show disconnected toast only if we were connected before
      if (wasConnectedRef.current) {
        toast({
          title: 'Disconnected',
          description: 'Lost connection to vega-hub. Reconnecting...',
          variant: 'destructive',
        })
      }

      // Schedule reconnect
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
      }
      reconnectTimeoutRef.current = window.setTimeout(() => {
        connect()
      }, RECONNECT_DELAY)
    }
  }, [])

  useEffect(() => {
    connect()

    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close()
      }
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
      }
    }
  }, [connect])

  return { connected }
}
