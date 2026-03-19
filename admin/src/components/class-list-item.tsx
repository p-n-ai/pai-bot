import { cn } from "@/lib/utils";

export function ClassListItem({
  name,
  syllabus,
  joinCode,
  summary,
  active,
  onClick,
}: {
  name: string;
  syllabus: string;
  joinCode: string;
  summary: string;
  active: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "w-full rounded-[24px] border p-4 text-left transition",
        active
          ? "border-sky-300 bg-sky-50 shadow-[0_12px_35px_rgba(14,165,233,0.12)] dark:border-sky-400/30 dark:bg-sky-400/10"
          : "border-slate-200/80 bg-slate-50/80 hover:border-slate-300 hover:bg-white dark:border-white/10 dark:bg-slate-900/60 dark:hover:bg-slate-900/80",
      )}
    >
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="text-sm font-semibold text-slate-900 dark:text-slate-100">{name}</p>
          <p className="mt-1 text-xs uppercase tracking-[0.18em] text-slate-500 dark:text-slate-400">{syllabus}</p>
        </div>
        <span className="rounded-full bg-slate-950 px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.16em] text-white dark:bg-slate-200 dark:text-slate-950">
          {joinCode}
        </span>
      </div>
      <p className="mt-3 text-sm leading-6 text-slate-600 dark:text-slate-300">{summary}</p>
    </button>
  );
}
