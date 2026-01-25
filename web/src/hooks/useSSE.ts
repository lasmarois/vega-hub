import { useEffect, useState, useRef } from 'react'

export interface SSEHandlers {
  onQuestion?: () => void
  onAnswered?: (data: { id: string }) => void
  onExecutorStarted?: (data: { goal_id: string; session_id: string }) => void
  onExecutorStopped?: (data: { goal_id: string; session_id: string; output?: string }) => void
  onGoalUpdated?: (data: { goal_id: string }) => void
  onRegistryUpdated?: () => void
  onGoalIced?: (data: { goal_id: string }) => void
  onGoalCompleted?: (data: { goal_id: string }) => void
}

export function useSSE(handlers: SSEHandlers) {
  const [connected, setConnected] = useState(false)
  const handlersRef = useRef(handlers)

  // Keep handlers ref updated
  useEffect(() => {
    handlersRef.current = handlers
  }, [handlers])

  useEffect(() => {
    const eventSource = new EventSource('/api/events')

    eventSource.addEventListener('connected', () => {
      setConnected(true)
    })

    eventSource.addEventListener('question', () => {
      handlersRef.current.onQuestion?.()
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

    eventSource.onerror = () => {
      setConnected(false)
    }

    return () => {
      eventSource.close()
    }
  }, [])

  return { connected }
}
