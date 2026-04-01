"use client";

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
import { Badge } from "@/components/ui/badge";
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
import { getStoredAccessToken, getStoredUser, hasStoredSession } from "@/lib/api";
import { getBreadcrumbs, getCurrentSection, getNavigationForUser, isRouteActive } from "@/lib/navigation.mjs";
import { isPublicEntryRoute } from "@/lib/rbac.mjs";
import { getClientSessionSnapshot, syncSessionCookies } from "@/lib/session-state.mjs";

const navIcons: Record<string, typeof IconChartBar> = {
  "/dashboard": IconChartBar,
  "/dashboard/classes": IconBooks,
  "/dashboard/ai-usage": IconCoins,
  "/parents/parent-1": IconUsers,
};

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
  const section = getCurrentSection(pathname);
  const breadcrumbs = getBreadcrumbs(pathname, currentUser);
  const isPublicRoute = isPublicEntryRoute(pathname);
  const isDashboardRoot = pathname === "/dashboard";
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
      <div className="relative min-h-screen bg-background text-foreground">
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
      className="isolate min-h-screen bg-background text-foreground"
    >
      <AdminSidebar
        pathname={pathname}
        currentUser={currentUser}
        hydrated={hydrated}
        isLoggedIn={isLoggedIn}
      />

      <SidebarInset className="min-h-screen bg-transparent shadow-none md:m-0 md:rounded-none">
        <div className="sticky top-0 z-40 bg-background/88 px-4 py-2 backdrop-blur lg:px-6">
          <div className="mx-auto flex max-w-7xl items-center gap-3">
            <div className="flex min-w-0 items-center gap-3">
              <SidebarTrigger className="rounded-xl" />
              {!isDashboardRoot ? (
                <div className="min-w-0">
                  <p className="truncate text-sm font-semibold text-foreground">{section.title}</p>
                  <p className="truncate text-xs text-muted-foreground">{section.eyebrow}</p>
                </div>
              ) : null}
            </div>
          </div>
        </div>

        {!isDashboardRoot ? (
          <div className="relative z-30 px-4 pb-3 pt-4 lg:px-6 lg:pb-0 lg:pt-6">
            <div className="mx-auto max-w-7xl">
              <div className="flex flex-col gap-3">
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
                <div className="flex flex-col gap-2">
                  <h1 className="text-3xl font-semibold tracking-tight text-foreground">{section.title}</h1>
                  <p className="max-w-2xl text-sm leading-6 text-muted-foreground">{section.description}</p>
                </div>
              </div>
            </div>
          </div>
        ) : null}

        <main className="relative z-10 flex-1 px-4 pb-8 pt-4 lg:px-6 lg:pt-4">
          <div className="mx-auto max-w-7xl">{children}</div>
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
}: {
  pathname: string | null;
  currentUser: ReturnType<typeof getStoredUser>;
  hydrated: boolean;
  isLoggedIn: boolean;
}) {
  const { isMobile, setOpenMobile } = useSidebar();
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

  function handleNavigate() {
    if (isMobile) {
      setOpenMobile(false);
    }
  }

  return (
    <Sidebar
      collapsible="offcanvas"
      className="border-r-0 p-0 [&>[data-slot=sidebar-inner]]:border-r [&>[data-slot=sidebar-inner]]:border-sidebar-border [&>[data-slot=sidebar-inner]]:bg-sidebar"
    >
      <SidebarHeader className="gap-4 px-4 pb-5 pt-6">
        <Badge variant="outline" className="w-fit rounded-full border-sidebar-border/80 bg-white/55 text-sidebar-foreground dark:bg-white/5">
          P&AI Bot
        </Badge>
        <Link href="/dashboard" onClick={handleNavigate} className="flex items-start gap-3 rounded-xl transition hover:opacity-90">
          <div className="mt-0.5 flex size-10 items-center justify-center rounded-[18px] bg-sidebar-primary text-sidebar-primary-foreground shadow-[inset_0_1px_0_rgba(255,255,255,0.2),0_10px_24px_rgba(15,23,42,0.12)] dark:shadow-[inset_0_1px_0_rgba(255,255,255,0.14),0_12px_28px_rgba(2,8,23,0.4)]">
            <ClassroomHubMark className="size-5" />
          </div>
          <div className="min-w-0">
            <p className="text-base font-medium tracking-tight text-sidebar-foreground">Classroom Hub</p>
            <p className="text-sm text-sidebar-foreground/70">Teacher workspace</p>
          </div>
        </Link>
      </SidebarHeader>

      <SidebarContent className="scrollbar-thin-subtle gap-4 px-0 py-0">
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
                        className="rounded-xl px-3 data-[active=true]:bg-white/80 data-[active=true]:text-sidebar-foreground data-[active=true]:shadow-sm hover:bg-white/60 dark:data-[active=true]:bg-white/10 dark:hover:bg-white/5"
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
          <ThemeToggle className="rounded-xl border border-sidebar-border/80 bg-white/50 dark:bg-white/5" />
        </div>
        {hydrated && currentUser?.tenant_name ? (
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
