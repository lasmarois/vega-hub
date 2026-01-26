import { useState, useEffect } from 'react'
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
import { AlertTriangle, CheckCircle2, Pause, Trash2, Plus, Play, RefreshCw, GitBranch, XCircle, Loader2, AlertCircle } from 'lucide-react'
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
  const [removeWorktree, setRemoveWorktree] = useState(false)
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
        body: JSON.stringify({
          project,
          reason: reason.trim(),
          remove_worktree: removeWorktree,
          force: removeWorktree && force,
        }),
      })

      const data = await res.json()
      if (data.success) {
        onSuccess()
        onOpenChange(false)
        setReason('')
        setRemoveWorktree(false)
        setForce(false)
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
            Pause this goal for later. Branch and worktree will be preserved for easy resume.
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

          <div className="space-y-2">
            <label className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={removeWorktree}
                onChange={(e) => {
                  setRemoveWorktree(e.target.checked)
                  if (!e.target.checked) setForce(false)
                }}
                className="rounded border-gray-300"
              />
              Remove worktree (frees disk space)
            </label>

            {removeWorktree && (
              <label className="flex items-center gap-2 text-sm ml-6 text-muted-foreground">
                <input
                  type="checkbox"
                  checked={force}
                  onChange={(e) => setForce(e.target.checked)}
                  className="rounded border-gray-300"
                />
                Force (ignore uncommitted changes)
              </label>
            )}
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

// Resume Goal Dialog
interface ResumeGoalDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  goal: GoalDetail
  onSuccess: () => void
}

export function ResumeGoalDialog({
  open,
  onOpenChange,
  goal,
  onSuccess,
}: ResumeGoalDialogProps) {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const project = goal.projects[0] || ''

  const handleResume = async () => {
    if (!project) {
      setError('No project associated with this goal')
      return
    }

    setLoading(true)
    setError(null)

    try {
      const res = await fetch(`/api/goals/${goal.id}/resume`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ project }),
      })

      const data = await res.json()
      if (data.success) {
        onSuccess()
        onOpenChange(false)
      } else {
        setError(data.error?.message || 'Failed to resume goal')
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
            <Play className="h-5 w-5 text-green-500" />
            Resume Goal #{goal.id}
          </DialogTitle>
          <DialogDescription>
            Resume this iced goal. The worktree will be recreated if needed.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="text-sm">
            <strong>Title:</strong> {goal.title}
          </div>
          <div className="text-sm">
            <strong>Project:</strong> {project}
          </div>

          <div className="p-3 bg-blue-50 border border-blue-200 rounded-md text-sm text-blue-700">
            This will move the goal back to active status and restore the worktree if it was removed during ice.
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
          <Button onClick={handleResume} disabled={loading}>
            {loading ? 'Resuming...' : 'Resume Goal'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// Create MR/PR Dialog
interface CreateMRDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  goalId: string
  goalTitle: string
  baseBranch: string
  lastCommitMessage?: string
  onSuccess: () => void
}

export function CreateMRDialog({
  open,
  onOpenChange,
  goalId,
  goalTitle,
  baseBranch,
  lastCommitMessage,
  onSuccess,
}: CreateMRDialogProps) {
  // Pre-fill title with goal title or last commit message
  const defaultTitle = lastCommitMessage || goalTitle
  const [title, setTitle] = useState(defaultTitle)
  const [description, setDescription] = useState('')
  const [targetBranch, setTargetBranch] = useState(baseBranch)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [mrUrl, setMrUrl] = useState<string | null>(null)
  const [mrService, setMrService] = useState<string | null>(null)

  // Reset form when dialog opens
  const handleOpenChange = (newOpen: boolean) => {
    if (newOpen) {
      setTitle(defaultTitle)
      setDescription('')
      setTargetBranch(baseBranch)
      setError(null)
      setMrUrl(null)
      setMrService(null)
    }
    onOpenChange(newOpen)
  }

  const handleCreateMR = async () => {
    if (!title.trim()) {
      setError('Title is required')
      return
    }

    setLoading(true)
    setError(null)

    try {
      const res = await fetch(`/api/goals/${goalId}/create-mr`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          title: title.trim(),
          description: description.trim() || undefined,
          target_branch: targetBranch || undefined,
        }),
      })

      const data = await res.json()
      if (data.success) {
        setMrUrl(data.mr_url)
        setMrService(data.service)
        onSuccess()
      } else {
        setError(data.error || 'Failed to create MR/PR')
      }
    } catch (err) {
      setError('Network error')
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Plus className="h-5 w-5 text-purple-500" />
            Create Merge Request
          </DialogTitle>
          <DialogDescription>
            Create a merge/pull request for goal #{goalId}
          </DialogDescription>
        </DialogHeader>

        {mrUrl ? (
          // Success state - show MR link
          <div className="space-y-4 py-4">
            <div className="p-4 bg-green-50 border border-green-200 rounded-md">
              <div className="flex items-center gap-2 text-green-700 font-medium mb-2">
                <CheckCircle2 className="h-5 w-5" />
                {mrService === 'github' ? 'Pull Request' : 'Merge Request'} Created!
              </div>
              <a
                href={mrUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="text-sm text-blue-600 hover:underline break-all"
              >
                {mrUrl}
              </a>
            </div>
          </div>
        ) : (
          // Form state
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">Title</label>
              <Input
                placeholder="MR/PR title"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
              />
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">Description (optional)</label>
              <textarea
                placeholder="Describe the changes..."
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                className="flex min-h-[100px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
              />
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">Target Branch</label>
              <Input
                placeholder="main"
                value={targetBranch}
                onChange={(e) => setTargetBranch(e.target.value)}
              />
              <p className="text-xs text-muted-foreground">
                The branch to merge into (defaults to base branch)
              </p>
            </div>

            {error && (
              <div className="text-sm text-red-500 flex items-center gap-2">
                <AlertTriangle className="h-4 w-4" />
                {error}
              </div>
            )}
          </div>
        )}

        <DialogFooter>
          {mrUrl ? (
            <Button onClick={() => handleOpenChange(false)}>
              Done
            </Button>
          ) : (
            <>
              <Button variant="outline" onClick={() => handleOpenChange(false)}>
                Cancel
              </Button>
              <Button onClick={handleCreateMR} disabled={loading || !title.trim()}>
                {loading ? 'Creating...' : 'Create MR/PR'}
              </Button>
            </>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// Recreate Worktree Dialog
interface RecreateWorktreeDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  goal: GoalDetail
  onSuccess: () => void
}

export function RecreateWorktreeDialog({
  open,
  onOpenChange,
  goal,
  onSuccess,
}: RecreateWorktreeDialogProps) {
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const project = goal.projects[0] || ''
  const branchName = goal.branch_info?.branch || 'unknown'
  const canRecreate = goal.can_recreate
  const branchStatus = goal.branch_status

  const handleRecreate = async () => {
    setLoading(true)
    setError(null)

    try {
      const res = await fetch(`/api/goals/${goal.id}/recreate-worktree`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ project }),
      })

      const data = await res.json()
      if (data.success) {
        onSuccess()
        onOpenChange(false)
      } else {
        setError(data.error || 'Failed to recreate worktree')
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
            {canRecreate ? (
              <>
                <RefreshCw className="h-5 w-5 text-yellow-500" />
                Recreate Worktree
              </>
            ) : (
              <>
                <XCircle className="h-5 w-5 text-red-500" />
                Branch Not Found
              </>
            )}
          </DialogTitle>
          <DialogDescription>
            {canRecreate
              ? `Recreate the worktree for goal #${goal.id}`
              : `The branch for goal #${goal.id} no longer exists`}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="text-sm">
            <strong>Goal:</strong> {goal.title}
          </div>
          <div className="text-sm">
            <strong>Project:</strong> {project}
          </div>

          {canRecreate ? (
            <>
              <div className="text-sm flex items-center gap-2">
                <GitBranch className="h-4 w-4" />
                <strong>Branch:</strong>
                <code className="text-xs bg-muted px-1.5 py-0.5 rounded">{branchName}</code>
                <span className={`text-xs ${branchStatus === 'local' ? 'text-green-600' : 'text-yellow-600'}`}>
                  ({branchStatus === 'local' ? 'local' : 'remote only'})
                </span>
              </div>

              <div className="p-3 bg-yellow-50 border border-yellow-200 rounded-md text-sm text-yellow-700">
                <strong>Warning:</strong> If there were uncommitted changes in the previous
                worktree, they are permanently lost.
              </div>
            </>
          ) : (
            <>
              <div className="p-3 bg-red-50 border border-red-200 rounded-md text-sm text-red-700">
                <strong>Branch not found:</strong> The branch{' '}
                <code className="bg-red-100 px-1 rounded">{branchName}</code> no longer exists
                locally or on the remote.
              </div>

              <div className="text-sm text-muted-foreground">
                <strong>Options:</strong>
                <ul className="list-disc ml-5 mt-1 space-y-1">
                  <li>Create a fresh worktree using the Resume action</li>
                  <li>Keep the goal without a worktree</li>
                </ul>
              </div>
            </>
          )}

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
          {canRecreate && (
            <Button onClick={handleRecreate} disabled={loading}>
              {loading ? 'Recreating...' : 'Recreate Worktree'}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

// Delete Goal Dialog
interface DeleteGoalDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  goal: GoalDetail
  onSuccess: () => void
}

interface PreflightWarning {
  level: 'error' | 'warning' | 'info'
  message: string
  details?: string[]
}

interface PreflightResult {
  success: boolean
  require_force?: boolean
  warnings?: PreflightWarning[]
  error?: string
}

type DialogState = 'loading' | 'warnings' | 'confirm' | 'deleting'

export function DeleteGoalDialog({
  open,
  onOpenChange,
  goal,
  onSuccess,
}: DeleteGoalDialogProps) {
  const [state, setState] = useState<DialogState>('loading')
  const [deleteBranch, setDeleteBranch] = useState(true)
  const [force, setForce] = useState(false)
  const [requireForce, setRequireForce] = useState(false)
  const [warnings, setWarnings] = useState<PreflightWarning[]>([])
  const [error, setError] = useState<string | null>(null)

  // Run preflight check when dialog opens
  const runPreflightCheck = async () => {
    setState('loading')
    setError(null)
    setWarnings([])
    setRequireForce(false)
    setForce(false)

    try {
      const res = await fetch(`/api/goals/${goal.id}/delete`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ force: false, delete_branch: deleteBranch }),
      })

      const data: PreflightResult = await res.json()

      if (data.success) {
        // No issues - go straight to confirm
        setState('confirm')
      } else if (data.require_force) {
        // Has warnings that require force
        setWarnings(data.warnings || [])
        setRequireForce(true)
        setState('warnings')
      } else {
        // Error that can't be forced
        setError(data.error || 'Cannot delete this goal')
        setState('warnings')
      }
    } catch (err) {
      setError('Network error')
      setState('warnings')
    }
  }

  // Run preflight check when dialog opens
  useEffect(() => {
    if (open) {
      runPreflightCheck()
    }
  }, [open])

  // Handle dialog open/close
  const handleOpenChange = (newOpen: boolean) => {
    if (!newOpen) {
      // Reset state when closing
      setState('loading')
      setDeleteBranch(true)
      setForce(false)
      setRequireForce(false)
      setWarnings([])
      setError(null)
    }
    onOpenChange(newOpen)
  }

  // Re-run preflight when deleteBranch changes (only in warnings/confirm state)
  const handleDeleteBranchChange = async (checked: boolean) => {
    setDeleteBranch(checked)
    // Re-run preflight with new setting
    setState('loading')
    setError(null)
    setWarnings([])
    setRequireForce(false)
    setForce(false)

    try {
      const res = await fetch(`/api/goals/${goal.id}/delete`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ force: false, delete_branch: checked }),
      })

      const data: PreflightResult = await res.json()

      if (data.success) {
        setState('confirm')
      } else if (data.require_force) {
        setWarnings(data.warnings || [])
        setRequireForce(true)
        setState('warnings')
      } else {
        setError(data.error || 'Cannot delete this goal')
        setState('warnings')
      }
    } catch (err) {
      setError('Network error')
      setState('warnings')
    }
  }

  // Execute the actual deletion
  const handleDelete = async () => {
    setState('deleting')
    setError(null)

    try {
      const res = await fetch(`/api/goals/${goal.id}/delete`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ force: true, delete_branch: deleteBranch }),
      })

      const data = await res.json()

      if (data.success) {
        onSuccess()
        handleOpenChange(false)
      } else {
        setError(data.error || 'Failed to delete goal')
        setState('warnings')
      }
    } catch (err) {
      setError('Network error')
      setState('warnings')
    }
  }

  const canDelete = state === 'confirm' || (state === 'warnings' && (!requireForce || force))

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-red-600">
            <Trash2 className="h-5 w-5" />
            Delete Goal
          </DialogTitle>
          <DialogDescription>
            Delete goal #{goal.id}?
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          {/* Loading state */}
          {state === 'loading' && (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
              <span className="ml-2 text-muted-foreground">Checking for issues...</span>
            </div>
          )}

          {/* Deleting state */}
          {state === 'deleting' && (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-6 w-6 animate-spin text-red-500" />
              <span className="ml-2 text-muted-foreground">Deleting goal...</span>
            </div>
          )}

          {/* Warnings/Confirm state */}
          {(state === 'warnings' || state === 'confirm') && (
            <>
              <div className="text-sm">
                <strong>Title:</strong> {goal.title}
              </div>

              {/* Display warnings */}
              {warnings.length > 0 && (
                <div className="space-y-3">
                  {warnings.map((warning, i) => (
                    <div
                      key={i}
                      className={`p-3 rounded-md text-sm ${
                        warning.level === 'error'
                          ? 'bg-red-50 border border-red-200 text-red-700'
                          : warning.level === 'warning'
                          ? 'bg-yellow-50 border border-yellow-200 text-yellow-700'
                          : 'bg-blue-50 border border-blue-200 text-blue-700'
                      }`}
                    >
                      <div className="flex items-start gap-2">
                        {warning.level === 'error' ? (
                          <AlertCircle className="h-4 w-4 mt-0.5 shrink-0" />
                        ) : (
                          <AlertTriangle className="h-4 w-4 mt-0.5 shrink-0" />
                        )}
                        <div className="flex-1">
                          <div className="font-medium">{warning.message}</div>
                          {warning.details && warning.details.length > 0 && (
                            <ul className="mt-1 text-xs space-y-0.5">
                              {warning.details.slice(0, 5).map((detail, j) => (
                                <li key={j}>• {detail}</li>
                              ))}
                              {warning.details.length > 5 && (
                                <li className="text-muted-foreground">
                                  ... and {warning.details.length - 5} more
                                </li>
                              )}
                            </ul>
                          )}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}

              {/* Network/API error */}
              {error && !requireForce && (
                <div className="text-sm text-red-500 flex items-center gap-2">
                  <AlertTriangle className="h-4 w-4" />
                  {error}
                </div>
              )}

              {/* Checkboxes */}
              <div className="space-y-3 pt-2">
                <label className="flex items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    checked={deleteBranch}
                    onChange={(e) => handleDeleteBranchChange(e.target.checked)}
                    className="rounded border-gray-300"
                  />
                  Delete branch from git
                </label>

                {requireForce && (
                  <label className="flex items-center gap-2 text-sm text-red-600 font-medium">
                    <input
                      type="checkbox"
                      checked={force}
                      onChange={(e) => setForce(e.target.checked)}
                      className="rounded border-red-300"
                    />
                    Force delete (required due to warnings above)
                  </label>
                )}
              </div>

              {/* Final warning */}
              <div className="p-3 bg-muted rounded-md text-sm text-muted-foreground">
                <strong>This action cannot be undone.</strong>
                {deleteBranch && ' The git branch will be permanently deleted.'}
              </div>

              {/* Error from delete attempt */}
              {error && requireForce && (
                <div className="text-sm text-red-500 flex items-center gap-2">
                  <AlertTriangle className="h-4 w-4" />
                  {error}
                </div>
              )}
            </>
          )}
        </div>

        <DialogFooter>
          {(state === 'warnings' || state === 'confirm') && (
            <>
              <Button variant="outline" onClick={() => handleOpenChange(false)}>
                Cancel
              </Button>
              <Button
                variant="destructive"
                onClick={handleDelete}
                disabled={!canDelete}
              >
                Delete Goal
              </Button>
            </>
          )}
          {(state === 'loading' || state === 'deleting') && (
            <Button variant="outline" onClick={() => handleOpenChange(false)}>
              Cancel
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
