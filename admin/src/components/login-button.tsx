"use client";

import { useRouter } from "next/navigation";
import { LogIn } from "lucide-react";
import { Button } from "@/components/ui/button";

export function LoginButton({ onClick }: { onClick?: () => void }) {
  const router = useRouter();

  return (
    <Button
      type="button"
      variant="outline"
      size="sm"
      onClick={() => {
        onClick?.();
        router.push("/login");
      }}
      className="w-full rounded-full border-white/50 bg-white/75 text-slate-700 shadow-[0_12px_30px_rgba(15,23,42,0.08)] backdrop-blur hover:bg-white dark:border-white/10 dark:bg-slate-950/75 dark:text-slate-100 dark:hover:bg-slate-900"
    >
      <LogIn className="size-4" />
      Login
    </Button>
  );
}
