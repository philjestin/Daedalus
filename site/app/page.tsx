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
      <div className="h-full rounded-none border border-border bg-surface p-6 shadow-card transition-transform duration-300 hover:-translate-y-0.5">
        <div className="mb-4 flex h-11 w-11 items-center justify-center rounded-none border border-border bg-primary/10 text-primary">
          {icon}
        </div>
        <h3 className="type-heading-xs mb-2 text-ink">{title}</h3>
        <p className="type-body-sm text-body">
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
      <div className="type-section text-primary">{value}</div>
      <div className="type-body-sm mt-1 text-body">{label}</div>
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
      <div className="flex items-center gap-4 rounded-none border border-border bg-surface px-6 py-5 shadow-card">
        <div className="flex h-11 w-11 items-center justify-center rounded-none border border-border bg-primary/10">
          <Wifi className="h-5 w-5 text-primary" />
        </div>
        <div>
          <div className="type-heading-xs text-ink">{name}</div>
          <div className="type-caption font-mono text-body">{protocol}</div>
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
    <AnimatedSection className="mx-auto mb-section max-w-2xl text-center">
      <div className="badge mb-4">
        <span className="h-1.5 w-1.5 rounded-full bg-accent" />
        {badge}
      </div>
      <h2 className="type-section mb-4 text-ink">
        {title}
      </h2>
      <p className="type-body text-body">{description}</p>
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
      className={`w-full rounded-none border border-border ${className}`}
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
      className={`flex flex-col gap-section lg:flex-row lg:items-center lg:gap-16 ${
        reversed ? "lg:flex-row-reverse" : ""
      }`}
    >
      <AnimatedSection className="flex-1">
        <div className="mb-4 flex h-11 w-11 items-center justify-center rounded-none border border-border bg-primary/10 text-primary">
          {icon}
        </div>
        <h3 className="type-heading-sm mb-3 text-ink">{title}</h3>
        <p className="type-body mb-6 text-body">{description}</p>
        <ul className="space-y-3">
          {bullets.map((b, i) => (
            <li key={i} className="type-body-sm flex items-start gap-3">
              <CheckCircle2 className="mt-0.5 h-4 w-4 shrink-0 text-primary" />
              <span className="text-body">{b}</span>
            </li>
          ))}
        </ul>
      </AnimatedSection>

      <AnimatedSection className="flex-1" delay={150}>
        <div className="rounded-none border border-border bg-surface p-1 shadow-card">
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
          ? "border-b border-border bg-canvas/95 backdrop-blur-xl"
          : "border-b border-transparent bg-transparent"
      }`}
    >
      <div className="mx-auto flex max-w-6xl items-center justify-between px-6 py-3">
        <a href="#" className="flex items-center gap-2.5">
          <div className="flex h-8 w-8 items-center justify-center rounded-none border border-border bg-primary/10">
            <Layers className="h-4.5 w-4.5 text-primary" />
          </div>
          <span className="type-heading-xs text-ink">
            RankOS
          </span>
        </a>

        <div className="hidden items-center gap-6 lg:flex xl:gap-8">
          {links.map((l) => (
            <a
              key={l.href}
              href={l.href}
              className="type-body-sm text-body transition-colors hover:text-primary xl:text-[17px]"
            >
              {l.label}
            </a>
          ))}
        </div>

        <div className="hidden items-center gap-3 lg:flex">
          <a
            href="#get-started"
            className="btn btn-primary"
          >
            Get Started
          </a>
        </div>

        <button
          onClick={() => setOpen(!open)}
          className="inline-flex h-11 w-11 items-center justify-center text-body lg:hidden"
        >
          {open ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
        </button>
      </div>

      {open && (
        <div className="border-t border-border bg-canvas px-6 py-4 lg:hidden">
          {links.map((l) => (
            <a
              key={l.href}
              href={l.href}
              onClick={() => setOpen(false)}
              className="type-body block py-2 text-body transition-colors hover:text-primary"
            >
              {l.label}
            </a>
          ))}
          <a
            href="#get-started"
            onClick={() => setOpen(false)}
            className="btn btn-primary mt-3 w-full"
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
      <section className="relative overflow-hidden pt-[calc(82px+60px)] pb-section lg:pt-[calc(82px+120px)] lg:pb-band">
        <div className="absolute inset-0 grid-bg" />
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 h-[600px] w-[600px] rounded-full bg-accent/15 blur-[120px]" />

        <div className="relative mx-auto max-w-6xl px-6">
          <AnimatedSection className="mx-auto max-w-3xl text-center">
            <div className="badge mb-6">
              <Zap className="h-3.5 w-3.5 text-primary" />
              Print Farm Management + Order Fulfillment
            </div>

            <h1 className="type-hero mb-6 text-ink">
              Run your print farm
              <br />
              <span className="text-primary">like a factory</span>
            </h1>

            <p className="type-body mx-auto mb-10 max-w-xl text-body">
              RankOS is the all-in-one desktop platform for managing printers,
              fulfilling orders from Etsy, Shopify & Squarespace, tracking
              materials, and growing your maker business.
            </p>

            <div className="flex flex-col items-center gap-4 sm:flex-row sm:justify-center">
              <a
                href="#get-started"
                className="btn btn-primary group transition-all hover:gap-3"
              >
                Get Started
                <ArrowRight className="h-4 w-4 transition-transform group-hover:translate-x-0.5" />
              </a>
              <a
                href="#features"
                className="btn btn-ghost"
              >
                See Features
                <ChevronRight className="h-4 w-4" />
              </a>
            </div>
          </AnimatedSection>

          {/* Hero stats */}
          <AnimatedSection
            className="mx-auto mt-section flex max-w-md justify-between"
            delay={300}
          >
            <Stat value="4" label="Printer Protocols" />
            <Stat value="4" label="Sales Channels" />
            <Stat value="50+" label="API Endpoints" />
          </AnimatedSection>

          {/* Hero screenshot */}
          <AnimatedSection className="mt-section" delay={400}>
            <div className="rounded-none border border-border bg-surface p-2 shadow-card">
              <Screenshot
                src="/screenshots/dashboard.png"
                alt="RankOS dashboard showing financial overview, revenue charts, and print farm status"
              />
            </div>
          </AnimatedSection>
        </div>
      </section>

      {/* ─── Features Grid ─────────────────────────────────────────────── */}
      <section id="features" className="py-section lg:py-band">
        <div className="mx-auto max-w-6xl px-6">
          <SectionHeading
            badge="Features"
            title="Everything you need to scale"
            description="From a single printer to a full farm, RankOS gives you complete visibility and control over every part of your operation."
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
        className="py-section lg:py-band border-t border-border"
      >
        <div className="mx-auto max-w-6xl px-6 space-y-section lg:space-y-band">
          <ShowcaseRow
            icon={<Printer className="h-5 w-5" />}
            title="Fleet management for every printer"
            description="Connect your entire fleet regardless of manufacturer. RankOS speaks every protocol so you don't have to."
            bullets={[
              "Bambu Lab via MQTT with AMS spool & tray management",
              "OctoPrint REST API with file upload and direct control",
              "Klipper/Moonraker for advanced open-source setups",
              "Manual printers for logging jobs without a connection",
              "Automatic network discovery and Bambu Cloud pairing",
              "Real-time temperature, progress, and HMS error tracking",
            ]}
            screenshot="/screenshots/printers.png"
            screenshotAlt="RankOS printer fleet showing Bambu Lab P1S, A1, and H2S printers with status and temperature readings"
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
            screenshotAlt="RankOS task view showing order fulfillment progress with completion tracking"
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
            screenshotAlt="RankOS sales analytics showing revenue by project, profit breakdown, and weekly insights"
          />
        </div>
      </section>

      {/* ─── Production & Materials ───────────────────────────────────── */}
      <section
        id="analytics"
        className="py-section lg:py-band border-t border-border"
      >
        <div className="mx-auto max-w-6xl px-6">
          <SectionHeading
            badge="Production"
            title="Visibility across your entire operation"
            description="From spool inventory to production timelines, see everything happening in your print farm at a glance."
          />
          <div className="grid gap-6 lg:grid-cols-2">
            <AnimatedSection>
              <div className="rounded-none border border-border bg-surface p-1 shadow-card">
                <Screenshot
                  src="/screenshots/timeline.png"
                  alt="RankOS production timeline showing Gantt-style order scheduling"
                />
              </div>
              <div className="mt-4 px-2">
                <h3 className="type-heading-xs text-ink">
                  Production Timeline
                </h3>
                <p className="type-body-sm mt-1 text-body">
                  Gantt-style view of all active orders, tasks, and print jobs.
                  Configurable date ranges with status color coding: pending, in
                  progress, printing, completed, shipped, and cancelled.
                </p>
              </div>
            </AnimatedSection>
            <AnimatedSection delay={150}>
              <div className="rounded-none border border-border bg-surface p-1 shadow-card">
                <Screenshot
                  src="/screenshots/materials.png"
                  alt="RankOS material inventory showing spool tracking with weight and color"
                />
              </div>
              <div className="mt-4 px-2">
                <h3 className="type-heading-xs text-ink">
                  Material Inventory
                </h3>
                <p className="type-body-sm mt-1 text-body">
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
      <section className="py-section lg:py-band border-t border-border">
        <div className="mx-auto max-w-6xl px-6 space-y-section lg:space-y-band">
          <ShowcaseRow
            icon={<Receipt className="h-5 w-5" />}
            title="Receipt OCR that actually works"
            description="Upload a photo of any receipt and RankOS extracts the details automatically. Confirm with a click and categorize across 8 expense types."
            bullets={[
              "Automatic vendor, amount, date, and line item extraction",
              "Confidence scores so you know when to double-check",
              "8 categories: filament, parts, tools, shipping, fees, subscriptions, advertising, and other",
              "Auto-create spool entries from filament expenses",
              "Retry OCR processing if results need improvement",
            ]}
            screenshot="/screenshots/expenses.png"
            screenshotAlt="RankOS expense tracking showing OCR-parsed receipts from Amazon, Bambu Lab, and Home Depot"
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
            screenshotAlt="RankOS project catalog showing maker products with descriptions and dates"
          />
        </div>
      </section>

      {/* ─── 3MF + Design Versioning ───────────────────────────────────── */}
      <section className="py-section lg:py-band border-t border-border">
        <div className="mx-auto max-w-6xl px-6">
          <SectionHeading
            badge="Smart Workflows"
            title="From file to finished product"
            description="Upload 3MF files and RankOS extracts everything it needs. Paired with immutable design versioning and automated job management, you always know exactly what was printed."
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
        className="py-section lg:py-band border-t border-border"
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
            <div className="mx-auto flex max-w-sm items-center gap-4 rounded-none border border-border bg-surface px-6 py-5 shadow-card">
              <div className="flex h-11 w-11 items-center justify-center rounded-none border border-border bg-canvas">
                <Printer className="h-5 w-5 text-body" />
              </div>
              <div>
                <div className="type-heading-xs text-ink">
                  Manual Printers
                </div>
                <div className="type-caption font-mono text-body">
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
        className="py-section lg:py-band border-t border-border"
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
                <div className="h-full rounded-none border border-border bg-surface p-6 shadow-card transition-transform duration-300 hover:-translate-y-0.5">
                  <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-none border border-border bg-primary/10 text-primary">
                    {item.icon}
                  </div>
                  <h3 className="type-heading-xs mb-2 text-ink">
                    {item.title}
                  </h3>
                  <p className="type-body-sm mb-4 text-body">
                    {item.desc}
                  </p>
                  <div className="rounded-none border border-border bg-canvas px-4 py-2.5 font-mono text-[15px] leading-none text-ink">
                    <span className="text-body">$ </span>
                    {item.code}
                  </div>
                </div>
              </AnimatedSection>
            ))}
          </div>
        </div>
      </section>

      {/* ─── Tech Stack ────────────────────────────────────────────────── */}
      <section className="py-section lg:py-band border-t border-border">
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
                    className="rounded-none border border-border bg-surface px-4 py-3 text-center"
                  >
                    <div className="type-body-sm-strong text-ink">
                      {t.label}
                    </div>
                    <div className="type-caption mt-0.5 font-mono text-body">
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
        className="relative py-section lg:py-band border-t border-border"
      >
        <div className="absolute inset-0 grid-bg" />
        <div className="absolute top-1/2 left-1/2 h-[500px] w-[500px] -translate-x-1/2 -translate-y-1/2 rounded-full bg-accent/15 blur-[100px]" />

        <div className="relative mx-auto max-w-6xl px-6">
          <AnimatedSection className="mx-auto max-w-2xl text-center">
            <h2 className="type-section mb-4 text-ink">
              Ready to take control?
            </h2>
            <p className="type-body mb-8 text-body">
              Stop juggling spreadsheets, browser tabs, and printer UIs.
              RankOS brings your entire operation into one place.
            </p>

            <div className="flex flex-col items-center gap-4 sm:flex-row sm:justify-center">
              <a
                href="https://github.com/cadimartech/rankos"
                className="btn btn-primary"
              >
                <Github className="h-4 w-4" />
                View on GitHub
                <ExternalLink className="h-3.5 w-3.5" />
              </a>
              <div className="inline-flex max-w-full items-center rounded-none border border-border bg-surface px-[30px] py-4 text-center font-mono text-[15px] leading-[1.6] text-ink">
                <span className="text-body">$ </span>
                git clone cadimartech/rankos && make dev
              </div>
            </div>
          </AnimatedSection>
        </div>
      </section>

      {/* ─── Footer ────────────────────────────────────────────────────── */}
      <footer className="border-t border-border py-section">
        <div className="mx-auto max-w-6xl px-6">
          <div className="flex flex-col items-center justify-between gap-6 sm:flex-row">
            <div className="flex items-center gap-2.5">
              <div className="flex h-7 w-7 items-center justify-center rounded-none border border-border bg-primary/10">
                <Layers className="h-4 w-4 text-primary" />
              </div>
              <span className="type-heading-xs text-ink">
                RankOS
              </span>
            </div>
            <div className="type-caption flex items-center gap-6 text-body">
              <span>Built with Go, React & Wails</span>
              <span className="hidden text-muted sm:inline">|</span>
              <span>Print Farm Management</span>
            </div>
          </div>
        </div>
      </footer>
    </>
  );
}
