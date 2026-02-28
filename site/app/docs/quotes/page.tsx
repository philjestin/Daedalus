export const metadata = {
  title: "Quotes - Daedalus Docs",
  description: "Create itemized quotes for custom jobs, link line items to projects, and convert accepted quotes into orders.",
};

export default function QuotesDocs() {
  return (
    <div className="docs-prose">
      <h1>Quotes</h1>
      <p className="text-lg !text-surface-300">
        Build detailed, multi-option quotes for custom jobs. Link line items to your
        product catalog, then convert accepted quotes directly into orders.
      </p>

      <h2>Overview</h2>
      <p>
        The Quotes page lists all quotes with their number, title, customer, status,
        total, and date. Filter by status to focus on drafts that need finishing or sent
        quotes awaiting a response.
      </p>

      <h2>Creating a Quote</h2>
      <ol>
        <li>Click <strong>New Quote</strong> (from the Quotes list or a Customer detail page)</li>
        <li>Select a customer and enter a title</li>
        <li>Click <strong>Create Quote</strong> to open the quote editor</li>
      </ol>

      <h2>Quote Structure</h2>
      <p>
        Quotes are organized into <strong>options</strong>, each containing one or more
        <strong> line items</strong>. This lets you present multiple pricing tiers or
        configurations to the customer.
      </p>

      <h3>Options</h3>
      <p>
        An option is a named grouping (e.g. &ldquo;Standard&rdquo; or &ldquo;Premium&rdquo;).
        Each option has its own subtotal calculated from its line items.
      </p>

      <h3>Line Items</h3>
      <p>
        Each line item has:
      </p>
      <ul>
        <li><strong>Type</strong> &mdash; Printing, design, finishing, hardware, shipping, or other</li>
        <li><strong>Description</strong> &mdash; What the item is</li>
        <li><strong>Quantity</strong> and <strong>unit</strong> &mdash; e.g. 4 pcs, 2.5 hours</li>
        <li><strong>Unit price</strong> &mdash; Price per unit in dollars</li>
        <li><strong>Project link</strong> (optional) &mdash; Link to a product in your catalog</li>
      </ul>

      <h3>Linking Line Items to Projects</h3>
      <p>
        When adding items, you can optionally select a project from the dropdown. This:
      </p>
      <ul>
        <li>Auto-fills the description with the project name</li>
        <li>Pre-fills the unit price from the project&apos;s configured price</li>
        <li>Carries the project link through to orders when the quote is accepted</li>
      </ul>

      <h2>Adding Options with Items</h2>
      <p>
        The <strong>Add Option</strong> modal lets you create an option and its line items
        in a single step:
      </p>
      <ol>
        <li>Click <strong>Add Option</strong> on the quote detail page</li>
        <li>Enter an option name and optional description</li>
        <li>Add one or more line items with type, description, quantity, unit, and rate</li>
        <li>Optionally link each item to a project</li>
        <li>Click <strong>Create Option</strong> to save everything at once</li>
      </ol>
      <p>
        You can also add more items to an existing option using the inline form on each
        option card.
      </p>

      <h2>Quote Workflow</h2>
      <table>
        <thead>
          <tr>
            <th>Status</th>
            <th>Description</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td><strong>Draft</strong></td>
            <td>Quote is being built. Editable. Can be deleted.</td>
          </tr>
          <tr>
            <td><strong>Sent</strong></td>
            <td>Marked as sent to the customer. Still editable.</td>
          </tr>
          <tr>
            <td><strong>Accepted</strong></td>
            <td>Customer approved. An order is created from the selected option.</td>
          </tr>
          <tr>
            <td><strong>Rejected</strong></td>
            <td>Customer declined the quote.</td>
          </tr>
          <tr>
            <td><strong>Expired</strong></td>
            <td>Quote passed its expiration date without a response.</td>
          </tr>
        </tbody>
      </table>

      <h3>Marking as Sent</h3>
      <p>
        Click <strong>Mark as Sent</strong> to move the quote from Draft to Sent status.
        This is a workflow gate &mdash; it records that the customer has received the quote.
      </p>

      <h3>Accepting a Quote</h3>
      <p>
        When a customer approves, click <strong>Accept</strong> on the winning option.
        Daedalus will:
      </p>
      <ol>
        <li>Create a new order linked to the same customer</li>
        <li>Convert printing-type line items and project-linked items into order line items</li>
        <li>Carry project links through so the order references the correct products</li>
        <li>Set the quote status to <strong>Accepted</strong></li>
      </ol>
      <p>
        The new order appears in <a href="/docs/orders">Orders</a> ready to be processed
        into tasks and print jobs.
      </p>

      <h2>Deleting Quotes</h2>
      <p>
        Draft quotes can be deleted from the Quotes list using the trash icon. Quotes in
        other statuses cannot be deleted to preserve history.
      </p>

      <div className="docs-callout">
        <strong>Tip:</strong> Use multiple options to present good/better/best pricing.
        Each option calculates its own total, making it easy for customers to compare.
      </div>
    </div>
  );
}
