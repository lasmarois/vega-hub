import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'

interface StatusBadgeProps {
  status: 'active' | 'iced' | 'completed' | 'running' | 'waiting' | 'stopped' | 'idle'
  className?: string
}

export function StatusBadge({ status, className }: StatusBadgeProps) {
  const variant =
    status === 'completed' || status === 'running' ? 'success' :
    status === 'waiting' ? 'destructive' :
    'secondary'

  const label = status.toUpperCase()

  return (
    <Badge variant={variant} className={cn(className)}>
      {label}
    </Badge>
  )
}
