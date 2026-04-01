import type { ComponentType, ReactNode } from "react";
import { Card, CardHeader, CardTitle } from "@/components/ui/card";
import { cn } from "@/lib/utils";

export function StatCard({
  icon: Icon,
  title,
  value,
  note,
  noteClassName,
  className,
  headerClassName,
}: {
  icon?: ComponentType<{ className?: string }>;
  title: string;
  value: ReactNode;
  note: ReactNode;
  noteClassName?: string;
  className?: string;
  headerClassName?: string;
}) {
  return (
    <Card
      className={cn(
        "rounded-xl border bg-card shadow-xs",
        className,
      )}
    >
      <CardHeader className={cn(Icon ? "space-y-3" : undefined, headerClassName)}>
        {Icon ? (
          <div className="flex size-11 items-center justify-center rounded-xl bg-primary/10 text-primary">
            <Icon className="size-5" />
          </div>
        ) : null}
        <p className="text-xs font-medium tracking-[0.08em] text-muted-foreground">{title}</p>
        <CardTitle className="text-3xl tracking-tight text-foreground">{value}</CardTitle>
        <div className={cn("text-sm text-muted-foreground", noteClassName)}>{note}</div>
      </CardHeader>
    </Card>
  );
}
