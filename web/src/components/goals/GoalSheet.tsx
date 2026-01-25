import { useState } from 'react'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from '@/components/ui/sheet'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { useMobile } from '@/hooks/useMobile'
import { Play, FileText, AlertCircle, CheckCircle2, Circle, MessageSquare } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { GoalDetail, GoalStatus, Question } from '@/lib/types'

interface GoalSheetProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  goal: GoalDetail | null
  goalStatus: GoalStatus | null
  onRefresh: () => void
}

export function GoalSheet({ open, onOpenChange, goal, onRefresh }: GoalSheetProps) {
  const { isDesktop } = useMobile()
  const [answerText, setAnswerText] = useState<Record<string, string>>({})
  const [spawning, setSpawning] = useState(false)
  const [showSpawnInput, setShowSpawnInput] = useState(false)
  const [spawnContext, setSpawnContext] = useState('')

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
        onRefresh()
      }
    } catch (err) {
      console.error('Failed to submit answer:', err)
    }
  }

  const handleSpawnExecutor = async () => {
    if (!goal) return
    setSpawning(true)
    try {
      const res = await fetch(`/api/goals/${goal.id}/spawn`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ context: spawnContext || undefined }),
      })

      const data = await res.json()
      if (data.success) {
        setShowSpawnInput(false)
        setSpawnContext('')
        onRefresh()
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

  if (!goal) {
    return (
      <Sheet open={open} onOpenChange={onOpenChange}>
        <SheetContent side={isDesktop ? 'right' : 'bottom'} className={cn(
          isDesktop ? 'w-[480px] sm:max-w-[480px]' : 'h-[90vh]'
        )}>
          <div className="flex items-center justify-center h-full">
            <Skeleton className="h-48 w-full" />
          </div>
        </SheetContent>
      </Sheet>
    )
  }

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side={isDesktop ? 'right' : 'bottom'}
        className={cn(
          'p-0 flex flex-col',
          isDesktop ? 'w-[480px] sm:max-w-[480px]' : 'h-[90vh]'
        )}
      >
        {/* Sticky Header */}
        <div className="sticky top-0 z-10 bg-background border-b p-4">
          <SheetHeader className="text-left">
            <div className="flex items-center gap-2">
              <div className={cn(
                'h-2 w-2 rounded-full',
                goal.executor_status === 'running' ? 'bg-green-500 animate-pulse' :
                goal.executor_status === 'waiting' ? 'bg-red-500' :
                'bg-muted-foreground'
              )} />
              <SheetTitle className="text-lg">#{goal.id}</SheetTitle>
              <Badge variant={
                goal.status === 'completed' ? 'success' :
                goal.status === 'iced' ? 'secondary' :
                goal.executor_status === 'running' ? 'success' :
                goal.executor_status === 'waiting' ? 'destructive' :
                'secondary'
              }>
                {goal.status === 'completed' ? 'COMPLETE' :
                 goal.status === 'iced' ? 'ICED' :
                 goal.executor_status.toUpperCase()}
              </Badge>
            </div>
            <SheetDescription className="text-foreground font-medium">
              {goal.title}
            </SheetDescription>
            <div className="flex items-center gap-3 text-xs text-muted-foreground mt-1">
              <span>Phase: {goal.phase}</span>
              {goal.projects.length > 0 && (
                <span>{goal.projects.join(', ')}</span>
              )}
            </div>
          </SheetHeader>

          {/* Action Bar */}
          <div className="flex items-center gap-2 mt-4">
            {goal.status === 'active' && (
              <>
                {showSpawnInput ? (
                  <div className="flex-1 flex gap-2">
                    <Input
                      placeholder="Context (optional)"
                      value={spawnContext}
                      onChange={(e) => setSpawnContext(e.target.value)}
                      className="flex-1"
                    />
                    <Button
                      onClick={handleSpawnExecutor}
                      disabled={spawning}
                      size="sm"
                    >
                      {spawning ? 'Starting...' : 'Start'}
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setShowSpawnInput(false)}
                    >
                      Cancel
                    </Button>
                  </div>
                ) : (
                  <Button
                    onClick={() => setShowSpawnInput(true)}
                    disabled={goal.executor_status === 'running' || goal.executor_status === 'waiting'}
                    className="gap-2"
                  >
                    <Play className="h-4 w-4" />
                    {goal.executor_status === 'running' ? 'Running' :
                     goal.executor_status === 'waiting' ? 'Waiting' :
                     'Resume'}
                  </Button>
                )}
              </>
            )}
          </div>
        </div>

        {/* Tabs Content */}
        <Tabs defaultValue="overview" className="flex-1 flex flex-col">
          <TabsList className="w-full justify-start rounded-none border-b bg-transparent p-0 h-auto">
            <TabsTrigger
              value="overview"
              className="rounded-none border-b-2 border-transparent data-[state=active]:border-primary px-4 py-2"
            >
              Overview
            </TabsTrigger>
            <TabsTrigger
              value="phases"
              className="rounded-none border-b-2 border-transparent data-[state=active]:border-primary px-4 py-2"
            >
              Phases
            </TabsTrigger>
            <TabsTrigger
              value="qa"
              className="rounded-none border-b-2 border-transparent data-[state=active]:border-primary px-4 py-2 relative"
            >
              Q&A
              {goal.pending_questions.length > 0 && (
                <Badge variant="destructive" className="ml-1.5 h-5 px-1.5">
                  {goal.pending_questions.length}
                </Badge>
              )}
            </TabsTrigger>
          </TabsList>

          <ScrollArea className="flex-1">
            <TabsContent value="overview" className="p-4 m-0">
              {goal.overview && (
                <div className="mb-4">
                  <h4 className="font-medium mb-2">Description</h4>
                  <p className="text-sm text-muted-foreground whitespace-pre-wrap">
                    {goal.overview}
                  </p>
                </div>
              )}

              {goal.acceptance && goal.acceptance.length > 0 && (
                <div className="mb-4">
                  <h4 className="font-medium mb-2">Acceptance Criteria</h4>
                  <ul className="space-y-1 text-sm text-muted-foreground">
                    {goal.acceptance.map((item, i) => (
                      <li key={i} className="flex items-start gap-2">
                        <Circle className="h-3 w-3 mt-1.5 text-muted-foreground/50" />
                        <span>{item}</span>
                      </li>
                    ))}
                  </ul>
                </div>
              )}

              {goal.notes && goal.notes.length > 0 && (
                <div>
                  <h4 className="font-medium mb-2">Notes</h4>
                  <ul className="space-y-1 text-sm text-muted-foreground">
                    {goal.notes.map((note, i) => (
                      <li key={i}>• {note}</li>
                    ))}
                  </ul>
                </div>
              )}
            </TabsContent>

            <TabsContent value="phases" className="p-4 m-0">
              {goal.phases && goal.phases.length > 0 ? (
                <div className="space-y-3">
                  {goal.phases.map((phase) => (
                    <Card key={phase.number}>
                      <CardHeader className="p-4 pb-2">
                        <div className="flex items-center justify-between">
                          <CardTitle className="text-sm">
                            Phase {phase.number}: {phase.title}
                          </CardTitle>
                          <Badge variant={
                            phase.status === 'complete' ? 'success' :
                            phase.status === 'in_progress' ? 'default' :
                            'secondary'
                          }>
                            {phase.status}
                          </Badge>
                        </div>
                      </CardHeader>
                      <CardContent className="p-4 pt-0">
                        {phase.tasks && phase.tasks.length > 0 && (
                          <ul className="space-y-1 text-sm">
                            {phase.tasks.map((task, i) => (
                              <li key={i} className="flex items-start gap-2">
                                {task.completed ? (
                                  <CheckCircle2 className="h-4 w-4 mt-0.5 text-green-500" />
                                ) : (
                                  <Circle className="h-4 w-4 mt-0.5 text-muted-foreground" />
                                )}
                                <span className={cn(
                                  task.completed && 'line-through text-muted-foreground'
                                )}>
                                  {task.description}
                                </span>
                              </li>
                            ))}
                          </ul>
                        )}
                      </CardContent>
                    </Card>
                  ))}
                </div>
              ) : (
                <div className="flex flex-col items-center justify-center py-8 text-center text-muted-foreground">
                  <FileText className="h-12 w-12 mb-4 opacity-50" />
                  <p>No phases defined</p>
                </div>
              )}
            </TabsContent>

            <TabsContent value="qa" className="p-4 m-0">
              {goal.pending_questions && goal.pending_questions.length > 0 ? (
                <div className="space-y-4">
                  <h4 className="font-medium flex items-center gap-2 text-destructive">
                    <AlertCircle className="h-4 w-4" />
                    Pending Questions
                  </h4>
                  {goal.pending_questions.map((q) => (
                    <QuestionCard
                      key={q.id}
                      question={q}
                      answerText={answerText[q.id] || ''}
                      onAnswerChange={(text) => setAnswerText((prev) => ({ ...prev, [q.id]: text }))}
                      onSubmit={() => handleAnswer(q.id)}
                    />
                  ))}
                </div>
              ) : (
                <div className="flex flex-col items-center justify-center py-8 text-center text-muted-foreground">
                  <MessageSquare className="h-12 w-12 mb-4 opacity-50" />
                  <p>No pending questions</p>
                  <p className="text-sm">Questions from executors will appear here</p>
                </div>
              )}
            </TabsContent>
          </ScrollArea>
        </Tabs>
      </SheetContent>
    </Sheet>
  )
}

function QuestionCard({
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
    <Card className="border-destructive/50 bg-destructive/5">
      <CardContent className="p-4">
        <div className="flex items-center gap-2 text-xs text-muted-foreground mb-2">
          <span>Session: {question.session_id.slice(0, 8)}...</span>
          <span>{new Date(question.created_at).toLocaleTimeString()}</span>
        </div>
        <p className="mb-4">{question.question}</p>

        {question.options && question.options.length > 0 && (
          <div className="space-y-2 mb-4">
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
  )
}
