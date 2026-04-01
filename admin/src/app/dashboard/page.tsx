"use client";

import { motion, useReducedMotion } from "framer-motion";
import Link from "next/link";
import { useState } from "react";
import { IconBellRinging, IconChevronRight } from "@tabler/icons-react";
import { AnimatedNumber } from "@/components/animated-number";
import { AdminSurface, AdminSurfaceHeader } from "@/components/admin-surface";
import { StatePanel } from "@/components/state-panel";
import { StatCard } from "@/components/stat-card";
import { Button } from "@/components/ui/button";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useAsyncResource } from "@/hooks/use-async-resource";
import { getDashboardSummary } from "@/lib/dashboard-view.mjs";
import { getClassProgress, sendStudentNudge, type ClassProgress } from "@/lib/api";
import { getMockClassProgress } from "@/lib/mock-classes.mjs";
import { formatTopicLabel } from "@/lib/topic-labels.mjs";

const dashboardEase = [0.22, 1, 0.36, 1] as const;

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

function attentionTone(attentionCount: number) {
  if (attentionCount > 0) return "text-amber-700 dark:text-amber-300";
  return "text-emerald-700 dark:text-emerald-300";
}

function masteryTone(score: number) {
  if (score >= 80) return "text-emerald-700 dark:text-emerald-300";
  if (score >= 60) return "text-amber-700 dark:text-amber-300";
  return "text-rose-700 dark:text-rose-300";
}

export default function DashboardPage() {
  const prefersReducedMotion = useReducedMotion();
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
  const sectionMotion = prefersReducedMotion
    ? { initial: false, animate: { opacity: 1 }, transition: { duration: 0 } }
    : {
        initial: { opacity: 0, y: 14, filter: "blur(12px)" },
        animate: { opacity: 1, y: 0, filter: "blur(0px)" },
        transition: { duration: 0.32, ease: dashboardEase },
      };

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
      <motion.header {...sectionMotion} transition={{ ...sectionMotion.transition, delay: 0.03 }} className="space-y-3">
        <div className="space-y-2">
          <p className="text-xs font-medium tracking-[0.08em] text-muted-foreground">Dashboard</p>
          <h1 className="text-3xl font-semibold tracking-tight text-foreground">Dashboard</h1>
          <p className="max-w-2xl text-sm leading-6 text-muted-foreground">
            {heroDescription}
          </p>
        </div>
      </motion.header>

      <motion.section
        {...sectionMotion}
        transition={{ ...sectionMotion.transition, delay: 0.08 }}
        className="grid gap-4 md:grid-cols-2 xl:grid-cols-4"
      >
        <StatCard
          title="Learners"
          value={String(summary.studentCount)}
          note={learnerNote}
          noteClassName={attentionTone(summary.attentionCount)}
        />
        <StatCard
          title="Class grade"
          value={classGrade}
          note={`${summary.averageMastery}% average mastery`}
          noteClassName={masteryTone(summary.averageMastery)}
        />
        <StatCard
          title="Average mastery"
          value={<AnimatedNumber value={summary.averageMastery} delay={0.12} formatter={(value) => `${value}%`} />}
          note={
            <>
              <span className="text-rose-700 dark:text-rose-300">Weakest: {weakestTopicLabel}</span>
              <span className="text-muted-foreground"> · </span>
              <span className="text-emerald-700 dark:text-emerald-300">Strongest: {strongestTopicLabel}</span>
            </>
          }
          noteClassName="flex flex-wrap gap-1"
        />
        <StatCard
          title="Coverage"
          value={<AnimatedNumber value={summary.coveragePercent} delay={0.18} formatter={(value) => `${value}%`} />}
          note={`${summary.trackedScores} of ${summary.studentCount * summary.topicCount} score slots filled`}
        />
      </motion.section>

      <motion.div {...sectionMotion} transition={{ ...sectionMotion.transition, delay: 0.14 }}>
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
                            <Tooltip>
                              <TooltipTrigger
                                render={
                                  <span className="inline-block max-w-28 truncate align-middle outline-none focus-visible:ring-0" tabIndex={0} />
                                }
                              >
                                {formatTopicLabel(topicId)}
                              </TooltipTrigger>
                              <TooltipContent>
                                <p>{formatTopicLabel(topicId)}</p>
                              </TooltipContent>
                            </Tooltip>
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
                            const score = student.topics[topicId];
                            const hasScore = typeof score === "number";
                            return (
                              <TableCell key={`${student.id}-${topicId}`} className="bg-muted/40 px-3 py-3">
                                {hasScore ? (
                                  <span className={`inline-flex rounded-full px-3 py-1 text-sm font-semibold ${scoreTone(score)}`}>
                                    {Math.round(score * 100)}%
                                  </span>
                                ) : (
                                  <span className="inline-flex px-3 py-1 text-sm font-medium text-muted-foreground">--</span>
                                )}
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
      </motion.div>
    </div>
  );
}
