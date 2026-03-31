"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import { LoginGateContext } from "@/components/login-gate/login-gate-context";
import { LoginError, login, persistSession, type TenantChoice } from "@/lib/api";
import { getSafeNextPath, hasAdminUIAccess } from "@/lib/rbac.mjs";

export function LoginGateProvider({
  children,
  nextPath,
}: {
  children: React.ReactNode;
  nextPath: string | null;
}) {
  const router = useRouter();
  const [isPending, startTransition] = useTransition();
  const [email, setEmailState] = useState("");
  const [password, setPassword] = useState("");
  const [tenantID, setTenantID] = useState("");
  const [tenantChoices, setTenantChoices] = useState<TenantChoice[]>([]);
  const [error, setError] = useState("");

  function setEmail(value: string) {
    setEmailState(value);
    if (tenantChoices.length > 0) {
      setTenantChoices([]);
      setTenantID("");
      setError("");
    }
  }

  function submit(event: React.FormEvent<HTMLFormElement>) {
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
          setError("");
          return;
        }

        setTenantChoices([]);
        setTenantID("");
        setError(err instanceof Error ? err.message : "Login failed");
      }
    });
  }

  return (
    <LoginGateContext
      value={{
        email,
        password,
        tenantID,
        tenantChoices,
        error,
        isPending,
        setEmail,
        setPassword,
        setTenantID,
        submit,
      }}
    >
      {children}
    </LoginGateContext>
  );
}
