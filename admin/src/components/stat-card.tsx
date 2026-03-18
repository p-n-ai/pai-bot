import type { ComponentType } from "react";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { cn } from "@/lib/utils";

export function StatCard({
  icon: Icon,
  title,
  value,
  note,
  className,
  headerClassName,
}: {
  icon?: ComponentType<{ className?: string }>;
  title: string;
  value: string;
  note: string;
  className?: string;
  headerClassName?: string;
}) {
  return (
    <Card
      className={cn(
        "rounded-[24px] border-white/70 bg-white/85 shadow-[0_16px_40px_rgba(15,23,42,0.04)] dark:border-white/10 dark:bg-slate-950/60 dark:shadow-[0_20px_50px_rgba(2,8,23,0.35)]",
        className,
      )}
    >
      <CardHeader className={cn(Icon ? "space-y-3" : undefined, headerClassName)}>
        {Icon ? (
          <div className="flex size-11 items-center justify-center rounded-2xl bg-sky-100 text-sky-700 dark:bg-sky-400/15 dark:text-sky-200">
            <Icon className="size-5" />
          </div>
        ) : null}
        <p className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500 dark:text-slate-400">{title}</p>
        <CardTitle className="text-3xl tracking-tight text-slate-950 dark:text-slate-50">{value}</CardTitle>
        <p className="text-sm text-slate-500 dark:text-slate-400">{note}</p>
      </CardHeader>
    </Card>
  );
}
