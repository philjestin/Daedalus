import { useState } from 'react'
import { MessageSquare } from 'lucide-react'
import FeedbackModal from './FeedbackModal'

export default function FeedbackButton() {
  const [open, setOpen] = useState(false)

  return (
    <>
      <button
        onClick={() => setOpen(true)}
        className="fixed bottom-6 right-6 z-40 flex items-center gap-2 px-4 py-2.5 bg-accent-600 hover:bg-accent-500 text-white rounded-full shadow-lg shadow-black/30 font-medium text-sm transition-colors"
      >
        <MessageSquare className="h-4 w-4" />
        Feedback
      </button>

      {open && <FeedbackModal onClose={() => setOpen(false)} />}
    </>
  )
}
