import { useCallback, useEffect, useMemo, useState } from 'react'

import type { GroupDetail, GroupRecord } from '@/lib/group-types'
import { AssignedTopicsPanel } from '@/components/assigned-topics-panel'
import { ClassCreateForm } from '@/components/class-create-form'
import { ClassInvitePanel } from '@/components/class-invite-panel'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { DataTable } from '@/components/data-table'
import { LoadingStatus, StatePanel } from '@/components/state-panel'
import { StatItem } from '@/components/stat-item'
import { SurfaceSection } from '@/components/surface-section'
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

function getClassSummary(groups: Array<GroupRecord>): {
  activeStudents: number
  averageMastery: string
  classCount: number
  totalMembers: number
} {
  const totalMembers = groups.reduce(
    (sum, group) => sum + group.member_count,
    0,
  )

  return {
    activeStudents: totalMembers,
    averageMastery: '--',
    classCount: groups.length,
    totalMembers,
  }
}

export function ClassesContent({
  onCreated,
  onSelectClass,
  selectedClassID,
  state,
}: {
  onCreated: () => void
  onSelectClass: (value: string) => void
  selectedClassID: string
  state: ClassesState
}) {
  if (isInitialClassLoad(state)) {
    return <LoadingStatus>Loading classes...</LoadingStatus>
  }

  if (state.status === 'error') {
    return (
      <StatePanel role='alert' title='Classes unavailable'>
        {state.error}
      </StatePanel>
    )
  }

  const selectedGroup = findSelectedGroup(state.groups, selectedClassID)
  const summary = getClassSummary(state.groups)

  return (
    <div className='mt-6 flex flex-col gap-5'>
      <ClassCreateForm onCreated={onCreated} />
      <ClassInvitePanel />
      <ClassSummaryStats summary={summary} />
      <ClassPicker
        groups={state.groups}
        onSelectClass={onSelectClass}
        selectedClassID={selectedClassID}
      />
      <SelectedClassPanel group={selectedGroup} />
    </div>
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
      <StatItem label='Members' value={String(summary.totalMembers)} />
      <StatItem
        label='Active learners'
        value={String(summary.activeStudents)}
      />
      <StatItem label='Avg mastery' value={summary.averageMastery} />
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
  if (groups.length === 0) {
    return (
      <StatePanel title='No classes yet'>
        Create a class to generate a join code.
      </StatePanel>
    )
  }

  return (
    <SurfaceSection
      description='Select a class to view its roster and join code.'
      title='Classes'
    >
      <div className='grid gap-3'>
        <div className='flex flex-col gap-2'>
          <Label>Class</Label>
          <Select onValueChange={onSelectClass} value={selectedClassID}>
            <SelectTrigger>
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
        <div className='grid gap-3'>
          {groups.map((group) => (
            <ClassListButton
              group={group}
              key={group.id}
              onSelectClass={onSelectClass}
            />
          ))}
        </div>
      </div>
    </SurfaceSection>
  )
}

function ClassListButton({
  group,
  onSelectClass,
}: {
  group: GroupRecord
  onSelectClass: (value: string) => void
}) {
  const selectClass = useCallback(() => {
    onSelectClass(group.id)
  }, [group.id, onSelectClass])

  return (
    <button
      className='flex w-full flex-col gap-1.5 rounded-lg border border-border bg-card p-3.5 text-left text-inherit'
      onClick={selectClass}
      type='button'
    >
      <strong>{group.name}</strong>
      <span className='text-muted-foreground'>{readGroupSummary(group)}</span>
    </button>
  )
}

function SelectedClassPanel({ group }: { group: GroupRecord | null }) {
  const detail = useSelectedClassDetail(group)

  if (!group) {
    return null
  }

  return (
    <div className='mt-6 flex flex-col gap-5'>
      <SelectedClassSummary group={group} />
      <RosterTable detail={detail.record} error={detail.error} />
      <AssignedTopicsPanel />
    </div>
  )
}

function SelectedClassSummary({ group }: { group: GroupRecord }) {
  return (
    <div className='mt-6 rounded-lg border border-border bg-card p-4'>
      <p className='mb-2 text-xs font-bold tracking-[0.14em] text-muted-foreground uppercase'>
        Selected class
      </p>
      <h2 className='text-xl leading-tight font-semibold'>{group.name}</h2>
      <p>{getGroupDescription(group)}</p>
      <ClassMetadataTags group={group} />
      <strong>Join code: {group.join_code}</strong>
      <p className='text-muted-foreground'>
        Share this code with students. They join via /join {group.join_code} in
        the bot.
      </p>
    </div>
  )
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
          className='min-h-7 rounded-full bg-muted px-2.5 py-1 text-muted-foreground'
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

function useSelectedClassDetail(group: GroupRecord | null): {
  record: GroupDetail | null
  error: string
} {
  const [record, setRecord] = useState<GroupDetail | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!group) {
      setRecord(null)
      return
    }

    setError('')
    getGroupDetail(group.id)
      .then(setRecord)
      .catch((caught: unknown) => {
        setError(readClassDetailError(caught))
      })
  }, [group])

  return { record, error }
}

function readClassDetailError(caught: unknown): string {
  return caught instanceof Error ? caught.message : 'Class detail failed'
}

function RosterTable({
  detail,
  error,
}: {
  detail: GroupDetail | null
  error: string
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
      description='Class membership and mastery overview.'
      title='Member roster'
    >
      {renderRoster({ detail, error, rows })}
    </SurfaceSection>
  )
}

function renderRoster({
  detail,
  error,
  rows,
}: {
  detail: GroupDetail | null
  error: string
  rows: Array<{ id: string; cells: Array<{ key: string; value: string }> }>
}) {
  if (error) {
    return <RosterError error={error} />
  }

  if (!detail) {
    return <LoadingStatus>Loading roster...</LoadingStatus>
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

  return <DataTable columns={rosterColumns} rows={rows} />
}

function RosterError({ error }: { error: string }) {
  return (
    <StatePanel role='alert' title='Roster unavailable'>
      {error}
    </StatePanel>
  )
}

function EmptyRoster() {
  return (
    <StatePanel title='No members yet'>
      Share the join code to get started.
    </StatePanel>
  )
}
