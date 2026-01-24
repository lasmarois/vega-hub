import { useEffect, useState } from 'react'

interface Question {
  id: string
  goal_id: number
  session_id: string
  question: string
  options?: { label: string; description?: string }[]
  created_at: string
}

interface GoalSummary {
  id: number
  title: string
  projects: string[]
  status: string
  phase: string
  executor_status: string
  pending_questions: number
  active_executors: number
}

interface PhaseDetail {
  number: number
  title: string
  tasks: { description: string; completed: boolean }[]
  status: string
}

interface GoalDetail {
  id: number
  title: string
  projects: string[]
  status: string
  phase: string
  overview: string
  phases: PhaseDetail[]
  acceptance: string[]
  notes: string[]
  executor_status: string
  pending_questions: Question[]
  active_executors: { session_id: string; goal_id: number; cwd: string; started_at: string }[]
}

interface GoalStatus {
  current_phase: string
  recent_actions: string[]
  progress_log: string
  task_plan: string
  findings: string
  has_worktree: boolean
  worktree_path: string
  phase_progress: { number: number; title: string; status: string; tasks_total: number; tasks_done: number }[]
}

function App() {
  const [goals, setGoals] = useState<GoalSummary[]>([])
  const [selectedGoal, setSelectedGoal] = useState<GoalDetail | null>(null)
  const [goalStatus, setGoalStatus] = useState<GoalStatus | null>(null)
  const [connected, setConnected] = useState(false)
  const [answerText, setAnswerText] = useState<Record<string, string>>({})
  const [loading, setLoading] = useState(true)
  const [spawnContext, setSpawnContext] = useState('')
  const [showSpawnModal, setShowSpawnModal] = useState(false)
  const [spawning, setSpawning] = useState(false)
  const [executorOutput, setExecutorOutput] = useState<string>('')
  const [showOutput, setShowOutput] = useState(false)
  const [outputLoading, setOutputLoading] = useState(false)

  // Fetch goals on mount
  useEffect(() => {
    fetchGoals()
  }, [])

  // SSE connection for real-time updates
  useEffect(() => {
    const eventSource = new EventSource('/api/events')

    eventSource.addEventListener('connected', () => {
      setConnected(true)
    })

    eventSource.addEventListener('question', () => {
      // Refresh goals to update pending counts
      fetchGoals()
      // If we have a selected goal, refresh it
      if (selectedGoal) {
        fetchGoalDetail(selectedGoal.id)
      }
    })

    eventSource.addEventListener('answered', (e) => {
      const { id } = JSON.parse(e.data)
      setAnswerText((prev) => {
        const next = { ...prev }
        delete next[id]
        return next
      })
      fetchGoals()
      if (selectedGoal) {
        fetchGoalDetail(selectedGoal.id)
      }
    })

    eventSource.addEventListener('executor_started', () => {
      fetchGoals()
      if (selectedGoal) {
        fetchGoalDetail(selectedGoal.id)
      }
    })

    eventSource.addEventListener('executor_stopped', (e) => {
      const data = JSON.parse(e.data)
      // Show output from the stopped executor
      if (data.output && data.goal_id === selectedGoal?.id) {
        setExecutorOutput(data.output)
        setShowOutput(true)
      }
      fetchGoals()
      if (selectedGoal) {
        fetchGoalDetail(selectedGoal.id)
      }
    })

    eventSource.onerror = () => {
      setConnected(false)
    }

    return () => eventSource.close()
  }, [selectedGoal])

  const fetchGoals = async () => {
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
  }

  const fetchGoalDetail = async (id: number) => {
    try {
      const res = await fetch(`/api/goals/${id}`)
      if (res.ok) {
        const data = await res.json()
        setSelectedGoal(data)
        // Also fetch status from planning files
        fetchGoalStatus(id)
      }
    } catch (err) {
      console.error('Failed to fetch goal detail:', err)
    }
  }

  const fetchGoalStatus = async (id: number) => {
    try {
      const res = await fetch(`/api/goals/${id}/status`)
      if (res.ok) {
        const data = await res.json()
        setGoalStatus(data)
      }
    } catch (err) {
      console.error('Failed to fetch goal status:', err)
    }
  }

  const fetchExecutorOutput = async (id: number) => {
    setOutputLoading(true)
    try {
      const res = await fetch(`/api/goals/${id}/output`)
      if (res.ok) {
        const data = await res.json()
        if (data.available) {
          setExecutorOutput(data.output)
          setShowOutput(true)
        } else {
          setExecutorOutput('No output available')
          setShowOutput(true)
        }
      }
    } catch (err) {
      console.error('Failed to fetch executor output:', err)
    } finally {
      setOutputLoading(false)
    }
  }

  const handleSpawnExecutor = async () => {
    if (!selectedGoal) return
    setSpawning(true)
    try {
      const res = await fetch(`/api/goals/${selectedGoal.id}/spawn`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ context: spawnContext || undefined }),
      })

      const data = await res.json()

      if (data.success) {
        setShowSpawnModal(false)
        setSpawnContext('')
        setExecutorOutput('')
        setShowOutput(true) // Auto-show output panel
        // Refresh after spawn
        fetchGoals()
        fetchGoalDetail(selectedGoal.id)
      } else {
        alert('Failed to spawn executor: ' + data.message)
      }
    } catch (err) {
      console.error('Failed to spawn executor:', err)
      alert('Failed to spawn executor')
    } finally {
      setSpawning(false)
    }
  }

  // Auto-refresh goal status while executor is running
  useEffect(() => {
    if (!selectedGoal || selectedGoal.executor_status !== 'running') return

    const interval = setInterval(() => {
      fetchGoalStatus(selectedGoal.id)
    }, 5000) // Refresh every 5 seconds

    return () => clearInterval(interval)
  }, [selectedGoal])

  // Auto-refresh output while executor is running and output panel is shown
  useEffect(() => {
    if (!selectedGoal || selectedGoal.executor_status !== 'running' || !showOutput) return

    const fetchOutput = async () => {
      try {
        const res = await fetch(`/api/goals/${selectedGoal.id}/output?tail=100`)
        if (res.ok) {
          const data = await res.json()
          if (data.available) {
            setExecutorOutput(data.output)
          }
        }
      } catch (err) {
        // Ignore errors during polling
      }
    }

    // Fetch immediately
    fetchOutput()

    // Then poll every 2 seconds
    const interval = setInterval(fetchOutput, 2000)

    return () => clearInterval(interval)
  }, [selectedGoal, showOutput])

  const handleAnswer = async (questionId: string) => {
    const answer = answerText[questionId]
    if (!answer?.trim()) return

    try {
      const res = await fetch(`/api/answer/${questionId}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ answer }),
      })

      if (res.ok) {
        setAnswerText((prev) => {
          const next = { ...prev }
          delete next[questionId]
          return next
        })
      }
    } catch (err) {
      console.error('Failed to submit answer:', err)
    }
  }

  const totalPendingQuestions = goals.reduce((sum, g) => sum + g.pending_questions, 0)

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'running':
        return 'bg-green-500'
      case 'waiting':
        return 'bg-red-500'
      case 'stopped':
        return 'bg-gray-500'
      default:
        return 'bg-gray-600'
    }
  }

  const getStatusLabel = (status: string) => {
    switch (status) {
      case 'running':
        return 'RUNNING'
      case 'waiting':
        return 'WAITING'
      case 'stopped':
        return 'STOPPED'
      default:
        return 'IDLE'
    }
  }

  return (
    <div className="min-h-screen bg-gray-900 text-gray-100">
      {/* Header */}
      <header className="border-b border-gray-800 p-4">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <h1 className="text-2xl font-bold">vega-hub</h1>
            <span
              className={`inline-block w-3 h-3 rounded-full ${
                connected ? 'bg-green-500' : 'bg-red-500'
              }`}
              title={connected ? 'Connected' : 'Disconnected'}
            />
          </div>
          {totalPendingQuestions > 0 && (
            <div className="flex items-center gap-2 bg-red-600 px-3 py-1 rounded-full">
              <span className="text-xl">🔔</span>
              <span className="font-medium">{totalPendingQuestions}</span>
            </div>
          )}
        </div>
      </header>

      <div className="flex h-[calc(100vh-65px)]">
        {/* Goal List - Left Panel */}
        <div className="w-1/3 border-r border-gray-800 overflow-y-auto">
          <div className="p-4">
            <h2 className="text-lg font-semibold text-gray-400 mb-4">Active Goals</h2>

            {loading ? (
              <div className="text-gray-500 text-center py-8">Loading goals...</div>
            ) : goals.length === 0 ? (
              <div className="text-gray-500 text-center py-8">No goals found</div>
            ) : (
              <div className="space-y-3">
                {goals.filter(g => g.status === 'active').map((goal) => (
                  <div
                    key={goal.id}
                    onClick={() => fetchGoalDetail(goal.id)}
                    className={`p-4 rounded-lg border cursor-pointer transition-colors ${
                      selectedGoal?.id === goal.id
                        ? 'border-blue-500 bg-blue-900/20'
                        : 'border-gray-700 bg-gray-800 hover:border-gray-600'
                    }`}
                  >
                    <div className="flex items-start justify-between">
                      <div className="flex items-center gap-2">
                        <span className={`w-2 h-2 rounded-full ${getStatusColor(goal.executor_status)}`} />
                        <span className="font-medium">Goal #{goal.id}</span>
                      </div>
                      <div className="flex items-center gap-2">
                        <span className={`text-xs px-2 py-0.5 rounded ${
                          goal.executor_status === 'waiting' ? 'bg-red-600' :
                          goal.executor_status === 'running' ? 'bg-green-600' :
                          'bg-gray-600'
                        }`}>
                          {getStatusLabel(goal.executor_status)}
                        </span>
                        {goal.pending_questions > 0 && (
                          <span className="bg-red-500 text-white text-xs px-2 py-0.5 rounded-full">
                            {goal.pending_questions}
                          </span>
                        )}
                      </div>
                    </div>
                    <h3 className="mt-2 text-sm">{goal.title}</h3>
                    <div className="mt-2 flex items-center gap-3 text-xs text-gray-500">
                      <span>Phase: {goal.phase}</span>
                      {goal.projects.length > 0 && (
                        <span>{goal.projects.join(', ')}</span>
                      )}
                    </div>
                  </div>
                ))}

                {/* Completed goals section */}
                {goals.filter(g => g.status === 'completed').length > 0 && (
                  <>
                    <h2 className="text-lg font-semibold text-gray-400 mt-6 mb-4">Completed</h2>
                    {goals.filter(g => g.status === 'completed').slice(0, 5).map((goal) => (
                      <div
                        key={goal.id}
                        onClick={() => fetchGoalDetail(goal.id)}
                        className={`p-4 rounded-lg border cursor-pointer transition-colors opacity-60 ${
                          selectedGoal?.id === goal.id
                            ? 'border-blue-500 bg-blue-900/20'
                            : 'border-gray-700 bg-gray-800 hover:border-gray-600'
                        }`}
                      >
                        <div className="flex items-center gap-2">
                          <span className="text-green-500">✓</span>
                          <span className="font-medium">Goal #{goal.id}</span>
                        </div>
                        <h3 className="mt-2 text-sm">{goal.title}</h3>
                      </div>
                    ))}
                  </>
                )}
              </div>
            )}
          </div>
        </div>

        {/* Goal Detail - Right Panel */}
        <div className="flex-1 overflow-y-auto">
          {selectedGoal ? (
            <div className="p-6">
              {/* Goal Header */}
              <div className="mb-6">
                <div className="flex items-center gap-3 mb-2">
                  <span className={`w-3 h-3 rounded-full ${getStatusColor(selectedGoal.executor_status)}`} />
                  <h2 className="text-2xl font-bold">Goal #{selectedGoal.id}: {selectedGoal.title}</h2>
                </div>
                <div className="flex items-center gap-4 text-sm text-gray-400">
                  <span className={`px-2 py-0.5 rounded ${
                    selectedGoal.executor_status === 'waiting' ? 'bg-red-600' :
                    selectedGoal.executor_status === 'running' ? 'bg-green-600' :
                    'bg-gray-600'
                  }`}>
                    {getStatusLabel(selectedGoal.executor_status)}
                  </span>
                  <span>Phase: {selectedGoal.phase}</span>
                  {selectedGoal.projects.length > 0 && (
                    <span>Projects: {selectedGoal.projects.join(', ')}</span>
                  )}
                </div>
              </div>

              {/* Pending Questions */}
              {selectedGoal.pending_questions && selectedGoal.pending_questions.length > 0 && (
                <div className="mb-6">
                  <h3 className="text-lg font-semibold text-red-400 mb-3 flex items-center gap-2">
                    <span>🔔</span>
                    Pending Questions ({selectedGoal.pending_questions.length})
                  </h3>
                  <div className="space-y-4">
                    {selectedGoal.pending_questions.map((q) => (
                      <div
                        key={q.id}
                        className="bg-red-900/20 border border-red-700 rounded-lg p-4"
                      >
                        <div className="flex items-center gap-2 text-sm text-gray-400 mb-2">
                          <span>Session: {q.session_id.slice(0, 8)}...</span>
                          <span>{new Date(q.created_at).toLocaleTimeString()}</span>
                        </div>
                        <p className="text-lg mb-4">{q.question}</p>

                        {q.options && q.options.length > 0 && (
                          <div className="mb-4 space-y-2">
                            {q.options.map((opt, i) => (
                              <button
                                key={i}
                                onClick={() =>
                                  setAnswerText((prev) => ({ ...prev, [q.id]: opt.label }))
                                }
                                className={`block w-full text-left p-3 rounded border ${
                                  answerText[q.id] === opt.label
                                    ? 'border-blue-500 bg-blue-900/30'
                                    : 'border-gray-600 hover:border-gray-500'
                                }`}
                              >
                                <span className="font-medium">{opt.label}</span>
                                {opt.description && (
                                  <span className="text-gray-400 ml-2">- {opt.description}</span>
                                )}
                              </button>
                            ))}
                          </div>
                        )}

                        <div className="flex gap-3">
                          <input
                            type="text"
                            value={answerText[q.id] || ''}
                            onChange={(e) =>
                              setAnswerText((prev) => ({ ...prev, [q.id]: e.target.value }))
                            }
                            placeholder="Type your answer..."
                            className="flex-1 bg-gray-700 border border-gray-600 rounded px-4 py-2 focus:outline-none focus:border-blue-500"
                            onKeyDown={(e) => {
                              if (e.key === 'Enter') handleAnswer(q.id)
                            }}
                          />
                          <button
                            onClick={() => handleAnswer(q.id)}
                            disabled={!answerText[q.id]?.trim()}
                            className="bg-blue-600 hover:bg-blue-700 disabled:bg-gray-600 disabled:cursor-not-allowed px-6 py-2 rounded font-medium"
                          >
                            Answer
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Executor Control Panel */}
              <div className="mb-6">
                <h3 className="text-lg font-semibold text-gray-300 mb-3">Executor Control</h3>
                <div className="bg-gray-800 border border-gray-700 rounded-lg p-4">
                  {/* Status display */}
                  {goalStatus && goalStatus.has_worktree ? (
                    <div className="mb-4">
                      <div className="flex items-center gap-2 mb-2">
                        <span className="text-sm text-gray-400">Current Phase:</span>
                        <span className="text-sm font-medium">{goalStatus.current_phase || 'Unknown'}</span>
                      </div>

                      {/* Phase progress bars */}
                      {goalStatus.phase_progress && goalStatus.phase_progress.length > 0 && (
                        <div className="space-y-2 mb-4">
                          {goalStatus.phase_progress.map((p) => (
                            <div key={p.number} className="flex items-center gap-2 text-sm">
                              <span className="w-20 text-gray-400">Phase {p.number}</span>
                              <div className="flex-1 bg-gray-700 rounded-full h-2">
                                <div
                                  className={`h-2 rounded-full ${
                                    p.status === 'complete' ? 'bg-green-500' :
                                    p.status === 'in_progress' ? 'bg-blue-500' :
                                    'bg-gray-600'
                                  }`}
                                  style={{ width: `${p.tasks_total > 0 ? (p.tasks_done / p.tasks_total) * 100 : 0}%` }}
                                />
                              </div>
                              <span className="text-gray-500 w-16 text-right">
                                {p.tasks_done}/{p.tasks_total}
                              </span>
                            </div>
                          ))}
                        </div>
                      )}

                      {/* Recent actions */}
                      {goalStatus.recent_actions && goalStatus.recent_actions.length > 0 && (
                        <div className="mb-4">
                          <h4 className="text-sm text-gray-400 mb-2">Recent Actions:</h4>
                          <ul className="text-sm text-gray-500 space-y-1">
                            {goalStatus.recent_actions.map((action, i) => (
                              <li key={i} className="flex items-start gap-2">
                                <span className="text-gray-600">•</span>
                                <span>{action}</span>
                              </li>
                            ))}
                          </ul>
                        </div>
                      )}
                    </div>
                  ) : (
                    <div className="mb-4 text-sm text-gray-500">
                      No worktree found - executor may not have started yet
                    </div>
                  )}

                  {/* Active Executors */}
                  {selectedGoal.active_executors && selectedGoal.active_executors.length > 0 ? (
                    <div className="mb-4">
                      <h4 className="text-sm text-gray-400 mb-2">
                        Active Sessions ({selectedGoal.active_executors.length})
                      </h4>
                      <div className="space-y-2">
                        {selectedGoal.active_executors.map((e) => (
                          <div
                            key={e.session_id}
                            className="bg-gray-700/50 rounded p-2 text-sm"
                          >
                            <div className="flex items-center gap-2">
                              <span className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />
                              <span className="font-mono text-xs">{e.session_id.slice(0, 16)}...</span>
                            </div>
                            <div className="mt-1 text-gray-500 text-xs">
                              Started: {new Date(e.started_at).toLocaleString()}
                            </div>
                          </div>
                        ))}
                      </div>
                    </div>
                  ) : (
                    <div className="mb-4 text-sm text-gray-500">
                      No active executors
                    </div>
                  )}

                  {/* Action buttons */}
                  <div className="flex gap-3">
                    {selectedGoal.status === 'active' && (
                      <button
                        onClick={() => setShowSpawnModal(true)}
                        disabled={selectedGoal.executor_status === 'running' || selectedGoal.executor_status === 'waiting'}
                        className="flex items-center gap-2 bg-green-600 hover:bg-green-700 disabled:bg-gray-600 disabled:cursor-not-allowed px-4 py-2 rounded font-medium"
                      >
                        <span>▶</span>
                        <span>
                          {selectedGoal.executor_status === 'running' ? 'Executor Running' :
                           selectedGoal.executor_status === 'waiting' ? 'Waiting for Answer' :
                           'Resume Executor'}
                        </span>
                      </button>
                    )}
                    <button
                      onClick={() => fetchExecutorOutput(selectedGoal.id)}
                      disabled={outputLoading}
                      className="flex items-center gap-2 bg-gray-600 hover:bg-gray-500 disabled:bg-gray-700 px-4 py-2 rounded font-medium"
                    >
                      <span>📄</span>
                      <span>{outputLoading ? 'Loading...' : 'View Output'}</span>
                    </button>
                  </div>

                  {/* Executor Output Panel */}
                  {showOutput && (
                    <div className="mt-4 bg-gray-900 border border-gray-700 rounded-lg">
                      <div className="flex items-center justify-between p-3 border-b border-gray-700">
                        <h4 className="text-sm font-medium text-gray-300">Executor Output</h4>
                        <button
                          onClick={() => setShowOutput(false)}
                          className="text-gray-500 hover:text-gray-300"
                        >
                          ✕
                        </button>
                      </div>
                      <pre className="p-4 text-sm text-gray-300 overflow-x-auto max-h-96 overflow-y-auto whitespace-pre-wrap font-mono">
                        {executorOutput || 'No output available'}
                      </pre>
                    </div>
                  )}
                </div>
              </div>

              {/* Spawn Modal */}
              {showSpawnModal && (
                <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
                  <div className="bg-gray-800 border border-gray-700 rounded-lg p-6 w-full max-w-lg mx-4">
                    <h3 className="text-lg font-semibold mb-4">Resume Executor for Goal #{selectedGoal.id}</h3>

                    <div className="mb-4">
                      <label className="block text-sm text-gray-400 mb-2">
                        Additional Context (optional)
                      </label>
                      <textarea
                        value={spawnContext}
                        onChange={(e) => setSpawnContext(e.target.value)}
                        placeholder="Add any additional instructions or context for the executor..."
                        className="w-full h-32 bg-gray-700 border border-gray-600 rounded px-4 py-2 text-sm focus:outline-none focus:border-blue-500 resize-none"
                      />
                      <p className="text-xs text-gray-500 mt-1">
                        Default: "Continue working on your assigned goal."
                      </p>
                    </div>

                    <div className="flex justify-end gap-3">
                      <button
                        onClick={() => {
                          setShowSpawnModal(false)
                          setSpawnContext('')
                        }}
                        className="px-4 py-2 rounded border border-gray-600 hover:border-gray-500"
                      >
                        Cancel
                      </button>
                      <button
                        onClick={handleSpawnExecutor}
                        disabled={spawning}
                        className="bg-green-600 hover:bg-green-700 disabled:bg-gray-600 px-4 py-2 rounded font-medium flex items-center gap-2"
                      >
                        {spawning ? (
                          <>
                            <span className="animate-spin">⟳</span>
                            <span>Spawning...</span>
                          </>
                        ) : (
                          <>
                            <span>▶</span>
                            <span>Spawn Executor</span>
                          </>
                        )}
                      </button>
                    </div>
                  </div>
                </div>
              )}

              {/* Overview */}
              {selectedGoal.overview && (
                <div className="mb-6">
                  <h3 className="text-lg font-semibold text-gray-300 mb-3">Overview</h3>
                  <p className="text-gray-400 whitespace-pre-wrap">{selectedGoal.overview}</p>
                </div>
              )}

              {/* Phases */}
              {selectedGoal.phases && selectedGoal.phases.length > 0 && (
                <div className="mb-6">
                  <h3 className="text-lg font-semibold text-gray-300 mb-3">Phases</h3>
                  <div className="space-y-4">
                    {selectedGoal.phases.map((phase) => (
                      <div
                        key={phase.number}
                        className="bg-gray-800 border border-gray-700 rounded-lg p-4"
                      >
                        <div className="flex items-center justify-between mb-2">
                          <h4 className="font-medium">
                            Phase {phase.number}: {phase.title}
                          </h4>
                          <span className={`text-xs px-2 py-0.5 rounded ${
                            phase.status === 'complete' ? 'bg-green-600' :
                            phase.status === 'in_progress' ? 'bg-blue-600' :
                            'bg-gray-600'
                          }`}>
                            {phase.status}
                          </span>
                        </div>
                        {phase.tasks && phase.tasks.length > 0 && (
                          <ul className="space-y-1 text-sm text-gray-400">
                            {phase.tasks.map((task, i) => (
                              <li key={i} className="flex items-center gap-2">
                                <span className={task.completed ? 'text-green-500' : 'text-gray-600'}>
                                  {task.completed ? '✓' : '○'}
                                </span>
                                <span className={task.completed ? 'line-through opacity-60' : ''}>
                                  {task.description}
                                </span>
                              </li>
                            ))}
                          </ul>
                        )}
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Acceptance Criteria */}
              {selectedGoal.acceptance && selectedGoal.acceptance.length > 0 && (
                <div className="mb-6">
                  <h3 className="text-lg font-semibold text-gray-300 mb-3">Acceptance Criteria</h3>
                  <ul className="space-y-1 text-sm text-gray-400">
                    {selectedGoal.acceptance.map((item, i) => (
                      <li key={i} className="flex items-center gap-2">
                        <span className="text-gray-600">○</span>
                        <span>{item}</span>
                      </li>
                    ))}
                  </ul>
                </div>
              )}

              {/* Notes */}
              {selectedGoal.notes && selectedGoal.notes.length > 0 && (
                <div className="mb-6">
                  <h3 className="text-lg font-semibold text-gray-300 mb-3">Notes</h3>
                  <ul className="space-y-1 text-sm text-gray-400">
                    {selectedGoal.notes.map((note, i) => (
                      <li key={i}>• {note}</li>
                    ))}
                  </ul>
                </div>
              )}
            </div>
          ) : (
            <div className="flex items-center justify-center h-full text-gray-500">
              <div className="text-center">
                <p className="text-xl mb-2">Select a goal to view details</p>
                <p>Click on a goal from the list to see its phases, questions, and status</p>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

export default App
