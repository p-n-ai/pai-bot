import { useCallback, useState } from 'react'
import type { ChangeEvent, FormEvent } from 'react'

import type { InviteIssueRole, InviteRecord } from '@/lib/user-management-types'
import {
  copyInviteLink,
  resolveInviteLink,
} from '@/components/users/user-management-invite-form'
import { issueInvite, reissueInvite } from '@/lib/admin-api'

interface InviteWorkflowOptions {
  onInviteChanged?: () => void
}

export function useInviteWorkflow(options: InviteWorkflowOptions = {}) {
  const [copyFeedback, setCopyFeedback] = useState('')
  const [email, setEmail] = useState('')
  const [error, setError] = useState('')
  const [isSubmitting, setSubmitting] = useState(false)
  const [latestInvite, setLatestInvite] = useState<InviteRecord | null>(null)
  const [role, setRole] = useState<InviteIssueRole>('teacher')

  const updateEmail = useCallback((event: ChangeEvent<HTMLInputElement>) => {
    setEmail(event.target.value)
  }, [])

  const submitInvite = useCallback(
    (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault()
      submitInviteAction({
        action: () => issueInvite({ email: email.trim(), role }),
        fallbackError:
          'Unable to create the invite. Check the email and try again.',
        onInviteChanged: options.onInviteChanged,
        setCopyFeedback,
        setError,
        setLatestInvite,
        setSubmitting,
      })
    },
    [email, options.onInviteChanged, role],
  )

  const reissue = useCallback(
    (inviteID: string) => {
      submitInviteAction({
        action: () => reissueInvite(inviteID),
        fallbackError:
          'Unable to resend the invite. Check your connection and try again.',
        onInviteChanged: options.onInviteChanged,
        setCopyFeedback,
        setError,
        setLatestInvite,
        setSubmitting,
      })
    },
    [options.onInviteChanged],
  )

  const copyLatestInvite = useCallback(() => {
    if (!latestInvite) {
      return
    }

    copyInviteLink(resolveInviteLink(latestInvite))
      .then(() => {
        setCopyFeedback('Activation link copied.')
      })
      .catch(() => {
        setError(
          'Unable to copy the activation link. Copy it from the field and try again.',
        )
      })
  }, [latestInvite])

  return {
    copyFeedback,
    email,
    error,
    isSubmitting,
    latestInvite,
    role,
    handleCopyLatestInvite: copyLatestInvite,
    handleReissue: reissue,
    handleRoleChange: setRole,
    handleSubmitInvite: submitInvite,
    handleUpdateEmail: updateEmail,
  }
}

function submitInviteAction({
  action,
  fallbackError,
  onInviteChanged,
  setCopyFeedback,
  setError,
  setLatestInvite,
  setSubmitting,
}: {
  action: () => Promise<InviteRecord>
  fallbackError: string
  onInviteChanged: (() => void) | undefined
  setCopyFeedback: (value: string) => void
  setError: (value: string) => void
  setLatestInvite: (value: InviteRecord | null) => void
  setSubmitting: (value: boolean) => void
}) {
  setSubmitting(true)
  setError('')
  setLatestInvite(null)
  setCopyFeedback('')

  action()
    .then((invite) => {
      setLatestInvite(invite)
      onInviteChanged?.()
    })
    .catch((cause: unknown) => {
      setError(cause instanceof Error ? cause.message : fallbackError)
    })
    .finally(() => {
      setSubmitting(false)
    })
}
