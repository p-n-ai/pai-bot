"use client";

import Link from "next/link";
import { useState } from "react";
import { BellRing, ChevronRight, Sparkles } from "lucide-react";
import { PageHero } from "@/components/page-hero";
import { StatePanel } from "@/components/state-panel";
import { StatCard } from "@/components/stat-card";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useAsyncResource } from "@/hooks/use-async-resource";
import { getDashboardSummary } from "@/lib/dashboard-view.mjs";
import { getClassProgress, sendStudentNudge, type ClassProgress } from "@/lib/api";
import { formatTopicLabel } from "@/lib/topic-labels.mjs";

function scoreTone(score: number) {
  if (score >= 0.8) return "border border-emerald-200 bg-emerald-100 text-emerald-900 dark:border-emerald-400/20 dark:bg-emerald-400/18 dark:text-emerald-50";
  if (score >= 0.6) return "border border-lime-200 bg-lime-100 text-lime-900 dark:border-lime-400/20 dark:bg-lime-400/18 dark:text-lime-50";
  if (score >= 0.4) return "border border-amber-200 bg-amber-100 text-amber-900 dark:border-amber-400/20 dark:bg-amber-400/18 dark:text-amber-50";
  return "border border-rose-200 bg-rose-100 text-rose-900 dark:border-rose-400/20 dark:bg-rose-400/18 dark:text-rose-50";
}

export default function DashboardPage() {
  const { data, loading, error } = useAsyncResource<ClassProgress>(() => getClassProgress("all-students"), []);
  const [nudgeMessage, setNudgeMessage] = useState("");
  const [sendingStudentID, setSendingStudentID] = useState("");
  const summary = getDashboardSummary(data);

  async function handleNudge(studentID: string, studentName: string) {
    setSendingStudentID(studentID);
    setNudgeMessage("");
    try {
      await sendStudentNudge(studentID);
      setNudgeMessage(`Nudge sent to ${studentName} on Telegram.`);
    } catch (error) {
      const message = error instanceof Error ? error.message : "Couldn't send the nudge right now. Please try again.";
      setNudgeMessage(message);
    } finally {
      setSendingStudentID("");
    }
  }

  return (
    <div className="space-y-6">
        <PageHero
          eyebrow="Teacher cockpit"
          title="Class mastery at a glance"
          description="Review topic-by-topic mastery and open each learner profile for a closer look."
          aside={
            <div className="grid gap-3 rounded-[24px] bg-slate-950 p-4 text-white dark:bg-slate-900/90">
            <div>
              <p className="text-xs uppercase tracking-[0.22em] text-slate-400">Average mastery</p>
              <p className="mt-2 text-4xl font-semibold">{summary.averageMastery}%</p>
            </div>
            <div className="flex items-center gap-2 text-sm text-slate-300">
              <Sparkles className="size-4 text-amber-300" />
              Live data from the Go admin API
            </div>
          </div>
          }
        />

        <section className="grid gap-4 md:grid-cols-3">
          <StatCard title="Students" value={String(summary.studentCount)} note="Tracked in this view" />
          <StatCard title="Topics" value={String(summary.topicCount)} note="Algebra sequence" />
          <StatCard title="Tracked Scores" value={String(summary.trackedScores)} note="Real mastery entries loaded" />
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
              <CardTitle className="text-xl tracking-tight text-slate-950 dark:text-slate-50">Mastery heatmap</CardTitle>
              <p className="mt-1 text-sm text-slate-600 dark:text-slate-300">Students by topic with direct navigation into detail views.</p>
            </div>
            <Link href="/" className="text-sm font-medium text-sky-700 hover:text-sky-900 dark:text-sky-300 dark:hover:text-sky-200">
              Back home
            </Link>
          </CardHeader>
          <CardContent>
            {loading ? (
              <StatePanel
                tone="loading"
                title="Preparing the latest class snapshot"
                description="Pulling student mastery, tracked topics, and direct links into the learner detail pages."
              />
            ) : data ? (
              !summary.hasHeatmap ? (
                <StatePanel
                  tone="empty"
                  title="No class heatmap yet"
                  description="Class progress will appear here once students start working through assigned topics."
                />
              ) : (
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
                          <td className="rounded-l-2xl bg-slate-50/80 px-3 py-3 text-sm font-medium text-slate-900 dark:bg-slate-900/70 dark:text-slate-100">
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
                              <td key={`${student.id}-${topicId}`} className="bg-slate-50/80 px-3 py-3 dark:bg-slate-900/70">
                                <span className={`inline-flex rounded-full px-3 py-1 text-sm font-semibold ${scoreTone(score)}`}>
                                  {Math.round(score * 100)}%
                                </span>
                              </td>
                            );
                          })}
                          <td className="rounded-r-2xl bg-slate-50/80 px-3 py-3 dark:bg-slate-900/70">
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
              )
            ) : (
              <StatePanel
                tone={error ? "error" : "empty"}
                title={error ? "Class data is unavailable" : "Waiting for class data"}
                description={error ? "Class data isn't available right now. Please try again in a moment." : "Class data will appear here once it is available."}
              />
            )}
          </CardContent>
        </Card>
    </div>
  );
}
