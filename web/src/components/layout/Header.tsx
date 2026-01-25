import { Badge } from '@/components/ui/badge'
import { cn } from '@/lib/utils'

interface HeaderProps {
  connected: boolean
  pendingQuestions: number
}

export function Header({ connected, pendingQuestions }: HeaderProps) {
  return (
    <header className="sticky top-0 z-40 border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="flex h-14 items-center justify-between px-4">
        <div className="flex items-center gap-3">
          <h1 className="text-xl font-bold">vega-hub</h1>
          <div
            className={cn(
              "h-2 w-2 rounded-full",
              connected ? "bg-green-500" : "bg-red-500"
            )}
            title={connected ? "Connected" : "Disconnected"}
          />
        </div>

        {pendingQuestions > 0 && (
          <Badge variant="destructive" className="gap-1.5">
            <span className="text-sm">Pending</span>
            <span className="font-bold">{pendingQuestions}</span>
          </Badge>
        )}
      </div>
    </header>
  )
}
