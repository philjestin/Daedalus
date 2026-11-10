import { useState } from 'react'
import { MessageSquare } from 'lucide-react'
import FeedbackModal from './FeedbackModal'

export default function FeedbackButton() {
  const [open, setOpen] = useState(false)

  return (
    <>
      <button
        onClick={() => setOpen(true)}
        className="fixed bottom-6 right-6 z-40 flex items-center gap-2 rounded-full bg-primary px-[30px] py-[7px] font-sans text-[17px] font-medium leading-none text-primary-fg shadow-card transition-colors hover:bg-primary/90 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-focus focus-visible:ring-offset-2 focus-visible:ring-offset-canvas"
      >
        <MessageSquare className="h-4 w-4" />
        Feedback
      </button>

      {open && <FeedbackModal onClose={() => setOpen(false)} />}
    </>
  )
}
