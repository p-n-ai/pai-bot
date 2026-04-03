"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { IconLogout2 } from "@tabler/icons-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { logout } from "@/lib/api";

export function LogoutButton() {
  const router = useRouter();
  const [pending, setPending] = useState(false);

  async function handleLogout() {
    setPending(true);
    try {
      await logout();
      router.push("/login");
      router.refresh();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Sign out failed");
    } finally {
      setPending(false);
    }
  }

  return (
    <Button
      type="button"
      variant="destructive"
      size="default"
      onClick={handleLogout}
      disabled={pending}
      className="w-full justify-center rounded-xl"
    >
      <IconLogout2 data-icon="inline-start" />
      {pending ? "Signing out..." : "Log out"}
    </Button>
  );
}
