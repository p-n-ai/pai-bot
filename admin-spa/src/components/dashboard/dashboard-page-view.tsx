import type { DashboardProgressResult } from '@/lib/dashboard-progress'
import type { GroupRecord } from '@/lib/group-types'
import type { LeaderboardState } from '@/components/dashboard/class-leaderboard'
import { AdminPageSection } from '@/components/shared/admin-page-section'
import { AdminSurface } from '@/components/shared/admin-surface'
import { DashboardReady } from '@/components/dashboard/dashboard-ready'
import { StatePanel } from '@/components/shared/state-panel'
import { Card, CardHeader } from '@/components/ui/card'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { Skeleton } from '@/components/ui/skeleton'

const TABLE_HEADER_SKELETON_KEYS = [
  'name',
  'status',
  'score',
  'activity',
  'action',
]
const TABLE_ROW_SKELETON_KEYS = ['first', 'second', 'third', 'fourth']

export type DashboardState =
  | {
      status: 'loading'
      progress: null
      error: null
    }
  | {
      status: 'ready'
      result: DashboardProgressResult
      error: null
    }
  | {
      status: 'error'
      progress: null
      error: string
    }

interface DashboardPageViewProps {
  classes?: ReadonlyArray<GroupRecord>
  classesError?: string
  classesLoading?: boolean
  leaderboardState?: LeaderboardState
  nudgeMessage: string
  onCloseStudent: () => void
  onNudge: (studentID: string, studentName: string) => void
  onRetryLeaderboard?: () => void
  onSelectClass?: (classID: string | undefined) => void
  onSelectStudent: (studentID: string) => void
  selectedStudentID: string | undefined
  selectedClassID?: string
  sendingStudentID: string
  state: DashboardState
}

export function DashboardPageView({
  classes = [],
  classesError = '',
  classesLoading = false,
  leaderboardState = { status: 'unavailable' },
  nudgeMessage,
  onCloseStudent,
  onNudge,
  onRetryLeaderboard = noop,
  onSelectClass = noop,
  onSelectStudent,
  selectedStudentID,
  selectedClassID,
  sendingStudentID,
  state,
}: DashboardPageViewProps) {
  return (
    <AdminPageSection
      description='Welcome back. Start with the learners who need you most today.'
      eyebrow='Dashboard'
      hideEyebrow
      title='Today'
    >
      <ClassSelector
        classes={classes}
        error={classesError}
        loading={classesLoading}
        onSelectClass={onSelectClass}
        selectedClassID={selectedClassID}
      />
      <DashboardContent
        leaderboardState={leaderboardState}
        nudgeMessage={nudgeMessage}
        onCloseStudent={onCloseStudent}
        onNudge={onNudge}
        onRetryLeaderboard={onRetryLeaderboard}
        onSelectStudent={onSelectStudent}
        selectedStudentID={selectedStudentID}
        sendingStudentID={sendingStudentID}
        state={state}
      />
    </AdminPageSection>
  )
}

function DashboardContent({
  leaderboardState,
  nudgeMessage,
  onCloseStudent,
  onNudge,
  onRetryLeaderboard,
  onSelectStudent,
  selectedStudentID,
  sendingStudentID,
  state,
}: Pick<
  DashboardPageViewProps,
  | 'leaderboardState'
  | 'nudgeMessage'
  | 'onCloseStudent'
  | 'onNudge'
  | 'onRetryLeaderboard'
  | 'onSelectStudent'
  | 'selectedStudentID'
  | 'sendingStudentID'
  | 'state'
>) {
  if (state.status === 'loading') {
    return <DashboardSkeleton />
  }

  if (state.status === 'error') {
    return (
      <StatePanel role='alert' title='Class data unavailable'>
        {state.error}
      </StatePanel>
    )
  }

  return (
    <DashboardReady
      nudgeMessage={nudgeMessage}
      onCloseStudent={onCloseStudent}
      onNudge={onNudge}
      onRetryLeaderboard={onRetryLeaderboard}
      onSelectStudent={onSelectStudent}
      progress={state.result.progress}
      leaderboardState={leaderboardState}
      selectedStudentID={selectedStudentID}
      sendingStudentID={sendingStudentID}
    />
  )
}

function ClassSelector({
  classes,
  error,
  loading,
  onSelectClass,
  selectedClassID,
}: {
  classes: ReadonlyArray<GroupRecord>
  error: string
  loading: boolean
  onSelectClass: (classID: string | undefined) => void
  selectedClassID: string | undefined
}) {
  return (
    <div className='mt-6 max-w-sm space-y-2'>
      <label className='text-sm font-medium' htmlFor='dashboard-class'>
        Class view
      </label>
      <NativeSelect
        className='w-full [&_select]:min-h-11'
        disabled={loading}
        id='dashboard-class'
        onChange={(event) => onSelectClass(event.target.value || undefined)}
        value={selectedClassID ?? ''}
      >
        <NativeSelectOption value=''>All learners</NativeSelectOption>
        {classes
          .filter((group) => !group.closed)
          .map((group) => (
            <NativeSelectOption key={group.id} value={group.id}>
              {group.name}
            </NativeSelectOption>
          ))}
      </NativeSelect>
      {error ? (
        <p className='text-sm text-destructive' role='alert'>
          {error}
        </p>
      ) : null}
    </div>
  )
}

function DashboardSkeleton() {
  return (
    <div
      className='mt-8 flex flex-col gap-5'
      data-testid='dashboard-skeleton'
      role='status'
    >
      <span className='sr-only'>Preparing class snapshot</span>
      <section className='grid gap-3 md:grid-cols-2 xl:grid-cols-5'>
        <DashboardStatSkeleton className='xl:col-span-2' />
        <DashboardStatSkeleton />
        <DashboardStatSkeleton />
        <DashboardStatSkeleton />
      </section>
      <AdminSurface className='shadow-none' contentClassName='p-4 sm:p-5'>
        <div className='flex flex-col gap-6'>
          <div className='flex flex-col gap-2'>
            <Skeleton className='h-6 w-44' />
            <Skeleton className='h-4 w-full max-w-md' />
          </div>
          <div className='min-w-[760px]'>
            <div className='grid grid-cols-[1.4fr_repeat(3,1fr)_0.8fr] gap-2 px-3 py-2'>
              {TABLE_HEADER_SKELETON_KEYS.map((key) => (
                <Skeleton className='h-4' key={key} />
              ))}
            </div>
            <div className='flex flex-col gap-2'>
              {TABLE_ROW_SKELETON_KEYS.map((key) => (
                <div
                  className='grid grid-cols-[1.4fr_repeat(3,1fr)_0.8fr] gap-2 rounded-lg bg-muted/40 p-3'
                  key={key}
                >
                  <Skeleton className='h-7 w-32' />
                  <Skeleton className='h-7 w-16 rounded-full' />
                  <Skeleton className='h-7 w-16 rounded-full' />
                  <Skeleton className='h-7 w-16 rounded-full' />
                  <Skeleton className='h-7 w-20 rounded-md' />
                </div>
              ))}
            </div>
          </div>
        </div>
      </AdminSurface>
    </div>
  )
}

function DashboardStatSkeleton({ className }: { className?: string }) {
  return (
    <Card className={className}>
      <CardHeader className='gap-3'>
        <Skeleton className='h-3 w-24' />
        <Skeleton className='h-9 w-16' />
        <Skeleton className='h-4 w-36' />
      </CardHeader>
    </Card>
  )
}

function noop() {}
