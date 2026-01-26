import { useState, useEffect, useCallback } from 'react'

export interface User {
  username: string
  home_dir: string
  uid: string
}

export interface CredentialStatus {
  source: string
  valid: boolean
  user?: string
  error?: string
}

export interface CredentialValidation {
  valid: boolean
  service: {
    name: string
    host: string
  }
  statuses: CredentialStatus[]
  fix_options: { command: string; description: string }[]
}

export function useUser() {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchUser = useCallback(async () => {
    try {
      setError(null)
      const res = await fetch('/api/user')
      if (!res.ok) {
        throw new Error(`HTTP ${res.status}: ${res.statusText}`)
      }
      const data = await res.json()
      setUser(data)
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to fetch user'
      console.error('Failed to fetch user:', err)
      setError(message)
    } finally {
      setLoading(false)
    }
  }, [])

  const fetchCredentials = useCallback(async (project: string): Promise<CredentialValidation | null> => {
    try {
      const res = await fetch(`/api/user/credentials/${encodeURIComponent(project)}`)
      if (!res.ok) {
        throw new Error(`HTTP ${res.status}: ${res.statusText}`)
      }
      return await res.json()
    } catch (err) {
      console.error('Failed to fetch credentials:', err)
      return null
    }
  }, [])

  // Fetch user on mount
  useEffect(() => {
    fetchUser()
  }, [fetchUser])

  return {
    user,
    loading,
    error,
    fetchUser,
    fetchCredentials,
  }
}
