import { useState, useEffect, useRef, useCallback } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  Play,
  Square,
  MessageSquare,
  Send,
  User,
  Bot,
  Clock,
  ChevronDown,
  ChevronUp,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { useSSE } from '@/hooks/useSSE'
import type { ChatMessage, Question } from '@/lib/types'

interface ChatThreadProps {
  goalId: string
  pendingQuestions: Question[]
  onAnswerSubmit: (questionId: string, answer: string) => void
}

export function ChatThread({ goalId, pendingQuestions, onAnswerSubmit }: ChatThreadProps) {
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [loading, setLoading] = useState(true)
  const [answerText, setAnswerText] = useState<Record<string, string>>({})
  const [userMessage, setUserMessage] = useState('')
  const [sendingMessage, setSendingMessage] = useState(false)
  const [historyMode, setHistoryMode] = useState<'session' | 'all'>('all')
  const [currentSessionId, setCurrentSessionId] = useState<string | null>(null)
  const [executorRunning, setExecutorRunning] = useState(false)
  const scrollRef = useRef<HTMLDivElement>(null)

  // Determine if executor is running from messages
  useEffect(() => {
    if (messages.length === 0) {
      setExecutorRunning(false)
      return
    }
    // Find the most recent session event
    const sessionEvents = messages.filter(m => m.type === 'session_start' || m.type === 'session_stop')
    if (sessionEvents.length === 0) {
      setExecutorRunning(false)
      return
    }
    const lastEvent = sessionEvents[sessionEvents.length - 1]
    setExecutorRunning(lastEvent.type === 'session_start')
  }, [messages])

  // Fetch chat history
  const fetchChat = useCallback(async () => {
    try {
      const url = historyMode === 'session' && currentSessionId
        ? `/api/goals/${goalId}/chat?session=${currentSessionId}`
        : `/api/goals/${goalId}/chat`
      const res = await fetch(url)
      if (res.ok) {
        const data = await res.json()
        setMessages(data || [])
        // Detect current session from latest session_start
        if (data && data.length > 0) {
          const lastStart = [...data].reverse().find((m: ChatMessage) => m.type === 'session_start')
          if (lastStart) {
            setCurrentSessionId(lastStart.session_id)
          }
        }
      }
    } catch (err) {
      console.error('Failed to fetch chat:', err)
    } finally {
      setLoading(false)
    }
  }, [goalId, historyMode, currentSessionId])

  // Initial fetch
  useEffect(() => {
    fetchChat()
  }, [fetchChat])

  // SSE handlers for real-time updates
  useSSE({
    onExecutorStarted: (data) => {
      if (data.goal_id === goalId) {
        setExecutorRunning(true)
        fetchChat()
      }
    },
    onExecutorStopped: (data) => {
      if (data.goal_id === goalId) {
        setExecutorRunning(false)
        fetchChat()
      }
    },
    onQuestion: () => {
      // Questions come through pendingQuestions prop, but refetch for history
      fetchChat()
    },
    onAnswered: () => {
      fetchChat()
    },
    onUserMessage: (data) => {
      if (data.goal_id === goalId) {
        fetchChat()
      }
    },
  })

  // Auto-scroll to bottom when new messages arrive
  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [messages, pendingQuestions])

  // Send user message
  const handleSendMessage = async () => {
    if (!userMessage.trim()) return
    setSendingMessage(true)
    try {
      const res = await fetch(`/api/goals/${goalId}/messages`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content: userMessage }),
      })
      if (res.ok) {
        setUserMessage('')
        // Refetch chat to show the new message
        const chatRes = await fetch(`/api/goals/${goalId}/chat`)
        if (chatRes.ok) {
          setMessages(await chatRes.json() || [])
        }
      }
    } catch (err) {
      console.error('Failed to send message:', err)
    } finally {
      setSendingMessage(false)
    }
  }

  // Handle answer submission
  const handleAnswer = (questionId: string) => {
    const answer = answerText[questionId]
    if (!answer?.trim()) return
    onAnswerSubmit(questionId, answer)
    setAnswerText((prev) => {
      const next = { ...prev }
      delete next[questionId]
      return next
    })
  }

  // Render a single chat message
  const renderMessage = (msg: ChatMessage) => {
    const isSystemMessage = msg.type === 'session_start' || msg.type === 'session_stop'

    if (isSystemMessage) {
      return (
        <SystemMessage key={msg.id} message={msg} />
      )
    }

    if (msg.type === 'question') {
      return (
        <QuestionBubble
          key={msg.id}
          message={msg}
          isPending={msg.pending && !msg.answer}
        />
      )
    }

    if (msg.type === 'user_message' || msg.type === 'user_message_delivered') {
      return (
        <UserMessageBubble
          key={msg.id}
          message={msg}
          delivered={msg.type === 'user_message_delivered'}
        />
      )
    }

    // Default: activity or other
    return (
      <ActivityMessage key={msg.id} message={msg} />
    )
  }

  return (
    <div className="flex flex-col h-full">
      {/* Header with history toggle */}
      <div className="flex items-center justify-between pb-2 border-b">
        <span className="text-sm font-medium">Chat</span>
        <Tabs value={historyMode} onValueChange={(v) => setHistoryMode(v as 'session' | 'all')}>
          <TabsList className="h-7">
            <TabsTrigger value="all" className="text-xs px-2 h-6">All History</TabsTrigger>
            <TabsTrigger value="session" className="text-xs px-2 h-6">This Session</TabsTrigger>
          </TabsList>
        </Tabs>
      </div>

      {/* Messages area */}
      <ScrollArea className="flex-1 pr-2" ref={scrollRef}>
        <div className="space-y-3 py-3">
          {loading ? (
            <div className="text-center text-muted-foreground py-8">Loading...</div>
          ) : messages.length === 0 && pendingQuestions.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-8 text-center text-muted-foreground">
              <MessageSquare className="h-12 w-12 mb-4 opacity-50" />
              <p>No messages yet</p>
              <p className="text-sm">Questions and messages will appear here</p>
            </div>
          ) : (
            <>
              {messages.map(renderMessage)}
              {/* Pending questions from live state */}
              {pendingQuestions.map((q) => (
                <PendingQuestionInput
                  key={q.id}
                  question={q}
                  answerText={answerText[q.id] || ''}
                  onAnswerChange={(text) => setAnswerText((prev) => ({ ...prev, [q.id]: text }))}
                  onSubmit={() => handleAnswer(q.id)}
                />
              ))}
            </>
          )}
        </div>
      </ScrollArea>

      {/* Message input */}
      <div className="pt-3 border-t">
        <div className="flex gap-2">
          <Input
            placeholder={executorRunning
              ? "Send a message to the executor..."
              : "Queue a message for the next session..."
            }
            value={userMessage}
            onChange={(e) => setUserMessage(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault()
                handleSendMessage()
              }
            }}
            disabled={sendingMessage}
            className={cn(!executorRunning && 'border-dashed')}
          />
          <Button
            size="icon"
            onClick={handleSendMessage}
            disabled={!userMessage.trim() || sendingMessage}
            variant={executorRunning ? 'default' : 'outline'}
          >
            <Send className="h-4 w-4" />
          </Button>
        </div>
        <p className="text-xs text-muted-foreground mt-1">
          {executorRunning ? (
            <>Message will be delivered when the executor pauses</>
          ) : (
            <span className="text-amber-600 dark:text-amber-500">
              No executor running — message will be queued and delivered when executor starts
            </span>
          )}
        </p>
      </div>
    </div>
  )
}

// System message (session start/stop)
function SystemMessage({ message }: { message: ChatMessage }) {
  const [expanded, setExpanded] = useState(false)
  const output = message.type === 'session_stop' && message.data?.output ? String(message.data.output) : null

  return (
    <div className="py-2">
      <div className="flex items-center justify-center gap-2 text-xs text-muted-foreground">
        {message.type === 'session_start' ? (
          <>
            <Play className="h-3 w-3 text-green-500" />
            <span>Executor started</span>
          </>
        ) : (
          <>
            <Square className="h-3 w-3 text-red-500" />
            <span>Executor stopped{message.stop_reason ? `: ${message.stop_reason}` : ''}</span>
            {output && (
              <button
                className="text-xs text-blue-500 hover:underline ml-1"
                onClick={() => setExpanded(!expanded)}
              >
                {expanded ? 'Hide output' : 'Show output'}
              </button>
            )}
          </>
        )}
        <span className="text-muted-foreground/50">
          {new Date(message.timestamp).toLocaleTimeString()}
        </span>
      </div>
      {expanded && output && (
        <div className="mt-2 mx-4 p-3 bg-muted rounded-md">
          <pre className="text-xs whitespace-pre-wrap font-mono overflow-auto max-h-48">
            {output}
          </pre>
        </div>
      )}
    </div>
  )
}

// Question bubble (from executor)
function QuestionBubble({ message, isPending }: { message: ChatMessage; isPending?: boolean }) {
  return (
    <div className="flex gap-2 max-w-[85%]">
      <div className="w-6 h-6 rounded-full bg-primary/10 flex items-center justify-center shrink-0">
        <Bot className="h-3 w-3 text-primary" />
      </div>
      <div className="space-y-1">
        <Card className={cn(
          'bg-muted',
          isPending && 'border-yellow-500/50'
        )}>
          <CardContent className="p-3">
            <p className="text-sm">{message.content}</p>
            {message.answer && (
              <div className="mt-2 pt-2 border-t">
                <p className="text-xs text-muted-foreground">Answer:</p>
                <p className="text-sm">{message.answer}</p>
              </div>
            )}
          </CardContent>
        </Card>
        <div className="flex items-center gap-2 text-xs text-muted-foreground px-1">
          <Clock className="h-3 w-3" />
          {new Date(message.timestamp).toLocaleTimeString()}
          {isPending && (
            <Badge variant="outline" className="text-xs h-4 px-1 text-yellow-600">
              Awaiting answer
            </Badge>
          )}
        </div>
      </div>
    </div>
  )
}

// User message bubble (to executor)
function UserMessageBubble({ message, delivered }: { message: ChatMessage; delivered?: boolean }) {
  return (
    <div className="flex gap-2 max-w-[85%] ml-auto flex-row-reverse">
      <div className="w-6 h-6 rounded-full bg-blue-500/10 flex items-center justify-center shrink-0">
        <User className="h-3 w-3 text-blue-500" />
      </div>
      <div className="space-y-1">
        <Card className={cn(
          "bg-blue-500/10 border-blue-500/20",
          !delivered && "border-dashed"
        )}>
          <CardContent className="p-3">
            <p className="text-sm">{message.content}</p>
          </CardContent>
        </Card>
        <div className="flex items-center gap-2 text-xs text-muted-foreground px-1 justify-end">
          {delivered ? (
            <Badge variant="outline" className="text-xs h-4 px-1 text-green-600">
              Delivered
            </Badge>
          ) : (
            <Badge variant="outline" className="text-xs h-4 px-1 text-amber-600">
              Queued
            </Badge>
          )}
          <Clock className="h-3 w-3" />
          {new Date(message.timestamp).toLocaleTimeString()}
        </div>
      </div>
    </div>
  )
}

// Activity message (collapsible)
function ActivityMessage({ message }: { message: ChatMessage }) {
  const [expanded, setExpanded] = useState(false)

  return (
    <div className="text-xs text-muted-foreground">
      <button
        className="flex items-center gap-1 hover:text-foreground"
        onClick={() => setExpanded(!expanded)}
      >
        {expanded ? <ChevronUp className="h-3 w-3" /> : <ChevronDown className="h-3 w-3" />}
        <span>{message.type}</span>
        <span className="text-muted-foreground/50">
          {new Date(message.timestamp).toLocaleTimeString()}
        </span>
      </button>
      {expanded && message.data && (
        <pre className="mt-1 p-2 bg-muted rounded text-xs overflow-auto max-h-32">
          {JSON.stringify(message.data, null, 2)}
        </pre>
      )}
    </div>
  )
}

// Pending question with input
function PendingQuestionInput({
  question,
  answerText,
  onAnswerChange,
  onSubmit,
}: {
  question: Question
  answerText: string
  onAnswerChange: (text: string) => void
  onSubmit: () => void
}) {
  return (
    <div className="flex gap-2 max-w-[85%]">
      <div className="w-6 h-6 rounded-full bg-yellow-500/10 flex items-center justify-center shrink-0 animate-pulse">
        <Bot className="h-3 w-3 text-yellow-500" />
      </div>
      <div className="space-y-2 flex-1">
        <Card className="border-yellow-500/50 bg-yellow-500/5">
          <CardContent className="p-3">
            <p className="text-sm font-medium mb-3">{question.question}</p>

            {/* Option buttons if available */}
            {question.options && question.options.length > 0 && (
              <div className="space-y-2 mb-3">
                {question.options.map((opt, i) => (
                  <Button
                    key={i}
                    variant={answerText === opt.label ? 'default' : 'outline'}
                    className="w-full justify-start h-auto py-2 px-3"
                    onClick={() => onAnswerChange(opt.label)}
                  >
                    <div className="text-left">
                      <span className="font-medium">{opt.label}</span>
                      {opt.description && (
                        <span className="text-muted-foreground ml-2 text-xs">
                          - {opt.description}
                        </span>
                      )}
                    </div>
                  </Button>
                ))}
              </div>
            )}

            {/* Text input */}
            <div className="flex gap-2">
              <Input
                value={answerText}
                onChange={(e) => onAnswerChange(e.target.value)}
                placeholder="Type your answer..."
                onKeyDown={(e) => {
                  if (e.key === 'Enter') onSubmit()
                }}
              />
              <Button onClick={onSubmit} disabled={!answerText.trim()}>
                Answer
              </Button>
            </div>
          </CardContent>
        </Card>
        <div className="flex items-center gap-2 text-xs text-muted-foreground px-1">
          <Clock className="h-3 w-3" />
          {new Date(question.created_at).toLocaleTimeString()}
          <Badge variant="destructive" className="text-xs h-4 px-1">
            Executor waiting
          </Badge>
        </div>
      </div>
    </div>
  )
}
