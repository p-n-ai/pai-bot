import { AdminInsetPanel } from "@/components/admin-inset-panel";

function masteryTone(score: number) {
  if (score >= 0.75) return "bg-emerald-500";
  if (score >= 0.55) return "bg-sky-500";
  if (score >= 0.35) return "bg-amber-400";
  return "bg-rose-400";
}

export function TopicProgressRow({
  title,
  status,
  progress,
}: {
  title: string;
  status: string;
  progress: number;
}) {
  const percentage = Math.round(progress * 100);

  return (
    <AdminInsetPanel>
      <div className="flex items-center justify-between gap-3">
        <div>
          <p className="text-sm font-semibold text-slate-900 dark:text-slate-100">{title}</p>
          <p className="text-xs uppercase tracking-[0.18em] text-slate-500 dark:text-slate-400">{status}</p>
        </div>
        <span className="text-sm font-medium text-slate-600 dark:text-slate-300">{percentage}%</span>
      </div>
      <div className="mt-2 h-3 overflow-hidden rounded-full bg-slate-200 dark:bg-slate-800">
        <div className={`h-full rounded-full ${masteryTone(progress)}`} style={{ width: `${percentage}%` }} />
      </div>
    </AdminInsetPanel>
  );
}
