"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { BellRing, ChevronRight, Sparkles } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { getClassProgress, type ClassProgress } from "@/lib/api";

const TOPIC_LABELS: Record<string, string> = {
  "linear-equations": "Linear Equations",
  "algebraic-expressions": "Expressions",
  inequalities: "Inequalities",
  functions: "Functions",
};

function scoreTone(score: number) {
  if (score >= 0.8) return "bg-emerald-500 text-white";
  if (score >= 0.6) return "bg-lime-400 text-slate-950";
  if (score >= 0.4) return "bg-amber-300 text-slate-950";
  return "bg-rose-400 text-white";
}

export default function DashboardPage() {
  const [data, setData] = useState<ClassProgress | null>(null);
  const [loading, setLoading] = useState(true);
  const [nudgeMessage, setNudgeMessage] = useState("");

  useEffect(() => {
    let active = true;
    getClassProgress("form-1-algebra").then((result) => {
      if (!active) return;
      setData(result);
      setLoading(false);
    });
    return () => {
      active = false;
    };
  }, []);

  const averageMastery = data
    ? Math.round(
        (data.students.reduce((total, student) => {
          return (
            total + data.topic_ids.reduce((topicTotal, topicId) => topicTotal + (student.topics[topicId] ?? 0), 0)
          );
        }, 0) /
          (data.students.length * data.topic_ids.length)) *
          100
      )
    : 0;

  return (
    <main className="min-h-screen bg-[linear-gradient(180deg,#f8fcff_0%,#eef8f7_100%)] px-6 py-8 lg:px-10">
      <div className="mx-auto max-w-7xl space-y-6">
        <header className="grid gap-4 rounded-[28px] border border-white/70 bg-white/80 p-6 shadow-[0_18px_60px_rgba(15,23,42,0.08)] backdrop-blur lg:grid-cols-[1.3fr_0.7fr]">
          <div className="space-y-3">
            <p className="text-xs font-semibold uppercase tracking-[0.24em] text-sky-700">Teacher cockpit</p>
            <h1 className="text-3xl font-semibold tracking-tight text-slate-950">Class mastery at a glance</h1>
            <p className="max-w-2xl text-sm leading-6 text-slate-600">
              This screen stays useful before the backend admin API is complete by falling back to typed mock data.
            </p>
          </div>
          <div className="grid gap-3 rounded-[24px] bg-slate-950 p-4 text-white">
            <div>
              <p className="text-xs uppercase tracking-[0.22em] text-slate-400">Average mastery</p>
              <p className="mt-2 text-4xl font-semibold">{averageMastery}%</p>
            </div>
            <div className="flex items-center gap-2 text-sm text-slate-300">
              <Sparkles className="size-4 text-amber-300" />
              API-first, fallback-safe UI
            </div>
          </div>
        </header>

        <section className="grid gap-4 md:grid-cols-3">
          <StatCard title="Students" value={String(data?.students.length ?? 0)} note="Tracked in this view" />
          <StatCard title="Topics" value={String(data?.topic_ids.length ?? 0)} note="Algebra sequence" />
          <StatCard title="Nudges" value="Ready" note="UI action stubbed" />
        </section>

        <Card className="rounded-[28px] border-white/70 bg-white/85 shadow-[0_18px_60px_rgba(15,23,42,0.05)]">
          <CardHeader className="flex flex-row items-center justify-between gap-3">
            <div>
              <CardTitle className="text-xl tracking-tight">Mastery heatmap</CardTitle>
              <p className="mt-1 text-sm text-slate-500">Students by topic with direct navigation into detail views.</p>
            </div>
            <Link href="/" className="text-sm font-medium text-sky-700 hover:text-sky-900">
              Back home
            </Link>
          </CardHeader>
          <CardContent>
            {loading ? (
              <p className="text-sm text-slate-500">Loading dashboard...</p>
            ) : data ? (
              <div className="space-y-5">
                <div className="overflow-x-auto">
                  <table className="w-full min-w-[760px] border-separate border-spacing-y-2">
                    <thead>
                      <tr>
                        <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-[0.18em] text-slate-500">Student</th>
                        {data.topic_ids.map((topicId) => (
                          <th key={topicId} className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-[0.18em] text-slate-500">
                            {TOPIC_LABELS[topicId] ?? topicId}
                          </th>
                        ))}
                        <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-[0.18em] text-slate-500">Action</th>
                      </tr>
                    </thead>
                    <tbody>
                      {data.students.map((student) => (
                        <tr key={student.id}>
                          <td className="rounded-l-2xl bg-slate-50 px-3 py-3 text-sm font-medium text-slate-900">
                            <Link href={`/students/${student.id}`} className="inline-flex items-center gap-2 hover:text-sky-700">
                              {student.name}
                              <ChevronRight className="size-4" />
                            </Link>
                          </td>
                          {data.topic_ids.map((topicId) => {
                            const score = student.topics[topicId] ?? 0;
                            return (
                              <td key={`${student.id}-${topicId}`} className="bg-slate-50 px-3 py-3">
                                <span className={`inline-flex rounded-full px-3 py-1 text-sm font-semibold ${scoreTone(score)}`}>
                                  {Math.round(score * 100)}%
                                </span>
                              </td>
                            );
                          })}
                          <td className="rounded-r-2xl bg-slate-50 px-3 py-3">
                            <Button
                              size="sm"
                              className="gap-2"
                              onClick={() => setNudgeMessage(`Queued nudge for ${student.name}. Backend event hook comes next.`)}
                            >
                              <BellRing className="size-4" />
                              Nudge
                            </Button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
                {nudgeMessage ? <p className="text-sm text-slate-600">{nudgeMessage}</p> : null}
              </div>
            ) : (
              <p className="text-sm text-rose-600">Failed to load class data.</p>
            )}
          </CardContent>
        </Card>
      </div>
    </main>
  );
}

function StatCard({ title, value, note }: { title: string; value: string; note: string }) {
  return (
    <Card className="rounded-[24px] border-white/70 bg-white/85 shadow-[0_16px_40px_rgba(15,23,42,0.04)]">
      <CardHeader>
        <p className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500">{title}</p>
        <CardTitle className="text-3xl tracking-tight">{value}</CardTitle>
        <p className="text-sm text-slate-500">{note}</p>
      </CardHeader>
    </Card>
  );
}
