"use client";

import type { FormEvent } from "react";
import { FormField } from "@/components/form-field";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";

export function InviteIssueForm({
  email,
  role,
  error,
  inviteLink,
  copyFeedback,
  isPending,
  onEmailChange,
  onRoleChange,
  onCopyLink,
  onSubmit,
}: {
  email: string;
  role: "teacher" | "parent" | "admin";
  error: string;
  inviteLink: string;
  copyFeedback?: string;
  isPending: boolean;
  onEmailChange: (value: string) => void;
  onRoleChange: (value: "teacher" | "parent" | "admin") => void;
  onCopyLink?: () => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}) {
  return (
    <form className="space-y-5" onSubmit={onSubmit}>
      <FormField label="Email" htmlFor="invite-email">
        <Input
          id="invite-email"
          name="invite-email"
          type="email"
          value={email}
          onChange={(event) => onEmailChange(event.target.value)}
          placeholder="teacher@example.com"
          autoComplete="email"
          required
        />
      </FormField>

      <FormField
        label="Role"
        htmlFor="invite-role"
        description="Admin invite issuance is currently for teacher, parent, and admin accounts."
      >
        <Select value={role} onValueChange={(value) => onRoleChange(value as "teacher" | "parent" | "admin")}>
          <SelectTrigger id="invite-role" className="h-10 w-full">
            <SelectValue placeholder="Select a role" />
          </SelectTrigger>
          <SelectContent align="start">
            <SelectItem value="teacher">Teacher</SelectItem>
            <SelectItem value="parent">Parent</SelectItem>
            <SelectItem value="admin">Admin</SelectItem>
          </SelectContent>
        </Select>
      </FormField>

      {error ? (
        <div className="rounded-2xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700 dark:border-rose-400/30 dark:bg-rose-500/10 dark:text-rose-200">
          {error}
        </div>
      ) : null}

      {inviteLink ? (
        <FormField
          label="Activation link"
          htmlFor="invite-link"
          description="Copy this link into the invite email or message. The recipient will set their password through this URL."
        >
          <Input id="invite-link" value={inviteLink} readOnly />
          <div className="mt-3 flex items-center gap-3">
            <Button type="button" variant="outline" size="sm" onClick={onCopyLink} disabled={!onCopyLink}>
              Copy link
            </Button>
            {copyFeedback ? <p className="text-sm text-muted-foreground">{copyFeedback}</p> : null}
          </div>
        </FormField>
      ) : null}

      <Button type="submit" className="w-full" disabled={isPending}>
        {isPending ? "Sending invite..." : "Send invite"}
      </Button>
    </form>
  );
}
