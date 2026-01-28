import { Card, CardContent } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Badge } from '@/components/ui/badge'
import { Target, Snowflake, CheckCircle2, FolderOpen, AlertCircle, Activity, MessageCircle, Ban } from 'lucide-react'
import { ActivityItem } from '@/components/shared/ActivityItem'
import { EmptyState } from '@/components/shared/EmptyState'
import { GoalCard } from '@/components/goals'
import type { GoalSummary, Activity as ActivityType } from '@/lib/types'

interface HomeProps {
  goals: GoalSummary[]
  loading: boolean
  pendingQuestions: number
  activities: ActivityType[]
  onGoalClick: (id: string) => void
}

export function Home({ goals, loading, pendingQuestions, activities, onGoalClick }: HomeProps) {
  const activeGoals = goals.filter(g => g.status === 'active')
  const icedGoals = goals.filter(g => g.status === 'iced')
  const completedGoals = goals.filter(g => g.status === 'completed')
  const projects = [...new Set(goals.flatMap(g => g.projects))]

  // Needs Attention: goals that need user action
  const waitingOnYou = activeGoals.filter(g => g.pending_questions > 0)
  const blockedGoals = activeGoals.filter(g => g.is_blocked && g.pending_questions === 0)
  const readyToComplete = activeGoals.filter(g => g.completion_status?.complete && !g.is_blocked)
  const runningGoals = activeGoals.filter(g => g.executor_status === 'running')
  
  // Other active goals (not needing attention)
  const otherActiveGoals = activeGoals.filter(g => 
    g.pending_questions === 0 && 
    !g.is_blocked && 
    !g.completion_status?.complete &&
    g.executor_status !== 'running'
  )

  const needsAttentionCount = waitingOnYou.length + blockedGoals.length + readyToComplete.length

  if (loading) {
    return (
      <div className="p-4 space-y-6">
        <Skeleton className="h-16 w-full" />
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
          {[1, 2, 3, 4].map(i => (
            <Skeleton key={i} className="h-24" />
          ))}
        </div>
        <Skeleton className="h-48" />
      </div>
    )
  }

  return (
    <div className="p-4 space-y-6">
      {/* Alert Banner for Pending Questions */}
      {pendingQuestions > 0 && (
        <Card className="border-destructive bg-destructive/10">
          <CardContent className="flex items-center gap-4 p-4">
            <AlertCircle className="h-6 w-6 text-destructive" />
            <div className="flex-1">
              <p className="font-medium text-destructive">
                {pendingQuestions} question{pendingQuestions > 1 ? 's' : ''} need{pendingQuestions === 1 ? 's' : ''} your attention
              </p>
              <p className="text-sm text-muted-foreground">
                Executors are waiting for answers to continue
              </p>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Stats Row */}
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-4">
        <Card>
          <CardContent className="flex items-center gap-3 p-4">
            <div className="rounded-lg bg-primary/10 p-2">
              <Target className="h-5 w-5 text-primary" />
            </div>
            <div>
              <p className="text-2xl font-bold">{activeGoals.length}</p>
              <p className="text-sm text-muted-foreground">Active</p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="flex items-center gap-3 p-4">
            <div className="rounded-lg bg-blue-500/10 p-2">
              <Snowflake className="h-5 w-5 text-blue-500" />
            </div>
            <div>
              <p className="text-2xl font-bold">{icedGoals.length}</p>
              <p className="text-sm text-muted-foreground">Iced</p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="flex items-center gap-3 p-4">
            <div className="rounded-lg bg-green-500/10 p-2">
              <CheckCircle2 className="h-5 w-5 text-green-500" />
            </div>
            <div>
              <p className="text-2xl font-bold">{completedGoals.length}</p>
              <p className="text-sm text-muted-foreground">Completed</p>
            </div>
          </CardContent>
        </Card>

        <Card>
          <CardContent className="flex items-center gap-3 p-4">
            <div className="rounded-lg bg-orange-500/10 p-2">
              <FolderOpen className="h-5 w-5 text-orange-500" />
            </div>
            <div>
              <p className="text-2xl font-bold">{projects.length}</p>
              <p className="text-sm text-muted-foreground">Projects</p>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Needs Attention */}
      {needsAttentionCount > 0 && (
        <section>
          <div className="flex items-center gap-2 mb-4">
            <h2 className="text-lg font-semibold">Needs Attention</h2>
            <Badge variant="destructive">{needsAttentionCount}</Badge>
          </div>
          <div className="space-y-3">
            {/* Waiting on You - highest priority */}
            {waitingOnYou.map(goal => (
              <div key={goal.id} className="relative">
                <div className="absolute -left-2 top-3 flex items-center gap-1 text-xs text-red-500">
                  <MessageCircle className="h-3 w-3" />
                </div>
                <GoalCard goal={goal} onClick={() => onGoalClick(goal.id)} />
              </div>
            ))}
            {/* Blocked */}
            {blockedGoals.map(goal => (
              <div key={goal.id} className="relative">
                <div className="absolute -left-2 top-3 flex items-center gap-1 text-xs text-yellow-500">
                  <Ban className="h-3 w-3" />
                </div>
                <GoalCard goal={goal} onClick={() => onGoalClick(goal.id)} />
              </div>
            ))}
            {/* Ready to Complete */}
            {readyToComplete.map(goal => (
              <div key={goal.id} className="relative">
                <div className="absolute -left-2 top-3 flex items-center gap-1 text-xs text-green-500">
                  <CheckCircle2 className="h-3 w-3" />
                </div>
                <GoalCard goal={goal} onClick={() => onGoalClick(goal.id)} />
              </div>
            ))}
          </div>
        </section>
      )}

      {/* Running Goals */}
      {runningGoals.length > 0 && (
        <section>
          <h2 className="text-lg font-semibold mb-4">In Progress</h2>
          <div className="space-y-3">
            {runningGoals.map(goal => (
              <GoalCard key={goal.id} goal={goal} onClick={() => onGoalClick(goal.id)} />
            ))}
          </div>
        </section>
      )}

      {/* Other Active Goals */}
      <section>
        <h2 className="text-lg font-semibold mb-4">Active Goals</h2>
        {activeGoals.length === 0 ? (
          <Card>
            <CardContent className="flex flex-col items-center justify-center py-8 text-center">
              <Target className="h-12 w-12 text-muted-foreground/50 mb-4" />
              <p className="text-muted-foreground">No active goals</p>
              <p className="text-sm text-muted-foreground">Create a goal to get started</p>
            </CardContent>
          </Card>
        ) : otherActiveGoals.length === 0 && needsAttentionCount > 0 ? (
          <Card>
            <CardContent className="flex flex-col items-center justify-center py-6 text-center">
              <p className="text-sm text-muted-foreground">All active goals are shown above</p>
            </CardContent>
          </Card>
        ) : (
          <div className="space-y-3">
            {otherActiveGoals.slice(0, 5).map(goal => (
              <GoalCard key={goal.id} goal={goal} onClick={() => onGoalClick(goal.id)} />
            ))}
          </div>
        )}
      </section>

      {/* Recent Activity */}
      <section>
        <h2 className="text-lg font-semibold mb-4">Recent Activity</h2>
        <Card>
          <CardContent className="p-4">
            {activities.length === 0 ? (
              <EmptyState
                icon={Activity}
                title="No recent activity"
                description="Activity from executors will appear here"
                className="py-6"
              />
            ) : (
              <div className="divide-y divide-border">
                {activities.slice(0, 10).map((activity) => (
                  <ActivityItem
                    key={activity.id}
                    activity={activity}
                    onClick={activity.goal_id ? () => onGoalClick(activity.goal_id!) : undefined}
                  />
                ))}
              </div>
            )}
          </CardContent>
        </Card>
      </section>
    </div>
  )
}
