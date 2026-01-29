import { Navigate } from 'react-router-dom'

export default function Landing() {
  // Desktop app - redirect directly to dashboard
  return <Navigate to="/dashboard" replace />
}
