"use client";

import { useState, useTransition } from "react";
import { IconCopy } from "@tabler/icons-react";
import { AdminSurface, AdminSurfaceHeader } from "@/components/admin-surface";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { issueInvite, type InviteRecord, type SubmitOnboardingResult } from "@/lib/api";

type InviteOutcome = {
  email: string;
  invite: InviteRecord | null;
  error: string;
};

function resolveInviteLink(invite: Pick<InviteRecord, "activation_url" | "invite_token">) {
  if (invite.activation_url?.trim()) {
    return invite.activation_url;
  }

  if (typeof window === "undefined") {
    return `/activate?token=${encodeURIComponent(invite.invite_token)}`;
  }

  return `${window.location.origin}/activate?token=${encodeURIComponent(invite.invite_token)}`;
}

function parseTeacherEmails(raw: string) {
  return Array.from(
    new Set(
      raw
        .split(/[\n,;]+/)
        .map((value) => value.trim().toLowerCase())
        .filter(Boolean),
    ),
  );
}

export function OnboardingSuccessSection({
  copyFeedback,
  onCopyJoinLink,
  onEditSetup,
  result,
  resultSchoolName,
}: {
  copyFeedback: string;
  onCopyJoinLink: () => void;
  onEditSetup: () => void;
  result: SubmitOnboardingResult;
  resultSchoolName: string;
}) {
  const [teacherEmails, setTeacherEmails] = useState("");
  const [inviteError, setInviteError] = useState("");
  const [inviteCopyFeedback, setInviteCopyFeedback] = useState("");
  const [inviteOutcomes, setInviteOutcomes] = useState<InviteOutcome[]>([]);
  const [isInvitePending, startInviteTransition] = useTransition();

  const sentCount = inviteOutcomes.filter((item) => item.invite?.delivery_status === "sent").length;
  const failedCount = inviteOutcomes.filter((item) => item.error || item.invite?.delivery_status === "failed").length;

  function handleInviteSubmit() {
    const emails = parseTeacherEmails(teacherEmails);
    if (emails.length === 0) {
      setInviteError("Add at least one teacher email.");
      return;
    }

    setInviteError("");
    setInviteCopyFeedback("");

    startInviteTransition(async () => {
      const outcomes: InviteOutcome[] = [];

      for (const email of emails) {
        try {
          const invite = await issueInvite({
            email,
            role: "teacher",
          });
          outcomes.push({ email, invite, error: "" });
        } catch (error) {
          outcomes.push({
            email,
            invite: null,
            error: error instanceof Error ? error.message : "Invite issuance failed",
          });
        }
      }

      setInviteOutcomes(outcomes);
      if (outcomes.some((item) => item.invite)) {
        setTeacherEmails("");
      }
    });
  }

  async function handleCopyInviteLink(invite: InviteRecord) {
    try {
      await navigator.clipboard.writeText(resolveInviteLink(invite));
      setInviteCopyFeedback(`Copied link for ${invite.email}`);
      window.setTimeout(() => {
        setInviteCopyFeedback((current) => (current === `Copied link for ${invite.email}` ? "" : current));
      }, 2000);
    } catch {
      setInviteError("Could not copy the activation link.");
    }
  }

  return (
    <AdminSurface className="overflow-hidden">
      <div className="flex flex-col gap-4">
        <AdminSurfaceHeader
          title="Setup complete"
          description="Your class is ready. Share it with students and invite teachers now or later."
        />

        <div className="rounded-2xl border bg-card p-5 md:p-6">
          <div className="mb-5 border-b pb-4">
            <p className="text-base font-medium text-foreground">
              <span className="font-semibold">{result.class_name}</span> is ready for <span className="font-semibold">{resultSchoolName}</span>.
            </p>
          </div>
          <div className="grid gap-6 lg:grid-cols-[minmax(0,0.9fr)_minmax(0,1.1fr)] lg:gap-8">
            <div className="space-y-4 lg:pr-2">
              <div className="space-y-1">
                <p className="text-sm font-medium text-foreground">Share with students</p>
                <p className="text-sm text-muted-foreground">Send this link to students so they can join the class.</p>
              </div>
              <Input readOnly value={result.join_link} aria-label="Join link" />
              <div className="flex flex-wrap items-center gap-3">
                <Button type="button" variant="outline" onClick={onCopyJoinLink}>
                  <IconCopy data-icon="inline-start" />
                  Copy link
                </Button>
                <Button type="button" variant="outline" onClick={() => window.open(result.join_link, "_blank", "noopener,noreferrer")}>
                  Open link
                </Button>
                {copyFeedback ? <p className="text-sm text-muted-foreground">{copyFeedback}</p> : null}
              </div>
            </div>

            <div className="space-y-4 border-t pt-5 lg:border-l lg:border-t-0 lg:pl-8 lg:pt-0">
              <div className="space-y-1">
                <p className="text-sm font-medium text-foreground">Invite teachers</p>
                <p className="text-sm text-muted-foreground">Add one email per line.</p>
              </div>

              <FieldGroup>
              <Field>
                <FieldLabel htmlFor="onboarding-teacher-emails">Teacher emails</FieldLabel>
                <Textarea
                  id="onboarding-teacher-emails"
                  value={teacherEmails}
                  onChange={(event) => setTeacherEmails(event.target.value)}
                  placeholder={"cikgu.aminah@school.my\ncikgu.rizal@school.my"}
                  rows={5}
                />
                <FieldDescription>Each teacher will receive an invite to set their password and open this workspace.</FieldDescription>
              </Field>
              </FieldGroup>

              <div className="flex flex-wrap gap-3">
                <Button type="button" onClick={handleInviteSubmit} disabled={isInvitePending}>
                  {isInvitePending ? "Sending invites..." : "Send invites"}
                </Button>
                <Button type="button" variant="outline" onClick={onEditSetup}>
                  Edit setup
                </Button>
              </div>
            </div>
          </div>
        </div>

        {inviteError ? (
          <Alert variant="destructive" className="mt-1">
            <AlertTitle>We could not send the invite</AlertTitle>
            <AlertDescription>{inviteError}</AlertDescription>
          </Alert>
        ) : null}

        {inviteOutcomes.length > 0 ? (
          <div className="space-y-4">
            <Alert>
              <AlertTitle>Teacher invite results</AlertTitle>
              <AlertDescription>
                {sentCount} sent, {failedCount} needing follow-up, {inviteOutcomes.length} processed.
              </AlertDescription>
            </Alert>

            <div className="space-y-3">
              {inviteOutcomes.map((item) => {
                const invite = item.invite;

                return (
                  <div key={item.email} className="rounded-xl border bg-muted/20 p-4">
                    <div className="flex flex-col gap-3 md:flex-row md:items-start md:justify-between">
                      <div className="space-y-1">
                        <p className="text-sm font-medium text-foreground">{item.email}</p>
                        <p className="text-sm text-muted-foreground">
                          {item.error
                            ? item.error
                            : invite?.delivery_status === "sent"
                              ? "Invite email sent."
                              : invite?.delivery_status === "failed"
                                ? "We could not send the email, but the teacher invite is ready."
                                : "Teacher invite is ready."}
                        </p>
                        {invite?.delivery_error ? (
                          <p className="text-sm text-rose-600 dark:text-rose-300">{invite.delivery_error}</p>
                        ) : null}
                      </div>

                      {invite ? (
                        <Button type="button" variant="outline" size="sm" onClick={() => void handleCopyInviteLink(invite)}>
                          Copy invite link
                        </Button>
                      ) : null}
                    </div>

                    {invite ? (
                      <Input className="mt-3" readOnly value={resolveInviteLink(invite)} aria-label={`Activation link for ${item.email}`} />
                    ) : null}
                  </div>
                );
              })}
            </div>

            {inviteCopyFeedback ? <p className="text-sm text-muted-foreground">{inviteCopyFeedback}</p> : null}
          </div>
        ) : null}
      </div>
    </AdminSurface>
  );
}
