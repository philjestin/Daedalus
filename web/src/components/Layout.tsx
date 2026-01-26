import { Outlet, NavLink } from 'react-router-dom'
import { 
  LayoutDashboard, 
  FolderKanban, 
  Printer, 
  Package,
  Settings,
  Layers
} from 'lucide-react'
import { cn } from '../lib/utils'

const navigation = [
  { name: 'Dashboard', href: '/', icon: LayoutDashboard },
  { name: 'Projects', href: '/projects', icon: FolderKanban },
  { name: 'Printers', href: '/printers', icon: Printer },
  { name: 'Materials', href: '/materials', icon: Package },
]

export default function Layout() {
  return (
    <div className="flex h-screen bg-surface-950">
      {/* Sidebar */}
      <aside className="w-64 border-r border-surface-800 bg-surface-900/50 flex flex-col">
        {/* Logo */}
        <div className="h-16 flex items-center px-6 border-b border-surface-800">
          <Layers className="h-8 w-8 text-accent-500" />
          <span className="ml-3 text-xl font-display font-semibold text-surface-100">
            PrintFarm
          </span>
        </div>

        {/* Navigation */}
        <nav className="flex-1 px-3 py-4 space-y-1">
          {navigation.map((item) => (
            <NavLink
              key={item.name}
              to={item.href}
              className={({ isActive }) =>
                cn(
                  'flex items-center px-3 py-2.5 rounded-lg text-sm font-medium transition-colors',
                  isActive
                    ? 'bg-accent-500/10 text-accent-400'
                    : 'text-surface-400 hover:text-surface-100 hover:bg-surface-800'
                )
              }
            >
              <item.icon className="h-5 w-5 mr-3" />
              {item.name}
            </NavLink>
          ))}
        </nav>

        {/* Settings */}
        <div className="px-3 py-4 border-t border-surface-800">
          <NavLink
            to="/settings"
            className={({ isActive }) =>
              cn(
                'flex items-center px-3 py-2.5 rounded-lg text-sm font-medium transition-colors',
                isActive
                  ? 'bg-accent-500/10 text-accent-400'
                  : 'text-surface-400 hover:text-surface-100 hover:bg-surface-800'
              )
            }
          >
            <Settings className="h-5 w-5 mr-3" />
            Settings
          </NavLink>
        </div>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-auto">
        <Outlet />
      </main>
    </div>
  )
}

