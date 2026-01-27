import { useEffect, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useAuth } from '../contexts/AuthContext'

export default function VerifyMagicLink() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const { verifyToken, isAuthenticated } = useAuth()
  const [error, setError] = useState<string | null>(null)
  const [isVerifying, setIsVerifying] = useState(true)

  const token = searchParams.get('token')

  useEffect(() => {
    // Already authenticated, redirect to dashboard
    if (isAuthenticated) {
      navigate('/dashboard', { replace: true })
      return
    }

    // No token provided
    if (!token) {
      setError('Invalid magic link - no token provided')
      setIsVerifying(false)
      return
    }

    // Verify the token
    const verify = async () => {
      try {
        await verifyToken(token)
        // Verification successful, redirect to dashboard
        navigate('/dashboard', { replace: true })
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to verify magic link')
        setIsVerifying(false)
      }
    }

    verify()
  }, [token, verifyToken, navigate, isAuthenticated])

  if (isVerifying) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 px-4">
        <div className="max-w-md w-full">
          <div className="bg-white rounded-lg shadow-md p-8 text-center">
            <div className="w-16 h-16 border-4 border-blue-600 border-t-transparent rounded-full animate-spin mx-auto mb-4" />
            <h2 className="text-xl font-semibold text-gray-900 mb-2">Signing you in...</h2>
            <p className="text-gray-600">Please wait while we verify your magic link.</p>
          </div>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 px-4">
        <div className="max-w-md w-full">
          <div className="bg-white rounded-lg shadow-md p-8 text-center">
            <div className="w-16 h-16 bg-red-100 rounded-full flex items-center justify-center mx-auto mb-4">
              <svg
                className="w-8 h-8 text-red-600"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
            </div>
            <h2 className="text-xl font-semibold text-gray-900 mb-2">Verification failed</h2>
            <p className="text-gray-600 mb-6">{error}</p>
            <button
              onClick={() => navigate('/login', { replace: true })}
              className="px-6 py-2 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-md transition-colors"
            >
              Try again
            </button>
          </div>
        </div>
      </div>
    )
  }

  return null
}
