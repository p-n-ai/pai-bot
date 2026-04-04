import type { Metadata } from "next";
import { cookies } from "next/headers";
import { Suspense } from "react";
import { Agentation } from "agentation";
import { APP_SIDEBAR_COOKIE_NAME } from "@/components/app-sidebar";
import { AppShell } from "@/components/app-shell";
import { QueryProvider } from "@/components/query-provider";
import { ThemeProvider } from "@/components/theme-provider";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import "./globals.css";
import { Geist } from "next/font/google";
import { getServerAuthSession } from "@/lib/server-api";
import { buildSchoolSwitchState } from "@/lib/school-switch-state";
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
  const session = await getServerAuthSession();
  const defaultSidebarOpen = cookieStore.get(APP_SIDEBAR_COOKIE_NAME)?.value !== "false";
  const initialCurrentUser = session?.user ?? null;
  const initialSchoolSwitchState =
    session ? buildSchoolSwitchState(session.user.email, session.user.tenant_id, session.tenant_choices ?? []) : null;

  return (
    <html lang="en" suppressHydrationWarning className={cn("font-sans", geist.variable)}>
      <body className="antialiased">
        <ThemeProvider>
          <TooltipProvider>
            <Suspense fallback={null}>
              <QueryProvider>
                <AppShell
                  defaultSidebarOpen={defaultSidebarOpen}
                  initialCurrentUser={initialCurrentUser}
                  initialSchoolSwitchState={initialSchoolSwitchState}
                >
                  {children}
                </AppShell>
              </QueryProvider>
            </Suspense>
            <Toaster richColors position="top-right" />
          </TooltipProvider>
        </ThemeProvider>
        {showAgentation ? <Agentation endpoint={agentationEndpoint} /> : null}
      </body>
    </html>
  );
}
