"use client";

import { IconBrandGoogle } from "@tabler/icons-react";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { useLoginGate } from "@/components/login-gate/use-login-gate";

export function LoginGateGoogleButton() {
  const { isPending, isGooglePending, startGoogleLogin } = useLoginGate();

  return (
    <Button
      type="button"
      variant="outline"
      size="lg"
      onClick={startGoogleLogin}
      disabled={isPending || isGooglePending}
      className="h-12 w-full rounded-full border-border bg-background text-foreground transition-all duration-150 ease-out hover:bg-accent"
    >
      {isGooglePending ? (
        <Spinner aria-hidden="true" />
      ) : (
        <IconBrandGoogle data-icon="inline-start" />
      )}
      Continue with Google
    </Button>
  );
}
