import { CopyIcon, MailPlusIcon } from 'lucide-react'
import type { ChangeEvent, FormEvent } from 'react'

import type { InviteIssueRole, InviteRecord } from '@/lib/user-management-types'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

// fallow-ignore-next-line complexity
export function InviteForm({
  copyFeedback,
  email,
  error,
  isSubmitting,
  latestInvite,
  onCopyLink,
  onEmailChange,
  onRoleChange,
  onSubmit,
  role,
}: {
  copyFeedback: string
  email: string
  error: string
  isSubmitting: boolean
  latestInvite: InviteRecord | null
  onCopyLink: () => void
  onEmailChange: (event: ChangeEvent<HTMLInputElement>) => void
  onRoleChange: (value: InviteIssueRole) => void
  onSubmit: (event: FormEvent<HTMLFormElement>) => void
  role: InviteIssueRole
}) {
  return (
    <form className='grid gap-3.5' onSubmit={onSubmit}>
      <InviteEmailField email={email} onEmailChange={onEmailChange} />
      <InviteRoleField onRoleChange={onRoleChange} role={role} />
      {error ? <p className='text-muted-foreground'>{error}</p> : null}
      {latestInvite ? (
        <LatestInvite
          copyFeedback={copyFeedback}
          invite={latestInvite}
          onCopyLink={onCopyLink}
        />
      ) : null}
      <Button disabled={isSubmitting || !email.trim()} type='submit'>
        <MailPlusIcon data-icon='inline-start' />
        {isSubmitting ? 'Sending invite...' : 'Send invite'}
      </Button>
    </form>
  )
}

export function resolveInviteLink(invite: InviteRecord): string {
  if (invite.activation_url?.trim()) {
    return invite.activation_url
  }

  return buildInviteLink(invite.invite_token)
}

export async function copyInviteLink(inviteLink: string): Promise<void> {
  await navigator.clipboard.writeText(inviteLink)
}

function InviteEmailField({
  email,
  onEmailChange,
}: {
  email: string
  onEmailChange: (event: ChangeEvent<HTMLInputElement>) => void
}) {
  return (
    <div className='flex flex-col gap-2'>
      <Label htmlFor='invite-email'>Email</Label>
      <Input
        autoComplete='email'
        id='invite-email'
        onChange={onEmailChange}
        placeholder='teacher@example.com'
        required
        type='email'
        value={email}
      />
    </div>
  )
}

function InviteRoleField({
  onRoleChange,
  role,
}: {
  onRoleChange: (value: InviteIssueRole) => void
  role: InviteIssueRole
}) {
  return (
    <div className='flex flex-col gap-2'>
      <Label htmlFor='invite-role'>Role</Label>
      <Select onValueChange={onRoleChange} value={role}>
        <SelectTrigger id='invite-role'>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value='teacher'>Teacher</SelectItem>
          <SelectItem value='parent'>Parent</SelectItem>
          <SelectItem value='admin'>Admin</SelectItem>
        </SelectContent>
      </Select>
    </div>
  )
}

function LatestInvite({
  copyFeedback,
  invite,
  onCopyLink,
}: {
  copyFeedback: string
  invite: InviteRecord
  onCopyLink: () => void
}) {
  return (
    <div className='latest-invite'>
      <strong>{invite.email}</strong>
      <p>{getInviteDeliveryDescription(invite.delivery_status)}</p>
      <Label htmlFor='invite-link'>Activation link</Label>
      <Input id='invite-link' readOnly value={resolveInviteLink(invite)} />
      <div className='mt-6 flex flex-wrap gap-3'>
        <Button onClick={onCopyLink} type='button' variant='outline'>
          <CopyIcon data-icon='inline-start' />
          Copy link
        </Button>
        {copyFeedback ? <p>{copyFeedback}</p> : null}
      </div>
      {invite.delivery_error ? (
        <p className='text-muted-foreground'>{invite.delivery_error}</p>
      ) : null}
    </div>
  )
}

function getInviteDeliveryDescription(
  deliveryStatus: InviteRecord['delivery_status'],
): string {
  if (deliveryStatus === 'sent') {
    return 'Invite email sent.'
  }

  if (deliveryStatus === 'failed') {
    return 'Invite delivery failed.'
  }

  return 'Copy the activation link.'
}

function buildInviteLink(token: string): string {
  if (typeof window === 'undefined') {
    return `/activate?token=${encodeURIComponent(token)}`
  }

  return `${window.location.origin}/activate?token=${encodeURIComponent(token)}`
}
