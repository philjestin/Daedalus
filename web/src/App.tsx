import { Routes, Route, Navigate } from 'react-router-dom'
import * as Sentry from '@sentry/react'
import Layout from './components/Layout'
import Dashboard from './pages/Dashboard'
import Projects from './pages/Projects'
import ProjectDetail from './pages/ProjectDetail'
import { Tasks } from './pages/Tasks'
import { TaskDetail } from './pages/TaskDetail'
import Printers from './pages/Printers'
import PrinterDetail from './pages/PrinterDetail'
import Materials from './pages/Materials'
import Expenses from './pages/Expenses'
import Sales from './pages/Sales'
import Settings from './pages/Settings'
import Channels from './pages/Channels'
import Orders from './pages/Orders'
import OrderDetail from './pages/OrderDetail'
import Timeline from './pages/Timeline'
import Quotes from './pages/Quotes'
import QuoteDetail from './pages/QuoteDetail'
import Customers from './pages/Customers'
import CustomerDetail from './pages/CustomerDetail'
import ErrorFallback from './components/ErrorFallback'
import { useWebSocket } from './hooks/useWebSocket'

function App() {
  // Establish WebSocket connection for real-time updates
  const { status } = useWebSocket()

  // Log WebSocket status in development
  if (import.meta.env.DEV && status !== 'connected') {
    console.log('[App] WebSocket status:', status)
  }

  return (
    <Sentry.ErrorBoundary fallback={<ErrorFallback />}>
    <Routes>
      <Route element={<Layout />}>
        <Route path="/" element={<Navigate to="/dashboard" replace />} />
        <Route path="/dashboard" element={<Dashboard />} />
        <Route path="/orders" element={<Orders />} />
        <Route path="/orders/:id" element={<OrderDetail />} />
        <Route path="/quotes" element={<Quotes />} />
        <Route path="/quotes/:id" element={<QuoteDetail />} />
        <Route path="/customers" element={<Customers />} />
        <Route path="/customers/:id" element={<CustomerDetail />} />
        <Route path="/timeline" element={<Timeline />} />
        <Route path="/projects" element={<Projects />} />
        <Route path="/projects/:id" element={<ProjectDetail />} />
        <Route path="/tasks" element={<Tasks />} />
        <Route path="/tasks/:id" element={<TaskDetail />} />
        {/* Legacy template routes redirect to projects */}
        <Route path="/templates/*" element={<Navigate to="/projects" replace />} />
        <Route path="/printers" element={<Printers />} />
        <Route path="/printers/:id" element={<PrinterDetail />} />
        <Route path="/materials" element={<Materials />} />
        <Route path="/expenses" element={<Expenses />} />
        <Route path="/sales" element={<Sales />} />
        <Route path="/channels" element={<Channels />} />
        {/* Legacy routes redirect to unified Channels page */}
        <Route path="/etsy/*" element={<Navigate to="/channels" replace />} />
        <Route path="/squarespace/*" element={<Navigate to="/channels" replace />} />
        <Route path="/settings" element={<Settings />} />
      </Route>
    </Routes>
    </Sentry.ErrorBoundary>
  )
}

export default App
