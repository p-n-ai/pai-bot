"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { Metric } from "@/components/metric";
import { PageHero } from "@/components/page-hero";
import { PolarAngleAxis, PolarGrid, Radar, RadarChart, ResponsiveContainer, Tooltip } from "recharts";
import { StatePanel } from "@/components/state-panel";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useAsyncResource } from "@/hooks/use-async-resource";
import { formatAdminDateTime } from "@/lib/dates.mjs";
import { getStudentConversations, getStudentDetail, type StudentConversation } from "@/lib/api";
import { buildStudentViewModel } from "@/lib/student-view.mjs";
import { formatTopicLabel } from "@/lib/topic-labels.mjs";

function activityTone(level: number) {
  if (level >= 4) return "bg-sky-600 dark:bg-sky-300";
  if (level === 3) return "bg-sky-500 dark:bg-sky-400";
  if (level === 2) return "bg-sky-300 dark:bg-sky-500";
  if (level === 1) return "bg-sky-200 dark:bg-sky-700";
  return "bg-slate-200 dark:bg-slate-800";
}

export default function StudentPage() {
  const params = useParams<{ id: string }>();
  const id = params.id;
  const { data, error } = useAsyncResource<{ detail: Awaited<ReturnType<typeof getStudentDetail>>; conversations: StudentConversation[] }>(
    async () => {
      const [detail, conversations] = await Promise.all([getStudentDetail(id), getStudentConversations(id)]);
      return { detail, conversations };
    },
    [id],
  );

  const detail = data?.detail ?? null;
  const conversations = data?.conversations ?? [];

  const view = buildStudentViewModel(detail, conversations);
  const radarData = view.radarData.map((item) => ({
    topic: formatTopicLabel(item.topic),
    mastery: item.mastery,
  }));
  const struggleAreas = view.struggleAreas;
  const activityGrid = view.activityGrid;

  return (
    <div className="space-y-6">
        <PageHero
          eyebrow="Student detail"
          title={detail?.student.name ?? "Student summary"}
          description={
            detail
              ? `${detail.student.form} | ${detail.student.channel} | ${detail.student.external_id}`
              : error
                ? "Student information isn't available right now."
                : "Student details will appear here once the latest record is ready."
          }
          aside={
            <div className="grid gap-3 rounded-[24px] bg-slate-950 p-4 text-white dark:bg-slate-900/90 sm:grid-cols-3 lg:grid-cols-1">
              <Metric label="Current streak" value={detail ? `${detail.streak.current} days` : "-"} />
              <Metric label="Longest streak" value={detail ? `${detail.streak.longest} days` : "-"} />
              <Metric label="Total XP" value={detail ? String(detail.streak.total_xp) : "-"} />
            </div>
          }
          className="bg-white/85 dark:bg-slate-950/60 lg:grid-cols-[1.15fr_0.85fr]"
        >
            <Link href="/dashboard" className="text-sm font-medium text-sky-700 hover:text-sky-900 dark:text-sky-300 dark:hover:text-sky-200">
              Back to dashboard
            </Link>
        </PageHero>

        <section className="grid gap-4 xl:grid-cols-[0.75fr_1fr_0.9fr]">
          <Card className="rounded-[28px] border-white/70 bg-white/85 shadow-[0_16px_50px_rgba(15,23,42,0.05)] dark:border-white/10 dark:bg-slate-950/60 dark:shadow-[0_20px_60px_rgba(2,8,23,0.35)]">
            <CardHeader>
              <CardTitle className="text-xl tracking-tight text-slate-800 dark:text-slate-100">Profile card</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="rounded-2xl border border-slate-200/80 bg-slate-50 p-4 dark:border-white/10 dark:bg-slate-900/70">
                <p className="text-xs font-semibold uppercase tracking-[0.18em] text-slate-500 dark:text-slate-400">Form</p>
                <p className="mt-2 text-sm font-medium text-slate-900 dark:text-slate-100">{detail?.student.form ?? "-"}</p>
              </div>
              <div className="rounded-2xl border border-slate-200/80 bg-slate-50 p-4 dark:border-white/10 dark:bg-slate-900/70">
                <p className="text-xs font-semibold uppercase tracking-[0.18em] text-slate-500 dark:text-slate-400">Channel</p>
                <p className="mt-2 text-sm font-medium capitalize text-slate-900 dark:text-slate-100">{detail?.student.channel ?? "-"}</p>
              </div>
              <div className="rounded-2xl border border-slate-200/80 bg-slate-50 p-4 dark:border-white/10 dark:bg-slate-900/70">
                <p className="text-xs font-semibold uppercase tracking-[0.18em] text-slate-500 dark:text-slate-400">External ID</p>
                <p className="mt-2 break-all text-sm font-medium text-slate-900 dark:text-slate-100">{detail?.student.external_id ?? "-"}</p>
              </div>
              <div className="rounded-2xl border border-slate-200/80 bg-slate-50 p-4 dark:border-white/10 dark:bg-slate-900/70">
                <p className="text-xs font-semibold uppercase tracking-[0.18em] text-slate-500 dark:text-slate-400">Joined</p>
                <p className="mt-2 text-sm font-medium text-slate-900 dark:text-slate-100">
                  {detail?.student.created_at ? formatAdminDateTime(detail.student.created_at) : "-"}
                </p>
              </div>
            </CardContent>
          </Card>

          <Card className="rounded-[28px] border-white/70 bg-white/85 shadow-[0_16px_50px_rgba(15,23,42,0.05)] dark:border-white/10 dark:bg-slate-950/60 dark:shadow-[0_20px_60px_rgba(2,8,23,0.35)]">
            <CardHeader>
              <CardTitle className="text-xl tracking-tight text-slate-800 dark:text-slate-100">Mastery radar</CardTitle>
            </CardHeader>
            <CardContent className="h-[320px]">
              {radarData.length ? (
                <ResponsiveContainer width="100%" height="100%">
                  <RadarChart data={radarData}>
                    <PolarGrid />
                    <PolarAngleAxis dataKey="topic" />
                    <Tooltip />
                    <Radar dataKey="mastery" stroke="#0284c7" fill="#38bdf8" fillOpacity={0.35} />
                  </RadarChart>
                </ResponsiveContainer>
              ) : (
                <div className="flex h-full items-center justify-center">
                  <StatePanel
                    tone="empty"
                    title="No mastery radar yet"
                    description="Progress details will appear after the student completes some work."
                    className="w-full"
                  />
                </div>
              )}
            </CardContent>
          </Card>

          <Card className="rounded-[28px] border-white/70 bg-white/85 shadow-[0_16px_50px_rgba(15,23,42,0.05)] dark:border-white/10 dark:bg-slate-950/60 dark:shadow-[0_20px_60px_rgba(2,8,23,0.35)]">
            <CardHeader>
              <CardTitle className="text-xl tracking-tight text-slate-800 dark:text-slate-100">Struggle areas</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex flex-wrap gap-2">
                {struggleAreas.length ? (
                  struggleAreas.map((item) => (
                    <Badge
                      key={item.topic_id}
                      className="rounded-full bg-amber-100 px-3 py-1 text-amber-900 hover:bg-amber-100 dark:bg-amber-300/15 dark:text-amber-100 dark:hover:bg-amber-300/15"
                    >
                      {formatTopicLabel(item.topic_id)} {Math.round(item.mastery_score * 100)}%
                    </Badge>
                  ))
                ) : (
                  <StatePanel
                    tone="empty"
                    title="No active struggle areas"
                    description="This learner does not currently have any topics below the intervention threshold."
                  />
                )}
              </div>
              <div className="space-y-3">
                {view.hasProgress ? (
                  (detail?.progress ?? []).map((item) => (
                    <div key={item.topic_id} className="rounded-2xl border border-slate-200/70 bg-slate-50 px-4 py-3 dark:border-white/10 dark:bg-slate-900/70">
                      <div className="flex items-center justify-between gap-3">
                        <p className="text-sm font-medium text-slate-900 dark:text-slate-100">{formatTopicLabel(item.topic_id)}</p>
                        <span className="text-xs uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">
                          {Math.round(item.mastery_score * 100)}% mastery
                        </span>
                      </div>
                      <p className="mt-2 text-xs text-slate-500 dark:text-slate-400">
                        Last studied: {item.last_studied_at ? formatAdminDateTime(item.last_studied_at) : "Not recorded yet"}
                      </p>
                      <p className="text-xs text-slate-500 dark:text-slate-400">
                        Next review: {item.next_review_at ? formatAdminDateTime(item.next_review_at) : "To be scheduled"}
                      </p>
                    </div>
                  ))
                ) : (
                  <StatePanel
                    tone="empty"
                    title="No topic progress yet"
                    description="No topic progress has been recorded for this student yet."
                  />
                )}
              </div>
            </CardContent>
          </Card>
        </section>

        <Card className="rounded-[28px] border-white/70 bg-white/85 shadow-[0_16px_50px_rgba(15,23,42,0.05)] dark:border-white/10 dark:bg-slate-950/60 dark:shadow-[0_20px_60px_rgba(2,8,23,0.35)]">
          <CardHeader>
            <CardTitle className="text-xl tracking-tight text-slate-800 dark:text-slate-100">Activity grid</CardTitle>
            <p className="text-sm text-slate-500 dark:text-slate-400">Conversation activity over the last 14 days.</p>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="grid grid-cols-7 gap-2 md:grid-cols-14">
              {activityGrid.map((item) => (
                <div key={item.date} className="space-y-2 text-center">
                  <div
                    className={`h-12 rounded-2xl border border-white/60 shadow-inner dark:border-white/10 ${activityTone(item.level)}`}
                    title={`${item.shortLabel}: ${item.count} messages`}
                  />
                  <p className="text-[11px] uppercase tracking-[0.14em] text-slate-500 dark:text-slate-400">{item.shortLabel}</p>
                </div>
              ))}
            </div>
            <div className="flex flex-wrap items-center gap-3 text-xs text-slate-500 dark:text-slate-400">
              <span>Less active</span>
              {[0, 1, 2, 3, 4].map((level) => (
                <span
                  key={level}
                  className={`inline-flex size-4 rounded-full border border-white/60 dark:border-white/10 ${activityTone(level)}`}
                />
              ))}
              <span>More active</span>
            </div>
          </CardContent>
        </Card>

        <Card className="rounded-[28px] border-white/70 bg-white/85 shadow-[0_16px_50px_rgba(15,23,42,0.05)] dark:border-white/10 dark:bg-slate-950/60 dark:shadow-[0_20px_60px_rgba(2,8,23,0.35)]">
          <CardHeader>
            <CardTitle className="text-xl tracking-tight text-slate-800 dark:text-slate-100">Recent conversations</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {error ? (
              <StatePanel
                tone="error"
                title="Conversation history unavailable"
                description="Conversation history isn't available right now."
              />
            ) : null}
            {!error && !view.hasConversations ? (
              <StatePanel
                tone="empty"
                title="No tutoring messages yet"
                description="Recent tutoring messages will appear here once the student has chatted."
              />
            ) : null}
            {conversations.map((item) => (
              <div
                key={item.id}
                className={`rounded-2xl border p-4 ${
                  item.role === "student"
                    ? "bg-slate-50 dark:border-white/10 dark:bg-slate-900/80"
                    : "bg-sky-50 dark:border-sky-400/20 dark:bg-sky-400/10"
                }`}
              >
                <div className="mb-2 flex items-center justify-between text-xs font-medium uppercase tracking-[0.18em] text-slate-500 dark:text-slate-400">
                  <span>{item.role}</span>
                  <span>{formatAdminDateTime(item.timestamp)}</span>
                </div>
                <p className="text-sm leading-6 text-slate-700 dark:text-slate-200">{item.text}</p>
              </div>
            ))}
          </CardContent>
        </Card>
    </div>
  );
}
