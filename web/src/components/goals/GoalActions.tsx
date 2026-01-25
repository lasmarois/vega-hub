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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { AlertTriangle, CheckCircle2, Pause, Trash2, Plus } from 'lucide-react'
import type { GoalDetail, Project } from '@/lib/types'

// Complete Goal Dialog
interface CompleteGoalDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  goal: GoalDetail
  onSuccess: () => void
}

export function CompleteGoalDialog({
  open,
  onOpenChange,
  goal,
  onSuccess,
}: CompleteGoalDialogProps) {
  const [noMerge, setNoMerge] = useState(false)
  const [force, setForce] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const project = goal.projects[0] || ''

  const handleComplete = async () => {
    if (!project) {
      setError('No project associated with this goal')
      return
    }

    setLoading(true)
    setError(null)

    try {
      const res = await fetch(`/api/goals/${goal.id}/complete`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ project, no_merge: noMerge, force }),
      })

      const data = await res.json()
      if (data.success) {
        onSuccess()
        onOpenChange(false)
      } else {
        setError(data.error?.message || 'Failed to complete goal')
      }
    } catch (err) {
      setError('Network error')
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <CheckCircle2 className="h-5 w-5 text-green-500" />
            Complete Goal #{goal.id}
          </DialogTitle>
          <DialogDescription>
            This will merge the goal branch and clean up the worktree.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="text-sm">
            <strong>Title:</strong> {goal.title}
          </div>
          <div className="text-sm">
            <strong>Project:</strong> {project}
          </div>

          <div className="space-y-3">
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={noMerge}
                onChange={(e) => setNoMerge(e.target.checked)}
                className="rounded border-gray-300"
              />
              Skip merge (keep branch for MR/PR)
            </label>
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={force}
                onChange={(e) => setForce(e.target.checked)}
                className="rounded border-gray-300"
              />
              Force (ignore uncommitted changes)
            </label>
          </div>

          {error && (
            <div className="text-sm text-red-500 flex items-center gap-2">
              <AlertTriangle className="h-4 w-4" />
              {error}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleComplete} disabled={loading}>
            {loading ? 'Completing...' : 'Complete Goal'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// Ice Goal Dialog
interface IceGoalDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  goal: GoalDetail
  onSuccess: () => void
}

export function IceGoalDialog({
  open,
  onOpenChange,
  goal,
  onSuccess,
}: IceGoalDialogProps) {
  const [reason, setReason] = useState('')
  const [force, setForce] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const project = goal.projects[0] || ''

  const handleIce = async () => {
    if (!project) {
      setError('No project associated with this goal')
      return
    }
    if (!reason.trim()) {
      setError('Reason is required')
      return
    }

    setLoading(true)
    setError(null)

    try {
      const res = await fetch(`/api/goals/${goal.id}/ice`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ project, reason: reason.trim(), force }),
      })

      const data = await res.json()
      if (data.success) {
        onSuccess()
        onOpenChange(false)
        setReason('')
      } else {
        setError(data.error?.message || 'Failed to ice goal')
      }
    } catch (err) {
      setError('Network error')
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Pause className="h-5 w-5 text-blue-500" />
            Ice Goal #{goal.id}
          </DialogTitle>
          <DialogDescription>
            Pause this goal for later. The branch will be preserved.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="text-sm">
            <strong>Title:</strong> {goal.title}
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">Reason (required)</label>
            <Input
              placeholder="Why is this goal being paused?"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
            />
          </div>

          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={force}
              onChange={(e) => setForce(e.target.checked)}
              className="rounded border-gray-300"
            />
            Force (ignore uncommitted changes)
          </label>

          {error && (
            <div className="text-sm text-red-500 flex items-center gap-2">
              <AlertTriangle className="h-4 w-4" />
              {error}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleIce} disabled={loading || !reason.trim()}>
            {loading ? 'Icing...' : 'Ice Goal'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// Create Goal Dialog
interface CreateGoalDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  projects: Project[]
  onSuccess: () => void
}

export function CreateGoalDialog({
  open,
  onOpenChange,
  projects,
  onSuccess,
}: CreateGoalDialogProps) {
  const [title, setTitle] = useState('')
  const [project, setProject] = useState('')
  const [baseBranch, setBaseBranch] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Update base branch when project changes
  const handleProjectChange = (value: string) => {
    setProject(value)
    const p = projects.find((p) => p.name === value)
    if (p) {
      setBaseBranch(p.base_branch || '')
    }
  }

  const handleCreate = async () => {
    if (!title.trim()) {
      setError('Title is required')
      return
    }
    if (!project) {
      setError('Project is required')
      return
    }

    setLoading(true)
    setError(null)

    try {
      const res = await fetch('/api/goals', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          title: title.trim(),
          project,
          base_branch: baseBranch || undefined,
        }),
      })

      const data = await res.json()
      if (data.success) {
        onSuccess()
        onOpenChange(false)
        setTitle('')
        setProject('')
        setBaseBranch('')
      } else {
        setError(data.error?.message || 'Failed to create goal')
      }
    } catch (err) {
      setError('Network error')
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Plus className="h-5 w-5 text-green-500" />
            Create New Goal
          </DialogTitle>
          <DialogDescription>
            Create a new goal with a worktree for development.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <label className="text-sm font-medium">Title</label>
            <Input
              placeholder="What do you want to accomplish?"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
            />
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">Project</label>
            <Select value={project} onValueChange={handleProjectChange}>
              <SelectTrigger>
                <SelectValue placeholder="Select a project" />
              </SelectTrigger>
              <SelectContent>
                {projects.map((p) => (
                  <SelectItem key={p.name} value={p.name}>
                    {p.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">Base Branch (optional)</label>
            <Input
              placeholder={baseBranch || 'main'}
              value={baseBranch}
              onChange={(e) => setBaseBranch(e.target.value)}
            />
            <p className="text-xs text-muted-foreground">
              Leave empty to use project's default branch
            </p>
          </div>

          {error && (
            <div className="text-sm text-red-500 flex items-center gap-2">
              <AlertTriangle className="h-4 w-4" />
              {error}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleCreate} disabled={loading || !title.trim() || !project}>
            {loading ? 'Creating...' : 'Create Goal'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// Cleanup Branch Dialog
interface CleanupGoalDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  goalId: string
  project: string
  onSuccess: () => void
}

export function CleanupGoalDialog({
  open,
  onOpenChange,
  goalId,
  project,
  onSuccess,
}: CleanupGoalDialogProps) {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleCleanup = async () => {
    setLoading(true)
    setError(null)

    try {
      const res = await fetch(`/api/goals/${goalId}/cleanup`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ project }),
      })

      const data = await res.json()
      if (data.success) {
        onSuccess()
        onOpenChange(false)
      } else {
        setError(data.error?.message || 'Failed to cleanup branch')
      }
    } catch (err) {
      setError('Network error')
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-red-500">
            <Trash2 className="h-5 w-5" />
            Cleanup Branch
          </DialogTitle>
          <DialogDescription>
            This will permanently delete the goal branch. This action cannot be undone.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="text-sm">
            <strong>Goal ID:</strong> {goalId}
          </div>
          <div className="text-sm">
            <strong>Project:</strong> {project}
          </div>

          <div className="p-3 bg-red-50 border border-red-200 rounded-md text-sm text-red-700">
            <strong>Warning:</strong> This will delete the git branch permanently.
            Only use this after the MR/PR has been merged.
          </div>

          {error && (
            <div className="text-sm text-red-500 flex items-center gap-2">
              <AlertTriangle className="h-4 w-4" />
              {error}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button variant="destructive" onClick={handleCleanup} disabled={loading}>
            {loading ? 'Deleting...' : 'Delete Branch'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// Stop Executor Dialog
interface StopExecutorDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  goalId: string
  sessionId: string
  onSuccess: () => void
}

export function StopExecutorDialog({
  open,
  onOpenChange,
  goalId,
  sessionId,
  onSuccess,
}: StopExecutorDialogProps) {
  const [reason, setReason] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleStop = async () => {
    setLoading(true)
    setError(null)

    try {
      const res = await fetch('/api/executor/stop', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          goal_id: goalId,
          session_id: sessionId,
          reason: reason.trim() || 'Stopped by user',
        }),
      })

      const data = await res.json()
      if (data.ok) {
        onSuccess()
        onOpenChange(false)
        setReason('')
      } else {
        setError('Failed to stop executor')
      }
    } catch (err) {
      setError('Network error')
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-orange-500">
            <AlertTriangle className="h-5 w-5" />
            Stop Executor
          </DialogTitle>
          <DialogDescription>
            This will stop the running executor for goal #{goalId}.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <label className="text-sm font-medium">Reason (optional)</label>
            <Input
              placeholder="Why are you stopping the executor?"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
            />
          </div>

          {error && (
            <div className="text-sm text-red-500 flex items-center gap-2">
              <AlertTriangle className="h-4 w-4" />
              {error}
            </div>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button variant="destructive" onClick={handleStop} disabled={loading}>
            {loading ? 'Stopping...' : 'Stop Executor'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
