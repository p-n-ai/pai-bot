import type { Metadata } from "next";
import { cookies } from "next/headers";
import { Suspense } from "react";
import { Agentation } from "agentation";
import { APP_SIDEBAR_COOKIE_NAME } from "@/components/app-sidebar";
import { AppShell } from "@/components/app-shell";
import { RefineProvider } from "@/components/refine-provider";
import { ThemeProvider } from "@/components/theme-provider";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import "./globals.css";
import { Geist } from "next/font/google";
import type { AuthUser } from "@/lib/api";
import { USER_COOKIE, parseCookieJSON } from "@/lib/auth-session";
import { SCHOOL_SWITCH_STATE_COOKIE, type SchoolSwitchState } from "@/lib/school-switch-state";
import { cn } from "@/lib/utils";

const geist = Geist({subsets:['latin'],variable:'--font-sans'});

export const metadata: Metadata = {
  title: "P&AI Admin",
  description: "Teacher and parent dashboard for P&AI Bot",
};

const agentationEndpoint = process.env.NEXT_PUBLIC_AGENTATION_ENDPOINT;
const showAgentation = process.env.NODE_ENV === "development" && Boolean(agentationEndpoint);

export default async function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  const cookieStore = await cookies();
  const defaultSidebarOpen = cookieStore.get(APP_SIDEBAR_COOKIE_NAME)?.value !== "false";
  const initialCurrentUser = parseCookieJSON<AuthUser>(cookieStore.get(USER_COOKIE)?.value);
  const initialSchoolSwitchState = parseCookieJSON<SchoolSwitchState>(
    cookieStore.get(SCHOOL_SWITCH_STATE_COOKIE)?.value,
  );

  return (
    <html lang="en" suppressHydrationWarning className={cn("font-sans", geist.variable)}>
      <body className="antialiased">
        <ThemeProvider>
          <TooltipProvider>
            <Suspense fallback={null}>
              <RefineProvider>
                <AppShell
                  defaultSidebarOpen={defaultSidebarOpen}
                  initialCurrentUser={initialCurrentUser}
                  initialSchoolSwitchState={initialSchoolSwitchState}
                >
                  {children}
                </AppShell>
              </RefineProvider>
            </Suspense>
            <Toaster richColors position="top-right" />
          </TooltipProvider>
        </ThemeProvider>
        {showAgentation ? <Agentation endpoint={agentationEndpoint} /> : null}
      </body>
    </html>
  );
}
