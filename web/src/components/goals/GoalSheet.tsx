import { useState, useEffect, useCallback } from 'react'
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
import { Play, FileText, CheckCircle2, Circle, BookOpen, Clock, Maximize2, Minimize2, MoreVertical, Pause, Square, Trash2, AlertTriangle, GitBranch, GitCommit, ArrowUp, ArrowDown, FileWarning, GitPullRequest, RefreshCw, XCircle, Info, Activity, Sparkles, ListTodo, Ban, Link2, GitFork, ChevronRight, FolderOpen, FileCode, Zap, Bell } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { GoalDetail, GoalStatus, GoalState, PlanningFile } from '@/lib/types'

// Helper to get badge variant, label, and description for state
function getStateInfo(state: GoalState): { variant: 'default' | 'secondary' | 'destructive' | 'success' | 'outline'; label: string; description: string } {
  switch (state) {
    case 'pending':
      return { variant: 'secondary', label: 'Pending', description: 'Goal created, waiting to start' }
    case 'branching':
      return { variant: 'default', label: 'Branching', description: 'Creating branch and worktree' }
    case 'working':
      return { variant: 'default', label: 'Working', description: 'Active development in progress' }
    case 'pushing':
      return { variant: 'default', label: 'Pushing', description: 'Committing and pushing changes' }
    case 'merging':
      return { variant: 'default', label: 'Merging', description: 'Merging branch to base' }
    case 'done':
      return { variant: 'success', label: 'Done', description: 'Goal completed successfully' }
    case 'iced':
      return { variant: 'secondary', label: 'Iced', description: 'Goal paused for later' }
    case 'failed':
      return { variant: 'destructive', label: 'Failed', description: 'Something went wrong (recoverable)' }
    case 'conflict':
      return { variant: 'destructive', label: 'Conflict', description: 'Merge conflict needs resolution' }
    default:
      return { variant: 'outline', label: state, description: 'Unknown state' }
  }
}

function StateBadge({ state }: { state: GoalState }) {
  const { variant, label, description } = getStateInfo(state)
  return (
    <span title={description}>
      <Badge variant={variant} className="gap-1 cursor-help">
        <Activity className="h-3 w-3" />
        {label}
      </Badge>
    </span>
  )
}
import { CompleteGoalDialog, IceGoalDialog, StopExecutorDialog, CleanupGoalDialog, ResumeGoalDialog, CreateMRDialog, RecreateWorktreeDialog, DeleteGoalDialog } from './GoalActions'
import { ChatThread } from './ChatThread'
import type { CompletionStatus } from '@/lib/types'

// CompletionStatusBadge shows completion detection status in the goal header
function CompletionStatusBadge({ completionStatus }: { completionStatus: CompletionStatus }) {
  const { complete, confidence, signals, missing_tasks, completed_phases, total_phases } = completionStatus
  const confidencePct = Math.round(confidence * 100)

  if (complete) {
    return (
      <div className="mt-3 p-3 bg-green-50 border border-green-200 rounded-md">
        <div className="flex items-center gap-2">
          <Sparkles className="h-4 w-4 text-green-600" />
          <span className="text-sm font-medium text-green-700">Ready to Complete</span>
          <Badge variant="success" className="ml-auto">
            {confidencePct}% confidence
          </Badge>
        </div>
        {signals.length > 0 && (
          <div className="mt-2 text-xs text-green-600 space-y-1">
            {signals.map((signal, i) => (
              <div key={i} className="flex items-start gap-1.5">
                <CheckCircle2 className="h-3 w-3 mt-0.5 shrink-0" />
                <span>{signal.message}</span>
              </div>
            ))}
          </div>
        )}
      </div>
    )
  }

  // Not complete - show what's missing
  if (missing_tasks.length === 0 && confidence < 0.3) {
    return null // Not enough data to show anything meaningful
  }

  return (
    <div className="mt-3 p-3 bg-muted/50 border rounded-md">
      <div className="flex items-center gap-2">
        <ListTodo className="h-4 w-4 text-muted-foreground" />
        <span className="text-sm font-medium">Completion Status</span>
        {total_phases > 0 && (
          <Badge variant="outline" className="ml-auto">
            {completed_phases}/{total_phases} phases
          </Badge>
        )}
      </div>

      {/* Progress bar */}
      {confidence > 0 && (
        <div className="mt-2">
          <div className="flex items-center justify-between text-xs text-muted-foreground mb-1">
            <span>Progress</span>
            <span>{confidencePct}%</span>
          </div>
          <div className="h-1.5 bg-muted rounded-full overflow-hidden">
            <div 
              className={cn(
                "h-full rounded-full transition-all",
                confidence >= 0.7 ? "bg-green-500" : 
                confidence >= 0.4 ? "bg-yellow-500" : "bg-muted-foreground"
              )} 
              style={{ width: `${confidencePct}%` }}
            />
          </div>
        </div>
      )}

      {/* Signals detected */}
      {signals.length > 0 && (
        <div className="mt-2 text-xs text-muted-foreground space-y-1">
          {signals.map((signal, i) => (
            <div key={i} className="flex items-start gap-1.5">
              <CheckCircle2 className="h-3 w-3 mt-0.5 shrink-0 text-green-500" />
              <span>{signal.message}</span>
            </div>
          ))}
        </div>
      )}

      {/* Missing items */}
      {missing_tasks.length > 0 && (
        <div className="mt-2">
          <div className="text-xs font-medium text-muted-foreground mb-1">
            Missing ({missing_tasks.length}):
          </div>
          <div className="text-xs text-muted-foreground space-y-0.5 max-h-24 overflow-y-auto">
            {missing_tasks.slice(0, 5).map((task, i) => (
              <div key={i} className="flex items-start gap-1.5">
                <Circle className="h-3 w-3 mt-0.5 shrink-0" />
                <span className="truncate">{task}</span>
              </div>
            ))}
            {missing_tasks.length > 5 && (
              <div className="text-muted-foreground/60 pl-4">
                +{missing_tasks.length - 5} more...
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

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
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [planningFiles, setPlanningFiles] = useState<PlanningFile[]>([])
  const [loadingPlanningFiles, setLoadingPlanningFiles] = useState(false)
  const [expandedFiles, setExpandedFiles] = useState<Set<string>>(new Set())
  const [isMetaExecutor, setIsMetaExecutor] = useState(false)
  const [hasNewPlanningFiles, setHasNewPlanningFiles] = useState(false)
  const [spawningMeta, setSpawningMeta] = useState(false)

  // Listen for SSE planning_file_received events to show badge
  useEffect(() => {
    if (!goal) return
    
    const handleSSE = (event: Event) => {
      const customEvent = event as CustomEvent<{ goal_id: string; project: string; filename: string }>
      if (customEvent.detail?.goal_id === goal.id) {
        setHasNewPlanningFiles(true)
      }
    }
    
    window.addEventListener('planning_file_received', handleSSE)
    return () => window.removeEventListener('planning_file_received', handleSSE)
  }, [goal?.id])

  // Clear new files indicator when files are loaded
  const fetchPlanningFiles = useCallback(async () => {
    if (!goal) return
    setLoadingPlanningFiles(true)
    try {
      const res = await fetch(`/api/goals/${goal.id}/planning-files?full=true`)
      if (res.ok) {
        const data = await res.json()
        // Convert { files: { project: { filename: content } } } to PlanningFile[]
        const files: PlanningFile[] = []
        if (data.files) {
          for (const [project, fileMap] of Object.entries(data.files)) {
            for (const [filename, content] of Object.entries(fileMap as Record<string, string>)) {
              files.push({ project, filename, content })
            }
          }
        }
        setPlanningFiles(files)
        setHasNewPlanningFiles(false)
      }
    } catch (err) {
      console.error('Failed to fetch planning files:', err)
    } finally {
      setLoadingPlanningFiles(false)
    }
  }, [goal?.id])

  // Handle spawning meta-executor
  const handleSpawnMetaExecutor = async () => {
    if (!goal) return
    setSpawningMeta(true)
    try {
      console.log('[META-EXECUTOR] Intent to spawn meta-executor for goal:', goal.id)
      console.log('[META-EXECUTOR] Planning files available:', planningFiles.length)
      console.log('[META-EXECUTOR] Projects:', goal.projects)
      
      // For now, just log the intent. Full implementation will come in Phase 7.
      const res = await fetch(`/api/goals/${goal.id}/spawn`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          context: `Aggregate findings from ${planningFiles.length} planning files across projects: ${goal.projects.join(', ')}`,
          mode: 'implement',
          meta: true,
        }),
      })

      const data = await res.json()
      if (data.success) {
        onRefresh()
      } else {
        alert('Failed to spawn meta-executor: ' + data.message)
      }
    } catch (err) {
      console.error('Failed to spawn meta-executor:', err)
      alert('Failed to spawn meta-executor')
    } finally {
      setSpawningMeta(false)
    }
  }

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
          meta: isMetaExecutor || undefined,
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
            {/* State Machine Badge */}
            {goal.state && (
              <div className="flex items-center gap-2 mt-2">
                <StateBadge state={goal.state} />
                {goal.state_since && (
                  <span className="text-xs text-muted-foreground">
                    since {new Date(goal.state_since).toLocaleString()}
                  </span>
                )}
              </div>
            )}

            {/* Completion Status Indicator */}
            {goal.status === 'active' && goal.completion_status && (
              <CompletionStatusBadge completionStatus={goal.completion_status} />
            )}
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
                    <div className="flex items-center justify-between">
                      <label className="flex items-center gap-2 text-sm text-muted-foreground cursor-pointer">
                        <input
                          type="checkbox"
                          checked={isMetaExecutor}
                          onChange={(e) => setIsMetaExecutor(e.target.checked)}
                          className="rounded border-gray-300"
                        />
                        Meta-executor (spawns sub-executors)
                      </label>
                      <div className="flex gap-2">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => {
                            setShowSpawnInput(false)
                            setSpawnMode('')
                            setSpawnContext('')
                            setIsMetaExecutor(false)
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

                {/* Delete - available for all goals */}
                <Button
                  variant="ghost"
                  className="w-full justify-start gap-2 text-red-600"
                  onClick={() => {
                    setActionMenuOpen(false)
                    setDeleteDialogOpen(true)
                  }}
                >
                  <Trash2 className="h-4 w-4" />
                  Delete Goal
                </Button>
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
              value="dependencies"
              className="rounded-none border-b-2 border-transparent data-[state=active]:border-primary px-4 py-2"
            >
              Deps
              {goal.dependencies?.is_blocked && (
                <Ban className="ml-1 h-3 w-3 text-yellow-500" />
              )}
            </TabsTrigger>
            <TabsTrigger
              value="planning"
              className="rounded-none border-b-2 border-transparent data-[state=active]:border-primary px-4 py-2"
            >
              Planning
            </TabsTrigger>
            <TabsTrigger
              value="files"
              className="rounded-none border-b-2 border-transparent data-[state=active]:border-primary px-4 py-2 relative"
            >
              Files
              {hasNewPlanningFiles && (
                <span className="absolute -top-1 -right-1 flex h-3 w-3">
                  <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-blue-400 opacity-75"></span>
                  <span className="relative inline-flex rounded-full h-3 w-3 bg-blue-500"></span>
                </span>
              )}
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

              {/* Hierarchy Section */}
              {goal.hierarchy && (goal.hierarchy.parent_id || goal.hierarchy.children.length > 0) && (
                <Card className="mb-4">
                  <CardHeader className="p-3 pb-2">
                    <CardTitle className="text-sm flex items-center gap-2">
                      <GitFork className="h-4 w-4" />
                      Goal Hierarchy
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="p-3 pt-0 space-y-2">
                    {goal.hierarchy.parent_id && (
                      <div className="text-sm">
                        <span className="text-muted-foreground">Parent:</span>
                        <Button
                          variant="link"
                          size="sm"
                          className="h-auto p-0 ml-2 text-primary"
                          onClick={() => {
                            // Navigate to parent goal
                            onOpenChange(false)
                            // The parent component should handle this
                            window.location.hash = `#goal-${goal.hierarchy!.parent_id}`
                          }}
                        >
                          #{goal.hierarchy.parent_id}
                        </Button>
                      </div>
                    )}
                    {goal.hierarchy.children.length > 0 && (
                      <div className="text-sm">
                        <span className="text-muted-foreground">Children ({goal.hierarchy.children.length}):</span>
                        <div className="flex flex-wrap gap-1 mt-1">
                          {goal.hierarchy.children.map((childId) => (
                            <Badge key={childId} variant="outline" className="cursor-pointer hover:bg-accent">
                              #{childId}
                            </Badge>
                          ))}
                        </div>
                      </div>
                    )}
                    {goal.hierarchy.depth > 0 && (
                      <div className="text-xs text-muted-foreground">
                        Depth: {goal.hierarchy.depth}
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

            <TabsContent value="dependencies" className="p-4 m-0">
              {goal.dependencies ? (
                <div className="space-y-4">
                  {/* Blocked By Section */}
                  {goal.dependencies.blockers.length > 0 && (
                    <Card className="border-yellow-500/50">
                      <CardHeader className="p-3 pb-2">
                        <CardTitle className="text-sm flex items-center gap-2 text-yellow-600">
                          <Ban className="h-4 w-4" />
                          Blocked By ({goal.dependencies.blockers.length})
                        </CardTitle>
                      </CardHeader>
                      <CardContent className="p-3 pt-0">
                        <div className="flex flex-wrap gap-2">
                          {goal.dependencies.blockers.map((blockerId) => (
                            <Badge key={blockerId} variant="warning" className="cursor-pointer hover:opacity-80">
                              #{blockerId}
                            </Badge>
                          ))}
                        </div>
                      </CardContent>
                    </Card>
                  )}

                  {/* Dependencies Section - Goals this one depends on */}
                  {goal.dependencies.dependencies.length > 0 && (
                    <Card>
                      <CardHeader className="p-3 pb-2">
                        <CardTitle className="text-sm flex items-center gap-2">
                          <ChevronRight className="h-4 w-4" />
                          Depends On ({goal.dependencies.dependencies.length})
                        </CardTitle>
                      </CardHeader>
                      <CardContent className="p-3 pt-0">
                        <div className="space-y-2">
                          {goal.dependencies.dependencies.map((dep) => (
                            <div key={dep.goal_id} className="flex items-center gap-2">
                              <Badge variant={dep.type === 'blocks' ? 'destructive' : 'outline'} className="cursor-pointer">
                                #{dep.goal_id}
                              </Badge>
                              <span className="text-xs text-muted-foreground">
                                {dep.type === 'blocks' ? 'blocking' : 'related'}
                              </span>
                            </div>
                          ))}
                        </div>
                      </CardContent>
                    </Card>
                  )}

                  {/* Dependents Section - Goals that depend on this one */}
                  {goal.dependencies.dependents.length > 0 && (
                    <Card>
                      <CardHeader className="p-3 pb-2">
                        <CardTitle className="text-sm flex items-center gap-2">
                          <Link2 className="h-4 w-4" />
                          Blocking Others ({goal.dependencies.dependents.length})
                        </CardTitle>
                      </CardHeader>
                      <CardContent className="p-3 pt-0">
                        <div className="space-y-2">
                          {goal.dependencies.dependents.map((dep) => (
                            <div key={dep.goal_id} className="flex items-center gap-2">
                              <Badge variant={dep.type === 'blocks' ? 'warning' : 'outline'} className="cursor-pointer">
                                #{dep.goal_id}
                              </Badge>
                              <span className="text-xs text-muted-foreground">
                                {dep.type === 'blocks' ? 'blocked by this' : 'related'}
                              </span>
                            </div>
                          ))}
                        </div>
                      </CardContent>
                    </Card>
                  )}

                  {/* Empty State */}
                  {goal.dependencies.dependencies.length === 0 && 
                   goal.dependencies.dependents.length === 0 && 
                   goal.dependencies.blockers.length === 0 && (
                    <EmptyState
                      icon={Link2}
                      title="No dependencies"
                      description="This goal has no dependencies or dependents"
                    />
                  )}
                </div>
              ) : (
                <EmptyState
                  icon={Link2}
                  title="No dependencies"
                  description="Dependency tracking not available for this goal"
                />
              )}
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

            <TabsContent value="files" className="p-4 m-0">
              {goal.projects.length > 0 ? (
                <div className="space-y-4">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <h4 className="font-medium">Planning Files</h4>
                      {hasNewPlanningFiles && (
                        <Badge variant="secondary" className="gap-1">
                          <Bell className="h-3 w-3" />
                          New
                        </Badge>
                      )}
                    </div>
                    <Button
                      variant="outline"
                      size="sm"
                      disabled={loadingPlanningFiles}
                      onClick={fetchPlanningFiles}
                    >
                      <RefreshCw className={cn("h-4 w-4 mr-1", loadingPlanningFiles && "animate-spin")} />
                      Refresh
                    </Button>
                  </div>

                  {/* Meta-Executor Trigger - only show when there are planning files */}
                  {planningFiles.length > 0 && (
                    <Card className="border-blue-500/30 bg-blue-50/50">
                      <CardContent className="p-3">
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-2">
                            <Zap className="h-4 w-4 text-blue-600" />
                            <div>
                              <p className="text-sm font-medium text-blue-900">
                                {planningFiles.length} planning file{planningFiles.length !== 1 ? 's' : ''} collected
                              </p>
                              <p className="text-xs text-blue-600">
                                From {new Set(planningFiles.map(f => f.project)).size} project{new Set(planningFiles.map(f => f.project)).size !== 1 ? 's' : ''}
                              </p>
                            </div>
                          </div>
                          <Button
                            size="sm"
                            variant="default"
                            className="gap-1 bg-blue-600 hover:bg-blue-700"
                            disabled={spawningMeta || goal.executor_status === 'running'}
                            onClick={handleSpawnMetaExecutor}
                          >
                            <Zap className="h-3 w-3" />
                            {spawningMeta ? 'Spawning...' : 'Spawn Meta-Executor'}
                          </Button>
                        </div>
                      </CardContent>
                    </Card>
                  )}

                  {planningFiles.length > 0 ? (
                    <div className="space-y-3">
                      {/* Group by project */}
                      {goal.projects.map((project) => {
                        const projectFiles = planningFiles.filter(f => f.project === project)
                        if (projectFiles.length === 0) return null
                        return (
                          <Card key={project}>
                            <CardHeader className="p-3 pb-2">
                              <CardTitle className="text-sm flex items-center gap-2">
                                <FolderOpen className="h-4 w-4" />
                                {project}
                              </CardTitle>
                            </CardHeader>
                            <CardContent className="p-3 pt-0 space-y-2">
                              {projectFiles.map((file) => {
                                const fileKey = `${file.project}/${file.filename}`
                                const isExpanded = expandedFiles.has(fileKey)
                                return (
                                  <div key={fileKey}>
                                    <Button
                                      variant="ghost"
                                      size="sm"
                                      className="w-full justify-between h-auto py-2"
                                      onClick={() => {
                                        const newExpanded = new Set(expandedFiles)
                                        if (isExpanded) {
                                          newExpanded.delete(fileKey)
                                        } else {
                                          newExpanded.add(fileKey)
                                        }
                                        setExpandedFiles(newExpanded)
                                      }}
                                    >
                                      <span className="flex items-center gap-2">
                                        <FileCode className="h-4 w-4" />
                                        {file.filename}
                                      </span>
                                      <ChevronRight className={cn(
                                        "h-4 w-4 transition-transform",
                                        isExpanded && "rotate-90"
                                      )} />
                                    </Button>
                                    {isExpanded && file.content && (
                                      <pre className="mt-2 p-3 bg-muted rounded-md text-xs overflow-auto max-h-64 whitespace-pre-wrap font-mono">
                                        {file.content}
                                      </pre>
                                    )}
                                  </div>
                                )
                              })}
                            </CardContent>
                          </Card>
                        )
                      })}
                    </div>
                  ) : (
                    <EmptyState
                      icon={FileCode}
                      title="No planning files loaded"
                      description="Click Refresh to load planning files for this goal"
                    />
                  )}
                </div>
              ) : (
                <EmptyState
                  icon={FileCode}
                  title="No projects"
                  description="This goal has no associated projects"
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

      <DeleteGoalDialog
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        goal={goal}
        onSuccess={() => {
          onRefresh()
          onOpenChange(false) // Close the sheet after deletion
        }}
      />
    </Sheet>
  )
}
