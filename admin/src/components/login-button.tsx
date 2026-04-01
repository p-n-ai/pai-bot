"use client";

import { useRouter } from "next/navigation";
import { IconLogin2 } from "@tabler/icons-react";
import { Button } from "@/components/ui/button";

export function LoginButton({ onClick }: { onClick?: () => void }) {
  const router = useRouter();

  return (
    <Button
      type="button"
      variant="outline"
      size="default"
      onClick={() => {
        onClick?.();
        router.push("/login");
      }}
      className="w-full justify-center rounded-xl"
    >
      <IconLogin2 data-icon="inline-start" />
      Log in
    </Button>
  );
}
