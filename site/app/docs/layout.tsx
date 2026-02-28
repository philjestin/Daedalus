"use client";

import { useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  Layers,
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
  { href: "/docs/orders", label: "Orders", icon: ShoppingBag },
  { href: "/docs/tasks", label: "Tasks & Print Jobs", icon: ListChecks },
  { href: "/docs/materials", label: "Materials & Inventory", icon: Package },
  { href: "/docs/expenses", label: "Expenses & OCR", icon: Receipt },
  { href: "/docs/sales", label: "Sales & Analytics", icon: TrendingUp },
  { href: "/docs/channels", label: "Sales Channels", icon: ShoppingCart },
  { href: "/docs/timeline", label: "Timeline", icon: CalendarRange },
  { href: "/docs/settings", label: "Settings", icon: Settings },
  { href: "/docs/deployment", label: "Deployment", icon: Cloud },
];

export default function DocsLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const pathname = usePathname();

  return (
    <div className="min-h-screen bg-surface-950">
      {/* Top bar */}
      <header className="fixed top-0 z-40 w-full border-b border-surface-800/50 bg-surface-950/80 backdrop-blur-xl">
        <div className="flex h-14 items-center gap-4 px-6">
          <button
            onClick={() => setSidebarOpen(!sidebarOpen)}
            className="lg:hidden text-surface-400 hover:text-surface-100"
          >
            {sidebarOpen ? (
              <X className="h-5 w-5" />
            ) : (
              <Menu className="h-5 w-5" />
            )}
          </button>

          <Link href="/" className="flex items-center gap-2 text-surface-400 hover:text-surface-100 transition-colors">
            <ArrowLeft className="h-4 w-4" />
            <span className="text-sm">Back to Home</span>
          </Link>

          <div className="mx-auto flex items-center gap-2">
            <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-accent-500/10">
              <Layers className="h-4 w-4 text-accent-500" />
            </div>
            <span className="text-sm font-bold text-surface-50">
              Daedalus Docs
            </span>
          </div>

          <div className="w-24" />
        </div>
      </header>

      <div className="flex pt-14">
        {/* Sidebar overlay on mobile */}
        {sidebarOpen && (
          <div
            className="fixed inset-0 z-30 bg-black/50 lg:hidden"
            onClick={() => setSidebarOpen(false)}
          />
        )}

        {/* Sidebar */}
        <aside
          className={`fixed top-14 bottom-0 z-30 w-56 overflow-y-auto border-r border-surface-800/50 bg-surface-950 px-3 py-4 transition-transform lg:translate-x-0 ${
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
                  className={`flex items-center gap-2.5 rounded-lg px-3 py-2 text-sm transition-colors ${
                    isActive
                      ? "bg-accent-500/10 text-accent-400 font-medium"
                      : "text-surface-400 hover:bg-surface-900 hover:text-surface-200"
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
          <div className="mx-auto max-w-3xl px-6 py-12 lg:px-12">
            {children}
          </div>
        </main>
      </div>
    </div>
  );
}
