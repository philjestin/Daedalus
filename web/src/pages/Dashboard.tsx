import { Link } from 'react-router-dom'
import { 
  FolderKanban, 
  Printer, 
  Play,
  AlertCircle,
  Clock
} from 'lucide-react'
import { useProjects } from '../hooks/useProjects'
import { usePrinters, usePrinterStates } from '../hooks/usePrinters'
import { cn, getStatusBadge } from '../lib/utils'

export default function Dashboard() {
  const { data: projects = [], isLoading: projectsLoading } = useProjects()
  const { data: printers = [], isLoading: printersLoading } = usePrinters()
  const { data: printerStates = {} } = usePrinterStates()

  const activeProjects = projects.filter(p => p.status === 'active')
  const printingPrinters = printers.filter(p => printerStates[p.id]?.status === 'printing')
  const idlePrinters = printers.filter(p => 
    printerStates[p.id]?.status === 'idle' || !printerStates[p.id]
  )
  const errorPrinters = printers.filter(p => 
    printerStates[p.id]?.status === 'error'
  )

  return (
    <div className="p-8">
      {/* Header */}
      <div className="mb-8">
        <h1 className="text-3xl font-display font-bold text-surface-100">
          Dashboard
        </h1>
        <p className="text-surface-400 mt-1">
          Overview of your print farm
        </p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-4 gap-4 mb-8">
        <StatCard
          icon={FolderKanban}
          label="Active Projects"
          value={activeProjects.length}
          color="text-blue-400"
        />
        <StatCard
          icon={Play}
          label="Printing Now"
          value={printingPrinters.length}
          color="text-emerald-400"
        />
        <StatCard
          icon={Clock}
          label="Idle Printers"
          value={idlePrinters.length}
          color="text-surface-400"
        />
        <StatCard
          icon={AlertCircle}
          label="Errors"
          value={errorPrinters.length}
          color="text-red-400"
        />
      </div>

      <div className="grid grid-cols-2 gap-6">
        {/* Printer Status */}
        <div className="card p-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-surface-100">
              Printer Fleet
            </h2>
            <Link 
              to="/printers" 
              className="text-sm text-accent-400 hover:text-accent-300"
            >
              View all →
            </Link>
          </div>
          
          {printersLoading ? (
            <div className="text-surface-500">Loading...</div>
          ) : printers.length === 0 ? (
            <div className="text-center py-8 text-surface-500">
              <Printer className="h-12 w-12 mx-auto mb-3 opacity-50" />
              <p>No printers configured</p>
              <Link 
                to="/printers" 
                className="text-accent-400 hover:text-accent-300 text-sm"
              >
                Add your first printer
              </Link>
            </div>
          ) : (
            <div className="space-y-3">
              {printers.slice(0, 5).map((printer) => {
                const state = printerStates[printer.id]
                return (
                  <div 
                    key={printer.id}
                    className="flex items-center justify-between p-3 rounded-lg bg-surface-800/50"
                  >
                    <div className="flex items-center gap-3">
                      <div className={cn(
                        'w-2 h-2 rounded-full',
                        state?.status === 'printing' ? 'bg-emerald-400 animate-pulse' :
                        state?.status === 'idle' ? 'bg-surface-500' :
                        state?.status === 'error' ? 'bg-red-400' :
                        'bg-surface-600'
                      )} />
                      <div>
                        <div className="font-medium text-surface-100">
                          {printer.name}
                        </div>
                        <div className="text-xs text-surface-500">
                          {printer.model || printer.manufacturer || 'Unknown model'}
                        </div>
                      </div>
                    </div>
                    <div className="text-right">
                      <span className={cn(
                        'badge',
                        getStatusBadge(state?.status || 'offline')
                      )}>
                        {state?.status || 'offline'}
                      </span>
                      {state?.status === 'printing' && state.progress > 0 && (
                        <div className="text-xs text-surface-400 mt-1">
                          {state.progress.toFixed(0)}%
                        </div>
                      )}
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </div>

        {/* Active Projects */}
        <div className="card p-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-surface-100">
              Active Projects
            </h2>
            <Link 
              to="/projects" 
              className="text-sm text-accent-400 hover:text-accent-300"
            >
              View all →
            </Link>
          </div>
          
          {projectsLoading ? (
            <div className="text-surface-500">Loading...</div>
          ) : activeProjects.length === 0 ? (
            <div className="text-center py-8 text-surface-500">
              <FolderKanban className="h-12 w-12 mx-auto mb-3 opacity-50" />
              <p>No active projects</p>
              <Link 
                to="/projects" 
                className="text-accent-400 hover:text-accent-300 text-sm"
              >
                Create a project
              </Link>
            </div>
          ) : (
            <div className="space-y-3">
              {activeProjects.slice(0, 5).map((project) => (
                <Link 
                  key={project.id}
                  to={`/projects/${project.id}`}
                  className="block p-3 rounded-lg bg-surface-800/50 hover:bg-surface-800 transition-colors"
                >
                  <div className="flex items-center justify-between">
                    <div className="font-medium text-surface-100">
                      {project.name}
                    </div>
                    <span className={cn('badge', getStatusBadge(project.status))}>
                      {project.status}
                    </span>
                  </div>
                  {project.description && (
                    <p className="text-sm text-surface-500 mt-1 truncate">
                      {project.description}
                    </p>
                  )}
                </Link>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

function StatCard({ 
  icon: Icon, 
  label, 
  value, 
  color 
}: { 
  icon: React.ElementType
  label: string
  value: number
  color: string
}) {
  return (
    <div className="card p-4">
      <div className="flex items-center gap-3">
        <div className={cn('p-2 rounded-lg bg-surface-800', color)}>
          <Icon className="h-5 w-5" />
        </div>
        <div>
          <div className="text-2xl font-bold text-surface-100">{value}</div>
          <div className="text-sm text-surface-500">{label}</div>
        </div>
      </div>
    </div>
  )
}

