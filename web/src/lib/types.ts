export interface Question {
  id: string
  goal_id: string
  session_id: string
  question: string
  options?: { label: string; description?: string }[]
  created_at: string
}

export interface GoalSummary {
  id: string
  title: string
  projects: string[]
  status: 'active' | 'iced' | 'completed'
  phase: string
  executor_status: 'running' | 'waiting' | 'stopped' | 'idle'
  pending_questions: number
  active_executors: number
}

export interface PhaseDetail {
  number: number
  title: string
  tasks: { description: string; completed: boolean }[]
  status: 'pending' | 'in_progress' | 'complete'
}

export interface Executor {
  session_id: string
  goal_id: string
  cwd: string
  started_at: string
}

export interface GoalDetail {
  id: string
  title: string
  projects: string[]
  status: 'active' | 'iced' | 'completed'
  phase: string
  overview: string
  phases: PhaseDetail[]
  acceptance: string[]
  notes: string[]
  executor_status: 'running' | 'waiting' | 'stopped' | 'idle'
  pending_questions: Question[]
  active_executors: Executor[]
}

export interface GoalStatus {
  current_phase: string
  recent_actions: string[]
  progress_log: string
  task_plan: string
  findings: string
  has_worktree: boolean
  worktree_path: string
  phase_progress: {
    number: number
    title: string
    status: string
    tasks_total: number
    tasks_done: number
  }[]
}

export interface SSEEvent {
  type: string
  data?: Record<string, unknown>
}

export interface Activity {
  id: string
  type: 'executor_started' | 'executor_stopped' | 'question' | 'answered' | 'goal_updated' | 'goal_iced' | 'goal_completed'
  goal_id?: string
  session_id?: string
  message: string
  timestamp: string
}

export interface Project {
  name: string
  base_branch: string
  workspace?: string
  upstream?: string
}
