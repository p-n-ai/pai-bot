"use client";

import { Suspense, useTransition } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { InviteAcceptanceCard } from "@/components/invite-acceptance-card";
import { useInviteActivationFlowBootstrap } from "@/hooks/use-auth-flow-bootstrap";
import { useSessionRedirect } from "@/hooks/use-session-redirect";
import { acceptInvite, persistSession } from "@/lib/api";
import { hasAdminUIAccess } from "@/lib/rbac.mjs";
import { useAppStore } from "@/stores/app-store";

export default function ActivatePage() {
  return (
    <Suspense fallback={<ActivatePageFallback />}>
      <ActivatePageContent />
    </Suspense>
  );
}

function ActivatePageContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const isHydrated = useAppStore((state) => state.hydrated);
  const currentUser = useAppStore((state) => state.currentUser);
  const hasActiveSession = useAppStore((state) => state.isLoggedIn) && Boolean(currentUser && hasAdminUIAccess(currentUser));
  const [, startTransition] = useTransition();
  const token = searchParams.get("token") || "";
  const inviteActivationFlow = useAppStore((state) => state.inviteActivationFlow);
  const setName = useAppStore((state) => state.setInviteActivationName);
  const setPassword = useAppStore((state) => state.setInviteActivationPassword);
  const startInviteActivationSubmit = useAppStore((state) => state.startInviteActivationSubmit);
  const failInviteActivation = useAppStore((state) => state.failInviteActivation);
  const isPending = inviteActivationFlow.phase.kind === "submitting";
  const error = inviteActivationFlow.phase.kind === "error" ? inviteActivationFlow.phase.message : "";

  useInviteActivationFlowBootstrap(token);

  useSessionRedirect({
    enabled: isHydrated,
    router,
    user: hasActiveSession ? currentUser : null,
  });

  if (!isHydrated) {
    return <ActivatePageFallback />;
  }

  if (hasActiveSession) {
    return null;
  }

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();

    if (!token.trim()) {
      failInviteActivation("Invite token missing. Open the full invite link from the email and try again.");
      return;
    }

    startInviteActivationSubmit();

    startTransition(async () => {
      try {
        const session = await acceptInvite({
          token,
          name: inviteActivationFlow.draft.name.trim(),
          password: inviteActivationFlow.draft.password,
        });
        persistSession(session);
        router.push("/");
      } catch (err) {
        failInviteActivation(err instanceof Error ? err.message : "Invite activation failed");
      }
    });
  }

  return (
    <InviteAcceptanceCard
      token={token}
      name={inviteActivationFlow.draft.name}
      password={inviteActivationFlow.draft.password}
      error={error}
      isPending={isPending}
      onNameChange={setName}
      onPasswordChange={setPassword}
      onSubmit={handleSubmit}
    />
  );
}

function ActivatePageFallback() {
  return (
    <InviteAcceptanceCard
      token=""
      name=""
      password=""
      error=""
      isPending={false}
      onNameChange={() => {}}
      onPasswordChange={() => {}}
      onSubmit={(event) => event.preventDefault()}
    />
  );
}
