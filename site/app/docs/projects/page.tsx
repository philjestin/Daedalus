export const metadata = {
  title: "Projects - Daedalus Docs",
  description: "Manage your product catalog with multi-part designs, 3MF parsing, versioning, and bill of materials.",
};

export default function ProjectsDocs() {
  return (
    <div className="docs-prose">
      <h1>Projects</h1>
      <p className="text-lg !text-surface-300">
        Build a product catalog with multi-part designs, automatic 3MF parsing, immutable
        design versioning, and full bill of materials tracking.
      </p>

      <div className="docs-screenshot">
        <img src="/screenshots/projects.png" alt="Daedalus project catalog showing maker products with descriptions and dates" />
      </div>

      <h2>Overview</h2>
      <p>
        Projects represent the products you make and sell. Each project can contain multiple
        parts, each with its own design files, material requirements, and print settings.
        When you create an order, you select which projects to fulfill.
      </p>

      <h2>Key Features</h2>

      <h3>Product Catalog</h3>
      <ul>
        <li>Create projects with a name, description, images, and pricing</li>
        <li>Tag-based filtering and search across your catalog</li>
        <li>Track per-product profitability with unit cost analysis</li>
      </ul>

      <h3>Parts</h3>
      <p>
        Each project contains one or more parts. A part represents a single 3D-printed
        component of the final product.
      </p>
      <ul>
        <li>Define material type, color, and weight requirements per part</li>
        <li>Set printer constraints (e.g., only certain printers can print this part)</li>
        <li>Parts are used to generate task checklists when fulfilling orders</li>
      </ul>

      <h3>Design Versions</h3>
      <p>
        Every part has an immutable version history. When you update a design file, a new
        version is created rather than overwriting the old one.
      </p>
      <ul>
        <li>Each version records the design file, print settings, and timestamp</li>
        <li>Print jobs reference a specific version so you always know what was printed</li>
        <li>Roll back to any previous version at any time</li>
      </ul>

      <h3>3MF File Parsing</h3>
      <p>
        Upload a sliced 3MF file and Daedalus automatically extracts:
      </p>
      <ul>
        <li><strong>Print time</strong> &mdash; Estimated duration from slicer</li>
        <li><strong>Weight</strong> &mdash; Filament weight in grams</li>
        <li><strong>Filament type</strong> &mdash; PLA, PETG, ABS, etc.</li>
        <li><strong>Nozzle size</strong> &mdash; Required nozzle diameter</li>
        <li><strong>Printer model</strong> &mdash; Target printer from slicer profile</li>
      </ul>

      <h3>Bill of Materials (BOM)</h3>
      <p>
        Define materials and supplies needed for each product:
      </p>
      <ul>
        <li>Filament requirements per part (type, color, weight)</li>
        <li>Additional supplies (screws, magnets, packaging, etc.)</li>
        <li>Automatic cost rollups based on current material prices</li>
        <li>Material costs are snapshot at print time for accurate historical tracking</li>
      </ul>

      <h2>Creating a Project</h2>
      <ol>
        <li>Click <strong>New Project</strong> and fill in the name, description, and pricing</li>
        <li>Add parts to the project &mdash; each part represents a printed component</li>
        <li>For each part, upload a 3MF file or manually enter print settings</li>
        <li>Define material requirements and any additional supplies</li>
        <li>Tag the project for easy filtering (e.g., &ldquo;tabletop&rdquo;, &ldquo;functional&rdquo;, &ldquo;gift&rdquo;)</li>
      </ol>

      <div className="docs-callout">
        <strong>Tip:</strong> Upload 3MF files whenever possible &mdash; the automatic metadata
        extraction saves time and reduces errors compared to manually entering print settings.
      </div>
    </div>
  );
}
