import { useState, useRef, useEffect } from 'react'
import { Info } from 'lucide-react'

interface TooltipProps {
  text: string
  children?: React.ReactNode
}

export function Tooltip({ text, children }: TooltipProps) {
  const [show, setShow] = useState(false)
  const [position, setPosition] = useState<'bottom' | 'top'>('bottom')
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (show && ref.current) {
      const rect = ref.current.getBoundingClientRect()
      // If tooltip would go below viewport, show above
      if (rect.bottom + 80 > window.innerHeight) {
        setPosition('top')
      } else {
        setPosition('bottom')
      }
    }
  }, [show])

  return (
    <div
      className="relative inline-flex"
      ref={ref}
      onMouseEnter={() => setShow(true)}
      onMouseLeave={() => setShow(false)}
    >
      {children || <Info className="h-3.5 w-3.5 text-surface-600 hover:text-surface-400 cursor-help transition-colors" />}
      {show && (
        <div
          className={`absolute z-50 left-1/2 -translate-x-1/2 px-3 py-2 text-xs text-surface-200 bg-surface-800 border border-surface-700 rounded-lg shadow-lg w-56 leading-relaxed pointer-events-none ${
            position === 'bottom' ? 'top-full mt-1.5' : 'bottom-full mb-1.5'
          }`}
        >
          {text}
        </div>
      )}
    </div>
  )
}
