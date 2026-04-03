"use client";

import { AnimatePresence, motion, useReducedMotion } from "framer-motion";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { Fragment, type CSSProperties } from "react";
import { AuthRedirectNotice } from "@/components/auth-redirect-notice";
import { AppSidebar } from "@/components/app-sidebar";
import { useAdminSessionBootstrap } from "@/hooks/use-admin-session-bootstrap";
import { useHydrated } from "@/hooks/use-hydrated";
import {
  Breadcrumb,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbList,
  BreadcrumbPage,
  BreadcrumbSeparator,
} from "@/components/ui/breadcrumb";
import {
  SidebarInset,
  SidebarProvider,
  SidebarTrigger,
} from "@/components/ui/sidebar";
import { ThemeToggle } from "@/components/theme-toggle";
import type { AuthUser } from "@/lib/api";
import { getBreadcrumbs } from "@/lib/navigation.mjs";
import { isPublicEntryRoute } from "@/lib/rbac.mjs";
import { type SchoolSwitchState } from "@/lib/school-switch-state";
import { useAppStore } from "@/stores/app-store";

const shellEase = [0.22, 1, 0.36, 1] as const;

export function AppShell({
  children,
  defaultSidebarOpen = true,
  initialCurrentUser = null,
  initialSchoolSwitchState = null,
}: {
  children: React.ReactNode;
  defaultSidebarOpen?: boolean;
  initialCurrentUser?: AuthUser | null;
  initialSchoolSwitchState?: SchoolSwitchState | null;
}) {
  const pathname = usePathname();
  const hydrated = useAppStore((state) => state.hydrated);
  const storeCurrentUser = useAppStore((state) => state.currentUser);
  const hasMounted = useHydrated();
  const currentUser = hasMounted ? storeCurrentUser : initialCurrentUser;
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

  useAdminSessionBootstrap(initialCurrentUser, initialSchoolSwitchState);

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
      defaultOpen={defaultSidebarOpen}
      style={sidebarLayoutStyle}
      className="theme-transition isolate min-h-screen bg-background text-foreground"
    >
      <AuthRedirectNotice />
      <AppSidebar
        pathname={pathname}
        hydrated={hydrated}
        hasMounted={hasMounted}
        initialCurrentUser={initialCurrentUser}
        initialSchoolSwitchState={initialSchoolSwitchState}
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
