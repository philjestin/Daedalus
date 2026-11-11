import { Layers } from 'lucide-react'
import ThemeToggle from './ThemeToggle'

export default function AuthScreen({
  title,
  subtitle,
  children,
}: {
  title: string
  subtitle: string
  children: React.ReactNode
}) {
  return (
    <div className="relative min-h-screen bg-canvas text-ink flex items-center justify-center p-6">
      <div className="absolute top-4 right-4">
        <ThemeToggle />
      </div>
      <div className="w-full max-w-md border border-line bg-surface p-8">
        <div className="flex items-center gap-3 mb-8">
          <Layers className="h-8 w-8 text-primary" />
          <span className="font-display text-2xl font-semibold">RankOS</span>
        </div>
        <h1 className="font-display text-heading-sm font-semibold mb-2">{title}</h1>
        <p className="text-body mb-6">{subtitle}</p>
        {children}
      </div>
    </div>
  )
}
