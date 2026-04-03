"use client";

import { IconAlertCircle, IconLock } from "@tabler/icons-react";
import { useState } from "react";
import { LoginGateGoogleButton } from "@/components/login-gate/login-gate-google-button";
import { useLoginGate } from "@/components/login-gate/use-login-gate";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { cn } from "@/lib/utils";

export function LoginGateForm() {
  const {
    email,
    password,
    tenantID,
    tenantChoices,
    error,
    isPending,
    isGooglePending,
    setEmail,
    setPassword,
    setTenantID,
    submit,
  } = useLoginGate();

  const errorMessage =
    error === "Failed to fetch"
      ? "We couldn't reach the sign-in service. Check your connection and try again."
      : error;
  const showTenantChooser = tenantChoices.length > 0;
  const selectedTenant = tenantChoices.find((tenant) => tenant.tenant_id === tenantID) ?? tenantChoices[0] ?? null;
  const [emailUnlocked, setEmailUnlocked] = useState(false);
  const [passwordUnlocked, setPasswordUnlocked] = useState(false);
  const emailLocked = showTenantChooser && !emailUnlocked && Boolean(password);
  const passwordLocked = showTenantChooser && !passwordUnlocked && Boolean(password);

  function unlockEmail() {
    if (showTenantChooser && emailLocked) {
      setEmailUnlocked(true);
    }
  }

  function unlockPassword() {
    if (showTenantChooser && passwordLocked) {
      setPasswordUnlocked(true);
    }
  }

  return (
    <form id="sign-in-form" className="space-y-5 transition-[opacity,transform] duration-200 ease-out" onSubmit={submit}>
      <div className="space-y-4">
        <LoginGateGoogleButton />
        {!showTenantChooser ? (
          <div className="flex items-center gap-3">
            <div className="h-px flex-1 bg-slate-200/80 dark:bg-white/10" />
            <span className="text-[11px] font-semibold uppercase tracking-[0.22em] text-slate-400 dark:text-slate-500">
              or use email
            </span>
            <div className="h-px flex-1 bg-slate-200/80 dark:bg-white/10" />
          </div>
        ) : null}
      </div>

      <div className="space-y-2">
        <Label htmlFor="email" className="text-slate-700 dark:text-slate-200">
          Email
        </Label>
        <div className="relative">
          <Input
            id="email"
            name="email"
            type="email"
            value={email}
            onChange={(event) => setEmail(event.target.value)}
            disabled={showTenantChooser && emailLocked}
            placeholder="teacher@school.edu"
            autoComplete="email"
            className={cn(
              "h-12 rounded-2xl border-slate-300/80 bg-white pr-11 text-slate-950 placeholder:text-slate-400 transition-all duration-150 ease-out dark:border-white/10 dark:bg-slate-900/80 dark:text-slate-100 dark:placeholder:text-slate-500",
              showTenantChooser && emailLocked
                ? "border-slate-200 bg-slate-100 text-slate-700 opacity-100 disabled:cursor-default disabled:bg-slate-100 disabled:text-slate-700 disabled:opacity-100 dark:border-white/10 dark:bg-slate-950/90 dark:text-slate-300 dark:disabled:bg-slate-950/90 dark:disabled:text-slate-300"
                : "",
            )}
            required
          />
          {showTenantChooser && emailLocked ? (
            <>
              <IconLock className="pointer-events-none absolute top-1/2 right-4 z-10 size-4 -translate-y-1/2 text-slate-400 dark:text-slate-500" />
              <div
                onClick={unlockEmail}
                className="absolute inset-0 z-20 rounded-2xl bg-slate-950/10 transition-colors duration-150 ease-out hover:bg-slate-950/4 active:bg-white/45 dark:bg-black/18 dark:hover:bg-black/10 dark:active:bg-white/12"
                style={{ cursor: "text" }}
              />
            </>
          ) : null}
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor="password" className="text-slate-700 dark:text-slate-200">
          Password
        </Label>
        <div className="relative">
          <Input
            id="password"
            name="password"
            type="password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            disabled={showTenantChooser && passwordLocked}
            placeholder="Enter your password"
            autoComplete="current-password"
            className={cn(
              "h-12 rounded-2xl border-slate-300/80 bg-white pr-11 text-slate-950 placeholder:text-slate-400 transition-all duration-150 ease-out dark:border-white/10 dark:bg-slate-900/80 dark:text-slate-100 dark:placeholder:text-slate-500",
              showTenantChooser && passwordLocked
                ? "border-slate-200 bg-slate-100 text-slate-700 opacity-100 disabled:cursor-default disabled:bg-slate-100 disabled:text-slate-700 disabled:opacity-100 dark:border-white/10 dark:bg-slate-950/90 dark:text-slate-300 dark:disabled:bg-slate-950/90 dark:disabled:text-slate-300"
                : "",
            )}
            required
          />
          {showTenantChooser && passwordLocked ? (
            <>
              <IconLock className="pointer-events-none absolute top-1/2 right-4 z-10 size-4 -translate-y-1/2 text-slate-400 dark:text-slate-500" />
              <div
                onClick={unlockPassword}
                className="absolute inset-0 z-20 rounded-2xl bg-slate-950/10 transition-colors duration-150 ease-out hover:bg-slate-950/4 active:bg-white/45 dark:bg-black/18 dark:hover:bg-black/10 dark:active:bg-white/12"
                style={{ cursor: "text" }}
              />
            </>
          ) : null}
        </div>
      </div>

      {showTenantChooser ? (
        <Alert className="animate-in fade-in-0 slide-in-from-top-1 gap-y-1 rounded-2xl border-sky-200/80 bg-sky-50/90 px-4 py-3 text-sky-950 shadow-none duration-200 dark:border-sky-400/20 dark:bg-sky-400/10 dark:text-sky-50">
          <IconAlertCircle className="text-sky-600 dark:text-sky-300" />
          <AlertTitle>Choose one school to continue.</AlertTitle>
          <AlertDescription className="leading-6 text-sky-800 dark:text-sky-100">
            We kept your email and password. Pick the school name below, then sign in.
          </AlertDescription>
        </Alert>
      ) : null}

      {showTenantChooser ? (
        <div className="animate-in fade-in-0 slide-in-from-top-1 space-y-2 duration-200">
          <div className="flex items-center justify-between gap-3">
            <Label htmlFor="tenant_id" className="text-slate-700 dark:text-slate-200">
              School
            </Label>
            <span className="text-xs font-medium text-slate-500 dark:text-slate-400">{tenantChoices.length} found</span>
          </div>
          <Select value={tenantID} onValueChange={(value) => setTenantID(value ?? "")}>
            <SelectTrigger
              id="tenant_id"
              aria-label="School"
              className="h-14 w-full rounded-2xl border-slate-300/80 bg-white px-4 text-sm text-slate-950 shadow-none transition-all duration-200 ease-out dark:border-white/10 dark:bg-slate-900/80 dark:text-slate-100"
            >
              {selectedTenant ? (
                <span className="flex min-w-0 flex-1 items-center justify-between gap-3 text-left">
                  <span className="truncate font-medium text-slate-950 dark:text-slate-100">{selectedTenant.tenant_name}</span>
                </span>
              ) : (
                <SelectValue placeholder="Choose school" />
              )}
            </SelectTrigger>
            <SelectContent
              align="start"
              className="rounded-2xl border border-slate-200/80 bg-white p-1 shadow-[0_20px_50px_rgba(15,23,42,0.12)] dark:border-white/10 dark:bg-slate-950"
            >
              {tenantChoices.map((tenant) => (
                <SelectItem
                  key={tenant.tenant_id}
                  value={tenant.tenant_id}
                  className="rounded-xl px-3 py-2 text-slate-950 focus:bg-slate-100 dark:text-slate-100 dark:focus:bg-slate-900"
                >
                  {tenant.tenant_name}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          {selectedTenant ? (
            <p className="text-xs leading-5 text-slate-500 dark:text-slate-400">Selected workspace: {selectedTenant.tenant_name}</p>
          ) : null}
        </div>
      ) : null}

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
