import { ReactNode } from "react";
import { cn } from "@/lib/utils";

export function PageHero({
  eyebrow,
  title,
  description,
  children,
  aside,
  surface = "card",
  className,
  contentClassName,
}: {
  eyebrow: string;
  title: ReactNode;
  description: ReactNode;
  children?: ReactNode;
  aside?: ReactNode;
  surface?: "card" | "plain";
  className?: string;
  contentClassName?: string;
}) {
  return (
    <header
      className={cn(
        "grid gap-4 text-card-foreground",
        surface === "card" ? "rounded-xl border bg-card p-6 shadow-xs" : "p-0",
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
