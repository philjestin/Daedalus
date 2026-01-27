import { Routes, Route, Navigate } from 'react-router-dom'
import Layout from './components/Layout'
import Dashboard from './pages/Dashboard'
import Projects from './pages/Projects'
import ProjectDetail from './pages/ProjectDetail'
import Templates from './pages/Templates'
import TemplateDetail from './pages/TemplateDetail'
import Printers from './pages/Printers'
import Materials from './pages/Materials'
import Expenses from './pages/Expenses'
import Settings from './pages/Settings'
import EtsyOrders from './pages/EtsyOrders'
import EtsyListings from './pages/EtsyListings'
import { useWebSocket } from './hooks/useWebSocket'

function App() {
  // Establish WebSocket connection for real-time updates
  const { status } = useWebSocket()

  // Log WebSocket status in development
  if (import.meta.env.DEV && status !== 'connected') {
    console.log('[App] WebSocket status:', status)
  }

  return (
    <Routes>
      <Route element={<Layout />}>
        <Route path="/" element={<Navigate to="/dashboard" replace />} />
        <Route path="/dashboard" element={<Dashboard />} />
        <Route path="/projects" element={<Projects />} />
        <Route path="/projects/:id" element={<ProjectDetail />} />
        <Route path="/templates" element={<Templates />} />
        <Route path="/templates/:id" element={<TemplateDetail />} />
        <Route path="/printers" element={<Printers />} />
        <Route path="/materials" element={<Materials />} />
        <Route path="/expenses" element={<Expenses />} />
        <Route path="/etsy/orders" element={<EtsyOrders />} />
        <Route path="/etsy/listings" element={<EtsyListings />} />
        <Route path="/settings" element={<Settings />} />
      </Route>
    </Routes>
  )
}

export default App
