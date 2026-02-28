export const metadata = {
  title: "Dashboard - Daedalus Docs",
  description: "Learn about the Daedalus dashboard: stats cards, financial overview, charts, and fleet panel.",
};

export default function DashboardDocs() {
  return (
    <div className="docs-prose">
      <h1>Dashboard</h1>
      <p className="text-lg !text-surface-300">
        Your command center for the entire print farm. The dashboard gives you a real-time
        financial overview, fleet status, and production metrics at a glance.
      </p>

      <div className="docs-screenshot">
        <img src="/screenshots/dashboard.png" alt="Daedalus dashboard showing financial overview, revenue charts, and print farm status" />
      </div>

      <h2>Overview</h2>
      <p>
        The dashboard is the first page you see when you open Daedalus. It aggregates data
        from across the platform into a single view so you can understand the health of your
        business without clicking through multiple pages.
      </p>

      <h2>Key Features</h2>

      <h3>Stats Cards</h3>
      <p>
        The top row shows key financial metrics for the selected time period:
      </p>
      <ul>
        <li><strong>Total Revenue</strong> &mdash; Sum of all recorded sales</li>
        <li><strong>Total Expenses</strong> &mdash; Sum of all tracked expenses across all categories</li>
        <li><strong>Net Profit</strong> &mdash; Revenue minus expenses</li>
        <li><strong>Gross Margin</strong> &mdash; Profit as a percentage of revenue</li>
      </ul>

      <h3>Revenue Chart</h3>
      <p>
        A time-series chart showing revenue trends over the selected period. Hover over
        data points to see exact figures for each day or week.
      </p>

      <h3>Fleet Panel</h3>
      <p>
        A compact overview of all connected printers showing:
      </p>
      <ul>
        <li>Printer name and model</li>
        <li>Current status (idle, printing, error, offline)</li>
        <li>Active job progress percentage</li>
        <li>Nozzle and bed temperatures for connected printers</li>
      </ul>

      <h3>Recent Activity</h3>
      <p>
        A feed of recent events across the platform including completed print jobs,
        new orders, and status changes.
      </p>

      <div className="docs-callout">
        <strong>Tip:</strong> The dashboard updates in real-time via WebSocket. You don&apos;t
        need to refresh the page to see new data &mdash; printer status, job progress,
        and order counts update automatically.
      </div>
    </div>
  );
}
