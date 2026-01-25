import { Play, Square, MessageSquare, CheckCircle2, Snowflake, RefreshCw } from 'lucide-react'
import { cn } from '@/lib/utils'
import type { Activity } from '@/lib/types'

interface ActivityItemProps {
  activity: Activity
  className?: string
  onClick?: () => void
}

const iconMap = {
  executor_started: Play,
  executor_stopped: Square,
  question: MessageSquare,
  answered: CheckCircle2,
  goal_updated: RefreshCw,
  goal_iced: Snowflake,
  goal_completed: CheckCircle2,
}

const colorMap = {
  executor_started: 'text-green-500',
  executor_stopped: 'text-muted-foreground',
  question: 'text-destructive',
  answered: 'text-green-500',
  goal_updated: 'text-blue-500',
  goal_iced: 'text-blue-400',
  goal_completed: 'text-green-500',
}

export function ActivityItem({ activity, className, onClick }: ActivityItemProps) {
  const Icon = iconMap[activity.type]
  const iconColor = colorMap[activity.type]

  const timeAgo = getTimeAgo(activity.timestamp)

  return (
    <div
      className={cn(
        'flex items-start gap-3 py-2',
        onClick && 'cursor-pointer hover:bg-accent/50 rounded-lg px-2 -mx-2',
        className
      )}
      onClick={onClick}
    >
      <div className={cn('mt-0.5 shrink-0', iconColor)}>
        <Icon className="h-4 w-4" />
      </div>
      <div className="flex-1 min-w-0">
        <p className="text-sm leading-tight">{activity.message}</p>
        <div className="flex items-center gap-2 mt-0.5 text-xs text-muted-foreground">
          {activity.goal_id && <span>#{activity.goal_id}</span>}
          <span>{timeAgo}</span>
        </div>
      </div>
    </div>
  )
}

function getTimeAgo(timestamp: string): string {
  const now = new Date()
  const then = new Date(timestamp)
  const diffMs = now.getTime() - then.getTime()
  const diffSec = Math.floor(diffMs / 1000)

  if (diffSec < 60) return 'just now'
  if (diffSec < 3600) return `${Math.floor(diffSec / 60)}m ago`
  if (diffSec < 86400) return `${Math.floor(diffSec / 3600)}h ago`
  return then.toLocaleDateString()
}
