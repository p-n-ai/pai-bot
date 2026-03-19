import Link from "next/link";
import type { ComponentType } from "react";
import { cn } from "@/lib/utils";

export function AdminLinkCard({
  href,
  title,
  description,
  icon: Icon,
  className,
}: {
  href: string;
  title: string;
  description: string;
  icon?: ComponentType<{ className?: string }>;
  className?: string;
}) {
  return (
    <Link
      href={href}
      className={cn(
        "group rounded-[28px] border border-slate-200/70 bg-white/80 p-6 shadow-[0_16px_50px_rgba(15,23,42,0.05)] transition hover:-translate-y-1 hover:border-sky-300 hover:shadow-[0_24px_80px_rgba(14,165,233,0.14)] dark:border-white/10 dark:bg-slate-950/55 dark:shadow-[0_20px_60px_rgba(2,8,23,0.35)] dark:hover:border-sky-400 dark:hover:shadow-[0_24px_80px_rgba(56,189,248,0.16)]",
        className,
      )}
    >
      {Icon ? <Icon className="mb-5 size-6 text-sky-600 transition group-hover:text-amber-500 dark:text-sky-300 dark:group-hover:text-amber-300" /> : null}
      <h2 className="text-xl font-semibold tracking-tight dark:text-white">{title}</h2>
      <p className="mt-3 text-sm leading-6 text-slate-600 dark:text-slate-300">{description}</p>
    </Link>
  );
}
