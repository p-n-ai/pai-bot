"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Buildings, GearSix } from "@phosphor-icons/react";
import { useRouter } from "next/navigation";
import { startTransition, useMemo, useState } from "react";
import { toast } from "sonner";
import { LinkedIdentitiesCard } from "@/components/account/linked-identities-card";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import type { AuthUser, TenantChoice } from "@/lib/api";
import { persistSession, switchTenantSession } from "@/lib/api";
import { fetchDashboardProgress, fetchPreviewDashboardProgress, getDashboardProgressQueryKey } from "@/lib/dashboard-progress-query";
import { type SchoolSwitchState } from "@/lib/school-switch-state";

export function AccountSettingsDialog({
  currentUser,
  schoolSwitchState,
  nextPath,
}: {
  currentUser: AuthUser | null;
  schoolSwitchState: SchoolSwitchState | null;
  nextPath: string | null;
}) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [tenantID, setTenantID] = useState(currentUser?.tenant_id ?? "");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");

  const schoolChoices = useMemo<TenantChoice[]>(() => {
    if (!currentUser?.email || schoolSwitchState?.email !== currentUser.email) {
      return [];
    }
    return schoolSwitchState.tenantChoices;
  }, [currentUser?.email, schoolSwitchState]);

  const canSwitchSchools = schoolChoices.length > 1 && Boolean(currentUser?.tenant_id);

  const switchSchoolMutation = useMutation({
    mutationFn: async ({ nextTenantID, currentPassword }: { nextTenantID: string; currentPassword: string }) => {
      if (!currentUser) {
        throw new Error("A signed-in session is required to switch schools");
      }
      return switchTenantSession(nextTenantID, currentPassword);
    },
    onMutate: () => {
      setError("");
      toast.loading("Switching school...", { id: "school-switch" });
    },
    onSuccess: async (session) => {
      persistSession(session);
      setPassword("");
      setTenantID(session.user.tenant_id);
      setOpen(false);
      const dashboardQueryKey = getDashboardProgressQueryKey(session.user.tenant_id);
      try {
        await queryClient.ensureQueryData({
          queryKey: dashboardQueryKey,
          queryFn: () => fetchDashboardProgress(session.user.tenant_id),
        });
      } catch {
        queryClient.setQueryData(dashboardQueryKey, await fetchPreviewDashboardProgress());
      }
      toast.success(`School changed to ${session.user.tenant_name}.`, { id: "school-switch" });
      startTransition(() => {
        router.replace("/dashboard");
      });
    },
    onError: (mutationError) => {
      const message = mutationError instanceof Error ? mutationError.message : "Couldn't switch schools right now.";
      setError(message);
      toast.error(message, { id: "school-switch" });
    },
  });

  function handleOpenChange(nextOpen: boolean) {
    setOpen(nextOpen);
    if (nextOpen) {
      setTenantID(currentUser?.tenant_id ?? "");
      setPassword("");
      setError("");
    }
  }

  function handleSwitchSchool() {
    if (!currentUser || !canSwitchSchools || !tenantID || tenantID === currentUser.tenant_id || !password.trim()) {
      return;
    }
    switchSchoolMutation.mutate({
      nextTenantID: tenantID,
      currentPassword: password,
    });
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger
        render={
          <Button
            type="button"
            variant="outline"
            size="default"
            className="w-full justify-center rounded-xl"
          />
        }
      >
        <GearSix data-icon="inline-start" />
        Settings
      </DialogTrigger>
      <DialogContent className="max-w-lg gap-5">
        <DialogHeader>
          <DialogTitle>Account settings</DialogTitle>
          <DialogDescription>
            Manage sign-in methods and switch schools after you log in.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="rounded-2xl border border-border/80 bg-muted/30 p-4">
            <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">
              Current workspace
            </p>
            <div className="mt-3 flex items-center gap-3">
              <div className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-background text-muted-foreground">
                <Buildings className="size-5" weight="duotone" />
              </div>
              <div className="min-w-0">
                <p className="truncate text-sm font-semibold text-foreground">
                  {currentUser?.tenant_name || "No school selected"}
                </p>
                <p className="truncate text-xs text-muted-foreground">
                  {currentUser?.tenant_id ? "Active school session" : "Platform-level session"}
                </p>
              </div>
            </div>
          </div>

          {canSwitchSchools ? (
            <div className="rounded-2xl border border-border/80 bg-muted/30 p-4">
              <div className="space-y-1">
                <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">
                  Switch school
                </p>
                <p className="text-sm text-muted-foreground">
                  Pick another school for this account and confirm with your password.
                </p>
              </div>
              <div className="mt-4 space-y-3">
                <div className="space-y-2">
                  <Label htmlFor="settings-tenant">School</Label>
                  <Select value={tenantID} onValueChange={(value) => setTenantID(value ?? "")}>
                    <SelectTrigger id="settings-tenant" className="rounded-xl">
                      <SelectValue placeholder="Choose school" />
                    </SelectTrigger>
                    <SelectContent>
                      {schoolChoices.map((tenant) => (
                        <SelectItem key={tenant.tenant_id} value={tenant.tenant_id}>
                          {tenant.tenant_name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="settings-password">Password</Label>
                  <Input
                    id="settings-password"
                    type="password"
                    value={password}
                    onChange={(event) => setPassword(event.target.value)}
                    placeholder="Enter password"
                    autoComplete="current-password"
                    disabled={switchSchoolMutation.isPending}
                  />
                </div>
                {error ? <p className="text-sm text-destructive">{error}</p> : null}
                <Button
                  type="button"
                  onClick={handleSwitchSchool}
                  disabled={
                    switchSchoolMutation.isPending ||
                    !password.trim() ||
                    !tenantID ||
                    tenantID === currentUser?.tenant_id
                  }
                  className="rounded-xl"
                >
                  {switchSchoolMutation.isPending ? "Switching..." : "Switch school"}
                </Button>
              </div>
            </div>
          ) : null}

          <LinkedIdentitiesCard enabled={Boolean(currentUser)} nextPath={nextPath} />
        </div>

        <DialogFooter showCloseButton />
      </DialogContent>
    </Dialog>
  );
}
