"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useEffect, useState, useTransition } from "react";
import { LoginFormCard } from "@/components/login-form-card";
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
    <LoginFormCard
      email={email}
      password={password}
      tenantID={tenantID}
      tenantChoices={tenantChoices}
      error={error}
      isPending={isPending}
      onEmailChange={onEmailChange}
      onPasswordChange={onPasswordChange}
      onTenantChange={onTenantChange}
      onSubmit={onSubmit}
    />
  );
}
