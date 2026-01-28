import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { AlertTriangle, CheckCircle2, Ban, GitFork, Network } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { GoalSummary } from '@/lib/types'

// Parse phase string like "2/4", "1/?", or "Phase 2" into { current, total }
function parsePhase(phase: string): { current: number; total: number } | null {
  // Try "X/Y" format (e.g., "2/4")
  const slashMatch = phase.match(/(\d+)\s*\/\s*(\d+)/)
  if (slashMatch) {
    return { current: parseInt(slashMatch[1]), total: parseInt(slashMatch[2]) }
  }
  // Try "X/?" format (e.g., "1/?") - unknown total
  const unknownMatch = phase.match(/(\d+)\s*\/\s*\?/)
  if (unknownMatch) {
    return { current: parseInt(unknownMatch[1]), total: 0 }
  }
  // Try "Phase X" format
  const phaseMatch = phase.match(/phase\s*(\d+)/i)
  if (phaseMatch) {
    return { current: parseInt(phaseMatch[1]), total: 0 }
  }
  return null
}

interface GoalCardProps {
  goal: GoalSummary
  onClick: () => void
}

export function GoalCard({ goal, onClick }: GoalCardProps) {
  const phaseInfo = parsePhase(goal.phase)
  const completionStatus = goal.completion_status

  // Determine if goal appears ready to complete
  const appearsComplete = completionStatus?.complete && goal.status === 'active'
  const hasHighConfidence = completionStatus && completionStatus.confidence >= 0.7 && !completionStatus.complete
  const confidencePct = completionStatus ? Math.round(completionStatus.confidence * 100) : 0

  // Determine border state (priority: questions > blocked > running > idle)
  const borderState = goal.pending_questions > 0 ? 'needs-attention' :
                      goal.is_blocked ? 'blocked' :
                      goal.executor_status === 'running' ? 'running' :
                      goal.executor_status === 'waiting' ? 'needs-attention' : 'idle'

  return (
    <Card
      className={cn(
        "cursor-pointer hover:bg-accent/50 transition-all border-l-4",
        goal.status === 'completed' && "opacity-60 border-l-green-500",
        goal.status !== 'completed' && borderState === 'needs-attention' && "border-l-red-500",
        goal.status !== 'completed' && borderState === 'blocked' && "border-l-yellow-500 opacity-80",
        goal.status !== 'completed' && borderState === 'running' && "border-l-green-500",
        goal.status !== 'completed' && borderState === 'idle' && "border-l-border",
        appearsComplete && "border-l-green-500 bg-green-50/50 dark:bg-green-950/20 shadow-md shadow-green-500/10"
      )}
      onClick={onClick}
    >
      <CardHeader className="p-4 pb-2">
        <div className="flex items-start justify-between gap-2">
          <div className="flex items-center gap-2">
            <CardTitle className="text-base font-semibold flex items-center gap-1.5">
              #{goal.id}
              {goal.has_children && (
                <span title="Has sub-goals">
                  <Network className="h-4 w-4 text-primary" />
                </span>
              )}
            </CardTitle>
            {/* Project badge - more prominent */}
            {goal.projects.length > 0 && (
              <Badge variant="outline" className="text-xs font-medium">
                {goal.projects.join(', ')}
                {goal.workspace_status && goal.workspace_status !== 'ready' && (
                  <AlertTriangle className="h-3 w-3 ml-1 text-yellow-500" />
                )}
              </Badge>
            )}
          </div>
          <div className="flex items-center gap-2">
            {goal.is_blocked && (
              <Badge variant="warning" className="gap-1" title={goal.blockers?.length ? `Blocked by: ${goal.blockers.join(', ')}` : 'Blocked by dependencies'}>
                <Ban className="h-3 w-3" />
                {goal.blockers?.length ? `Blocked by ${goal.blockers.length}` : 'Blocked'}
              </Badge>
            )}
            {(goal.parent_id || goal.has_children) && (
              <Badge variant="outline" className="gap-1">
                <GitFork className="h-3 w-3" />
                {goal.parent_id && goal.has_children ? 'Child+' : goal.parent_id ? 'Child' : 'Parent'}
              </Badge>
            )}
            {goal.pending_questions > 0 && (
              <Badge variant="destructive" className="gap-1">
                {goal.pending_questions} Q
              </Badge>
            )}
            {/* Only show badge for actionable/notable states - not idle/stopped */}
            {goal.status === 'completed' && (
              <Badge variant="success">COMPLETE</Badge>
            )}
            {goal.status === 'iced' && (
              <Badge variant="secondary">ICED</Badge>
            )}
            {goal.status === 'active' && goal.executor_status === 'running' && (
              <Badge variant="success">RUNNING</Badge>
            )}
            {goal.status === 'active' && goal.executor_status === 'waiting' && (
              <Badge variant="destructive">WAITING</Badge>
            )}
          </div>
        </div>
      </CardHeader>
      <CardContent className="p-4 pt-0 space-y-3">
        <p className="text-sm font-medium">{goal.title}</p>

        {/* Ready to Complete banner - prominent when ready */}
        {appearsComplete && (
          <div className="flex items-center gap-2 p-2 bg-green-100 dark:bg-green-900/30 border border-green-200 dark:border-green-800 rounded-md">
            <CheckCircle2 className="h-4 w-4 text-green-600 dark:text-green-400" />
            <span className="text-sm font-medium text-green-700 dark:text-green-300">Ready to Complete</span>
            {completionStatus && completionStatus.total_phases > 0 && (
              <Badge variant="success" className="ml-auto">
                {completionStatus.completed_phases}/{completionStatus.total_phases} phases
              </Badge>
            )}
          </div>
        )}

        {/* High confidence banner - almost ready */}
        {hasHighConfidence && (
          <div className="flex items-center gap-2 p-2 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-md">
            <AlertTriangle className="h-4 w-4 text-yellow-600 dark:text-yellow-400" />
            <span className="text-sm font-medium text-yellow-700 dark:text-yellow-300">Almost Ready</span>
            <span className="text-xs text-yellow-600 dark:text-yellow-400 ml-auto">{confidencePct}% confidence</span>
          </div>
        )}

        {/* Progress section for non-ready active goals */}
        {goal.status === 'active' && !appearsComplete && completionStatus && (
          <div className="space-y-1.5">
            <div className="flex items-center justify-between text-xs text-muted-foreground">
              <span className="flex items-center gap-2">
                {completionStatus.total_phases > 0 ? (
                  <span>{completionStatus.completed_phases}/{completionStatus.total_phases} phases</span>
                ) : (
                  <span>Phase: {goal.phase}</span>
                )}
              </span>
              {confidencePct > 0 && (
                <span>{confidencePct}% complete</span>
              )}
            </div>
            {/* Confidence progress bar */}
            {confidencePct > 0 && (
              <div className="h-1.5 bg-muted rounded-full overflow-hidden">
                <div 
                  className={cn(
                    "h-full rounded-full transition-all",
                    confidencePct >= 70 ? "bg-green-500" : 
                    confidencePct >= 40 ? "bg-yellow-500" : "bg-muted-foreground"
                  )} 
                  style={{ width: `${confidencePct}%` }}
                />
              </div>
            )}
          </div>
        )}

        {/* Fallback phase display for goals without completion status */}
        {goal.status === 'active' && !completionStatus && (
          <div className="text-xs text-muted-foreground">
            {phaseInfo && phaseInfo.total > 0 ? (
              <div className="space-y-1.5">
                <div className="flex justify-between">
                  <span>Phase {phaseInfo.current}/{phaseInfo.total}</span>
                </div>
                <Progress value={phaseInfo.current} max={phaseInfo.total} />
              </div>
            ) : (
              <span>Phase: {goal.phase}</span>
            )}
          </div>
        )}

        {/* Iced/completed goals - simpler display */}
        {(goal.status === 'iced' || goal.status === 'completed') && (
          <div className="text-xs text-muted-foreground">
            Phase: {goal.phase}
          </div>
        )}
      </CardContent>
    </Card>
  )
}
