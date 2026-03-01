export const metadata = {
  title: "Sales & Analytics - Daedalus Docs",
  description: "Record sales, track revenue trends, analyze per-project profitability, and view weekly insights.",
};

export default function SalesDocs() {
  return (
    <div className="docs-prose">
      <h1>Sales &amp; Analytics</h1>
      <p className="text-lg !text-surface-300">
        Understand where your money comes from and where it goes. Built-in analytics show
        revenue trends, per-project profitability, and channel breakdowns.
      </p>

      <div className="docs-screenshot">
        <img src="/screenshots/sales.png" alt="Daedalus sales analytics showing revenue by project, profit breakdown, and weekly insights" />
      </div>

      <h2>Overview</h2>
      <p>
        The Sales page combines recorded sales data with expense tracking to give you a
        complete financial picture. View revenue over time, drill down by project or channel,
        and identify your most profitable products.
      </p>

      <h2>Key Features</h2>

      <h3>Recording Sales</h3>
      <p>
        Sales are recorded automatically when orders from connected channels are marked
        as shipped. You can also record manual sales:
      </p>
      <ol>
        <li>Click <strong>Record Sale</strong></li>
        <li>Optionally link to a project and/or customer</li>
        <li>Enter the item description, quantity, and sale price</li>
        <li>Choose the sales channel (or &ldquo;Direct&rdquo; for non-marketplace sales)</li>
        <li>The sale is added to your analytics immediately</li>
      </ol>

      <h3>Customer Linking</h3>
      <p>
        When recording a sale, you can select a customer from the dropdown to associate
        the revenue with a specific buyer. Selecting a customer auto-fills the customer
        name. For one-off sales without a customer record, leave the dropdown empty and
        type a name in the text field instead. See{" "}
        <a href="/docs/customers">Customers</a> for more on managing customer records.
      </p>

      <h3>Weekly Insights</h3>
      <p>
        Daedalus generates weekly insight summaries showing:
      </p>
      <ul>
        <li>Total revenue and comparison to previous week</li>
        <li>Top-selling products by quantity and revenue</li>
        <li>Channel performance comparison</li>
        <li>Notable trends or anomalies</li>
      </ul>

      <h3>Project Profitability</h3>
      <p>
        For each project, Daedalus calculates:
      </p>
      <ul>
        <li><strong>Revenue</strong> &mdash; Total sales for this project</li>
        <li><strong>Material cost</strong> &mdash; Filament and supplies used per unit</li>
        <li><strong>Print time cost</strong> &mdash; Based on configured hourly rate</li>
        <li><strong>Marketplace fees</strong> &mdash; Transaction and listing fees</li>
        <li><strong>Profit per unit</strong> &mdash; Revenue minus all costs</li>
        <li><strong>Profit per hour</strong> &mdash; Helps optimize your product mix</li>
      </ul>

      <h3>Channel Breakdown</h3>
      <p>
        See how each sales channel performs:
      </p>
      <ul>
        <li>Revenue by channel (Etsy, Shopify, Squarespace, Direct)</li>
        <li>Order volume and average order value per channel</li>
        <li>Fee comparison across marketplaces</li>
        <li>Trends over time for each channel</li>
      </ul>

      <h3>Revenue Charts</h3>
      <p>
        Time-series charts with configurable date ranges:
      </p>
      <ul>
        <li>Daily, weekly, or monthly aggregation</li>
        <li>Revenue and profit overlay</li>
        <li>Filter by project or channel</li>
        <li>Hover for exact values on any data point</li>
      </ul>

      <div className="docs-callout">
        <strong>Tip:</strong> Use the &ldquo;Profit per hour&rdquo; metric to decide what
        to print next. A product with lower revenue but faster print time may actually
        be more profitable per hour of printer usage.
      </div>
    </div>
  );
}
