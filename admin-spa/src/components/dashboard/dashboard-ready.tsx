import { useCallback, useMemo, useState } from 'react'
import type { ChangeEvent, ReactNode } from 'react'

import type {
  DashboardLearner,
  LearnerProgressFilter,
} from '@/lib/dashboard-learners'
import type { ClassProgress } from '@/lib/dashboard-types'
import type { LeaderboardState } from '@/components/dashboard/class-leaderboard'
import type { PandaiIconName } from '@/components/ui/pandai-icon'
import { ClassLeaderboard } from '@/components/dashboard/class-leaderboard'
import {
  AdminSurface,
  AdminSurfaceHeader,
} from '@/components/shared/admin-surface'
import { StatePanel } from '@/components/shared/state-panel'
import { StudentDetailPanel } from '@/components/dashboard/student-detail-panel'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { NativeSelect, NativeSelectOption } from '@/components/ui/native-select'
import { PandaiIcon } from '@/components/ui/pandai-icon'
import { Progress } from '@/components/ui/progress'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { getDashboardLearners } from '@/lib/dashboard-learners'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { getDashboardSummary } from '@/lib/dashboard-summary'
import { cn } from '@/lib/utils'

const masteryLegend = <MasteryLegend />

export function DashboardReady({
  nudgeMessage,
  onCloseStudent = noop,
  onNudge,
  onRetryLeaderboard,
  onSelectStudent = noop,
  progress,
  leaderboardState,
  selectedStudentID,
  sendingStudentID,
}: {
  nudgeMessage: string
  onCloseStudent?: () => void
  onNudge: (studentID: string, studentName: string) => void
  onRetryLeaderboard?: () => void
  onSelectStudent?: (studentID: string) => void
  progress: ClassProgress
  leaderboardState?: LeaderboardState
  selectedStudentID?: string
  sendingStudentID: string
}) {
  const summary = getDashboardSummary(progress)
  const selectedStudent = useMemo(
    () =>
      progress.students.find((student) => student.id === selectedStudentID) ??
      null,
    [progress.students, selectedStudentID],
  )

  return (
    <div className='mt-7 flex flex-col gap-6'>
      <DashboardStats progress={progress} />
      {nudgeMessage ? <NudgeFeedback message={nudgeMessage} /> : null}
      {leaderboardState ? (
        <ClassLeaderboard
          onRetry={onRetryLeaderboard ?? noop}
          onSelectStudent={onSelectStudent}
          state={leaderboardState}
        />
      ) : null}
      <DashboardHeatmap
        hasHeatmap={summary.hasHeatmap}
        onNudge={onNudge}
        onSelectStudent={onSelectStudent}
        progress={progress}
        sendingStudentID={sendingStudentID}
      />
      <StudentDetailSheet
        onOpenChange={onCloseStudent}
        studentID={selectedStudentID}
        studentName={selectedStudent?.name}
      />
    </div>
  )
}

function NudgeFeedback({ message }: { message: string }) {
  return (
    <div
      className='sticky top-16 z-20 flex min-h-11 items-center gap-2.5 rounded-xl border border-[var(--border-primary-default)] bg-[var(--surface-primary-default-subtle)] px-4 py-3 text-sm font-medium text-[var(--text-tertiary-default)]'
      role='status'
    >
      <PandaiIcon className='size-4 shrink-0' name='check-circle' />
      {message}
    </div>
  )
}

function DashboardStats({ progress }: { progress: ClassProgress }) {
  const summary = getDashboardSummary(progress)
  const classGrade = getMasteryGrade(summary.averageMastery)
  const learnerNote = getLearnerAttentionNote(summary.attentionCount)
  const weakestTopicLabel = summary.weakestTopic
    ? formatTopicLabel(summary.weakestTopic.topicID)
    : 'No topic data'
  const strongestTopicLabel = summary.strongestTopic
    ? formatTopicLabel(summary.strongestTopic.topicID)
    : 'No topic data'

  return (
    <section
      aria-label='Class overview'
      className='grid gap-4 md:grid-cols-2 xl:grid-cols-12'
    >
      <Card className='relative border-0 bg-[var(--admin-navigation)] py-0 text-[var(--admin-nav-text)] ring-0 md:col-span-2 xl:col-span-5'>
        <CardContent className='flex h-full min-h-48 flex-col justify-between p-5 sm:p-6'>
          <div className='flex items-start justify-between gap-4'>
            <div className='flex items-center gap-2.5'>
              <span className='flex size-9 items-center justify-center rounded-lg bg-white/10 text-[var(--admin-nav-muted)] ring-1 ring-white/20'>
                <PandaiIcon className='size-[18px]' name='users' />
              </span>
              <div>
                <p className='text-sm font-medium text-[var(--admin-nav-text)]'>
                  Learners
                </p>
                <p className='text-xs text-[var(--admin-nav-muted)]'>
                  Active class roster
                </p>
              </div>
            </div>
            <span className='inline-flex items-center gap-1.5 rounded-full bg-white/8 px-2.5 py-1 text-xs font-medium text-[var(--admin-nav-muted)] ring-1 ring-white/20'>
              <span className='size-1.5 rounded-full bg-[var(--admin-accent)]' />
              Current
            </span>
          </div>
          <div className='mt-8 flex items-end justify-between gap-5'>
            <div>
              <p className='text-4xl leading-none font-semibold tracking-[-0.025em] text-[var(--admin-nav-text)] tabular-nums'>
                {summary.studentCount}
              </p>
              <p
                className={cn(
                  'mt-3 flex items-center gap-1.5 text-sm font-medium',
                  summary.attentionCount > 0
                    ? 'text-[var(--status-danger-surface)]'
                    : 'text-[var(--admin-nav-muted)]',
                )}
              >
                {summary.attentionCount > 0 ? (
                  <PandaiIcon className='size-4' name='alert-triangle' />
                ) : (
                  <PandaiIcon className='size-4' name='check-circle' />
                )}
                {learnerNote}
              </p>
            </div>
            <span className='hidden max-w-36 text-right text-xs leading-5 text-[var(--admin-nav-muted)] sm:block'>
              Select a learner below to open their full progress.
            </span>
          </div>
        </CardContent>
      </Card>

      <DashboardMetricCard
        className='xl:col-span-3'
        icon='trending-up'
        label='Average mastery'
        progress={summary.averageMastery}
        tone={getMasteryTone(summary.averageMastery)}
        value={`${summary.averageMastery}%`}
      >
        <span>Class grade</span>
        <span className='rounded-md bg-[var(--admin-surface-muted)] px-2 py-0.5 font-semibold text-[var(--admin-ink)]'>
          {classGrade}
        </span>
        <span className='sr-only'>
          {summary.averageMastery}% average mastery
        </span>
        <span className='sr-only'>Weakest: {weakestTopicLabel}</span>
        <span className='sr-only'>Strongest: {strongestTopicLabel}</span>
      </DashboardMetricCard>

      <DashboardMetricCard
        className='xl:col-span-4'
        icon='book-open'
        label='Coverage'
        progress={summary.coveragePercent}
        value={`${summary.coveragePercent}%`}
      >
        <span>
          {summary.trackedScores} of {summary.studentCount * summary.topicCount}{' '}
          topic scores recorded
        </span>
      </DashboardMetricCard>

      <Card className='py-0 md:col-span-2 xl:col-span-7'>
        <CardContent className='grid gap-4 p-5 sm:grid-cols-2 sm:p-6'>
          <TopicSignal
            description={
              summary.weakestTopic
                ? 'Start here for the highest-leverage review.'
                : 'Topic signals appear after learners start work.'
            }
            label='Weakest topic'
            name={weakestTopicLabel}
            tone='rose'
          />
          <TopicSignal
            description={
              summary.strongestTopic
                ? 'The class is most secure on this topic.'
                : 'Topic signals appear after learners start work.'
            }
            label='Strongest topic'
            name={strongestTopicLabel}
            tone='emerald'
          />
        </CardContent>
      </Card>

      <Card className='overflow-hidden border-0 bg-[var(--surface-primary-default-subtle)] py-0 shadow-none ring-0 md:col-span-2 xl:col-span-5'>
        <CardContent className='relative flex h-full min-h-36 items-center gap-4 overflow-hidden p-5 pr-28 sm:p-6 sm:pr-36'>
          <span className='flex size-11 shrink-0 items-center justify-center rounded-xl bg-[var(--admin-surface)] text-[var(--icon-tertiary-default)] ring-1 ring-[var(--border-primary-default)]'>
            <PandaiIcon className='size-5' name='star' />
          </span>
          <div>
            <p className='font-semibold text-[var(--text-tertiary-default)]'>
              Today’s focus
            </p>
            <p className='mt-1 text-sm leading-5 text-[var(--text-default-body)]'>
              {summary.weakestTopic
                ? `Review ${weakestTopicLabel.toLowerCase()} with the learners marked for attention.`
                : 'Learner activity will shape the next recommended review.'}
            </p>
          </div>
          <img
            alt=''
            aria-hidden='true'
            className='pointer-events-none absolute -right-3 -bottom-6 w-32 drop-shadow-[0_18px_18px_rgba(12,58,44,0.14)] sm:right-1 sm:w-36'
            src='/illustrations/pbot.svg'
          />
        </CardContent>
      </Card>
    </section>
  )
}

function DashboardMetricCard({
  children,
  className,
  icon,
  label,
  progress,
  tone,
  value,
}: {
  children: ReactNode
  className?: string
  icon: PandaiIconName
  label: string
  progress: number
  tone?: string
  value: string
}) {
  return (
    <Card className={cn('py-0', className)}>
      <CardContent className='flex h-full min-h-48 flex-col justify-between p-5 sm:p-6'>
        <div className='flex items-center justify-between'>
          <p className='text-sm font-medium text-[var(--admin-ink-soft)]'>
            {label}
          </p>
          <span className='flex size-8 items-center justify-center rounded-lg bg-[var(--admin-surface-muted)] text-[var(--icon-tertiary-default)]'>
            <PandaiIcon className='size-4' name={icon} />
          </span>
        </div>
        <div className='mt-7'>
          <p
            className={cn(
              'text-3xl leading-none font-semibold tracking-[-0.025em] text-[var(--admin-ink)] tabular-nums',
              tone,
            )}
          >
            {value}
          </p>
          <Progress
            aria-label={`${label}: ${progress}%`}
            className='mt-5 h-1.5 bg-[var(--border-general-default)] [&_[data-slot=progress-indicator]]:bg-[var(--surface-primary-default)] [&_[data-slot=progress-indicator]]:transition-transform [&_[data-slot=progress-indicator]]:duration-200 [&_[data-slot=progress-indicator]]:ease-[cubic-bezier(0.23,1,0.32,1)]'
            value={progress}
          />
          <div className='mt-3 flex flex-wrap items-center gap-1.5 text-xs text-[var(--admin-muted)]'>
            {children}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

function TopicSignal({
  description,
  label,
  name,
  tone,
}: {
  description: string
  label: string
  name: string
  tone: 'emerald' | 'rose'
}) {
  return (
    <div className='flex gap-3 rounded-xl bg-[var(--surface-secondary-default-hover)] p-4'>
      <span
        className={cn(
          'mt-1 size-2.5 shrink-0 rounded-full ring-4',
          tone === 'rose'
            ? 'bg-[var(--status-danger-text)] ring-[var(--status-danger-surface)]'
            : 'bg-[var(--surface-primary-default)] ring-[var(--surface-primary-default-subtle)]',
        )}
      />
      <div>
        <p className='text-xs font-medium tracking-[0.08em] text-[var(--admin-muted)] uppercase'>
          {label}
        </p>
        <p className='mt-1 font-semibold text-[var(--admin-ink)]'>{name}</p>
        <p className='mt-1 text-xs leading-5 text-[var(--admin-muted)]'>
          {description}
        </p>
      </div>
    </div>
  )
}

function DashboardHeatmap({
  hasHeatmap,
  onNudge,
  onSelectStudent,
  progress,
  sendingStudentID,
}: {
  hasHeatmap: boolean
  onNudge: (studentID: string, studentName: string) => void
  onSelectStudent: (studentID: string) => void
  progress: ClassProgress
  sendingStudentID: string
}) {
  const [filter, setFilter] = useState<LearnerProgressFilter>('all')
  const [query, setQuery] = useState('')
  const learners = useMemo(
    () => getDashboardLearners(progress, query, filter),
    [filter, progress, query],
  )
  const hasLearners = progress.students.length > 0

  return (
    <AdminSurface contentClassName='p-0'>
      <section aria-label='Learner progress'>
        <AdminSurfaceHeader
          action={masteryLegend}
          className='flex-col items-start border-b border-[var(--border-general-default)] p-5 sm:flex-row sm:items-center sm:p-6'
          description='Start with learners who need support, then open progress or send a nudge.'
          title='Learner progress'
        />
        {hasLearners ? (
          <LearnerProgressTools
            filter={filter}
            onFilterChange={setFilter}
            onQueryChange={setQuery}
            query={query}
            resultCount={learners.length}
            totalCount={progress.students.length}
          />
        ) : null}
        <div className='overflow-x-auto px-3 pb-3 sm:px-5 sm:pb-5'>
          {hasHeatmap ? (
            learners.length > 0 ? (
              <Table className='min-w-[960px] border-separate border-spacing-y-1.5'>
                <TableHeader>
                  <TableRow className='border-0 hover:bg-transparent'>
                    <TableHead className='sticky left-0 z-10 min-w-52 bg-[var(--admin-surface)] px-3 py-3 text-xs font-semibold tracking-[0.1em] text-[var(--admin-muted)] uppercase'>
                      Student
                    </TableHead>
                    {progress.topic_ids.map((topicID) => (
                      <TableHead
                        className='min-w-28 px-2 py-3 text-center text-xs font-semibold tracking-[0.1em] text-[var(--admin-muted)] uppercase'
                        key={topicID}
                      >
                        <TopicHeaderLabel topicID={topicID} />
                      </TableHead>
                    ))}
                    <TableHead className='min-w-24 px-2 py-3 text-center text-xs font-semibold tracking-[0.1em] text-[var(--admin-muted)] uppercase'>
                      Average
                    </TableHead>
                    <TableHead className='min-w-64 px-3 py-3 text-right text-xs font-semibold tracking-[0.1em] text-[var(--admin-muted)] uppercase'>
                      Next action
                    </TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {learners.map((learner) => {
                    const rowSurfaceClass = learner.needsAttention
                      ? 'bg-[var(--status-danger-surface)]'
                      : 'bg-[var(--surface-secondary-default-hover)]'
                    const { student } = learner

                    return (
                      <TableRow
                        className='group/heatmap-row border-0 hover:bg-transparent'
                        key={student.id}
                      >
                        <TableCell
                          className={cn(
                            'sticky left-0 z-10 rounded-l-xl p-3 font-medium text-[var(--admin-ink)] transition-[background-color] duration-150 group-hover/heatmap-row:bg-[var(--admin-surface-muted)]',
                            rowSurfaceClass,
                          )}
                        >
                          <StudentDetailButton
                            learner={learner}
                            onSelectStudent={onSelectStudent}
                            studentID={student.id}
                            studentName={student.name}
                          />
                        </TableCell>
                        {progress.topic_ids.map((topicID) => (
                          <TableCell
                            className={cn(
                              'p-2 text-center transition-[background-color] duration-150 group-hover/heatmap-row:bg-[var(--admin-surface-muted)]',
                              rowSurfaceClass,
                            )}
                            key={`${student.id}-${topicID}`}
                          >
                            <MasteryScore score={student.topics[topicID]} />
                          </TableCell>
                        ))}
                        <TableCell
                          className={cn(
                            'p-2 text-center transition-[background-color] duration-150 group-hover/heatmap-row:bg-[var(--admin-surface-muted)]',
                            rowSurfaceClass,
                          )}
                        >
                          <StudentAverage learner={learner} />
                        </TableCell>
                        <TableCell
                          className={cn(
                            'rounded-r-xl p-3 text-right transition-[background-color] duration-150 group-hover/heatmap-row:bg-[var(--admin-surface-muted)]',
                            rowSurfaceClass,
                          )}
                        >
                          <div className='flex items-center justify-end gap-3'>
                            <p className='max-w-44 text-right text-xs leading-5 font-medium text-[var(--admin-ink-soft)]'>
                              {learner.nextAction}
                            </p>
                            <NudgeButton
                              isSending={sendingStudentID === student.id}
                              onNudge={onNudge}
                              studentID={student.id}
                              studentName={student.name}
                            />
                          </div>
                        </TableCell>
                      </TableRow>
                    )
                  })}
                </TableBody>
              </Table>
            ) : (
              <div className='py-5'>
                <StatePanel title='No learners match your search'>
                  Change the name or progress filter, or clear the search field.
                </StatePanel>
              </div>
            )
          ) : (
            <div className='py-5'>
              <StatePanel title='No class progress yet'>
                Topic scores will appear after students begin practising.
              </StatePanel>
            </div>
          )}
        </div>
      </section>
    </AdminSurface>
  )
}

function LearnerProgressTools({
  filter,
  onFilterChange,
  onQueryChange,
  query,
  resultCount,
  totalCount,
}: {
  filter: LearnerProgressFilter
  onFilterChange: (filter: LearnerProgressFilter) => void
  onQueryChange: (query: string) => void
  query: string
  resultCount: number
  totalCount: number
}) {
  const handleQueryChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      onQueryChange(event.target.value)
    },
    [onQueryChange],
  )
  const handleFilterChange = useCallback(
    (event: ChangeEvent<HTMLSelectElement>) => {
      onFilterChange(event.target.value as LearnerProgressFilter)
    },
    [onFilterChange],
  )

  return (
    <div className='flex flex-col gap-3 border-b border-[var(--border-general-default)] p-4 sm:flex-row sm:items-end sm:justify-between sm:px-6'>
      <div className='grid flex-1 gap-3 sm:max-w-2xl sm:grid-cols-[minmax(0,1fr)_13rem]'>
        <label
          className='grid gap-1.5 text-xs font-medium text-[var(--admin-ink-soft)]'
          htmlFor='dashboard-learner-search'
        >
          Search learners
          <span className='relative'>
            <PandaiIcon
              className='pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-[var(--admin-muted)]'
              name='search'
            />
            <Input
              className='h-11 bg-white pl-9'
              id='dashboard-learner-search'
              onChange={handleQueryChange}
              placeholder='Search by name…'
              type='search'
              value={query}
            />
          </span>
        </label>
        <label
          className='grid gap-1.5 text-xs font-medium text-[var(--admin-ink-soft)]'
          htmlFor='dashboard-progress-status'
        >
          Progress status
          <NativeSelect
            className='w-full [&_select]:h-11 [&_select]:bg-white [&_select]:text-base md:[&_select]:text-sm'
            id='dashboard-progress-status'
            onChange={handleFilterChange}
            value={filter}
          >
            <NativeSelectOption value='all'>All learners</NativeSelectOption>
            <NativeSelectOption value='attention'>
              Needs attention
            </NativeSelectOption>
            <NativeSelectOption value='unmeasured'>
              No activity yet
            </NativeSelectOption>
            <NativeSelectOption value='on-track'>On track</NativeSelectOption>
          </NativeSelect>
        </label>
      </div>
      <p
        aria-live='polite'
        className='text-sm text-[var(--admin-muted)] tabular-nums'
      >
        Showing {resultCount} of {totalCount}
      </p>
    </div>
  )
}

function MasteryLegend() {
  return (
    <div
      aria-label='Mastery score legend'
      className='flex flex-wrap items-center gap-3 text-xs text-[var(--admin-muted)]'
    >
      <LegendItem
        className='bg-[var(--status-danger-text)]'
        label='Below 40%'
      />
      <LegendItem
        className='bg-[var(--status-warning-border)]'
        label='40–59%'
      />
      <LegendItem
        className='bg-[var(--surface-secondary-default)]'
        label='60–79%'
      />
      <LegendItem
        className='bg-[var(--surface-primary-default)]'
        label='80%+'
      />
    </div>
  )
}

function LegendItem({
  className,
  label,
}: {
  className: string
  label: string
}) {
  return (
    <span className='inline-flex items-center gap-1.5'>
      <span className={cn('size-2 rounded-sm', className)} />
      {label}
    </span>
  )
}

function StudentDetailSheet({
  onOpenChange,
  studentID,
  studentName,
}: {
  onOpenChange: () => void
  studentID: string | undefined
  studentName: string | undefined
}) {
  const handleSheetOpenChange = useCallback(
    (open: boolean) => {
      if (!open) {
        onOpenChange()
      }
    },
    [onOpenChange],
  )

  return (
    <Sheet open={Boolean(studentID)} onOpenChange={handleSheetOpenChange}>
      <SheetContent className='w-full overflow-y-auto border-l border-[var(--border-primary-default)] bg-[var(--admin-canvas)] p-0 sm:max-w-[720px]'>
        <SheetHeader className='border-b border-[var(--border-primary-default)] bg-[var(--admin-surface)] p-5 sm:p-6'>
          <p className='text-xs font-semibold tracking-[0.1em] text-[var(--text-tertiary-default)] uppercase'>
            Learner profile
          </p>
          <SheetTitle className='text-xl'>
            {studentName ?? 'Learner profile'}
          </SheetTitle>
          <SheetDescription>
            Progress, streaks, and chat history.
          </SheetDescription>
        </SheetHeader>
        <div className='p-4 sm:p-6'>
          {studentID ? (
            <StudentDetailPanel studentID={studentID} variant='sheet' />
          ) : null}
        </div>
      </SheetContent>
    </Sheet>
  )
}

function TopicHeaderLabel({ topicID }: { topicID: string }) {
  const topicLabel = formatTopicLabel(topicID)

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <button
            className='inline-block min-h-11 max-w-28 truncate rounded-md border-0 bg-transparent px-1 py-1 align-middle font-[inherit] tracking-[inherit] text-inherit normal-case transition-[background-color,color] duration-150 hover:bg-[var(--admin-surface-muted)] hover:text-[var(--admin-ink)] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[var(--border-primary-focus)] sm:min-h-8'
            type='button'
          >
            {topicLabel}
          </button>
        </TooltipTrigger>
        <TooltipContent>
          <p>{topicLabel}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}

function MasteryScore({ score }: { score: number | undefined }) {
  if (typeof score !== 'number') {
    return (
      <span className='inline-flex min-h-9 min-w-16 items-center justify-center rounded-lg border border-dashed border-[var(--border-general-default)] bg-white/50 px-3 py-1 text-xs font-medium text-[var(--admin-muted)]'>
        --
      </span>
    )
  }

  return (
    <span
      className={cn(
        'inline-flex min-h-9 min-w-16 items-center justify-center rounded-lg border px-3 py-1 text-xs font-semibold tabular-nums',
        getMasteryScoreTone(score),
      )}
    >
      {Math.round(score * 100)}%
    </span>
  )
}

function StudentAverage({ learner }: { learner: DashboardLearner }) {
  if (learner.averageMastery === null) {
    return (
      <span
        aria-label='No average mastery data'
        className='inline-flex min-h-9 min-w-16 items-center justify-center rounded-lg border border-dashed border-[var(--border-general-default)] bg-white/50 px-3 py-1 text-xs font-medium text-[var(--admin-muted)]'
      >
        --
      </span>
    )
  }

  return (
    <span
      aria-label={`${learner.averageMastery}% average mastery`}
      className={cn(
        'inline-flex min-h-9 min-w-16 items-center justify-center rounded-lg border bg-white px-3 py-1 text-xs font-semibold tabular-nums',
        learner.needsAttention
          ? 'border-[var(--status-danger-border)] text-[var(--status-danger-text)]'
          : 'border-[var(--border-general-default)] text-[var(--admin-ink)]',
      )}
    >
      {learner.averageMastery}%
    </span>
  )
}

function StudentDetailButton({
  learner,
  onSelectStudent,
  studentID,
  studentName,
}: {
  learner: DashboardLearner
  onSelectStudent: (studentID: string) => void
  studentID: string
  studentName: string
}) {
  const selectStudent = useCallback(() => {
    onSelectStudent(studentID)
  }, [onSelectStudent, studentID])

  return (
    <button
      aria-label={`Open ${studentName} progress`}
      className='inline-flex w-full items-center justify-between gap-3 rounded-lg border-0 bg-transparent p-0 text-left font-[inherit] text-[var(--admin-ink)] transition-[color,transform] duration-150 ease-[cubic-bezier(0.23,1,0.32,1)] hover:text-[var(--admin-ink-hover)] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[var(--border-primary-focus)] active:scale-[0.96]'
      onClick={selectStudent}
      type='button'
    >
      <span className='inline-flex min-w-0 items-center gap-3'>
        <span
          className={cn(
            'flex size-9 shrink-0 items-center justify-center rounded-lg text-xs font-semibold',
            learner.needsAttention
              ? 'bg-[var(--status-danger-surface)] text-[var(--status-danger-text)]'
              : 'bg-[var(--admin-surface-muted)] text-[var(--text-tertiary-default)]',
          )}
        >
          {getInitials(studentName)}
        </span>
        <span className='min-w-0'>
          <span className='block truncate'>{studentName}</span>
          <span className='mt-0.5 flex items-center gap-1.5 text-xs font-normal text-[var(--admin-muted)]'>
            <AttentionDot learner={learner} />
            {getLearnerStatusLabel(learner.status)}
          </span>
        </span>
      </span>
      <PandaiIcon
        className='size-4 shrink-0 text-[var(--admin-muted)]'
        name='chevron-right'
      />
    </button>
  )
}

function AttentionDot({ learner }: { learner: DashboardLearner }) {
  return (
    <span
      aria-label={
        learner.averageMastery === null
          ? 'No mastery data'
          : learner.needsAttention
            ? `${learner.averageMastery}% average mastery`
            : 'No attention flag'
      }
      className={cn(
        'size-1.5 shrink-0 rounded-full',
        learner.needsAttention
          ? 'bg-[var(--status-danger-text)]'
          : learner.status === 'unmeasured'
            ? 'bg-[var(--admin-muted)]'
            : 'bg-[var(--surface-primary-default)]',
      )}
      role='img'
    />
  )
}

function getLearnerStatusLabel(status: DashboardLearner['status']): string {
  if (status === 'attention') {
    return 'Needs attention'
  }

  return status === 'unmeasured' ? 'No activity yet' : 'On track'
}

function NudgeButton({
  isSending,
  onNudge,
  studentID,
  studentName,
}: {
  isSending: boolean
  onNudge: (studentID: string, studentName: string) => void
  studentID: string
  studentName: string
}) {
  const sendNudge = useCallback(() => {
    onNudge(studentID, studentName)
  }, [onNudge, studentID, studentName])

  return (
    <Button
      aria-label={
        isSending
          ? `Sending nudge to ${studentName}`
          : `Send nudge to ${studentName}`
      }
      className='min-h-11 px-3 sm:min-h-8'
      disabled={isSending}
      onClick={sendNudge}
      size='sm'
      type='button'
    >
      <PandaiIcon data-icon='inline-start' name='bell' />
      {getNudgeButtonLabel(isSending)}
    </Button>
  )
}

function getNudgeButtonLabel(isSending: boolean): string {
  return isSending ? 'Sending…' : 'Send nudge'
}

function getLearnerAttentionNote(attentionCount: number): string {
  if (attentionCount === 0) {
    return 'No learners need attention right now'
  }

  const verb = attentionCount === 1 ? 'needs' : 'need'
  return `${attentionCount} learner${attentionCount === 1 ? '' : 's'} ${verb} attention`
}

function getMasteryTone(averageMastery: number): string {
  if (averageMastery >= 80) {
    return 'text-[var(--status-success-text)]'
  }

  if (averageMastery >= 60) {
    return 'text-[var(--status-warning-text)]'
  }

  return 'text-[var(--status-danger-text)]'
}

function getMasteryGrade(averageMastery: number): string {
  return (
    masteryGrades.find((grade) => averageMastery >= grade.minimum)?.label ?? 'F'
  )
}

function getMasteryScoreTone(score: number): string {
  return (
    masteryScoreTones.find((tone) => score >= tone.minimum)?.className ??
    'border-[var(--status-danger-border)] bg-[var(--status-danger-surface)] text-[var(--status-danger-text)]'
  )
}

const masteryScoreTones = [
  {
    className:
      'border-[var(--status-success-border)] bg-[var(--status-success-surface)] text-[var(--status-success-text)]',
    minimum: 0.8,
  },
  {
    className:
      'border-[var(--border-secondary-focus)] bg-[var(--surface-secondary-default-subtle)] text-[var(--text-tertiary-default)]',
    minimum: 0.6,
  },
  {
    className:
      'border-[var(--status-warning-border)] bg-[var(--status-warning-surface)] text-[var(--status-warning-text)]',
    minimum: 0.4,
  },
] as const

const masteryGrades = [
  { label: 'A', minimum: 90 },
  { label: 'B', minimum: 80 },
  { label: 'C', minimum: 70 },
  { label: 'D', minimum: 60 },
  { label: 'E', minimum: 50 },
] as const

function noop() {}

function getInitials(name: string): string {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0].toUpperCase())
    .join('')
}

function formatTopicLabel(topicID: string): string {
  return topicID
    .split('-')
    .filter(Boolean)
    .map((word) => `${word[0].toUpperCase()}${word.slice(1)}`)
    .join(' ')
}
