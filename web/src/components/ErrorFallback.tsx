export default function ErrorFallback() {
  return (
    <div className="flex items-center justify-center min-h-screen bg-surface-950">
      <div className="text-center p-8 max-w-md">
        <div className="w-16 h-16 rounded-full bg-red-500/20 flex items-center justify-center mx-auto mb-6">
          <svg className="w-8 h-8 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
          </svg>
        </div>
        <h1 className="text-2xl font-display font-semibold text-surface-100 mb-3">
          Something went wrong
        </h1>
        <p className="text-surface-400 mb-6">
          An unexpected error occurred. The error has been reported and we'll look into it.
        </p>
        <button
          onClick={() => window.location.reload()}
          className="px-6 py-2.5 bg-accent-600 hover:bg-accent-500 text-white rounded-lg font-medium transition-colors"
        >
          Reload Page
        </button>
      </div>
    </div>
  )
}
