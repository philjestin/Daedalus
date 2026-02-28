"use client";

import {
  Layers,
  Printer,
  ShoppingBag,
  BarChart3,
  Package,
  Receipt,
  Wifi,
  FileText,
  Clock,
  Zap,
  Monitor,
  Container,
  Cloud,
  ChevronRight,
  ArrowRight,
  FolderKanban,
  CheckCircle2,
  DollarSign,
  RefreshCw,
  Shield,
  Gauge,
  GitBranch,
  ScanLine,
  Boxes,
  TrendingUp,
  PieChart,
  CalendarRange,
  Menu,
  X,
  Github,
  ExternalLink,
  Bell,
  Database,
  ListChecks,
  ShoppingCart,
} from "lucide-react";
import { useState, useEffect, useRef, type ReactNode } from "react";

// ─── Animated section wrapper ───────────────────────────────────────────────
function AnimatedSection({
  children,
  className = "",
  delay = 0,
}: {
  children: ReactNode;
  className?: string;
  delay?: number;
}) {
  const ref = useRef<HTMLDivElement>(null);
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setTimeout(() => setVisible(true), delay);
          observer.unobserve(el);
        }
      },
      { threshold: 0.1 }
    );
    observer.observe(el);
    return () => observer.disconnect();
  }, [delay]);

  return (
    <div
      ref={ref}
      className={`transition-all duration-700 ${
        visible ? "opacity-100 translate-y-0" : "opacity-0 translate-y-8"
      } ${className}`}
    >
      {children}
    </div>
  );
}

// ─── Feature card ───────────────────────────────────────────────────────────
function FeatureCard({
  icon,
  title,
  description,
  delay = 0,
}: {
  icon: ReactNode;
  title: string;
  description: string;
  delay?: number;
}) {
  return (
    <AnimatedSection delay={delay}>
      <div className="glow-card rounded-xl border border-surface-800 bg-surface-900/50 p-6 h-full">
        <div className="mb-4 flex h-11 w-11 items-center justify-center rounded-lg bg-accent-500/10 text-accent-500">
          {icon}
        </div>
        <h3 className="mb-2 text-lg font-semibold text-surface-50">{title}</h3>
        <p className="text-sm leading-relaxed text-surface-400">
          {description}
        </p>
      </div>
    </AnimatedSection>
  );
}

// ─── Stat pill ──────────────────────────────────────────────────────────────
function Stat({ value, label }: { value: string; label: string }) {
  return (
    <div className="text-center">
      <div className="text-3xl font-bold gradient-text">{value}</div>
      <div className="mt-1 text-sm text-surface-400">{label}</div>
    </div>
  );
}

// ─── Integration badge ──────────────────────────────────────────────────────
function IntegrationBadge({
  name,
  protocol,
  delay = 0,
}: {
  name: string;
  protocol: string;
  delay?: number;
}) {
  return (
    <AnimatedSection delay={delay}>
      <div className="glow-card flex items-center gap-4 rounded-xl border border-surface-800 bg-surface-900/50 px-6 py-5">
        <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-accent-500/10">
          <Wifi className="h-5 w-5 text-accent-500" />
        </div>
        <div>
          <div className="font-semibold text-surface-50">{name}</div>
          <div className="font-mono text-xs text-surface-500">{protocol}</div>
        </div>
      </div>
    </AnimatedSection>
  );
}

// ─── Section heading ────────────────────────────────────────────────────────
function SectionHeading({
  badge,
  title,
  description,
}: {
  badge: string;
  title: string;
  description: string;
}) {
  return (
    <AnimatedSection className="mx-auto mb-16 max-w-2xl text-center">
      <div className="mb-4 inline-flex items-center gap-2 rounded-full border border-accent-500/20 bg-accent-500/5 px-4 py-1.5 text-xs font-medium text-accent-400">
        <span className="h-1.5 w-1.5 rounded-full bg-accent-500" />
        {badge}
      </div>
      <h2 className="mb-4 text-3xl font-bold tracking-tight text-surface-50 sm:text-4xl">
        {title}
      </h2>
      <p className="text-lg leading-relaxed text-surface-400">{description}</p>
    </AnimatedSection>
  );
}

// ─── Screenshot component ───────────────────────────────────────────────────
function Screenshot({
  src,
  alt,
  className = "",
}: {
  src: string;
  alt: string;
  className?: string;
}) {
  return (
    <img
      src={src}
      alt={alt}
      className={`w-full rounded-xl ${className}`}
      loading="lazy"
    />
  );
}

// ─── Showcase row (alternating image + text) ────────────────────────────────
function ShowcaseRow({
  icon,
  title,
  description,
  bullets,
  reversed = false,
  screenshot,
  screenshotAlt,
}: {
  icon: ReactNode;
  title: string;
  description: string;
  bullets: string[];
  reversed?: boolean;
  screenshot: string;
  screenshotAlt: string;
}) {
  return (
    <div
      className={`flex flex-col gap-12 lg:flex-row lg:items-center lg:gap-16 ${
        reversed ? "lg:flex-row-reverse" : ""
      }`}
    >
      <AnimatedSection className="flex-1">
        <div className="mb-4 flex h-11 w-11 items-center justify-center rounded-lg bg-accent-500/10 text-accent-500">
          {icon}
        </div>
        <h3 className="mb-3 text-2xl font-bold text-surface-50">{title}</h3>
        <p className="mb-6 leading-relaxed text-surface-400">{description}</p>
        <ul className="space-y-3">
          {bullets.map((b, i) => (
            <li key={i} className="flex items-start gap-3 text-sm">
              <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0 text-accent-500" />
              <span className="text-surface-300">{b}</span>
            </li>
          ))}
        </ul>
      </AnimatedSection>

      <AnimatedSection className="flex-1" delay={150}>
        <div className="glow-accent rounded-2xl border border-surface-800 bg-surface-900/80 p-1">
          <Screenshot src={screenshot} alt={screenshotAlt} />
        </div>
      </AnimatedSection>
    </div>
  );
}

// ─── Navigation ─────────────────────────────────────────────────────────────
function Nav() {
  const [open, setOpen] = useState(false);
  const [scrolled, setScrolled] = useState(false);

  useEffect(() => {
    const handle = () => setScrolled(window.scrollY > 20);
    window.addEventListener("scroll", handle, { passive: true });
    return () => window.removeEventListener("scroll", handle);
  }, []);

  const links = [
    { href: "#features", label: "Features" },
    { href: "#printers", label: "Printers" },
    { href: "#orders", label: "Orders" },
    { href: "#analytics", label: "Analytics" },
    { href: "#integrations", label: "Integrations" },
    { href: "#deploy", label: "Deploy" },
    { href: "/docs", label: "Docs" },
  ];

  return (
    <nav
      className={`fixed top-0 z-50 w-full transition-all duration-300 ${
        scrolled
          ? "bg-surface-950/80 backdrop-blur-xl border-b border-surface-800/50"
          : "bg-transparent"
      }`}
    >
      <div className="mx-auto flex max-w-6xl items-center justify-between px-6 py-4">
        <a href="#" className="flex items-center gap-2.5">
          <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-accent-500/10">
            <Layers className="h-4.5 w-4.5 text-accent-500" />
          </div>
          <span className="text-lg font-bold tracking-tight text-surface-50">
            Daedalus
          </span>
        </a>

        <div className="hidden items-center gap-8 md:flex">
          {links.map((l) => (
            <a
              key={l.href}
              href={l.href}
              className="text-sm text-surface-400 transition-colors hover:text-surface-100"
            >
              {l.label}
            </a>
          ))}
        </div>

        <div className="hidden md:flex items-center gap-3">
          <a
            href="#get-started"
            className="rounded-lg bg-accent-500 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-accent-600"
          >
            Get Started
          </a>
        </div>

        <button
          onClick={() => setOpen(!open)}
          className="md:hidden text-surface-400"
        >
          {open ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
        </button>
      </div>

      {open && (
        <div className="border-t border-surface-800/50 bg-surface-950/95 backdrop-blur-xl px-6 py-4 md:hidden">
          {links.map((l) => (
            <a
              key={l.href}
              href={l.href}
              onClick={() => setOpen(false)}
              className="block py-2 text-sm text-surface-400 transition-colors hover:text-surface-100"
            >
              {l.label}
            </a>
          ))}
          <a
            href="#get-started"
            onClick={() => setOpen(false)}
            className="mt-3 block rounded-lg bg-accent-500 px-4 py-2 text-center text-sm font-medium text-white"
          >
            Get Started
          </a>
        </div>
      )}
    </nav>
  );
}

// ─── Page ───────────────────────────────────────────────────────────────────
export default function Home() {
  return (
    <>
      <Nav />

      {/* ─── Hero ──────────────────────────────────────────────────────── */}
      <section className="relative overflow-hidden pt-32 pb-24 sm:pt-40 sm:pb-32">
        <div className="absolute inset-0 grid-bg" />
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[600px] h-[600px] rounded-full bg-accent-500/5 blur-[120px]" />

        <div className="relative mx-auto max-w-6xl px-6">
          <AnimatedSection className="mx-auto max-w-3xl text-center">
            <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-accent-500/20 bg-accent-500/5 px-4 py-1.5 text-xs font-medium text-accent-400">
              <Zap className="h-3.5 w-3.5" />
              Print Farm Management + Order Fulfillment
            </div>

            <h1 className="mb-6 text-4xl font-bold tracking-tight text-surface-50 sm:text-6xl lg:text-7xl">
              Run your print farm
              <br />
              <span className="gradient-text">like a factory</span>
            </h1>

            <p className="mx-auto mb-10 max-w-xl text-lg leading-relaxed text-surface-400">
              Daedalus is the all-in-one desktop platform for managing printers,
              fulfilling orders from Etsy, Shopify & Squarespace, tracking
              materials, and growing your maker business.
            </p>

            <div className="flex flex-col items-center gap-4 sm:flex-row sm:justify-center">
              <a
                href="#get-started"
                className="group inline-flex items-center gap-2 rounded-xl bg-accent-500 px-6 py-3 text-sm font-semibold text-white transition-all hover:bg-accent-600 hover:gap-3"
              >
                Get Started
                <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
              </a>
              <a
                href="#features"
                className="inline-flex items-center gap-2 rounded-xl border border-surface-700 px-6 py-3 text-sm font-medium text-surface-300 transition-colors hover:border-surface-600 hover:text-surface-100"
              >
                See Features
                <ChevronRight className="h-4 w-4" />
              </a>
            </div>
          </AnimatedSection>

          {/* Hero stats */}
          <AnimatedSection
            className="mx-auto mt-20 flex max-w-md justify-between"
            delay={300}
          >
            <Stat value="4" label="Printer Protocols" />
            <Stat value="4" label="Sales Channels" />
            <Stat value="50+" label="API Endpoints" />
          </AnimatedSection>

          {/* Hero screenshot */}
          <AnimatedSection className="mt-16" delay={400}>
            <div className="glow-accent rounded-2xl border border-surface-800 bg-surface-900/80 p-2">
              <Screenshot
                src="/screenshots/dashboard.png"
                alt="Daedalus dashboard showing financial overview, revenue charts, and print farm status"
              />
            </div>
          </AnimatedSection>
        </div>
      </section>

      {/* ─── Features Grid ─────────────────────────────────────────────── */}
      <section id="features" className="py-24 sm:py-32">
        <div className="mx-auto max-w-6xl px-6">
          <SectionHeading
            badge="Features"
            title="Everything you need to scale"
            description="From a single printer to a full farm, Daedalus gives you complete visibility and control over every part of your operation."
          />

          <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
            <FeatureCard
              icon={<Printer className="h-5 w-5" />}
              title="Multi-Printer Fleet Control"
              description="Manage Bambu Lab, OctoPrint, Klipper, and manual printers from a single dashboard. Real-time status, temperatures, and progress via MQTT and REST."
              delay={0}
            />
            <FeatureCard
              icon={<ShoppingBag className="h-5 w-5" />}
              title="Multi-Channel Orders"
              description="Unified order management for Etsy, Shopify, Squarespace, and direct sales. Import, track, and fulfill from one interface."
              delay={80}
            />
            <FeatureCard
              icon={<FolderKanban className="h-5 w-5" />}
              title="Product Catalog"
              description="Define products with multi-part designs, pricing, and printer constraints. Immutable design versioning with 3MF file parsing."
              delay={160}
            />
            <FeatureCard
              icon={<Zap className="h-5 w-5" />}
              title="Auto-Dispatch"
              description="Automatically assign print jobs to available printers based on material compatibility and constraints. Human-in-the-loop confirm, reject, or skip."
              delay={0}
            />
            <FeatureCard
              icon={<Package className="h-5 w-5" />}
              title="Material & Spool Tracking"
              description="Track filament inventory by type, color, and weight. Bambu Lab AMS integration with automatic spool and tray management."
              delay={80}
            />
            <FeatureCard
              icon={<BarChart3 className="h-5 w-5" />}
              title="Financial Analytics"
              description="Revenue trends, profit-per-hour, gross margins, and per-product profitability. Know exactly how your business is performing."
              delay={160}
            />
            <FeatureCard
              icon={<Receipt className="h-5 w-5" />}
              title="Expense Tracking with OCR"
              description="Upload receipts and let OCR parse vendor, amount, date, and items. Categorize across filament, tools, shipping, fees, and more."
              delay={0}
            />
            <FeatureCard
              icon={<ListChecks className="h-5 w-5" />}
              title="Task Checklists"
              description="Auto-generated checklists from product parts. Track completion per item, print directly from checklist entries, and regenerate as needed."
              delay={80}
            />
            <FeatureCard
              icon={<Bell className="h-5 w-5" />}
              title="Material Alerts"
              description="Low spool inventory alerts with configurable thresholds. Warning and critical severity levels with dismissible notifications."
              delay={160}
            />
            <FeatureCard
              icon={<CalendarRange className="h-5 w-5" />}
              title="Production Timeline"
              description="Gantt-style timeline of orders, tasks, and print jobs. Configurable date ranges with progress tracking and status color coding."
              delay={0}
            />
            <FeatureCard
              icon={<Database className="h-5 w-5" />}
              title="Automated Backups"
              description="Scheduled daily or weekly backups with retention policies. Manual backup creation, restoration, and automatic backup on startup."
              delay={80}
            />
            <FeatureCard
              icon={<Wifi className="h-5 w-5" />}
              title="Real-Time Updates"
              description="WebSocket-powered live updates for printer state, job progress, dispatch requests, and order sync. No manual refreshing needed."
              delay={160}
            />
          </div>
        </div>
      </section>

      {/* ─── Printer Management Deep Dive ──────────────────────────────── */}
      <section
        id="printers"
        className="py-24 sm:py-32 border-t border-surface-800/50"
      >
        <div className="mx-auto max-w-6xl px-6 space-y-32">
          <ShowcaseRow
            icon={<Printer className="h-5 w-5" />}
            title="Fleet management for every printer"
            description="Connect your entire fleet regardless of manufacturer. Daedalus speaks every protocol so you don't have to."
            bullets={[
              "Bambu Lab via MQTT with AMS spool & tray management",
              "OctoPrint REST API with file upload and direct control",
              "Klipper/Moonraker for advanced open-source setups",
              "Manual printers for logging jobs without a connection",
              "Automatic network discovery and Bambu Cloud pairing",
              "Real-time temperature, progress, and HMS error tracking",
            ]}
            screenshot="/screenshots/printers.png"
            screenshotAlt="Daedalus printer fleet showing Bambu Lab P1S, A1, and H2S printers with status and temperature readings"
          />

          <ShowcaseRow
            icon={<ShoppingBag className="h-5 w-5" />}
            title="Orders from everywhere, managed in one place"
            description="Stop switching between Etsy, Shopify, Squarespace, and spreadsheets. See every order in a unified interface with full status tracking."
            bullets={[
              "Etsy OAuth with PKCE and real-time webhook sync",
              "Shopify OAuth integration with order and inventory sync",
              "Squarespace API integration for storefront orders",
              "Manual order creation for direct sales and custom work",
              "Task-based workflow: orders become trackable print jobs",
              "Status pipeline from pending through printing to shipped",
            ]}
            reversed
            screenshot="/screenshots/tasks.png"
            screenshotAlt="Daedalus task view showing order fulfillment progress with completion tracking"
          />

          <ShowcaseRow
            icon={<TrendingUp className="h-5 w-5" />}
            title="Know your numbers"
            description="Built-in analytics show you exactly where your money is going and where it's coming from. Make data-driven decisions about what to print."
            bullets={[
              "Revenue trends with time-series charts and profit overlays",
              "Sales breakdown by channel and by project",
              "Profit-per-hour metrics to optimize your product mix",
              "Expense tracking across 8 categories with OCR receipts",
              "Gross margin and net revenue after marketplace fees",
              "Per-product profitability with unit cost analysis",
            ]}
            screenshot="/screenshots/sales.png"
            screenshotAlt="Daedalus sales analytics showing revenue by project, profit breakdown, and weekly insights"
          />
        </div>
      </section>

      {/* ─── Production & Materials ───────────────────────────────────── */}
      <section
        id="analytics"
        className="py-24 sm:py-32 border-t border-surface-800/50"
      >
        <div className="mx-auto max-w-6xl px-6">
          <SectionHeading
            badge="Production"
            title="Visibility across your entire operation"
            description="From spool inventory to production timelines, see everything happening in your print farm at a glance."
          />
          <div className="grid gap-6 lg:grid-cols-2">
            <AnimatedSection>
              <div className="glow-card rounded-2xl border border-surface-800 bg-surface-900/50 p-1">
                <Screenshot
                  src="/screenshots/timeline.png"
                  alt="Daedalus production timeline showing Gantt-style order scheduling"
                />
              </div>
              <div className="mt-4 px-2">
                <h3 className="font-semibold text-surface-100">
                  Production Timeline
                </h3>
                <p className="mt-1 text-sm text-surface-400">
                  Gantt-style view of all active orders, tasks, and print jobs.
                  Configurable date ranges with status color coding: pending, in
                  progress, printing, completed, shipped, and cancelled.
                </p>
              </div>
            </AnimatedSection>
            <AnimatedSection delay={150}>
              <div className="glow-card rounded-2xl border border-surface-800 bg-surface-900/50 p-1">
                <Screenshot
                  src="/screenshots/materials.png"
                  alt="Daedalus material inventory showing spool tracking with weight and color"
                />
              </div>
              <div className="mt-4 px-2">
                <h3 className="font-semibold text-surface-100">
                  Material Inventory
                </h3>
                <p className="mt-1 text-sm text-surface-400">
                  Track every spool across PLA, PETG, ABS, ASA, and TPU.
                  Automatic weight updates with Bambu Lab AMS integration, color
                  visualization, and low-stock alerts.
                </p>
              </div>
            </AnimatedSection>
          </div>
        </div>
      </section>

      {/* ─── Expenses & Projects ─────────────────────────────────────── */}
      <section className="py-24 sm:py-32 border-t border-surface-800/50">
        <div className="mx-auto max-w-6xl px-6 space-y-32">
          <ShowcaseRow
            icon={<Receipt className="h-5 w-5" />}
            title="Receipt OCR that actually works"
            description="Upload a photo of any receipt and Daedalus extracts the details automatically. Confirm with a click and categorize across 8 expense types."
            bullets={[
              "Automatic vendor, amount, date, and line item extraction",
              "Confidence scores so you know when to double-check",
              "8 categories: filament, parts, tools, shipping, fees, subscriptions, advertising, and other",
              "Auto-create spool entries from filament expenses",
              "Retry OCR processing if results need improvement",
            ]}
            screenshot="/screenshots/expenses.png"
            screenshotAlt="Daedalus expense tracking showing OCR-parsed receipts from Amazon, Bambu Lab, and Home Depot"
          />

          <ShowcaseRow
            icon={<FolderKanban className="h-5 w-5" />}
            title="Organize products your way"
            description="Build a product catalog with multi-part designs, reusable templates, and automatic cost rollups. Everything you need to know about what you make."
            bullets={[
              "Multi-part product definitions with design versioning",
              "3MF parsing extracts print time, weight, filament, and nozzle size",
              "Reusable templates with materials and supplies per recipe",
              "Automatic cost rollups: material, printing time, and supplies",
              "Tag-based filtering and search across parts and designs",
              "Per-product profitability tracking with unit cost analysis",
            ]}
            reversed
            screenshot="/screenshots/projects.png"
            screenshotAlt="Daedalus project catalog showing maker products with descriptions and dates"
          />
        </div>
      </section>

      {/* ─── 3MF + Design Versioning ───────────────────────────────────── */}
      <section className="py-24 sm:py-32 border-t border-surface-800/50">
        <div className="mx-auto max-w-6xl px-6">
          <SectionHeading
            badge="Smart Workflows"
            title="From file to finished product"
            description="Upload 3MF files and Daedalus extracts everything it needs. Paired with immutable design versioning and automated job management, you always know exactly what was printed."
          />

          <div className="grid gap-5 sm:grid-cols-2 lg:grid-cols-4">
            {[
              {
                icon: <FileText className="h-5 w-5" />,
                title: "3MF Parsing",
                desc: "Automatically extract print time, weight, filament type, nozzle size, and printer model from sliced files.",
              },
              {
                icon: <GitBranch className="h-5 w-5" />,
                title: "Design Versions",
                desc: "Immutable version history for every design. Know exactly which version was used for each print job.",
              },
              {
                icon: <ScanLine className="h-5 w-5" />,
                title: "Print Job Lifecycle",
                desc: "Full job tracking from queued to completed. Retry chains for failures, immutable event logs, and material usage recording.",
              },
              {
                icon: <Boxes className="h-5 w-5" />,
                title: "Bill of Materials",
                desc: "Define materials and supplies per product. Automatic cost rollups with snapshot-based material cost recording at print time.",
              },
            ].map((item, i) => (
              <FeatureCard
                key={item.title}
                icon={item.icon}
                title={item.title}
                description={item.desc}
                delay={i * 80}
              />
            ))}
          </div>
        </div>
      </section>

      {/* ─── Integrations ──────────────────────────────────────────────── */}
      <section
        id="integrations"
        className="py-24 sm:py-32 border-t border-surface-800/50"
      >
        <div className="mx-auto max-w-6xl px-6">
          <SectionHeading
            badge="Integrations"
            title="Connects to your stack"
            description="First-class integrations with the printers you own and the marketplaces you sell on."
          />

          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            <IntegrationBadge
              name="Bambu Lab"
              protocol="MQTT (LAN + Cloud)"
              delay={0}
            />
            <IntegrationBadge
              name="OctoPrint"
              protocol="REST API"
              delay={80}
            />
            <IntegrationBadge
              name="Klipper / Moonraker"
              protocol="REST API"
              delay={160}
            />
            <IntegrationBadge
              name="Etsy"
              protocol="OAuth 2.0 + PKCE + Webhooks"
              delay={0}
            />
            <IntegrationBadge
              name="Shopify"
              protocol="OAuth + Order Sync"
              delay={80}
            />
            <IntegrationBadge
              name="Squarespace"
              protocol="API Key"
              delay={160}
            />
          </div>

          <AnimatedSection className="mt-6">
            <div className="glow-card flex items-center gap-4 rounded-xl border border-surface-800 bg-surface-900/50 px-6 py-5 max-w-sm mx-auto">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-surface-800">
                <Printer className="h-5 w-5 text-surface-400" />
              </div>
              <div>
                <div className="font-semibold text-surface-50">
                  Manual Printers
                </div>
                <div className="font-mono text-xs text-surface-500">
                  No connection required
                </div>
              </div>
            </div>
          </AnimatedSection>
        </div>
      </section>

      {/* ─── Deploy ────────────────────────────────────────────────────── */}
      <section
        id="deploy"
        className="py-24 sm:py-32 border-t border-surface-800/50"
      >
        <div className="mx-auto max-w-6xl px-6">
          <SectionHeading
            badge="Deploy"
            title="Run it your way"
            description="Desktop app for macOS and Windows, self-hosted Docker container, or deploy to the cloud. Your data, your infrastructure."
          />

          <div className="grid gap-5 sm:grid-cols-3">
            {[
              {
                icon: <Monitor className="h-6 w-6" />,
                title: "Desktop App",
                desc: "Native macOS and Windows app powered by Wails. Full filesystem access, system tray, and native performance.",
                code: "wails build",
              },
              {
                icon: <Container className="h-6 w-6" />,
                title: "Docker",
                desc: "Multi-stage Docker build with embedded SQLite. Mount a volume for persistence and you're running.",
                code: "docker compose up",
              },
              {
                icon: <Cloud className="h-6 w-6" />,
                title: "Self-Hosted",
                desc: "Deploy anywhere with Docker. Mount a volume for your database, configure with environment variables, and you're live.",
                code: "docker compose up -d",
              },
            ].map((item, i) => (
              <AnimatedSection key={item.title} delay={i * 100}>
                <div className="glow-card h-full rounded-xl border border-surface-800 bg-surface-900/50 p-6">
                  <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-xl bg-accent-500/10 text-accent-500">
                    {item.icon}
                  </div>
                  <h3 className="mb-2 text-lg font-semibold text-surface-50">
                    {item.title}
                  </h3>
                  <p className="mb-4 text-sm leading-relaxed text-surface-400">
                    {item.desc}
                  </p>
                  <div className="rounded-lg bg-surface-950 px-4 py-2.5 font-mono text-sm text-accent-400">
                    <span className="text-surface-600">$ </span>
                    {item.code}
                  </div>
                </div>
              </AnimatedSection>
            ))}
          </div>
        </div>
      </section>

      {/* ─── Tech Stack ────────────────────────────────────────────────── */}
      <section className="py-24 sm:py-32 border-t border-surface-800/50">
        <div className="mx-auto max-w-6xl px-6">
          <SectionHeading
            badge="Under the Hood"
            title="Built with modern tools"
            description="A performant, type-safe stack designed for reliability and speed."
          />

          <AnimatedSection>
            <div className="mx-auto max-w-3xl">
              <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">
                {[
                  { label: "Go", detail: "Backend" },
                  { label: "React 19", detail: "Frontend" },
                  { label: "TypeScript", detail: "Type Safety" },
                  { label: "SQLite", detail: "Database" },
                  { label: "Wails", detail: "Desktop" },
                  { label: "Tailwind CSS 4", detail: "Styling" },
                  { label: "WebSocket", detail: "Real-time" },
                  { label: "TanStack Query", detail: "Data Fetching" },
                  { label: "Vite", detail: "Build Tool" },
                  { label: "Chi Router", detail: "HTTP" },
                  { label: "MQTT", detail: "Bambu Protocol" },
                  { label: "Docker", detail: "Containers" },
                ].map((t) => (
                  <div
                    key={t.label}
                    className="rounded-lg border border-surface-800 bg-surface-900/50 px-4 py-3 text-center"
                  >
                    <div className="text-sm font-semibold text-surface-100">
                      {t.label}
                    </div>
                    <div className="mt-0.5 font-mono text-[10px] text-surface-500">
                      {t.detail}
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </AnimatedSection>
        </div>
      </section>

      {/* ─── CTA ───────────────────────────────────────────────────────── */}
      <section
        id="get-started"
        className="relative py-24 sm:py-32 border-t border-surface-800/50"
      >
        <div className="absolute inset-0 grid-bg" />
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[500px] h-[500px] rounded-full bg-accent-500/5 blur-[100px]" />

        <div className="relative mx-auto max-w-6xl px-6">
          <AnimatedSection className="mx-auto max-w-2xl text-center">
            <h2 className="mb-4 text-3xl font-bold tracking-tight text-surface-50 sm:text-4xl">
              Ready to take control?
            </h2>
            <p className="mb-8 text-lg leading-relaxed text-surface-400">
              Stop juggling spreadsheets, browser tabs, and printer UIs.
              Daedalus brings your entire operation into one place.
            </p>

            <div className="flex flex-col items-center gap-4 sm:flex-row sm:justify-center">
              <a
                href="https://github.com"
                className="group inline-flex items-center gap-2 rounded-xl bg-accent-500 px-8 py-3.5 text-sm font-semibold text-white transition-all hover:bg-accent-600"
              >
                <Github className="h-4 w-4" />
                View on GitHub
                <ExternalLink className="h-3.5 w-3.5 opacity-50" />
              </a>
              <div className="rounded-xl border border-surface-700 bg-surface-900/50 px-6 py-3.5 font-mono text-sm text-surface-300">
                <span className="text-surface-600">$ </span>
                git clone daedalus && wails dev
              </div>
            </div>
          </AnimatedSection>
        </div>
      </section>

      {/* ─── Footer ────────────────────────────────────────────────────── */}
      <footer className="border-t border-surface-800/50 py-12">
        <div className="mx-auto max-w-6xl px-6">
          <div className="flex flex-col items-center justify-between gap-6 sm:flex-row">
            <div className="flex items-center gap-2.5">
              <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-accent-500/10">
                <Layers className="h-4 w-4 text-accent-500" />
              </div>
              <span className="text-sm font-bold text-surface-300">
                Daedalus
              </span>
            </div>
            <div className="flex items-center gap-6 text-xs text-surface-500">
              <span>Built with Go, React & Wails</span>
              <span className="hidden sm:inline text-surface-700">|</span>
              <span>Print Farm Management</span>
            </div>
          </div>
        </div>
      </footer>
    </>
  );
}
