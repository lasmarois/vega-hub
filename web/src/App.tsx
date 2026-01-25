import { useEffect, useCallback, useState } from 'react'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { Layout } from '@/components/layout'
import { Home, Goals, Projects, History } from '@/pages'
import type { ProjectStats } from '@/pages/Projects'
import { useSSE } from '@/hooks/useSSE'
import { useGoals } from '@/hooks/useGoals'
import { useActivity } from '@/hooks/useActivity'
import { GoalSheet } from '@/components/goals/GoalSheet'
import { ProjectSheet } from '@/components/projects/ProjectSheet'

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

  const {
    activities,
    recordExecutorStarted,
    recordExecutorStopped,
    recordQuestion,
    recordAnswered,
    recordGoalUpdated,
    recordGoalIced,
    recordGoalCompleted,
  } = useActivity()

  const [sheetOpen, setSheetOpen] = useState(false)
  const [projectSheetOpen, setProjectSheetOpen] = useState(false)
  const [selectedProject, setSelectedProject] = useState<ProjectStats | null>(null)

  // SSE handlers
  const handleQuestion = useCallback(() => {
    recordQuestion()
    fetchGoals()
    if (selectedGoal) {
      fetchGoalDetail(selectedGoal.id)
    }
  }, [recordQuestion, fetchGoals, selectedGoal, fetchGoalDetail])

  const handleAnswered = useCallback(() => {
    recordAnswered()
    fetchGoals()
    if (selectedGoal) {
      fetchGoalDetail(selectedGoal.id)
    }
  }, [recordAnswered, fetchGoals, selectedGoal, fetchGoalDetail])

  const handleExecutorStarted = useCallback((data: { goal_id: string; session_id: string }) => {
    recordExecutorStarted(data.goal_id, data.session_id)
    fetchGoals()
    if (selectedGoal) {
      fetchGoalDetail(selectedGoal.id)
    }
  }, [recordExecutorStarted, fetchGoals, selectedGoal, fetchGoalDetail])

  const handleExecutorStopped = useCallback((data: { goal_id: string; session_id: string }) => {
    recordExecutorStopped(data.goal_id, data.session_id)
    fetchGoals()
    if (selectedGoal) {
      fetchGoalDetail(selectedGoal.id)
    }
  }, [recordExecutorStopped, fetchGoals, selectedGoal, fetchGoalDetail])

  const handleGoalUpdated = useCallback((data: { goal_id: string }) => {
    recordGoalUpdated(data.goal_id)
    fetchGoals()
    if (selectedGoal && data.goal_id === selectedGoal.id) {
      fetchGoalDetail(selectedGoal.id)
    }
  }, [recordGoalUpdated, fetchGoals, selectedGoal, fetchGoalDetail])

  const handleGoalIced = useCallback((data: { goal_id: string }) => {
    recordGoalIced(data.goal_id)
    fetchGoals()
  }, [recordGoalIced, fetchGoals])

  const handleGoalCompleted = useCallback((data: { goal_id: string }) => {
    recordGoalCompleted(data.goal_id)
    fetchGoals()
  }, [recordGoalCompleted, fetchGoals])

  const handleRegistryUpdated = useCallback(() => {
    fetchGoals()
  }, [fetchGoals])

  const { connected } = useSSE({
    onQuestion: handleQuestion,
    onAnswered: handleAnswered,
    onExecutorStarted: handleExecutorStarted,
    onExecutorStopped: handleExecutorStopped,
    onGoalUpdated: handleGoalUpdated,
    onGoalIced: handleGoalIced,
    onGoalCompleted: handleGoalCompleted,
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

  const handleProjectClick = useCallback((project: ProjectStats) => {
    setSelectedProject(project)
    setProjectSheetOpen(true)
  }, [])

  const handleProjectSheetClose = useCallback(() => {
    setProjectSheetOpen(false)
    setTimeout(() => {
      setSelectedProject(null)
    }, 300)
  }, [])

  const handleGoalClickFromProject = useCallback((id: string) => {
    // Close project sheet, open goal sheet
    handleProjectSheetClose()
    setTimeout(() => {
      handleGoalClick(id)
    }, 300)
  }, [handleProjectSheetClose, handleGoalClick])

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
                activities={activities}
                onGoalClick={handleGoalClick}
              />
            }
          />
          <Route
            path="/projects"
            element={<Projects goals={goals} loading={loading} onProjectClick={handleProjectClick} />}
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

      {/* Project Detail Sheet */}
      <ProjectSheet
        open={projectSheetOpen}
        onOpenChange={(open) => {
          if (!open) handleProjectSheetClose()
        }}
        project={selectedProject}
        goals={goals}
        onGoalClick={handleGoalClickFromProject}
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
