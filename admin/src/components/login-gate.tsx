"use client";

import dynamic from "next/dynamic";
import { CircleAlert } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState, useTransition } from "react";
import { useTheme } from "@/components/theme-provider";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button, buttonVariants } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { getStoredUser, hasStoredSession, LoginError, login, persistSession, type TenantChoice } from "@/lib/api";
import { getSafeNextPath, hasAdminUIAccess } from "@/lib/rbac.mjs";

const Aurora = dynamic(() => import("@/components/Aurora"), { ssr: false });

export function LoginGate({ nextPath = null }: { nextPath?: string | null }) {
  return <LoginGateContent nextPath={nextPath} />;
}

function LoginGateContent({ nextPath = null }: { nextPath?: string | null }) {
  const router = useRouter();
  const { theme } = useTheme();
  const [isPending, startTransition] = useTransition();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [tenantID, setTenantID] = useState("");
  const [tenantChoices, setTenantChoices] = useState<TenantChoice[]>([]);
  const [error, setError] = useState("");

  useEffect(() => {
    const user = getStoredUser();
    if (user && hasStoredSession() && hasAdminUIAccess(user)) {
      router.replace(getSafeNextPath(user, nextPath));
    }
  }, [nextPath, router]);

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
        const resolvedTenant = tenantChoices.find((tenant) => tenant.tenant_id === (tenantID || session.user.tenant_id));

        if (!session.user.tenant_name && resolvedTenant) {
          session.user = {
            ...session.user,
            tenant_name: resolvedTenant.tenant_name,
            tenant_slug: resolvedTenant.tenant_slug,
          };
        }

        if (!hasAdminUIAccess(session.user)) {
          setError("This account does not have access to the admin UI.");
          return;
        }

        persistSession(session);
        router.push(getSafeNextPath(session.user, nextPath));
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
    <LoginGateFrame
      isDark={theme === "dark"}
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

function LoginGateFrame({
  isDark,
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
  isDark: boolean;
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
  const errorMessage =
    error === "Failed to fetch"
      ? "We couldn't reach the sign-in service. Check your connection and try again."
      : error;

  return (
    <div className="relative flex min-h-screen w-full items-stretch">
      <div className="relative grid min-h-screen w-full gap-0 overflow-hidden bg-white/76 backdrop-blur dark:bg-slate-950/62 lg:grid-cols-[minmax(0,2.15fr)_minmax(24rem,1fr)]">
        <section className={`relative overflow-hidden px-8 py-10 lg:px-12 lg:py-12 ${isDark ? "bg-[#06101d]" : "bg-[#edf4fb]"}`}>
          <div className="pointer-events-none absolute inset-0 overflow-hidden">
            <Aurora
              className="absolute inset-0 opacity-90 dark:opacity-100"
              colorStops={isDark ? ["#0ea5e9", "#2563eb", "#06101d"] : ["#60a5fa", "#bfdbfe", "#ecfeff"]}
              amplitude={isDark ? 1.15 : 1.35}
              blend={isDark ? 0.42 : 0.66}
              speed={isDark ? 0.9 : 0.72}
            />
            <div className="absolute inset-0 bg-[radial-gradient(circle_at_18%_22%,rgba(255,255,255,0.7),transparent_24%),radial-gradient(circle_at_76%_18%,rgba(59,130,246,0.18),transparent_22%),linear-gradient(135deg,rgba(255,255,255,0.22),rgba(255,255,255,0.02)_44%,rgba(96,165,250,0.12)_100%)] dark:bg-[radial-gradient(circle_at_18%_22%,rgba(125,211,252,0.18),transparent_24%),radial-gradient(circle_at_76%_18%,rgba(37,99,235,0.22),transparent_22%),linear-gradient(135deg,rgba(2,6,23,0.12),rgba(2,6,23,0.02)_44%,rgba(14,165,233,0.1)_100%)]" />
            <div className="absolute left-[-8%] top-[8%] h-[28rem] w-[28rem] rounded-full bg-white/55 blur-3xl dark:bg-cyan-300/10" />
            <div className="absolute right-[-12%] top-[-4%] h-[24rem] w-[26rem] rounded-full bg-sky-300/25 blur-3xl dark:bg-blue-500/18" />
            <div className="absolute bottom-[-12%] left-[28%] h-[22rem] w-[28rem] rounded-full bg-cyan-100/50 blur-3xl dark:bg-sky-400/12" />
            <div className="absolute inset-y-[-4%] right-[8%] w-[40%] rotate-[14deg] rounded-[3rem] border border-white/30 bg-[linear-gradient(180deg,rgba(255,255,255,0.36),rgba(255,255,255,0.04))] opacity-80 shadow-[0_30px_90px_rgba(59,130,246,0.12)] dark:border-white/8 dark:bg-[linear-gradient(180deg,rgba(148,163,184,0.12),rgba(15,23,42,0.02))] dark:shadow-[0_30px_100px_rgba(2,8,23,0.28)]" />
            <div className="absolute inset-0 bg-[linear-gradient(90deg,rgba(237,244,251,0.14),transparent_38%,rgba(59,130,246,0.1)_100%)] dark:bg-[linear-gradient(90deg,rgba(6,16,29,0.12),transparent_38%,rgba(37,99,235,0.08)_100%)]" />
            <div className="absolute inset-x-0 bottom-0 h-44 bg-[linear-gradient(180deg,transparent,rgba(237,244,251,0.96))] dark:bg-[linear-gradient(180deg,transparent,rgba(6,16,29,0.98))]" />
          </div>
          <div className="relative flex h-full flex-col justify-center">
            <div className="max-w-xl space-y-6">
              <div className="space-y-5">
                <h1
                  className="max-w-xl text-4xl leading-[0.95] font-semibold tracking-[-0.04em] text-slate-950 md:text-5xl lg:text-6xl dark:text-white"
                >
                  See who needs help before the exam.
                </h1>
                <p className="max-w-3xl text-base leading-8 text-slate-600 md:text-lg dark:text-slate-300">
                  P&amp;AI is a proactive AI learning agent that teaches students through chat. This workspace gives teachers, parents, and school admins visibility into mastery, momentum, and the right moment to intervene.
                </p>
              </div>
              <div className="pt-0">
                <Link
                  href="#sign-in-form"
                  className={buttonVariants({
                    variant: "default",
                    size: "lg",
                    className:
                      "h-11 rounded-full px-5 text-sm font-semibold shadow-[0_18px_44px_rgba(15,23,42,0.16)] hover:shadow-[0_22px_56px_rgba(15,23,42,0.2)] active:translate-y-px dark:shadow-[0_20px_54px_rgba(2,8,23,0.36)]",
                  })}
                >
                  Try now
                </Link>
              </div>
            </div>
          </div>
        </section>

        <div className="flex h-full border-t border-black/8 bg-black/[0.02] dark:border-white/8 dark:bg-white/[0.02] lg:border-l lg:border-t-0">
          <Card className="flex h-full w-full rounded-none border-0 bg-transparent py-0 shadow-none">
            <CardContent className="flex flex-1 flex-col justify-center px-8 py-8 lg:px-10 lg:py-10">
              <div className="mb-6 space-y-2">
                <p className="text-[11px] font-semibold uppercase tracking-[0.26em] text-slate-500 dark:text-slate-400">
                  Sign in
                </p>
                <CardTitle className="text-4xl font-semibold tracking-[-0.04em] text-slate-950 dark:text-white">
                  Admin access
                </CardTitle>
                <CardDescription className="text-sm leading-7 text-slate-600 dark:text-slate-300">
                  See your students&apos; data.
                </CardDescription>
              </div>
              <form id="sign-in-form" className="space-y-5" onSubmit={onSubmit}>
                <div className="space-y-2">
                  <Label htmlFor="email" className="text-slate-700 dark:text-slate-200">
                    Email
                  </Label>
                  <Input
                    id="email"
                    name="email"
                    type="email"
                    value={email}
                    onChange={(event) => onEmailChange(event.target.value)}
                    placeholder="teacher@school.edu"
                    autoComplete="email"
                    className="h-12 rounded-2xl border-slate-300/80 bg-white text-slate-950 placeholder:text-slate-400 dark:border-white/10 dark:bg-slate-900/80 dark:text-slate-100 dark:placeholder:text-slate-500"
                    required
                  />
                </div>

                {tenantChoices.length > 0 ? (
                  <div className="space-y-2">
                    <Label htmlFor="tenant_id" className="text-slate-700 dark:text-slate-200">
                      School
                    </Label>
                    <select
                      id="tenant_id"
                      name="tenant_id"
                      value={tenantID}
                      onChange={(event) => onTenantChange(event.target.value)}
                      className="h-12 w-full rounded-2xl border border-input bg-white px-4 text-sm text-slate-950 outline-none transition-colors focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 dark:border-white/10 dark:bg-slate-900/80 dark:text-slate-100"
                      required
                    >
                      {tenantChoices.map((tenant) => (
                        <option key={tenant.tenant_id} value={tenant.tenant_id}>
                          {tenant.tenant_name} ({tenant.tenant_slug})
                        </option>
                      ))}
                    </select>
                    <p className="text-xs leading-6 text-slate-500 dark:text-slate-400">
                      This email exists under multiple tenants, so the backend needs the school context before login.
                    </p>
                  </div>
                ) : null}

                <div className="space-y-2">
                  <Label htmlFor="password" className="text-slate-700 dark:text-slate-200">
                    Password
                  </Label>
                  <Input
                    id="password"
                    name="password"
                    type="password"
                    value={password}
                    onChange={(event) => onPasswordChange(event.target.value)}
                    placeholder="Enter your password"
                    autoComplete="current-password"
                    className="h-12 rounded-2xl border-slate-300/80 bg-white text-slate-950 placeholder:text-slate-400 dark:border-white/10 dark:bg-slate-900/80 dark:text-slate-100 dark:placeholder:text-slate-500"
                    required
                  />
                </div>

                {error ? (
                  <Alert
                    variant="destructive"
                    className="gap-y-1 rounded-2xl border-rose-200/80 bg-rose-50/90 px-4 py-3 shadow-none dark:border-rose-400/25 dark:bg-rose-500/10"
                  >
                    <CircleAlert />
                    <AlertTitle>There&apos;s a problem signing you in.</AlertTitle>
                    <AlertDescription className="leading-6 text-rose-700 dark:text-rose-200">
                      {errorMessage}
                    </AlertDescription>
                  </Alert>
                ) : null}

                <Button
                  type="submit"
                  size="lg"
                  className="h-12 w-full rounded-full bg-slate-950 text-white transition hover:bg-slate-800 active:translate-y-px dark:bg-slate-100 dark:text-slate-950 dark:hover:bg-white"
                  disabled={isPending}
                >
                  {isPending ? "Signing in..." : "Sign in"}
                </Button>
              </form>

            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
