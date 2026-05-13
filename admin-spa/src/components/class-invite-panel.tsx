import { InviteForm } from './user-management-invite-form'
import { useInviteWorkflow } from '@/hooks/use-invite-workflow'

export function ClassInvitePanel() {
  const invite = useInviteWorkflow()

  return (
    <InviteForm
      copyFeedback={invite.copyFeedback}
      email={invite.email}
      error={invite.error}
      isSubmitting={invite.isSubmitting}
      latestInvite={invite.latestInvite}
      onCopyLink={invite.handleCopyLatestInvite}
      onEmailChange={invite.handleUpdateEmail}
      onRoleChange={invite.handleRoleChange}
      onSubmit={invite.handleSubmitInvite}
      role={invite.role}
    />
  )
}
