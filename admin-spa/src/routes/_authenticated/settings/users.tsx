import { createFileRoute } from '@tanstack/react-router'

import { AdminPageSection } from '@/components/shared/admin-page-section'
import { UserManagementPanel } from '@/components/users/user-management-panel'

export const Route = createFileRoute('/_authenticated/settings/users')({
  component: UsersSettingsRoute,
})

function UsersSettingsRoute() {
  return (
    <AdminPageSection
      description='Review school access and invite teachers, parents, or administrators.'
      title='Staff access'
    >
      <UserManagementPanel />
    </AdminPageSection>
  )
}
