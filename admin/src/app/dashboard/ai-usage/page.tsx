import Link from "next/link";
import { Coins, Cpu, MessagesSquare, Orbit } from "lucide-react";
import { AdminSurface, AdminSurfaceHeader } from "@/components/admin-surface";
import { PageHero } from "@/components/page-hero";
import { StatePanel } from "@/components/state-panel";
import { StatCard } from "@/components/stat-card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import type { AIUsageSummary } from "@/lib/api";
import { formatCompactNumber, formatUSD, getAIUsageBudgetViewModel } from "@/lib/ai-usage.mjs";
import { getServerAIUsage } from "@/lib/server-api";

type DailyUsagePoint = NonNullable<AIUsageSummary["daily_usage"]>[number];
type ProviderCostItem = NonNullable<AIUsageSummary["provider_costs"]>[number];

function providerTone(provider: string) {
  switch (provider) {
    case "openai":
      return "bg-sky-500";
    case "anthropic":
      return "bg-amber-400";
    case "google":
      return "bg-emerald-500";
    case "openrouter":
      return "bg-fuchsia-500";
    default:
      return "bg-slate-500";
  }
}

export const dynamic = "force-dynamic";

export default async function AIUsagePage() {
  let usage: AIUsageSummary | null = null;
  let loadError = "";

  try {
    usage = await getServerAIUsage();
  } catch {
    loadError = "AI usage isn't available right now.";
  }

  const view = getAIUsageBudgetViewModel(usage);

  return (
    <div className="space-y-6">
      <PageHero
        eyebrow="AI operations"
        title="Budget and provider usage"
        description="Track message volume, token load, and the current budget contract as the admin API grows into a full cost dashboard."
        aside={
          <div className="grid gap-3 rounded-[24px] bg-slate-950 p-4 text-white dark:bg-slate-900/90">
          <div>
            <p className="text-xs uppercase tracking-[0.22em] text-slate-400">Top provider</p>
            <p className="mt-2 text-3xl font-semibold">
              {view.topProvider ? `${view.topProvider.provider} / ${view.topProvider.model}` : "Usage snapshot pending"}
            </p>
          </div>
          <div className="text-sm text-slate-300">
            {view.topProvider ? `${formatCompactNumber(view.topProvider.total_tokens)} tokens handled in this snapshot.` : "Usage details will appear once model activity has been recorded."}
          </div>
        </div>
        }
      />

      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <StatCard icon={MessagesSquare} title="AI messages" value={formatCompactNumber(view.total_messages)} note="Messages with a recorded model" />
        <StatCard icon={Coins} title="Monthly cost" value={formatUSD(view.monthlyCost)} note="Real amount once backend budget fields are exposed" />
        <StatCard icon={Cpu} title="Budget limit" value={formatUSD(view.budgetLimit)} note={view.budgetStatus.label} />
        <StatCard icon={Orbit} title="Remaining budget" value={formatUSD(view.remainingBudget)} note="Derived when cost and limit are both available" />
      </section>

      <section className="grid gap-6 xl:grid-cols-[1.1fr_0.9fr]">
        <AdminSurface>
          <AdminSurfaceHeader
            title="Budget trend"
            description="Daily token and cost trend once the admin API exposes time-series budget fields."
          />
          <div className="mt-6 space-y-4">
            {view.hasDailyTrend ? (
              view.daily_usage.map((point: DailyUsagePoint) => {
                const width = `${Math.max(8, Math.round((point.tokens / Math.max(view.dailyTrendPeak, 1)) * 100))}%`;
                return (
                  <div key={point.date} className="space-y-2">
                    <div className="flex items-center justify-between gap-3 text-sm text-slate-600 dark:text-slate-300">
                      <span>{point.date}</span>
                      <span>{formatCompactNumber(point.tokens)} tokens</span>
                    </div>
                    <div className="h-3 overflow-hidden rounded-full bg-slate-200 dark:bg-slate-800">
                      <div className="h-full rounded-full bg-sky-500" style={{ width }} />
                    </div>
                    <p className="text-xs text-slate-500 dark:text-slate-400">
                      Cost {formatUSD(point.cost_usd)} {point.messages > 0 ? `• ${formatCompactNumber(point.messages)} messages` : ""}
                    </p>
                  </div>
                );
              })
            ) : (
              <StatePanel
                tone={loadError ? "error" : "empty"}
                title={loadError ? "Budget trend unavailable" : "Budget trend pending backend support"}
                description={
                  loadError ||
                  "The current backend returns aggregate usage only. Daily cost and token trend data still need to be added to the admin API."
                }
              />
            )}
          </div>
        </AdminSurface>

        <AdminSurface>
          <AdminSurfaceHeader
            title="Per-student averages"
            description="Average cost and token load per learner once the backend exposes student-level budget aggregates."
          />
          <div className="mt-6 grid gap-4 md:grid-cols-2 xl:grid-cols-1">
            <StatCard
              title="Avg tokens / student"
              value={
                view.per_student_average_tokens !== null
                  ? formatCompactNumber(view.per_student_average_tokens)
                  : "Pending"
              }
              note="Requires per-student budget aggregation from the admin API"
            />
            <StatCard
              title="Avg cost / student"
              value={formatUSD(view.per_student_average_cost_usd)}
              note="Requires cost attribution from the admin API"
            />
          </div>
        </AdminSurface>
      </section>

      <AdminSurface>
        <AdminSurfaceHeader
          title="Provider breakdown"
          description="Use this table to spot which providers and models are carrying most of the token load while cost attribution is still landing."
          action={
            <Link href="/dashboard" className="text-sm font-medium text-sky-700 hover:text-sky-900 dark:text-sky-300 dark:hover:text-sky-200">
              Back to dashboard
            </Link>
          }
        />
        <div className="mt-6 space-y-4">
          {loadError ? <p className="text-sm text-slate-500 dark:text-slate-400">{loadError}</p> : null}
          {view.hasProviderCosts ? (
            <div className="grid gap-3 md:grid-cols-2">
              {view.provider_costs.map((item: ProviderCostItem) => {
                const share =
                  view.providerCostTotal > 0 ? Math.round(((item.cost_usd ?? 0) / view.providerCostTotal) * 100) : 0;
                return (
                  <div key={`cost:${item.provider}`} className="rounded-2xl border border-slate-200/80 p-4 dark:border-white/10">
                    <div className="flex items-center justify-between gap-3">
                      <p className="text-sm font-medium capitalize text-slate-900 dark:text-slate-100">{item.provider}</p>
                      <span className="text-xs uppercase tracking-[0.18em] text-slate-500 dark:text-slate-400">{share}%</span>
                    </div>
                    <p className="mt-3 text-2xl font-semibold text-slate-950 dark:text-slate-50">{formatUSD(item.cost_usd)}</p>
                    <div className="mt-3 h-2 overflow-hidden rounded-full bg-slate-200 dark:bg-slate-800">
                      <div className={`h-full rounded-full ${providerTone(item.provider)}`} style={{ width: `${share}%` }} />
                    </div>
                  </div>
                );
              })}
            </div>
          ) : (
            <StatePanel
              tone={loadError ? "error" : "empty"}
              title={loadError ? "Provider cost share unavailable" : "Provider cost share pending backend support"}
              description={
                loadError ||
                "The current backend returns provider token totals, but not provider cost breakdown yet. Token load is still shown in the table below."
              }
            />
          )}
          {usage && usage.providers.length > 0 ? (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="text-slate-600 dark:text-slate-300">Provider</TableHead>
                <TableHead className="text-slate-600 dark:text-slate-300">Model</TableHead>
                <TableHead className="text-slate-600 dark:text-slate-300">Messages</TableHead>
                <TableHead className="text-slate-600 dark:text-slate-300">Input</TableHead>
                <TableHead className="text-slate-600 dark:text-slate-300">Output</TableHead>
                <TableHead className="text-slate-600 dark:text-slate-300">Total</TableHead>
                <TableHead className="text-slate-600 dark:text-slate-300">Load</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(view.providers ?? []).map((item: AIUsageSummary["providers"][number]) => {
                const share = view.totalTokens > 0 ? Math.round((item.total_tokens / view.totalTokens) * 100) : 0;
                return (
                  <TableRow key={`${item.provider}:${item.model}`}>
                    <TableCell className="font-medium capitalize text-slate-600 dark:text-slate-300">{item.provider}</TableCell>
                    <TableCell className="text-slate-600 dark:text-slate-300">{item.model}</TableCell>
                    <TableCell className="text-slate-600 dark:text-slate-300">{formatCompactNumber(item.messages)}</TableCell>
                    <TableCell className="text-slate-600 dark:text-slate-300">{formatCompactNumber(item.input_tokens)}</TableCell>
                    <TableCell className="text-slate-600 dark:text-slate-300">{formatCompactNumber(item.output_tokens)}</TableCell>
                    <TableCell className="text-slate-600 dark:text-slate-300">{formatCompactNumber(item.total_tokens)}</TableCell>
                    <TableCell>
                      <div className="min-w-[150px] space-y-2">
                        <div className="h-2 overflow-hidden rounded-full bg-slate-200 dark:bg-slate-800">
                          <div className={`h-full rounded-full ${providerTone(item.provider)}`} style={{ width: `${share}%` }} />
                        </div>
                        <span className="text-xs uppercase tracking-[0.18em] text-slate-500 dark:text-slate-400">{share}%</span>
                      </div>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
          ) : (
            <p className="text-sm text-slate-500 dark:text-slate-400">
              {loadError ? "Please check back after the next sync." : "No AI usage has been recorded yet."}
            </p>
          )}
        </div>
      </AdminSurface>
    </div>
  );
}
