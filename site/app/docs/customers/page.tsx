export const metadata = {
  title: "Customers - Daedalus Docs",
  description: "Manage your customer database, track quotes and orders per customer, and link customers to sales.",
};

export default function CustomersDocs() {
  return (
    <div className="docs-prose">
      <h1>Customers</h1>
      <p className="text-lg !text-surface-300">
        Keep track of everyone you do business with. Customers tie together quotes,
        orders, and sales into a single profile so you can see your full history with
        each buyer.
      </p>

      <h2>Overview</h2>
      <p>
        The Customers page lists all your contacts in a searchable table. Each customer
        record stores a name, email, company, phone number, and freeform notes. Click
        any row to open the customer detail view.
      </p>

      <h2>Creating a Customer</h2>
      <ol>
        <li>Click <strong>New Customer</strong></li>
        <li>Enter a name (required) plus optional email, company, and phone</li>
        <li>Click <strong>Create</strong></li>
      </ol>
      <p>
        You can also create customers implicitly &mdash; the customer dropdown on the
        Quotes and Sales forms lets you pick an existing customer or type a name for a
        one-off sale.
      </p>

      <h2>Customer Detail</h2>
      <p>
        The detail page has two sections:
      </p>

      <h3>Contact Info</h3>
      <p>
        Displays the customer&apos;s email, company, phone, and notes. Click
        <strong> Edit</strong> to update any field.
      </p>

      <h3>Quotes &amp; Orders Tabs</h3>
      <p>
        Two tabs show everything related to this customer:
      </p>
      <ul>
        <li>
          <strong>Quotes</strong> &mdash; Every quote created for this customer, with
          status badges and option counts. Click <strong>New Quote</strong> to create a
          quote pre-linked to this customer.
        </li>
        <li>
          <strong>Orders</strong> &mdash; All orders associated with this customer, with
          current status and source channel.
        </li>
      </ul>

      <h2>Customer Links</h2>
      <p>
        Customers are referenced across several areas of Daedalus:
      </p>
      <ul>
        <li>
          <strong>Quotes</strong> &mdash; Every quote is tied to a customer. See{" "}
          <a href="/docs/quotes">Quotes</a>.
        </li>
        <li>
          <strong>Sales</strong> &mdash; When recording a sale, you can select a customer
          from the dropdown to link the revenue record. See{" "}
          <a href="/docs/sales">Sales &amp; Analytics</a>.
        </li>
        <li>
          <strong>Orders</strong> &mdash; Orders created from accepted quotes inherit the
          customer link.
        </li>
      </ul>

      <h2>Deleting a Customer</h2>
      <p>
        Click <strong>Delete</strong> on the customer detail page. This is permanent and
        cannot be undone. Linked quotes and sales will have their customer reference
        cleared (set to null) rather than being deleted.
      </p>

      <div className="docs-callout">
        <strong>Tip:</strong> Use the search bar on the Customers list to filter by name,
        email, or company. The search is applied server-side so it works well even with
        large customer lists.
      </div>
    </div>
  );
}
