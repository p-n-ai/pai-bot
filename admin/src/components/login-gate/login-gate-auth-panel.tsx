"use client";

import {
  Card,
  CardContent,
  CardDescription,
} from "@/components/ui/card";

export function LoginGateAuthPanel({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <section
      aria-labelledby="login-gate-title"
      className="order-1 flex min-h-svh border-b border-border bg-card/80 lg:order-2 lg:min-h-0 lg:border-b-0 lg:border-l"
    >
      <Card className="flex h-full w-full rounded-none border-0 bg-transparent py-0 shadow-none">
        <CardContent className="flex flex-1 flex-col justify-center px-6 py-10 sm:px-8 lg:px-10 lg:py-10">
          <div className="mb-6 space-y-2 transition-all duration-200 ease-out">
            <p className="text-[11px] font-semibold uppercase tracking-[0.26em] text-muted-foreground">
              P&amp;AI Admin
            </p>
            <h1
              id="login-gate-title"
              className="text-4xl font-semibold tracking-[-0.04em] text-foreground"
            >
              Sign in to your workspace
            </h1>
            <CardDescription className="text-sm leading-7 text-muted-foreground">
              Continue with your invited email or your linked Google account.
            </CardDescription>
          </div>
          {children}
        </CardContent>
      </Card>
    </section>
  );
}
