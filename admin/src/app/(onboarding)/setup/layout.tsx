import { ThemeToggle } from "@/components/theme-toggle";

export default function SetupLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <div className="theme-transition relative min-h-screen overflow-x-hidden bg-[radial-gradient(circle_at_top_left,_rgba(15,23,42,0.05),_transparent_35%),linear-gradient(180deg,_rgba(248,250,252,1)_0%,_rgba(241,245,249,0.9)_100%)] text-foreground dark:bg-[radial-gradient(circle_at_top_left,_rgba(148,163,184,0.12),_transparent_35%),linear-gradient(180deg,_rgba(2,6,23,1)_0%,_rgba(15,23,42,0.95)_100%)]">
      <div className="pointer-events-none fixed right-3 top-3 z-20 lg:right-6 lg:top-5">
        <div className="pointer-events-auto">
          <ThemeToggle />
        </div>
      </div>
      <main className="mx-auto flex min-h-screen w-full max-w-6xl flex-col px-4 py-8 sm:px-6 lg:px-8 lg:py-12">
        {children}
      </main>
    </div>
  );
}
