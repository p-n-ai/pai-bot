import type { Metadata } from "next";
import { Suspense } from "react";
import { DevelopmentAgentation } from "@/components/development-agentation";
import { QueryProvider } from "@/components/query-provider";
import { ThemeProvider } from "@/components/theme-provider";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import "./globals.css";
import { Geist } from "next/font/google";
import { cn } from "@/lib/utils";

const geist = Geist({subsets:['latin'],variable:'--font-sans'});

export const metadata: Metadata = {
  title: "P&AI Admin",
  description: "Teacher and parent dashboard for P&AI Bot",
};

export default async function RootLayout({
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
              <QueryProvider>{children}</QueryProvider>
            </Suspense>
            <Toaster richColors position="top-right" />
          </TooltipProvider>
        </ThemeProvider>
        <DevelopmentAgentation />
      </body>
    </html>
  );
}
