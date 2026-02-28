export const metadata = {
  title: "Sales Channels - Daedalus Docs",
  description: "Connect Etsy, Shopify, and Squarespace to Daedalus for automatic order import and product linking.",
};

export default function ChannelsDocs() {
  return (
    <div className="docs-prose">
      <h1>Sales Channels</h1>
      <p className="text-lg !text-surface-300">
        Connect your online stores to automatically import orders, sync inventory,
        and link marketplace listings to your Daedalus product catalog.
      </p>

      <div className="docs-screenshot">
        <img src="/screenshots/channels.png" alt="Daedalus sales channels showing Etsy, Shopify, and Squarespace integrations" />
      </div>

      <h2>Overview</h2>
      <p>
        Sales channels connect Daedalus to the marketplaces where you sell. Once connected,
        orders are imported automatically, products can be linked between systems, and
        you can manage fulfillment from a single interface.
      </p>

      <h2>Supported Channels</h2>

      <h3>Etsy</h3>
      <p>
        Full OAuth 2.0 integration with PKCE for secure authentication:
      </p>
      <ol>
        <li>Go to <strong>Settings &gt; Integrations &gt; Etsy</strong></li>
        <li>Click <strong>Connect Etsy</strong> to start the OAuth flow</li>
        <li>Authorize Daedalus in your Etsy account</li>
        <li>Orders are synced automatically via webhooks</li>
      </ol>
      <p>Etsy integration features:</p>
      <ul>
        <li>Real-time order import via webhooks</li>
        <li>Listing sync to link Etsy products to Daedalus projects</li>
        <li>Transaction fee tracking for accurate profitability</li>
        <li>Automatic token refresh for uninterrupted access</li>
      </ul>

      <h3>Shopify</h3>
      <p>
        OAuth integration with order and inventory sync:
      </p>
      <ol>
        <li>Go to <strong>Settings &gt; Integrations &gt; Shopify</strong></li>
        <li>Enter your Shopify store URL</li>
        <li>Click <strong>Connect Shopify</strong> and authorize the app</li>
        <li>Orders begin syncing immediately</li>
      </ol>
      <p>Shopify integration features:</p>
      <ul>
        <li>Automatic order import with customer details</li>
        <li>Product linking between Shopify and Daedalus</li>
        <li>Inventory level sync</li>
        <li>Fulfillment status updates back to Shopify</li>
      </ul>

      <h3>Squarespace</h3>
      <p>
        API key-based integration for storefront orders:
      </p>
      <ol>
        <li>Go to <strong>Settings &gt; Integrations &gt; Squarespace</strong></li>
        <li>Generate an API key in your Squarespace admin panel</li>
        <li>Paste the API key into Daedalus and click <strong>Connect</strong></li>
        <li>Orders are polled on a configurable interval</li>
      </ol>
      <p>Squarespace integration features:</p>
      <ul>
        <li>Order import with line items and customer info</li>
        <li>Product linking to Daedalus projects</li>
        <li>Configurable sync interval</li>
      </ul>

      <h2>Linking Products</h2>
      <p>
        After connecting a channel, link your marketplace listings to Daedalus projects:
      </p>
      <ol>
        <li>Open a project and go to the <strong>Channels</strong> tab</li>
        <li>Click <strong>Link Product</strong> and select the matching listing from each channel</li>
        <li>When an order comes in for that listing, Daedalus knows which project to fulfill</li>
      </ol>

      <h2>Processing Imported Orders</h2>
      <p>
        Imported orders follow the same workflow as manual orders:
      </p>
      <ol>
        <li>New orders appear on the Orders page with the channel icon</li>
        <li>If products are linked, items are automatically matched to projects</li>
        <li>Process the order into tasks and dispatch print jobs as usual</li>
        <li>Mark as shipped to update the fulfillment status</li>
      </ol>

      <div className="docs-callout">
        <strong>Tip:</strong> Link your products before your first order sync. That way,
        incoming orders are automatically matched to the right projects and ready to process
        immediately.
      </div>
    </div>
  );
}
