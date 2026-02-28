export const metadata = {
  title: "Materials & Inventory - Daedalus Docs",
  description: "Track filament spools, manage supplies, integrate with Bambu Lab AMS, and set up low-stock alerts.",
};

export default function MaterialsDocs() {
  return (
    <div className="docs-prose">
      <h1>Materials &amp; Inventory</h1>
      <p className="text-lg !text-surface-300">
        Track every spool of filament, manage additional supplies, integrate with Bambu Lab
        AMS, and get alerts before you run out of materials.
      </p>

      <div className="docs-screenshot">
        <img src="/screenshots/materials.png" alt="Daedalus material inventory showing spool tracking with weight and color" />
      </div>

      <h2>Overview</h2>
      <p>
        The Materials page is your inventory hub. It shows all filament spools organized by
        material type, with color swatches, current weight, and cost information. You can
        also track non-filament supplies like screws, magnets, and packaging materials.
      </p>

      <h2>Key Features</h2>

      <h3>Material Catalog</h3>
      <p>
        Define the types of materials you use:
      </p>
      <ul>
        <li><strong>Filament types</strong> &mdash; PLA, PETG, ABS, ASA, TPU, and custom types</li>
        <li><strong>Colors</strong> &mdash; Visual color swatches for easy identification</li>
        <li><strong>Cost per gram</strong> &mdash; Used for automatic cost calculations</li>
        <li><strong>Vendor info</strong> &mdash; Track where you buy each material</li>
      </ul>

      <h3>Spool Tracking</h3>
      <p>
        Track individual spools of filament:
      </p>
      <ul>
        <li>Current weight remaining (grams)</li>
        <li>Original weight when purchased</li>
        <li>Material type and color</li>
        <li>Which printer or AMS tray the spool is loaded in</li>
        <li>Cost per spool and cost per gram</li>
      </ul>

      <h3>Supplies</h3>
      <p>
        Beyond filament, track additional supplies used in production:
      </p>
      <ul>
        <li>Hardware (screws, magnets, inserts)</li>
        <li>Packaging materials (boxes, bags, tissue)</li>
        <li>Consumables (glue, tape, labels)</li>
        <li>Set quantities and unit costs for BOM calculations</li>
      </ul>

      <h3>Bambu Lab AMS Integration</h3>
      <p>
        If you use Bambu Lab printers with AMS units, Daedalus syncs automatically:
      </p>
      <ul>
        <li>Detects loaded spools and tray positions</li>
        <li>Reads RFID spool data for Bambu-branded filament</li>
        <li>Updates spool weight estimates based on print usage</li>
        <li>Maps AMS trays to your inventory for accurate tracking</li>
      </ul>

      <h3>Low-Stock Alerts</h3>
      <p>
        Configure alerts to notify you when materials are running low:
      </p>
      <ul>
        <li><strong>Warning threshold</strong> &mdash; Yellow alert when a spool drops below a set weight</li>
        <li><strong>Critical threshold</strong> &mdash; Red alert when a spool is nearly empty</li>
        <li>Alerts appear on the dashboard and in the notification area</li>
        <li>Dismiss individual alerts or configure auto-dismiss rules</li>
      </ul>

      <h2>Adding Spools</h2>
      <ol>
        <li>Click <strong>Add Spool</strong></li>
        <li>Select the material type and color (or create a new material)</li>
        <li>Enter the spool weight, cost, and vendor</li>
        <li>Optionally assign it to a printer or AMS tray</li>
      </ol>

      <div className="docs-callout">
        <strong>Tip:</strong> When you record an expense for filament via the Expenses page,
        Daedalus can automatically create spool entries from the parsed receipt items.
        See <a href="/docs/expenses">Expenses &amp; OCR</a>.
      </div>
    </div>
  );
}
