"use client";

import { useQuery } from "@tanstack/react-query";
import { IconBrandGoogle, IconPlugConnected } from "@tabler/icons-react";
import { formatDistanceToNowStrict } from "date-fns";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { getLinkedIdentities, startGoogleLink } from "@/lib/api";

export function LinkedIdentitiesCard({
  enabled = true,
  nextPath,
}: {
  enabled?: boolean;
  nextPath: string | null;
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
    <div className="rounded-2xl border border-sidebar-border/80 bg-background/75 p-3">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-sidebar-foreground/55">
            Sign-in methods
          </p>
          <div className="mt-2 flex items-center gap-2 text-sm font-medium text-sidebar-foreground">
            <IconBrandGoogle className="size-4 text-[#0f172a] dark:text-white" />
            <span>{googleIdentity ? "Google linked" : "Google available"}</span>
          </div>
          <p className="mt-1 text-xs leading-5 text-sidebar-foreground/70">
            {isLoading
              ? "Checking linked providers..."
              : isError
                ? "We couldn't load linked providers right now."
              : googleIdentity
                ? googleIdentity.email
                : "Link Google to let the same person sign in without typing a password."}
          </p>
          {googleIdentity?.last_used_at ? (
            <p className="mt-1 text-[11px] text-sidebar-foreground/55">
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
