import { useState, useCallback } from 'react'
import type { Activity } from '@/lib/types'

const MAX_ACTIVITIES = 50

export interface ActivityWithRead extends Activity {
  read: boolean
}

export function useActivity() {
  const [activities, setActivities] = useState<ActivityWithRead[]>([])

  const addActivity = useCallback((
    type: Activity['type'],
    message: string,
    goalId?: string,
    sessionId?: string
  ) => {
    const activity: ActivityWithRead = {
      id: crypto.randomUUID(),
      type,
      goal_id: goalId,
      session_id: sessionId,
      message,
      timestamp: new Date().toISOString(),
      read: false,
    }

    setActivities((prev) => [activity, ...prev].slice(0, MAX_ACTIVITIES))
  }, [])

  const markAsRead = useCallback((id: string) => {
    setActivities((prev) =>
      prev.map((a) => (a.id === id ? { ...a, read: true } : a))
    )
  }, [])

  const markAllAsRead = useCallback(() => {
    setActivities((prev) => prev.map((a) => ({ ...a, read: true })))
  }, [])

  const unreadCount = activities.filter((a) => !a.read).length

  const recordExecutorStarted = useCallback((goalId: string, sessionId: string) => {
    addActivity('executor_started', `Executor started for goal`, goalId, sessionId)
  }, [addActivity])

  const recordExecutorStopped = useCallback((goalId: string, sessionId: string) => {
    addActivity('executor_stopped', `Executor stopped`, goalId, sessionId)
  }, [addActivity])

  const recordQuestion = useCallback((goalId?: string) => {
    addActivity('question', 'New question needs answer', goalId)
  }, [addActivity])

  const recordAnswered = useCallback((goalId?: string) => {
    addActivity('answered', 'Question answered', goalId)
  }, [addActivity])

  const recordGoalUpdated = useCallback((goalId: string) => {
    addActivity('goal_updated', 'Goal updated', goalId)
  }, [addActivity])

  const recordGoalIced = useCallback((goalId: string) => {
    addActivity('goal_iced', 'Goal iced', goalId)
  }, [addActivity])

  const recordGoalCompleted = useCallback((goalId: string) => {
    addActivity('goal_completed', 'Goal completed', goalId)
  }, [addActivity])

  return {
    activities,
    unreadCount,
    markAsRead,
    markAllAsRead,
    recordExecutorStarted,
    recordExecutorStopped,
    recordQuestion,
    recordAnswered,
    recordGoalUpdated,
    recordGoalIced,
    recordGoalCompleted,
  }
}
