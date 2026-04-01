"use client";

import { useMemo, useState, useTransition } from "react";
import { IconBook2, IconPlus, IconUsers, IconWand } from "@tabler/icons-react";
import { AdminHighlightPanel } from "@/components/admin-highlight-panel";
import { AdminSurface, AdminSurfaceHeader } from "@/components/admin-surface";
import { ClassListItem } from "@/components/class-list-item";
import { InviteIssueForm } from "@/components/invite-issue-form";
import { PageHero } from "@/components/page-hero";
import { StatePanel } from "@/components/state-panel";
import { StatCard } from "@/components/stat-card";
import { TopicProgressRow } from "@/components/topic-progress-row";
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
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { issueInvite } from "@/lib/api";
import { getClassManagementSummary, getMockClasses } from "@/lib/mock-classes.mjs";

export default function ClassManagementPage() {
  const classes = useMemo(() => getMockClasses(), []);
  const summary = getClassManagementSummary(classes);
  const [selectedClassID, setSelectedClassID] = useState(classes[0]?.id ?? "");
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteRole, setInviteRole] = useState<"teacher" | "parent" | "admin">("teacher");
  const [inviteError, setInviteError] = useState("");
  const [inviteLink, setInviteLink] = useState("");
  const [isInvitePending, startInviteTransition] = useTransition();
  const selectedClass = classes.find((item) => item.id === selectedClassID) ?? classes[0] ?? null;

  function buildInviteLink(token: string) {
    if (typeof window === "undefined") {
      return `/activate?token=${encodeURIComponent(token)}`;
    }

    return `${window.location.origin}/activate?token=${encodeURIComponent(token)}`;
  }

  function handleInviteSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setInviteError("");
    setInviteLink("");

    startInviteTransition(async () => {
      try {
        const invite = await issueInvite({
          email: inviteEmail.trim(),
          role: inviteRole,
        });
        setInviteLink(buildInviteLink(invite.invite_token));
      } catch (error) {
        setInviteError(error instanceof Error ? error.message : "Invite issuance failed");
      }
    });
  }

  return (
    <div className="space-y-6">
      <PageHero
        eyebrow="Teaching operations"
        title="Class management"
        description="Frontend scaffold with mock data for class setup, join codes, member roster, and topic assignment. The real class backend contract is not implemented yet."
        aside={
          <AdminHighlightPanel>
            <div>
              <p className="text-xs uppercase tracking-[0.22em] text-slate-400">Current scaffold</p>
              <p className="mt-2 text-3xl font-semibold">Mock data only</p>
            </div>
            <p className="text-sm text-slate-300">
              Use this page to validate layout and information architecture only. None of the class-management actions below persist yet.
            </p>
          </AdminHighlightPanel>
        }
      >
        <div className="flex flex-wrap gap-3">
          <Dialog>
            <DialogTrigger render={<Button className="gap-2" />}>
              <IconPlus data-icon="inline-start" />
              New class
            </DialogTrigger>
            <DialogContent className="max-w-lg">
              <DialogHeader>
                <DialogTitle>Create class</DialogTitle>
                <DialogDescription>
                  This dialog is currently a UI scaffold. Submission is intentionally disabled until create-class and class-edit APIs exist.
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
            <IconWand data-icon="inline-start" />
            Assign topics
          </Button>

          <Dialog>
            <DialogTrigger render={<Button variant="outline" className="gap-2" />}>
              <IconUsers data-icon="inline-start" />
              Invite user
            </DialogTrigger>
            <DialogContent className="max-w-lg">
              <DialogHeader>
                <DialogTitle>Invite teacher, parent, or admin</DialogTitle>
                <DialogDescription>
                  This action uses the live admin invite endpoint. The returned activation link opens the new `/activate` flow.
                </DialogDescription>
              </DialogHeader>
              <InviteIssueForm
                email={inviteEmail}
                role={inviteRole}
                error={inviteError}
                inviteLink={inviteLink}
                isPending={isInvitePending}
                onEmailChange={setInviteEmail}
                onRoleChange={setInviteRole}
                onSubmit={handleInviteSubmit}
              />
            </DialogContent>
          </Dialog>
        </div>
      </PageHero>

      <StatePanel
        tone="empty"
        title="Mock-data scaffold"
        description="The class list, join codes, member roster, and topic assignment panels below are static mock data for Day 18 frontend planning. They are not backed by live class, membership, join-code, or topic-assignment APIs yet."
      />

      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <StatCard icon={IconUsers} title="Classes" value={String(summary.classCount)} note="Mock classes represented in this scaffold" />
        <StatCard icon={IconUsers} title="Members" value={String(summary.totalMembers)} note="Total learners across the mock roster" />
        <StatCard icon={IconBook2} title="Active learners" value={String(summary.activeStudents)} note="Students active in the latest mock snapshot" />
        <StatCard icon={IconWand} title="Avg mastery" value={`${summary.averageMastery}%`} note="Average mastery across the mock classes" />
      </section>

      <section className="grid gap-6 xl:grid-cols-[0.8fr_1.2fr]">
        <AdminSurface>
          <AdminSurfaceHeader title="Classes" description="Switch between mock classes to inspect the intended management layout." />
          <div className="mt-6 space-y-4">
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
          </div>
          <div className="mt-6 space-y-3">
            {classes.map((item) => {
              return (
                <ClassListItem
                  key={item.id}
                  name={item.name}
                  syllabus={item.syllabus}
                  joinCode={item.joinCode}
                  summary={item.summary}
                  active={item.id === selectedClassID}
                  onClick={() => setSelectedClassID(item.id)}
                />
              );
            })}
          </div>
        </AdminSurface>

        <div className="space-y-6">
          {selectedClass ? (
            <>
              <AdminSurface>
                <div className="grid gap-4 md:grid-cols-[1.1fr_0.9fr]">
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
                  <AdminHighlightPanel>
                    <div>
                      <p className="text-xs uppercase tracking-[0.22em] text-slate-400">Join code</p>
                      <p className="mt-2 text-4xl font-semibold">{selectedClass.joinCode}</p>
                    </div>
                    <p className="text-sm text-slate-300">Mock join code display only. Generation, refresh, and member enrollment remain blocked on backend APIs.</p>
                  </AdminHighlightPanel>
                </div>
              </AdminSurface>

              <section className="grid gap-6 xl:grid-cols-[1fr_0.95fr]">
                <AdminSurface>
                  <AdminSurfaceHeader
                    title="Member roster"
                    description="Planned teacher view for class membership, current status, and quick intervention context."
                  />
                  <div className="mt-6">
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
                  </div>
                </AdminSurface>

                <AdminSurface>
                  <AdminSurfaceHeader
                    title="Assigned topics"
                    description="Planned topic-assignment panel for sequencing class-level study focus."
                  />
                  <div className="mt-6 space-y-4">
                    {selectedClass.assignedTopics.map((topic) => (
                      <TopicProgressRow key={topic.id} title={topic.title} status={topic.status} progress={topic.progress} />
                    ))}
                    <Button variant="outline" disabled className="w-full">
                      Assign topics to class
                    </Button>
                  </div>
                </AdminSurface>
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
