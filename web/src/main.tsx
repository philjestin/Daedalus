import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { BrowserRouter } from 'react-router-dom'
import * as Sentry from '@sentry/react'
import App from './App'
import './index.css'

// In the Wails desktop app, <a target="_blank"> links don't open in the system
// browser. Intercept clicks on external links and use the Wails runtime to open
// them properly. In a regular browser this is a no-op.
document.addEventListener('click', (e) => {
  const link = (e.target as HTMLElement).closest('a')
  if (!link) return

  const href = link.getAttribute('href')
  if (!href || !href.startsWith('http')) return

  // Only intercept if Wails runtime is available (desktop app)
  const runtime = (window as Record<string, unknown>).runtime as
    | { BrowserOpenURL?: (url: string) => void }
    | undefined
  if (runtime?.BrowserOpenURL) {
    e.preventDefault()
    runtime.BrowserOpenURL(href)
  }
})

const sentryDsn = import.meta.env.VITE_SENTRY_DSN
if (sentryDsn) {
  Sentry.init({
    dsn: sentryDsn,
    environment: import.meta.env.MODE,
    tracesSampleRate: 0.2,
  })
}

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000 * 60, // 1 minute
      retry: 1,
    },
  },
})

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </QueryClientProvider>
  </StrictMode>,
)
