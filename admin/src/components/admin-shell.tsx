"use client";

import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { Fragment, type CSSProperties, useEffect, useRef, useState } from "react";
import {
  IconBooks,
  IconChartBar,
  IconCoins,
  IconUsers,
} from "@tabler/icons-react";
import { LoginButton } from "@/components/login-button";
import { LogoutButton } from "@/components/logout-button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarHeader,
  SidebarInset,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarProvider,
  SidebarGroupLabel,
  SidebarTrigger,
  useSidebar,
} from "@/components/ui/sidebar";
import { ThemeToggle } from "@/components/theme-toggle";
import { SESSION_CHANGED_EVENT } from "@/lib/auth-session";
import { clearSession, getStoredAccessToken, getStoredUser, hasStoredSession } from "@/lib/api";
import { getBreadcrumbs, getNavigationForUser, isRouteActive } from "@/lib/navigation.mjs";
import { isPublicEntryRoute } from "@/lib/rbac.mjs";
import { readSchoolSwitchState, writeSchoolSwitchState, type SchoolSwitchState } from "@/lib/school-switch-state";
import { getClientSessionSnapshot, syncSessionCookies } from "@/lib/session-state.mjs";

const navIcons: Record<string, typeof IconChartBar> = {
  "/dashboard": IconChartBar,
  "/dashboard/classes": IconBooks,
  "/dashboard/ai-usage": IconCoins,
  "/parents/parent-1": IconUsers,
};
const shellEase = [0.22, 1, 0.36, 1] as const;

function ClassroomHubMark({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 32 32" fill="none" aria-hidden="true" className={className}>
      <path
        d="M7.5 8.25c0-1.52 1.23-2.75 2.75-2.75h6.9c4.72 0 8.55 3.83 8.55 8.55s-3.83 8.55-8.55 8.55H13v4.15h-2.75a2.75 2.75 0 0 1-2.75-2.75V8.25Z"
        fill="currentColor"
      />
      <path
        d="M12.6 10.4h4.05a3.55 3.55 0 1 1 0 7.1H12.6"
        stroke="white"
        strokeWidth="2"
        strokeLinecap="round"
      />
      <path d="m20.45 20.7 3.2 3.2" stroke="white" strokeWidth="2" strokeLinecap="round" />
      <circle cx="24.8" cy="24.8" r="1.55" fill="white" />
    </svg>
  );
}

export function AdminShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const router = useRouter();
  const refreshSessionStateRef = useRef<() => void>(() => {});
  const [hydrated, setHydrated] = useState(false);
  const [currentUser, setCurrentUser] = useState<ReturnType<typeof getStoredUser>>(null);
  const [isLoggedIn, setIsLoggedIn] = useState(false);
  const [schoolSwitchState, setSchoolSwitchState] = useState<SchoolSwitchState | null>(null);
  const breadcrumbs = getBreadcrumbs(pathname, currentUser);
  const isPublicRoute = isPublicEntryRoute(pathname);
  const isDashboardRoot = pathname === "/dashboard";
  const prefersReducedMotion = useReducedMotion();
  const shellHeaderMotion = prefersReducedMotion
    ? { initial: false, animate: { opacity: 1 }, transition: { duration: 0 } }
    : {
        initial: { opacity: 0, y: 10, filter: "blur(10px)" },
        animate: { opacity: 1, y: 0, filter: "blur(0px)" },
        transition: { duration: 0.18, ease: shellEase },
      };
  const sidebarLayoutStyle = {
    "--sidebar-width": "16.75rem",
    "--sidebar-width-icon": "4rem",
  } as CSSProperties;

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
      setSchoolSwitchState(readSchoolSwitchState());
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
      <div className="theme-transition relative min-h-screen bg-background text-foreground">
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
    <SidebarProvider
      defaultOpen
      style={sidebarLayoutStyle}
      className="theme-transition isolate min-h-screen bg-background text-foreground"
    >
      <AdminSidebar
        pathname={pathname}
        currentUser={currentUser}
        hydrated={hydrated}
        isLoggedIn={isLoggedIn}
        schoolSwitchState={schoolSwitchState}
      />

      <SidebarInset className="min-h-screen bg-transparent shadow-none md:m-0 md:rounded-none">
        <div className="sticky top-0 z-40 bg-background/88 px-3 py-1.5 backdrop-blur lg:px-3">
          <div className="flex items-center gap-3">
            <SidebarTrigger className="rounded-xl" />
          </div>
        </div>

        {!isDashboardRoot ? (
          <motion.div
            {...shellHeaderMotion}
            className="relative z-30 px-4 pb-2 pt-3 lg:px-6 lg:pb-0 lg:pt-4"
          >
            <div className="mx-auto max-w-7xl">
              <div className="flex flex-col gap-2">
                <Breadcrumb>
                  <BreadcrumbList className="text-xs">
                    {breadcrumbs.map((item, index) => (
                      <Fragment key={`${item.href}-${item.label}`}>
                        <BreadcrumbItem>
                          {index === breadcrumbs.length - 1 ? (
                            <BreadcrumbPage>{item.label}</BreadcrumbPage>
                          ) : (
                            <BreadcrumbLink render={<Link href={item.href} />}>{item.label}</BreadcrumbLink>
                          )}
                        </BreadcrumbItem>
                        {index < breadcrumbs.length - 1 ? <BreadcrumbSeparator /> : null}
                      </Fragment>
                    ))}
                  </BreadcrumbList>
                </Breadcrumb>
              </div>
            </div>
          </motion.div>
        ) : null}

        <main className="relative z-10 flex-1 px-4 pb-8 pt-2 lg:px-6 lg:pt-2">
          <div className="mx-auto max-w-7xl">
            <AnimatePresence mode="wait" initial={false}>
              <motion.div
                key={pathname ?? "shell"}
                initial={prefersReducedMotion ? false : { opacity: 0, y: 12, filter: "blur(14px)" }}
                animate={prefersReducedMotion ? { opacity: 1 } : { opacity: 1, y: 0, filter: "blur(0px)" }}
                exit={prefersReducedMotion ? { opacity: 1 } : { opacity: 0, y: 10, filter: "blur(10px)" }}
                transition={{ duration: 0.18, ease: shellEase }}
              >
                {children}
              </motion.div>
            </AnimatePresence>
          </div>
        </main>
      </SidebarInset>
    </SidebarProvider>
  );
}

function AdminSidebar({
  pathname,
  currentUser,
  hydrated,
  isLoggedIn,
  schoolSwitchState,
}: {
  pathname: string | null;
  currentUser: ReturnType<typeof getStoredUser>;
  hydrated: boolean;
  isLoggedIn: boolean;
  schoolSwitchState: SchoolSwitchState | null;
}) {
  const router = useRouter();
  const { isMobile, setOpenMobile } = useSidebar();
  const navigationItems = getNavigationForUser(currentUser);
  const schoolChoices =
    hydrated && isLoggedIn && currentUser?.email && schoolSwitchState?.email === currentUser.email
      ? schoolSwitchState.tenantChoices
      : [];
  const canSwitchSchools = schoolChoices.length > 1 && Boolean(currentUser?.tenant_id);
  const selectedSchool =
    schoolChoices.find((tenant) => tenant.tenant_id === currentUser?.tenant_id) ??
    (currentUser?.tenant_id
      ? {
          tenant_id: currentUser.tenant_id,
          tenant_name: currentUser.tenant_name || "Current school",
          tenant_slug: currentUser.tenant_slug || "",
        }
      : null);
  type NavigationItem = (typeof navigationItems)[number];
  const groupedNavigation = navigationItems.reduce<Record<string, NavigationItem[]>>((result, item) => {
    const group = item.group || "Workspace";
    if (!result[group]) {
      result[group] = [];
    }
    result[group].push(item);
    return result;
  }, {});

  function handleNavigate() {
    if (isMobile) {
      setOpenMobile(false);
    }
  }

  function handleSchoolSwitch(nextTenantID: string) {
    if (!currentUser || !canSwitchSchools || nextTenantID === currentUser.tenant_id) {
      return;
    }

    writeSchoolSwitchState({
      email: currentUser.email,
      currentTenantID: nextTenantID,
      tenantChoices: schoolChoices,
    });
    clearSession();
    router.push(`/login?next=${encodeURIComponent(pathname || "/dashboard")}`);
  }

  return (
    <Sidebar
      collapsible="offcanvas"
      className="border-r-0 p-0 [&>[data-slot=sidebar-inner]]:border-r [&>[data-slot=sidebar-inner]]:border-sidebar-border [&>[data-slot=sidebar-inner]]:bg-sidebar"
    >
      <SidebarHeader className="gap-3 px-4 pb-5 pt-4">
        <Link href="/dashboard" onClick={handleNavigate} className="flex items-center gap-3 rounded-xl text-sidebar-foreground transition hover:opacity-90">
          <div className="flex size-9 items-center justify-center rounded-[18px] bg-sidebar-primary text-sidebar-primary-foreground shadow-[inset_0_1px_0_rgba(255,255,255,0.2),0_10px_24px_rgba(15,23,42,0.12)] dark:shadow-[inset_0_1px_0_rgba(255,255,255,0.14),0_12px_28px_rgba(2,8,23,0.4)]">
            <ClassroomHubMark className="size-5" />
          </div>
          <div className="min-w-0">
            <p className="text-base font-medium tracking-tight text-sidebar-foreground">Classroom Hub</p>
            <p className="text-sm text-sidebar-foreground/70">Teacher workspace</p>
          </div>
        </Link>
      </SidebarHeader>

      <SidebarContent className="scrollbar-thin-subtle gap-4 px-0 py-1">
        {Object.entries(groupedNavigation).map(([group, items]) => (
          <SidebarGroup key={group} className="px-3 py-0">
            {Object.keys(groupedNavigation).length > 1 ? (
              <SidebarGroupLabel className="px-3 pb-2 text-[11px] font-medium tracking-[0.08em] text-sidebar-foreground/55">
                {group}
              </SidebarGroupLabel>
            ) : null}
            <SidebarGroupContent>
              <SidebarMenu className="gap-1">
                {items.map((item) => {
                  const Icon = navIcons[item.href] ?? (item.href.startsWith("/parents/") ? IconUsers : IconChartBar);
                  const active = isRouteActive(pathname, item.href);

                  return (
                    <SidebarMenuItem key={item.href}>
                      <SidebarMenuButton
                        render={<Link href={item.href} onClick={handleNavigate} />}
                        isActive={active}
                        tooltip={item.title}
                        size="default"
                        className="rounded-xl px-3 data-[active=true]:bg-sidebar-accent data-[active=true]:text-sidebar-accent-foreground data-[active=true]:shadow-sm hover:bg-sidebar-accent/70 hover:text-sidebar-accent-foreground"
                      >
                        <Icon />
                        <span className="truncate font-medium">{item.title}</span>
                      </SidebarMenuButton>
                    </SidebarMenuItem>
                  );
                })}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        ))}
      </SidebarContent>

      <SidebarFooter className="mt-auto gap-3 border-t border-sidebar-border/80 px-4 py-4">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <p className="truncate text-sm font-semibold text-sidebar-foreground">
              {hydrated ? currentUser?.name || "Guest session" : "Guest session"}
            </p>
            <p className="truncate text-xs text-sidebar-foreground/65">
              {hydrated ? currentUser?.email || "No stored profile" : "No stored profile"}
            </p>
            <p className="mt-2 text-[11px] font-medium uppercase tracking-[0.18em] text-sidebar-foreground/55">
              {hydrated ? currentUser?.role?.replaceAll("_", " ") || "not signed in" : "not signed in"}
            </p>
          </div>
          <ThemeToggle className="rounded-xl border border-sidebar-border bg-background/70 hover:bg-sidebar-accent" />
        </div>
        {canSwitchSchools ? (
          <div className="space-y-2">
            <p className="text-[11px] font-medium uppercase tracking-[0.18em] text-sidebar-foreground/55">School</p>
            <Select value={currentUser?.tenant_id ?? ""} onValueChange={(value) => handleSchoolSwitch(value ?? "")}>
              <SelectTrigger
                aria-label="Switch school"
                className="h-10 w-full rounded-xl border-sidebar-border bg-background/70 text-sidebar-foreground hover:bg-sidebar-accent"
              >
                {selectedSchool ? (
                  <span className="truncate text-left font-medium">{selectedSchool.tenant_name}</span>
                ) : (
                  <SelectValue placeholder="Choose school" />
                )}
              </SelectTrigger>
              <SelectContent align="start">
                {schoolChoices.map((tenant) => (
                  <SelectItem key={tenant.tenant_id} value={tenant.tenant_id}>
                    {tenant.tenant_name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        ) : hydrated && currentUser?.tenant_name ? (
          <p className="truncate text-sm text-sidebar-foreground/70">{currentUser.tenant_name}</p>
        ) : null}

        {hydrated && isLoggedIn ? (
          <LogoutButton />
        ) : (
          <LoginButton />
        )}
      </SidebarFooter>
    </Sidebar>
  );
}
