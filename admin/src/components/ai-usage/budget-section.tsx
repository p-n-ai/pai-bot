import { AdminInsetPanel } from "@/components/admin-inset-panel";
import { AdminSurface, AdminSurfaceHeader } from "@/components/admin-surface";
import { TokenBudgetEditor } from "@/components/token-budget-editor";
import { formatCompactNumber } from "@/lib/ai-usage.mjs";
import type { AIUsageView } from "@/components/ai-usage/types";
import { formatBudgetWindowLabel } from "@/components/ai-usage/format";

export function AIUsageBudgetSection({
  view,
}: {
  view: AIUsageView;
}) {
  return (
    <AdminSurface>
      <div className="space-y-5">
        <AdminSurfaceHeader
          title="Token budget window"
          description="Budget thresholds and remaining allowance for the active tenant-wide token window."
        />

        <div className="grid gap-4 md:grid-cols-3">
          <AdminInsetPanel title="Window">
            <p className="text-lg font-semibold text-slate-950 dark:text-slate-50">
              {formatBudgetWindowLabel(view.budget_period_start, view.budget_period_end)}
            </p>
          </AdminInsetPanel>
          <AdminInsetPanel title="Limit">
            <p className="text-lg font-semibold text-slate-950 dark:text-slate-50">
              {view.budgetTokenLimit !== null ? formatCompactNumber(view.budgetTokenLimit) : "--"}
            </p>
          </AdminInsetPanel>
          <AdminInsetPanel title="Remaining">
            <p className="text-lg font-semibold text-slate-950 dark:text-slate-50">
              {view.budgetTokenRemaining !== null ? formatCompactNumber(view.budgetTokenRemaining) : "--"}
            </p>
          </AdminInsetPanel>
        </div>

        <TokenBudgetEditor
          initialBudgetTokens={view.budgetTokenLimit}
          initialPeriodStart={view.budget_period_start}
          initialPeriodEnd={view.budget_period_end}
        />
      </div>
    </AdminSurface>
  );
}
