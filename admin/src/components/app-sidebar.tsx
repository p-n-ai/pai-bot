"use client";

import { Buildings } from "@phosphor-icons/react";
import { IconBooks, IconChartBar, IconCoins, IconUsers } from "@tabler/icons-react";
import Link from "next/link";
import { AccountSettingsDialog } from "@/components/account/account-settings-dialog";
import { LoginButton } from "@/components/login-button";
import { LogoutButton } from "@/components/logout-button";
import { ThemeToggle } from "@/components/theme-toggle";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from "@/components/ui/sidebar";
import type { AuthUser } from "@/lib/api";
import { getNavigationForUser, isRouteActive } from "@/lib/navigation.mjs";
import { type SchoolSwitchState } from "@/lib/school-switch-state";
import { useAppStore } from "@/stores/app-store";

export const APP_SIDEBAR_COOKIE_NAME = "sidebar_state";

const navIcons: Record<string, typeof IconChartBar> = {
  "/dashboard": IconChartBar,
  "/dashboard/classes": IconBooks,
  "/dashboard/ai-usage": IconCoins,
  "/parents/parent-1": IconUsers,
};

export function AppSidebar({
  pathname,
  hydrated,
  hasMounted,
  initialCurrentUser,
  initialSchoolSwitchState,
}: {
  pathname: string | null;
  hydrated: boolean;
  hasMounted: boolean;
  initialCurrentUser: AuthUser | null;
  initialSchoolSwitchState: SchoolSwitchState | null;
}) {
  const { isMobile, setOpenMobile } = useSidebar();
  const storeCurrentUser = useAppStore((state) => state.currentUser);
  const storeIsLoggedIn = useAppStore((state) => state.isLoggedIn);
  const storeSchoolSwitchState = useAppStore((state) => state.schoolSwitchState);
  const currentUser = hasMounted ? storeCurrentUser : initialCurrentUser;
  const isLoggedIn = hasMounted ? storeIsLoggedIn : Boolean(initialCurrentUser);
  const schoolSwitchState = hasMounted ? storeSchoolSwitchState : initialSchoolSwitchState;
  const schoolChoices = schoolSwitchState?.tenantChoices ?? [];
  const navigationItems = getNavigationForUser(currentUser);
  const sessionReady = hasMounted ? hydrated : Boolean(initialCurrentUser);

  const groupedNavigation = navigationItems.reduce<Record<string, (typeof navigationItems)[number][]>>((result, item) => {
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
      <SidebarHeader className="gap-3 px-4 pb-4 pt-4">
        {sessionReady && currentUser?.tenant_name ? (
          <Link
            href="/dashboard"
            onClick={handleNavigate}
            className="flex items-center gap-3 rounded-xl border border-sidebar-border bg-background/70 p-3 text-sidebar-foreground shadow-none transition hover:bg-sidebar-accent/50"
          >
            <div className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-muted text-muted-foreground">
              <Buildings className="size-5" weight="duotone" />
            </div>
            <div className="min-w-0">
              <p className="truncate text-base font-semibold tracking-tight text-sidebar-foreground">{currentUser.tenant_name}</p>
              {schoolChoices.length > 1 ? (
                <p className="text-xs text-sidebar-foreground/65">Switch school from settings</p>
              ) : null}
            </div>
          </Link>
        ) : (
          <Link
            href="/dashboard"
            onClick={handleNavigate}
            className="flex items-center gap-3 rounded-xl border border-sidebar-border bg-background/70 p-3 text-sidebar-foreground shadow-none transition hover:bg-sidebar-accent/50"
          >
            <div className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-muted text-muted-foreground">
              <Buildings className="size-5" weight="duotone" />
            </div>
            <div className="min-w-0">
              <p className="text-base font-medium tracking-tight text-sidebar-foreground">Classroom Hub</p>
              <p className="text-sm text-sidebar-foreground/70">Teacher workspace</p>
            </div>
          </Link>
        )}
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
              {sessionReady ? currentUser?.name || "Guest session" : "Guest session"}
            </p>
            <p className="truncate text-xs text-sidebar-foreground/65">
              {sessionReady ? currentUser?.email || "No stored profile" : "No stored profile"}
            </p>
            <p className="mt-2 text-[11px] font-medium uppercase tracking-[0.18em] text-sidebar-foreground/55">
              {sessionReady ? currentUser?.role?.replaceAll("_", " ") || "not signed in" : "not signed in"}
            </p>
          </div>
          <ThemeToggle className="rounded-xl border border-sidebar-border bg-background/70 hover:bg-sidebar-accent" />
        </div>
        {sessionReady && isLoggedIn ? <AccountSettingsDialog currentUser={currentUser} schoolChoices={schoolChoices} nextPath={pathname} /> : null}
        {sessionReady && isLoggedIn ? <LogoutButton /> : <LoginButton />}
      </SidebarFooter>
    </Sidebar>
  );
}
