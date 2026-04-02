import type { ReactNode } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { cn } from "@/lib/utils";

export function AdminSurface({
  children,
  className,
  contentClassName,
}: {
  children: ReactNode;
  className?: string;
  contentClassName?: string;
}) {
  return (
    <Card
      className={cn(
        "rounded-xl border bg-card shadow-xs",
        className,
      )}
    >
      <CardContent className={cn("p-6", contentClassName)}>{children}</CardContent>
    </Card>
  );
}

export function AdminSurfaceHeader({
  title,
  description,
  action,
  className,
}: {
  title: ReactNode;
  description?: ReactNode;
  action?: ReactNode;
  className?: string;
}) {
  return (
    <CardHeader className={cn("flex flex-row items-center justify-between gap-3 p-0", className)}>
      <div>
        <CardTitle className="text-xl tracking-tight text-foreground">{title}</CardTitle>
        {description ? <p className="mt-1 text-sm text-muted-foreground">{description}</p> : null}
      </div>
      {action}
    </CardHeader>
  );
}
