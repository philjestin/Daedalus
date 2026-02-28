export const metadata = {
  title: "Expenses & OCR - Daedalus Docs",
  description: "Upload receipts, parse with OCR, review extracted data, and auto-create spool entries in Daedalus.",
};

export default function ExpensesDocs() {
  return (
    <div className="docs-prose">
      <h1>Expenses &amp; OCR</h1>
      <p className="text-lg !text-surface-300">
        Upload photos of receipts and let OCR extract the details automatically. Categorize
        expenses, review parsed data, and optionally create spool entries from filament purchases.
      </p>

      <div className="docs-screenshot">
        <img src="/screenshots/expenses.png" alt="Daedalus expense tracking showing OCR-parsed receipts from Amazon, Bambu Lab, and Home Depot" />
      </div>

      <h2>Overview</h2>
      <p>
        The Expenses page tracks all business spending across 8 categories. The standout
        feature is OCR receipt parsing &mdash; snap a photo of any receipt and Daedalus
        extracts vendor, amount, date, and individual line items.
      </p>

      <h2>Receipt Upload &amp; OCR</h2>
      <ol>
        <li>Click <strong>Upload Receipt</strong> and select an image file (JPG, PNG, or PDF)</li>
        <li>Daedalus runs OCR processing to extract receipt data</li>
        <li>Review the parsed results: vendor name, total amount, date, and line items</li>
        <li>Each extracted field shows a confidence score so you know when to double-check</li>
        <li>Edit any fields that need correction, then confirm to save</li>
      </ol>

      <h3>Retry OCR</h3>
      <p>
        If the initial OCR results are poor (blurry image, unusual format), you can retry
        the processing. Click <strong>Retry OCR</strong> to re-run extraction on the same image.
      </p>

      <h2>Expense Categories</h2>
      <table>
        <thead>
          <tr>
            <th>Category</th>
            <th>Examples</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td><strong>Filament</strong></td>
            <td>PLA, PETG, ABS spools</td>
          </tr>
          <tr>
            <td><strong>Parts</strong></td>
            <td>Screws, magnets, inserts, hardware</td>
          </tr>
          <tr>
            <td><strong>Tools</strong></td>
            <td>Nozzles, build plates, scrapers</td>
          </tr>
          <tr>
            <td><strong>Shipping</strong></td>
            <td>Postage, packaging materials, labels</td>
          </tr>
          <tr>
            <td><strong>Fees</strong></td>
            <td>Marketplace transaction fees, payment processing</td>
          </tr>
          <tr>
            <td><strong>Subscriptions</strong></td>
            <td>Software licenses, cloud services</td>
          </tr>
          <tr>
            <td><strong>Advertising</strong></td>
            <td>Etsy ads, social media promotion</td>
          </tr>
          <tr>
            <td><strong>Other</strong></td>
            <td>Anything that doesn&apos;t fit the above</td>
          </tr>
        </tbody>
      </table>

      <h2>Auto-Spool Creation</h2>
      <p>
        When you categorize an expense as <strong>Filament</strong>, Daedalus can
        automatically create spool entries in your material inventory:
      </p>
      <ol>
        <li>After confirming the OCR results, Daedalus detects filament line items</li>
        <li>For each filament item, it suggests a new spool entry with the parsed details</li>
        <li>Review and confirm to add the spools to your inventory</li>
        <li>The expense and spool entries are linked for accurate cost tracking</li>
      </ol>

      <h2>Manual Expense Entry</h2>
      <p>
        You can also add expenses manually without a receipt:
      </p>
      <ol>
        <li>Click <strong>Add Expense</strong></li>
        <li>Enter the vendor, amount, date, and category</li>
        <li>Add optional notes or line item details</li>
      </ol>

      <div className="docs-callout">
        <strong>Tip:</strong> For best OCR results, photograph receipts on a flat surface with
        good lighting. The entire receipt should be visible and in focus.
      </div>
    </div>
  );
}
