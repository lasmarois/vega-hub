import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from '@/components/ui/sheet'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { useMobile } from '@/hooks/useMobile'
import { EmptyState } from '@/components/shared/EmptyState'
import { FolderOpen, Target, Snowflake, CheckCircle2 } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { GoalSummary } from '@/lib/types'

interface ProjectStats {
  name: string
  active: number
  iced: number
  completed: number
}

interface ProjectSheetProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  project: ProjectStats | null
  goals: GoalSummary[]
  onGoalClick: (id: string) => void
}

export function ProjectSheet({ open, onOpenChange, project, goals, onGoalClick }: ProjectSheetProps) {
  const { isDesktop } = useMobile()

  if (!project) {
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

  const projectGoals = goals.filter(g => g.projects.includes(project.name))
  const activeGoals = projectGoals.filter(g => g.status === 'active')
  const icedGoals = projectGoals.filter(g => g.status === 'iced')
  const completedGoals = projectGoals.filter(g => g.status === 'completed')

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
              <FolderOpen className="h-5 w-5 text-muted-foreground" />
              <SheetTitle className="text-lg">{project.name}</SheetTitle>
            </div>
            <SheetDescription>
              {projectGoals.length} goal{projectGoals.length !== 1 ? 's' : ''}
            </SheetDescription>
          </SheetHeader>

          {/* Stats Row */}
          <div className="flex items-center gap-4 mt-4 text-sm">
            <div className="flex items-center gap-1.5">
              <Target className="h-4 w-4 text-primary" />
              <span>{project.active} active</span>
            </div>
            <div className="flex items-center gap-1.5">
              <Snowflake className="h-4 w-4 text-blue-500" />
              <span>{project.iced} iced</span>
            </div>
            <div className="flex items-center gap-1.5">
              <CheckCircle2 className="h-4 w-4 text-green-500" />
              <span>{project.completed} done</span>
            </div>
          </div>
        </div>

        {/* Tabs Content */}
        <Tabs defaultValue="active" className="flex-1 flex flex-col">
          <TabsList className="w-full justify-start rounded-none border-b bg-transparent p-0 h-auto">
            <TabsTrigger
              value="active"
              className="rounded-none border-b-2 border-transparent data-[state=active]:border-primary px-4 py-2"
            >
              Active
              <Badge variant="secondary" className="ml-1.5">{activeGoals.length}</Badge>
            </TabsTrigger>
            <TabsTrigger
              value="iced"
              className="rounded-none border-b-2 border-transparent data-[state=active]:border-primary px-4 py-2"
            >
              Iced
              <Badge variant="secondary" className="ml-1.5">{icedGoals.length}</Badge>
            </TabsTrigger>
            <TabsTrigger
              value="completed"
              className="rounded-none border-b-2 border-transparent data-[state=active]:border-primary px-4 py-2"
            >
              Completed
              <Badge variant="secondary" className="ml-1.5">{completedGoals.length}</Badge>
            </TabsTrigger>
          </TabsList>

          <ScrollArea className="flex-1">
            <TabsContent value="active" className="p-4 m-0">
              {activeGoals.length > 0 ? (
                <div className="space-y-2">
                  {activeGoals.map(goal => (
                    <GoalRow key={goal.id} goal={goal} onClick={() => onGoalClick(goal.id)} />
                  ))}
                </div>
              ) : (
                <EmptyState
                  icon={Target}
                  title="No active goals"
                  description="No goals are currently in progress for this project"
                />
              )}
            </TabsContent>

            <TabsContent value="iced" className="p-4 m-0">
              {icedGoals.length > 0 ? (
                <div className="space-y-2">
                  {icedGoals.map(goal => (
                    <GoalRow key={goal.id} goal={goal} onClick={() => onGoalClick(goal.id)} />
                  ))}
                </div>
              ) : (
                <EmptyState
                  icon={Snowflake}
                  title="No iced goals"
                  description="No goals are currently paused for this project"
                />
              )}
            </TabsContent>

            <TabsContent value="completed" className="p-4 m-0">
              {completedGoals.length > 0 ? (
                <div className="space-y-2">
                  {completedGoals.map(goal => (
                    <GoalRow key={goal.id} goal={goal} onClick={() => onGoalClick(goal.id)} />
                  ))}
                </div>
              ) : (
                <EmptyState
                  icon={CheckCircle2}
                  title="No completed goals"
                  description="No goals have been completed for this project yet"
                />
              )}
            </TabsContent>
          </ScrollArea>
        </Tabs>
      </SheetContent>
    </Sheet>
  )
}

function GoalRow({ goal, onClick }: { goal: GoalSummary; onClick: () => void }) {
  return (
    <Card
      className="cursor-pointer hover:bg-accent/50 transition-colors"
      onClick={onClick}
    >
      <CardHeader className="p-3 pb-1">
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm font-medium">#{goal.id}</CardTitle>
          <Badge variant={
            goal.status === 'completed' ? 'success' :
            goal.status === 'iced' ? 'secondary' :
            goal.executor_status === 'running' ? 'success' :
            goal.executor_status === 'waiting' ? 'destructive' :
            'secondary'
          } className="text-xs">
            {goal.status === 'completed' ? 'COMPLETE' :
             goal.status === 'iced' ? 'ICED' :
             goal.executor_status.toUpperCase()}
          </Badge>
        </div>
      </CardHeader>
      <CardContent className="p-3 pt-0">
        <p className="text-sm text-muted-foreground line-clamp-2">{goal.title}</p>
      </CardContent>
    </Card>
  )
}
