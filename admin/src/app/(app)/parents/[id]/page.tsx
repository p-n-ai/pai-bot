import { AdminHighlightPanel } from "@/components/admin-highlight-panel";
import { AdminInsetPanel } from "@/components/admin-inset-panel";
import { AdminSurface, AdminSurfaceHeader } from "@/components/admin-surface";
import { PageHero } from "@/components/page-hero";
import { Metric } from "@/components/metric";
import { StatePanel } from "@/components/state-panel";
import { StatCard } from "@/components/stat-card";
import { IconBolt, IconHeartHandshake, IconMessages, IconTrophy } from "@tabler/icons-react";
import { formatAdminDateTime } from "@/lib/dates.mjs";
import { getParentViewModel } from "@/lib/parent-view.mjs";
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
    loadError = "The parent summary isn't available right now.";
  }

  const view = getParentViewModel(summary);
  const masteryRows = view.masteryRows;

  return (
    <div className="space-y-6">
      <PageHero
        eyebrow="Parent support summary"
        title={summary ? `${summary.child.name} this week` : "Child summary"}
        description={buildParentContextLine(summary)}
        aside={
          <AdminHighlightPanel className="sm:grid-cols-3 lg:grid-cols-1">
            <Metric label="Current streak" value={summary ? `${summary.streak.current} days` : "-"} />
            <Metric label="Longest streak" value={summary ? `${summary.streak.longest} days` : "-"} />
            <Metric label="Total XP" value={summary ? String(summary.streak.total_xp) : "-"} />
          </AdminHighlightPanel>
        }
        className="bg-white/85 dark:bg-slate-950/60"
      />

      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <StatCard icon={IconBolt} title="Active days" value={String(summary?.weekly_stats.days_active ?? 0)} note="Days with study activity in the last 7 days" />
        <StatCard
          icon={IconMessages}
          title="Messages"
          value={String(summary?.weekly_stats.messages_exchanged ?? 0)}
          note="Student and AI exchanges this week"
        />
        <StatCard icon={IconTrophy} title="Quizzes" value={String(summary?.weekly_stats.quizzes_completed ?? 0)} note="Quiz completions logged this week" />
        <StatCard icon={IconHeartHandshake} title="Needs review" value={String(summary?.weekly_stats.needs_review_count ?? 0)} note="Topics asking for parent encouragement" />
      </section>

      <section className="grid gap-4 xl:grid-cols-[1.05fr_0.95fr]">
        <AdminSurface>
          <AdminSurfaceHeader title="Mastery progress" />
          <div className="mt-6 space-y-4">
            {loadError ? (
              <StatePanel
                tone="error"
                title="Parent summary unavailable"
                description={loadError}
              />
            ) : null}
            {view.hasMastery ? (
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
              <StatePanel
                tone="empty"
                title="No mastery data yet"
                description="No mastery data is available yet for this learner."
              />
            )}
          </div>
        </AdminSurface>

        <AdminSurface>
          <AdminSurfaceHeader title="Encouragement suggestion" />
          <div className="mt-6 space-y-4">
            <AdminHighlightPanel className="p-5">
              <p className="text-xs font-semibold uppercase tracking-[0.22em] text-sky-300">Suggested message</p>
              <p className="mt-3 text-2xl font-semibold tracking-tight">{view.encouragementHeadline}</p>
              <p className="mt-3 text-sm leading-7 text-slate-200">
                {view.encouragementText}
              </p>
            </AdminHighlightPanel>

            <AdminInsetPanel>
              <p className="text-sm font-medium text-slate-900 dark:text-slate-100">What this means for home support</p>
              <p className="mt-2 text-sm leading-6 text-slate-600 dark:text-slate-300">
                Keep praise specific, focus on one topic only, and ask for a short follow-up practice session instead of a long catch-up.
              </p>
            </AdminInsetPanel>
          </div>
        </AdminSurface>
      </section>
    </div>
  );
}
