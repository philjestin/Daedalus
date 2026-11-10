import { useState } from 'react'
import { MessageSquare } from 'lucide-react'
import FeedbackModal from './FeedbackModal'

export default function FeedbackButton() {
  const [open, setOpen] = useState(false)

  return (
    <>
      <button
        onClick={() => setOpen(true)}
        className="btn btn-primary fixed bottom-6 right-6 z-40 gap-2"
      >
        <MessageSquare className="h-4 w-4" />
        Feedback
      </button>

      {open && <FeedbackModal onClose={() => setOpen(false)} />}
    </>
  )
}
