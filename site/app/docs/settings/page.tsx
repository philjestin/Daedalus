export const metadata = {
  title: "Settings - Daedalus Docs",
  description: "Configure integrations, API keys, backup management, and application preferences in Daedalus.",
};

export default function SettingsDocs() {
  return (
    <div className="docs-prose">
      <h1>Settings</h1>
      <p className="text-lg !text-surface-300">
        Configure integrations, manage API keys, set up automated backups, and customize
        your Daedalus instance.
      </p>

      <h2>Overview</h2>
      <p>
        The Settings page is where you connect external services, configure system behavior,
        and manage your data. It&apos;s organized into sections for integrations, backup
        management, and general preferences.
      </p>

      <h2>Integrations</h2>

      <h3>Printer Connections</h3>
      <p>
        Configure default connection settings for each printer protocol:
      </p>
      <ul>
        <li><strong>Bambu Lab</strong> &mdash; Default MQTT broker, cloud account credentials</li>
        <li><strong>OctoPrint</strong> &mdash; Default API key format and connection timeout</li>
        <li><strong>Klipper</strong> &mdash; Default Moonraker API URL pattern</li>
      </ul>

      <h3>Sales Channels</h3>
      <p>
        Manage OAuth connections and API keys for each marketplace:
      </p>
      <ul>
        <li><strong>Etsy</strong> &mdash; OAuth status, token refresh, webhook URL</li>
        <li><strong>Shopify</strong> &mdash; Store URL, OAuth status, sync interval</li>
        <li><strong>Squarespace</strong> &mdash; API key, sync interval</li>
      </ul>
      <p>
        See <a href="/docs/channels">Sales Channels</a> for detailed setup instructions.
      </p>

      <h3>OCR Configuration</h3>
      <p>
        Configure the OCR engine used for receipt parsing:
      </p>
      <ul>
        <li>API key for the OCR service</li>
        <li>Default expense category for new receipts</li>
        <li>Auto-spool creation toggle for filament expenses</li>
      </ul>

      <h2>API Keys</h2>
      <p>
        Manage API keys for external service integrations. Each key shows:
      </p>
      <ul>
        <li>Service name and connection status</li>
        <li>Last used timestamp</li>
        <li>Options to rotate or revoke keys</li>
      </ul>

      <h2>Backup Management</h2>
      <p>
        Daedalus stores all data in a local SQLite database. The backup system protects
        your data with scheduled and manual backups:
      </p>

      <h3>Automated Backups</h3>
      <ul>
        <li><strong>Schedule</strong> &mdash; Daily or weekly automatic backups</li>
        <li><strong>Retention</strong> &mdash; Configure how many backups to keep</li>
        <li><strong>Startup backup</strong> &mdash; Optionally create a backup every time the app starts</li>
      </ul>

      <h3>Manual Backups</h3>
      <ol>
        <li>Click <strong>Create Backup</strong> to generate an immediate backup</li>
        <li>Backups are stored in the configured backup directory</li>
        <li>Each backup includes the full database and any uploaded files</li>
      </ol>

      <h3>Restoring from Backup</h3>
      <ol>
        <li>Go to <strong>Settings &gt; Backups</strong></li>
        <li>Select a backup from the list</li>
        <li>Click <strong>Restore</strong> and confirm</li>
        <li>Daedalus restarts with the restored data</li>
      </ol>

      <h2>General Preferences</h2>
      <ul>
        <li><strong>Auto-dispatch</strong> &mdash; Enable or disable automatic print job dispatch</li>
        <li><strong>Currency</strong> &mdash; Set the display currency for financial data</li>
        <li><strong>Hourly rate</strong> &mdash; Used to calculate profit-per-hour metrics</li>
        <li><strong>Alert thresholds</strong> &mdash; Default warning and critical levels for material alerts</li>
      </ul>

      <div className="docs-callout">
        <strong>Tip:</strong> Enable startup backups as a safety net. They add minimal startup
        time and ensure you always have a recent backup available.
      </div>
    </div>
  );
}
