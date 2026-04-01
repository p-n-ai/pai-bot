import { ReactNode } from "react";
import { cn } from "@/lib/utils";

export function PageHero({
  eyebrow,
  title,
  description,
  children,
  aside,
  className,
  contentClassName,
}: {
  eyebrow: string;
  title: ReactNode;
  description: ReactNode;
  children?: ReactNode;
  aside?: ReactNode;
  className?: string;
  contentClassName?: string;
}) {
  return (
    <header
      className={cn(
        "grid gap-4 rounded-xl border bg-card p-6 text-card-foreground shadow-xs",
        aside ? "lg:grid-cols-[1.3fr_0.7fr]" : undefined,
        className,
      )}
    >
      <div className={cn("space-y-3", contentClassName)}>
        <p className="text-xs font-medium tracking-[0.08em] text-muted-foreground">{eyebrow}</p>
        <h1 className="text-3xl font-semibold tracking-tight text-foreground">{title}</h1>
        <div className="max-w-2xl text-sm leading-6 text-muted-foreground">{description}</div>
        {children}
      </div>
      {aside}
    </header>
  );
}
