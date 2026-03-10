"use client";

import Link from "next/link";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { PolarAngleAxis, PolarGrid, Radar, RadarChart, ResponsiveContainer, Tooltip } from "recharts";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { getStudentConversations, getStudentDetail, type StudentConversation } from "@/lib/api";

const TOPIC_LABELS: Record<string, string> = {
  "linear-equations": "Linear Equations",
  "algebraic-expressions": "Expressions",
  inequalities: "Inequalities",
  functions: "Functions",
};

export default function StudentPage() {
  const params = useParams<{ id: string }>();
  const id = params.id;
  const [detail, setDetail] = useState<Awaited<ReturnType<typeof getStudentDetail>> | null>(null);
  const [conversations, setConversations] = useState<StudentConversation[]>([]);

  useEffect(() => {
    let active = true;
    Promise.all([getStudentDetail(id), getStudentConversations(id)]).then(([student, convo]) => {
      if (!active) return;
      setDetail(student);
      setConversations(convo);
    });
    return () => {
      active = false;
    };
  }, [id]);

  const radarData = detail?.progress.map((item) => ({
    topic: TOPIC_LABELS[item.topic_id] ?? item.topic_id,
    mastery: Math.round(item.mastery_score * 100),
  })) ?? [];

  const struggleAreas = detail?.progress.filter((item) => item.mastery_score < 0.6) ?? [];

  return (
    <main className="min-h-screen bg-[linear-gradient(180deg,#fcfffd_0%,#f3f8ff_100%)] px-6 py-8 lg:px-10">
      <div className="mx-auto max-w-7xl space-y-6">
        <header className="grid gap-4 rounded-[28px] border border-white/70 bg-white/85 p-6 shadow-[0_18px_60px_rgba(15,23,42,0.06)] lg:grid-cols-[1.15fr_0.85fr]">
          <div className="space-y-3">
            <Link href="/dashboard" className="text-sm font-medium text-sky-700 hover:text-sky-900">
              Back to dashboard
            </Link>
            <h1 className="text-3xl font-semibold tracking-tight text-slate-950">{detail?.student.name ?? "Loading student..."}</h1>
            <p className="text-sm text-slate-600">
              {detail ? `${detail.student.form} | ${detail.student.channel} | ${detail.student.external_id}` : "Fetching student record"}
            </p>
          </div>
          <div className="grid gap-3 rounded-[24px] bg-slate-950 p-4 text-white sm:grid-cols-3 lg:grid-cols-1">
            <Metric label="Current streak" value={detail ? `${detail.streak.current} days` : "-"} />
            <Metric label="Longest streak" value={detail ? `${detail.streak.longest} days` : "-"} />
            <Metric label="Total XP" value={detail ? String(detail.streak.total_xp) : "-"} />
          </div>
        </header>

        <section className="grid gap-4 xl:grid-cols-[1fr_0.9fr]">
          <Card className="rounded-[28px] border-white/70 bg-white/85 shadow-[0_16px_50px_rgba(15,23,42,0.05)]">
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

          <Card className="rounded-[28px] border-white/70 bg-white/85 shadow-[0_16px_50px_rgba(15,23,42,0.05)]">
            <CardHeader>
              <CardTitle className="text-xl tracking-tight">Struggle areas</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex flex-wrap gap-2">
                {struggleAreas.length ? (
                  struggleAreas.map((item) => (
                    <Badge key={item.topic_id} className="rounded-full bg-amber-100 px-3 py-1 text-amber-900 hover:bg-amber-100">
                      {TOPIC_LABELS[item.topic_id] ?? item.topic_id} {Math.round(item.mastery_score * 100)}%
                    </Badge>
                  ))
                ) : (
                  <p className="text-sm text-slate-500">No active struggle areas.</p>
                )}
              </div>
              <div className="grid grid-cols-7 gap-2">
                {(detail?.progress ?? []).flatMap((item, index) =>
                  Array.from({ length: 3 }).map((_, block) => {
                    const opacity = Math.max(0.18, item.mastery_score);
                    return (
                      <div
                        key={`${item.topic_id}-${index}-${block}`}
                        className="h-9 rounded-xl border border-white/60 bg-sky-500"
                        style={{ opacity }}
                      />
                    );
                  })
                )}
              </div>
            </CardContent>
          </Card>
        </section>

        <Card className="rounded-[28px] border-white/70 bg-white/85 shadow-[0_16px_50px_rgba(15,23,42,0.05)]">
          <CardHeader>
            <CardTitle className="text-xl tracking-tight">Recent conversations</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            {conversations.map((item) => (
              <div
                key={item.id}
                className={`rounded-2xl border p-4 ${item.role === "student" ? "bg-slate-50" : "bg-sky-50"}`}
              >
                <div className="mb-2 flex items-center justify-between text-xs font-medium uppercase tracking-[0.18em] text-slate-500">
                  <span>{item.role}</span>
                  <span>{new Date(item.timestamp).toLocaleString()}</span>
                </div>
                <p className="text-sm leading-6 text-slate-700">{item.text}</p>
              </div>
            ))}
          </CardContent>
        </Card>
      </div>
    </main>
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
