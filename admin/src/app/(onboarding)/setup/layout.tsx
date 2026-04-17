import { ThemeProvider } from "@/components/theme-provider";

export default function SetupLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <ThemeProvider forcedTheme="light">
      <div className="theme-transition relative min-h-screen overflow-x-hidden bg-[radial-gradient(circle_at_top_left,_rgba(15,23,42,0.05),_transparent_35%),linear-gradient(180deg,_rgba(248,250,252,1)_0%,_rgba(241,245,249,0.9)_100%)] text-foreground">
        <main className="mx-auto flex min-h-screen w-full max-w-6xl flex-col px-4 py-8 sm:px-6 lg:px-8 lg:py-12">
          {children}
        </main>
      </div>
    </ThemeProvider>
  );
}
