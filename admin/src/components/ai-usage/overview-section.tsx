import { AdminSurface } from "@/components/admin-surface";
import { formatBudgetWindowLabel } from "@/components/ai-usage/format";
import type { AIUsageView } from "@/components/ai-usage/types";
import { formatCompactNumber, formatUSD } from "@/lib/ai-usage.mjs";

function SummaryMetric({
  label,
  value,
  note,
  tone = "text-slate-500 dark:text-slate-400",
}: {
  label: string;
  value: string;
  note: string;
  tone?: string;
}) {
  return (
    <div className="rounded-2xl border border-slate-200/70 bg-white/80 px-4 py-3 dark:border-white/10 dark:bg-slate-950/40">
      <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-slate-500 dark:text-slate-400">
        {label}
      </p>
      <p className="mt-2 text-3xl font-semibold tracking-tight text-slate-950 dark:text-slate-50">
        {value}
      </p>
      <p className={`mt-1 text-sm ${tone}`}>{note}</p>
    </div>
  );
}

export function AIUsageOverviewSection({
  view,
}: {
  view: AIUsageView;
}) {
  return (
    <AdminSurface
      className="border-transparent bg-transparent shadow-none"
      contentClassName="space-y-4 px-0 pb-0 pt-1"
    >
      <div className="space-y-1">
        <h1 className="text-[2rem] font-semibold tracking-tight text-slate-950 dark:text-slate-50">
          AI usage
        </h1>
        <p className="max-w-3xl text-sm text-slate-600 dark:text-slate-300">
          Token volume, budget status, and provider mix for the current school workspace.
        </p>
      </div>

      <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
        <SummaryMetric
          label="Tokens"
          value={formatCompactNumber(view.totalTokens)}
          note={view.topProvider ? `${view.topProvider.provider} top source` : "No provider activity yet"}
        />
        <SummaryMetric
          label="Messages"
          value={formatCompactNumber(view.total_messages)}
          note="Current aggregate window"
        />
        <SummaryMetric
          label="Budget"
          value={view.budgetStatus.label}
          note={formatBudgetWindowLabel(view.budget_period_start, view.budget_period_end)}
          tone={view.budgetStatus.tone}
        />
        <SummaryMetric
          label="Per learner"
          value={view.per_student_average_tokens !== null ? formatCompactNumber(view.per_student_average_tokens) : "--"}
          note={
            view.per_student_average_cost_usd !== null
              ? formatUSD(view.per_student_average_cost_usd)
              : "Token average"
          }
        />
      </div>
    </AdminSurface>
  );
}
