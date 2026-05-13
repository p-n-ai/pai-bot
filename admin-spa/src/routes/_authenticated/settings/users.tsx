import { createFileRoute } from '@tanstack/react-router'

import { AdminPageSection } from '@/components/shared/admin-page-section'
import { UserManagementPanel } from '@/components/users/user-management-panel'

export const Route = createFileRoute('/_authenticated/settings/users')({
  component: UsersSettingsRoute,
})

function UsersSettingsRoute() {
  return (
    <AdminPageSection
      description='Review active access, search the current workspace, and issue new teacher, parent, or admin invites.'
      eyebrow='Administration'
      hideEyebrow
      title='User and invite management'
    >
      <UserManagementPanel />
    </AdminPageSection>
  )
}
