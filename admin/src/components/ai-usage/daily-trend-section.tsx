import { AdminSurface, AdminSurfaceHeader } from "@/components/admin-surface";
import { StatePanel } from "@/components/state-panel";
import { formatCompactNumber } from "@/lib/ai-usage.mjs";
import type { AIUsageView } from "@/components/ai-usage/types";
import { formatAIUsageDateLabel } from "@/components/ai-usage/format";

function DailyTrendBars({
  view,
}: {
  view: AIUsageView;
}) {
  return (
    <div className="space-y-3">
      {view.daily_usage.map((point) => {
        const width =
          view.dailyTrendPeak > 0
            ? `${Math.max((point.tokens / view.dailyTrendPeak) * 100, 4)}%`
            : "4%";

        return (
          <div key={point.date} className="space-y-1.5">
            <div className="flex items-center justify-between gap-3 text-sm">
              <span className="font-medium text-slate-700 dark:text-slate-200">
                {formatAIUsageDateLabel(point.date)}
              </span>
              <span className="text-slate-500 dark:text-slate-400">
                {formatCompactNumber(point.tokens)} tokens
              </span>
            </div>
            <div className="h-2.5 rounded-full bg-slate-100 dark:bg-slate-800">
              <div
                className="h-full rounded-full bg-primary/85 transition-[width]"
                style={{ width }}
              />
            </div>
          </div>
        );
      })}
    </div>
  );
}

export function AIUsageDailyTrendSection({
  view,
}: {
  view: AIUsageView;
}) {
  return (
    <AdminSurface>
      <div className="space-y-5">
        <AdminSurfaceHeader
          title="Daily token trend"
          description="Recent day-by-day token volume from the admin AI usage API."
        />

        {view.hasDailyTrend ? (
          <DailyTrendBars view={view} />
        ) : (
          <StatePanel
            tone="empty"
            title="No daily token trend yet"
            description="Daily usage bars will appear once the tenant starts generating AI traffic."
          />
        )}
      </div>
    </AdminSurface>
  );
}
