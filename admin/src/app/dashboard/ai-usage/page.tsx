import Link from "next/link";
import { cookies } from "next/headers";
import { Coins, Cpu, MessagesSquare, Orbit } from "lucide-react";
import { AdminSurface, AdminSurfaceHeader } from "@/components/admin-surface";
import { PageHero } from "@/components/page-hero";
import { StatePanel } from "@/components/state-panel";
import { StatCard } from "@/components/stat-card";
import { TokenBudgetEditor } from "@/components/token-budget-editor";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import type { AIUsageSummary, AuthUser } from "@/lib/api";
import { USER_COOKIE, parseCookieJSON } from "@/lib/auth-session";
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
  const cookieStore = await cookies();
  const currentUser = parseCookieJSON<AuthUser>(cookieStore.get(USER_COOKIE)?.value);
  const canManageBudget = currentUser?.role === "admin";
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
        description="Track model traffic, token load, and the current budget contract. Token analytics are live now; cost limits can slot in as backend budget fields land."
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
        <StatCard icon={Cpu} title="Total tokens" value={formatCompactNumber(view.totalTokens)} note="Aggregate input and output tokens across tracked models" />
        <StatCard
          icon={Orbit}
          title="Avg tokens / student"
          value={view.per_student_average_tokens !== null ? formatCompactNumber(view.per_student_average_tokens) : "Pending"}
          note={view.hasPerStudentAverages ? "Derived from students with recorded model traffic" : "Waiting for student-level activity to accumulate"}
        />
        <StatCard
          icon={Coins}
          title="Budget status"
          value={view.budgetStatus.label}
          note={view.budgetTokenLimit !== null ? "Active tenant token budget window" : "Create a tenant token budget window to activate tracking"}
        />
      </section>

      <section className="grid gap-6 xl:grid-cols-[1.1fr_0.9fr]">
        <AdminSurface>
          <AdminSurfaceHeader
            title="Budget trend"
            description="Daily token activity is live now. Cost overlays can join this trend once the admin API exposes budget fields."
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
                      {point.messages > 0 ? `${formatCompactNumber(point.messages)} messages` : "Messages pending"}
                      {point.cost_usd !== null && point.cost_usd !== undefined ? ` | Cost ${formatUSD(point.cost_usd)}` : ""}
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
            title="Budget contract"
            description={
              view.budgetTokenLimit !== null
                ? "The active tenant budget window is tracked in tokens and can be updated directly from this screen."
                : "Token analytics are live. Configure the first tenant budget window here when you are ready to enforce a limit."
            }
          />
          <div className="mt-6 space-y-4">
            {canManageBudget ? (
              <TokenBudgetEditor
                initialBudgetTokens={view.budgetTokenLimit}
                initialPeriodStart={view.budget_period_start}
                initialPeriodEnd={view.budget_period_end}
              />
            ) : null}
            <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-1">
              <StatCard
                title="Token budget"
                value={view.budgetTokenLimit !== null ? formatCompactNumber(view.budgetTokenLimit) : "Pending"}
                note={
                  view.budget_period_start && view.budget_period_end
                    ? `${view.budget_period_start} to ${view.budget_period_end}`
                    : "Configure a tenant token budget window to activate this card"
                }
              />
              <StatCard
                title="Used in window"
                value={view.budgetTokenUsed !== null ? formatCompactNumber(view.budgetTokenUsed) : "Pending"}
                note={view.budgetStatus.label}
              />
              <StatCard
                title="Remaining tokens"
                value={view.budgetTokenRemaining !== null ? formatCompactNumber(view.budgetTokenRemaining) : "Pending"}
                note={
                  view.tokenUsageRatio !== null
                    ? `${Math.round(view.tokenUsageRatio * 100)}% of the current token budget is already used`
                    : "Derived when an active tenant budget window exists"
                }
              />
            </div>
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
