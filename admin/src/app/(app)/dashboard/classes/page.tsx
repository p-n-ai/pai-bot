"use client";

import { motion, useReducedMotion } from "framer-motion";
import { useMemo, useState, useTransition } from "react";
import { IconBook2, IconPlus, IconUsers, IconWand } from "@tabler/icons-react";
import { useAsyncResource } from "@/hooks/use-async-resource";
import { AdminHighlightPanel } from "@/components/admin-highlight-panel";
import { AdminSurface, AdminSurfaceHeader } from "@/components/admin-surface";
import { ClassListItem } from "@/components/class-list-item";
import { InviteIssueForm } from "@/components/invite-issue-form";
import { PageHero } from "@/components/page-hero";
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
import {
  issueInvite,
  listGroups,
  createGroup,
  getGroupDetail,
  type InviteRecord,
  type GroupRecord,
  type GroupDetail,
} from "@/lib/api";
import { getMockAssignedTopics } from "@/lib/mock-classes.mjs";

const classPageEase = [0.22, 1, 0.36, 1] as const;

export default function ClassManagementPage() {
  const [refreshKey, setRefreshKey] = useState(0);
  const { data: classes, error: loadError } = useAsyncResource<GroupRecord[]>(
    () => listGroups("class").then((d) => d ?? []),
    [refreshKey],
  );
  const safeClasses = useMemo(() => classes ?? [], [classes]);
  const [selectedClassID, setSelectedClassID] = useState("");
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteRole, setInviteRole] = useState<"teacher" | "parent" | "admin">("teacher");
  const [inviteError, setInviteError] = useState("");
  const [latestInvite, setLatestInvite] = useState<InviteRecord | null>(null);
  const [isInvitePending, startInviteTransition] = useTransition();
  const [isCreatePending, startCreateTransition] = useTransition();
  const [createError, setCreateError] = useState("");
  const [createName, setCreateName] = useState("");
  const [createSyllabus, setCreateSyllabus] = useState("KSSM Form 1");
  const [createCadence, setCreateCadence] = useState("");
  const [createOpen, setCreateOpen] = useState(false);

  // Auto-select first class when data loads
  const effectiveSelectedID = selectedClassID || safeClasses[0]?.id || "";
  const selectedClass = safeClasses.find((item) => item.id === effectiveSelectedID) ?? safeClasses[0] ?? null;

  const { data: selectedDetail } = useAsyncResource<GroupDetail | null>(
    () => effectiveSelectedID ? getGroupDetail(effectiveSelectedID) : Promise.resolve(null),
    [effectiveSelectedID],
  );

  const summary = useMemo(() => {
    const totalMembers = safeClasses.reduce((sum, c) => sum + c.member_count, 0);
    return {
      classCount: safeClasses.length,
      totalMembers,
      activeStudents: totalMembers,
      averageMastery: 0,
    };
  }, [safeClasses]);

  const assignedTopics = useMemo(() => getMockAssignedTopics(), []);

  function handleCreateClass(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setCreateError("");
    startCreateTransition(async () => {
      try {
        const g = await createGroup({
          name: createName.trim(),
          type: "class",
          syllabus: createSyllabus,
          cadence: createCadence.trim(),
          subject: "Mathematics",
        });
        setCreateOpen(false);
        setCreateName("");
        setCreateCadence("");
        setRefreshKey((k) => k + 1);
        setSelectedClassID(g.id);
      } catch (error) {
        setCreateError(error instanceof Error ? error.message : "Failed to create class");
      }
    });
  }

  const prefersReducedMotion = useReducedMotion();
  const sectionMotion = prefersReducedMotion
    ? { initial: false, animate: { opacity: 1 }, transition: { duration: 0 } }
    : {
        initial: { opacity: 0, y: 14, filter: "blur(12px)" },
        animate: { opacity: 1, y: 0, filter: "blur(0px)" },
        transition: { duration: 0.32, ease: classPageEase },
      };

  function buildInviteLink(token: string) {
    if (typeof window === "undefined") {
      return `/activate?token=${encodeURIComponent(token)}`;
    }

    return `${window.location.origin}/activate?token=${encodeURIComponent(token)}`;
  }

  function resolveInviteLink(invite: Pick<InviteRecord, "activation_url" | "invite_token">) {
    if (invite.activation_url?.trim()) {
      return invite.activation_url;
    }
    return buildInviteLink(invite.invite_token);
  }

  function handleInviteSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setInviteError("");
    setLatestInvite(null);

    startInviteTransition(async () => {
      try {
        const invite = await issueInvite({
          email: inviteEmail.trim(),
          role: inviteRole,
        });
        setLatestInvite(invite);
      } catch (error) {
        setInviteError(error instanceof Error ? error.message : "Invite issuance failed");
      }
    });
  }

  return (
    <div className="space-y-6">
      <motion.div {...sectionMotion} transition={{ ...sectionMotion.transition, delay: 0.03 }}>
        <PageHero
          eyebrow="Teaching operations"
          title="Class management"
          description={loadError || "Create and manage classes, join codes, member rosters, and topic assignments."
          }
          surface="plain"
          aside={
            <AdminHighlightPanel>
              <div>
                <p className="text-xs uppercase tracking-[0.22em] text-slate-400">Classes</p>
                <p className="mt-2 text-3xl font-semibold">{safeClasses.length}</p>
              </div>
              <p className="text-sm text-slate-300">
                Students join via bot with /join CODE
              </p>
            </AdminHighlightPanel>
          }
        >
          <div className="flex flex-wrap gap-3">
            <Dialog open={createOpen} onOpenChange={setCreateOpen}>
              <DialogTrigger render={<Button className="gap-2" />}>
                <IconPlus data-icon="inline-start" />
                New class
              </DialogTrigger>
              <DialogContent className="max-w-lg">
                <DialogHeader>
                  <DialogTitle>Create class</DialogTitle>
                  <DialogDescription>
                    Create a new class. Students can join using the generated join code via the bot.
                  </DialogDescription>
                </DialogHeader>
                <form onSubmit={handleCreateClass}>
                  <div className="grid gap-4 py-2">
                    <div className="space-y-2">
                      <Label htmlFor="class-name">Class name</Label>
                      <Input id="class-name" placeholder="Form 1 Algebra A" value={createName} onChange={(e) => setCreateName(e.target.value)} required />
                    </div>
                    <div className="grid gap-4 sm:grid-cols-2">
                      <div className="space-y-2">
                        <Label>Syllabus</Label>
                        <Select value={createSyllabus} onValueChange={(v) => setCreateSyllabus(v ?? "KSSM Form 1")}>
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
                        <Input placeholder="Mon, Wed, Fri" value={createCadence} onChange={(e) => setCreateCadence(e.target.value)} />
                      </div>
                    </div>
                    {createError && <p className="text-sm text-red-600">{createError}</p>}
                  </div>
                  <DialogFooter showCloseButton>
                    <Button type="submit" disabled={isCreatePending || !createName.trim()}>
                      {isCreatePending ? "Creating..." : "Create class"}
                    </Button>
                  </DialogFooter>
                </form>
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
                    This action uses the live admin invite endpoint and emails the shared `/activate` flow when invite delivery is configured.
                  </DialogDescription>
                </DialogHeader>
                <InviteIssueForm
                  email={inviteEmail}
                  role={inviteRole}
                  error={inviteError}
                  inviteLink={latestInvite ? resolveInviteLink(latestInvite) : ""}
                  deliveryStatus={latestInvite?.delivery_status}
                  deliveryError={latestInvite?.delivery_error}
                  isPending={isInvitePending}
                  onEmailChange={setInviteEmail}
                  onRoleChange={setInviteRole}
                  onSubmit={handleInviteSubmit}
                />
              </DialogContent>
            </Dialog>
          </div>
        </PageHero>
      </motion.div>

      <motion.section
        {...sectionMotion}
        transition={{ ...sectionMotion.transition, delay: 0.08 }}
        className="grid gap-4 md:grid-cols-2 xl:grid-cols-4"
      >
        <StatCard icon={IconUsers} title="Classes" value={String(summary.classCount)} note="Total class groups" />
        <StatCard icon={IconUsers} title="Members" value={String(summary.totalMembers)} note="Total learners across all classes" />
        <StatCard icon={IconBook2} title="Active learners" value={String(summary.activeStudents)} note="Students enrolled" />
        <StatCard icon={IconWand} title="Avg mastery" value={summary.averageMastery ? `${summary.averageMastery}%` : "--"} note="Average mastery across classes" />
      </motion.section>

      <motion.section
        {...sectionMotion}
        transition={{ ...sectionMotion.transition, delay: 0.14 }}
        className="grid gap-6 xl:grid-cols-[0.8fr_1.2fr]"
      >
        <AdminSurface>
          <AdminSurfaceHeader title="Classes" description="Select a class to view its roster and join code." />
          <div className="mt-6 space-y-4">
            <Select
              value={selectedClassID}
              onValueChange={(value) => setSelectedClassID(value ?? safeClasses[0]?.id ?? "")}
            >
              <SelectTrigger className="w-full dark:border-white/15 dark:bg-slate-900/80 dark:text-slate-100 dark:hover:bg-slate-900">
                <SelectValue placeholder="Select a class">
                  {selectedClass?.name ?? "Select a class"}
                </SelectValue>
              </SelectTrigger>
              <SelectContent className="dark:border-white/10 dark:bg-slate-900 dark:text-slate-100">
                {safeClasses.map((item) => (
                  <SelectItem key={item.id} value={item.id}>
                    {item.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="mt-6 space-y-3">
            {safeClasses.map((item) => {
              return (
                <ClassListItem
                  key={item.id}
                  name={item.name}
                  syllabus={item.syllabus}
                  joinCode={item.join_code}
                  summary={`${item.syllabus}, ${item.member_count} learner${item.member_count !== 1 ? "s" : ""}`}
                  active={item.id === effectiveSelectedID}
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
                    <p className="text-sm leading-6 text-slate-600 dark:text-slate-300">{selectedClass.description || `${selectedClass.syllabus}, ${selectedClass.member_count} learner${selectedClass.member_count !== 1 ? "s" : ""}`}</p>
                    <div className="flex flex-wrap gap-2 text-xs font-medium text-slate-500 dark:text-slate-400">
                      {selectedClass.subject && <span className="rounded-full bg-slate-100 px-3 py-1 dark:bg-slate-900/80">{selectedClass.subject}</span>}
                      {selectedClass.syllabus && <span className="rounded-full bg-slate-100 px-3 py-1 dark:bg-slate-900/80">{selectedClass.syllabus}</span>}
                      {selectedClass.cadence && <span className="rounded-full bg-slate-100 px-3 py-1 dark:bg-slate-900/80">{selectedClass.cadence}</span>}
                    </div>
                  </div>
                  <AdminHighlightPanel>
                    <div>
                      <p className="text-xs uppercase tracking-[0.22em] text-slate-400">Join code</p>
                      <p className="mt-2 text-4xl font-semibold">{selectedClass.join_code}</p>
                    </div>
                    <p className="text-sm text-slate-300">Share this code with students. They join via /join {selectedClass.join_code} in the bot.</p>
                  </AdminHighlightPanel>
                </div>
              </AdminSurface>

              <section className="grid gap-6 xl:grid-cols-[1fr_0.95fr]">
                <AdminSurface>
                  <AdminSurfaceHeader
                    title="Member roster"
                    description="Class membership and mastery overview."
                  />
                  <div className="mt-6">
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead className="text-slate-700 dark:text-slate-200">Learner</TableHead>
                          <TableHead className="text-slate-700 dark:text-slate-200">Role</TableHead>
                          <TableHead className="text-slate-700 dark:text-slate-200">Channel</TableHead>
                          <TableHead className="text-slate-700 dark:text-slate-200">Mastery</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {(selectedDetail?.members ?? []).length === 0 ? (
                          <TableRow>
                            <TableCell colSpan={4} className="text-center text-slate-500">No members yet. Share the join code to get started.</TableCell>
                          </TableRow>
                        ) : (
                          (selectedDetail?.members ?? []).map((member) => (
                            <TableRow key={member.id}>
                              <TableCell className="font-medium text-slate-900 dark:text-slate-100">{member.name}</TableCell>
                              <TableCell className="capitalize text-slate-600 dark:text-slate-200">{member.role}</TableCell>
                              <TableCell className="capitalize text-slate-600 dark:text-slate-200">{member.channel}</TableCell>
                              <TableCell className="text-slate-700 dark:text-slate-100">{Math.round(member.mastery * 100)}%</TableCell>
                            </TableRow>
                          ))
                        )}
                      </TableBody>
                    </Table>
                  </div>
                </AdminSurface>

                <AdminSurface>
                  <AdminSurfaceHeader
                    title="Assigned topics"
                    description="Topic assignment (coming soon)"
                  />
                  <div className="mt-6 space-y-4">
                    {assignedTopics.map((topic: { id: string; title: string; status: string; progress: number }) => (
                      <TopicProgressRow key={topic.id} title={topic.title} status={topic.status} progress={topic.progress} />
                    ))}
                    <Button variant="outline" disabled className="w-full">
                      Assign topics to class (coming soon)
                    </Button>
                  </div>
                </AdminSurface>
              </section>
            </>
          ) : (
            <p className="rounded-xl border border-dashed border-border px-4 py-6 text-sm text-muted-foreground">
              Choose a mock class from the list to inspect the planned class-management layout.
            </p>
          )}
        </div>
      </motion.section>
    </div>
  );
}
