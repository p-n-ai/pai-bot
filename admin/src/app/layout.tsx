import type { Metadata } from "next";
import { Suspense } from "react";
import { Agentation } from "agentation";
import { AdminShell } from "@/components/admin-shell";
import { RefineProvider } from "@/components/refine-provider";
import { ThemeProvider } from "@/components/theme-provider";
import { TooltipProvider } from "@/components/ui/tooltip";
import "./globals.css";
import { Geist } from "next/font/google";
import { cn } from "@/lib/utils";

const geist = Geist({subsets:['latin'],variable:'--font-sans'});

export const metadata: Metadata = {
  title: "P&AI Admin",
  description: "Teacher and parent dashboard for P&AI Bot",
};

const agentationEndpoint = process.env.NEXT_PUBLIC_AGENTATION_ENDPOINT;
const showAgentation = process.env.NODE_ENV === "development" && Boolean(agentationEndpoint);

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" suppressHydrationWarning className={cn("font-sans", geist.variable)}>
      <body className="antialiased">
        <ThemeProvider>
          <TooltipProvider>
            <Suspense fallback={null}>
              <RefineProvider>
                <AdminShell>{children}</AdminShell>
              </RefineProvider>
            </Suspense>
          </TooltipProvider>
        </ThemeProvider>
        {showAgentation ? <Agentation endpoint={agentationEndpoint} /> : null}
      </body>
    </html>
  );
}
