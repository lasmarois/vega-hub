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
import { Loader2, Trash2, AlertTriangle } from 'lucide-react'

interface RemoveProjectDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectName: string
  onSuccess?: () => void
}

interface RemoveResult {
  success: boolean
  error?: string
  active_goals?: string[]
  config_removed?: boolean
  index_updated?: boolean
  workspace_removed?: boolean
  goals_warning?: string
}

export function RemoveProjectDialog({
  open,
  onOpenChange,
  projectName,
  onSuccess,
}: RemoveProjectDialogProps) {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [activeGoals, setActiveGoals] = useState<string[]>([])
  const [needsForce, setNeedsForce] = useState(false)

  const handleRemove = async (force = false) => {
    setLoading(true)
    setError(null)

    try {
      const url = `/api/projects/${encodeURIComponent(projectName)}${force ? '?force=true' : ''}`
      const response = await fetch(url, {
        method: 'DELETE',
      })

      const data: RemoveResult = await response.json()

      if (!data.success) {
        if (data.active_goals && data.active_goals.length > 0) {
          setActiveGoals(data.active_goals)
          setNeedsForce(true)
          setError(`Project has ${data.active_goals.length} active/iced goal(s)`)
        } else {
          setError(data.error || 'Failed to remove project')
        }
        return
      }

      // Success
      onOpenChange(false)
      onSuccess?.()
    } catch (err) {
      setError('Network error. Please try again.')
    } finally {
      setLoading(false)
    }
  }

  const handleClose = () => {
    setError(null)
    setActiveGoals([])
    setNeedsForce(false)
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className="sm:max-w-[450px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-destructive">
            <Trash2 className="h-5 w-5" />
            Remove Project
          </DialogTitle>
          <DialogDescription>
            Are you sure you want to remove <strong>{projectName}</strong> from vega-missile?
          </DialogDescription>
        </DialogHeader>

        <div className="py-4">
          <p className="text-sm text-muted-foreground mb-4">This will:</p>
          <ul className="text-sm space-y-1 list-disc list-inside text-muted-foreground">
            <li>Remove the project configuration file</li>
            <li>Remove the project from the index</li>
            <li>Remove the workspace symlink (if applicable)</li>
          </ul>
          <p className="text-sm text-muted-foreground mt-4">
            <strong>Note:</strong> Your actual repository will not be deleted.
          </p>

          {error && (
            <div className="mt-4 rounded-md bg-destructive/10 border border-destructive/20 p-3">
              <div className="flex items-start gap-2">
                <AlertTriangle className="h-4 w-4 text-destructive mt-0.5" />
                <div>
                  <p className="text-sm text-destructive">{error}</p>
                  {activeGoals.length > 0 && (
                    <div className="mt-2">
                      <p className="text-xs text-muted-foreground">Active/iced goals:</p>
                      <ul className="text-xs text-muted-foreground list-disc list-inside">
                        {activeGoals.map(g => (
                          <li key={g}>{g}</li>
                        ))}
                      </ul>
                    </div>
                  )}
                </div>
              </div>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={handleClose} disabled={loading}>
            Cancel
          </Button>
          {needsForce ? (
            <Button
              variant="destructive"
              onClick={() => handleRemove(true)}
              disabled={loading}
            >
              {loading ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Removing...
                </>
              ) : (
                'Force Remove'
              )}
            </Button>
          ) : (
            <Button variant="destructive" onClick={() => handleRemove(false)} disabled={loading}>
              {loading ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Removing...
                </>
              ) : (
                'Remove Project'
              )}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
