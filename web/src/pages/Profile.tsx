import { useEffect, useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { User, Home, FolderOpen, CheckCircle2, XCircle, Terminal } from 'lucide-react'
import { useUser } from '@/hooks/useUser'
import type { CredentialValidation } from '@/hooks/useUser'
import type { Project } from '@/lib/types'

interface ProjectCredentials {
  project: Project
  validation: CredentialValidation | null
  loading: boolean
}

export function Profile() {
  const { user, loading: userLoading, fetchCredentials } = useUser()
  const [projects, setProjects] = useState<Project[]>([])
  const [credentials, setCredentials] = useState<Map<string, ProjectCredentials>>(new Map())
  const [projectsLoading, setProjectsLoading] = useState(true)

  // Fetch projects
  useEffect(() => {
    fetch('/api/projects')
      .then(res => res.json())
      .then((data: Project[]) => {
        setProjects(data)
        // Initialize credentials map
        const credMap = new Map<string, ProjectCredentials>()
        data.forEach(p => {
          credMap.set(p.name, { project: p, validation: null, loading: true })
        })
        setCredentials(credMap)
        setProjectsLoading(false)
      })
      .catch(() => {
        setProjects([])
        setProjectsLoading(false)
      })
  }, [])

  // Fetch credentials for each project
  useEffect(() => {
    if (projects.length === 0) return

    projects.forEach(async (project) => {
      const validation = await fetchCredentials(project.name)
      setCredentials(prev => {
        const newMap = new Map(prev)
        newMap.set(project.name, { project, validation, loading: false })
        return newMap
      })
    })
  }, [projects, fetchCredentials])

  if (userLoading) {
    return (
      <div className="p-4 space-y-6">
        <h1 className="text-2xl font-bold">Profile</h1>
        <Skeleton className="h-32 w-full" />
        <Skeleton className="h-48 w-full" />
      </div>
    )
  }

  return (
    <div className="p-4 space-y-6">
      <h1 className="text-2xl font-bold">Profile</h1>

      {/* User Info Card */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <User className="h-5 w-5" />
            User Information
          </CardTitle>
          <CardDescription>
            Current system user running vega-hub
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          {user ? (
            <>
              <div className="flex items-center gap-3">
                <span className="text-muted-foreground min-w-24">Username:</span>
                <span className="font-medium">{user.username}</span>
              </div>
              <div className="flex items-center gap-3">
                <span className="text-muted-foreground min-w-24">
                  <Home className="h-4 w-4 inline mr-1" />
                  Home:
                </span>
                <code className="text-sm bg-muted px-2 py-0.5 rounded">{user.home_dir}</code>
              </div>
              <div className="flex items-center gap-3">
                <span className="text-muted-foreground min-w-24">UID:</span>
                <span className="font-mono text-sm">{user.uid}</span>
              </div>
            </>
          ) : (
            <p className="text-muted-foreground">Unable to fetch user information</p>
          )}
        </CardContent>
      </Card>

      {/* Project Credentials */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <FolderOpen className="h-5 w-5" />
            Project Credentials
          </CardTitle>
          <CardDescription>
            Git authentication status for each project
          </CardDescription>
        </CardHeader>
        <CardContent>
          {projectsLoading ? (
            <div className="space-y-3">
              {[1, 2, 3].map(i => (
                <Skeleton key={i} className="h-16" />
              ))}
            </div>
          ) : projects.length === 0 ? (
            <p className="text-muted-foreground text-center py-4">No projects configured</p>
          ) : (
            <div className="space-y-4">
              {Array.from(credentials.values()).map(({ project, validation, loading }) => (
                <div
                  key={project.name}
                  className="border rounded-lg p-4 space-y-2"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <FolderOpen className="h-4 w-4 text-muted-foreground" />
                      <span className="font-medium">{project.name}</span>
                    </div>
                    {loading ? (
                      <Skeleton className="h-5 w-16" />
                    ) : validation ? (
                      <Badge variant={validation.valid ? 'success' : 'destructive'}>
                        {validation.valid ? (
                          <>
                            <CheckCircle2 className="h-3 w-3 mr-1" />
                            Valid
                          </>
                        ) : (
                          <>
                            <XCircle className="h-3 w-3 mr-1" />
                            Invalid
                          </>
                        )}
                      </Badge>
                    ) : (
                      <Badge variant="secondary">Unknown</Badge>
                    )}
                  </div>

                  {validation && (
                    <>
                      <div className="text-sm text-muted-foreground">
                        Service: {validation.service.name} ({validation.service.host})
                      </div>

                      {/* Credential Sources */}
                      {validation.statuses.length > 0 && (
                        <div className="text-sm space-y-1">
                          {validation.statuses.map((status, i) => (
                            <div key={i} className="flex items-center gap-2">
                              {status.valid ? (
                                <CheckCircle2 className="h-3 w-3 text-green-500" />
                              ) : (
                                <XCircle className="h-3 w-3 text-muted-foreground" />
                              )}
                              <span className="text-muted-foreground">{status.source}:</span>
                              {status.valid ? (
                                <span className="text-green-600">{status.user || 'authenticated'}</span>
                              ) : (
                                <span className="text-muted-foreground">{status.error || 'not configured'}</span>
                              )}
                            </div>
                          ))}
                        </div>
                      )}

                      {/* Fix Options */}
                      {!validation.valid && validation.fix_options.length > 0 && (
                        <div className="mt-3 p-3 bg-muted rounded-md space-y-2">
                          <div className="text-sm font-medium flex items-center gap-1">
                            <Terminal className="h-4 w-4" />
                            To fix:
                          </div>
                          {validation.fix_options.map((fix, i) => (
                            <div key={i} className="text-sm">
                              <code className="bg-background px-2 py-0.5 rounded text-xs">
                                {fix.command}
                              </code>
                              <span className="text-muted-foreground ml-2 text-xs">
                                {fix.description}
                              </span>
                            </div>
                          ))}
                        </div>
                      )}
                    </>
                  )}
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
