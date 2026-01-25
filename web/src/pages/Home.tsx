import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Target, Snowflake, CheckCircle2, FolderOpen, AlertCircle } from 'lucide-react'
import type { GoalSummary } from '@/lib/types'

interface HomeProps {
  goals: GoalSummary[]
  loading: boolean
  pendingQuestions: number
  onGoalClick: (id: string) => void
}

export function Home({ goals, loading, pendingQuestions, onGoalClick }: HomeProps) {
  const activeGoals = goals.filter(g => g.status === 'active')
  const icedGoals = goals.filter(g => g.status === 'iced')
  const completedGoals = goals.filter(g => g.status === 'completed')
  const projects = [...new Set(goals.flatMap(g => g.projects))]

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

      {/* Active Goals */}
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
        ) : (
          <div className="space-y-3">
            {activeGoals.slice(0, 5).map(goal => (
              <Card
                key={goal.id}
                className="cursor-pointer hover:bg-accent/50 transition-colors"
                onClick={() => onGoalClick(goal.id)}
              >
                <CardHeader className="p-4 pb-2">
                  <div className="flex items-start justify-between gap-2">
                    <div className="flex items-center gap-2">
                      <div className={`h-2 w-2 rounded-full ${
                        goal.executor_status === 'running' ? 'bg-green-500 animate-pulse' :
                        goal.executor_status === 'waiting' ? 'bg-red-500' :
                        'bg-muted-foreground'
                      }`} />
                      <CardTitle className="text-base">#{goal.id}</CardTitle>
                    </div>
                    <div className="flex items-center gap-2">
                      <Badge variant={
                        goal.executor_status === 'running' ? 'success' :
                        goal.executor_status === 'waiting' ? 'destructive' :
                        'secondary'
                      }>
                        {goal.executor_status.toUpperCase()}
                      </Badge>
                      {goal.pending_questions > 0 && (
                        <Badge variant="destructive">{goal.pending_questions}</Badge>
                      )}
                    </div>
                  </div>
                </CardHeader>
                <CardContent className="p-4 pt-0">
                  <p className="text-sm mb-2">{goal.title}</p>
                  <div className="flex items-center gap-3 text-xs text-muted-foreground">
                    <span>Phase: {goal.phase}</span>
                    {goal.projects.length > 0 && (
                      <span>{goal.projects.join(', ')}</span>
                    )}
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </section>
    </div>
  )
}
