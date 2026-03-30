"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { BarChart3, ChevronDown, ChevronLeft, Coins, Home, Menu, Sparkles, UserRound, Users, LibraryBig } from "lucide-react";
import { LoginButton } from "@/components/login-button";
import { LogoutButton } from "@/components/logout-button";
import { Button } from "@/components/ui/button";
import { ThemeToggle } from "@/components/theme-toggle";
import { SESSION_CHANGED_EVENT } from "@/lib/auth-session";
import { getStoredAccessToken, getStoredUser, hasStoredSession } from "@/lib/api";
import { getBreadcrumbs, getCurrentSection, getNavigationForUser, isRouteActive } from "@/lib/navigation.mjs";
import { isPublicEntryRoute } from "@/lib/rbac.mjs";
import { getClientSessionSnapshot, syncSessionCookies } from "@/lib/session-state.mjs";
import { cn } from "@/lib/utils";

const navIcons: Record<string, typeof Home> = {
  "/": Home,
  "/dashboard": BarChart3,
  "/dashboard/classes": LibraryBig,
  "/dashboard/metrics": BarChart3,
  "/dashboard/ai-usage": Coins,
  "/parents/parent-1": Users,
};

export function AdminShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const refreshSessionStateRef = useRef<() => void>(() => {});
  const [mobileOpen, setMobileOpen] = useState(false);
  const [hydrated, setHydrated] = useState(false);
  const [currentUser, setCurrentUser] = useState<ReturnType<typeof getStoredUser>>(null);
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const section = getCurrentSection(pathname);
  const breadcrumbs = getBreadcrumbs(pathname, currentUser);
  const isPublicRoute = isPublicEntryRoute(pathname);

  useEffect(() => {
    refreshSessionStateRef.current = () => {
      const snapshot = getClientSessionSnapshot({
        accessToken: getStoredAccessToken(),
        user: getStoredUser(),
      });
      const syncedCookies = syncSessionCookies({
        accessToken: getStoredAccessToken(),
        user: snapshot.currentUser,
        cookieString: document.cookie,
        writeCookie(value: string) {
          document.cookie = value;
        },
      });

      setCurrentUser(snapshot.currentUser);
      setIsLoggedIn(snapshot.isLoggedIn && hasStoredSession());
      setHydrated(true);

      if (syncedCookies) {
        router.refresh();
      }
    };
  }, [router]);

  useEffect(() => {
    function handleSessionChange() {
      refreshSessionStateRef.current();
    }

    handleSessionChange();
    window.addEventListener(SESSION_CHANGED_EVENT, handleSessionChange);
    window.addEventListener("storage", handleSessionChange);

    return () => {
      window.removeEventListener(SESSION_CHANGED_EVENT, handleSessionChange);
      window.removeEventListener("storage", handleSessionChange);
    };
  }, []);

  useEffect(() => {
    refreshSessionStateRef.current();
  }, [pathname]);

  if (isPublicRoute) {
    return (
      <div className="relative min-h-screen bg-[radial-gradient(circle_at_top_left,rgba(15,23,42,0.08),transparent_28%),radial-gradient(circle_at_85%_12%,rgba(71,85,105,0.08),transparent_20%),linear-gradient(180deg,#fcfcfb_0%,#f4f4f2_48%,#efefec_100%)] text-slate-900 dark:bg-[radial-gradient(circle_at_top_left,rgba(255,255,255,0.05),transparent_28%),radial-gradient(circle_at_85%_12%,rgba(148,163,184,0.06),transparent_20%),linear-gradient(180deg,#0b0d10_0%,#101419_48%,#14181d_100%)] dark:text-slate-100">
        <div className="pointer-events-none fixed right-3 top-3 z-20 lg:right-6 lg:top-3.5">
          <div className="pointer-events-auto">
            <ThemeToggle />
          </div>
        </div>
        <main className="w-full">{children}</main>
      </div>
    );
  }

  return (
    <div className="isolate min-h-screen bg-[radial-gradient(circle_at_top_left,rgba(14,165,233,0.16),transparent_24%),radial-gradient(circle_at_85%_12%,rgba(249,115,22,0.16),transparent_18%),linear-gradient(180deg,#fffef7_0%,#f5fbff_45%,#eef8f5_100%)] text-slate-900 dark:bg-[radial-gradient(circle_at_top_left,rgba(56,189,248,0.14),transparent_24%),radial-gradient(circle_at_85%_12%,rgba(251,191,36,0.12),transparent_18%),linear-gradient(180deg,#07111c_0%,#0c1724_45%,#101926_100%)] dark:text-slate-100">
      <div className="mx-auto flex min-h-screen max-w-[1600px]">
        <aside className="sticky top-0 hidden h-screen w-80 shrink-0 overflow-hidden border-r border-white/70 bg-white/72 px-6 py-6 backdrop-blur dark:border-white/10 dark:bg-slate-950/58 lg:flex lg:flex-col">
          <div className="scrollbar-thin-subtle min-h-0 flex-1 overflow-y-auto pr-2">
            <SidebarContent pathname={pathname} currentUser={currentUser} />
          </div>
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
              <div className="flex items-center gap-2">
                <ThemeToggle />
                <SessionControls currentUser={currentUser} isLoggedIn={isLoggedIn} hydrated={hydrated} />
              </div>
            </div>
            <div
              className={cn(
                "grid transition-[grid-template-rows,opacity,transform] duration-300 ease-out",
                mobileOpen ? "grid-rows-[1fr] pt-4 opacity-100" : "grid-rows-[0fr] opacity-0",
              )}
              aria-hidden={!mobileOpen}
            >
              <div className="overflow-hidden">
                <div
                  className={cn(
                    "scrollbar-thin-subtle max-h-[calc(100vh-7rem)] overflow-y-auto pr-1 transition duration-300 ease-out",
                    mobileOpen ? "translate-y-0 scale-100" : "-translate-y-2 scale-[0.98]",
                  )}
                >
                  <SidebarContent pathname={pathname} currentUser={currentUser} compact onNavigate={() => setMobileOpen(false)} />
                </div>
              </div>
            </div>
          </div>

          <div className="relative z-30 px-4 pb-4 pt-5 lg:px-8 lg:pb-0 lg:pt-8">
            <div className="relative overflow-visible rounded-[28px] border border-white/75 bg-white/72 p-6 shadow-[0_18px_60px_rgba(15,23,42,0.06)] backdrop-blur dark:border-white/10 dark:bg-slate-950/55 dark:shadow-[0_24px_80px_rgba(2,8,23,0.36)]">
              <div className="flex flex-col gap-4 lg:flex-row lg:items-start lg:justify-between">
                <div className="space-y-3">
                  <p className="text-xs font-semibold uppercase tracking-[0.24em] text-sky-700 dark:text-sky-300">{section.eyebrow}</p>
                  <div className="flex flex-wrap items-center gap-2 text-xs font-medium text-slate-500 dark:text-slate-400">
                    {breadcrumbs.map((item, index) => (
                      <div key={`${item.href}-${item.label}`} className="flex items-center gap-2">
                        {index > 0 ? <span>/</span> : null}
                        {index === breadcrumbs.length - 1 ? (
                          <span className="text-slate-700 dark:text-slate-200">{item.label}</span>
                        ) : (
                          <Link href={item.href} className="hover:text-sky-700 dark:hover:text-sky-300">
                            {item.label}
                          </Link>
                        )}
                      </div>
                    ))}
                  </div>
                  <div className="space-y-2">
                    <h1 className="text-3xl font-semibold tracking-tight text-slate-950 dark:text-white">{section.title}</h1>
                    <p className="max-w-2xl text-sm leading-6 text-slate-600 dark:text-slate-300">{section.description}</p>
                  </div>
                </div>
                <div className="hidden lg:flex lg:items-center lg:gap-3">
                  <ThemeToggle />
                  <SessionControls currentUser={currentUser} isLoggedIn={isLoggedIn} hydrated={hydrated} />
                </div>
              </div>
            </div>
          </div>

          <main className="relative z-10 flex-1 px-4 pb-8 lg:px-8 lg:pt-6">
            <div className="mx-auto max-w-7xl">{children}</div>
          </main>
        </div>
      </div>
    </div>
  );
}

function SidebarContent({
  pathname,
  currentUser,
  compact = false,
  onNavigate,
}: {
  pathname: string | null;
  currentUser: ReturnType<typeof getStoredUser>;
  compact?: boolean;
  onNavigate?: () => void;
}) {
  const navigationItems = getNavigationForUser(currentUser);
  type NavigationItem = (typeof navigationItems)[number];
  const groupedNavigation = navigationItems.reduce<Record<string, NavigationItem[]>>((result, item) => {
    const group = item.group || "Workspace";
    if (!result[group]) {
      result[group] = [];
    }
    result[group].push(item);
    return result;
  }, {});

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
            Teacher operations, student review, and parent-facing progress snapshots in one workspace.
          </p>
        </Link>

        <nav className="space-y-4">
          {Object.entries(groupedNavigation).map(([group, items]) => (
            <div key={group} className="space-y-2">
              <p className="px-1 text-[11px] font-semibold uppercase tracking-[0.2em] text-slate-500 dark:text-slate-400">{group}</p>
              {items.map((item) => {
                const Icon = navIcons[item.href] ?? (item.href.startsWith("/parents/") ? Users : Home);
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
            </div>
          ))}
        </nav>
      </div>

      <div className="space-y-3">
        <p className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500 dark:text-slate-400">Focus</p>
        <div className="rounded-[24px] border border-slate-200/70 bg-white/85 p-4 dark:border-white/10 dark:bg-slate-950/45">
          <div className="flex items-center gap-3">
            <div className="flex size-10 shrink-0 items-center justify-center rounded-full bg-amber-100 text-amber-700 dark:bg-amber-300/15 dark:text-amber-100">
              <UserRound className="size-5" />
            </div>
            <div>
              <p className="text-sm font-medium text-slate-900 dark:text-slate-100">
                {pathname?.startsWith("/students/")
                  ? "Student profile open"
                  : pathname?.startsWith("/parents/")
                    ? "Parent summary open"
                    : "Teacher workflow active"}
              </p>
              <p className="text-xs text-slate-500 dark:text-slate-400">
                {pathname?.startsWith("/students/")
                  ? "Use this view to review struggles before sending the next intervention."
                  : pathname?.startsWith("/parents/")
                    ? "Translate learner progress into a concise weekly summary a parent can act on."
                    : pathname?.startsWith("/dashboard/ai-usage")
                      ? "Review model traffic and token totals before budget controls and provider tuning."
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
            <li>Parent view for weekly momentum, mastery, and encouragement</li>
            <li>AI usage analytics by provider and model</li>
          </ul>
        </div>
      ) : null}
    </div>
  );
}

function SessionControls({
  currentUser,
  isLoggedIn,
  hydrated,
}: {
  currentUser: ReturnType<typeof getStoredUser>;
  isLoggedIn: boolean;
  hydrated: boolean;
}) {
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    function handlePointerDown(event: MouseEvent) {
      if (!containerRef.current?.contains(event.target as Node)) {
        setOpen(false);
      }
    }

    function handleEscape(event: KeyboardEvent) {
      if (event.key === "Escape") {
        setOpen(false);
      }
    }

    document.addEventListener("mousedown", handlePointerDown);
    document.addEventListener("keydown", handleEscape);
    return () => {
      document.removeEventListener("mousedown", handlePointerDown);
      document.removeEventListener("keydown", handleEscape);
    };
  }, []);

  return (
    <div className="relative z-120" ref={containerRef}>
      <Button
        type="button"
        variant="outline"
        size="sm"
        onClick={() => setOpen((value) => !value)}
        aria-haspopup="menu"
        aria-expanded={open}
        className="rounded-full border-white/50 bg-white/75 px-3 text-slate-700 shadow-[0_12px_30px_rgba(15,23,42,0.08)] backdrop-blur hover:bg-white dark:border-white/10 dark:bg-slate-950/75 dark:text-slate-100 dark:hover:bg-slate-900"
      >
        <UserRound className="size-4" />
        <span className="hidden max-w-[140px] truncate sm:inline">
          {hydrated ? currentUser?.name || "Account" : "Account"}
        </span>
        <ChevronDown className={cn("size-4 transition-transform", open ? "rotate-180" : "")} />
      </Button>

      <div
        className={cn(
          "absolute right-0 top-[calc(100%+0.65rem)] z-160 w-72 origin-top-right rounded-[24px] border border-white/75 bg-white/95 p-4 shadow-[0_24px_80px_rgba(15,23,42,0.18)] backdrop-blur transition duration-200 ease-out dark:border-white/10 dark:bg-slate-950/95 dark:shadow-[0_24px_80px_rgba(2,8,23,0.55)]",
          open
            ? "pointer-events-auto translate-y-0 scale-100 opacity-100"
            : "pointer-events-none -translate-y-2 scale-95 opacity-0",
        )}
        role="menu"
        aria-hidden={!open}
      >
          <div className="flex items-start gap-3">
            <div className="flex size-11 shrink-0 items-center justify-center rounded-2xl bg-sky-100 text-sky-700 dark:bg-sky-400/15 dark:text-sky-200">
              <UserRound className="size-5" />
            </div>
            <div className="min-w-0 space-y-1">
              <p className="truncate text-sm font-semibold text-slate-950 dark:text-slate-100">
                {hydrated ? currentUser?.name || "Guest session" : "Guest session"}
              </p>
              <p className="truncate text-xs text-slate-500 dark:text-slate-400">
                {hydrated ? currentUser?.email || "No stored profile" : "No stored profile"}
              </p>
              <p className="text-xs uppercase tracking-[0.18em] text-slate-400 dark:text-slate-500">
                {hydrated ? currentUser?.role?.replaceAll("_", " ") || "not signed in" : "not signed in"}
              </p>
              {hydrated && currentUser?.tenant_name ? (
                <div className="pt-2">
                  <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-slate-400 dark:text-slate-500">
                    Tenant
                  </p>
                  <p className="truncate text-sm font-medium text-slate-700 dark:text-slate-200">
                    {currentUser.tenant_name}
                  </p>
                </div>
              ) : null}
            </div>
          </div>
          <div className="mt-4 border-t border-slate-200/80 pt-4 dark:border-white/10">
            {hydrated && isLoggedIn ? (
              <LogoutButton />
            ) : (
              <LoginButton onClick={() => setOpen(false)} />
            )}
          </div>
      </div>
    </div>
  );
}
