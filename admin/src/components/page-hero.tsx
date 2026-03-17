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
        "grid gap-4 rounded-[28px] border border-white/70 bg-white/80 p-6 shadow-[0_18px_60px_rgba(15,23,42,0.08)] backdrop-blur dark:border-white/10 dark:bg-slate-950/60 dark:shadow-[0_24px_80px_rgba(2,8,23,0.4)]",
        aside ? "lg:grid-cols-[1.3fr_0.7fr]" : undefined,
        className,
      )}
    >
      <div className={cn("space-y-3", contentClassName)}>
        <p className="text-xs font-semibold uppercase tracking-[0.24em] text-sky-700 dark:text-sky-300">{eyebrow}</p>
        <h1 className="text-3xl font-semibold tracking-tight text-slate-950 dark:text-white">{title}</h1>
        <div className="max-w-2xl text-sm leading-6 text-slate-600 dark:text-slate-300">{description}</div>
        {children}
      </div>
      {aside}
    </header>
  );
}
