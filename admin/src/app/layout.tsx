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

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" suppressHydrationWarning>
      <head>
        <script
          dangerouslySetInnerHTML={{
            __html: `
              (() => {
                const key = "pai-admin-theme";
                const saved = window.localStorage.getItem(key);
                const system = window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
                const theme = saved === "light" || saved === "dark" ? saved : system;
                document.documentElement.classList.toggle("dark", theme === "dark");
                document.documentElement.style.colorScheme = theme;
              })();
            `,
          }}
        />
      </head>
      <body className="antialiased">
        <ThemeProvider>
          <Suspense fallback={null}>
            <RefineProvider>
              <AdminShell>{children}</AdminShell>
            </RefineProvider>
          </Suspense>
        </ThemeProvider>
        <Agentation />
      </body>
    </html>
  );
}
