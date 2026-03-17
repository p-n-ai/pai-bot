"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { BellRing, ChevronRight, Sparkles } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { getAverageMastery, getTrackedScores } from "@/lib/class-progress.mjs";
import { getClassProgress, sendStudentNudge, type ClassProgress } from "@/lib/api";

function scoreTone(score: number) {
  if (score >= 0.8) return "bg-emerald-500 text-white";
  if (score >= 0.6) return "bg-lime-400 text-slate-950";
  if (score >= 0.4) return "bg-amber-300 text-slate-950";
  return "bg-rose-400 text-white";
}

function formatTopicLabel(topicId: string) {
  return topicId
    .split("-")
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

export default function DashboardPage() {
  const [data, setData] = useState<ClassProgress | null>(null);
  const [loading, setLoading] = useState(true);
  const [nudgeMessage, setNudgeMessage] = useState("");
  const [sendingStudentID, setSendingStudentID] = useState("");

  useEffect(() => {
    let active = true;
    getClassProgress("all-students")
      .then((result) => {
        if (!active) return;
        setData(result);
        setLoading(false);
      })
      .catch(() => {
        if (!active) return;
        setData(null);
        setLoading(false);
      });
    return () => {
      active = false;
    };
  }, []);

  const averageMastery = data ? getAverageMastery(data) : 0;
  const trackedScores = data ? getTrackedScores(data) : 0;

  async function handleNudge(studentID: string, studentName: string) {
    setSendingStudentID(studentID);
    setNudgeMessage("");
    try {
      await sendStudentNudge(studentID);
      setNudgeMessage(`Nudge sent to ${studentName} on Telegram.`);
    } catch (error) {
      const message = error instanceof Error ? error.message : "Failed to send nudge.";
      setNudgeMessage(message);
    } finally {
      setSendingStudentID("");
    }
  }

  return (
    <div className="space-y-6">
        <header className="grid gap-4 rounded-[28px] border border-white/70 bg-white/80 p-6 shadow-[0_18px_60px_rgba(15,23,42,0.08)] backdrop-blur dark:border-white/10 dark:bg-slate-950/60 dark:shadow-[0_24px_80px_rgba(2,8,23,0.4)] lg:grid-cols-[1.3fr_0.7fr]">
          <div className="space-y-3">
            <p className="text-xs font-semibold uppercase tracking-[0.24em] text-sky-700 dark:text-sky-300">Teacher cockpit</p>
            <h1 className="text-3xl font-semibold tracking-tight text-slate-950 dark:text-white">Class mastery at a glance</h1>
            <p className="max-w-2xl text-sm leading-6 text-slate-600 dark:text-slate-300">
              Review topic-by-topic mastery and open each learner profile for a closer look.
            </p>
          </div>
          <div className="grid gap-3 rounded-[24px] bg-slate-950 p-4 text-white dark:bg-slate-900/90">
            <div>
              <p className="text-xs uppercase tracking-[0.22em] text-slate-400">Average mastery</p>
              <p className="mt-2 text-4xl font-semibold">{averageMastery}%</p>
            </div>
            <div className="flex items-center gap-2 text-sm text-slate-300">
              <Sparkles className="size-4 text-amber-300" />
              Live data from the Go admin API
            </div>
          </div>
        </header>

        <section className="grid gap-4 md:grid-cols-3">
          <StatCard title="Students" value={String(data?.students.length ?? 0)} note="Tracked in this view" />
          <StatCard title="Topics" value={String(data?.topic_ids.length ?? 0)} note="Algebra sequence" />
          <StatCard title="Tracked Scores" value={String(trackedScores)} note="Real mastery entries loaded" />
        </section>

        <Card className="rounded-[28px] border-white/70 bg-slate-950 text-white shadow-[0_18px_60px_rgba(15,23,42,0.1)] dark:border-white/10 dark:bg-slate-900/85">
          <CardContent className="flex flex-col gap-4 p-6 md:flex-row md:items-center md:justify-between">
            <div className="space-y-2">
              <p className="text-xs font-semibold uppercase tracking-[0.22em] text-sky-300">Operations</p>
              <h2 className="text-2xl font-semibold tracking-tight">Check AI usage before costs drift.</h2>
              <p className="max-w-2xl text-sm leading-6 text-slate-300">
                Open the usage view to inspect token volume by provider and model across the current admin API snapshot.
              </p>
            </div>
            <Link
              href="/dashboard/ai-usage"
              className="inline-flex items-center justify-center rounded-full bg-white px-5 py-3 text-sm font-medium text-slate-950 transition hover:bg-sky-100"
            >
              Open AI usage
            </Link>
          </CardContent>
        </Card>

        <Card className="rounded-[28px] border-white/70 bg-white/85 shadow-[0_18px_60px_rgba(15,23,42,0.05)] dark:border-white/10 dark:bg-slate-950/60 dark:shadow-[0_24px_80px_rgba(2,8,23,0.35)]">
          <CardHeader className="flex flex-row items-center justify-between gap-3">
            <div>
              <CardTitle className="text-xl tracking-tight">Mastery heatmap</CardTitle>
              <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">Students by topic with direct navigation into detail views.</p>
            </div>
            <Link href="/" className="text-sm font-medium text-sky-700 hover:text-sky-900 dark:text-sky-300 dark:hover:text-sky-200">
              Back home
            </Link>
          </CardHeader>
          <CardContent>
            {loading ? (
              <p className="text-sm text-slate-500 dark:text-slate-400">Loading dashboard...</p>
            ) : data ? (
              <div className="space-y-5">
                <div className="overflow-x-auto">
                  <table className="w-full min-w-[760px] border-separate border-spacing-y-2">
                    <thead>
                      <tr>
                        <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-[0.18em] text-slate-500 dark:text-slate-400">Student</th>
                        {data.topic_ids.map((topicId) => (
                          <th
                            key={topicId}
                            className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-[0.18em] text-slate-500 dark:text-slate-400"
                          >
                            {formatTopicLabel(topicId)}
                          </th>
                        ))}
                        <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-[0.18em] text-slate-500 dark:text-slate-400">Nudge</th>
                      </tr>
                    </thead>
                    <tbody>
                      {data.students.map((student) => (
                        <tr key={student.id}>
                          <td className="rounded-l-2xl bg-slate-50 px-3 py-3 text-sm font-medium text-slate-900 dark:bg-slate-900/80 dark:text-slate-100">
                            <Link
                              href={`/students/${student.id}`}
                              className="inline-flex items-center gap-2 hover:text-sky-700 dark:hover:text-sky-300"
                            >
                              {student.name}
                              <ChevronRight className="size-4" />
                            </Link>
                          </td>
                          {data.topic_ids.map((topicId) => {
                            const score = student.topics[topicId] ?? 0;
                            return (
                              <td key={`${student.id}-${topicId}`} className="bg-slate-50 px-3 py-3 dark:bg-slate-900/80">
                                <span className={`inline-flex rounded-full px-3 py-1 text-sm font-semibold ${scoreTone(score)}`}>
                                  {Math.round(score * 100)}%
                                </span>
                              </td>
                            );
                          })}
                          <td className="rounded-r-2xl bg-slate-50 px-3 py-3 dark:bg-slate-900/80">
                            <Button
                              size="sm"
                              className="gap-2"
                              disabled={sendingStudentID === student.id}
                              onClick={() => handleNudge(student.id, student.name)}
                            >
                              <BellRing className="size-4" />
                              {sendingStudentID === student.id ? "Sending..." : "Nudge"}
                            </Button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
                {nudgeMessage ? <p className="text-sm text-slate-600 dark:text-slate-300">{nudgeMessage}</p> : null}
              </div>
            ) : (
              <p className="text-sm text-rose-600">Failed to load class data.</p>
            )}
          </CardContent>
        </Card>
    </div>
  );
}

function StatCard({ title, value, note }: { title: string; value: string; note: string }) {
  return (
    <Card className="rounded-[24px] border-white/70 bg-white/85 shadow-[0_16px_40px_rgba(15,23,42,0.04)] dark:border-white/10 dark:bg-slate-950/60 dark:shadow-[0_20px_50px_rgba(2,8,23,0.35)]">
      <CardHeader>
        <p className="text-xs font-semibold uppercase tracking-[0.2em] text-slate-500 dark:text-slate-400">{title}</p>
        <CardTitle className="text-3xl tracking-tight text-slate-950 dark:text-slate-50">{value}</CardTitle>
        <p className="text-sm text-slate-500 dark:text-slate-400">{note}</p>
      </CardHeader>
    </Card>
  );
}
