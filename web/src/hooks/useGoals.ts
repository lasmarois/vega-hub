import { useState, useCallback } from 'react'
import type { GoalSummary, GoalDetail, GoalStatus } from '@/lib/types'

export function useGoals() {
  const [goals, setGoals] = useState<GoalSummary[]>([])
  const [selectedGoal, setSelectedGoal] = useState<GoalDetail | null>(null)
  const [goalStatus, setGoalStatus] = useState<GoalStatus | null>(null)
  const [loading, setLoading] = useState(true)

  const fetchGoals = useCallback(async () => {
    try {
      const res = await fetch('/api/goals')
      if (res.ok) {
        const data = await res.json()
        setGoals(data || [])
      }
    } catch (err) {
      console.error('Failed to fetch goals:', err)
    } finally {
      setLoading(false)
    }
  }, [])

  const fetchGoalDetail = useCallback(async (id: string) => {
    try {
      const res = await fetch(`/api/goals/${id}`)
      if (res.ok) {
        const data = await res.json()
        setSelectedGoal(data)
        // Also fetch status
        fetchGoalStatus(id)
      }
    } catch (err) {
      console.error('Failed to fetch goal detail:', err)
    }
  }, [])

  const fetchGoalStatus = useCallback(async (id: string) => {
    try {
      const res = await fetch(`/api/goals/${id}/status`)
      if (res.ok) {
        const data = await res.json()
        setGoalStatus(data)
      }
    } catch (err) {
      console.error('Failed to fetch goal status:', err)
    }
  }, [])

  const clearSelectedGoal = useCallback(() => {
    setSelectedGoal(null)
    setGoalStatus(null)
  }, [])

  const totalPendingQuestions = goals.reduce((sum, g) => sum + g.pending_questions, 0)

  return {
    goals,
    selectedGoal,
    goalStatus,
    loading,
    totalPendingQuestions,
    fetchGoals,
    fetchGoalDetail,
    fetchGoalStatus,
    clearSelectedGoal,
  }
}
