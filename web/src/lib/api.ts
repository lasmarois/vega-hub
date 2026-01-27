// API client for vega-hub endpoints
import type { GoalSummary, Dependency, PlanningFile } from './types'

const API_BASE = '/api'

// Dependencies API
export async function getGoalDependencies(goalId: string): Promise<{
  dependencies: Dependency[]
  dependents: Dependency[]
  is_blocked: boolean
  blockers: string[]
}> {
  const res = await fetch(`${API_BASE}/goals/${goalId}/dependencies`)
  if (!res.ok) throw new Error(`Failed to fetch dependencies: ${res.statusText}`)
  return res.json()
}

export async function addDependency(
  goalId: string,
  dependsOnId: string,
  type: 'blocks' | 'related'
): Promise<{ success: boolean }> {
  const res = await fetch(`${API_BASE}/goals/${goalId}/dependencies`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ depends_on: dependsOnId, type }),
  })
  if (!res.ok) throw new Error(`Failed to add dependency: ${res.statusText}`)
  return res.json()
}

export async function removeDependency(
  goalId: string,
  dependsOnId: string
): Promise<{ success: boolean }> {
  const res = await fetch(`${API_BASE}/goals/${goalId}/dependencies/${dependsOnId}`, {
    method: 'DELETE',
  })
  if (!res.ok) throw new Error(`Failed to remove dependency: ${res.statusText}`)
  return res.json()
}

// Ready goals (not blocked)
export async function getReadyGoals(projectFilter?: string): Promise<GoalSummary[]> {
  const params = new URLSearchParams()
  if (projectFilter) params.set('project', projectFilter)
  const url = `${API_BASE}/goals/ready${params.toString() ? `?${params}` : ''}`
  const res = await fetch(url)
  if (!res.ok) throw new Error(`Failed to fetch ready goals: ${res.statusText}`)
  return res.json()
}

// Planning files API
export async function getPlanningFiles(goalId: string): Promise<PlanningFile[]> {
  const res = await fetch(`${API_BASE}/goals/${goalId}/planning-files`)
  if (!res.ok) throw new Error(`Failed to fetch planning files: ${res.statusText}`)
  return res.json()
}

export async function getPlanningFile(
  goalId: string,
  project: string,
  filename: string
): Promise<PlanningFile> {
  const res = await fetch(
    `${API_BASE}/goals/${goalId}/planning-files/${encodeURIComponent(project)}/${encodeURIComponent(filename)}`
  )
  if (!res.ok) throw new Error(`Failed to fetch planning file: ${res.statusText}`)
  return res.json()
}

// Hierarchy API
export async function setGoalParent(
  goalId: string,
  parentId: string | null
): Promise<{ success: boolean }> {
  const res = await fetch(`${API_BASE}/goals/${goalId}/parent`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ parent_id: parentId }),
  })
  if (!res.ok) throw new Error(`Failed to set goal parent: ${res.statusText}`)
  return res.json()
}

// Spawn executor with meta option
export async function spawnExecutor(
  goalId: string,
  options: {
    context?: string
    mode?: string
    meta?: boolean
  } = {}
): Promise<{ success: boolean; session_id?: string; message?: string }> {
  const res = await fetch(`${API_BASE}/goals/${goalId}/spawn`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(options),
  })
  return res.json()
}
