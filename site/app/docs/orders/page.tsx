export const metadata = {
  title: "Orders - Daedalus Docs",
  description: "Create and manage orders in Daedalus: manual orders, channel imports, status pipeline, and task processing.",
};

export default function OrdersDocs() {
  return (
    <div className="docs-prose">
      <h1>Orders</h1>
      <p className="text-lg !text-surface-300">
        Unified order management for Etsy, Shopify, Squarespace, and direct sales.
        Track every order from receipt through printing to shipment.
      </p>

      <div className="docs-screenshot">
        <img src="/screenshots/orders.png" alt="Daedalus order management showing orders from multiple sales channels" />
      </div>

      <h2>Overview</h2>
      <p>
        The Orders page shows all orders from every source in a single list. Each order
        displays its source channel, customer info, items, status, and creation date.
        Orders can be filtered by status, channel, and date range.
      </p>

      <h2>Creating Orders</h2>

      <h3>Manual Orders</h3>
      <ol>
        <li>Click <strong>New Order</strong></li>
        <li>Enter customer name and any notes</li>
        <li>Add line items by selecting projects from your catalog</li>
        <li>Set quantities and any customization details</li>
        <li>The order is created in <strong>Pending</strong> status</li>
      </ol>

      <h3>From Accepted Quotes</h3>
      <p>
        When a <a href="/docs/quotes">quote</a> is accepted, Daedalus automatically
        creates an order from the winning option&apos;s line items. Project links carry
        through so each order item references the correct product in your catalog.
      </p>

      <h3>Channel Imports</h3>
      <p>
        Orders from connected sales channels (Etsy, Shopify, Squarespace) are imported
        automatically. See <a href="/docs/channels">Sales Channels</a> for setup details.
        Imported orders include:
      </p>
      <ul>
        <li>Customer name and shipping address</li>
        <li>Line items with quantities</li>
        <li>Order total and marketplace fees</li>
        <li>External order ID for cross-reference</li>
      </ul>

      <h2>Status Pipeline</h2>
      <p>
        Orders move through a defined status pipeline:
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
            <td><strong>Pending</strong></td>
            <td>Order received, not yet processed</td>
          </tr>
          <tr>
            <td><strong>In Progress</strong></td>
            <td>Order has been processed into tasks</td>
          </tr>
          <tr>
            <td><strong>Printing</strong></td>
            <td>Print jobs are actively running</td>
          </tr>
          <tr>
            <td><strong>Completed</strong></td>
            <td>All items printed and assembled</td>
          </tr>
          <tr>
            <td><strong>Shipped</strong></td>
            <td>Order has been shipped to customer</td>
          </tr>
          <tr>
            <td><strong>Cancelled</strong></td>
            <td>Order was cancelled</td>
          </tr>
        </tbody>
      </table>

      <h2>Processing Orders into Tasks</h2>
      <ol>
        <li>Open an order and click <strong>Process</strong></li>
        <li>Daedalus creates a task for each line item</li>
        <li>Each task generates a checklist based on the project&apos;s parts</li>
        <li>Tasks can then be dispatched as print jobs to your printers</li>
      </ol>
      <p>
        See <a href="/docs/tasks">Tasks &amp; Print Jobs</a> for details on the task
        lifecycle and auto-dispatch system.
      </p>

      <div className="docs-callout">
        <strong>Tip:</strong> Use the status filter to focus on orders that need attention.
        The &ldquo;Pending&rdquo; filter shows orders waiting to be processed into tasks.
      </div>
    </div>
  );
}
