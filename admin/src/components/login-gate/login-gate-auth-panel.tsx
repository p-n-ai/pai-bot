"use client";

import {
  Card,
  CardContent,
  CardDescription,
  CardTitle,
} from "@/components/ui/card";

export function LoginGateAuthPanel({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex h-full border-t border-black/8 bg-black/[0.02] dark:border-white/8 dark:bg-white/[0.02] lg:border-l lg:border-t-0">
      <Card className="flex h-full w-full rounded-none border-0 bg-transparent py-0 shadow-none">
        <CardContent className="flex flex-1 flex-col justify-center px-8 py-8 lg:px-10 lg:py-10">
          <div className="mb-6 space-y-2 transition-all duration-200 ease-out">
            <p className="text-[11px] font-semibold uppercase tracking-[0.26em] text-slate-500 dark:text-slate-400">
              Sign in
            </p>
            <CardTitle className="text-4xl font-semibold tracking-[-0.04em] text-slate-950 dark:text-white">
              Admin access
            </CardTitle>
            <CardDescription className="text-sm leading-7 text-slate-600 dark:text-slate-300">
              Use Google for same-email sign-in, or continue with the invite-based email flow.
            </CardDescription>
          </div>
          {children}
        </CardContent>
      </Card>
    </div>
  );
}
