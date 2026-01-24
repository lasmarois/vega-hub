import { useEffect, useState } from 'react'

interface Question {
  id: string
  goal_id: number
  session_id: string
  question: string
  options?: { label: string; description?: string }[]
  created_at: string
}

function App() {
  const [questions, setQuestions] = useState<Question[]>([])
  const [connected, setConnected] = useState(false)
  const [answerText, setAnswerText] = useState<Record<string, string>>({})

  useEffect(() => {
    // Connect to SSE
    const eventSource = new EventSource('/api/events')

    eventSource.addEventListener('connected', () => {
      setConnected(true)
    })

    eventSource.addEventListener('question', (e) => {
      const question = JSON.parse(e.data) as Question
      setQuestions((prev) => {
        if (prev.find((q) => q.id === question.id)) return prev
        return [...prev, question]
      })
    })

    eventSource.addEventListener('answered', (e) => {
      const { id } = JSON.parse(e.data)
      setQuestions((prev) => prev.filter((q) => q.id !== id))
    })

    eventSource.onerror = () => {
      setConnected(false)
    }

    return () => eventSource.close()
  }, [])

  const handleAnswer = async (id: string) => {
    const answer = answerText[id]
    if (!answer?.trim()) return

    try {
      const res = await fetch(`/api/answer/${id}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ answer }),
      })

      if (res.ok) {
        setAnswerText((prev) => {
          const next = { ...prev }
          delete next[id]
          return next
        })
      }
    } catch (err) {
      console.error('Failed to submit answer:', err)
    }
  }

  return (
    <div className="min-h-screen bg-gray-900 text-gray-100 p-8">
      <header className="mb-8">
        <h1 className="text-3xl font-bold flex items-center gap-3">
          vega-hub
          <span
            className={`inline-block w-3 h-3 rounded-full ${
              connected ? 'bg-green-500' : 'bg-red-500'
            }`}
          />
        </h1>
        <p className="text-gray-400 mt-1">
          {questions.length} pending question{questions.length !== 1 ? 's' : ''}
        </p>
      </header>

      {questions.length === 0 ? (
        <div className="text-center text-gray-500 py-16">
          <p className="text-xl">No pending questions</p>
          <p className="mt-2">Waiting for executors to ask...</p>
        </div>
      ) : (
        <div className="space-y-6">
          {questions.map((q) => (
            <div
              key={q.id}
              className="bg-gray-800 rounded-lg p-6 border border-gray-700"
            >
              <div className="flex items-center gap-2 text-sm text-gray-400 mb-3">
                <span className="bg-blue-600 text-white px-2 py-0.5 rounded">
                  Goal #{q.goal_id}
                </span>
                <span>Session: {q.session_id.slice(0, 8)}...</span>
                <span>
                  {new Date(q.created_at).toLocaleTimeString()}
                </span>
              </div>

              <p className="text-xl mb-4">{q.question}</p>

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
                        <span className="text-gray-400 ml-2">
                          - {opt.description}
                        </span>
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
      )}
    </div>
  )
}

export default App
