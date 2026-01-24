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

function App() {
  const [goals, setGoals] = useState<GoalSummary[]>([])
  const [selectedGoal, setSelectedGoal] = useState<GoalDetail | null>(null)
  const [connected, setConnected] = useState(false)
  const [answerText, setAnswerText] = useState<Record<string, string>>({})
  const [loading, setLoading] = useState(true)

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

    eventSource.addEventListener('executor_stopped', () => {
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
      }
    } catch (err) {
      console.error('Failed to fetch goal detail:', err)
    }
  }

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

              {/* Active Executors */}
              {selectedGoal.active_executors && selectedGoal.active_executors.length > 0 && (
                <div className="mb-6">
                  <h3 className="text-lg font-semibold text-gray-300 mb-3">
                    Active Executors ({selectedGoal.active_executors.length})
                  </h3>
                  <div className="space-y-2">
                    {selectedGoal.active_executors.map((e) => (
                      <div
                        key={e.session_id}
                        className="bg-gray-800 border border-gray-700 rounded-lg p-3 text-sm"
                      >
                        <div className="flex items-center gap-2">
                          <span className="w-2 h-2 rounded-full bg-green-500" />
                          <span className="font-mono">{e.session_id.slice(0, 16)}...</span>
                        </div>
                        <div className="mt-1 text-gray-500">
                          <span>Started: {new Date(e.started_at).toLocaleString()}</span>
                        </div>
                      </div>
                    ))}
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
