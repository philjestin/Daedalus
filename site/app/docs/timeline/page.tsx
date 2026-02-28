export const metadata = {
  title: "Timeline - Daedalus Docs",
  description: "Gantt-style production timeline with status color coding, date range controls, and progress tracking.",
};

export default function TimelineDocs() {
  return (
    <div className="docs-prose">
      <h1>Timeline</h1>
      <p className="text-lg !text-surface-300">
        A Gantt-style view of your entire production pipeline. See orders, tasks, and print
        jobs laid out across time with color-coded status indicators.
      </p>

      <div className="docs-screenshot">
        <img src="/screenshots/timeline.png" alt="Daedalus production timeline showing Gantt-style order scheduling" />
      </div>

      <h2>Overview</h2>
      <p>
        The Timeline page gives you a bird&apos;s-eye view of all active production. Each row
        represents an order, with nested tasks and print jobs shown as horizontal bars
        across a time axis. This makes it easy to spot bottlenecks, see what&apos;s printing
        now, and plan ahead.
      </p>

      <h2>Key Features</h2>

      <h3>Gantt View</h3>
      <p>
        The timeline displays items as horizontal bars:
      </p>
      <ul>
        <li><strong>Orders</strong> &mdash; Span from creation date to expected completion</li>
        <li><strong>Tasks</strong> &mdash; Nested under their parent order</li>
        <li><strong>Print jobs</strong> &mdash; Show actual print duration on the timeline</li>
      </ul>

      <h3>Status Color Coding</h3>
      <table>
        <thead>
          <tr>
            <th>Color</th>
            <th>Status</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td>Gray</td>
            <td>Pending</td>
          </tr>
          <tr>
            <td>Blue</td>
            <td>In Progress</td>
          </tr>
          <tr>
            <td>Orange</td>
            <td>Printing</td>
          </tr>
          <tr>
            <td>Green</td>
            <td>Completed</td>
          </tr>
          <tr>
            <td>Purple</td>
            <td>Shipped</td>
          </tr>
          <tr>
            <td>Red</td>
            <td>Cancelled / Failed</td>
          </tr>
        </tbody>
      </table>

      <h3>Date Range Controls</h3>
      <p>
        Adjust the visible time window:
      </p>
      <ul>
        <li>Preset ranges: Today, This Week, This Month, Last 30 Days</li>
        <li>Custom date range picker for specific periods</li>
        <li>Zoom in/out to see more or less detail</li>
        <li>Scroll horizontally to navigate through time</li>
      </ul>

      <h3>Progress Tracking</h3>
      <p>
        Each bar shows completion progress:
      </p>
      <ul>
        <li>Orders show percentage of tasks completed</li>
        <li>Tasks show percentage of checklist items done</li>
        <li>Print jobs show real-time progress from the printer</li>
      </ul>

      <h2>Using the Timeline</h2>
      <ol>
        <li>Open the <strong>Timeline</strong> page from the sidebar</li>
        <li>Set your desired date range using the controls at the top</li>
        <li>Click any bar to jump to the detail view for that order, task, or job</li>
        <li>Use the color legend to quickly identify items that need attention</li>
      </ol>

      <div className="docs-callout">
        <strong>Tip:</strong> Check the timeline at the start of each day to plan your
        production schedule. Items in gray (pending) or blue (in progress) need your attention.
      </div>
    </div>
  );
}
