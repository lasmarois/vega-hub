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
import { Loader2, FolderPlus } from 'lucide-react'

interface AddProjectDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onSuccess?: () => void
}

export function AddProjectDialog({ open, onOpenChange, onSuccess }: AddProjectDialogProps) {
  const [name, setName] = useState('')
  const [path, setPath] = useState('')
  const [baseBranch, setBaseBranch] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError(null)

    try {
      const response = await fetch('/api/projects', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: name.trim(),
          path: path.trim(),
          base_branch: baseBranch.trim() || undefined,
        }),
      })

      const data = await response.json()

      if (!data.success) {
        setError(data.error || 'Failed to add project')
        return
      }

      // Success
      setName('')
      setPath('')
      setBaseBranch('')
      onOpenChange(false)
      onSuccess?.()
    } catch (err) {
      setError('Network error. Please try again.')
    } finally {
      setLoading(false)
    }
  }

  const handleClose = () => {
    setName('')
    setPath('')
    setBaseBranch('')
    setError(null)
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <FolderPlus className="h-5 w-5" />
            Add Project
          </DialogTitle>
          <DialogDescription>
            Add an existing local repository to vega-missile management.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit}>
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

            <div className="grid gap-2">
              <Label htmlFor="path">Local Path</Label>
              <Input
                id="path"
                value={path}
                onChange={e => setPath(e.target.value)}
                placeholder="/path/to/repository"
                required
              />
              <p className="text-xs text-muted-foreground">
                Absolute path to the git repository on your machine.
              </p>
            </div>

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

          <DialogFooter>
            <Button type="button" variant="outline" onClick={handleClose} disabled={loading}>
              Cancel
            </Button>
            <Button type="submit" disabled={loading || !name.trim() || !path.trim()}>
              {loading ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Adding...
                </>
              ) : (
                'Add Project'
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
