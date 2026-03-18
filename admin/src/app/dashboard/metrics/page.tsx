import Link from "next/link";
import { Activity, ArrowUpRight, BellRing, LineChart } from "lucide-react";
import { PageHero } from "@/components/page-hero";
import { StatCard } from "@/components/stat-card";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { MetricsSummary } from "@/lib/api";
import { formatCompactNumber } from "@/lib/ai-usage.mjs";
import { getServerMetrics } from "@/lib/server-api";

function formatPercent(value: number) {
  return `${Math.round((value || 0) * 100)}%`;
}

export const dynamic = "force-dynamic";

export default async function MetricsPage() {
  let metrics: MetricsSummary | null = null;
  let loadError = "";

  try {
    metrics = await getServerMetrics();
  } catch {
    loadError = "Metrics aren't available right now.";
  }

  const latestDAU = metrics?.daily_active_users.at(-1)?.users ?? 0;
  const latestRetention = metrics?.retention.at(-1);
  const aiUsage = metrics?.ai_usage;
  const dauPeak = Math.max(...(metrics?.daily_active_users.map((point) => point.users) ?? [1]));

  return (
    <div className="space-y-6">
      <PageHero
        eyebrow="Operations"
        title="Daily learning metrics"
        description="Track active learners, retention, nudge response, and model activity from the Go admin API."
        aside={
          <div className="grid gap-3 rounded-[24px] bg-slate-950 p-4 text-white dark:bg-slate-900/90">
            <div>
              <p className="text-xs uppercase tracking-[0.22em] text-slate-400">Latest DAU</p>
              <p className="mt-2 text-4xl font-semibold">{formatCompactNumber(latestDAU)}</p>
            </div>
            <div className="text-sm text-slate-300">
              {metrics ? `Window: ${metrics.window_days} days` : "Waiting for the first metrics snapshot."}
            </div>
          </div>
        }
      />

      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <StatCard icon={Activity} title="Latest DAU" value={formatCompactNumber(latestDAU)} note="Distinct active learners today" />
        <StatCard icon={ArrowUpRight} title="Day 7 retention" value={formatPercent(latestRetention?.day_7_rate ?? 0)} note="Most recent cohort snapshot" />
        <StatCard
          icon={BellRing}
          title="Nudge response"
          value={formatPercent(metrics?.nudge_rate.response_rate ?? 0)}
          note={`${metrics?.nudge_rate.responses_within_24h ?? 0} responses within 24 hours`}
        />
        <StatCard icon={LineChart} title="AI messages" value={formatCompactNumber(aiUsage?.total_messages ?? 0)} note="Teacher workspace model traffic" />
      </section>

      <div className="grid gap-6 xl:grid-cols-[1.1fr_0.9fr]">
        <Card className="rounded-[28px] border-white/70 bg-white/85 shadow-[0_18px_60px_rgba(15,23,42,0.05)] dark:border-white/10 dark:bg-slate-950/60 dark:shadow-[0_24px_80px_rgba(2,8,23,0.35)]">
          <CardHeader className="flex flex-row items-center justify-between gap-3">
            <div>
              <CardTitle className="text-xl tracking-tight text-slate-800 dark:text-slate-100">Daily active learners</CardTitle>
              <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">Last {metrics?.window_days ?? 14} days of activity.</p>
            </div>
            <Link href="/dashboard" className="text-sm font-medium text-sky-700 hover:text-sky-900 dark:text-sky-300 dark:hover:text-sky-200">
              Back to dashboard
            </Link>
          </CardHeader>
          <CardContent className="space-y-4">
            {metrics?.daily_active_users.length ? (
              metrics.daily_active_users.map((point) => {
                const width = `${Math.max(8, Math.round((point.users / dauPeak) * 100))}%`;
                return (
                  <div key={point.date} className="space-y-2">
                    <div className="flex items-center justify-between text-sm text-slate-600 dark:text-slate-300">
                      <span>{point.date}</span>
                      <span>{formatCompactNumber(point.users)}</span>
                    </div>
                    <div className="h-3 overflow-hidden rounded-full bg-slate-200 dark:bg-slate-800">
                      <div className="h-full rounded-full bg-sky-500" style={{ width }} />
                    </div>
                  </div>
                );
              })
            ) : (
              <p className="text-sm text-slate-500 dark:text-slate-400">
                {loadError || "Daily activity will appear once metrics have been recorded."}
              </p>
            )}
          </CardContent>
        </Card>

        <Card className="rounded-[28px] border-white/70 bg-white/85 shadow-[0_18px_60px_rgba(15,23,42,0.05)] dark:border-white/10 dark:bg-slate-950/60 dark:shadow-[0_24px_80px_rgba(2,8,23,0.35)]">
          <CardHeader>
            <CardTitle className="text-xl tracking-tight text-slate-800 dark:text-slate-100">Retention cohorts</CardTitle>
            <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">Day 1, Day 7, and Day 14 follow-through by signup cohort.</p>
          </CardHeader>
          <CardContent className="space-y-4">
            {metrics?.retention.length ? (
              metrics.retention.map((row) => (
                <div key={row.cohort_date} className="rounded-2xl border border-slate-200/80 p-4 dark:border-white/10">
                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <p className="text-sm font-medium text-slate-900 dark:text-slate-100">{row.cohort_date}</p>
                      <p className="text-xs uppercase tracking-[0.18em] text-slate-500 dark:text-slate-400">
                        Cohort size {row.cohort_size}
                      </p>
                    </div>
                    <div className="text-right text-sm text-slate-600 dark:text-slate-300">
                      <p>D1 {formatPercent(row.day_1_rate)}</p>
                      <p>D7 {formatPercent(row.day_7_rate)}</p>
                      <p>D14 {formatPercent(row.day_14_rate)}</p>
                    </div>
                  </div>
                </div>
              ))
            ) : (
              <p className="text-sm text-slate-500 dark:text-slate-400">
                {loadError || "Retention snapshots will appear after multiple cohorts are available."}
              </p>
            )}
          </CardContent>
        </Card>
      </div>

      <Card className="rounded-[28px] border-white/70 bg-slate-950 text-white shadow-[0_18px_60px_rgba(15,23,42,0.1)] dark:border-white/10 dark:bg-slate-900/85">
        <CardContent className="grid gap-4 p-6 md:grid-cols-3">
          <div>
            <p className="text-xs uppercase tracking-[0.22em] text-slate-400">Nudge follow-through</p>
            <p className="mt-2 text-2xl font-semibold">{formatPercent(metrics?.nudge_rate.response_rate ?? 0)}</p>
            <p className="mt-2 text-sm text-slate-300">
              {metrics ? `${metrics.nudge_rate.responses_within_24h} of ${metrics.nudge_rate.nudges_sent} nudges led to a learner response within 24 hours.` : loadError}
            </p>
          </div>
          <div>
            <p className="text-xs uppercase tracking-[0.22em] text-slate-400">Token activity</p>
            <p className="mt-2 text-2xl font-semibold">{formatCompactNumber((aiUsage?.total_input_tokens ?? 0) + (aiUsage?.total_output_tokens ?? 0))}</p>
            <p className="mt-2 text-sm text-slate-300">Prompt and completion tokens across the current metrics snapshot.</p>
          </div>
          <div>
            <p className="text-xs uppercase tracking-[0.22em] text-slate-400">A/B comparison</p>
            <p className="mt-2 text-2xl font-semibold">Pending</p>
            <p className="mt-2 text-sm text-slate-300">
              Experiment comparison stays disabled until user flag assignment is persisted in the backend.
            </p>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
