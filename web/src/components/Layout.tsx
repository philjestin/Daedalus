import { useState } from 'react'
import { Outlet, NavLink } from 'react-router-dom'
import {
  LayoutDashboard,
  FolderKanban,
  ListTodo,
  Printer,
  Package,
  Settings,
  Layers,
  Receipt,
  DollarSign,
  ShoppingBag,
  GanttChart,
  PanelLeftClose,
  PanelLeftOpen,
} from 'lucide-react'
import { cn } from '../lib/utils'
import DispatchNotificationProvider from './DispatchNotificationProvider'
import FeedbackButton from './FeedbackButton'

const navigation = [
  { name: 'Dashboard', href: '/dashboard', icon: LayoutDashboard },
  { name: 'Projects', href: '/projects', icon: FolderKanban },
  { name: 'Tasks', href: '/tasks', icon: ListTodo },
  { name: 'Printers', href: '/printers', icon: Printer },
  { name: 'Materials', href: '/materials', icon: Package },
  { name: 'Expenses', href: '/expenses', icon: Receipt },
  { name: 'Sales', href: '/sales', icon: DollarSign },
  { name: 'Channels', href: '/channels', icon: ShoppingBag },
  { name: 'Timeline', href: '/timeline', icon: GanttChart },
]

export default function Layout() {
  const [collapsed, setCollapsed] = useState(false)

  return (
    <div className="flex h-screen bg-surface-950">
      {/* Sidebar */}
      <aside
        className={cn(
          'border-r border-surface-800 bg-surface-900/50 flex flex-col shrink-0 transition-[width] duration-200',
          collapsed ? 'w-16' : 'w-64'
        )}
      >
        {/* Logo + collapse toggle */}
        <div className="h-16 flex items-center border-b border-surface-800 px-3">
          {collapsed ? (
            <button
              onClick={() => setCollapsed(false)}
              className="w-10 h-10 flex items-center justify-center rounded-lg text-surface-400 hover:text-surface-100 hover:bg-surface-800 transition-colors mx-auto"
              title="Expand sidebar"
            >
              <PanelLeftOpen className="h-5 w-5" />
            </button>
          ) : (
            <>
              <div className="flex items-center flex-1 pl-3">
                <Layers className="h-7 w-7 text-accent-500 shrink-0" />
                <span className="ml-3 text-xl font-display font-semibold text-surface-100 truncate">
                  Daedalus
                </span>
              </div>
              <button
                onClick={() => setCollapsed(true)}
                className="w-10 h-10 flex items-center justify-center rounded-lg text-surface-400 hover:text-surface-100 hover:bg-surface-800 transition-colors"
                title="Collapse sidebar"
              >
                <PanelLeftClose className="h-5 w-5" />
              </button>
            </>
          )}
        </div>

        {/* Navigation */}
        <nav className={cn('flex-1 py-4 space-y-1', collapsed ? 'px-2' : 'px-3')}>
          {navigation.map((item) => (
            <NavLink
              key={item.name}
              to={item.href}
              title={collapsed ? item.name : undefined}
              className={({ isActive }) =>
                cn(
                  'flex items-center rounded-lg text-sm font-medium transition-colors',
                  collapsed ? 'justify-center px-0 py-2.5' : 'px-3 py-2.5',
                  isActive
                    ? 'bg-accent-500/10 text-accent-400'
                    : 'text-surface-400 hover:text-surface-100 hover:bg-surface-800'
                )
              }
            >
              <item.icon className={cn('h-5 w-5 shrink-0', !collapsed && 'mr-3')} />
              {!collapsed && item.name}
            </NavLink>
          ))}
        </nav>

        {/* Settings */}
        <div className={cn('py-4 border-t border-surface-800', collapsed ? 'px-2' : 'px-3')}>
          <NavLink
            to="/settings"
            title={collapsed ? 'Settings' : undefined}
            className={({ isActive }) =>
              cn(
                'flex items-center rounded-lg text-sm font-medium transition-colors',
                collapsed ? 'justify-center px-0 py-2.5' : 'px-3 py-2.5',
                isActive
                  ? 'bg-accent-500/10 text-accent-400'
                  : 'text-surface-400 hover:text-surface-100 hover:bg-surface-800'
              )
            }
          >
            <Settings className={cn('h-5 w-5 shrink-0', !collapsed && 'mr-3')} />
            {!collapsed && 'Settings'}
          </NavLink>
        </div>
      </aside>

      {/* Main content */}
      <main className="flex-1 overflow-auto min-w-0">
        <Outlet />
      </main>

      {/* Global dispatch notification */}
      <DispatchNotificationProvider />

      {/* Beta feedback button */}
      <FeedbackButton />
    </div>
  )
}
