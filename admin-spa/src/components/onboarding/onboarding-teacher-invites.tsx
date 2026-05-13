import { CopyIcon, MailPlusIcon } from 'lucide-react'
import { useCallback, useMemo, useState, useTransition } from 'react'
import type { ChangeEvent, ReactNode } from 'react'

import type { InviteRecord } from '@/lib/user-management-types'
import { issueInvite } from '@/lib/admin-api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

interface InviteOutcome {
  email: string
  error: string
  invite: InviteRecord | null
}

export function OnboardingTeacherInvites() {
  const [teacherEmails, setTeacherEmails] = useState('')
  const [inviteError, setInviteError] = useState('')
  const [copyFeedback, setCopyFeedback] = useState('')
  const [inviteOutcomes, setInviteOutcomes] = useState<Array<InviteOutcome>>([])
  const [isPending, startTransition] = useTransition()
  const inviteCounts = useMemo(
    () => getInviteCounts(inviteOutcomes),
    [inviteOutcomes],
  )

  const handleTeacherEmailsChange = useCallback(
    (event: ChangeEvent<HTMLTextAreaElement>) => {
      setTeacherEmails(event.target.value)
    },
    [],
  )
  const handleInviteSubmit = useCallback(() => {
    const emails = parseTeacherEmails(teacherEmails)
    if (emails.length === 0) {
      setInviteError('Add at least one teacher email.')
      return
    }

    setInviteError('')
    setCopyFeedback('')
    startTransition(() => {
      issueTeacherInvites(emails)
        .then((outcomes) => {
          setInviteOutcomes(outcomes)
          if (outcomes.some((item) => item.invite)) {
            setTeacherEmails('')
          }
        })
        .catch(() => {
          setInviteError('Invite issuance failed')
        })
    })
  }, [teacherEmails])

  return (
    <TeacherInviteSection>
      <TeacherEmailField
        onTeacherEmailsChange={handleTeacherEmailsChange}
        teacherEmails={teacherEmails}
      />
      <SendInvitesButton
        isPending={isPending}
        onInviteSubmit={handleInviteSubmit}
      />
      <InviteError message={inviteError} />
      {inviteOutcomes.length > 0 ? (
        <InviteOutcomeList
          copyFeedback={copyFeedback}
          inviteCounts={inviteCounts}
          inviteOutcomes={inviteOutcomes}
          onCopyFeedbackChange={setCopyFeedback}
          onInviteErrorChange={setInviteError}
        />
      ) : null}
    </TeacherInviteSection>
  )
}

function TeacherInviteSection({ children }: { children: ReactNode }) {
  return (
    <section
      className='mt-5 grid gap-3.5 border-t border-border pt-5'
      aria-labelledby='teacher-invite-title'
    >
      <p id='teacher-invite-title'>Teachers</p>
      {children}
    </section>
  )
}

function TeacherEmailField({
  onTeacherEmailsChange,
  teacherEmails,
}: {
  onTeacherEmailsChange: (event: ChangeEvent<HTMLTextAreaElement>) => void
  teacherEmails: string
}) {
  return (
    <div className='flex flex-col gap-2'>
      <label htmlFor='onboarding-teacher-emails'>Teacher emails</label>
      <textarea
        autoComplete='off'
        className='min-h-24 rounded-md border border-input bg-background px-3 py-2 text-sm shadow-xs transition-[border-color,box-shadow] outline-none placeholder:text-muted-foreground focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50'
        id='onboarding-teacher-emails'
        name='teacher-emails'
        onChange={onTeacherEmailsChange}
        placeholder={'cikgu.aminah@school.my\ncikgu.rizal@school.my'}
        rows={5}
        value={teacherEmails}
      />
    </div>
  )
}

function SendInvitesButton({
  isPending,
  onInviteSubmit,
}: {
  isPending: boolean
  onInviteSubmit: () => void
}) {
  const label = isPending ? 'Sending invites…' : 'Send invites'

  return (
    <Button disabled={isPending} onClick={onInviteSubmit} type='button'>
      <MailPlusIcon data-icon='inline-start' />
      {label}
    </Button>
  )
}

function InviteError({ message }: { message: string }) {
  if (!message) {
    return null
  }

  return (
    <div className='text-muted-foreground' role='alert'>
      <strong>We could not send the invite</strong>
      <p>{message}</p>
    </div>
  )
}

function InviteOutcomeList({
  copyFeedback,
  inviteCounts,
  inviteOutcomes,
  onCopyFeedbackChange,
  onInviteErrorChange,
}: {
  copyFeedback: string
  inviteCounts: { failed: number; sent: number; total: number }
  inviteOutcomes: Array<InviteOutcome>
  onCopyFeedbackChange: (message: string) => void
  onInviteErrorChange: (message: string) => void
}) {
  return (
    <div className='grid gap-3.5'>
      <p className='text-muted-foreground'>
        {inviteCounts.sent} sent, {inviteCounts.failed} needing follow-up,{' '}
        {inviteCounts.total} processed.
      </p>
      {inviteOutcomes.map((item) => (
        <InviteOutcomeRow
          item={item}
          key={item.email}
          onCopyFeedbackChange={onCopyFeedbackChange}
          onInviteErrorChange={onInviteErrorChange}
        />
      ))}
      {copyFeedback ? (
        <p className='text-muted-foreground'>{copyFeedback}</p>
      ) : null}
    </div>
  )
}

function InviteOutcomeRow({
  item,
  onCopyFeedbackChange,
  onInviteErrorChange,
}: {
  item: InviteOutcome
  onCopyFeedbackChange: (message: string) => void
  onInviteErrorChange: (message: string) => void
}) {
  return (
    <article className='grid gap-3.5 rounded-lg border border-border p-3.5'>
      <InviteOutcomeSummary item={item} />
      {item.invite ? (
        <InviteLinkActions
          email={item.email}
          invite={item.invite}
          onCopyFeedbackChange={onCopyFeedbackChange}
          onInviteErrorChange={onInviteErrorChange}
        />
      ) : null}
    </article>
  )
}

function InviteOutcomeSummary({ item }: { item: InviteOutcome }) {
  return (
    <div>
      <strong>{item.email}</strong>
      <p className='mt-1 text-muted-foreground'>
        {getInviteOutcomeMessage(item)}
      </p>
      <InviteDeliveryError invite={item.invite} />
    </div>
  )
}

function InviteDeliveryError({ invite }: { invite: InviteRecord | null }) {
  return invite?.delivery_error ? (
    <p className='text-muted-foreground'>{invite.delivery_error}</p>
  ) : null
}

function InviteLinkActions({
  email,
  invite,
  onCopyFeedbackChange,
  onInviteErrorChange,
}: {
  email: string
  invite: InviteRecord
  onCopyFeedbackChange: (message: string) => void
  onInviteErrorChange: (message: string) => void
}) {
  const inviteLink = resolveInviteLink(invite)
  const handleCopyInviteLink = useCallback(() => {
    navigator.clipboard
      .writeText(inviteLink)
      .then(() => {
        onCopyFeedbackChange(`Copied link for ${email}`)
      })
      .catch(() => {
        onInviteErrorChange('Could not copy the activation link.')
      })
  }, [email, inviteLink, onCopyFeedbackChange, onInviteErrorChange])

  return (
    <>
      <Input
        aria-label={`Activation link for ${email}`}
        readOnly
        value={inviteLink}
      />
      <Button onClick={handleCopyInviteLink} type='button' variant='outline'>
        <CopyIcon data-icon='inline-start' />
        Copy invite link
      </Button>
    </>
  )
}

function parseTeacherEmails(raw: string): Array<string> {
  return Array.from(
    new Set(
      raw
        .split(/[\n,;]+/)
        .map((value) => value.trim().toLowerCase())
        .filter(Boolean),
    ),
  )
}

async function issueTeacherInvites(
  emails: Array<string>,
): Promise<Array<InviteOutcome>> {
  return Promise.all(emails.map(issueTeacherInvite))
}

async function issueTeacherInvite(email: string): Promise<InviteOutcome> {
  try {
    const invite = await issueInvite({ email, role: 'teacher' })
    return { email, error: '', invite }
  } catch (error) {
    return {
      email,
      error: error instanceof Error ? error.message : 'Invite issuance failed',
      invite: null,
    }
  }
}

function getInviteCounts(inviteOutcomes: Array<InviteOutcome>): {
  failed: number
  sent: number
  total: number
} {
  return {
    failed: inviteOutcomes.filter(
      (item) => item.error || item.invite?.delivery_status === 'failed',
    ).length,
    sent: inviteOutcomes.filter(
      (item) => item.invite?.delivery_status === 'sent',
    ).length,
    total: inviteOutcomes.length,
  }
}

function getInviteOutcomeMessage(item: InviteOutcome): string {
  if (item.error) {
    return item.error
  }

  return getDeliveryStatusMessage(item.invite?.delivery_status)
}

function getDeliveryStatusMessage(
  status: InviteRecord['delivery_status'],
): string {
  const messages = {
    failed: 'We could not send the email, but the teacher invite is ready.',
    pending: 'Teacher invite is ready.',
    sent: 'Invite email sent.',
  }

  return messages[status ?? 'pending']
}

function resolveInviteLink(invite: InviteRecord): string {
  if (invite.activation_url?.trim()) {
    return invite.activation_url
  }

  if (typeof window === 'undefined') {
    return `/activate?token=${encodeURIComponent(invite.invite_token)}`
  }

  return `${window.location.origin}/activate?token=${encodeURIComponent(invite.invite_token)}`
}
