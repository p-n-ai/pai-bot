import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import { ThemeProvider } from "@/components/theme-provider";
import { ThemeToggle } from "@/components/theme-toggle";
import "./globals.css";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

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
      <body className={`${geistSans.variable} ${geistMono.variable} antialiased`}>
        <ThemeProvider>
          <div className="pointer-events-none fixed top-5 right-5 z-50">
            <div className="pointer-events-auto">
              <ThemeToggle />
            </div>
          </div>
          {children}
        </ThemeProvider>
      </body>
    </html>
  );
}
