"use client";

import { GearSix } from "@phosphor-icons/react";
import { useState } from "react";
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
import type { AuthUser } from "@/lib/api";

export function AccountSettingsDialog({
  currentUser,
  nextPath,
}: {
  currentUser: AuthUser | null;
  nextPath: string | null;
}) {
  const [open, setOpen] = useState(false);

  function handleOpenChange(nextOpen: boolean) {
    setOpen(nextOpen);
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
            Manage sign-in methods for this account.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <LinkedIdentitiesCard enabled={Boolean(currentUser)} nextPath={nextPath} />
        </div>

        <DialogFooter showCloseButton />
      </DialogContent>
    </Dialog>
  );
}
