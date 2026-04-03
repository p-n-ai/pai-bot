"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import { LoginGateContext } from "@/components/login-gate/login-gate-context";
import { buildGoogleLoginURL, login, persistSession } from "@/lib/api";
import { getGoogleAuthErrorMessage } from "@/lib/auth-flow-feedback";
import { getSafeNextPath, hasAdminUIAccess } from "@/lib/rbac.mjs";
import { clearSchoolSwitchState } from "@/lib/school-switch-state";

function mapGoogleAuthError(code: string | null): string {
  return getGoogleAuthErrorMessage(code);
}

export function LoginGateProvider({
  children,
  nextPath,
  authError,
}: {
  children: React.ReactNode;
  nextPath: string | null;
  authError: string | null;
}) {
  const router = useRouter();
  const [isPending, startTransition] = useTransition();
  const [email, setEmailState] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState(() => mapGoogleAuthError(authError));
  const [isGooglePending, setGooglePending] = useState(false);

  function setEmail(value: string) {
    setEmailState(value);
    setError("");
  }

  function startGoogleLogin() {
    if (isPending || isGooglePending) {
      return;
    }
    setError("");
    setGooglePending(true);
    window.location.assign(buildGoogleLoginURL(nextPath));
  }

  function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");

    startTransition(async () => {
      try {
        const session = await login({
          email: email.trim(),
          password,
        });

        if (!hasAdminUIAccess(session.user)) {
          setError("This account does not have access to the admin UI.");
          return;
        }

        persistSession(session);
        router.push(getSafeNextPath(session.user, nextPath));
      } catch (err) {
        clearSchoolSwitchState();
        setGooglePending(false);
        setError(err instanceof Error ? err.message : "Login failed");
      }
    });
  }

  return (
    <LoginGateContext
      value={{
        email,
        password,
        error,
        isPending,
        isGooglePending,
        setEmail,
        setPassword: (value) => {
          setPassword(value);
          setError("");
        },
        submit,
        startGoogleLogin,
      }}
    >
      {children}
    </LoginGateContext>
  );
}
