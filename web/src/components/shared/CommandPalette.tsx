import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from '@/components/ui/command'
import { Home, Target, FolderOpen, History, Search } from 'lucide-react'
import type { GoalSummary } from '@/lib/types'

interface CommandPaletteProps {
  goals: GoalSummary[]
  onGoalSelect: (id: string) => void
}

export function CommandPalette({ goals, onGoalSelect }: CommandPaletteProps) {
  const [open, setOpen] = useState(false)
  const navigate = useNavigate()

  useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (e.key === 'k' && (e.metaKey || e.ctrlKey)) {
        e.preventDefault()
        setOpen((open) => !open)
      }
    }

    document.addEventListener('keydown', down)
    return () => document.removeEventListener('keydown', down)
  }, [])

  const handleSelect = (value: string) => {
    setOpen(false)

    if (value.startsWith('nav:')) {
      const path = value.replace('nav:', '')
      navigate(path)
    } else if (value.startsWith('goal:')) {
      const goalId = value.replace('goal:', '')
      onGoalSelect(goalId)
    }
  }

  const activeGoals = goals.filter(g => g.status === 'active')
  const goalsWithQuestions = goals.filter(g => g.pending_questions > 0)

  return (
    <CommandDialog open={open} onOpenChange={setOpen}>
      <CommandInput placeholder="Type a command or search..." />
      <CommandList>
        <CommandEmpty>No results found.</CommandEmpty>

        {/* Quick Navigation */}
        <CommandGroup heading="Navigation">
          <CommandItem value="nav:/" onSelect={handleSelect}>
            <Home className="mr-2 h-4 w-4" />
            <span>Home</span>
          </CommandItem>
          <CommandItem value="nav:/goals" onSelect={handleSelect}>
            <Target className="mr-2 h-4 w-4" />
            <span>Goals</span>
          </CommandItem>
          <CommandItem value="nav:/projects" onSelect={handleSelect}>
            <FolderOpen className="mr-2 h-4 w-4" />
            <span>Projects</span>
          </CommandItem>
          <CommandItem value="nav:/history" onSelect={handleSelect}>
            <History className="mr-2 h-4 w-4" />
            <span>History</span>
          </CommandItem>
        </CommandGroup>

        {/* Goals with pending questions */}
        {goalsWithQuestions.length > 0 && (
          <>
            <CommandSeparator />
            <CommandGroup heading="Needs Attention">
              {goalsWithQuestions.map((goal) => (
                <CommandItem
                  key={goal.id}
                  value={`goal:${goal.id}`}
                  onSelect={handleSelect}
                >
                  <span className="flex h-2 w-2 rounded-full bg-destructive mr-2" />
                  <span className="flex-1 truncate">
                    #{goal.id}: {goal.title}
                  </span>
                  <span className="text-xs text-destructive">
                    {goal.pending_questions} question{goal.pending_questions > 1 ? 's' : ''}
                  </span>
                </CommandItem>
              ))}
            </CommandGroup>
          </>
        )}

        {/* Active Goals */}
        {activeGoals.length > 0 && (
          <>
            <CommandSeparator />
            <CommandGroup heading="Active Goals">
              {activeGoals.slice(0, 5).map((goal) => (
                <CommandItem
                  key={goal.id}
                  value={`goal:${goal.id}`}
                  onSelect={handleSelect}
                >
                  <span className={`flex h-2 w-2 rounded-full mr-2 ${
                    goal.executor_status === 'running' ? 'bg-green-500' :
                    goal.executor_status === 'waiting' ? 'bg-destructive' :
                    'bg-muted-foreground'
                  }`} />
                  <span className="flex-1 truncate">
                    #{goal.id}: {goal.title}
                  </span>
                  <span className="text-xs text-muted-foreground">
                    {goal.executor_status}
                  </span>
                </CommandItem>
              ))}
            </CommandGroup>
          </>
        )}

        {/* Search hint */}
        <CommandSeparator />
        <CommandGroup heading="Tips">
          <CommandItem disabled>
            <Search className="mr-2 h-4 w-4" />
            <span className="text-muted-foreground">
              Press <kbd className="bg-muted px-1 rounded">Cmd+K</kbd> to open anytime
            </span>
          </CommandItem>
        </CommandGroup>
      </CommandList>
    </CommandDialog>
  )
}
