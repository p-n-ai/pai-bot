"use client";

import { useQuery } from "@tanstack/react-query";
import { IconBrandGoogle, IconPlugConnected } from "@tabler/icons-react";
import { formatDistanceToNowStrict } from "date-fns";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { getLinkedIdentities, startGoogleLink } from "@/lib/api";
import { cn } from "@/lib/utils";

export function LinkedIdentitiesCard({
  enabled = true,
  nextPath,
  className,
}: {
  enabled?: boolean;
  nextPath: string | null;
  className?: string;
}) {
  const { data, isError, isLoading } = useQuery({
    queryKey: ["auth", "identities"],
    queryFn: getLinkedIdentities,
    enabled,
    staleTime: 30_000,
  });

  if (!enabled) {
    return null;
  }

  const googleIdentity = data?.find((identity) => identity.provider === "google") ?? null;

  async function handleStartLink() {
    try {
      const target = await startGoogleLink(nextPath);
      window.location.assign(target);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Couldn't start Google linking.");
    }
  }

  return (
    <div className={cn("rounded-2xl border border-border/80 bg-muted/30 p-3", className)}>
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">
            Sign-in methods
          </p>
          <div className="mt-2 flex items-center gap-2 text-sm font-medium text-foreground">
            <IconBrandGoogle className="size-4 text-foreground" />
            <span>{googleIdentity ? "Google linked" : "Google available"}</span>
          </div>
          <p className="mt-1 text-xs leading-5 text-muted-foreground">
            {isLoading
              ? "Checking linked providers..."
              : isError
                ? "We couldn't load linked providers right now."
              : googleIdentity
                ? googleIdentity.email
                : "Link Google to let the same person sign in without typing a password."}
          </p>
          {googleIdentity?.last_used_at ? (
            <p className="mt-1 text-[11px] text-muted-foreground">
              Last used {formatDistanceToNowStrict(new Date(googleIdentity.last_used_at), { addSuffix: true })}
            </p>
          ) : null}
        </div>
        <Button
          type="button"
          variant={googleIdentity ? "secondary" : "outline"}
          size="sm"
          onClick={() => {
            void handleStartLink();
          }}
          className="shrink-0 rounded-xl"
        >
          <IconPlugConnected data-icon="inline-start" />
          {googleIdentity ? "Change link" : "Link Google"}
        </Button>
      </div>
    </div>
  );
}
