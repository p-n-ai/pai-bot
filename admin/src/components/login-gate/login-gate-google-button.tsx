"use client";

import { IconBrandGoogle } from "@tabler/icons-react";
import { Button } from "@/components/ui/button";
import { useLoginGate } from "@/components/login-gate/use-login-gate";

export function LoginGateGoogleButton() {
  const { tenantChoices, isPending, isGooglePending, startGoogleLogin } = useLoginGate();

  if (tenantChoices.length > 0) {
    return null;
  }

  return (
    <Button
      type="button"
      variant="outline"
      size="lg"
      onClick={startGoogleLogin}
      disabled={isPending || isGooglePending}
      className="h-12 w-full rounded-full border-slate-300/80 bg-white text-slate-950 transition-all duration-150 ease-out hover:bg-slate-100 dark:border-white/10 dark:bg-slate-900/80 dark:text-slate-100 dark:hover:bg-slate-900"
    >
      <IconBrandGoogle data-icon="inline-start" className="size-4" />
      {isGooglePending ? "Redirecting to Google..." : "Continue with Google"}
    </Button>
  );
}
