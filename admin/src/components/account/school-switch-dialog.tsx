"use client";

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Buildings } from "@phosphor-icons/react";
import { useRouter } from "next/navigation";
import { startTransition, useState } from "react";
import { toast } from "sonner";
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
import type { AuthUser, SchoolChoice } from "@/lib/api";
import { persistSession, switchSchool } from "@/lib/api";
import { fetchDashboardProgress, fetchPreviewDashboardProgress, getDashboardProgressQueryKey } from "@/lib/dashboard-progress-query";

export function SchoolSwitchDialog({
  currentUser,
  schoolChoices,
  triggerLabel = "Switch school",
  triggerClassName,
}: {
  currentUser: AuthUser | null;
  schoolChoices: SchoolChoice[];
  triggerLabel?: string;
  triggerClassName?: string;
}) {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [selectedSchoolID, setSelectedSchoolID] = useState(currentUser?.tenant_id ?? "");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");

  const canSwitchSchools = schoolChoices.length > 1 && Boolean(currentUser?.tenant_id);

  const switchSchoolMutation = useMutation({
    mutationFn: async ({ nextSchoolID, currentPassword }: { nextSchoolID: string; currentPassword: string }) => {
      if (!currentUser) {
        throw new Error("A signed-in session is required to switch schools");
      }
      return switchSchool(nextSchoolID, currentPassword);
    },
    onMutate: () => {
      setError("");
      toast.loading("Switching school...", { id: "school-switch" });
    },
    onSuccess: async (session) => {
      persistSession(session);
      setPassword("");
      setSelectedSchoolID(session.user.tenant_id);
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
      setSelectedSchoolID(currentUser?.tenant_id ?? "");
      setPassword("");
      setError("");
    }
  }

  function handleSwitchSchool() {
    if (!currentUser || !canSwitchSchools || !selectedSchoolID || selectedSchoolID === currentUser.tenant_id || !password.trim()) {
      return;
    }
    switchSchoolMutation.mutate({
      nextSchoolID: selectedSchoolID,
      currentPassword: password,
    });
  }

  if (!canSwitchSchools) {
    return null;
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger
        render={
          <Button type="button" variant="outline" size="sm" className={triggerClassName}>
            <Buildings data-icon="inline-start" />
            {triggerLabel}
          </Button>
        }
      />
      <DialogContent className="max-w-lg gap-5">
        <DialogHeader>
          <DialogTitle>Switch school</DialogTitle>
          <DialogDescription>
            Pick another school for this account and confirm with your password.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 rounded-2xl border border-border/80 bg-muted/30 p-4">
          <div className="space-y-2">
            <Label htmlFor="school-switcher-tenant">School</Label>
            <Select value={selectedSchoolID} onValueChange={(value) => setSelectedSchoolID(value ?? "")}>
              <SelectTrigger id="school-switcher-tenant" className="rounded-xl">
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
            <Label htmlFor="school-switcher-password">Password</Label>
            <Input
              id="school-switcher-password"
              type="password"
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              placeholder="Enter password"
              autoComplete="current-password"
              disabled={switchSchoolMutation.isPending}
            />
          </div>

          {error ? <p className="text-sm text-destructive">{error}</p> : null}
        </div>

        <DialogFooter>
          <Button
            type="button"
            onClick={handleSwitchSchool}
            disabled={
              switchSchoolMutation.isPending ||
              !password.trim() ||
              !selectedSchoolID ||
              selectedSchoolID === currentUser?.tenant_id
            }
            className="rounded-xl"
          >
            {switchSchoolMutation.isPending ? "Switching..." : "Switch school"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
