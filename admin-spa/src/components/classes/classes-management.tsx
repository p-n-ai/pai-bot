import { useCallback, useEffect, useMemo, useState } from 'react'

import type { GroupDetail, GroupRecord } from '@/lib/group-types'
import {
  CheckIcon,
  CopyIcon,
  MailPlusIcon,
  PlusIcon,
  UsersIcon,
} from '@/components/ui/pandai-icons'
import { AssignedTopicsPanel } from '@/components/classes/assigned-topics-panel'
import { ClassCreateForm } from '@/components/classes/class-create-form'
import { ClassInvitePanel } from '@/components/classes/class-invite-panel'
import { ClassResourcesPanel } from '@/components/classes/class-resources-panel'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { DataTable } from '@/components/shared/data-table'
import { LoadingStatus, StatePanel } from '@/components/shared/state-panel'
import { ContentSkeleton } from '@/components/shared/content-skeleton'
import { StatItem } from '@/components/shared/stat-item'
import { SurfaceSection } from '@/components/shared/surface-section'
import { getGroupDetail } from '@/lib/admin-api'

export type ClassesState =
  | { status: 'loading'; groups: Array<GroupRecord>; error: null }
  | { status: 'ready'; groups: Array<GroupRecord>; error: null }
  | { status: 'error'; groups: Array<GroupRecord>; error: string }

const rosterColumns = ['Learner', 'Role', 'Channel', 'Mastery']

function isInitialClassLoad(state: ClassesState): boolean {
  return state.status === 'loading' && state.groups.length === 0
}

function findSelectedGroup(
  groups: Array<GroupRecord>,
  selectedClassID: string,
): GroupRecord | null {
  if (groups.length === 0) {
    return null
  }

  return groups.find((group) => group.id === selectedClassID) ?? groups[0]
}

interface ClassSummary {
  classCount: number
  totalMembers: number
}

function getClassSummary(groups: Array<GroupRecord>): ClassSummary {
  const totalMembers = groups.reduce(
    (sum, group) => sum + group.member_count,
    0,
  )

  return {
    classCount: groups.length,
    totalMembers,
  }
}

export function ClassesContent({
  canInviteAdults,
  onCreated,
  onRetry,
  onSelectClass,
  selectedClassID,
  state,
}: {
  canInviteAdults: boolean
  onCreated: () => void
  onRetry: () => void
  onSelectClass: (value: string) => void
  selectedClassID: string
  state: ClassesState
}) {
  const [openAction, setOpenAction] = useState<'create' | 'invite' | null>(null)
  const handleCreated = useCallback(() => {
    setOpenAction(null)
    onCreated()
  }, [onCreated])

  if (isInitialClassLoad(state)) {
    return <ContentSkeleton label='Loading classes' variant='classes' />
  }

  if (state.status === 'error' && state.groups.length === 0) {
    return (
      <StatePanel role='alert' title='Classes unavailable'>
        <div className='grid gap-4'>
          <p>{state.error}</p>
          <Button className='min-h-11 w-fit' onClick={onRetry} type='button'>
            Try again
          </Button>
        </div>
      </StatePanel>
    )
  }

  const selectedGroup = findSelectedGroup(state.groups, selectedClassID)
  const summary = getClassSummary(state.groups)
  const openAvailableAction =
    openAction === 'create' && state.groups.length === 0
      ? null
      : openAction === 'invite' && !canInviteAdults
        ? null
        : openAction

  return (
    <div className='mt-6 flex flex-col gap-5'>
      {state.status === 'error' ? (
        <div
          className='flex flex-col gap-3 rounded-lg border border-destructive/30 bg-destructive/5 p-4 sm:flex-row sm:items-center sm:justify-between'
          role='alert'
        >
          <p className='text-sm'>
            Unable to refresh classes. The last loaded list is still visible.
          </p>
          <Button
            className='min-h-11 w-full sm:w-auto'
            onClick={onRetry}
            type='button'
            variant='outline'
          >
            Try again
          </Button>
        </div>
      ) : null}
      {state.groups.length === 0 ? (
        <>
          <EmptyClasses onCreated={handleCreated} />
          {canInviteAdults ? (
            <ClassActions
              canCreateClass={false}
              canInviteAdults
              onCreated={handleCreated}
              onOpenAction={setOpenAction}
              openAction={openAvailableAction}
            />
          ) : null}
        </>
      ) : (
        <>
          <ClassActions
            canCreateClass
            canInviteAdults={canInviteAdults}
            onCreated={handleCreated}
            onOpenAction={setOpenAction}
            openAction={openAvailableAction}
          />
          <ClassSummaryStats summary={summary} />
          <div className='grid items-start gap-5 lg:grid-cols-[minmax(220px,0.34fr)_minmax(0,1fr)]'>
            <ClassPicker
              groups={state.groups}
              onSelectClass={onSelectClass}
              selectedClassID={selectedGroup?.id ?? ''}
            />
            <SelectedClassPanel group={selectedGroup} groups={state.groups} />
          </div>
        </>
      )}
    </div>
  )
}

function ClassActions({
  canCreateClass,
  canInviteAdults,
  onCreated,
  onOpenAction,
  openAction,
}: {
  canCreateClass: boolean
  canInviteAdults: boolean
  onCreated: () => void
  onOpenAction: (action: 'create' | 'invite' | null) => void
  openAction: 'create' | 'invite' | null
}) {
  const handleCreateAction = useCallback(() => {
    onOpenAction(openAction === 'create' ? null : 'create')
  }, [onOpenAction, openAction])
  const handleInviteAction = useCallback(() => {
    onOpenAction(openAction === 'invite' ? null : 'invite')
  }, [onOpenAction, openAction])

  return (
    <section className='grid gap-4' aria-label='Class actions'>
      <div className='flex flex-col gap-3 sm:flex-row'>
        {canCreateClass ? (
          <Button
            aria-controls='class-action-panel'
            aria-expanded={openAction === 'create'}
            className='min-h-11 w-full sm:w-auto'
            onClick={handleCreateAction}
            type='button'
          >
            <PlusIcon data-icon='inline-start' />
            Create class
          </Button>
        ) : null}
        {canInviteAdults ? (
          <Button
            aria-controls='class-action-panel'
            aria-expanded={openAction === 'invite'}
            className='min-h-11 w-full sm:w-auto'
            onClick={handleInviteAction}
            type='button'
            variant='outline'
          >
            <MailPlusIcon data-icon='inline-start' />
            Invite staff or parent
          </Button>
        ) : null}
      </div>
      {openAction ? (
        <div
          aria-labelledby='class-action-title'
          className='rounded-xl border border-border bg-card p-4 sm:p-6'
          id='class-action-panel'
        >
          <div className='mb-5'>
            <h2
              className='text-xl leading-tight font-semibold text-balance'
              id='class-action-title'
            >
              {openAction === 'create'
                ? 'Create a class'
                : 'Invite staff or parent'}
            </h2>
            <p className='mt-1 text-sm leading-6 text-muted-foreground'>
              {openAction === 'create'
                ? 'Add the class now. You can share its join code with students next.'
                : 'Invite a teacher, parent, or administrator. Students join with the class code.'}
            </p>
          </div>
          {openAction === 'create' ? (
            <ClassCreateForm onCreated={onCreated} />
          ) : (
            <ClassInvitePanel />
          )}
        </div>
      ) : null}
    </section>
  )
}

function EmptyClasses({ onCreated }: { onCreated: () => void }) {
  return (
    <section className='rounded-xl border border-dashed border-border bg-card p-5 sm:p-8'>
      <div className='mx-auto grid max-w-2xl gap-6'>
        <div>
          <UsersIcon className='mb-3 size-6 text-muted-foreground' />
          <h2 className='text-xl leading-tight font-semibold text-balance'>
            Create your first class
          </h2>
          <p className='mt-2 text-sm leading-6 text-muted-foreground'>
            Add a class to get a student join code and start building your
            roster.
          </p>
        </div>
        <ClassCreateForm onCreated={onCreated} />
      </div>
    </section>
  )
}

function ClassSummaryStats({
  summary,
}: {
  summary: ReturnType<typeof getClassSummary>
}) {
  return (
    <section
      className='grid grid-cols-[repeat(auto-fit,minmax(130px,1fr))] gap-3'
      aria-label='Class summary'
    >
      <StatItem label='Classes' value={String(summary.classCount)} />
      <StatItem label='Students' value={String(summary.totalMembers)} />
    </section>
  )
}

function ClassPicker({
  groups,
  onSelectClass,
  selectedClassID,
}: {
  groups: Array<GroupRecord>
  onSelectClass: (value: string) => void
  selectedClassID: string
}) {
  return (
    <section
      className='grid gap-4 rounded-xl border border-border bg-card p-4 lg:sticky lg:top-4'
      aria-labelledby='class-list-title'
    >
      <div>
        <h2
          className='text-lg leading-tight font-semibold text-balance'
          id='class-list-title'
        >
          School classes
        </h2>
        <p className='mt-1 text-sm text-muted-foreground'>
          Choose a class to manage its roster.
        </p>
      </div>
      <div className='flex flex-col gap-2 lg:hidden'>
        <Label htmlFor='class-picker'>Class</Label>
        <Select onValueChange={onSelectClass} value={selectedClassID}>
          <SelectTrigger className='min-h-11 w-full' id='class-picker'>
            <SelectValue placeholder='Select a class' />
          </SelectTrigger>
          <SelectContent>
            {groups.map((group) => (
              <SelectItem key={group.id} value={group.id}>
                {group.name}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className='hidden gap-2 lg:grid'>
        {groups.map((group) => (
          <ClassListButton
            group={group}
            isSelected={group.id === selectedClassID}
            key={group.id}
            onSelectClass={onSelectClass}
          />
        ))}
      </div>
    </section>
  )
}

function ClassListButton({
  group,
  isSelected,
  onSelectClass,
}: {
  group: GroupRecord
  isSelected: boolean
  onSelectClass: (value: string) => void
}) {
  const selectClass = useCallback(() => {
    onSelectClass(group.id)
  }, [group.id, onSelectClass])

  return (
    <button
      aria-current={isSelected ? 'true' : undefined}
      className='flex min-h-14 w-full flex-col gap-1.5 rounded-lg border border-transparent p-3.5 text-left text-inherit transition-[background-color,border-color,color] duration-150 ease-[cubic-bezier(0.23,1,0.32,1)] hover:bg-muted focus-visible:border-ring focus-visible:ring-3 focus-visible:ring-ring/50 focus-visible:outline-none aria-current:border-primary/30 aria-current:bg-primary/5'
      onClick={selectClass}
      type='button'
    >
      <span className='flex w-full min-w-0 items-center justify-between gap-2'>
        <strong className='truncate'>{group.name}</strong>
        {isSelected ? (
          <CheckIcon
            className='size-4 shrink-0 text-primary'
            aria-hidden='true'
          />
        ) : null}
      </span>
      <span className='text-sm text-muted-foreground tabular-nums'>
        {readGroupSummary(group)}
      </span>
    </button>
  )
}

function SelectedClassPanel({
  group,
  groups,
}: {
  group: GroupRecord | null
  groups: Array<GroupRecord>
}) {
  const detail = useSelectedClassDetail(group)

  if (!group) {
    return null
  }

  return (
    <div className='flex min-w-0 flex-col gap-5'>
      <SelectedClassSummary
        group={group}
        key={`${group.id}:${group.join_code}`}
      />
      <RosterTable
        detail={detail.record}
        error={detail.error}
        onRetry={detail.handleReload}
      />
      <ClassResourcesPanel
        groups={groups}
        key={group.id}
        selectedClass={group}
      />
      <AssignedTopicsPanel />
    </div>
  )
}

function SelectedClassSummary({ group }: { group: GroupRecord }) {
  const [copyStatus, setCopyStatus] = useState('')
  const copyJoinCode = useCallback(() => {
    setCopyStatus('')
    copyTextToClipboard(group.join_code)
      .then(() => setCopyStatus('Join code copied.'))
      .catch(() =>
        setCopyStatus(
          'Unable to copy the join code. Select and copy the code manually.',
        ),
      )
  }, [group.join_code])

  return (
    <section
      className='grid gap-5 rounded-xl border border-border bg-card p-4 sm:p-6'
      aria-labelledby='selected-class-title'
    >
      <div className='min-w-0'>
        <p className='mb-2 text-xs font-bold tracking-[0.14em] text-muted-foreground uppercase'>
          Current class
        </p>
        <h2
          className='truncate text-xl leading-tight font-semibold text-balance sm:text-2xl'
          id='selected-class-title'
          title={group.name}
        >
          {group.name}
        </h2>
        <p className='mt-2 text-sm leading-6 text-muted-foreground'>
          {getGroupDescription(group)}
        </p>
        <ClassMetadataTags group={group} />
      </div>
      <div className='flex flex-col gap-4 rounded-lg bg-muted/50 p-4 sm:flex-row sm:items-center sm:justify-between'>
        <div>
          <p className='text-sm font-medium text-muted-foreground'>
            Student join code
          </p>
          <p
            className='mt-1 text-2xl font-semibold tracking-[0.08em] tabular-nums'
            translate='no'
          >
            {group.join_code}
          </p>
          <p className='mt-1 text-sm text-muted-foreground'>
            Students send <span translate='no'>/join {group.join_code}</span> to
            P&amp;AI Bot.
          </p>
        </div>
        <div className='flex flex-col items-stretch gap-2 sm:items-end'>
          <Button
            className='min-h-11 w-full sm:w-auto'
            onClick={copyJoinCode}
            type='button'
            variant='outline'
          >
            <CopyIcon data-icon='inline-start' />
            Copy join code
          </Button>
          {copyStatus ? (
            <span
              className='text-sm text-muted-foreground'
              role={copyStatus.startsWith('Unable') ? 'alert' : 'status'}
            >
              {copyStatus}
            </span>
          ) : null}
        </div>
      </div>
    </section>
  )
}

async function copyTextToClipboard(value: string): Promise<void> {
  const clipboard = Reflect.get(navigator, 'clipboard') as Clipboard | undefined

  if (!clipboard) {
    throw new Error('Clipboard API unavailable')
  }

  await clipboard.writeText(value)
}

function ClassMetadataTags({ group }: { group: GroupRecord }) {
  const tags = [group.subject, group.syllabus, group.cadence].filter(
    (tag): tag is string => Boolean(tag),
  )

  if (tags.length === 0) {
    return null
  }

  return (
    <div className='flex flex-wrap gap-2'>
      {tags.map((tag) => (
        <span
          className='inline-flex h-5 items-center rounded-full bg-secondary px-2 text-xs font-medium text-secondary-foreground'
          key={tag}
        >
          {tag}
        </span>
      ))}
    </div>
  )
}

function getGroupDescription(group: GroupRecord): string {
  return group.description ?? readGroupSummary(group)
}

function readGroupSummary(group: GroupRecord): string {
  const learnerLabel = group.member_count === 1 ? 'learner' : 'learners'

  return `${group.syllabus ?? 'Class'}, ${group.member_count} ${learnerLabel}`
}

interface SelectedClassDetail {
  record: GroupDetail | null
  error: string
  handleReload: () => void
}

function useSelectedClassDetail(
  group: GroupRecord | null,
): SelectedClassDetail {
  const [record, setRecord] = useState<GroupDetail | null>(null)
  const [error, setError] = useState('')
  const [reloadCount, setReloadCount] = useState(0)
  const handleReload = useCallback(() => {
    setReloadCount((count) => count + 1)
  }, [])

  useEffect(() => {
    if (!group) {
      setRecord(null)
      return
    }

    let cancelled = false
    setRecord(null)
    setError('')
    getGroupDetail(group.id)
      .then((detail) => {
        if (!cancelled) {
          setRecord(detail)
        }
      })
      .catch((cause: unknown) => {
        if (!cancelled) {
          setError(readClassDetailError(cause))
        }
      })

    return () => {
      cancelled = true
    }
  }, [group, reloadCount])

  return { record, error, handleReload }
}

function readClassDetailError(cause: unknown): string {
  return cause instanceof Error
    ? cause.message
    : 'Unable to load this class roster.'
}

function RosterTable({
  detail,
  error,
  onRetry,
}: {
  detail: GroupDetail | null
  error: string
  onRetry: () => void
}) {
  const rows = useMemo(
    () =>
      detail?.members.map((member) => ({
        id: member.id,
        cells: [
          { key: 'name', value: member.name },
          { key: 'role', value: member.role },
          { key: 'channel', value: member.channel },
          { key: 'mastery', value: `${Math.round(member.mastery * 100)}%` },
        ],
      })) ?? [],
    [detail],
  )

  return (
    <SurfaceSection
      description='Students who joined this class and their current mastery.'
      title='Student roster'
    >
      {renderRoster({ detail, error, onRetry, rows })}
    </SurfaceSection>
  )
}

function renderRoster({
  detail,
  error,
  onRetry,
  rows,
}: {
  detail: GroupDetail | null
  error: string
  onRetry: () => void
  rows: Array<{ id: string; cells: Array<{ key: string; value: string }> }>
}) {
  if (error) {
    return <RosterError error={error} onRetry={onRetry} />
  }

  if (!detail) {
    return <LoadingStatus>Loading roster…</LoadingStatus>
  }

  return <RosterReady detail={detail} rows={rows} />
}

function RosterReady({
  detail,
  rows,
}: {
  detail: GroupDetail
  rows: Array<{ id: string; cells: Array<{ key: string; value: string }> }>
}) {
  if (detail.members.length === 0) {
    return <EmptyRoster />
  }

  return (
    <DataTable
      className='[&_td:last-child]:tabular-nums'
      columns={rosterColumns}
      rows={rows}
    />
  )
}

function RosterError({
  error,
  onRetry,
}: {
  error: string
  onRetry: () => void
}) {
  return (
    <StatePanel role='alert' title='Roster unavailable'>
      <div className='grid gap-4'>
        <p>{error} Check your connection and try again.</p>
        <Button
          className='min-h-11 w-fit'
          onClick={onRetry}
          type='button'
          variant='outline'
        >
          Reload roster
        </Button>
      </div>
    </StatePanel>
  )
}

function EmptyRoster() {
  return (
    <StatePanel title='No students yet'>
      Share the join code above. Students will appear here after they join.
    </StatePanel>
  )
}
