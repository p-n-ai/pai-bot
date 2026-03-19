import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

export function AdminHighlightPanel({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <div className={cn("grid gap-3 rounded-[24px] bg-slate-950 p-4 text-white dark:bg-slate-900/90", className)}>
      {children}
    </div>
  );
}
