import { AlertTriangle, FileSearch, LoaderCircle } from "lucide-react";
import { cn } from "@/lib/utils";

const toneClasses = {
  loading: {
    icon: LoaderCircle,
    iconClassName: "animate-spin text-sky-600 dark:text-sky-300",
    panelClassName: "border-sky-200/70 bg-sky-50/80 dark:border-sky-400/20 dark:bg-sky-400/10",
  },
  empty: {
    icon: FileSearch,
    iconClassName: "text-slate-500 dark:text-slate-300",
    panelClassName: "border-slate-200/80 bg-slate-50/80 dark:border-white/10 dark:bg-slate-900/70",
  },
  error: {
    icon: AlertTriangle,
    iconClassName: "text-rose-600 dark:text-rose-300",
    panelClassName: "border-rose-200/70 bg-rose-50/85 dark:border-rose-400/20 dark:bg-rose-400/10",
  },
} as const;

export function StatePanel({
  tone,
  title,
  description,
  className,
}: {
  tone: keyof typeof toneClasses;
  title: string;
  description: string;
  className?: string;
}) {
  const config = toneClasses[tone];
  const Icon = config.icon;

  return (
    <div className={cn("rounded-[24px] border p-4", config.panelClassName, className)}>
      <div className="flex items-start gap-3">
        <div className={cn("mt-0.5 flex size-10 shrink-0 items-center justify-center rounded-2xl bg-white/70 dark:bg-slate-950/40", config.iconClassName)}>
          <Icon className="size-5" />
        </div>
        <div className="space-y-1">
          <p className="text-sm font-semibold text-slate-900 dark:text-slate-100">{title}</p>
          <p className="text-sm leading-6 text-slate-600 dark:text-slate-300">{description}</p>
        </div>
      </div>
    </div>
  );
}
