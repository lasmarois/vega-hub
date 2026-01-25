import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { FolderOpen, Target, Snowflake, CheckCircle2 } from 'lucide-react'
import type { GoalSummary } from '@/lib/types'

export interface ProjectStats {
  name: string
  active: number
  iced: number
  completed: number
}

interface ProjectsProps {
  goals: GoalSummary[]
  loading: boolean
  onProjectClick: (project: ProjectStats) => void
}

export function Projects({ goals, loading, onProjectClick }: ProjectsProps) {
  // Derive projects from goals
  const projectMap = new Map<string, ProjectStats>()

  goals.forEach(goal => {
    goal.projects.forEach(project => {
      if (!projectMap.has(project)) {
        projectMap.set(project, { name: project, active: 0, iced: 0, completed: 0 })
      }
      const stats = projectMap.get(project)!
      if (goal.status === 'active') stats.active++
      else if (goal.status === 'iced') stats.iced++
      else if (goal.status === 'completed') stats.completed++
    })
  })

  const projects = Array.from(projectMap.values())

  if (loading) {
    return (
      <div className="p-4 space-y-4">
        <h1 className="text-2xl font-bold">Projects</h1>
        <div className="grid gap-4 sm:grid-cols-2">
          {[1, 2, 3, 4].map(i => (
            <Skeleton key={i} className="h-32" />
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className="p-4 space-y-4">
      <h1 className="text-2xl font-bold">Projects</h1>

      {projects.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-8 text-center">
            <FolderOpen className="h-12 w-12 text-muted-foreground/50 mb-4" />
            <p className="text-muted-foreground">No projects found</p>
            <p className="text-sm text-muted-foreground">Projects are detected from goals</p>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2">
          {projects.map(project => (
            <Card
              key={project.name}
              className="cursor-pointer hover:bg-accent/50 transition-colors"
              onClick={() => onProjectClick(project)}
            >
              <CardHeader className="p-4 pb-2">
                <div className="flex items-center gap-2">
                  <FolderOpen className="h-5 w-5 text-muted-foreground" />
                  <CardTitle className="text-base">{project.name}</CardTitle>
                </div>
                <CardDescription>
                  {project.active + project.iced + project.completed} total goals
                </CardDescription>
              </CardHeader>
              <CardContent className="p-4 pt-0">
                <div className="flex items-center gap-4 text-sm">
                  <div className="flex items-center gap-1.5">
                    <Target className="h-4 w-4 text-primary" />
                    <span>{project.active}</span>
                  </div>
                  <div className="flex items-center gap-1.5">
                    <Snowflake className="h-4 w-4 text-blue-500" />
                    <span>{project.iced}</span>
                  </div>
                  <div className="flex items-center gap-1.5">
                    <CheckCircle2 className="h-4 w-4 text-green-500" />
                    <span>{project.completed}</span>
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}
