import { useEffect, useCallback, useState } from 'react'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { Layout } from '@/components/layout'
import { Home, Goals, Projects, History } from '@/pages'
import { useSSE } from '@/hooks/useSSE'
import { useGoals } from '@/hooks/useGoals'
import { GoalSheet } from '@/components/goals/GoalSheet'

function AppContent() {
  const {
    goals,
    selectedGoal,
    goalStatus,
    loading,
    totalPendingQuestions,
    fetchGoals,
    fetchGoalDetail,
    fetchGoalStatus,
    clearSelectedGoal,
  } = useGoals()

  const [sheetOpen, setSheetOpen] = useState(false)

  // SSE handlers
  const handleQuestion = useCallback(() => {
    fetchGoals()
    if (selectedGoal) {
      fetchGoalDetail(selectedGoal.id)
    }
  }, [fetchGoals, selectedGoal, fetchGoalDetail])

  const handleAnswered = useCallback(() => {
    fetchGoals()
    if (selectedGoal) {
      fetchGoalDetail(selectedGoal.id)
    }
  }, [fetchGoals, selectedGoal, fetchGoalDetail])

  const handleExecutorStarted = useCallback(() => {
    fetchGoals()
    if (selectedGoal) {
      fetchGoalDetail(selectedGoal.id)
    }
  }, [fetchGoals, selectedGoal, fetchGoalDetail])

  const handleExecutorStopped = useCallback(() => {
    fetchGoals()
    if (selectedGoal) {
      fetchGoalDetail(selectedGoal.id)
    }
  }, [fetchGoals, selectedGoal, fetchGoalDetail])

  const handleGoalUpdated = useCallback((data: { goal_id: string }) => {
    fetchGoals()
    if (selectedGoal && data.goal_id === selectedGoal.id) {
      fetchGoalDetail(selectedGoal.id)
    }
  }, [fetchGoals, selectedGoal, fetchGoalDetail])

  const handleRegistryUpdated = useCallback(() => {
    fetchGoals()
  }, [fetchGoals])

  const { connected } = useSSE({
    onQuestion: handleQuestion,
    onAnswered: handleAnswered,
    onExecutorStarted: handleExecutorStarted,
    onExecutorStopped: handleExecutorStopped,
    onGoalUpdated: handleGoalUpdated,
    onRegistryUpdated: handleRegistryUpdated,
  })

  // Fetch goals on mount
  useEffect(() => {
    fetchGoals()
  }, [fetchGoals])

  // Auto-refresh goal status while executor is running
  useEffect(() => {
    if (!selectedGoal || selectedGoal.executor_status !== 'running') return

    const interval = setInterval(() => {
      fetchGoalStatus(selectedGoal.id)
    }, 5000)

    return () => clearInterval(interval)
  }, [selectedGoal, fetchGoalStatus])

  const handleGoalClick = useCallback((id: string) => {
    fetchGoalDetail(id)
    setSheetOpen(true)
  }, [fetchGoalDetail])

  const handleSheetClose = useCallback(() => {
    setSheetOpen(false)
    // Clear selected goal after animation
    setTimeout(() => {
      clearSelectedGoal()
    }, 300)
  }, [clearSelectedGoal])

  return (
    <>
      <Routes>
        <Route element={<Layout connected={connected} pendingQuestions={totalPendingQuestions} />}>
          <Route
            path="/"
            element={
              <Home
                goals={goals}
                loading={loading}
                pendingQuestions={totalPendingQuestions}
                onGoalClick={handleGoalClick}
              />
            }
          />
          <Route
            path="/projects"
            element={<Projects goals={goals} loading={loading} />}
          />
          <Route
            path="/goals"
            element={<Goals goals={goals} loading={loading} onGoalClick={handleGoalClick} />}
          />
          <Route
            path="/history"
            element={<History goals={goals} loading={loading} onGoalClick={handleGoalClick} />}
          />
        </Route>
      </Routes>

      {/* Goal Detail Sheet */}
      <GoalSheet
        open={sheetOpen}
        onOpenChange={(open) => {
          if (!open) handleSheetClose()
        }}
        goal={selectedGoal}
        goalStatus={goalStatus}
        onRefresh={() => selectedGoal && fetchGoalDetail(selectedGoal.id)}
      />
    </>
  )
}

function App() {
  return (
    <BrowserRouter>
      <AppContent />
    </BrowserRouter>
  )
}

export default App
