import { Link } from 'react-router-dom'
import { Layers, Printer, Package, BarChart3, Zap, Shield, ArrowRight } from 'lucide-react'
import { useAuth } from '../contexts/AuthContext'

export default function Landing() {
  const { isAuthenticated } = useAuth()

  return (
    <div className="min-h-screen bg-gradient-to-b from-surface-950 to-surface-900">
      {/* Header */}
      <header className="border-b border-surface-800">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center">
              <Layers className="h-8 w-8 text-accent-500" />
              <span className="ml-3 text-xl font-display font-semibold text-surface-100">
                Daedalus
              </span>
            </div>
            <div>
              {isAuthenticated ? (
                <Link
                  to="/dashboard"
                  className="inline-flex items-center px-4 py-2 bg-accent-500 hover:bg-accent-600 text-white font-medium rounded-lg transition-colors"
                >
                  Go to Dashboard
                  <ArrowRight className="ml-2 h-4 w-4" />
                </Link>
              ) : (
                <Link
                  to="/login"
                  className="inline-flex items-center px-4 py-2 bg-accent-500 hover:bg-accent-600 text-white font-medium rounded-lg transition-colors"
                >
                  Sign In
                  <ArrowRight className="ml-2 h-4 w-4" />
                </Link>
              )}
            </div>
          </div>
        </div>
      </header>

      {/* Hero Section */}
      <section className="py-20 px-4 sm:px-6 lg:px-8">
        <div className="max-w-4xl mx-auto text-center">
          <h1 className="text-4xl sm:text-5xl lg:text-6xl font-display font-bold text-surface-100 mb-6">
            Manage Your 3D Print Farm
            <span className="text-accent-500"> Like a Pro</span>
          </h1>
          <p className="text-xl text-surface-400 mb-10 max-w-2xl mx-auto">
            The complete solution for managing multiple 3D printers, tracking jobs,
            organizing materials, and running your print business efficiently.
          </p>
          <div className="flex flex-col sm:flex-row gap-4 justify-center">
            {isAuthenticated ? (
              <Link
                to="/dashboard"
                className="inline-flex items-center justify-center px-8 py-3 bg-accent-500 hover:bg-accent-600 text-white font-semibold rounded-lg transition-colors text-lg"
              >
                Go to Dashboard
                <ArrowRight className="ml-2 h-5 w-5" />
              </Link>
            ) : (
              <Link
                to="/login"
                className="inline-flex items-center justify-center px-8 py-3 bg-accent-500 hover:bg-accent-600 text-white font-semibold rounded-lg transition-colors text-lg"
              >
                Get Started Free
                <ArrowRight className="ml-2 h-5 w-5" />
              </Link>
            )}
          </div>
        </div>
      </section>

      {/* Features Grid */}
      <section className="py-20 px-4 sm:px-6 lg:px-8 border-t border-surface-800">
        <div className="max-w-6xl mx-auto">
          <h2 className="text-3xl font-display font-bold text-surface-100 text-center mb-12">
            Everything you need to run your print farm
          </h2>
          <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-8">
            <FeatureCard
              icon={Printer}
              title="Multi-Printer Management"
              description="Connect and monitor multiple printers from one dashboard. Support for Bambu Lab, OctoPrint, and more."
            />
            <FeatureCard
              icon={Package}
              title="Material Tracking"
              description="Keep track of your filament inventory, costs, and usage. Never run out mid-print again."
            />
            <FeatureCard
              icon={BarChart3}
              title="Job Analytics"
              description="Track print success rates, material usage, and costs. Make data-driven decisions."
            />
            <FeatureCard
              icon={Layers}
              title="Recipe Templates"
              description="Create reusable templates for common prints. Streamline order fulfillment."
            />
            <FeatureCard
              icon={Zap}
              title="Real-time Updates"
              description="Live printer status, progress tracking, and instant notifications when jobs complete."
            />
            <FeatureCard
              icon={Shield}
              title="Etsy Integration"
              description="Sync orders directly from your Etsy shop. Automate your workflow from sale to shipment."
            />
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="py-20 px-4 sm:px-6 lg:px-8 border-t border-surface-800">
        <div className="max-w-3xl mx-auto text-center">
          <h2 className="text-3xl font-display font-bold text-surface-100 mb-4">
            Ready to streamline your print farm?
          </h2>
          <p className="text-lg text-surface-400 mb-8">
            Join other makers who are scaling their 3D printing operations with Daedalus.
          </p>
          {!isAuthenticated && (
            <Link
              to="/login"
              className="inline-flex items-center justify-center px-8 py-3 bg-accent-500 hover:bg-accent-600 text-white font-semibold rounded-lg transition-colors text-lg"
            >
              Start Free Today
              <ArrowRight className="ml-2 h-5 w-5" />
            </Link>
          )}
        </div>
      </section>

      {/* Footer */}
      <footer className="py-8 px-4 sm:px-6 lg:px-8 border-t border-surface-800">
        <div className="max-w-6xl mx-auto flex flex-col sm:flex-row items-center justify-between gap-4">
          <div className="flex items-center text-surface-500">
            <Layers className="h-5 w-5 mr-2" />
            <span>Daedalus</span>
          </div>
          <p className="text-surface-500 text-sm">
            Built for makers, by makers.
          </p>
        </div>
      </footer>
    </div>
  )
}

function FeatureCard({
  icon: Icon,
  title,
  description,
}: {
  icon: React.ComponentType<{ className?: string }>
  title: string
  description: string
}) {
  return (
    <div className="bg-surface-900/50 border border-surface-800 rounded-xl p-6 hover:border-surface-700 transition-colors">
      <div className="w-12 h-12 bg-accent-500/10 rounded-lg flex items-center justify-center mb-4">
        <Icon className="h-6 w-6 text-accent-500" />
      </div>
      <h3 className="text-lg font-semibold text-surface-100 mb-2">{title}</h3>
      <p className="text-surface-400">{description}</p>
    </div>
  )
}
