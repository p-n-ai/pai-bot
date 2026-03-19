"use client";

import { Suspense, useEffect, useState, useTransition } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { InviteAcceptanceCard } from "@/components/invite-acceptance-card";
import { acceptInvite, getStoredUser, hasStoredSession, persistSession } from "@/lib/api";
import { getSafeNextPath, hasAdminUIAccess } from "@/lib/rbac.mjs";

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
  const [isPending, startTransition] = useTransition();
  const [name, setName] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const token = searchParams.get("token") || "";
  const currentUser = getStoredUser();
  const hasActiveSession = Boolean(currentUser && hasStoredSession() && hasAdminUIAccess(currentUser));

  useEffect(() => {
    if (hasActiveSession && currentUser) {
      router.replace(getSafeNextPath(currentUser, null));
    }
  }, [currentUser, hasActiveSession, router]);

  if (hasActiveSession) {
    return null;
  }

  function handleSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");

    if (!token.trim()) {
      setError("Invite token missing. Open the full invite link from the email and try again.");
      return;
    }

    startTransition(async () => {
      try {
        const session = await acceptInvite({
          token,
          name: name.trim(),
          password,
        });
        persistSession(session);
        router.push(getSafeNextPath(session.user, null));
      } catch (err) {
        setError(err instanceof Error ? err.message : "Invite activation failed");
      }
    });
  }

  return (
    <InviteAcceptanceCard
      token={token}
      name={name}
      password={password}
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
