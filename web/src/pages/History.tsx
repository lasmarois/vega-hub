import { useState } from 'react'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import { Skeleton } from '@/components/ui/skeleton'
import { History as HistoryIcon, Search, CheckCircle2 } from 'lucide-react'
import type { GoalSummary } from '@/lib/types'

interface HistoryProps {
  goals: GoalSummary[]
  loading: boolean
  onGoalClick: (id: string) => void
}

export function History({ goals, loading, onGoalClick }: HistoryProps) {
  const [searchQuery, setSearchQuery] = useState('')

  const completedGoals = goals.filter(g => g.status === 'completed')

  const filteredGoals = searchQuery
    ? completedGoals.filter(g =>
        g.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
        g.id.toLowerCase().includes(searchQuery.toLowerCase()) ||
        g.projects.some(p => p.toLowerCase().includes(searchQuery.toLowerCase()))
      )
    : completedGoals

  if (loading) {
    return (
      <div className="p-4 space-y-4">
        <h1 className="text-2xl font-bold">History</h1>
        <Skeleton className="h-10 w-full" />
        {[1, 2, 3].map(i => (
          <Skeleton key={i} className="h-24" />
        ))}
      </div>
    )
  }

  return (
    <div className="p-4 space-y-4">
      <h1 className="text-2xl font-bold">History</h1>

      {/* Search */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          placeholder="Search completed goals..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="pl-10"
        />
      </div>

      {/* Results */}
      {completedGoals.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-8 text-center">
            <HistoryIcon className="h-12 w-12 text-muted-foreground/50 mb-4" />
            <p className="text-muted-foreground">No completed goals yet</p>
            <p className="text-sm text-muted-foreground">Completed goals will appear here</p>
          </CardContent>
        </Card>
      ) : filteredGoals.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-8 text-center">
            <Search className="h-12 w-12 text-muted-foreground/50 mb-4" />
            <p className="text-muted-foreground">No matching goals</p>
            <p className="text-sm text-muted-foreground">Try a different search term</p>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-3">
          {filteredGoals.map(goal => (
            <Card
              key={goal.id}
              className="cursor-pointer hover:bg-accent/50 transition-colors"
              onClick={() => onGoalClick(goal.id)}
            >
              <CardHeader className="p-4 pb-2">
                <div className="flex items-start justify-between gap-2">
                  <div className="flex items-center gap-2">
                    <CheckCircle2 className="h-4 w-4 text-green-500" />
                    <CardTitle className="text-base">#{goal.id}</CardTitle>
                  </div>
                  <Badge variant="success">COMPLETE</Badge>
                </div>
              </CardHeader>
              <CardContent className="p-4 pt-0">
                <p className="text-sm mb-2">{goal.title}</p>
                <div className="flex items-center gap-3 text-xs text-muted-foreground">
                  {goal.projects.length > 0 && (
                    <span>{goal.projects.join(', ')}</span>
                  )}
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}
