"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState } from "react";
import { BarChart3, ChevronLeft, Home, Menu, Sparkles, UserRound } from "lucide-react";
import { ThemeToggle } from "@/components/theme-toggle";
import { getCurrentSection, isRouteActive, primaryNavigation } from "@/lib/navigation.mjs";
import { cn } from "@/lib/utils";

const navIcons: Record<string, typeof Home> = {
  "/": Home,
  "/dashboard": BarChart3,
};

export function AdminShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const [mobileOpen, setMobileOpen] = useState(false);
  const section = getCurrentSection(pathname);

  return (
    <div className="min-h-screen bg-[radial-gradient(circle_at_top_left,_rgba(14,165,233,0.16),transparent_24%),radial-gradient(circle_at_85%_12%,_rgba(249,115,22,0.16),transparent_18%),linear-gradient(180deg,#fffef7_0%,#f5fbff_45%,#eef8f5_100%)] text-slate-900 dark:bg-[radial-gradient(circle_at_top_left,_rgba(56,189,248,0.14),transparent_24%),radial-gradient(circle_at_85%_12%,_rgba(251,191,36,0.12),transparent_18%),linear-gradient(180deg,#07111c_0%,#0c1724_45%,#101926_100%)] dark:text-slate-100">
      <div className="mx-auto flex min-h-screen max-w-[1600px]">
        <aside className="sticky top-0 hidden h-screen w-80 shrink-0 border-r border-white/70 bg-white/72 px-6 py-6 backdrop-blur dark:border-white/10 dark:bg-slate-950/58 lg:flex lg:flex-col">
          <SidebarContent pathname={pathname} />
        </aside>

        <div className="flex min-h-screen min-w-0 flex-1 flex-col">
          <div className="sticky top-0 z-40 border-b border-white/70 bg-white/78 px-4 py-3 backdrop-blur dark:border-white/10 dark:bg-slate-950/64 lg:hidden">
            <div className="flex items-center justify-between gap-3">
              <button
                type="button"
                onClick={() => setMobileOpen((open) => !open)}
                className="inline-flex size-11 items-center justify-center rounded-2xl border border-slate-200/80 bg-white text-slate-900 shadow-sm dark:border-white/10 dark:bg-slate-900 dark:text-slate-100"
                aria-label="Toggle sidebar"
              >
                {mobileOpen ? <ChevronLeft className="size-5" /> : <Menu className="size-5" />}
              </button>
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-semibold text-slate-950 dark:text-white">{section.title}</p>
                <p className="truncate text-xs text-slate-500 dark:text-slate-400">{section.eyebrow}</p>
              </div>
              <ThemeToggle />
            </div>
            {mobileOpen ? (
              <div className="pt-4">
                <SidebarContent pathname={pathname} compact onNavigate={() => setMobileOpen(false)} />
              </div>
            ) : null}
          </div>

          <header className="px-4 pb-4 pt-5 lg:px-8 lg:pb-0 lg:pt-8">
            <div className="rounded-[28px] border border-white/75 bg-white/72 p-6 shadow-[0_18px_60px_rgba(15,23,42,0.06)] backdrop-blur dark:border-white/10 dark:bg-slate-950/55 dark:shadow-[0_24px_80px_rgba(2,8,23,0.36)]">
              <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                <div className="space-y-3">
                  <p className="text-xs font-semibold uppercase tracking-[0.24em] text-sky-700 dark:text-sky-300">{section.eyebrow}</p>
                  <div className="space-y-2">
                    <h1 className="text-3xl font-semibold tracking-tight text-slate-950 dark:text-white">{section.title}</h1>
                    <p className="max-w-2xl text-sm leading-6 text-slate-600 dark:text-slate-300">{section.description}</p>
                  </div>
                </div>
                <div className="hidden lg:block">
                  <ThemeToggle />
                </div>
              </div>
            </div>
          </header>

          <main className="flex-1 px-4 pb-8 lg:px-8 lg:pt-6">
            <div className="mx-auto max-w-7xl">{children}</div>
          </main>
        </div>
      </div>
    </div>
  );
}

function SidebarContent({
  pathname,
  compact = false,
  onNavigate,
}: {
  pathname: string | null;
  compact?: boolean;
  onNavigate?: () => void;
}) {
  return (
    <div className="flex h-full flex-col gap-6">
      <div className="space-y-4">
        <Link
          href="/"
          onClick={onNavigate}
          className="block rounded-[28px] border border-slate-200/70 bg-slate-950 p-5 text-slate-50 shadow-[0_16px_40px_rgba(15,23,42,0.18)] dark:border-white/10 dark:bg-slate-900"
        >
          <div className="flex items-center gap-3">
            <div className="flex size-12 items-center justify-center rounded-2xl bg-white/10">
              <Sparkles className="size-6 text-sky-300" />
            </div>
            <div>
              <p className="text-xs font-semibold uppercase tracking-[0.24em] text-sky-200">P&AI Bot</p>
              <p className="mt-1 text-lg font-semibold text-white">Admin cockpit</p>
            </div>
          </div>
          <p className="mt-4 text-sm leading-6 text-slate-300">
            Day 16 foundation for teacher operations, student review, and rollout visibility.
          </p>
        </Link>

        <nav className="space-y-2">
          {primaryNavigation.map((item) => {
            const Icon = navIcons[item.href] ?? Home;
            const active = isRouteActive(pathname, item.href);
            return (
              <Link
                key={item.href}
                href={item.href}
                onClick={onNavigate}
                className={cn(
                  "flex items-center gap-3 rounded-2xl border px-4 py-3 text-sm transition",
                  active
                    ? "border-sky-300 bg-sky-100 text-sky-950 shadow-[0_10px_30px_rgba(14,165,233,0.14)] dark:border-sky-400/40 dark:bg-sky-400/12 dark:text-sky-100"
                    : "border-slate-200/70 bg-white/85 text-slate-700 hover:border-slate-300 hover:bg-white dark:border-white/10 dark:bg-slate-950/45 dark:text-slate-200 dark:hover:border-white/20 dark:hover:bg-slate-950/65",
                )}
              >
                <Icon className="size-4" />
                <div className="min-w-0">
                  <p className="font-medium">{item.title}</p>
                  <p className="truncate text-xs text-slate-500 dark:text-slate-400">{item.description}</p>
                </div>
              </Link>
            );
          })}
        </nav>
      </div>

      <div className="space-y-3">
        <p className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500 dark:text-slate-400">Focus</p>
        <div className="rounded-[24px] border border-slate-200/70 bg-white/85 p-4 dark:border-white/10 dark:bg-slate-950/45">
          <div className="flex items-center gap-3">
            <div className="flex size-10 items-center justify-center rounded-2xl bg-amber-100 text-amber-700 dark:bg-amber-300/15 dark:text-amber-100">
              <UserRound className="size-5" />
            </div>
            <div>
              <p className="text-sm font-medium text-slate-900 dark:text-slate-100">
                {pathname?.startsWith("/students/") ? "Student profile open" : "Teacher workflow active"}
              </p>
              <p className="text-xs text-slate-500 dark:text-slate-400">
                {pathname?.startsWith("/students/")
                  ? "Use this view to review struggles before sending the next intervention."
                  : "Scan class mastery, then drill into a learner profile when intervention is needed."}
              </p>
            </div>
          </div>
        </div>
      </div>

      {!compact ? (
        <div className="mt-auto rounded-[24px] border border-slate-200/70 bg-white/85 p-4 dark:border-white/10 dark:bg-slate-950/45">
          <p className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500 dark:text-slate-400">Current scope</p>
          <ul className="mt-3 space-y-2 text-sm text-slate-600 dark:text-slate-300">
            <li>Teacher dashboard for class-wide mastery and nudges</li>
            <li>Student detail view for progress and recent tutoring activity</li>
            <li>Shared sidebar shell for the remaining Day 16 pages</li>
          </ul>
        </div>
      ) : null}
    </div>
  );
}
