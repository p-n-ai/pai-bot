import type { Metadata } from "next";
import { Suspense } from "react";
import { Agentation } from "agentation";
import { AdminShell } from "@/components/admin-shell";
import { RefineProvider } from "@/components/refine-provider";
import { ThemeProvider } from "@/components/theme-provider";
import "./globals.css";

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
    <html lang="en" suppressHydrationWarning>
      <body className="antialiased">
        <ThemeProvider>
          <Suspense fallback={null}>
            <RefineProvider>
              <AdminShell>{children}</AdminShell>
            </RefineProvider>
          </Suspense>
        </ThemeProvider>
        {showAgentation ? <Agentation endpoint={agentationEndpoint} /> : null}
      </body>
    </html>
  );
}
