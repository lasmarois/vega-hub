import { useState, useMemo, useEffect } from 'react'
import { Card, CardContent } from '@/components/ui/card'
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
import { Target, ChevronDown, Plus, GitFork, CheckCircle, ChevronRight } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { GoalSummary, Project } from '@/lib/types'
import { CreateGoalDialog, GoalCard } from '@/components/goals'

interface GoalsProps {
  goals: GoalSummary[]
  loading: boolean
  onGoalClick: (id: string) => void
  onRefresh: () => void
}

type FilterType = 'all' | 'active' | 'iced' | 'completed' | 'ready'
type SortType = 'newest' | 'oldest' | 'status' | 'questions'

export function Goals({ goals, loading, onGoalClick, onRefresh }: GoalsProps) {
  const [filter, setFilter] = useState<FilterType>('all')
  const [sort, setSort] = useState<SortType>('newest')
  const [showCompleted, setShowCompleted] = useState(false)
  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const [projects, setProjects] = useState<Project[]>([])
  const [treeView, setTreeView] = useState(false)
  const [expandedGoals, setExpandedGoals] = useState<Set<string>>(new Set())

  // Fetch projects for the create dialog
  useEffect(() => {
    fetch('/api/projects')
      .then(res => res.json())
      .then(data => setProjects(data))
      .catch(err => console.error('Failed to load projects:', err))
  }, [])

  const readyGoals = goals.filter(g => g.status === 'active' && !g.is_blocked)

  const filterOptions: { value: FilterType; label: string; count: number }[] = [
    { value: 'all', label: 'All', count: goals.length },
    { value: 'active', label: 'Active', count: goals.filter(g => g.status === 'active').length },
    { value: 'ready', label: 'Workable', count: readyGoals.length },
    { value: 'iced', label: 'Iced', count: goals.filter(g => g.status === 'iced').length },
    { value: 'completed', label: 'Completed', count: goals.filter(g => g.status === 'completed').length },
  ]

  const filteredGoals = useMemo(() => {
    if (filter === 'all') return goals
    if (filter === 'ready') return readyGoals
    return goals.filter(g => g.status === filter)
  }, [goals, filter, readyGoals])

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

  // Build tree structure for tree view
  const { rootGoals, childrenMap } = useMemo(() => {
    if (!treeView) return { rootGoals: activeAndIcedGoals, childrenMap: new Map() }
    
    const childrenMap = new Map<string, GoalSummary[]>()
    const rootGoals: GoalSummary[] = []
    
    // First pass: group children by parent
    activeAndIcedGoals.forEach(goal => {
      if (goal.parent_id) {
        const children = childrenMap.get(goal.parent_id) || []
        children.push(goal)
        childrenMap.set(goal.parent_id, children)
      }
    })
    
    // Second pass: find roots (no parent or parent not in list)
    const goalIds = new Set(activeAndIcedGoals.map(g => g.id))
    activeAndIcedGoals.forEach(goal => {
      if (!goal.parent_id || !goalIds.has(goal.parent_id)) {
        rootGoals.push(goal)
      }
    })
    
    return { rootGoals, childrenMap }
  }, [activeAndIcedGoals, treeView])

  // Expand/collapse all
  const expandAll = () => {
    const allParents = new Set<string>()
    childrenMap.forEach((_, parentId) => allParents.add(parentId))
    setExpandedGoals(allParents)
  }
  
  const collapseAll = () => setExpandedGoals(new Set())

  // Recursive tree node component
  const renderGoalTree = (goal: GoalSummary, depth: number = 0, isLast: boolean = true) => {
    const children = childrenMap.get(goal.id) || []
    const hasChildren = children.length > 0
    const isExpanded = expandedGoals.has(goal.id)
    
    const toggleExpand = (e: React.MouseEvent) => {
      e.stopPropagation()
      const newExpanded = new Set(expandedGoals)
      if (isExpanded) {
        newExpanded.delete(goal.id)
      } else {
        newExpanded.add(goal.id)
      }
      setExpandedGoals(newExpanded)
    }
    
    return (
      <div key={goal.id} className="relative">
        <div 
          className="flex items-start gap-1"
          style={{ paddingLeft: depth * 24 }}
        >
          {/* Tree connector for children */}
          {depth > 0 && (
            <div className="absolute flex items-center" style={{ left: (depth - 1) * 24 + 12 }}>
              {/* Vertical line */}
              <div 
                className={cn(
                  "absolute w-px bg-border",
                  isLast ? "h-7" : "h-full"
                )}
                style={{ top: 0 }}
              />
              {/* Horizontal connector */}
              <div 
                className="absolute w-3 h-px bg-border"
                style={{ top: 28, left: 0 }}
              />
            </div>
          )}
          
          {/* Expand/collapse button */}
          {hasChildren ? (
            <button
              onClick={toggleExpand}
              className={cn(
                "mt-2.5 p-1 rounded transition-all duration-150 z-10 relative shrink-0",
                "hover:bg-accent active:scale-95",
                isExpanded && "bg-accent/50"
              )}
              aria-label={isExpanded ? "Collapse" : "Expand"}
            >
              <ChevronRight className={cn(
                "h-4 w-4 transition-transform duration-150",
                isExpanded ? "rotate-90 text-foreground" : "text-muted-foreground"
              )} />
              {/* Child count badge */}
              {!isExpanded && (
                <span className="absolute -top-1 -right-1 text-[10px] bg-primary text-primary-foreground rounded-full min-w-[16px] h-4 flex items-center justify-center font-medium px-1">
                  {children.length}
                </span>
              )}
            </button>
          ) : depth > 0 ? (
            <div className="w-6 shrink-0" />
          ) : null}
          
          <div className="flex-1 min-w-0">
            <GoalCard goal={goal} onClick={() => onGoalClick(goal.id)} />
          </div>
        </div>
        
        {/* Children */}
        {hasChildren && isExpanded && (
          <div className="mt-2 space-y-2 relative">
            {/* Continuous vertical line for children */}
            <div 
              className="absolute w-px bg-border"
              style={{ 
                left: depth * 24 + 12,
                top: 0,
                bottom: 8
              }}
            />
            {children.map((child: GoalSummary, idx: number) => 
              renderGoalTree(child, depth + 1, idx === children.length - 1)
            )}
          </div>
        )}
      </div>
    )
  }

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
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Goals</h1>
        <Button onClick={() => setCreateDialogOpen(true)} className="gap-2">
          <Plus className="h-4 w-4" />
          New Goal
        </Button>
      </div>

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
              {option.value === 'ready' && <CheckCircle className="h-3 w-3" />}
              <Badge variant={filter === option.value ? 'secondary' : 'outline'} className="ml-1">
                {option.count}
              </Badge>
            </Button>
          ))}
        </div>
        <div className="flex items-center gap-2">
          <div className="flex items-center border rounded-lg overflow-hidden">
            <Button
              variant={treeView ? 'default' : 'ghost'}
              size="sm"
              onClick={() => setTreeView(!treeView)}
              className="gap-1.5 rounded-none border-0"
              title="Toggle tree view"
            >
              <GitFork className="h-4 w-4" />
              Tree
            </Button>
            {treeView && childrenMap.size > 0 && (
              <>
                <div className="w-px h-5 bg-border" />
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={expandAll}
                  className="rounded-none border-0 px-2 h-8"
                  title="Expand all"
                >
                  <ChevronDown className="h-3.5 w-3.5" />
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={collapseAll}
                  className="rounded-none border-0 px-2 h-8"
                  title="Collapse all"
                >
                  <ChevronRight className="h-3.5 w-3.5" />
                </Button>
              </>
            )}
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
          {treeView ? (
            // Tree view
            rootGoals.map(goal => renderGoalTree(goal))
          ) : (
            // Flat view
            activeAndIcedGoals.map(goal => (
              <GoalCard key={goal.id} goal={goal} onClick={() => onGoalClick(goal.id)} />
            ))
          )}

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

      <CreateGoalDialog
        open={createDialogOpen}
        onOpenChange={setCreateDialogOpen}
        projects={projects}
        onSuccess={onRefresh}
      />
    </div>
  )
}
