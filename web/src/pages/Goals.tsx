import { useState, useMemo } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Target, ChevronDown } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { GoalSummary } from '@/lib/types'

interface GoalsProps {
  goals: GoalSummary[]
  loading: boolean
  onGoalClick: (id: string) => void
}

type FilterType = 'all' | 'active' | 'iced' | 'completed'
type SortType = 'newest' | 'oldest' | 'status' | 'questions'

export function Goals({ goals, loading, onGoalClick }: GoalsProps) {
  const [filter, setFilter] = useState<FilterType>('all')
  const [sort, setSort] = useState<SortType>('newest')
  const [showCompleted, setShowCompleted] = useState(false)

  const filterOptions: { value: FilterType; label: string; count: number }[] = [
    { value: 'all', label: 'All', count: goals.length },
    { value: 'active', label: 'Active', count: goals.filter(g => g.status === 'active').length },
    { value: 'iced', label: 'Iced', count: goals.filter(g => g.status === 'iced').length },
    { value: 'completed', label: 'Completed', count: goals.filter(g => g.status === 'completed').length },
  ]

  const filteredGoals = filter === 'all'
    ? goals
    : goals.filter(g => g.status === filter)

  const sortedGoals = useMemo(() => {
    const sorted = [...filteredGoals]
    switch (sort) {
      case 'newest':
        // Assuming goals are added with increasing IDs or we'd need a created_at field
        sorted.reverse()
        break
      case 'oldest':
        // Keep original order
        break
      case 'status':
        sorted.sort((a, b) => {
          const statusOrder = { running: 0, waiting: 1, stopped: 2, idle: 3 }
          return (statusOrder[a.executor_status] || 3) - (statusOrder[b.executor_status] || 3)
        })
        break
      case 'questions':
        sorted.sort((a, b) => b.pending_questions - a.pending_questions)
        break
    }
    return sorted
  }, [filteredGoals, sort])

  const activeAndIcedGoals = sortedGoals.filter(g => g.status !== 'completed')
  const completedGoals = sortedGoals.filter(g => g.status === 'completed')

  if (loading) {
    return (
      <div className="p-4 space-y-4">
        <div className="flex gap-2">
          {[1, 2, 3, 4].map(i => (
            <Skeleton key={i} className="h-8 w-20" />
          ))}
        </div>
        {[1, 2, 3].map(i => (
          <Skeleton key={i} className="h-24" />
        ))}
      </div>
    )
  }

  return (
    <div className="p-4 space-y-4">
      <h1 className="text-2xl font-bold">Goals</h1>

      {/* Filter Chips & Sort */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="flex flex-wrap gap-2">
          {filterOptions.map(option => (
            <Button
              key={option.value}
              variant={filter === option.value ? 'default' : 'outline'}
              size="sm"
              onClick={() => setFilter(option.value)}
              className="gap-1.5"
            >
              {option.label}
              <Badge variant={filter === option.value ? 'secondary' : 'outline'} className="ml-1">
                {option.count}
              </Badge>
            </Button>
          ))}
        </div>
        <Select value={sort} onValueChange={(v) => setSort(v as SortType)}>
          <SelectTrigger className="w-[140px]">
            <SelectValue placeholder="Sort by" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="newest">Newest</SelectItem>
            <SelectItem value="oldest">Oldest</SelectItem>
            <SelectItem value="status">Status</SelectItem>
            <SelectItem value="questions">Questions</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {/* Goals List */}
      {filteredGoals.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-8 text-center">
            <Target className="h-12 w-12 text-muted-foreground/50 mb-4" />
            <p className="text-muted-foreground">No {filter === 'all' ? '' : filter} goals found</p>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-3">
          {/* Active/Iced Goals */}
          {activeAndIcedGoals.map(goal => (
            <GoalCard key={goal.id} goal={goal} onClick={() => onGoalClick(goal.id)} />
          ))}

          {/* Completed Goals (Collapsible) */}
          {completedGoals.length > 0 && (filter === 'all' || filter === 'completed') && (
            <div className="space-y-3">
              {filter === 'all' && (
                <Button
                  variant="ghost"
                  className="w-full justify-between"
                  onClick={() => setShowCompleted(!showCompleted)}
                >
                  <span className="flex items-center gap-2">
                    Completed
                    <Badge variant="secondary">{completedGoals.length}</Badge>
                  </span>
                  <ChevronDown className={cn(
                    "h-4 w-4 transition-transform",
                    showCompleted && "rotate-180"
                  )} />
                </Button>
              )}

              {(filter === 'completed' || showCompleted) && completedGoals.map(goal => (
                <GoalCard key={goal.id} goal={goal} onClick={() => onGoalClick(goal.id)} />
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

function GoalCard({ goal, onClick }: { goal: GoalSummary; onClick: () => void }) {
  return (
    <Card
      className={cn(
        "cursor-pointer hover:bg-accent/50 transition-colors",
        goal.status === 'completed' && "opacity-60"
      )}
      onClick={onClick}
    >
      <CardHeader className="p-4 pb-2">
        <div className="flex items-start justify-between gap-2">
          <div className="flex items-center gap-2">
            {goal.status !== 'completed' && (
              <div className={`h-2 w-2 rounded-full ${
                goal.executor_status === 'running' ? 'bg-green-500 animate-pulse' :
                goal.executor_status === 'waiting' ? 'bg-red-500' :
                'bg-muted-foreground'
              }`} />
            )}
            {goal.status === 'completed' && (
              <div className="h-2 w-2 rounded-full bg-green-500" />
            )}
            <CardTitle className="text-base">#{goal.id}</CardTitle>
          </div>
          <div className="flex items-center gap-2">
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
  )
}
