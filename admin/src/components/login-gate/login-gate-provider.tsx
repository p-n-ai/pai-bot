"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import { LoginGateContext } from "@/components/login-gate/login-gate-context";
import { LoginError, login, persistSession, type TenantChoice } from "@/lib/api";
import { getSafeNextPath, hasAdminUIAccess } from "@/lib/rbac.mjs";
import { clearSchoolSwitchState, readSchoolSwitchState, writeSchoolSwitchState } from "@/lib/school-switch-state";

export function LoginGateProvider({
  children,
  nextPath,
}: {
  children: React.ReactNode;
  nextPath: string | null;
}) {
  const router = useRouter();
  const [initialSchoolSwitchState] = useState(() => readSchoolSwitchState());
  const [isPending, startTransition] = useTransition();
  const [email, setEmailState] = useState(initialSchoolSwitchState?.email ?? "");
  const [password, setPassword] = useState("");
  const [tenantID, setTenantID] = useState(initialSchoolSwitchState?.currentTenantID ?? "");
  const [tenantChoices, setTenantChoices] = useState<TenantChoice[]>(initialSchoolSwitchState?.tenantChoices ?? []);
  const [error, setError] = useState("");

  function setEmail(value: string) {
    setEmailState(value);
    if (tenantChoices.length > 0) {
      setTenantChoices([]);
      setTenantID("");
      setError("");
    }
    clearSchoolSwitchState();
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

        if (tenantChoices.length > 0) {
          writeSchoolSwitchState({
            email: session.user.email,
            currentTenantID: tenantID || session.user.tenant_id,
            tenantChoices,
          });
        } else {
          clearSchoolSwitchState();
        }

        persistSession(session);
        router.push(getSafeNextPath(session.user, nextPath));
      } catch (err) {
        if (err instanceof LoginError && err.code === "tenant_required") {
          setTenantChoices(err.tenants);
          const resolvedTenantID = tenantID || err.tenants[0]?.tenant_id || "";
          setTenantID(resolvedTenantID);
          writeSchoolSwitchState({
            email: email.trim(),
            currentTenantID: resolvedTenantID,
            tenantChoices: err.tenants,
          });
          setError("");
          return;
        }

        setTenantChoices([]);
        setTenantID("");
        clearSchoolSwitchState();
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
