import { useState, useCallback } from 'react'
import { toast } from '@/hooks/useToast'
import type { GoalSummary, GoalDetail, GoalStatus } from '@/lib/types'

export function useGoals() {
  const [goals, setGoals] = useState<GoalSummary[]>([])
  const [selectedGoal, setSelectedGoal] = useState<GoalDetail | null>(null)
  const [goalStatus, setGoalStatus] = useState<GoalStatus | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchGoals = useCallback(async () => {
    try {
      setError(null)
      const res = await fetch('/api/goals')
      if (!res.ok) {
        throw new Error(`HTTP ${res.status}: ${res.statusText}`)
      }
      const data = await res.json()
      setGoals(data || [])
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to fetch goals'
      console.error('Failed to fetch goals:', err)
      setError(message)
      toast({
        title: 'Connection Error',
        description: 'Could not load goals. Is vega-hub running?',
        variant: 'destructive',
      })
    } finally {
      setLoading(false)
    }
  }, [])

  const fetchGoalDetail = useCallback(async (id: string) => {
    try {
      const res = await fetch(`/api/goals/${id}`)
      if (!res.ok) {
        throw new Error(`HTTP ${res.status}: ${res.statusText}`)
      }
      const data = await res.json()
      setSelectedGoal(data)
      // Also fetch status
      fetchGoalStatus(id)
    } catch (err) {
      console.error('Failed to fetch goal detail:', err)
      toast({
        title: 'Error',
        description: `Could not load goal #${id}`,
        variant: 'destructive',
      })
    }
  }, [])

  const fetchGoalStatus = useCallback(async (id: string) => {
    try {
      const res = await fetch(`/api/goals/${id}/status`)
      if (!res.ok) {
        // Status endpoint may 404 if no worktree - that's ok
        if (res.status !== 404) {
          throw new Error(`HTTP ${res.status}: ${res.statusText}`)
        }
        return
      }
      const data = await res.json()
      setGoalStatus(data)
    } catch (err) {
      console.error('Failed to fetch goal status:', err)
      // Don't toast for status errors - less critical
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
    error,
    totalPendingQuestions,
    fetchGoals,
    fetchGoalDetail,
    fetchGoalStatus,
    clearSelectedGoal,
  }
}
