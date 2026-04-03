"use client";

import { IconAlertCircle } from "@tabler/icons-react";
import { LoginGateGoogleButton } from "@/components/login-gate/login-gate-google-button";
import { useLoginGate } from "@/components/login-gate/use-login-gate";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export function LoginGateForm() {
  const { email, password, error, isPending, isGooglePending, setEmail, setPassword, submit } = useLoginGate();

  const errorMessage =
    error === "Failed to fetch"
      ? "We couldn't reach the sign-in service. Check your connection and try again."
      : error;

  return (
    <form id="sign-in-form" className="space-y-5 transition-[opacity,transform] duration-200 ease-out" onSubmit={submit}>
      <div className="space-y-4">
        <LoginGateGoogleButton />
        <div className="flex items-center gap-3">
          <div className="h-px flex-1 bg-slate-200/80 dark:bg-white/10" />
          <span className="text-[11px] font-semibold uppercase tracking-[0.22em] text-slate-400 dark:text-slate-500">
            or use email
          </span>
          <div className="h-px flex-1 bg-slate-200/80 dark:bg-white/10" />
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor="email" className="text-slate-700 dark:text-slate-200">
          Email
        </Label>
        <Input
          id="email"
          name="email"
          type="email"
          value={email}
          onChange={(event) => setEmail(event.target.value)}
          placeholder="teacher@school.edu"
          autoComplete="email"
          className="h-12 rounded-2xl border-slate-300/80 bg-white text-slate-950 placeholder:text-slate-400 transition-all duration-150 ease-out dark:border-white/10 dark:bg-slate-900/80 dark:text-slate-100 dark:placeholder:text-slate-500"
          required
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="password" className="text-slate-700 dark:text-slate-200">
          Password
        </Label>
        <Input
          id="password"
          name="password"
          type="password"
          value={password}
          onChange={(event) => setPassword(event.target.value)}
          placeholder="Enter your password"
          autoComplete="current-password"
          className="h-12 rounded-2xl border-slate-300/80 bg-white text-slate-950 placeholder:text-slate-400 transition-all duration-150 ease-out dark:border-white/10 dark:bg-slate-900/80 dark:text-slate-100 dark:placeholder:text-slate-500"
          required
        />
      </div>

      {error ? (
        <Alert
          variant="destructive"
          className="animate-in fade-in-0 slide-in-from-top-1 gap-y-1 rounded-2xl border-rose-200/80 bg-rose-50/90 px-4 py-3 shadow-none duration-200 dark:border-rose-400/25 dark:bg-rose-500/10"
        >
          <IconAlertCircle />
          <AlertTitle>{"We couldn't sign you in yet."}</AlertTitle>
          <AlertDescription className="leading-6 text-rose-700 dark:text-rose-200">
            {errorMessage}
          </AlertDescription>
        </Alert>
      ) : null}

      <Button
        type="submit"
        size="lg"
        className="h-12 w-full rounded-full bg-slate-950 text-white transition-all duration-150 ease-out hover:bg-slate-800 active:translate-y-px dark:bg-slate-100 dark:text-slate-950 dark:hover:bg-white"
        disabled={isPending || isGooglePending}
      >
        {isPending ? "Signing in..." : "Sign in"}
      </Button>
    </form>
  );
}
