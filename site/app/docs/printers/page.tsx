export const metadata = {
  title: "Printers - Daedalus Docs",
  description: "Add and manage printers in Daedalus: Bambu Lab, OctoPrint, Klipper, manual printers, and network scanning.",
};

export default function PrintersDocs() {
  return (
    <div className="docs-prose">
      <h1>Printers</h1>
      <p className="text-lg !text-surface-300">
        Connect your entire fleet regardless of manufacturer. Daedalus speaks every protocol
        so you can manage Bambu Lab, OctoPrint, Klipper, and manual printers from one place.
      </p>

      <div className="docs-screenshot">
        <img src="/screenshots/printers.png" alt="Daedalus printer fleet showing Bambu Lab printers with status and temperature readings" />
      </div>

      <h2>Overview</h2>
      <p>
        The Printers page shows every printer in your fleet with real-time status, temperatures,
        and active job progress. Each printer card displays its connection type, current state,
        and quick actions.
      </p>

      <h2>Adding Printers</h2>

      <h3>Bambu Lab (MQTT)</h3>
      <ol>
        <li>Click <strong>Add Printer</strong> and select <strong>Bambu Lab</strong></li>
        <li>Enter the printer&apos;s IP address, serial number, and access code</li>
        <li>Daedalus connects via MQTT for real-time status, temperatures, and HMS error tracking</li>
        <li>AMS spool and tray data is synced automatically</li>
      </ol>

      <h3>Network Scan</h3>
      <ol>
        <li>Click <strong>Scan Network</strong> to discover Bambu Lab printers on your local network</li>
        <li>Daedalus uses SSDP/mDNS to find printers automatically</li>
        <li>Select a discovered printer and enter its access code to connect</li>
      </ol>

      <h3>Bambu Cloud Pairing</h3>
      <ol>
        <li>Click <strong>Cloud Pairing</strong> and sign in with your Bambu Lab account</li>
        <li>Select printers from your cloud account to pair with Daedalus</li>
        <li>Cloud-paired printers use the Bambu Cloud MQTT broker for remote access</li>
      </ol>

      <h3>OctoPrint</h3>
      <ol>
        <li>Click <strong>Add Printer</strong> and select <strong>OctoPrint</strong></li>
        <li>Enter the OctoPrint server URL and API key</li>
        <li>Daedalus communicates via the OctoPrint REST API for status, file upload, and job control</li>
      </ol>

      <h3>Klipper / Moonraker</h3>
      <ol>
        <li>Click <strong>Add Printer</strong> and select <strong>Klipper</strong></li>
        <li>Enter the Moonraker API URL</li>
        <li>Daedalus uses the Moonraker REST API for printer control and status</li>
      </ol>

      <h3>Manual Printers</h3>
      <ol>
        <li>Click <strong>Add Printer</strong> and select <strong>Manual</strong></li>
        <li>Enter a name and model for the printer</li>
        <li>Manual printers don&apos;t require a network connection &mdash; they&apos;re used for logging jobs without direct printer control</li>
      </ol>

      <h2>Printer Detail</h2>
      <p>
        Click any printer to open its detail view. Here you can see:
      </p>
      <ul>
        <li>Real-time temperature graphs (nozzle, bed, chamber)</li>
        <li>Active print job progress with time remaining</li>
        <li>HMS error codes and warnings (Bambu Lab)</li>
        <li>AMS tray status and loaded filaments (Bambu Lab)</li>
        <li>Job history with completion stats</li>
        <li>Printer configuration and connection settings</li>
      </ul>

      <h2>Printer Analytics</h2>
      <p>
        Each printer tracks utilization metrics over time:
      </p>
      <ul>
        <li><strong>Print hours</strong> &mdash; Total time spent printing</li>
        <li><strong>Job count</strong> &mdash; Number of completed and failed jobs</li>
        <li><strong>Success rate</strong> &mdash; Percentage of jobs completed without failure</li>
        <li><strong>Material usage</strong> &mdash; Total filament consumed in grams</li>
      </ul>

      <div className="docs-callout">
        <strong>Tip:</strong> Bambu Lab printers provide the richest data via MQTT, including
        real-time layer progress, AMS spool weights, and HMS health alerts. If possible,
        connect via LAN for the lowest latency.
      </div>
    </div>
  );
}
