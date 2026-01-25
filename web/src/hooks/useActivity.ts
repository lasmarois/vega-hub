import { useState, useCallback } from 'react'
import type { Activity } from '@/lib/types'

const MAX_ACTIVITIES = 50

export function useActivity() {
  const [activities, setActivities] = useState<Activity[]>([])

  const addActivity = useCallback((
    type: Activity['type'],
    message: string,
    goalId?: string,
    sessionId?: string
  ) => {
    const activity: Activity = {
      id: crypto.randomUUID(),
      type,
      goal_id: goalId,
      session_id: sessionId,
      message,
      timestamp: new Date().toISOString(),
    }

    setActivities((prev) => [activity, ...prev].slice(0, MAX_ACTIVITIES))
  }, [])

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
    recordExecutorStarted,
    recordExecutorStopped,
    recordQuestion,
    recordAnswered,
    recordGoalUpdated,
    recordGoalIced,
    recordGoalCompleted,
  }
}
