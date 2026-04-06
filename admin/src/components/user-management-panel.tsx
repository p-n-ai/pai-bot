"use client";

import { useState, useTransition } from "react";
import { IconMailPlus, IconSearch, IconUsers } from "@tabler/icons-react";
import { InviteIssueForm } from "@/components/invite-issue-form";
import { StatePanel } from "@/components/state-panel";
import { StatCard } from "@/components/stat-card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { issueInvite, type UserManagementView } from "@/lib/api";

function buildInviteLink(token: string) {
  if (typeof window === "undefined") {
    return `/activate?token=${encodeURIComponent(token)}`;
  }

  return `${window.location.origin}/activate?token=${encodeURIComponent(token)}`;
}

export function UserManagementPanel({
  data,
  loadError = "",
}: {
  data: UserManagementView | null;
  loadError?: string;
}) {
  const [search, setSearch] = useState("");
  const [inviteEmail, setInviteEmail] = useState("");
  const [inviteRole, setInviteRole] = useState<"teacher" | "parent" | "admin">("teacher");
  const [inviteError, setInviteError] = useState("");
  const [inviteLink, setInviteLink] = useState("");
  const [isInvitePending, startInviteTransition] = useTransition();

  if (!data) {
    return (
      <StatePanel
        tone={loadError ? "error" : "empty"}
        title={loadError ? "User management unavailable" : "No user management data yet"}
        description={loadError || "Users and invites will appear here once the admin API returns workspace data."}
      />
    );
  }

  const summary = data.summary ?? {
    teachers: 0,
    parents: 0,
    pending_invites: 0,
    total_users: 0,
  };
  const activeUserList = Array.isArray(data.active_users) ? data.active_users : [];
  const pendingInviteList = Array.isArray(data.pending_invites) ? data.pending_invites : [];
  const query = search.trim().toLowerCase();
  const activeUsers = activeUserList.filter((item) => {
    if (!query) {
      return true;
    }
    return [item.name, item.email, item.role, item.tenant_name ?? ""].some((value) =>
      String(value ?? "").toLowerCase().includes(query),
    );
  });
  const pendingInvites = pendingInviteList.filter((item) => {
    if (!query) {
      return true;
    }
    return [item.email, item.role, item.invited_by, item.tenant_name ?? ""].some((value) =>
      String(value ?? "").toLowerCase().includes(query),
    );
  });

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
      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <StatCard icon={IconUsers} title="Teachers" value={String(summary.teachers)} note="Active teacher accounts in this workspace" />
        <StatCard icon={IconUsers} title="Parents" value={String(summary.parents)} note="Active parent accounts with access to summaries" />
        <StatCard icon={IconMailPlus} title="Pending invites" value={String(summary.pending_invites)} note="Outstanding invite links that have not been accepted" />
        <StatCard icon={IconUsers} title="Total users" value={String(summary.total_users)} note="All active teacher, parent, admin, and platform-admin users" />
      </section>

      <section className="space-y-4 rounded-[28px] border border-slate-200/80 bg-white/85 p-5 shadow-[0_18px_60px_rgba(15,23,42,0.06)] dark:border-white/10 dark:bg-slate-950/55 dark:shadow-[0_24px_70px_rgba(2,8,23,0.32)]">
        <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
          <div className="relative max-w-xl flex-1">
            <IconSearch className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-slate-400" />
            <Input
              aria-label="Search users"
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder="Search by name, email, role, or school"
              className="pl-9"
            />
          </div>
          <Dialog>
            <DialogTrigger render={<Button className="gap-2" />}>
              <IconMailPlus data-icon="inline-start" />
              Invite user
            </DialogTrigger>
            <DialogContent className="max-w-lg">
              <DialogHeader>
                <DialogTitle>Invite teacher, parent, or admin</DialogTitle>
                <DialogDescription>
                  Create a new invite and share the generated activation link through your existing email or messaging flow.
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

        <Tabs defaultValue="active" className="space-y-4">
          <TabsList>
            <TabsTrigger value="active">Active users</TabsTrigger>
            <TabsTrigger value="invites">Pending invites</TabsTrigger>
          </TabsList>

          <TabsContent value="active" className="space-y-3">
            {activeUsers.length > 0 ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Email</TableHead>
                    <TableHead>Role</TableHead>
                    <TableHead>Status</TableHead>
                    <TableHead>Created</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {activeUsers.map((item) => (
                    <TableRow key={item.id}>
                      <TableCell className="font-medium text-slate-900 dark:text-slate-100">
                        <div className="space-y-1">
                          <p>{item.name}</p>
                          {item.tenant_name ? <p className="text-xs text-slate-500 dark:text-slate-400">{item.tenant_name}</p> : null}
                        </div>
                      </TableCell>
                      <TableCell>{item.email || "No email linked"}</TableCell>
                      <TableCell className="capitalize">{item.role.replaceAll("_", " ")}</TableCell>
                      <TableCell className="capitalize">{item.status}</TableCell>
                      <TableCell>{item.created_at.slice(0, 10)}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            ) : (
              <StatePanel
                tone="empty"
                title="No active users match this search"
                description="Try a different name, role, or email filter to find someone in the workspace."
              />
            )}
          </TabsContent>

          <TabsContent value="invites" className="space-y-3">
            {pendingInvites.length > 0 ? (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Email</TableHead>
                    <TableHead>Role</TableHead>
                    <TableHead>Invited by</TableHead>
                    <TableHead>Expires</TableHead>
                    <TableHead>Status</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {pendingInvites.map((item) => (
                    <TableRow key={item.id}>
                      <TableCell className="font-medium text-slate-900 dark:text-slate-100">
                        <div className="space-y-1">
                          <p>{item.email}</p>
                          {item.tenant_name ? <p className="text-xs text-slate-500 dark:text-slate-400">{item.tenant_name}</p> : null}
                        </div>
                      </TableCell>
                      <TableCell className="capitalize">{item.role.replaceAll("_", " ")}</TableCell>
                      <TableCell>{item.invited_by || "System"}</TableCell>
                      <TableCell>{item.expires_at.slice(0, 10)}</TableCell>
                      <TableCell className="capitalize">{item.status}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            ) : (
              <StatePanel
                tone="empty"
                title="No pending invites match this search"
                description="Invite links that are still open will appear here until they are accepted or expire."
              />
            )}
          </TabsContent>
        </Tabs>
      </section>
    </div>
  );
}
