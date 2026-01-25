import { NavLink } from 'react-router-dom'
import { Home, FolderOpen, Target, History } from 'lucide-react'
import { cn } from '@/lib/utils'

interface BottomNavProps {
  pendingQuestions: number
}

const navItems = [
  { to: '/', icon: Home, label: 'Home' },
  { to: '/projects', icon: FolderOpen, label: 'Projects' },
  { to: '/goals', icon: Target, label: 'Goals' },
  { to: '/history', icon: History, label: 'History' },
]

export function BottomNav({ pendingQuestions }: BottomNavProps) {
  return (
    <nav className="fixed bottom-0 left-0 right-0 z-40 border-t bg-background lg:hidden">
      <div className="flex h-16 items-center justify-around">
        {navItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            className={({ isActive }) =>
              cn(
                "flex flex-col items-center justify-center gap-1 px-3 py-2 text-xs font-medium transition-colors relative",
                isActive
                  ? "text-primary"
                  : "text-muted-foreground hover:text-foreground"
              )
            }
          >
            <div className="relative">
              <item.icon className="h-5 w-5" />
              {item.to === '/' && pendingQuestions > 0 && (
                <span className="absolute -right-2 -top-1 flex h-4 w-4 items-center justify-center rounded-full bg-destructive text-[10px] font-bold text-destructive-foreground">
                  {pendingQuestions > 9 ? '9+' : pendingQuestions}
                </span>
              )}
            </div>
            <span>{item.label}</span>
          </NavLink>
        ))}
      </div>
    </nav>
  )
}
