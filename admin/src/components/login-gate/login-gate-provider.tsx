"use client";

import { useRouter } from "next/navigation";
import { useTransition } from "react";
import { LoginGateContext } from "@/components/login-gate/login-gate-context";
import { buildGoogleLoginURL, loginWithPassword, persistSession } from "@/lib/api";
import { getGoogleAuthErrorMessage } from "@/lib/auth-flow-feedback";
import { useLoginFlowBootstrap } from "@/hooks/use-auth-flow-bootstrap";
import { getSafeNextPath, hasAdminUIAccess } from "@/lib/rbac.mjs";
import { clearSchoolSwitchState } from "@/lib/school-switch-state";
import { useAppStore } from "@/stores/app-store";

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
  const [, startTransition] = useTransition();
  const seed = authError ?? "__default__";
  const loginFlow = useAppStore((state) => state.loginFlow);
  const setEmail = useAppStore((state) => state.setLoginEmail);
  const setPassword = useAppStore((state) => state.setLoginPassword);
  const startLoginSubmit = useAppStore((state) => state.startLoginSubmit);
  const startLoginGoogleRedirect = useAppStore((state) => state.startLoginGoogleRedirect);
  const failLogin = useAppStore((state) => state.failLogin);
  const initialError = mapGoogleAuthError(authError);
  const isPending = loginFlow.phase.kind === "submitting";
  const isGooglePending = loginFlow.phase.kind === "redirecting_google";
  const error = loginFlow.phase.kind === "error" ? loginFlow.phase.message : "";

  useLoginFlowBootstrap(seed, initialError);

  function startGoogleLogin() {
    if (isPending || isGooglePending) {
      return;
    }
    startLoginGoogleRedirect();
    window.location.assign(buildGoogleLoginURL(nextPath));
  }

  function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    startLoginSubmit();

    startTransition(async () => {
      try {
        const session = await loginWithPassword({
          email: loginFlow.draft.email.trim(),
          password: loginFlow.draft.password,
        });

        if (!hasAdminUIAccess(session.user)) {
          failLogin("This account does not have access to the admin UI.");
          return;
        }

        persistSession(session);
        router.push(nextPath && nextPath !== "/" && nextPath !== "/login" ? getSafeNextPath(session.user, nextPath) : "/");
      } catch (err) {
        clearSchoolSwitchState();
        failLogin(err instanceof Error ? err.message : "Login failed");
      }
    });
  }

  return (
    <LoginGateContext
      value={{
        email: loginFlow.draft.email,
        password: loginFlow.draft.password,
        error,
        isPending,
        isGooglePending,
        setEmail,
        setPassword,
        submit,
        startGoogleLogin,
      }}
    >
      {children}
    </LoginGateContext>
  );
}
