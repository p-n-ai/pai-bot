"use client";

export function LoginGateShell({ children }: { children: React.ReactNode }) {
  return (
    <main className="relative flex min-h-svh w-full items-stretch bg-background">
      <div className="relative grid min-h-svh w-full gap-0 overflow-hidden bg-background/80 backdrop-blur lg:grid-cols-[minmax(0,2.15fr)_minmax(24rem,1fr)]">
        {children}
      </div>
    </main>
  );
}
