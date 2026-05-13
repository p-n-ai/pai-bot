import { MailPlusIcon, RotateCcwIcon } from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import type { ChangeEvent, FormEvent, ReactNode } from 'react'

import type {
  InviteIssueRole,
  InviteRecord,
  UserManagementView,
} from '@/lib/user-management-types'
import { DataTable } from '@/components/data-table'
import { LoadState } from '@/components/load-state'
import { StatePanel } from '@/components/state-panel'
import { StatItem } from '@/components/stat-item'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from '@/components/ui/sheet'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { InviteForm } from '@/components/user-management-invite-form'
import { useInviteWorkflow } from '@/hooks/use-invite-workflow'
import { runWhenActive } from '@/lib/active-guard'
import { getUserManagement } from '@/lib/admin-api'

type UserManagementState =
  | { status: 'loading'; data: null; error: null }
  | { status: 'ready'; data: UserManagementView; error: null }
  | { status: 'error'; data: null; error: string }

const adminUserColumns = ['Name', 'Email', 'Role', 'Status', 'Created']
const allUserColumns = ['Name', 'Type', 'Contact', 'Scope', 'Access', 'Joined']
const inviteColumns = [
  'Email',
  'Role',
  'Invited by',
  'Delivery',
  'Expires',
  'Status',
  'Action',
]

export function UserManagementPanel() {
  const [state, setState] = useState<UserManagementState>({
    status: 'loading',
    data: null,
    error: null,
  })
  const [search, setSearch] = useState('')

  const reload = useCallback(() => {
    let active = true

    setState((current) =>
      current.status === 'ready'
        ? current
        : { status: 'loading', data: null, error: null },
    )

    const handleLoadError = () => {
      runWhenActive(active, () => {
        setState({
          status: 'error',
          data: null,
          error: "User management data isn't available right now.",
        })
      })
    }

    async function loadUsers() {
      const data = await getUserManagement()
      runWhenActive(active, () => {
        setState({ status: 'ready', data, error: null })
      })
    }

    loadUsers().catch(handleLoadError)

    return () => {
      active = false
    }
  }, [])

  useEffect(() => reload(), [reload])
  const invite = useInviteWorkflow({ onInviteChanged: reload })

  const updateSearch = useCallback((event: ChangeEvent<HTMLInputElement>) => {
    setSearch(event.target.value)
  }, [])

  if (state.status !== 'ready') {
    return (
      <LoadState
        error={state.error}
        errorTitle='User management unavailable'
        loadingTitle='Loading users and invites...'
        status={state.status}
      />
    )
  }

  return (
    <UserManagementReady
      data={state.data}
      inviteEmail={invite.email}
      inviteError={invite.error}
      inviteRole={invite.role}
      isSubmitting={invite.isSubmitting}
      latestInvite={invite.latestInvite}
      copyFeedback={invite.copyFeedback}
      onCopyLink={invite.handleCopyLatestInvite}
      onInviteEmailChange={invite.handleUpdateEmail}
      onInviteRoleChange={invite.handleRoleChange}
      onReissue={invite.handleReissue}
      onSearchChange={updateSearch}
      onSubmitInvite={invite.handleSubmitInvite}
      search={search}
    />
  )
}

// fallow-ignore-next-line complexity
function UserManagementReady({
  data,
  inviteEmail,
  inviteError,
  inviteRole,
  isSubmitting,
  latestInvite,
  onInviteEmailChange,
  onInviteRoleChange,
  onReissue,
  onSearchChange,
  onSubmitInvite,
  search,
  copyFeedback,
  onCopyLink,
}: {
  data: UserManagementView
  copyFeedback: string
  inviteEmail: string
  inviteError: string
  inviteRole: InviteIssueRole
  isSubmitting: boolean
  latestInvite: InviteRecord | null
  onCopyLink: () => void
  onInviteEmailChange: (event: ChangeEvent<HTMLInputElement>) => void
  onInviteRoleChange: (value: InviteIssueRole) => void
  onReissue: (inviteID: string) => void
  onSearchChange: (event: ChangeEvent<HTMLInputElement>) => void
  onSubmitInvite: (event: FormEvent<HTMLFormElement>) => void
  search: string
}) {
  const [tab, setTab] = useState('admin-users')
  const directoryUsers = useMemo(() => createDirectoryUsers(data), [data])
  const filteredAdminUsers = useMemo(
    () => filterRecords(data.active_users, search),
    [data.active_users, search],
  )
  const filteredInvites = useMemo(
    () => filterRecords(data.pending_invites, search),
    [data.pending_invites, search],
  )
  const filteredDirectoryUsers = useMemo(
    () => filterRecords(directoryUsers, search),
    [directoryUsers, search],
  )

  return (
    <div className='mt-6 flex flex-col gap-5'>
      <div className='grid grid-cols-[repeat(auto-fit,minmax(130px,1fr))] gap-3'>
        <StatItem label='Students' value={String(data.summary.students)} />
        <StatItem label='Teachers' value={String(data.summary.teachers)} />
        <StatItem label='Parents' value={String(data.summary.parents)} />
        <StatItem
          label='Pending invites'
          value={String(data.summary.pending_invites)}
        />
        <StatItem
          label='Total users'
          value={String(data.summary.total_users)}
        />
      </div>
      <Tabs className='flex flex-col gap-0' onValueChange={setTab} value={tab}>
        <UserDirectoryToolbar
          copyFeedback={copyFeedback}
          inviteEmail={inviteEmail}
          inviteError={inviteError}
          inviteRole={inviteRole}
          isSubmitting={isSubmitting}
          latestInvite={latestInvite}
          onCopyLink={onCopyLink}
          onInviteEmailChange={onInviteEmailChange}
          onInviteRoleChange={onInviteRoleChange}
          onSearchChange={onSearchChange}
          onSubmitInvite={onSubmitInvite}
          onTabChange={setTab}
          search={search}
        />

        <TabsContent className='m-0' value='admin-users'>
          <div className='user-management-layout'>
            <div className='mt-6 flex flex-col gap-5 md:col-span-2'>
              <div className='flex items-center justify-between gap-3'>
                <div>
                  <h2 className='m-0 text-sm font-semibold text-[#0c0a09]'>
                    Admin access
                  </h2>
                </div>
                <Badge variant='secondary'>
                  {filteredAdminUsers.length} active
                </Badge>
              </div>
              <TableFrame>
                <AdminUsersTable users={filteredAdminUsers} />
              </TableFrame>
              <InvitesTable
                invites={filteredInvites}
                isSubmitting={isSubmitting}
                onReissue={onReissue}
              />
            </div>
          </div>
        </TabsContent>

        <TabsContent className='m-0' value='all-users'>
          <TableFrame>
            <AllUsersTable users={filteredDirectoryUsers} />
          </TableFrame>
        </TabsContent>
      </Tabs>
    </div>
  )
}

function UserDirectoryToolbar({
  copyFeedback,
  inviteEmail,
  inviteError,
  inviteRole,
  isSubmitting,
  latestInvite,
  onCopyLink,
  onInviteEmailChange,
  onInviteRoleChange,
  onSearchChange,
  onSubmitInvite,
  onTabChange,
  search,
}: {
  copyFeedback: string
  inviteEmail: string
  inviteError: string
  inviteRole: InviteIssueRole
  isSubmitting: boolean
  latestInvite: InviteRecord | null
  onCopyLink: () => void
  onInviteEmailChange: (event: ChangeEvent<HTMLInputElement>) => void
  onInviteRoleChange: (value: InviteIssueRole) => void
  onSearchChange: (event: ChangeEvent<HTMLInputElement>) => void
  onSubmitInvite: (event: FormEvent<HTMLFormElement>) => void
  onTabChange: (value: string) => void
  search: string
}) {
  return (
    <div className='rounded-t-lg border border-border bg-card p-3'>
      <div className='flex flex-col gap-3 md:flex-row md:items-center md:justify-between'>
        <TabsList className='w-fit'>
          <TabsTrigger
            onClick={() => onTabChange('admin-users')}
            value='admin-users'
          >
            Admin users
          </TabsTrigger>
          <TabsTrigger
            onClick={() => onTabChange('all-users')}
            value='all-users'
          >
            All users
          </TabsTrigger>
        </TabsList>
        <div className='w-full md:max-w-md'>
          <Label className='sr-only' htmlFor='user-search'>
            Search users
          </Label>
          <Input
            id='user-search'
            onChange={onSearchChange}
            placeholder='Search users'
            value={search}
          />
        </div>
        <Sheet>
          <SheetTrigger asChild>
            <Button type='button'>
              <MailPlusIcon data-icon='inline-start' />
              Invite
            </Button>
          </SheetTrigger>
          <SheetContent className='w-full overflow-y-auto p-0 sm:max-w-md'>
            <SheetHeader className='border-b border-border p-5'>
              <SheetTitle>Invite user</SheetTitle>
              <SheetDescription>
                Send access to a teacher, parent, or admin.
              </SheetDescription>
            </SheetHeader>
            <div className='p-5'>
              <InviteForm
                email={inviteEmail}
                error={inviteError}
                isSubmitting={isSubmitting}
                latestInvite={latestInvite}
                copyFeedback={copyFeedback}
                onCopyLink={onCopyLink}
                onEmailChange={onInviteEmailChange}
                onRoleChange={onInviteRoleChange}
                onSubmit={onSubmitInvite}
                role={inviteRole}
              />
            </div>
          </SheetContent>
        </Sheet>
      </div>
    </div>
  )
}

function TableFrame({ children }: { children: ReactNode }) {
  return (
    <div className='overflow-hidden rounded-b-lg border-x border-b border-border bg-card'>
      <div className='max-h-[540px] overflow-auto'>{children}</div>
    </div>
  )
}

type DirectoryUser = {
  access: string
  contact: string
  id: string
  joined: string
  name: string
  scope: string
  type: string
}

function createDirectoryUsers(data: UserManagementView): Array<DirectoryUser> {
  return [
    ...data.active_users.map((user) => ({
      access: formatRole(user.role),
      contact: user.email || 'No email linked',
      id: `admin-${user.id}`,
      joined: user.created_at,
      name: user.name,
      scope: user.tenant_name || 'Platform',
      type: 'Admin user',
    })),
    ...data.students.map((student) => ({
      access: formatRole(student.channel),
      contact: student.external_id || student.id,
      id: `student-${student.id}`,
      joined: student.created_at,
      name: student.name,
      scope: student.form || 'Unassigned',
      type: 'Student',
    })),
  ].sort((a, b) => b.joined.localeCompare(a.joined))
}

function AllUsersTable({ users }: { users: Array<DirectoryUser> }) {
  const rows = useMemo(
    () =>
      users.map((user) => ({
        id: user.id,
        cells: [
          { key: 'name', value: user.name },
          { key: 'type', value: <Badge variant='outline'>{user.type}</Badge> },
          { key: 'contact', value: user.contact },
          { key: 'scope', value: user.scope },
          { key: 'access', value: <RoleBadge value={user.access} /> },
          { key: 'joined', value: user.joined.slice(0, 10) },
        ],
      })),
    [users],
  )

  if (users.length === 0) {
    return (
      <StatePanel title='No users match this search'>
        Try a different name, role, channel, form, or contact.
      </StatePanel>
    )
  }

  return (
    <DataTable
      className='min-w-[780px] [&_td]:h-11 [&_td]:px-3 [&_th]:sticky [&_th]:top-0 [&_th]:z-10 [&_th]:bg-card [&_th]:px-3'
      columns={allUserColumns}
      rows={rows}
    />
  )
}

function AdminUsersTable({
  users,
}: {
  users: UserManagementView['active_users']
}) {
  const rows = useMemo(
    () =>
      users.map((user) => ({
        id: user.id,
        cells: [
          { key: 'name', value: formatWithTenant(user.name, user.tenant_name) },
          { key: 'email', value: user.email || 'No email linked' },
          { key: 'role', value: <RoleBadge value={formatRole(user.role)} /> },
          { key: 'status', value: <StatusBadge value={user.status} /> },
          { key: 'created', value: user.created_at.slice(0, 10) },
        ],
      })),
    [users],
  )

  if (users.length === 0) {
    return (
      <StatePanel title='No active users match this search'>
        Try a different name, role, or email filter.
      </StatePanel>
    )
  }

  return (
    <DataTable
      className='min-w-[760px] [&_td]:h-11 [&_td]:px-3 [&_th]:sticky [&_th]:top-0 [&_th]:z-10 [&_th]:bg-card [&_th]:px-3'
      columns={adminUserColumns}
      rows={rows}
    />
  )
}

function InvitesTable({
  invites,
  isSubmitting,
  onReissue,
}: {
  invites: UserManagementView['pending_invites']
  isSubmitting: boolean
  onReissue: (inviteID: string) => void
}) {
  const rows = useMemo(
    () =>
      invites.map((invite) => ({
        id: invite.id,
        cells: [
          {
            key: 'email',
            value: formatWithTenant(invite.email, invite.tenant_name),
          },
          { key: 'role', value: formatRole(invite.role) },
          { key: 'invited-by', value: invite.invited_by || 'System' },
          {
            key: 'delivery',
            value: (
              <StatusBadge
                value={
                  invite.delivery_status
                    ? formatRole(invite.delivery_status)
                    : 'Pending'
                }
              />
            ),
          },
          { key: 'expires', value: invite.expires_at.slice(0, 10) },
          {
            key: 'status',
            value: <StatusBadge value={formatRole(invite.status)} />,
          },
          {
            key: 'action',
            value: (
              <ReissueInviteButton
                invite={invite}
                isSubmitting={isSubmitting}
                onReissue={onReissue}
              />
            ),
          },
        ],
      })),
    [invites, isSubmitting, onReissue],
  )

  if (invites.length === 0) {
    return (
      <StatePanel title='No pending invites match this search'>
        Open invite links appear here until they are accepted or expire.
      </StatePanel>
    )
  }

  return (
    <TableFrame>
      <DataTable
        className='min-w-[860px] [&_td]:h-11 [&_td]:px-3 [&_th]:sticky [&_th]:top-0 [&_th]:z-10 [&_th]:bg-card [&_th]:px-3'
        columns={inviteColumns}
        rows={rows}
      />
    </TableFrame>
  )
}

function ReissueInviteButton({
  invite,
  isSubmitting,
  onReissue,
}: {
  invite: UserManagementView['pending_invites'][number]
  isSubmitting: boolean
  onReissue: (inviteID: string) => void
}) {
  const reissue = useCallback(() => {
    onReissue(invite.id)
  }, [invite.id, onReissue])

  return (
    <Button
      disabled={isSubmitting}
      onClick={reissue}
      type='button'
      variant='outline'
    >
      <RotateCcwIcon data-icon='inline-start' />
      Resend {invite.email}
    </Button>
  )
}

function filterRecords<T extends object>(
  records: Array<T>,
  search: string,
): Array<T> {
  const query = search.trim().toLowerCase()

  if (!query) {
    return records
  }

  return records.filter((record) =>
    Object.values(record as Record<string, unknown>).some((value) =>
      String(value ?? '')
        .toLowerCase()
        .includes(query),
    ),
  )
}

function formatRole(value: string): string {
  return value.replaceAll('_', ' ')
}

function formatWithTenant(
  value: string,
  tenantName: string | undefined,
): string {
  return tenantName ? `${value} (${tenantName})` : value
}

function RoleBadge({ value }: { value: string }) {
  return <Badge variant='outline'>{value}</Badge>
}

function StatusBadge({ value }: { value: string }) {
  const normalized = value.toLowerCase()

  return (
    <Badge
      variant={
        normalized === 'active' || normalized === 'sent'
          ? 'secondary'
          : 'outline'
      }
    >
      {value}
    </Badge>
  )
}
