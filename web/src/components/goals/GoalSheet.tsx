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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { useMobile } from '@/hooks/useMobile'
import { EmptyState } from '@/components/shared/EmptyState'
import { Play, FileText, CheckCircle2, Circle, BookOpen, Clock, Maximize2, Minimize2, MoreVertical, Pause, Square, Trash2, AlertTriangle, GitBranch, GitCommit, ArrowUp, ArrowDown, FileWarning, GitPullRequest, RefreshCw, XCircle, Info } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { GoalDetail, GoalStatus } from '@/lib/types'
import { CompleteGoalDialog, IceGoalDialog, StopExecutorDialog, CleanupGoalDialog, ResumeGoalDialog, CreateMRDialog, RecreateWorktreeDialog } from './GoalActions'
import { ChatThread } from './ChatThread'

interface GoalSheetProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  goal: GoalDetail | null
  goalStatus: GoalStatus | null
  onRefresh: () => void
}

export function GoalSheet({ open, onOpenChange, goal, goalStatus, onRefresh }: GoalSheetProps) {
  const { isDesktop } = useMobile()
  const [spawning, setSpawning] = useState(false)
  const [showSpawnInput, setShowSpawnInput] = useState(false)
  const [spawnContext, setSpawnContext] = useState('')
  const [spawnMode, setSpawnMode] = useState('')
  const [expanded, setExpanded] = useState(false)
  const [actionMenuOpen, setActionMenuOpen] = useState(false)
  const [completeDialogOpen, setCompleteDialogOpen] = useState(false)
  const [iceDialogOpen, setIceDialogOpen] = useState(false)
  const [stopDialogOpen, setStopDialogOpen] = useState(false)
  const [cleanupDialogOpen, setCleanupDialogOpen] = useState(false)
  const [resumeDialogOpen, setResumeDialogOpen] = useState(false)
  const [createMRDialogOpen, setCreateMRDialogOpen] = useState(false)
  const [recreateWorktreeDialogOpen, setRecreateWorktreeDialogOpen] = useState(false)

  const handleAnswer = async (questionId: string, answer: string) => {
    if (!answer?.trim()) return

    try {
      const res = await fetch(`/api/answer/${questionId}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ answer }),
      })

      if (res.ok) {
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
        body: JSON.stringify({
          context: spawnContext || undefined,
          mode: spawnMode || undefined,
        }),
      })

      const data = await res.json()
      if (data.success) {
        setShowSpawnInput(false)
        setSpawnContext('')
        setSpawnMode('')
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
          <SheetHeader>
            <SheetTitle>Loading...</SheetTitle>
          </SheetHeader>
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
          'p-0 flex flex-col overflow-hidden transition-all duration-300',
          isDesktop
            ? expanded
              ? 'w-[90vw] sm:max-w-[90vw]'
              : 'w-[480px] sm:max-w-[480px]'
            : 'h-[90vh]'
        )}
      >
        {/* Sticky Header */}
        <div className="sticky top-0 z-10 bg-background border-b p-4">
          <SheetHeader className="text-left">
            <div className="flex items-center gap-2">
              {/* Expand/Collapse button - desktop only */}
              {isDesktop && (
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-8 w-8 shrink-0"
                  onClick={() => setExpanded(!expanded)}
                  title={expanded ? 'Collapse' : 'Expand'}
                >
                  {expanded ? (
                    <Minimize2 className="h-4 w-4" />
                  ) : (
                    <Maximize2 className="h-4 w-4" />
                  )}
                </Button>
              )}
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
                  <div className="flex-1 flex flex-col gap-2">
                    <div className="flex gap-2">
                      <Select value={spawnMode} onValueChange={setSpawnMode}>
                        <SelectTrigger className="w-[140px]">
                          <SelectValue placeholder="Mode" />
                        </SelectTrigger>
                        <SelectContent>
                          <SelectItem value="implement">Implement</SelectItem>
                          <SelectItem value="plan">Plan</SelectItem>
                          <SelectItem value="review">Review</SelectItem>
                          <SelectItem value="test">Test</SelectItem>
                          <SelectItem value="security">Security</SelectItem>
                          <SelectItem value="quick">Quick</SelectItem>
                        </SelectContent>
                      </Select>
                      <Input
                        placeholder="Context (optional)"
                        value={spawnContext}
                        onChange={(e) => setSpawnContext(e.target.value)}
                        className="flex-1"
                      />
                    </div>
                    <div className="flex gap-2 justify-end">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => {
                          setShowSpawnInput(false)
                          setSpawnMode('')
                          setSpawnContext('')
                        }}
                      >
                        Cancel
                      </Button>
                      <Button
                        onClick={handleSpawnExecutor}
                        disabled={spawning}
                        size="sm"
                      >
                        {spawning ? 'Starting...' : 'Start'}
                      </Button>
                    </div>
                  </div>
                ) : (
                  <Button
                    onClick={() => setShowSpawnInput(true)}
                    disabled={
                      goal.executor_status === 'running' ||
                      goal.executor_status === 'waiting' ||
                      (goal.workspace_status && goal.workspace_status !== 'ready')
                    }
                    className="gap-2"
                    title={goal.workspace_status !== 'ready' ? 'Workspace not configured' : undefined}
                  >
                    <Play className="h-4 w-4" />
                    {goal.executor_status === 'running' ? 'Running' :
                     goal.executor_status === 'waiting' ? 'Waiting' :
                     'Resume'}
                  </Button>
                )}
              </>
            )}

            {/* Action Menu */}
            <Popover open={actionMenuOpen} onOpenChange={setActionMenuOpen}>
              <PopoverTrigger asChild>
                <Button
                  variant="outline"
                  size="icon"
                  className="ml-auto"
                  disabled={goal.workspace_status && goal.workspace_status !== 'ready'}
                  title={goal.workspace_status !== 'ready' ? 'Workspace not configured' : undefined}
                >
                  <MoreVertical className="h-4 w-4" />
                </Button>
              </PopoverTrigger>
              <PopoverContent className="w-48 p-1" align="end">
                {/* Stop - only when running */}
                {(goal.executor_status === 'running' || goal.executor_status === 'waiting') &&
                  goal.active_executors &&
                  goal.active_executors.length > 0 && (
                  <Button
                    variant="ghost"
                    className="w-full justify-start gap-2 text-orange-600"
                    onClick={() => {
                      setActionMenuOpen(false)
                      setStopDialogOpen(true)
                    }}
                  >
                    <Square className="h-4 w-4" />
                    Stop Executor
                  </Button>
                )}

                {/* Complete - only for active goals */}
                {goal.status === 'active' && (
                  <Button
                    variant="ghost"
                    className="w-full justify-start gap-2 text-green-600"
                    onClick={() => {
                      setActionMenuOpen(false)
                      setCompleteDialogOpen(true)
                    }}
                  >
                    <CheckCircle2 className="h-4 w-4" />
                    Complete Goal
                  </Button>
                )}

                {/* Ice - only for active goals */}
                {goal.status === 'active' && (
                  <Button
                    variant="ghost"
                    className="w-full justify-start gap-2 text-blue-600"
                    onClick={() => {
                      setActionMenuOpen(false)
                      setIceDialogOpen(true)
                    }}
                  >
                    <Pause className="h-4 w-4" />
                    Ice Goal
                  </Button>
                )}

                {/* Cleanup - only for completed goals */}
                {goal.status === 'completed' && (
                  <Button
                    variant="ghost"
                    className="w-full justify-start gap-2 text-red-600"
                    onClick={() => {
                      setActionMenuOpen(false)
                      setCleanupDialogOpen(true)
                    }}
                  >
                    <Trash2 className="h-4 w-4" />
                    Cleanup Branch
                  </Button>
                )}

                {/* Resume - only for iced goals */}
                {goal.status === 'iced' && (
                  <Button
                    variant="ghost"
                    className="w-full justify-start gap-2 text-green-600"
                    onClick={() => {
                      setActionMenuOpen(false)
                      setResumeDialogOpen(true)
                    }}
                  >
                    <Play className="h-4 w-4" />
                    Resume Goal
                  </Button>
                )}
              </PopoverContent>
            </Popover>
          </div>

          {/* Workspace Status Warning */}
          {goal.workspace_status && goal.workspace_status !== 'ready' && (
            <div className="mt-3 p-3 bg-yellow-50 border border-yellow-200 rounded-md flex items-start gap-2">
              <AlertTriangle className="h-4 w-4 text-yellow-600 mt-0.5 shrink-0" />
              <div className="text-sm text-yellow-700">
                <strong>Workspace not configured</strong>
                <p className="mt-0.5 text-yellow-600">
                  {goal.workspace_error || 'Project workspace is not set up. Actions are unavailable.'}
                </p>
              </div>
            </div>
          )}

          {/* No Worktree Info */}
          {goal.worktree_status === 'never_created' && (
            <div className="mt-3 p-3 bg-blue-50 border-blue-200 border rounded-md flex items-start gap-2">
              <Info className="h-4 w-4 text-blue-600 mt-0.5 shrink-0" />
              <div className="flex-1">
                <div className="text-sm text-blue-700">
                  <strong>No Worktree</strong>
                  <p className="mt-0.5 text-blue-600">
                    This goal doesn't have a worktree yet. Create one to start working on a dedicated branch.
                  </p>
                </div>
                <Button
                  size="sm"
                  variant="outline"
                  className="mt-2 gap-1"
                  onClick={async () => {
                    try {
                      const res = await fetch(`/api/goals/${goal.id}/create-worktree`, { method: 'POST' })
                      if (res.ok) {
                        onRefresh?.()
                      }
                    } catch (e) {
                      console.error('Failed to create worktree:', e)
                    }
                  }}
                >
                  <GitBranch className="h-3 w-3" />
                  Create Worktree
                </Button>
              </div>
            </div>
          )}

          {/* Worktree Missing Warning */}
          {goal.worktree_status === 'missing' && (
            <div className={`mt-3 p-3 ${goal.can_recreate ? 'bg-yellow-50 border-yellow-200' : 'bg-red-50 border-red-200'} border rounded-md flex items-start gap-2`}>
              {goal.can_recreate ? (
                <AlertTriangle className="h-4 w-4 text-yellow-600 mt-0.5 shrink-0" />
              ) : (
                <XCircle className="h-4 w-4 text-red-600 mt-0.5 shrink-0" />
              )}
              <div className="flex-1">
                <div className={`text-sm ${goal.can_recreate ? 'text-yellow-700' : 'text-red-700'}`}>
                  <strong>Worktree Missing</strong>
                  <p className={`mt-0.5 ${goal.can_recreate ? 'text-yellow-600' : 'text-red-600'}`}>
                    {goal.can_recreate 
                      ? `The worktree directory was deleted, but the branch "${goal.branch_info?.branch}" still exists${goal.branch_status === 'remote_only' ? ' on remote' : ''}.`
                      : `The worktree and branch "${goal.branch_info?.branch}" no longer exist.`}
                  </p>
                </div>
                {goal.can_recreate && (
                  <Button
                    size="sm"
                    variant="outline"
                    className="mt-2 gap-1"
                    onClick={() => setRecreateWorktreeDialogOpen(true)}
                  >
                    <RefreshCw className="h-3 w-3" />
                    Recreate Worktree
                  </Button>
                )}
              </div>
            </div>
          )}
        </div>

        {/* Tabs Content */}
        <Tabs defaultValue="overview" className="flex-1 flex flex-col overflow-hidden">
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
              Chat
              {goal.pending_questions && goal.pending_questions.length > 0 && (
                <Badge variant="destructive" className="ml-1.5 h-5 px-1.5">
                  {goal.pending_questions.length}
                </Badge>
              )}
            </TabsTrigger>
            <TabsTrigger
              value="planning"
              className="rounded-none border-b-2 border-transparent data-[state=active]:border-primary px-4 py-2"
            >
              Planning
            </TabsTrigger>
            <TabsTrigger
              value="timeline"
              className="rounded-none border-b-2 border-transparent data-[state=active]:border-primary px-4 py-2"
            >
              Timeline
            </TabsTrigger>
          </TabsList>

          <ScrollArea className="flex-1 min-h-0">
            <TabsContent value="overview" className="p-4 m-0">
              {/* Branch Info */}
              {goal.branch_info && (
                <Card className="mb-4">
                  <CardHeader className="p-3 pb-2">
                    <CardTitle className="text-sm flex items-center gap-2">
                      <GitBranch className="h-4 w-4" />
                      Branch Info
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="p-3 pt-0">
                    <div className="grid grid-cols-2 gap-3 text-sm">
                      <div>
                        <span className="text-muted-foreground">Branch:</span>
                        <code className="ml-2 text-xs bg-muted px-1.5 py-0.5 rounded">
                          {goal.branch_info.branch}
                        </code>
                      </div>
                      <div>
                        <span className="text-muted-foreground">Base:</span>
                        <code className="ml-2 text-xs bg-muted px-1.5 py-0.5 rounded">
                          {goal.branch_info.base_branch}
                        </code>
                      </div>
                      <div className="flex items-center gap-3">
                        {goal.branch_info.ahead > 0 && (
                          <span className="flex items-center gap-1 text-green-600">
                            <ArrowUp className="h-3 w-3" />
                            {goal.branch_info.ahead} ahead
                          </span>
                        )}
                        {goal.branch_info.behind > 0 && (
                          <span className="flex items-center gap-1 text-orange-600">
                            <ArrowDown className="h-3 w-3" />
                            {goal.branch_info.behind} behind
                          </span>
                        )}
                        {goal.branch_info.ahead === 0 && goal.branch_info.behind === 0 && (
                          <span className="text-muted-foreground">Up to date</span>
                        )}
                      </div>
                      <div>
                        {goal.branch_info.uncommitted_files > 0 ? (
                          <span className="flex items-center gap-1 text-yellow-600">
                            <FileWarning className="h-3 w-3" />
                            {goal.branch_info.uncommitted_files} uncommitted
                          </span>
                        ) : (
                          <span className="text-muted-foreground">Clean</span>
                        )}
                      </div>
                    </div>
                    {goal.branch_info.last_commit && (
                      <div className="mt-3 pt-3 border-t text-xs text-muted-foreground">
                        <div className="flex items-center gap-2">
                          <GitCommit className="h-3 w-3" />
                          <code>{goal.branch_info.last_commit.slice(0, 7)}</code>
                          <span className="truncate">{goal.branch_info.last_commit_message}</span>
                        </div>
                      </div>
                    )}
                    {/* Create MR button - only show when there are commits ahead */}
                    {goal.branch_info.ahead > 0 && (
                      <div className="mt-3 pt-3 border-t">
                        <Button
                          size="sm"
                          variant="outline"
                          className="w-full gap-2"
                          onClick={() => setCreateMRDialogOpen(true)}
                          disabled={goal.branch_info.uncommitted_files > 0}
                          title={goal.branch_info.uncommitted_files > 0 
                            ? 'Commit or stash your changes before creating an MR' 
                            : 'Create a merge/pull request'}
                        >
                          <GitPullRequest className="h-4 w-4" />
                          {goal.branch_info.uncommitted_files > 0 
                            ? 'Commit changes first' 
                            : 'Create MR/PR'}
                        </Button>
                        {goal.branch_info.uncommitted_files > 0 && (
                          <p className="text-xs text-yellow-600 mt-1.5 text-center">
                            You have uncommitted changes
                          </p>
                        )}
                      </div>
                    )}
                  </CardContent>
                </Card>
              )}

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

            <TabsContent value="qa" className="p-4 m-0 h-[calc(100vh-300px)]">
              <ChatThread
                goalId={goal.id}
                pendingQuestions={goal.pending_questions || []}
                onAnswerSubmit={handleAnswer}
              />
            </TabsContent>

            <TabsContent value="planning" className="p-4 m-0">
              {goalStatus && goalStatus.has_worktree ? (
                <div className="space-y-4">
                  {/* Current Phase */}
                  {goalStatus.current_phase && (
                    <div>
                      <h4 className="font-medium mb-2">Current Phase</h4>
                      <p className="text-sm text-muted-foreground">{goalStatus.current_phase}</p>
                    </div>
                  )}

                  {/* Phase Progress */}
                  {goalStatus.phase_progress && goalStatus.phase_progress.length > 0 && (
                    <div>
                      <h4 className="font-medium mb-2">Phase Progress</h4>
                      <div className="space-y-2">
                        {goalStatus.phase_progress.map((phase) => (
                          <div key={phase.number} className="flex items-center gap-3">
                            <Badge variant={
                              phase.status === 'complete' ? 'success' :
                              phase.status === 'in_progress' ? 'default' :
                              'secondary'
                            } className="w-20 justify-center">
                              {phase.status}
                            </Badge>
                            <span className="text-sm flex-1">
                              Phase {phase.number}: {phase.title}
                            </span>
                            <span className="text-xs text-muted-foreground">
                              {phase.tasks_done}/{phase.tasks_total}
                            </span>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                  {/* Task Plan Preview */}
                  {goalStatus.task_plan && (
                    <div>
                      <h4 className="font-medium mb-2">Task Plan</h4>
                      <Card>
                        <CardContent className="p-3">
                          <pre className="text-xs text-muted-foreground whitespace-pre-wrap font-mono max-h-48 overflow-auto">
                            {goalStatus.task_plan.slice(0, 1000)}
                            {goalStatus.task_plan.length > 1000 && '...'}
                          </pre>
                        </CardContent>
                      </Card>
                    </div>
                  )}

                  {/* Findings Preview */}
                  {goalStatus.findings && (
                    <div>
                      <h4 className="font-medium mb-2">Findings</h4>
                      <Card>
                        <CardContent className="p-3">
                          <pre className="text-xs text-muted-foreground whitespace-pre-wrap font-mono max-h-48 overflow-auto">
                            {goalStatus.findings.slice(0, 1000)}
                            {goalStatus.findings.length > 1000 && '...'}
                          </pre>
                        </CardContent>
                      </Card>
                    </div>
                  )}

                  {/* Worktree Path */}
                  {goalStatus.worktree_path && (
                    <div>
                      <h4 className="font-medium mb-2">Worktree</h4>
                      <code className="text-xs text-muted-foreground bg-muted px-2 py-1 rounded">
                        {goalStatus.worktree_path}
                      </code>
                    </div>
                  )}
                </div>
              ) : (
                <EmptyState
                  icon={BookOpen}
                  title="No planning files"
                  description="Planning files will appear when an executor is working on this goal"
                />
              )}
            </TabsContent>

            <TabsContent value="timeline" className="p-4 m-0">
              {goalStatus && goalStatus.recent_actions && goalStatus.recent_actions.length > 0 ? (
                <div className="space-y-4">
                  <h4 className="font-medium">Recent Actions</h4>
                  <div className="space-y-2">
                    {goalStatus.recent_actions.map((action, i) => (
                      <div key={i} className="flex items-start gap-3 py-2 border-b border-border last:border-0">
                        <Clock className="h-4 w-4 text-muted-foreground mt-0.5 shrink-0" />
                        <span className="text-sm">{action}</span>
                      </div>
                    ))}
                  </div>

                  {/* Progress Log Preview */}
                  {goalStatus.progress_log && (
                    <div className="mt-4">
                      <h4 className="font-medium mb-2">Progress Log</h4>
                      <Card>
                        <CardContent className="p-3">
                          <pre className="text-xs text-muted-foreground whitespace-pre-wrap font-mono max-h-48 overflow-auto">
                            {goalStatus.progress_log.slice(0, 1500)}
                            {goalStatus.progress_log.length > 1500 && '...'}
                          </pre>
                        </CardContent>
                      </Card>
                    </div>
                  )}
                </div>
              ) : (
                <EmptyState
                  icon={Clock}
                  title="No timeline data"
                  description="Activity timeline will appear when an executor starts working"
                />
              )}
            </TabsContent>
          </ScrollArea>
        </Tabs>
      </SheetContent>

      {/* Goal Action Dialogs */}
      <CompleteGoalDialog
        open={completeDialogOpen}
        onOpenChange={setCompleteDialogOpen}
        goal={goal}
        onSuccess={onRefresh}
      />

      <IceGoalDialog
        open={iceDialogOpen}
        onOpenChange={setIceDialogOpen}
        goal={goal}
        onSuccess={onRefresh}
      />

      {goal.active_executors && goal.active_executors.length > 0 && (
        <StopExecutorDialog
          open={stopDialogOpen}
          onOpenChange={setStopDialogOpen}
          goalId={goal.id}
          sessionId={goal.active_executors[0].session_id}
          onSuccess={onRefresh}
        />
      )}

      <CleanupGoalDialog
        open={cleanupDialogOpen}
        onOpenChange={setCleanupDialogOpen}
        goalId={goal.id}
        project={goal.projects[0] || ''}
        onSuccess={onRefresh}
      />

      <ResumeGoalDialog
        open={resumeDialogOpen}
        onOpenChange={setResumeDialogOpen}
        goal={goal}
        onSuccess={onRefresh}
      />

      {goal.branch_info && (
        <CreateMRDialog
          open={createMRDialogOpen}
          onOpenChange={setCreateMRDialogOpen}
          goalId={goal.id}
          goalTitle={goal.title}
          baseBranch={goal.branch_info.base_branch}
          lastCommitMessage={goal.branch_info.last_commit_message}
          onSuccess={onRefresh}
        />
      )}

      {goal.worktree_status === 'missing' && (
        <RecreateWorktreeDialog
          open={recreateWorktreeDialogOpen}
          onOpenChange={setRecreateWorktreeDialogOpen}
          goal={goal}
          onSuccess={onRefresh}
        />
      )}
    </Sheet>
  )
}
