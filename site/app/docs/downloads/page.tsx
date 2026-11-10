import { Apple, Monitor, Terminal } from "lucide-react";

export const metadata = {
  title: "Downloads - RankOS Docs",
  description:
    "Download the latest RankOS desktop app for macOS, Windows, or Linux.",
};

const GITHUB_REPO = "https://github.com/cadimartech/rankos";
const LATEST_RELEASE = `${GITHUB_REPO}/releases/latest`;

const platforms = [
  {
    name: "macOS",
    description: "Universal binary (Apple Silicon & Intel)",
    icon: Apple,
    asset: "RankOS-macos-universal.zip",
    requirement: "macOS 11 Big Sur or later",
  },
  {
    name: "Windows",
    description: "64-bit (x86_64)",
    icon: Monitor,
    asset: "RankOS-windows-amd64.zip",
    requirement: "Windows 10 or later",
  },
  {
    name: "Linux",
    description: "64-bit (x86_64)",
    icon: Terminal,
    asset: "RankOS-linux-amd64.tar.gz",
    requirement: "GTK 3 and WebKit2GTK 4.1",
  },
];

export default function DownloadsDocs() {
  return (
    <div className="docs-prose">
      <h1>Downloads</h1>
      <p className="!text-body">
        Download the latest RankOS desktop app for your platform. Each release
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
              className="group flex items-center gap-5 rounded-none border border-border bg-surface px-6 py-5 no-underline shadow-card transition-colors hover:border-primary"
            >
              <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-none border border-border bg-primary/10 text-primary transition-colors group-hover:bg-primary/20">
                <Icon className="h-6 w-6" />
              </div>
              <div className="flex-1">
                <div className="type-heading-xs text-ink">
                  {platform.name}
                </div>
                <div className="type-body-sm text-body">
                  {platform.description}
                </div>
                <div className="type-caption mt-1 text-body">
                  {platform.requirement}
                </div>
              </div>
              <div className="btn btn-primary shrink-0 group-hover:brightness-95">
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
cd rankos

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
