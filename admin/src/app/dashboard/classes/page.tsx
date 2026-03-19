"use client";

import { useMemo, useState } from "react";
import { BookOpenText, Plus, Users, WandSparkles } from "lucide-react";
import { PageHero } from "@/components/page-hero";
import { StatePanel } from "@/components/state-panel";
import { StatCard } from "@/components/stat-card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { getClassManagementSummary, getMockClasses } from "@/lib/mock-classes.mjs";

function masteryTone(score: number) {
  if (score >= 0.75) return "bg-emerald-500";
  if (score >= 0.55) return "bg-sky-500";
  if (score >= 0.35) return "bg-amber-400";
  return "bg-rose-400";
}

export default function ClassManagementPage() {
  const classes = useMemo(() => getMockClasses(), []);
  const summary = getClassManagementSummary(classes);
  const [selectedClassID, setSelectedClassID] = useState(classes[0]?.id ?? "");
  const selectedClass = classes.find((item) => item.id === selectedClassID) ?? classes[0] ?? null;

  return (
    <div className="space-y-6">
      <PageHero
        eyebrow="Teaching operations"
        title="Class management"
        description="Frontend scaffold with mock data for class setup, join codes, member roster, and topic assignment. Backend integration is still pending."
        aside={
          <div className="grid gap-3 rounded-[24px] bg-slate-950 p-4 text-white dark:bg-slate-900/90">
            <div>
              <p className="text-xs uppercase tracking-[0.22em] text-slate-400">Current scaffold</p>
              <p className="mt-2 text-3xl font-semibold">Mock data only</p>
            </div>
            <p className="text-sm text-slate-300">
              Use this page to validate layout and information architecture before the real class backend lands.
            </p>
          </div>
        }
      >
        <div className="flex flex-wrap gap-3">
          <Dialog>
            <DialogTrigger render={<Button className="gap-2" />}>
              <Plus className="size-4" />
              New class
            </DialogTrigger>
            <DialogContent className="max-w-lg">
              <DialogHeader>
                <DialogTitle>Create class</DialogTitle>
                <DialogDescription>
                  This dialog is currently a UI scaffold. Submission and persistence will be connected once class management APIs exist.
                </DialogDescription>
              </DialogHeader>
              <div className="grid gap-4 py-2">
                <div className="space-y-2">
                  <Label htmlFor="class-name">Class name</Label>
                  <Input id="class-name" placeholder="Form 1 Algebra A" />
                </div>
                <div className="grid gap-4 sm:grid-cols-2">
                  <div className="space-y-2">
                    <Label>Syllabus</Label>
                    <Select defaultValue="KSSM Form 1">
                      <SelectTrigger className="w-full">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="KSSM Form 1">KSSM Form 1</SelectItem>
                        <SelectItem value="KSSM Form 2">KSSM Form 2</SelectItem>
                        <SelectItem value="KSSM Form 3">KSSM Form 3</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                  <div className="space-y-2">
                    <Label>Cadence</Label>
                    <Input placeholder="Mon, Wed, Fri" />
                  </div>
                </div>
              </div>
              <DialogFooter showCloseButton>
                <Button disabled>Create class</Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>

          <Button variant="outline" disabled className="gap-2">
            <WandSparkles className="size-4" />
            Assign topics
          </Button>
        </div>
      </PageHero>

      <StatePanel
        tone="empty"
        title="Mock-data scaffold"
        description="The class list, join codes, member roster, and topic assignment panels below are static mock data for Day 18 frontend planning. They are not backed by create/edit APIs yet."
      />

      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <StatCard icon={Users} title="Classes" value={String(summary.classCount)} note="Mock classes represented in this scaffold" />
        <StatCard icon={Users} title="Members" value={String(summary.totalMembers)} note="Total learners across the mock roster" />
        <StatCard icon={BookOpenText} title="Active learners" value={String(summary.activeStudents)} note="Students active in the latest mock snapshot" />
        <StatCard icon={WandSparkles} title="Avg mastery" value={`${summary.averageMastery}%`} note="Average mastery across the mock classes" />
      </section>

      <section className="grid gap-6 xl:grid-cols-[0.8fr_1.2fr]">
        <Card className="rounded-[28px] border-white/70 bg-white/85 shadow-[0_16px_50px_rgba(15,23,42,0.05)] dark:border-white/10 dark:bg-slate-950/60 dark:shadow-[0_20px_60px_rgba(2,8,23,0.35)]">
          <CardHeader className="space-y-4">
            <div>
              <CardTitle className="text-xl tracking-tight text-slate-900 dark:text-slate-100">Classes</CardTitle>
              <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">Switch between mock classes to inspect the intended management layout.</p>
            </div>
            <Select
              value={selectedClassID}
              onValueChange={(value) => setSelectedClassID(value ?? classes[0]?.id ?? "")}
            >
              <SelectTrigger className="w-full dark:border-white/15 dark:bg-slate-900/80 dark:text-slate-100 dark:hover:bg-slate-900">
                <SelectValue placeholder="Select a class" />
              </SelectTrigger>
              <SelectContent className="dark:border-white/10 dark:bg-slate-900 dark:text-slate-100">
                {classes.map((item) => (
                  <SelectItem key={item.id} value={item.id}>
                    {item.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </CardHeader>
          <CardContent className="space-y-3">
            {classes.map((item) => {
              const active = item.id === selectedClassID;
              return (
                <button
                  key={item.id}
                  type="button"
                  onClick={() => setSelectedClassID(item.id)}
                  className={`w-full rounded-[24px] border p-4 text-left transition ${
                    active
                      ? "border-sky-300 bg-sky-50 shadow-[0_12px_35px_rgba(14,165,233,0.12)] dark:border-sky-400/30 dark:bg-sky-400/10"
                      : "border-slate-200/80 bg-slate-50/80 hover:border-slate-300 hover:bg-white dark:border-white/10 dark:bg-slate-900/60 dark:hover:bg-slate-900/80"
                  }`}
                >
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <p className="text-sm font-semibold text-slate-900 dark:text-slate-100">{item.name}</p>
                      <p className="mt-1 text-xs uppercase tracking-[0.18em] text-slate-500 dark:text-slate-400">{item.syllabus}</p>
                    </div>
                    <span className="rounded-full bg-slate-950 px-3 py-1 text-[11px] font-semibold uppercase tracking-[0.16em] text-white dark:bg-slate-200 dark:text-slate-950">
                      {item.joinCode}
                    </span>
                  </div>
                  <p className="mt-3 text-sm leading-6 text-slate-600 dark:text-slate-300">{item.summary}</p>
                </button>
              );
            })}
          </CardContent>
        </Card>

        <div className="space-y-6">
          {selectedClass ? (
            <>
              <Card className="rounded-[28px] border-white/70 bg-white/85 shadow-[0_16px_50px_rgba(15,23,42,0.05)] dark:border-white/10 dark:bg-slate-950/60 dark:shadow-[0_20px_60px_rgba(2,8,23,0.35)]">
                <CardContent className="grid gap-4 p-6 md:grid-cols-[1.1fr_0.9fr]">
                  <div className="space-y-3">
                    <p className="text-xs font-semibold uppercase tracking-[0.2em] text-sky-700 dark:text-sky-300">Selected class</p>
                    <h2 className="text-2xl font-semibold tracking-tight text-slate-950 dark:text-white">{selectedClass.name}</h2>
                    <p className="text-sm leading-6 text-slate-600 dark:text-slate-300">{selectedClass.summary}</p>
                    <div className="flex flex-wrap gap-2 text-xs font-medium text-slate-500 dark:text-slate-400">
                      <span className="rounded-full bg-slate-100 px-3 py-1 dark:bg-slate-900/80">{selectedClass.subject}</span>
                      <span className="rounded-full bg-slate-100 px-3 py-1 dark:bg-slate-900/80">{selectedClass.syllabus}</span>
                      <span className="rounded-full bg-slate-100 px-3 py-1 dark:bg-slate-900/80">{selectedClass.cadence}</span>
                    </div>
                  </div>
                  <div className="grid gap-3 rounded-[24px] bg-slate-950 p-4 text-white dark:bg-slate-900/90">
                    <div>
                      <p className="text-xs uppercase tracking-[0.22em] text-slate-400">Join code</p>
                      <p className="mt-2 text-4xl font-semibold">{selectedClass.joinCode}</p>
                    </div>
                    <p className="text-sm text-slate-300">Mock join code display. Generation and refresh are pending backend support.</p>
                  </div>
                </CardContent>
              </Card>

              <section className="grid gap-6 xl:grid-cols-[1fr_0.95fr]">
                <Card className="rounded-[28px] border-white/70 bg-white/85 shadow-[0_16px_50px_rgba(15,23,42,0.05)] dark:border-white/10 dark:bg-slate-950/60 dark:shadow-[0_20px_60px_rgba(2,8,23,0.35)]">
                  <CardHeader>
                    <CardTitle className="text-xl tracking-tight text-slate-900 dark:text-slate-100">Member roster</CardTitle>
                    <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">Planned teacher view for class membership, current status, and quick intervention context.</p>
                  </CardHeader>
                  <CardContent>
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead className="text-slate-700 dark:text-slate-200">Learner</TableHead>
                          <TableHead className="text-slate-700 dark:text-slate-200">Status</TableHead>
                          <TableHead className="text-slate-700 dark:text-slate-200">Channel</TableHead>
                          <TableHead className="text-slate-700 dark:text-slate-200">Mastery</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {selectedClass.members.map((member) => (
                          <TableRow key={member.id}>
                            <TableCell className="font-medium text-slate-900 dark:text-slate-100">{member.name}</TableCell>
                            <TableCell className="text-slate-600 dark:text-slate-200">{member.status}</TableCell>
                            <TableCell className="capitalize text-slate-600 dark:text-slate-200">{member.channel}</TableCell>
                            <TableCell className="text-slate-700 dark:text-slate-100">{Math.round(member.mastery * 100)}%</TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </CardContent>
                </Card>

                <Card className="rounded-[28px] border-white/70 bg-white/85 shadow-[0_16px_50px_rgba(15,23,42,0.05)] dark:border-white/10 dark:bg-slate-950/60 dark:shadow-[0_20px_60px_rgba(2,8,23,0.35)]">
                  <CardHeader>
                    <CardTitle className="text-xl tracking-tight text-slate-900 dark:text-slate-100">Assigned topics</CardTitle>
                    <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">Planned topic-assignment panel for sequencing class-level study focus.</p>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    {selectedClass.assignedTopics.map((topic) => (
                      <div key={topic.id} className="space-y-2 rounded-[24px] border border-slate-200/80 p-4 dark:border-white/10">
                        <div className="flex items-center justify-between gap-3">
                          <div>
                            <p className="text-sm font-semibold text-slate-900 dark:text-slate-100">{topic.title}</p>
                            <p className="text-xs uppercase tracking-[0.18em] text-slate-500 dark:text-slate-400">{topic.status}</p>
                          </div>
                          <span className="text-sm font-medium text-slate-600 dark:text-slate-300">{Math.round(topic.progress * 100)}%</span>
                        </div>
                        <div className="h-3 overflow-hidden rounded-full bg-slate-200 dark:bg-slate-800">
                          <div className={`h-full rounded-full ${masteryTone(topic.progress)}`} style={{ width: `${Math.round(topic.progress * 100)}%` }} />
                        </div>
                      </div>
                    ))}
                    <Button variant="outline" disabled className="w-full">
                      Assign topics to class
                    </Button>
                  </CardContent>
                </Card>
              </section>
            </>
          ) : (
            <StatePanel
              tone="empty"
              title="No classes selected"
              description="Choose a mock class from the list to inspect the planned class-management layout."
            />
          )}
        </div>
      </section>
    </div>
  );
}
