"use client";

import Link from "next/link";
import type { FormEvent } from "react";
import { IconCircleCheck, IconMailCheck, IconShieldCheck } from "@tabler/icons-react";
import { FormField } from "@/components/form-field";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";

export function InviteAcceptanceCard({
  token,
  name,
  password,
  error,
  isPending,
  onNameChange,
  onPasswordChange,
  onSubmit,
}: {
  token: string;
  name: string;
  password: string;
  error: string;
  isPending: boolean;
  onNameChange: (value: string) => void;
  onPasswordChange: (value: string) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}) {
  const tokenMissing = token.trim() === "";

  return (
    <div className="mx-auto grid min-h-[calc(100vh-8rem)] max-w-6xl items-center gap-8 px-4 py-10 lg:grid-cols-[1.1fr_0.9fr] lg:px-8">
      <section className="space-y-6">
        <div className="inline-flex items-center gap-2 rounded-full border border-emerald-200/70 bg-white/70 px-4 py-2 text-xs font-semibold uppercase tracking-[0.24em] text-emerald-700 shadow-sm dark:border-emerald-400/20 dark:bg-slate-950/40 dark:text-emerald-200">
          <IconShieldCheck className="size-4" />
          Invite Activation
        </div>
        <div className="space-y-4">
          <h1 className="max-w-2xl text-4xl font-semibold tracking-tight text-slate-950 dark:text-white md:text-5xl">
            Accept your invite and set the password for this workspace.
          </h1>
          <p className="max-w-2xl text-base leading-7 text-slate-600 dark:text-slate-300">
            Teachers, parents, and admins activate once from an emailed invite, then return through the normal sign-in page.
          </p>
        </div>
        <div className="grid gap-3 sm:grid-cols-2">
          <div className="rounded-[24px] border border-white/70 bg-white/70 p-5 shadow-[0_18px_50px_rgba(15,23,42,0.08)] backdrop-blur dark:border-white/10 dark:bg-slate-950/45">
            <IconMailCheck className="size-5 text-emerald-600 dark:text-emerald-300" />
            <p className="mt-4 text-sm font-semibold text-slate-900 dark:text-slate-100">One-time activation</p>
            <p className="mt-2 text-sm leading-6 text-slate-600 dark:text-slate-300">
              The invite link identifies the tenant-scoped account that is being activated.
            </p>
          </div>
          <div className="rounded-[24px] border border-white/70 bg-white/70 p-5 shadow-[0_18px_50px_rgba(15,23,42,0.08)] backdrop-blur dark:border-white/10 dark:bg-slate-950/45">
            <IconCircleCheck className="size-5 text-sky-600 dark:text-sky-300" />
            <p className="mt-4 text-sm font-semibold text-slate-900 dark:text-slate-100">Immediate sign-in</p>
            <p className="mt-2 text-sm leading-6 text-slate-600 dark:text-slate-300">
              Once accepted, the session is stored and the user is redirected into the correct admin view automatically.
            </p>
          </div>
        </div>
      </section>

      <Card className="rounded-[32px] border border-white/75 bg-white/78 py-0 shadow-[0_28px_90px_rgba(15,23,42,0.12)] backdrop-blur dark:border-white/10 dark:bg-slate-950/58 dark:shadow-[0_28px_90px_rgba(2,8,23,0.45)]">
        <CardHeader className="px-8 pt-8">
          <h2 className="text-2xl font-semibold text-slate-950 dark:text-white">Accept your invite</h2>
          <p className="text-sm leading-6 text-slate-600 dark:text-slate-300">
            Set the display name and password that will be used for future sign-ins.
          </p>
        </CardHeader>
        <CardContent className="px-8 pb-8">
          <form className="space-y-5" onSubmit={onSubmit}>
            <FormField label="Full name" htmlFor="name">
              <Input
                id="name"
                name="name"
                type="text"
                value={name}
                onChange={(event) => onNameChange(event.target.value)}
                placeholder="Parent One"
                autoComplete="name"
                className="text-slate-950 placeholder:text-slate-400 dark:text-slate-100 dark:placeholder:text-slate-500"
                required
              />
            </FormField>

            <FormField
              label="Password"
              htmlFor="password"
              description="Use a strong password. This becomes the password for your normal email sign-in."
            >
              <Input
                id="password"
                name="password"
                type="password"
                value={password}
                onChange={(event) => onPasswordChange(event.target.value)}
                placeholder="Create a strong password"
                autoComplete="new-password"
                className="text-slate-950 placeholder:text-slate-400 dark:text-slate-100 dark:placeholder:text-slate-500"
                required
              />
            </FormField>

            {error ? (
              <div className="rounded-2xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700 dark:border-rose-400/30 dark:bg-rose-500/10 dark:text-rose-200">
                {error}
              </div>
            ) : null}

            <Button type="submit" size="lg" className="w-full" disabled={isPending || tokenMissing}>
              {isPending ? "Activating..." : "Accept invite"}
            </Button>
          </form>

          <div className="mt-4 text-sm text-slate-500 dark:text-slate-400">
            <Link href="/login" className="underline decoration-slate-300 underline-offset-4 hover:text-slate-900 dark:hover:text-white">
              Back to sign in
            </Link>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
