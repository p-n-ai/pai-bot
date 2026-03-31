import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

export function AdminInsetPanel({
  title,
  children,
  className,
}: {
  title?: ReactNode;
  children: ReactNode;
  className?: string;
}) {
  return (
    <div className={cn("rounded-[24px] border border-slate-200/70 bg-white/85 p-4 dark:border-white/10 dark:bg-slate-950/45", className)}>
      {title ? <p className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500 dark:text-slate-400">{title}</p> : null}
      <div className={title ? "mt-3" : undefined}>{children}</div>
    </div>
  );
}
