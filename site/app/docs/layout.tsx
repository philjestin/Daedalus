"use client";

import { useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  Layers,
  Gauge,
  Printer,
  FolderKanban,
  Users,
  FileText,
  ShoppingBag,
  ListChecks,
  Package,
  Receipt,
  TrendingUp,
  ShoppingCart,
  CalendarRange,
  Settings,
  Cloud,
  Download,
  Menu,
  X,
  ArrowLeft,
  BookOpen,
} from "lucide-react";

const navItems = [
  { href: "/docs", label: "Getting Started", icon: BookOpen },
  { href: "/docs/dashboard", label: "Dashboard", icon: Gauge },
  { href: "/docs/printers", label: "Printers", icon: Printer },
  { href: "/docs/projects", label: "Projects", icon: FolderKanban },
  { href: "/docs/customers", label: "Customers", icon: Users },
  { href: "/docs/quotes", label: "Quotes", icon: FileText },
  { href: "/docs/orders", label: "Orders", icon: ShoppingBag },
  { href: "/docs/tasks", label: "Tasks & Print Jobs", icon: ListChecks },
  { href: "/docs/materials", label: "Materials & Inventory", icon: Package },
  { href: "/docs/expenses", label: "Expenses & OCR", icon: Receipt },
  { href: "/docs/sales", label: "Sales & Analytics", icon: TrendingUp },
  { href: "/docs/channels", label: "Sales Channels", icon: ShoppingCart },
  { href: "/docs/timeline", label: "Timeline", icon: CalendarRange },
  { href: "/docs/settings", label: "Settings", icon: Settings },
  { href: "/docs/deployment", label: "Deployment", icon: Cloud },
  { href: "/docs/downloads", label: "Downloads", icon: Download },
];

export default function DocsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const pathname = usePathname();

  return (
    <div className="min-h-screen bg-canvas">
      {/* Top bar */}
      <header className="fixed top-0 z-40 w-full border-b border-border bg-canvas/95 backdrop-blur-xl">
        <div className="flex h-16 items-center gap-4 px-6">
          <button
            onClick={() => setSidebarOpen(!sidebarOpen)}
            className="inline-flex h-11 w-11 items-center justify-center text-body hover:text-primary lg:hidden"
          >
            {sidebarOpen ? (
              <X className="h-5 w-5" />
            ) : (
              <Menu className="h-5 w-5" />
            )}
          </button>

          <Link href="/" className="type-body-sm flex items-center gap-2 text-body transition-colors hover:text-primary">
            <ArrowLeft className="h-4 w-4" />
            <span>Back to Home</span>
          </Link>

          <div className="mx-auto flex items-center gap-2">
            <div className="flex h-7 w-7 items-center justify-center rounded-none border border-border bg-primary/10">
              <Layers className="h-4 w-4 text-primary" />
            </div>
            <span className="type-heading-xs text-ink">
              RankOS Docs
            </span>
          </div>

          <div className="w-24" />
        </div>
      </header>

      <div className="flex pt-16">
        {/* Sidebar overlay on mobile */}
        {sidebarOpen && (
          <div
            className="fixed inset-0 z-30 bg-black/50 lg:hidden"
            onClick={() => setSidebarOpen(false)}
          />
        )}

        {/* Sidebar */}
        <aside
          className={`fixed top-16 bottom-0 z-30 w-56 overflow-y-auto border-r border-border bg-canvas px-3 py-4 transition-transform lg:translate-x-0 ${
            sidebarOpen ? "translate-x-0" : "-translate-x-full"
          }`}
        >
          <nav className="space-y-1">
            {navItems.map((item) => {
              const Icon = item.icon;
              const isActive =
                pathname === item.href ||
                (item.href !== "/docs" && pathname?.startsWith(item.href));
              return (
                <Link
                  key={item.href}
                  href={item.href}
                  onClick={() => setSidebarOpen(false)}
                  className={`type-body-sm flex items-center gap-2.5 rounded-none px-3 py-2 transition-colors ${
                    isActive
                      ? "bg-primary/10 font-medium text-primary"
                      : "text-body hover:bg-surface hover:text-ink"
                  }`}
                >
                  <Icon className="h-4 w-4 shrink-0" />
                  {item.label}
                </Link>
              );
            })}
          </nav>
        </aside>

        {/* Main content */}
        <main className="w-full lg:pl-56">
          <div className="mx-auto max-w-3xl px-6 py-section lg:px-12">
            {children}
          </div>
        </main>
      </div>
    </div>
  );
}
