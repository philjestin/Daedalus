export const metadata = {
  title: "Deployment - Daedalus Docs",
  description: "Deploy Daedalus as a desktop app or Docker container with environment variables.",
};

export default function DeploymentDocs() {
  return (
    <div className="docs-prose">
      <h1>Deployment</h1>
      <p className="text-lg !text-surface-300">
        Run Daedalus as a native desktop app or a Docker container on your own server.
        Your data, your infrastructure.
      </p>

      <h2>Overview</h2>
      <p>
        Daedalus is designed to run wherever works best for your setup. The desktop app
        is ideal for a single workstation, and Docker works great for a home server or NAS.
      </p>

      <h2>Desktop App (Wails)</h2>
      <p>
        The desktop app runs as a native application on macOS and Windows, powered by
        the Wails framework.
      </p>

      <h3>Building from Source</h3>
      <pre><code>{`# Install Wails CLI
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Clone the repository
git clone https://github.com/philjestin/Daedalus.git
cd daedalus

# Run in development mode
wails dev

# Build for production
wails build`}</code></pre>

      <h3>Desktop Features</h3>
      <ul>
        <li>Full filesystem access for 3MF file parsing</li>
        <li>System tray icon with quick status</li>
        <li>Native window management and menus</li>
        <li>Automatic updates (when enabled)</li>
      </ul>

      <h2>Docker</h2>
      <p>
        Run Daedalus as a containerized web application with Docker:
      </p>

      <h3>Docker Compose</h3>
      <pre><code>{`version: "3.8"

services:
  daedalus:
    image: daedalus:latest
    build: .
    ports:
      - "8080:8080"
    volumes:
      - daedalus-data:/app/data
    environment:
      - DAEDALUS_PORT=8080
      - DAEDALUS_DATA_DIR=/app/data

volumes:
  daedalus-data:`}</code></pre>

      <h3>Running with Docker</h3>
      <pre><code>{`# Build the image
docker build -t daedalus .

# Run with a named volume for data persistence
docker run -d \\
  --name daedalus \\
  -p 8080:8080 \\
  -v daedalus-data:/app/data \\
  daedalus

# Or use Docker Compose
docker compose up -d`}</code></pre>

      <h3>Docker Features</h3>
      <ul>
        <li>Multi-stage build for minimal image size</li>
        <li>Embedded SQLite database (no external DB required)</li>
        <li>Volume mount for persistent data</li>
        <li>Health check endpoint at <code>/health</code></li>
      </ul>

      <h2>Environment Variables</h2>
      <table>
        <thead>
          <tr>
            <th>Variable</th>
            <th>Default</th>
            <th>Description</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td><code>DAEDALUS_PORT</code></td>
            <td><code>8080</code></td>
            <td>HTTP server port</td>
          </tr>
          <tr>
            <td><code>DAEDALUS_DATA_DIR</code></td>
            <td><code>./data</code></td>
            <td>Directory for database and uploads</td>
          </tr>
          <tr>
            <td><code>DAEDALUS_BACKUP_DIR</code></td>
            <td><code>./backups</code></td>
            <td>Directory for automatic backups</td>
          </tr>
          <tr>
            <td><code>DAEDALUS_LOG_LEVEL</code></td>
            <td><code>info</code></td>
            <td>Logging level (debug, info, warn, error)</td>
          </tr>
          <tr>
            <td><code>DAEDALUS_CORS_ORIGINS</code></td>
            <td><code>*</code></td>
            <td>Allowed CORS origins (comma-separated)</td>
          </tr>
          <tr>
            <td><code>ETSY_API_KEY</code></td>
            <td>&mdash;</td>
            <td>Etsy OAuth application key</td>
          </tr>
          <tr>
            <td><code>SHOPIFY_API_KEY</code></td>
            <td>&mdash;</td>
            <td>Shopify OAuth application key</td>
          </tr>
          <tr>
            <td><code>SHOPIFY_API_SECRET</code></td>
            <td>&mdash;</td>
            <td>Shopify OAuth secret</td>
          </tr>
          <tr>
            <td><code>SQUARESPACE_API_KEY</code></td>
            <td>&mdash;</td>
            <td>Squarespace API key</td>
          </tr>
          <tr>
            <td><code>OCR_API_KEY</code></td>
            <td>&mdash;</td>
            <td>OCR service API key for receipt parsing</td>
          </tr>
        </tbody>
      </table>

      <div className="docs-callout">
        <strong>Tip:</strong> For local development with LAN-connected printers, the desktop
        app or Docker on the same network gives the best experience.
      </div>
    </div>
  );
}
