import { useEffect, useState, useCallback } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  FolderOpen,
  Target,
  Snowflake,
  CheckCircle2,
  AlertTriangle,
  XCircle,
  Plus,
  MoreVertical,
  Trash2,
} from 'lucide-react'
import type { GoalSummary, Project } from '@/lib/types'
import { AddProjectDialog } from '@/components/projects/AddProjectDialog'
import { RemoveProjectDialog } from '@/components/projects/RemoveProjectDialog'

export interface ProjectStats {
  name: string
  active: number
  iced: number
  completed: number
  workspace_status?: 'ready' | 'missing' | 'error'
  workspace_error?: string
}

interface ProjectsProps {
  goals: GoalSummary[]
  loading: boolean
  onProjectClick: (project: ProjectStats) => void
}

export function Projects({ goals, loading, onProjectClick }: ProjectsProps) {
  const [projectsData, setProjectsData] = useState<Project[]>([])
  const [addDialogOpen, setAddDialogOpen] = useState(false)
  const [removeDialogOpen, setRemoveDialogOpen] = useState(false)
  const [selectedProject, setSelectedProject] = useState<string | null>(null)

  const fetchProjects = useCallback(() => {
    fetch('/api/projects')
      .then(res => res.json())
      .then(data => setProjectsData(data))
      .catch(() => setProjectsData([]))
  }, [])

  // Fetch projects with workspace status
  useEffect(() => {
    fetchProjects()
  }, [fetchProjects])

  // Derive project stats from goals and merge with workspace status
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

      // Use workspace status from goal if not already set
      if (!stats.workspace_status && goal.workspace_status) {
        stats.workspace_status = goal.workspace_status
        stats.workspace_error = goal.workspace_error
      }
    })
  })

  // Merge workspace status from projects API (overrides goal-derived status)
  projectsData.forEach(p => {
    if (projectMap.has(p.name)) {
      const stats = projectMap.get(p.name)!
      stats.workspace_status = p.workspace_status
      stats.workspace_error = p.workspace_error
    } else {
      // Project exists in config but has no goals
      projectMap.set(p.name, {
        name: p.name,
        active: 0,
        iced: 0,
        completed: 0,
        workspace_status: p.workspace_status,
        workspace_error: p.workspace_error,
      })
    }
  })

  const projects = Array.from(projectMap.values())

  const handleRemoveClick = (e: React.MouseEvent, projectName: string) => {
    e.stopPropagation()
    setSelectedProject(projectName)
    setRemoveDialogOpen(true)
  }

  const handleAddSuccess = () => {
    fetchProjects()
  }

  const handleRemoveSuccess = () => {
    fetchProjects()
    setSelectedProject(null)
  }

  if (loading) {
    return (
      <div className="p-4 space-y-4">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold">Projects</h1>
        </div>
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
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Projects</h1>
        <Button onClick={() => setAddDialogOpen(true)}>
          <Plus className="h-4 w-4 mr-2" />
          Add Project
        </Button>
      </div>

      {projects.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-8 text-center">
            <FolderOpen className="h-12 w-12 text-muted-foreground/50 mb-4" />
            <p className="text-muted-foreground">No projects found</p>
            <p className="text-sm text-muted-foreground mb-4">
              Add a local repository to get started
            </p>
            <Button onClick={() => setAddDialogOpen(true)}>
              <Plus className="h-4 w-4 mr-2" />
              Add Project
            </Button>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2">
          {projects.map(project => (
            <Card
              key={project.name}
              className="cursor-pointer hover:bg-accent/50 transition-colors relative group"
              onClick={() => onProjectClick(project)}
            >
              <CardHeader className="p-4 pb-2">
                <div className="flex items-center gap-2">
                  <FolderOpen className="h-5 w-5 text-muted-foreground" />
                  <CardTitle className="text-base flex-1">{project.name}</CardTitle>
                  {/* Workspace Status Badge */}
                  {project.workspace_status === 'ready' && (
                    <Badge variant="success" className="text-xs">
                      Ready
                    </Badge>
                  )}
                  {project.workspace_status === 'missing' && (
                    <Badge
                      variant="warning"
                      className="text-xs gap-1"
                      title={project.workspace_error || 'Workspace not configured'}
                    >
                      <AlertTriangle className="h-3 w-3" />
                      Not Set Up
                    </Badge>
                  )}
                  {project.workspace_status === 'error' && (
                    <Badge
                      variant="destructive"
                      className="text-xs gap-1"
                      title={project.workspace_error || 'Workspace error'}
                    >
                      <XCircle className="h-3 w-3" />
                      Error
                    </Badge>
                  )}
                  {/* Actions Menu */}
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8 opacity-0 group-hover:opacity-100 transition-opacity"
                        onClick={e => e.stopPropagation()}
                      >
                        <MoreVertical className="h-4 w-4" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuItem
                        className="text-destructive focus:text-destructive"
                        onClick={e => handleRemoveClick(e, project.name)}
                      >
                        <Trash2 className="h-4 w-4 mr-2" />
                        Remove Project
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
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

      {/* Add Project Dialog */}
      <AddProjectDialog
        open={addDialogOpen}
        onOpenChange={setAddDialogOpen}
        onSuccess={handleAddSuccess}
      />

      {/* Remove Project Dialog */}
      {selectedProject && (
        <RemoveProjectDialog
          open={removeDialogOpen}
          onOpenChange={setRemoveDialogOpen}
          projectName={selectedProject}
          onSuccess={handleRemoveSuccess}
        />
      )}
    </div>
  )
}
