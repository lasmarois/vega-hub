import { useState } from 'react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Loader2, FolderPlus, GitBranch, Folder } from 'lucide-react'

interface AddProjectDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess?: () => void
}

export function AddProjectDialog({ open, onOpenChange, onSuccess }: AddProjectDialogProps) {
  const [mode, setMode] = useState<'clone' | 'local'>('clone')
  const [name, setName] = useState('')
  const [url, setUrl] = useState('')
  const [path, setPath] = useState('')
  const [baseBranch, setBaseBranch] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError(null)

    try {
      const body: Record<string, string> = {
        name: name.trim(),
      }

      if (mode === 'clone') {
        body.url = url.trim()
      } else {
        body.path = path.trim()
      }

      if (baseBranch.trim()) {
        body.base_branch = baseBranch.trim()
      }

      const response = await fetch('/api/projects', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })

      const data = await response.json()

      if (!data.success) {
        setError(data.error || 'Failed to add project')
        return
      }

      // Success - reset and close
      resetForm()
      onOpenChange(false)
      onSuccess?.()
    } catch (err) {
      setError('Network error. Please try again.')
    } finally {
      setLoading(false)
    }
  }

  const resetForm = () => {
    setName('')
    setUrl('')
    setPath('')
    setBaseBranch('')
    setError(null)
  }

  const handleClose = () => {
    resetForm()
    onOpenChange(false)
  }

  const isValid = name.trim() && (mode === 'clone' ? url.trim() : path.trim())

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <FolderPlus className="h-5 w-5" />
            Add Project
          </DialogTitle>
          <DialogDescription>
            Clone a repository or link an existing local folder.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit}>
          <Tabs value={mode} onValueChange={(v) => setMode(v as 'clone' | 'local')} className="w-full">
            <TabsList className="grid w-full grid-cols-2">
              <TabsTrigger value="clone" className="flex items-center gap-2">
                <GitBranch className="h-4 w-4" />
                Clone URL
              </TabsTrigger>
              <TabsTrigger value="local" className="flex items-center gap-2">
                <Folder className="h-4 w-4" />
                Local Path
              </TabsTrigger>
            </TabsList>

            <div className="grid gap-4 py-4">
              <div className="grid gap-2">
                <Label htmlFor="name">Project Name</Label>
                <Input
                  id="name"
                  value={name}
                  onChange={e => setName(e.target.value)}
                  placeholder="my-project"
                  required
                  pattern="[a-zA-Z0-9_-]+"
                  title="Alphanumeric, dashes, and underscores only"
                />
                <p className="text-xs text-muted-foreground">
                  Used for identification. Alphanumeric, dashes, and underscores only.
                </p>
              </div>

              <TabsContent value="clone" className="mt-0 grid gap-2">
                <Label htmlFor="url">Repository URL</Label>
                <Input
                  id="url"
                  value={url}
                  onChange={e => setUrl(e.target.value)}
                  placeholder="https://github.com/user/repo.git"
                  required={mode === 'clone'}
                />
                <p className="text-xs text-muted-foreground">
                  HTTPS or SSH URL. The repository will be cloned to workspaces/{name || 'project'}/worktree-base/
                </p>
              </TabsContent>

              <TabsContent value="local" className="mt-0 grid gap-2">
                <Label htmlFor="path">Local Path</Label>
                <Input
                  id="path"
                  value={path}
                  onChange={e => setPath(e.target.value)}
                  placeholder="/path/to/repository"
                  required={mode === 'local'}
                />
                <p className="text-xs text-muted-foreground">
                  Absolute path to an existing git repository on your machine.
                </p>
              </TabsContent>

              <div className="grid gap-2">
                <Label htmlFor="baseBranch">Base Branch (optional)</Label>
                <Input
                  id="baseBranch"
                  value={baseBranch}
                  onChange={e => setBaseBranch(e.target.value)}
                  placeholder="main"
                />
                <p className="text-xs text-muted-foreground">
                  Default branch for the project. Auto-detected if not specified.
                </p>
              </div>

              {error && (
                <div className="rounded-md bg-destructive/10 border border-destructive/20 p-3">
                  <p className="text-sm text-destructive">{error}</p>
                </div>
              )}
            </div>
          </Tabs>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleClose} disabled={loading}>
              Cancel
            </Button>
            <Button type="submit" disabled={loading || !isValid}>
              {loading ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  {mode === 'clone' ? 'Cloning...' : 'Adding...'}
                </>
              ) : (
                mode === 'clone' ? 'Clone & Add' : 'Add Project'
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
