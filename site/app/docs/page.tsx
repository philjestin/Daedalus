import Link from "next/link";
import {
  Gauge,
  Printer,
  FolderKanban,
  ShoppingBag,
  ListChecks,
  Package,
  Receipt,
  TrendingUp,
  ShoppingCart,
  CalendarRange,
  Settings,
  Cloud,
  ArrowRight,
} from "lucide-react";

export const metadata = {
  title: "Getting Started - Daedalus Docs",
  description: "Get up and running with Daedalus, the all-in-one 3D print farm management platform.",
};

const sections = [
  { href: "/docs/dashboard", label: "Dashboard", icon: Gauge, desc: "Financial overview and fleet status" },
  { href: "/docs/printers", label: "Printers", icon: Printer, desc: "Add and manage your printer fleet" },
  { href: "/docs/projects", label: "Projects", icon: FolderKanban, desc: "Product catalog and design versions" },
  { href: "/docs/orders", label: "Orders", icon: ShoppingBag, desc: "Unified order management" },
  { href: "/docs/tasks", label: "Tasks & Print Jobs", icon: ListChecks, desc: "Task lifecycle and auto-dispatch" },
  { href: "/docs/materials", label: "Materials & Inventory", icon: Package, desc: "Spool tracking and AMS integration" },
  { href: "/docs/expenses", label: "Expenses & OCR", icon: Receipt, desc: "Receipt upload and expense tracking" },
  { href: "/docs/sales", label: "Sales & Analytics", icon: TrendingUp, desc: "Revenue insights and profitability" },
  { href: "/docs/channels", label: "Sales Channels", icon: ShoppingCart, desc: "Etsy, Shopify, and Squarespace" },
  { href: "/docs/timeline", label: "Timeline", icon: CalendarRange, desc: "Gantt-style production timeline" },
  { href: "/docs/settings", label: "Settings", icon: Settings, desc: "Integrations and backup management" },
  { href: "/docs/deployment", label: "Deployment", icon: Cloud, desc: "Desktop, Docker, and cloud deploy" },
];

export default function DocsHome() {
  return (
    <div className="docs-prose">
      <h1>Getting Started</h1>
      <p className="text-lg !text-surface-300">
        Welcome to the Daedalus documentation. This guide will walk you through setting up
        your print farm and using every feature of the platform.
      </p>

      <h2>What is Daedalus?</h2>
      <p>
        Daedalus is an all-in-one desktop platform for managing 3D printers, fulfilling orders
        from Etsy, Shopify &amp; Squarespace, tracking materials, and growing your maker business.
        It runs as a native desktop app (macOS/Windows) or a Docker container.
      </p>

      <h2>Prerequisites</h2>
      <ul>
        <li>A computer running macOS or Windows (for the desktop app), or a server with Docker</li>
        <li>One or more 3D printers (Bambu Lab, OctoPrint, Klipper, or manual logging)</li>
        <li>Marketplace accounts (Etsy, Shopify, or Squarespace) if you want order sync</li>
      </ul>

      <h2>Quick Start</h2>
      <ol>
        <li>
          <strong>Install Daedalus</strong> &mdash; Download the desktop app or run via Docker.
          See the <Link href="/docs/deployment">Deployment</Link> page for all options.
        </li>
        <li>
          <strong>Add your printers</strong> &mdash; Go to the Printers page and add your fleet.
          Bambu Lab printers can be discovered automatically via network scan or Cloud pairing.
          See <Link href="/docs/printers">Printers</Link>.
        </li>
        <li>
          <strong>Set up your product catalog</strong> &mdash; Create projects with parts and
          design versions. Upload 3MF files for automatic metadata extraction.
          See <Link href="/docs/projects">Projects</Link>.
        </li>
        <li>
          <strong>Connect your sales channels</strong> &mdash; Link Etsy, Shopify, or Squarespace
          to automatically import orders.
          See <Link href="/docs/channels">Sales Channels</Link>.
        </li>
        <li>
          <strong>Start fulfilling</strong> &mdash; Process orders into tasks, dispatch print jobs,
          and track everything from order to shipment.
          See <Link href="/docs/tasks">Tasks &amp; Print Jobs</Link>.
        </li>
      </ol>

      <div className="docs-callout">
        <strong>Tip:</strong> If you&apos;re just getting started with a single printer, you can skip
        the sales channel setup and create orders manually. Daedalus scales from one printer
        to a full farm.
      </div>

      <h2>Explore the Docs</h2>
      <div className="not-prose grid gap-3 sm:grid-cols-2">
        {sections.map((s) => {
          const Icon = s.icon;
          return (
            <Link
              key={s.href}
              href={s.href}
              className="group flex items-center gap-3 rounded-xl border border-surface-800 bg-surface-900/50 px-4 py-3 no-underline transition-colors hover:border-accent-500/30 hover:bg-surface-900"
            >
              <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-accent-500/10 text-accent-500">
                <Icon className="h-4 w-4" />
              </div>
              <div className="min-w-0">
                <div className="text-sm font-medium text-surface-100 group-hover:text-accent-400 transition-colors">
                  {s.label}
                </div>
                <div className="text-xs text-surface-500 truncate">{s.desc}</div>
              </div>
              <ArrowRight className="ml-auto h-4 w-4 shrink-0 text-surface-600 group-hover:text-accent-500 transition-colors" />
            </Link>
          );
        })}
      </div>
    </div>
  );
}
