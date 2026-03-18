"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, useState, useTransition } from "react";
import { ArrowRight, KeyRound, ShieldCheck } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { getStoredUser, hasStoredSession, LoginError, login, persistSession, type TenantChoice } from "@/lib/api";
import { getSafeNextPath, hasAdminUIAccess } from "@/lib/rbac.mjs";

export default function LoginPage() {
  return (
    <Suspense fallback={<LoginPageFallback />}>
      <LoginPageContent />
    </Suspense>
  );
}

function LoginPageContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [isPending, startTransition] = useTransition();
  const [checkedSession, setCheckedSession] = useState(false);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [tenantID, setTenantID] = useState("");
  const [tenantChoices, setTenantChoices] = useState<TenantChoice[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    const user = getStoredUser();
    if (user && hasStoredSession() && hasAdminUIAccess(user)) {
      router.replace(getSafeNextPath(user, searchParams.get("next")));
      return;
    }

    setCheckedSession(true);
  }, [router, searchParams]);

  if (!checkedSession) {
    return null;
  }

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");

    startTransition(async () => {
      try {
        const session = await login({
          tenant_id: tenantID || undefined,
          email: email.trim(),
          password,
        });
        if (!hasAdminUIAccess(session.user)) {
          setError("This account does not have access to the admin UI.");
          return;
        }
        persistSession(session);
        router.push(getSafeNextPath(session.user, searchParams.get("next")));
      } catch (err) {
        if (err instanceof LoginError && err.code === "tenant_required") {
          setTenantChoices(err.tenants);
          setTenantID((current) => current || err.tenants[0]?.tenant_id || "");
          setError("This email exists in more than one school. Choose the correct tenant to continue.");
          return;
        }
        setTenantChoices([]);
        setTenantID("");
        setError(err instanceof Error ? err.message : "Login failed");
      }
    });
  }

  return (
    <LoginPageLayout
      email={email}
      password={password}
      tenantID={tenantID}
      tenantChoices={tenantChoices}
      error={error}
      isPending={isPending}
      onEmailChange={(value) => {
        setEmail(value);
        if (tenantChoices.length > 0) {
          setTenantChoices([]);
          setTenantID("");
          setError("");
        }
      }}
      onPasswordChange={setPassword}
      onTenantChange={setTenantID}
      onSubmit={handleSubmit}
    />
  );
}

function LoginPageFallback() {
  return (
    <LoginPageLayout
      email=""
      password=""
      tenantID=""
      tenantChoices={[]}
      error=""
      isPending={false}
      onEmailChange={() => {}}
      onPasswordChange={() => {}}
      onTenantChange={() => {}}
      onSubmit={(event) => event.preventDefault()}
    />
  );
}

function LoginPageLayout({
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
  onSubmit: (event: React.FormEvent<HTMLFormElement>) => void;
}) {
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
          <CardTitle className="text-2xl font-semibold text-slate-950 dark:text-white">Sign in</CardTitle>
          <CardDescription className="text-sm leading-6 text-slate-600 dark:text-slate-300">
            Use the email and password set during invite activation.
          </CardDescription>
        </CardHeader>
        <CardContent className="px-8 pb-8">
          <form className="space-y-5" onSubmit={onSubmit}>
            <div className="space-y-2">
              <Label htmlFor="email">Email</Label>
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
            </div>
            {tenantChoices.length > 0 ? (
              <div className="space-y-2">
                <Label htmlFor="tenant_id">School</Label>
                <select
                  id="tenant_id"
                  name="tenant_id"
                  value={tenantID}
                  onChange={(event) => onTenantChange(event.target.value)}
                  className="h-10 w-full rounded-lg border border-input bg-white px-3 text-sm text-slate-950 outline-none transition-colors focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 dark:border-white/10 dark:bg-slate-900 dark:text-slate-100"
                  required
                >
                  {tenantChoices.map((tenant) => (
                    <option key={tenant.tenant_id} value={tenant.tenant_id}>
                      {tenant.tenant_name} ({tenant.tenant_slug})
                    </option>
                  ))}
                </select>
                <p className="text-xs leading-5 text-slate-500 dark:text-slate-400">
                  The same email is linked to multiple tenant-scoped accounts, so the backend needs the school context.
                </p>
              </div>
            ) : null}
            <div className="space-y-2">
              <Label htmlFor="password">Password</Label>
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
            </div>
            {error ? (
              <div className="rounded-2xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700 dark:border-rose-400/30 dark:bg-rose-500/10 dark:text-rose-200">
                {error}
              </div>
            ) : null}
            <Button type="submit" size="lg" className="w-full" disabled={isPending}>
              {isPending ? "Signing in..." : "Sign in"}
            </Button>
          </form>

          <div className="mt-6 rounded-[24px] border border-dashed border-slate-200 bg-slate-50/80 p-4 text-sm leading-6 text-slate-600 dark:border-white/10 dark:bg-slate-900/50 dark:text-slate-300">
            <p className="font-medium text-slate-900 dark:text-slate-100">Demo credentials</p>
            <p className="mt-2">
              Student: <span className="font-mono">student@example.com</span> / <span className="font-mono">demo-password</span>
            </p>
            <p>
              Teacher: <span className="font-mono">teacher@example.com</span> / <span className="font-mono">demo-password</span>
            </p>
            <p>
              Parent: <span className="font-mono">parent@example.com</span> / <span className="font-mono">demo-password</span>
            </p>
            <p>
              Admin: <span className="font-mono">admin@example.com</span> / <span className="font-mono">demo-password</span>
            </p>
            <p>
              Second school admin: <span className="font-mono">second-admin@example.com</span> / <span className="font-mono">demo-password</span>
            </p>
            <p>
              Platform admin: <span className="font-mono">platform-admin@example.com</span> / <span className="font-mono">demo-password</span>
            </p>
            <p className="mt-3">
              teacher@example.com exists in both demo schools, so that login will prompt for a tenant choice. Student accounts are still not allowed into the admin UI.
            </p>
          </div>

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
