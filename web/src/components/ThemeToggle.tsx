import { useEffect, useState } from 'react'
import { Moon, Sun } from 'lucide-react'
import { cn } from '../lib/utils'
import { applyTheme, getStoredTheme, type Theme } from '../lib/theme'

export default function ThemeToggle({ collapsed = false }: { collapsed?: boolean }) {
  const [theme, setTheme] = useState<Theme>(() => getStoredTheme())

  useEffect(() => {
    applyTheme(theme)
  }, [theme])

  const isLight = theme === 'light'
  const label = isLight ? 'Switch to dark theme' : 'Switch to light theme'

  return (
    <button
      type="button"
      onClick={() => setTheme(isLight ? 'dark' : 'light')}
      aria-label={label}
      title={label}
      className={cn(
        'flex items-center justify-center rounded-full text-body transition-colors hover:bg-surface-raised hover:text-ink',
        'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-focus focus-visible:ring-offset-2 focus-visible:ring-offset-canvas',
        collapsed ? 'mx-auto h-10 w-10' : 'h-10 w-10'
      )}
    >
      {isLight ? <Moon className="h-5 w-5" /> : <Sun className="h-5 w-5" />}
    </button>
  )
}
