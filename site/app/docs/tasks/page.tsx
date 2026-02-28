export const metadata = {
  title: "Tasks & Print Jobs - Daedalus Docs",
  description: "Task lifecycle, print jobs, auto-dispatch, checklists, and job event tracking in Daedalus.",
};

export default function TasksDocs() {
  return (
    <div className="docs-prose">
      <h1>Tasks &amp; Print Jobs</h1>
      <p className="text-lg !text-surface-300">
        The core production workflow. Tasks track what needs to be printed, print jobs
        execute the work on your printers, and auto-dispatch keeps everything moving.
      </p>

      <div className="docs-screenshot">
        <img src="/screenshots/tasks.png" alt="Daedalus task view showing order fulfillment progress with completion tracking" />
      </div>

      <h2>Overview</h2>
      <p>
        When you process an order, Daedalus creates tasks &mdash; one for each order line item.
        Each task has a checklist of parts that need to be printed. You can then create print
        jobs from checklist items and dispatch them to printers manually or automatically.
      </p>

      <h2>Task Lifecycle</h2>
      <table>
        <thead>
          <tr>
            <th>Status</th>
            <th>Description</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td><strong>Pending</strong></td>
            <td>Task created, no work started</td>
          </tr>
          <tr>
            <td><strong>In Progress</strong></td>
            <td>At least one print job is active</td>
          </tr>
          <tr>
            <td><strong>Completed</strong></td>
            <td>All checklist items printed successfully</td>
          </tr>
          <tr>
            <td><strong>Cancelled</strong></td>
            <td>Task was cancelled</td>
          </tr>
        </tbody>
      </table>

      <h2>Checklists</h2>
      <p>
        Each task has an auto-generated checklist based on the project&apos;s parts definition.
        For a product with 3 parts at quantity 2, you get 6 checklist items.
      </p>
      <ul>
        <li>Each checklist item represents one instance of one part</li>
        <li>Items track their status: pending, printing, completed, or failed</li>
        <li>Click a checklist item to create a print job for it</li>
        <li>Checklists can be regenerated if the project definition changes</li>
      </ul>

      <h2>Print Jobs</h2>
      <p>
        A print job represents a single print on a single printer. Jobs track the full
        lifecycle from creation to completion:
      </p>
      <table>
        <thead>
          <tr>
            <th>Status</th>
            <th>Description</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td><strong>Queued</strong></td>
            <td>Job created, waiting to be sent to a printer</td>
          </tr>
          <tr>
            <td><strong>Sending</strong></td>
            <td>File being uploaded to the printer</td>
          </tr>
          <tr>
            <td><strong>Printing</strong></td>
            <td>Printer is actively executing the job</td>
          </tr>
          <tr>
            <td><strong>Completed</strong></td>
            <td>Print finished successfully</td>
          </tr>
          <tr>
            <td><strong>Failed</strong></td>
            <td>Print failed (can be retried)</td>
          </tr>
          <tr>
            <td><strong>Cancelled</strong></td>
            <td>Job was manually cancelled</td>
          </tr>
        </tbody>
      </table>

      <h3>Job Events</h3>
      <p>
        Every print job maintains an immutable event log recording:
      </p>
      <ul>
        <li>Status transitions with timestamps</li>
        <li>Printer assignment and material used</li>
        <li>Progress updates during printing</li>
        <li>Error messages if the job fails</li>
        <li>Material cost snapshot at the time of printing</li>
      </ul>

      <h2>Auto-Dispatch</h2>
      <p>
        Auto-dispatch automatically assigns queued print jobs to available printers based on:
      </p>
      <ul>
        <li><strong>Material compatibility</strong> &mdash; The printer must have the required filament loaded</li>
        <li><strong>Printer constraints</strong> &mdash; The part may specify which printers can print it</li>
        <li><strong>Printer availability</strong> &mdash; The printer must be idle and online</li>
      </ul>
      <p>
        When a match is found, Daedalus presents a dispatch request with a human-in-the-loop
        confirmation step. You can <strong>confirm</strong>, <strong>reject</strong>, or
        <strong>skip</strong> each dispatch suggestion.
      </p>

      <h2>Creating Print Jobs</h2>
      <ol>
        <li>Open a task and find the checklist item you want to print</li>
        <li>Click the <strong>Print</strong> button on the checklist item</li>
        <li>Select a printer and confirm the material</li>
        <li>The job enters the queue and will be dispatched when the printer is ready</li>
      </ol>

      <h3>Retry on Failure</h3>
      <p>
        If a print job fails, you can retry it. Daedalus creates a new job linked to the
        original via a retry chain, preserving the full history of attempts.
      </p>

      <div className="docs-callout">
        <strong>Tip:</strong> Enable auto-dispatch in Settings to let Daedalus automatically
        match jobs to printers. You still get to confirm or reject each suggestion, so you
        stay in control.
      </div>
    </div>
  );
}
