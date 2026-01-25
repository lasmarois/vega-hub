import { Outlet } from 'react-router-dom'
import { Header } from './Header'
import { Sidebar } from './Sidebar'
import { BottomNav } from './BottomNav'

interface LayoutProps {
  connected: boolean
  pendingQuestions: number
}

export function Layout({ connected, pendingQuestions }: LayoutProps) {
  return (
    <div className="min-h-screen bg-background">
      {/* Desktop Sidebar */}
      <Sidebar pendingQuestions={pendingQuestions} />

      {/* Mobile/Tablet Header */}
      <div className="lg:hidden">
        <Header connected={connected} pendingQuestions={pendingQuestions} />
      </div>

      {/* Main Content */}
      <main className="lg:pl-60">
        {/* Desktop header with connection status */}
        <div className="hidden lg:block sticky top-0 z-30 border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
          <div className="flex h-14 items-center justify-end px-6">
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <span>{connected ? 'Connected' : 'Disconnected'}</span>
              <div
                className={`h-2 w-2 rounded-full ${
                  connected ? 'bg-green-500' : 'bg-red-500'
                }`}
              />
            </div>
          </div>
        </div>

        {/* Page content with bottom padding for mobile nav */}
        <div className="pb-20 lg:pb-0">
          <Outlet />
        </div>
      </main>

      {/* Mobile/Tablet Bottom Nav */}
      <BottomNav pendingQuestions={pendingQuestions} />
    </div>
  )
}
