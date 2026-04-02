"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Buildings } from "@phosphor-icons/react";
import { AnimatePresence, motion } from "framer-motion";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { startTransition, useState } from "react";
import { toast } from "sonner";
import { IconBooks, IconChartBar, IconCoins, IconUsers } from "@tabler/icons-react";
import { LoginButton } from "@/components/login-button";
import { LogoutButton } from "@/components/logout-button";
import { ThemeToggle } from "@/components/theme-toggle";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger } from "@/components/ui/select";
import type { AuthUser } from "@/lib/api";
import { persistSession, switchTenantSession } from "@/lib/api";
import { fetchDashboardProgress, fetchPreviewDashboardProgress, getDashboardProgressQueryKey } from "@/lib/dashboard-progress-query";
import { getNavigationForUser, isRouteActive } from "@/lib/navigation.mjs";
import { type SchoolSwitchState, writeSchoolSwitchState } from "@/lib/school-switch-state";
import { useAppStore } from "@/stores/app-store";

export const APP_SIDEBAR_COOKIE_NAME = "sidebar_state";

const navIcons: Record<string, typeof IconChartBar> = {
  "/dashboard": IconChartBar,
  "/dashboard/classes": IconBooks,
  "/dashboard/ai-usage": IconCoins,
  "/parents/parent-1": IconUsers,
};
const shellEase = [0.22, 1, 0.36, 1] as const;

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
  const router = useRouter();
  const queryClient = useQueryClient();
  const { isMobile, setOpenMobile } = useSidebar();
  const storeCurrentUser = useAppStore((state) => state.currentUser);
  const storeIsLoggedIn = useAppStore((state) => state.isLoggedIn);
  const storeSchoolSwitchState = useAppStore((state) => state.schoolSwitchState);
  const pendingTenantID = useAppStore((state) => state.pendingTenantID);
  const isSwitchingTenant = useAppStore((state) => state.isSwitchingTenant);
  const setSchoolSwitchState = useAppStore((state) => state.setSchoolSwitchState);
  const startTenantSwitch = useAppStore((state) => state.startTenantSwitch);
  const finishTenantSwitch = useAppStore((state) => state.finishTenantSwitch);
  const currentUser = hasMounted ? storeCurrentUser : initialCurrentUser;
  const isLoggedIn = hasMounted ? storeIsLoggedIn : Boolean(initialCurrentUser);
  const schoolSwitchState = hasMounted ? storeSchoolSwitchState : initialSchoolSwitchState;
  const navigationItems = getNavigationForUser(currentUser);
  const schoolChoices =
    (hasMounted ? hydrated : true) && isLoggedIn && currentUser?.email && schoolSwitchState?.email === currentUser.email
      ? schoolSwitchState.tenantChoices
      : [];
  const canSwitchSchools = schoolChoices.length > 1 && Boolean(currentUser?.tenant_id);
  const sessionReady = hasMounted ? hydrated : Boolean(initialCurrentUser);
  const [schoolSwitchError, setSchoolSwitchError] = useState("");
  const [schoolSwitchOverlayLabel, setSchoolSwitchOverlayLabel] = useState("");
  const [switchTargetTenantID, setSwitchTargetTenantID] = useState("");
  const [switchPassword, setSwitchPassword] = useState("");
  const selectedTenantID = pendingTenantID ?? currentUser?.tenant_id ?? "";
  const selectedSchool =
    schoolChoices.find((tenant) => tenant.tenant_id === selectedTenantID) ??
    (currentUser?.tenant_id
      ? {
          tenant_id: selectedTenantID,
          tenant_name: currentUser.tenant_name || "Current school",
          tenant_slug: currentUser.tenant_slug || "",
        }
      : null);

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

  const switchSchoolMutation = useMutation({
    mutationFn: async ({ nextTenantID, password }: { nextTenantID: string; password: string }) => {
      if (!currentUser) {
        throw new Error("A signed-in session is required to switch schools");
      }

      const nextState = {
        email: currentUser.email,
        currentTenantID: nextTenantID,
        tenantChoices: schoolChoices,
      };
      writeSchoolSwitchState(nextState);
      setSchoolSwitchState(nextState);
      startTenantSwitch(nextTenantID, nextState);
      return switchTenantSession(nextTenantID, password);
    },
    onMutate: () => {
      setSchoolSwitchError("");
      toast.loading("Switching school...", { id: "school-switch" });
    },
    onSuccess: async (nextSession) => {
      persistSession(nextSession);
      setSwitchPassword("");
      setSwitchTargetTenantID("");
      if (isMobile) {
        setOpenMobile(false);
      }
      setSchoolSwitchOverlayLabel(nextSession.user.tenant_name || "your school");
      const dashboardQueryKey = getDashboardProgressQueryKey(nextSession.user.tenant_id);
      try {
        await queryClient.ensureQueryData({
          queryKey: dashboardQueryKey,
          queryFn: () => fetchDashboardProgress(nextSession.user.tenant_id),
        });
      } catch {
        queryClient.setQueryData(dashboardQueryKey, await fetchPreviewDashboardProgress());
      }
      toast.success(`School changed to ${nextSession.user.tenant_name}.`, { id: "school-switch" });
      await new Promise((resolve) => window.setTimeout(resolve, 500));
      startTransition(() => {
        router.replace("/dashboard");
      });
    },
    onError: (error) => {
      finishTenantSwitch();
      setSwitchPassword("");
      const message = error instanceof Error ? error.message : "Couldn't switch schools right now.";
      setSchoolSwitchError(message);
      toast.error(message, { id: "school-switch" });
    },
    onSettled: () => {
      window.setTimeout(() => setSchoolSwitchOverlayLabel(""), 60);
    },
  });

  function handleSchoolSwitch(nextTenantID: string) {
    if (!currentUser || !canSwitchSchools || nextTenantID === currentUser.tenant_id) {
      return;
    }

    setSchoolSwitchError("");
    setSwitchPassword("");
    setSwitchTargetTenantID(nextTenantID);
  }

  const switchTargetSchool = schoolChoices.find((tenant) => tenant.tenant_id === switchTargetTenantID) ?? null;

  function closeSwitchDialog() {
    if (switchSchoolMutation.isPending) {
      return;
    }

    setSwitchTargetTenantID("");
    setSwitchPassword("");
  }

  function confirmSchoolSwitch() {
    if (!switchTargetTenantID || !switchPassword.trim()) {
      return;
    }

    switchSchoolMutation.mutate({
      nextTenantID: switchTargetTenantID,
      password: switchPassword,
    });
  }

  return (
    <Sidebar
      collapsible="offcanvas"
      className="border-r-0 p-0 [&>[data-slot=sidebar-inner]]:border-r [&>[data-slot=sidebar-inner]]:border-sidebar-border [&>[data-slot=sidebar-inner]]:bg-sidebar"
      >
      <Dialog open={Boolean(switchTargetTenantID)} onOpenChange={(open) => (!open ? closeSwitchDialog() : undefined)}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Confirm school switch</DialogTitle>
            <DialogDescription>
              {switchTargetSchool
                ? `Enter your password to switch this session to ${switchTargetSchool.tenant_name}.`
                : "Enter your password to switch schools."}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label htmlFor="school-switch-password">Password</Label>
            <Input
              id="school-switch-password"
              type="password"
              value={switchPassword}
              onChange={(event) => setSwitchPassword(event.target.value)}
              placeholder="Enter password"
              autoComplete="current-password"
              disabled={switchSchoolMutation.isPending}
            />
            {schoolSwitchError ? <p className="text-sm text-destructive">{schoolSwitchError}</p> : null}
          </div>
          <DialogFooter>
            <DialogClose render={<Button variant="outline" disabled={switchSchoolMutation.isPending} />}>Close</DialogClose>
            <Button onClick={confirmSchoolSwitch} disabled={switchSchoolMutation.isPending || !switchPassword.trim()}>
              {switchSchoolMutation.isPending ? "Switching..." : "Switch school"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AnimatePresence>
        {schoolSwitchOverlayLabel ? (
          <motion.div
            initial={{ opacity: 0, backdropFilter: "blur(0px)" }}
            animate={{ opacity: 1, backdropFilter: "blur(18px)" }}
            exit={{ opacity: 0, backdropFilter: "blur(0px)" }}
            transition={{ duration: 0.18, ease: shellEase }}
            className="pointer-events-none fixed inset-0 z-[120] flex items-center justify-center bg-background/28"
          >
            <motion.div
              initial={{ opacity: 0, y: 12, filter: "blur(10px)" }}
              animate={{ opacity: 1, y: 0, filter: "blur(0px)" }}
              exit={{ opacity: 0, y: 8, filter: "blur(8px)" }}
              transition={{ duration: 0.2, ease: shellEase }}
              className="border border-border/80 bg-background/90 px-5 py-3 text-center shadow-2xl"
            >
              <p className="text-[11px] font-medium uppercase tracking-[0.22em] text-muted-foreground">Switched to</p>
              <p className="mt-2 text-xl font-semibold tracking-tight text-foreground">{schoolSwitchOverlayLabel}</p>
            </motion.div>
          </motion.div>
        ) : null}
      </AnimatePresence>

      <SidebarHeader className="gap-3 px-4 pb-4 pt-4">
        {sessionReady && currentUser?.tenant_name ? (
          <div className="flex flex-col gap-2">
            {canSwitchSchools ? (
              <Select value={currentUser.tenant_id} onValueChange={(value) => handleSchoolSwitch(value ?? "")}>
                <SelectTrigger
                  aria-label="Switch school"
                  className="min-h-[68px] w-full rounded-xl border-sidebar-border bg-background/70 px-0 py-0 text-sidebar-foreground shadow-none hover:bg-sidebar-accent/50 [&>svg]:mr-3 [&>svg]:text-muted-foreground"
                  disabled={switchSchoolMutation.isPending || isSwitchingTenant}
                >
                  <div className="flex min-w-0 flex-1 items-center gap-3 px-3 py-3 text-left">
                    <div className="flex size-11 shrink-0 items-center justify-center rounded-xl bg-muted text-muted-foreground">
                      <Buildings className="size-5" weight="duotone" />
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-base font-semibold tracking-tight text-sidebar-foreground">
                        {selectedSchool?.tenant_name ?? currentUser.tenant_name}
                      </p>
                    </div>
                  </div>
                </SelectTrigger>
                <SelectContent
                  align="start"
                  sideOffset={6}
                  className="rounded-xl border-sidebar-border bg-popover/95 p-1 shadow-lg before:rounded-[inherit] before:backdrop-blur-xl"
                >
                  <SelectGroup>
                    {schoolChoices.map((tenant) => (
                      <SelectItem key={tenant.tenant_id} value={tenant.tenant_id} className="px-3 py-2.5 text-popover-foreground">
                        {tenant.tenant_name}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
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
                  <p className="truncate text-base font-semibold tracking-tight text-sidebar-foreground">{currentUser.tenant_name}</p>
                </div>
              </Link>
            )}
            {schoolSwitchError ? <p className="text-xs leading-5 text-sidebar-foreground/65">{schoolSwitchError}</p> : null}
          </div>
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
        {sessionReady && isLoggedIn ? <LogoutButton /> : <LoginButton />}
      </SidebarFooter>
    </Sidebar>
  );
}
