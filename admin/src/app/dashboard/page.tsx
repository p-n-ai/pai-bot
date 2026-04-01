"use client";

import Link from "next/link";
import { useState } from "react";
import { IconBellRinging, IconChevronRight } from "@tabler/icons-react";
import { AdminSurface, AdminSurfaceHeader } from "@/components/admin-surface";
import { StatePanel } from "@/components/state-panel";
import { StatCard } from "@/components/stat-card";
import { Button } from "@/components/ui/button";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { useAsyncResource } from "@/hooks/use-async-resource";
import { getDashboardSummary } from "@/lib/dashboard-view.mjs";
import { getClassProgress, sendStudentNudge, type ClassProgress } from "@/lib/api";
import { getMockClassProgress } from "@/lib/mock-classes.mjs";
import { formatTopicLabel } from "@/lib/topic-labels.mjs";

function scoreTone(score: number) {
  if (score >= 0.8) return "border border-emerald-200 bg-emerald-100 text-emerald-900 dark:border-emerald-400/20 dark:bg-emerald-400/18 dark:text-emerald-50";
  if (score >= 0.6) return "border border-lime-200 bg-lime-100 text-lime-900 dark:border-lime-400/20 dark:bg-lime-400/18 dark:text-lime-50";
  if (score >= 0.4) return "border border-amber-200 bg-amber-100 text-amber-900 dark:border-amber-400/20 dark:bg-amber-400/18 dark:text-amber-50";
  return "border border-rose-200 bg-rose-100 text-rose-900 dark:border-rose-400/20 dark:bg-rose-400/18 dark:text-rose-50";
}

function masteryGrade(averageMastery: number) {
  if (averageMastery >= 90) return "A";
  if (averageMastery >= 80) return "B";
  if (averageMastery >= 70) return "C";
  if (averageMastery >= 60) return "D";
  if (averageMastery >= 50) return "E";
  return "F";
}

export default function DashboardPage() {
  const { data, loading } = useAsyncResource<{
    progress: ClassProgress;
    source: "live" | "preview";
    issue?: string;
  }>(async () => {
    try {
      return {
        progress: await getClassProgress("all-students"),
        source: "live" as const,
      };
    } catch (error) {
      return {
        progress: getMockClassProgress("all-students"),
        source: "preview" as const,
        issue: error instanceof Error ? error.message : "Class data is unavailable right now.",
      };
    }
  }, []);
  const [nudgeMessage, setNudgeMessage] = useState("");
  const [sendingStudentID, setSendingStudentID] = useState("");
  const progress = data?.progress ?? null;
  const summary = getDashboardSummary(progress);
  const isPreview = data?.source === "preview";
  const weakestTopicLabel = summary.weakestTopic ? formatTopicLabel(summary.weakestTopic.topicId) : "No topic data";
  const strongestTopicLabel = summary.strongestTopic ? formatTopicLabel(summary.strongestTopic.topicId) : "No topic data";
  const heroDescription = "Track who needs support today across the class.";
  const learnerNote =
    summary.attentionCount > 0
      ? `${summary.attentionCount} learner${summary.attentionCount === 1 ? "" : "s"} need attention`
      : "No learners flagged right now";
  const classGrade = masteryGrade(summary.averageMastery);

  async function handleNudge(studentID: string, studentName: string) {
    if (isPreview) {
      setNudgeMessage("Preview mode only. Connect the live admin API to send nudges.");
      return;
    }

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
        <header className="space-y-4">
          <div className="space-y-2">
            <p className="text-xs font-medium tracking-[0.08em] text-muted-foreground">Dashboard</p>
            <h1 className="text-3xl font-semibold tracking-tight text-foreground">Dashboard</h1>
            <p className="max-w-2xl text-sm leading-6 text-muted-foreground">
              {heroDescription}
            </p>
          </div>
        </header>

        <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
          <StatCard title="Learners" value={String(summary.studentCount)} note={learnerNote} />
          <StatCard
            title="Class grade"
            value={classGrade}
            note={`${summary.averageMastery}% average mastery`}
          />
          <StatCard title="Average mastery" value={`${summary.averageMastery}%`} note={`Weakest: ${weakestTopicLabel} · strongest: ${strongestTopicLabel}`} />
          <StatCard
            title="Coverage"
            value={`${summary.coveragePercent}%`}
            note={`${summary.trackedScores} of ${summary.studentCount * summary.topicCount} score slots filled`}
          />
        </section>

        <AdminSurface>
          <AdminSurfaceHeader
            title="Mastery heatmap"
            description="Students by topic with direct navigation into detail views."
          />
          <div className="mt-6">
            {loading ? (
              <StatePanel
                tone="loading"
                title="Preparing the latest class snapshot"
                description="Pulling student mastery, tracked topics, and direct links into the learner detail pages."
              />
            ) : progress ? (
              !summary.hasHeatmap ? (
                <StatePanel
                  tone="empty"
                  title="No class heatmap yet"
                  description="Class progress will appear here once students start working through assigned topics."
                />
              ) : (
              <div className="space-y-5">
                <Table className="min-w-[760px] border-separate border-spacing-y-2">
                  <TableHeader>
                    <TableRow className="border-none hover:bg-transparent">
                      <TableHead className="px-3 py-2 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">Student</TableHead>
                        {progress.topic_ids.map((topicId) => (
                          <TableHead
                            key={topicId}
                            className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground"
                          >
                            {formatTopicLabel(topicId)}
                          </TableHead>
                        ))}
                      <TableHead className="px-3 py-2 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground">Nudge</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                      {progress.students.map((student) => (
                        <TableRow key={student.id} className="border-none hover:bg-transparent">
                          <TableCell className="rounded-l-2xl bg-muted/40 px-3 py-3 text-sm font-medium text-foreground">
                            <Link
                              href={`/students/${student.id}`}
                              className="inline-flex items-center gap-2 hover:text-primary"
                            >
                              {student.name}
                              <IconChevronRight />
                            </Link>
                          </TableCell>
                          {progress.topic_ids.map((topicId) => {
                            const score = student.topics[topicId] ?? 0;
                            return (
                              <TableCell key={`${student.id}-${topicId}`} className="bg-muted/40 px-3 py-3">
                                <span className={`inline-flex rounded-full px-3 py-1 text-sm font-semibold ${scoreTone(score)}`}>
                                  {Math.round(score * 100)}%
                                </span>
                              </TableCell>
                            );
                          })}
                          <TableCell className="rounded-r-2xl bg-muted/40 px-3 py-3">
                            <Button
                              size="sm"
                              className="gap-2"
                              disabled={sendingStudentID === student.id || isPreview}
                              onClick={() => handleNudge(student.id, student.name)}
                            >
                              <IconBellRinging data-icon="inline-start" />
                              {isPreview ? "Preview" : sendingStudentID === student.id ? "Sending..." : "Nudge"}
                            </Button>
                          </TableCell>
                        </TableRow>
                      ))}
                  </TableBody>
                </Table>
                {nudgeMessage ? <p className="text-sm text-muted-foreground">{nudgeMessage}</p> : null}
              </div>
              )
            ) : (
              <StatePanel
                tone="empty"
                title="Waiting for class data"
                description="Class data will appear here once it is available."
              />
            )}
          </div>
        </AdminSurface>
    </div>
  );
}
