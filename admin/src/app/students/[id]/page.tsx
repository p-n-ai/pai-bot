"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useState } from "react";
import { Metric } from "@/components/metric";
import { PageHero } from "@/components/page-hero";
import { PolarAngleAxis, PolarGrid, Radar, RadarChart, ResponsiveContainer, Tooltip } from "recharts";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useAsyncResource } from "@/hooks/use-async-resource";
import { formatAdminDateTime } from "@/lib/dates.mjs";
import { getStudentConversations, getStudentDetail, type StudentConversation } from "@/lib/api";
import { formatTopicLabel } from "@/lib/topic-labels.mjs";

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

  const radarData = detail?.progress.map((item) => ({
    topic: formatTopicLabel(item.topic_id),
    mastery: Math.round(item.mastery_score * 100),
  })) ?? [];

  const struggleAreas = detail?.progress.filter((item) => item.mastery_score < 0.6) ?? [];

  return (
    <div className="space-y-6">
        <PageHero
          eyebrow="Student detail"
          title={detail?.student.name ?? "Loading student..."}
          description={detail ? `${detail.student.form} | ${detail.student.channel} | ${detail.student.external_id}` : "Fetching student record"}
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

        <section className="grid gap-4 xl:grid-cols-[1fr_0.9fr]">
          <Card className="rounded-[28px] border-white/70 bg-white/85 shadow-[0_16px_50px_rgba(15,23,42,0.05)] dark:border-white/10 dark:bg-slate-950/60 dark:shadow-[0_20px_60px_rgba(2,8,23,0.35)]">
            <CardHeader>
              <CardTitle className="text-xl tracking-tight">Mastery radar</CardTitle>
            </CardHeader>
            <CardContent className="h-[320px]">
              <ResponsiveContainer width="100%" height="100%">
                <RadarChart data={radarData}>
                  <PolarGrid />
                  <PolarAngleAxis dataKey="topic" />
                  <Tooltip />
                  <Radar dataKey="mastery" stroke="#0284c7" fill="#38bdf8" fillOpacity={0.35} />
                </RadarChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>

          <Card className="rounded-[28px] border-white/70 bg-white/85 shadow-[0_16px_50px_rgba(15,23,42,0.05)] dark:border-white/10 dark:bg-slate-950/60 dark:shadow-[0_20px_60px_rgba(2,8,23,0.35)]">
            <CardHeader>
              <CardTitle className="text-xl tracking-tight">Struggle areas</CardTitle>
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
                  <p className="text-sm text-slate-500 dark:text-slate-400">No active struggle areas.</p>
                )}
              </div>
              <div className="space-y-3">
                {(detail?.progress ?? []).map((item) => (
                  <div key={item.topic_id} className="rounded-2xl border border-slate-200/70 bg-slate-50 px-4 py-3 dark:border-white/10 dark:bg-slate-900/70">
                    <div className="flex items-center justify-between gap-3">
                      <p className="text-sm font-medium text-slate-900 dark:text-slate-100">{formatTopicLabel(item.topic_id)}</p>
                      <span className="text-xs uppercase tracking-[0.16em] text-slate-500 dark:text-slate-400">
                        {Math.round(item.mastery_score * 100)}% mastery
                      </span>
                    </div>
                    <p className="mt-2 text-xs text-slate-500 dark:text-slate-400">
                      Last studied: {item.last_studied_at ? formatAdminDateTime(item.last_studied_at) : "Not available"}
                    </p>
                    <p className="text-xs text-slate-500 dark:text-slate-400">
                      Next review: {item.next_review_at ? formatAdminDateTime(item.next_review_at) : "Not scheduled"}
                    </p>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </section>

        <Card className="rounded-[28px] border-white/70 bg-white/85 shadow-[0_16px_50px_rgba(15,23,42,0.05)] dark:border-white/10 dark:bg-slate-950/60 dark:shadow-[0_20px_60px_rgba(2,8,23,0.35)]">
          <CardHeader>
            <CardTitle className="text-xl tracking-tight">Recent conversations</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {error ? <p className="text-sm text-rose-600">{error}</p> : null}
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
