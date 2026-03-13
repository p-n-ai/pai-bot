"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { PolarAngleAxis, PolarGrid, Radar, RadarChart, ResponsiveContainer, Tooltip } from "recharts";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { getStudentConversations, getStudentDetail, type StudentConversation } from "@/lib/api";

function formatTopicLabel(topicId: string) {
  return topicId
    .split("-")
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

export default function StudentPage() {
  const params = useParams<{ id: string }>();
  const id = params.id;
  const [detail, setDetail] = useState<Awaited<ReturnType<typeof getStudentDetail>> | null>(null);
  const [conversations, setConversations] = useState<StudentConversation[]>([]);
  const [loadError, setLoadError] = useState("");

  useEffect(() => {
    let active = true;
    Promise.all([getStudentDetail(id), getStudentConversations(id)])
      .then(([student, convo]) => {
        if (!active) return;
        setDetail(student);
        setConversations(convo);
        setLoadError("");
      })
      .catch(() => {
        if (!active) return;
        setLoadError("Failed to load student data.");
      });
    return () => {
      active = false;
    };
  }, [id]);

  const radarData = detail?.progress.map((item) => ({
    topic: formatTopicLabel(item.topic_id),
    mastery: Math.round(item.mastery_score * 100),
  })) ?? [];

  const struggleAreas = detail?.progress.filter((item) => item.mastery_score < 0.6) ?? [];

  return (
    <div className="space-y-6">
        <header className="grid gap-4 rounded-[28px] border border-white/70 bg-white/85 p-6 shadow-[0_18px_60px_rgba(15,23,42,0.06)] dark:border-white/10 dark:bg-slate-950/60 dark:shadow-[0_24px_80px_rgba(2,8,23,0.4)] lg:grid-cols-[1.15fr_0.85fr]">
          <div className="space-y-3">
            <Link href="/dashboard" className="text-sm font-medium text-sky-700 hover:text-sky-900 dark:text-sky-300 dark:hover:text-sky-200">
              Back to dashboard
            </Link>
            <h1 className="text-3xl font-semibold tracking-tight text-slate-950 dark:text-white">{detail?.student.name ?? "Loading student..."}</h1>
            <p className="text-sm text-slate-600 dark:text-slate-300">
              {detail ? `${detail.student.form} | ${detail.student.channel} | ${detail.student.external_id}` : "Fetching student record"}
            </p>
          </div>
          <div className="grid gap-3 rounded-[24px] bg-slate-950 p-4 text-white dark:bg-slate-900/90 sm:grid-cols-3 lg:grid-cols-1">
            <Metric label="Current streak" value={detail ? `${detail.streak.current} days` : "-"} />
            <Metric label="Longest streak" value={detail ? `${detail.streak.longest} days` : "-"} />
            <Metric label="Total XP" value={detail ? String(detail.streak.total_xp) : "-"} />
          </div>
        </header>

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
                      Last studied: {item.last_studied_at ? new Date(item.last_studied_at).toLocaleString() : "Not available"}
                    </p>
                    <p className="text-xs text-slate-500 dark:text-slate-400">
                      Next review: {item.next_review_at ? new Date(item.next_review_at).toLocaleString() : "Not scheduled"}
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
            {loadError ? <p className="text-sm text-rose-600">{loadError}</p> : null}
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
                  <span>{new Date(item.timestamp).toLocaleString()}</span>
                </div>
                <p className="text-sm leading-6 text-slate-700 dark:text-slate-200">{item.text}</p>
              </div>
            ))}
          </CardContent>
        </Card>
    </div>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="text-xs uppercase tracking-[0.22em] text-slate-400">{label}</p>
      <p className="mt-2 text-2xl font-semibold">{value}</p>
    </div>
  );
}
