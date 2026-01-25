import { NavLink } from 'react-router-dom'
import { Home, FolderOpen, Target, History } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'

interface SidebarProps {
  pendingQuestions: number
}

const navItems = [
  { to: '/', icon: Home, label: 'Home' },
  { to: '/projects', icon: FolderOpen, label: 'Projects' },
  { to: '/goals', icon: Target, label: 'Goals' },
  { to: '/history', icon: History, label: 'History' },
]

export function Sidebar({ pendingQuestions }: SidebarProps) {
  return (
    <aside className="hidden lg:flex lg:w-60 lg:flex-col lg:fixed lg:inset-y-0 lg:z-30 lg:border-r lg:bg-background">
      <div className="flex h-14 items-center border-b px-6">
        <h1 className="text-xl font-bold">vega-hub</h1>
      </div>
      <nav className="flex-1 space-y-1 p-4">
        {navItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            className={({ isActive }) =>
              cn(
                "flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors",
                isActive
                  ? "bg-secondary text-secondary-foreground"
                  : "text-muted-foreground hover:bg-secondary/50 hover:text-secondary-foreground"
              )
            }
          >
            <item.icon className="h-5 w-5" />
            <span>{item.label}</span>
            {item.to === '/' && pendingQuestions > 0 && (
              <Badge variant="destructive" className="ml-auto h-5 px-1.5 text-xs">
                {pendingQuestions}
              </Badge>
            )}
          </NavLink>
        ))}
      </nav>
    </aside>
  )
}
