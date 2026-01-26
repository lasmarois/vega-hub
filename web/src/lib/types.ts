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
  workspace_status?: 'ready' | 'missing' | 'error'
  workspace_error?: string
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

export interface BranchInfo {
  branch: string
  base_branch: string
  ahead: number
  behind: number
  uncommitted_files: number
  last_commit?: string
  last_commit_message?: string
  worktree_path?: string
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
  workspace_status?: 'ready' | 'missing' | 'error'
  workspace_error?: string
  branch_info?: BranchInfo
  worktree_status?: 'exists' | 'missing' | 'never_created'
  branch_status?: 'local' | 'remote_only' | 'missing'
  can_recreate?: boolean
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
  workspace_status: 'ready' | 'missing' | 'error'
  workspace_error?: string
}

// UserMessage represents a message from a user to an executor
// Used for bidirectional communication (user → executor)
export interface UserMessage {
  id: string
  goal_id: string
  content: string
  user?: string
  created_at: string
}

// ChatMessage represents a message in the chat thread
// Returned by GET /api/goals/:id/chat
export interface ChatMessage {
  id: string
  type: 'session_start' | 'session_stop' | 'question' | 'answer' | 'user_message' | 'user_message_delivered' | 'activity'
  timestamp: string
  session_id: string
  goal_id: string
  content?: string           // question/answer/user_message text
  answer?: string            // for question messages with answer
  activity_type?: string     // for activity messages
  data?: Record<string, unknown>  // activity details (expandable)
  pending?: boolean          // true for unanswered questions
  options?: { label: string; description?: string }[]
  user?: string              // who sent (executor user, answering user)
  stop_reason?: string       // for session_stop
}
