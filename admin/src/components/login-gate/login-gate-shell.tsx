"use client";

export function LoginGateShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="relative flex min-h-screen w-full items-stretch">
      <div className="relative grid min-h-screen w-full gap-0 overflow-hidden bg-white/76 backdrop-blur dark:bg-slate-950/62 lg:grid-cols-[minmax(0,2.15fr)_minmax(24rem,1fr)]">
        {children}
      </div>
    </div>
  );
}
