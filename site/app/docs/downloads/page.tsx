import { Apple, Monitor, Terminal } from "lucide-react";

export const metadata = {
  title: "Downloads - Daedalus Docs",
  description:
    "Download the latest Daedalus desktop app for macOS, Windows, or Linux.",
};

const GITHUB_REPO = "https://github.com/philjestin/Daedalus";
const LATEST_RELEASE = `${GITHUB_REPO}/releases/latest`;

const platforms = [
  {
    name: "macOS",
    description: "Universal binary (Apple Silicon & Intel)",
    icon: Apple,
    asset: "Daedalus-macos-universal.zip",
    requirement: "macOS 11 Big Sur or later",
  },
  {
    name: "Windows",
    description: "64-bit (x86_64)",
    icon: Monitor,
    asset: "Daedalus-windows-amd64.zip",
    requirement: "Windows 10 or later",
  },
  {
    name: "Linux",
    description: "64-bit (x86_64)",
    icon: Terminal,
    asset: "Daedalus-linux-amd64.tar.gz",
    requirement: "GTK 3 and WebKit2GTK 4.1",
  },
];

export default function DownloadsDocs() {
  return (
    <div className="docs-prose">
      <h1>Downloads</h1>
      <p className="text-lg !text-surface-300">
        Download the latest Daedalus desktop app for your platform. Each release
        includes the full application with an embedded database &mdash; no
        external dependencies required.
      </p>

      <div className="not-prose mt-8 grid gap-4">
        {platforms.map((platform) => {
          const Icon = platform.icon;
          return (
            <a
              key={platform.name}
              href={`${LATEST_RELEASE}/download/${platform.asset}`}
              className="group flex items-center gap-5 rounded-xl border border-surface-800 bg-surface-900 px-6 py-5 transition-all hover:border-accent-500/40 hover:bg-surface-800/60"
            >
              <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-lg bg-accent-500/10 text-accent-500 transition-colors group-hover:bg-accent-500/20">
                <Icon className="h-6 w-6" />
              </div>
              <div className="flex-1">
                <div className="text-base font-semibold text-surface-50">
                  {platform.name}
                </div>
                <div className="text-sm text-surface-400">
                  {platform.description}
                </div>
                <div className="mt-1 text-xs text-surface-500">
                  {platform.requirement}
                </div>
              </div>
              <div className="shrink-0 rounded-lg bg-accent-500/10 px-4 py-2 text-sm font-medium text-accent-400 transition-colors group-hover:bg-accent-500 group-hover:text-white">
                Download
              </div>
            </a>
          );
        })}
      </div>

      <h2>All Releases</h2>
      <p>
        Browse all versions, changelogs, and pre-release builds on the{" "}
        <a href={`${GITHUB_REPO}/releases`}>GitHub Releases</a> page.
      </p>

      <h2>Building from Source</h2>
      <p>
        If you prefer to build from source, or need a platform not listed above:
      </p>
      <pre>
        <code>{`# Install Go 1.24+ and Node.js 20+
go install github.com/wailsapp/wails/v2/cmd/wails@latest

git clone ${GITHUB_REPO}.git
cd Daedalus

# Development mode
wails dev

# Production build
wails build`}</code>
      </pre>

      <div className="docs-callout">
        <strong>Tip:</strong> See the{" "}
        <a href="/docs/deployment">Deployment</a> guide for Docker setup and
        environment configuration.
      </div>
    </div>
  );
}
