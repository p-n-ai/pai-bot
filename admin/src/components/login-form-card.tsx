"use client";

import Link from "next/link";
import type { FormEvent } from "react";
import { ArrowRight, KeyRound, ShieldCheck } from "lucide-react";
import { FormField } from "@/components/form-field";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { TenantChoice } from "@/lib/api";

export function LoginFormCard({
  email,
  password,
  tenantID,
  tenantChoices,
  error,
  isPending,
  onEmailChange,
  onPasswordChange,
  onTenantChange,
  onSubmit,
}: {
  email: string;
  password: string;
  tenantID: string;
  tenantChoices: TenantChoice[];
  error: string;
  isPending: boolean;
  onEmailChange: (value: string) => void;
  onPasswordChange: (value: string) => void;
  onTenantChange: (value: string) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}) {
  const tenantRequired = tenantChoices.length > 0;

  return (
    <div className="mx-auto grid min-h-[calc(100vh-8rem)] max-w-6xl items-center gap-8 px-4 py-10 lg:grid-cols-[1.1fr_0.9fr] lg:px-8">
      <section className="space-y-6">
        <div className="inline-flex items-center gap-2 rounded-full border border-sky-200/70 bg-white/70 px-4 py-2 text-xs font-semibold uppercase tracking-[0.24em] text-sky-700 shadow-sm dark:border-sky-400/20 dark:bg-slate-950/40 dark:text-sky-200">
          <ShieldCheck className="size-4" />
          Admin Login
        </div>
        <div className="space-y-4">
          <h1 className="max-w-2xl text-4xl font-semibold tracking-tight text-slate-950 dark:text-white md:text-5xl">
            Sign in with the account created from your invite.
          </h1>
          <p className="max-w-2xl text-base leading-7 text-slate-600 dark:text-slate-300">
            Teachers, parents, and admins activate once by invite, then return with email and password.
            Students continue through Telegram.
          </p>
        </div>
        <div className="grid gap-3 sm:grid-cols-2">
          <div className="rounded-[24px] border border-white/70 bg-white/70 p-5 shadow-[0_18px_50px_rgba(15,23,42,0.08)] backdrop-blur dark:border-white/10 dark:bg-slate-950/45">
            <KeyRound className="size-5 text-sky-600 dark:text-sky-300" />
            <p className="mt-4 text-sm font-semibold text-slate-900 dark:text-slate-100">Invite-first onboarding</p>
            <p className="mt-2 text-sm leading-6 text-slate-600 dark:text-slate-300">
              New teacher, parent, and admin accounts are created through an admin-issued invite.
            </p>
          </div>
          <div className="rounded-[24px] border border-white/70 bg-white/70 p-5 shadow-[0_18px_50px_rgba(15,23,42,0.08)] backdrop-blur dark:border-white/10 dark:bg-slate-950/45">
            <ArrowRight className="size-5 text-amber-600 dark:text-amber-300" />
            <p className="mt-4 text-sm font-semibold text-slate-900 dark:text-slate-100">Direct dashboard access</p>
            <p className="mt-2 text-sm leading-6 text-slate-600 dark:text-slate-300">
              After login, the session is stored locally so the admin API can authorize dashboard requests.
            </p>
          </div>
        </div>
      </section>

      <Card className="rounded-[32px] border border-white/75 bg-white/78 py-0 shadow-[0_28px_90px_rgba(15,23,42,0.12)] backdrop-blur dark:border-white/10 dark:bg-slate-950/58 dark:shadow-[0_28px_90px_rgba(2,8,23,0.45)]">
        <CardHeader className="px-8 pt-8">
          <h2 className="text-2xl font-semibold text-slate-950 dark:text-white">Sign in</h2>
          <p className="text-sm leading-6 text-slate-600 dark:text-slate-300">Use the email and password set during invite activation.</p>
        </CardHeader>
        <CardContent className="px-8 pb-8">
          <form className="space-y-5" onSubmit={onSubmit}>
            <FormField label="Email" htmlFor="email">
              <Input
                id="email"
                name="email"
                type="email"
                value={email}
                onChange={(event) => onEmailChange(event.target.value)}
                placeholder="teacher@example.com"
                autoComplete="email"
                className="text-slate-950 placeholder:text-slate-400 dark:text-slate-100 dark:placeholder:text-slate-500"
                required
              />
            </FormField>

            {tenantRequired ? (
              <FormField
                label="School"
                htmlFor="tenant_id"
                description="The same email is linked to multiple tenant-scoped accounts, so the backend needs the school context."
              >
                <Select value={tenantID} onValueChange={onTenantChange}>
                  <SelectTrigger id="tenant_id" className="h-10 w-full bg-white text-slate-950 dark:bg-slate-900 dark:text-slate-100">
                    <SelectValue placeholder="Choose a school" />
                  </SelectTrigger>
                  <SelectContent align="start">
                    {tenantChoices.map((tenant) => (
                      <SelectItem key={tenant.tenant_id} value={tenant.tenant_id}>
                        {tenant.tenant_name} ({tenant.tenant_slug})
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FormField>
            ) : null}

            <FormField label="Password" htmlFor="password">
              <Input
                id="password"
                name="password"
                type="password"
                value={password}
                onChange={(event) => onPasswordChange(event.target.value)}
                placeholder="Enter your password"
                autoComplete="current-password"
                className="text-slate-950 placeholder:text-slate-400 dark:text-slate-100 dark:placeholder:text-slate-500"
                required
              />
            </FormField>

            {error ? (
              <div className="rounded-2xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700 dark:border-rose-400/30 dark:bg-rose-500/10 dark:text-rose-200">
                {error}
              </div>
            ) : null}

            <Button type="submit" size="lg" className="w-full" disabled={isPending}>
              {isPending ? "Signing in..." : "Sign in"}
            </Button>
          </form>

          <div className="mt-4 text-sm text-slate-500 dark:text-slate-400">
            <Link href="/" className="underline decoration-slate-300 underline-offset-4 hover:text-slate-900 dark:hover:text-white">
              Back to admin overview
            </Link>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
