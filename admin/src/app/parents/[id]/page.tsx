import { PageHero } from "@/components/page-hero";
import { Metric } from "@/components/metric";
import { StatCard } from "@/components/stat-card";
import { HeartHandshake, MessageSquareText, Trophy, Zap } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { formatAdminDateTime } from "@/lib/dates.mjs";
import { getServerParentSummary } from "@/lib/server-api";
import { buildParentContextLine, formatParentTopicLabel, getParentMasteryTone } from "@/lib/parent-summary.mjs";

export const dynamic = "force-dynamic";

export default async function ParentPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  let summary = null;
  let loadError = "";

  try {
    summary = await getServerParentSummary(id);
  } catch {
    loadError = "Failed to load parent summary.";
  }

  const masteryRows = summary?.mastery ?? [];

  return (
    <div className="space-y-6">
      <PageHero
        eyebrow="Parent support summary"
        title={summary ? `${summary.child.name} this week` : "Loading child summary..."}
        description={buildParentContextLine(summary)}
        aside={
          <div className="grid gap-3 rounded-[24px] bg-slate-950 p-4 text-white dark:bg-slate-900/90 sm:grid-cols-3 lg:grid-cols-1">
            <Metric label="Current streak" value={summary ? `${summary.streak.current} days` : "-"} />
            <Metric label="Longest streak" value={summary ? `${summary.streak.longest} days` : "-"} />
            <Metric label="Total XP" value={summary ? String(summary.streak.total_xp) : "-"} />
          </div>
        }
        className="bg-white/85 dark:bg-slate-950/60"
      />

      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <StatCard icon={Zap} title="Active days" value={String(summary?.weekly_stats.days_active ?? 0)} note="Days with study activity in the last 7 days" />
        <StatCard
          icon={MessageSquareText}
          title="Messages"
          value={String(summary?.weekly_stats.messages_exchanged ?? 0)}
          note="Student and AI exchanges this week"
        />
        <StatCard icon={Trophy} title="Quizzes" value={String(summary?.weekly_stats.quizzes_completed ?? 0)} note="Quiz completions logged this week" />
        <StatCard icon={HeartHandshake} title="Needs review" value={String(summary?.weekly_stats.needs_review_count ?? 0)} note="Topics asking for parent encouragement" />
      </section>

      <section className="grid gap-4 xl:grid-cols-[1.05fr_0.95fr]">
        <Card className="rounded-[28px] border-white/70 bg-white/85 shadow-[0_16px_50px_rgba(15,23,42,0.05)] dark:border-white/10 dark:bg-slate-950/60 dark:shadow-[0_20px_60px_rgba(2,8,23,0.35)]">
          <CardHeader>
            <CardTitle className="text-xl tracking-tight">Mastery progress</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {loadError ? <p className="text-sm text-rose-600">{loadError}</p> : null}
            {masteryRows.length ? (
              masteryRows.map((item) => {
                const score = Math.round(item.mastery_score * 100);
                return (
                  <div key={item.topic_id} className="space-y-2">
                    <div className="flex items-center justify-between gap-3">
                      <p className="text-sm font-medium text-slate-900 dark:text-slate-100">{formatParentTopicLabel(item.topic_id)}</p>
                      <span className="text-xs uppercase tracking-[0.18em] text-slate-500 dark:text-slate-400">{score}%</span>
                    </div>
                    <div className="h-3 overflow-hidden rounded-full bg-slate-200 dark:bg-slate-800">
                      <div className={`h-full rounded-full ${getParentMasteryTone(item.mastery_score)}`} style={{ width: `${score}%` }} />
                    </div>
                    <p className="text-xs text-slate-500 dark:text-slate-400">
                      {item.next_review_at
                        ? `Next review ${formatAdminDateTime(item.next_review_at)}`
                        : "No review scheduled yet"}
                    </p>
                  </div>
                );
              })
            ) : (
              <p className="text-sm text-slate-500 dark:text-slate-400">No mastery data available yet.</p>
            )}
          </CardContent>
        </Card>

        <Card className="rounded-[28px] border-white/70 bg-white/85 shadow-[0_16px_50px_rgba(15,23,42,0.05)] dark:border-white/10 dark:bg-slate-950/60 dark:shadow-[0_20px_60px_rgba(2,8,23,0.35)]">
          <CardHeader>
            <CardTitle className="text-xl tracking-tight">Encouragement suggestion</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="rounded-[24px] bg-slate-950 p-5 text-white dark:bg-slate-900">
              <p className="text-xs font-semibold uppercase tracking-[0.22em] text-sky-300">Suggested message</p>
              <p className="mt-3 text-2xl font-semibold tracking-tight">
                {summary?.encouragement.headline ?? "Preparing a parent-friendly talking point..."}
              </p>
              <p className="mt-3 text-sm leading-7 text-slate-200">
                {summary?.encouragement.text ?? "One concise encouragement will appear here once the summary loads."}
              </p>
            </div>

            <div className="rounded-[24px] border border-slate-200/70 bg-slate-50 px-4 py-4 dark:border-white/10 dark:bg-slate-900/70">
              <p className="text-sm font-medium text-slate-900 dark:text-slate-100">What this means for home support</p>
              <p className="mt-2 text-sm leading-6 text-slate-600 dark:text-slate-300">
                Keep praise specific, focus on one topic only, and ask for a short follow-up practice session instead of a long catch-up.
              </p>
            </div>
          </CardContent>
        </Card>
      </section>
    </div>
  );
}
