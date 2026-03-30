import type { Metadata } from "next";
import { Suspense } from "react";
import { AdminShell } from "@/components/admin-shell";
import { RefineProvider } from "@/components/refine-provider";
import { ThemeProvider } from "@/components/theme-provider";
import "./globals.css";

export const metadata: Metadata = {
  title: "P&AI Admin",
  description: "Teacher and parent dashboard for P&AI Bot",
};

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
      </body>
    </html>
  );
}
