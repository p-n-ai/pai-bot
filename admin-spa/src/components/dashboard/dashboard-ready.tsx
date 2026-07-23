import {
  BellRingIcon,
  BookOpenCheckIcon,
  CheckCircle2Icon,
  ChevronRightIcon,
  CircleAlertIcon,
  SparklesIcon,
  TrendingUpIcon,
  UsersIcon,
} from 'lucide-react'
import { useCallback, useMemo } from 'react'
import type { ComponentType, ReactNode } from 'react'

import type { ClassProgress } from '@/lib/dashboard-types'
import {
  AdminSurface,
  AdminSurfaceHeader,
} from '@/components/shared/admin-surface'
import { StatePanel } from '@/components/shared/state-panel'
import { StudentDetailPanel } from '@/components/dashboard/student-detail-panel'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
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
  onSelectStudent = noop,
  progress,
  selectedStudentID,
  sendingStudentID,
}: {
  nudgeMessage: string
  onCloseStudent?: () => void
  onNudge: (studentID: string, studentName: string) => void
  onSelectStudent?: (studentID: string) => void
  progress: ClassProgress
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
      <DashboardHeatmap
        hasHeatmap={summary.hasHeatmap}
        onNudge={onNudge}
        onSelectStudent={onSelectStudent}
        progress={progress}
        sendingStudentID={sendingStudentID}
      />
      {nudgeMessage ? (
        <div
          className='flex items-center gap-2.5 rounded-xl border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm font-medium text-emerald-800 shadow-[0_1px_2px_rgb(16_24_40/0.04)]'
          role='status'
        >
          <CheckCircle2Icon aria-hidden='true' className='size-4 shrink-0' />
          {nudgeMessage}
        </div>
      ) : null}
      <StudentDetailSheet
        onOpenChange={onCloseStudent}
        studentID={selectedStudentID}
        studentName={selectedStudent?.name}
      />
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
      <Card className='relative border-0 bg-[#101828] py-0 text-white shadow-[0_1px_2px_rgb(16_24_40/0.08),0_12px_28px_rgb(16_24_40/0.18)] ring-0 md:col-span-2 xl:col-span-5'>
        <div
          aria-hidden='true'
          className='pointer-events-none absolute inset-0 overflow-hidden rounded-xl'
        >
          <span className='absolute -top-16 -right-12 size-48 rounded-full bg-[#2f80ed]/20 blur-3xl' />
          <span className='absolute -bottom-20 left-20 size-40 rounded-full bg-[#53d3a0]/10 blur-3xl' />
        </div>
        <CardContent className='relative flex h-full min-h-52 flex-col justify-between p-5 sm:p-6'>
          <div className='flex items-start justify-between gap-4'>
            <div className='flex items-center gap-2.5'>
              <span className='flex size-9 items-center justify-center rounded-lg bg-white/10 text-[#84adff] ring-1 ring-white/10'>
                <UsersIcon aria-hidden='true' className='size-[18px]' />
              </span>
              <div>
                <p className='text-sm font-medium text-white'>Learners</p>
                <p className='text-xs text-[#98a2b3]'>Active class roster</p>
              </div>
            </div>
            <span className='inline-flex items-center gap-1.5 rounded-full bg-white/8 px-2.5 py-1 text-[11px] font-medium text-[#d0d5dd] ring-1 ring-white/10'>
              <span className='size-1.5 rounded-full bg-[#53d3a0]' />
              Live
            </span>
          </div>
          <div className='mt-8 flex items-end justify-between gap-5'>
            <div>
              <p className='text-5xl leading-none font-semibold text-white'>
                {summary.studentCount}
              </p>
              <p
                className={cn(
                  'mt-3 flex items-center gap-1.5 text-sm font-medium',
                  summary.attentionCount > 0
                    ? 'text-[#fdb9a8]'
                    : 'text-[#75e0a7]',
                )}
              >
                {summary.attentionCount > 0 ? (
                  <CircleAlertIcon aria-hidden='true' className='size-4' />
                ) : (
                  <CheckCircle2Icon aria-hidden='true' className='size-4' />
                )}
                {learnerNote}
              </p>
            </div>
            <span className='hidden max-w-36 text-right text-xs leading-5 text-[#98a2b3] sm:block'>
              Select a learner below to open their full progress.
            </span>
          </div>
        </CardContent>
      </Card>

      <DashboardMetricCard
        className='xl:col-span-3'
        icon={TrendingUpIcon}
        label='Average mastery'
        progress={summary.averageMastery}
        tone={getMasteryTone(summary.averageMastery)}
        value={`${summary.averageMastery}%`}
      >
        <span>Class grade</span>
        <span className='rounded-md bg-[#f2f4f7] px-2 py-0.5 font-semibold text-[#344054]'>
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
        icon={BookOpenCheckIcon}
        label='Coverage'
        progress={summary.coveragePercent}
        value={`${summary.coveragePercent}%`}
      >
        <span>
          {summary.trackedScores} of {summary.studentCount * summary.topicCount}{' '}
          score slots filled
        </span>
      </DashboardMetricCard>

      <Card className='py-0 shadow-[0_1px_2px_rgb(16_24_40/0.04),0_8px_20px_rgb(16_24_40/0.05)] ring-[#e6e9ef] md:col-span-2 xl:col-span-7'>
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

      <Card className='overflow-hidden border-0 bg-[#eaf2ff] py-0 shadow-none ring-0 md:col-span-2 xl:col-span-5'>
        <CardContent className='flex h-full items-center gap-4 p-5 sm:p-6'>
          <span className='flex size-11 shrink-0 items-center justify-center rounded-xl bg-white text-[#175cd3] shadow-[0_1px_2px_rgb(16_24_40/0.05),0_6px_14px_rgb(23_92_211/0.12)]'>
            <SparklesIcon aria-hidden='true' className='size-5' />
          </span>
          <div>
            <p className='font-semibold text-[#194185]'>Today’s focus</p>
            <p className='mt-1 text-sm leading-5 text-[#175cd3]'>
              {summary.weakestTopic
                ? `Review ${weakestTopicLabel.toLowerCase()} with the learners marked for attention.`
                : 'Learner activity will shape the next recommended review.'}
            </p>
          </div>
        </CardContent>
      </Card>
    </section>
  )
}

function DashboardMetricCard({
  children,
  className,
  icon: Icon,
  label,
  progress,
  tone,
  value,
}: {
  children: ReactNode
  className?: string
  icon: ComponentType<{ 'aria-hidden'?: boolean; className?: string }>
  label: string
  progress: number
  tone?: string
  value: string
}) {
  return (
    <Card
      className={cn(
        'py-0 shadow-[0_1px_2px_rgb(16_24_40/0.04),0_8px_20px_rgb(16_24_40/0.05)] ring-[#e6e9ef]',
        className,
      )}
    >
      <CardContent className='flex h-full min-h-52 flex-col justify-between p-5 sm:p-6'>
        <div className='flex items-center justify-between'>
          <p className='text-sm font-medium text-[#475467]'>{label}</p>
          <span className='flex size-8 items-center justify-center rounded-lg bg-[#f2f4f7] text-[#667085]'>
            <Icon aria-hidden className='size-4' />
          </span>
        </div>
        <div className='mt-7'>
          <p
            className={cn(
              'text-4xl leading-none font-semibold text-[#101828]',
              tone,
            )}
          >
            {value}
          </p>
          <Progress
            aria-label={`${label}: ${progress}%`}
            className='mt-5 h-1.5 bg-[#eaecf0] [&_[data-slot=progress-indicator]]:bg-[#2f80ed] [&_[data-slot=progress-indicator]]:transition-transform [&_[data-slot=progress-indicator]]:duration-200 [&_[data-slot=progress-indicator]]:ease-[cubic-bezier(0.23,1,0.32,1)]'
            value={progress}
          />
          <div className='mt-3 flex flex-wrap items-center gap-1.5 text-xs text-[#667085]'>
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
    <div className='flex gap-3 rounded-xl bg-[#f8fafc] p-4'>
      <span
        className={cn(
          'mt-1 size-2.5 shrink-0 rounded-full ring-4',
          tone === 'rose'
            ? 'bg-rose-500 ring-rose-100'
            : 'bg-emerald-500 ring-emerald-100',
        )}
      />
      <div>
        <p className='text-xs font-medium tracking-[0.08em] text-[#667085] uppercase'>
          {label}
        </p>
        <p className='mt-1 font-semibold text-[#101828]'>{name}</p>
        <p className='mt-1 text-xs leading-5 text-[#667085]'>{description}</p>
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
  return (
    <AdminSurface
      className='shadow-[0_1px_2px_rgb(16_24_40/0.04),0_10px_28px_rgb(16_24_40/0.05)] ring-[#e6e9ef]'
      contentClassName='p-0'
    >
      <section aria-label='Mastery heatmap'>
        <AdminSurfaceHeader
          action={masteryLegend}
          className='flex-col items-start border-b border-[#eef0f3] p-5 sm:flex-row sm:items-center sm:p-6'
          description='Students by topic with direct navigation into detail views.'
          title='Mastery heatmap'
        />
        <div className='overflow-x-auto px-3 pb-3 sm:px-5 sm:pb-5'>
          {hasHeatmap ? (
            <Table className='min-w-[820px] border-separate border-spacing-y-1.5'>
              <TableHeader>
                <TableRow className='border-0 hover:bg-transparent'>
                  <TableHead className='sticky left-0 z-10 min-w-52 bg-white px-3 py-3 text-[10px] font-semibold tracking-[0.12em] text-[#98a2b3] uppercase'>
                    Student
                  </TableHead>
                  {progress.topic_ids.map((topicID) => (
                    <TableHead
                      className='min-w-28 px-2 py-3 text-center text-[10px] font-semibold tracking-[0.12em] text-[#98a2b3] uppercase'
                      key={topicID}
                    >
                      <TopicHeaderLabel topicID={topicID} />
                    </TableHead>
                  ))}
                  <TableHead className='min-w-24 px-2 py-3 text-center text-[10px] font-semibold tracking-[0.12em] text-[#98a2b3] uppercase'>
                    Average
                  </TableHead>
                  <TableHead className='min-w-28 px-3 py-3 text-right text-[10px] font-semibold tracking-[0.12em] text-[#98a2b3] uppercase'>
                    Nudge
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {progress.students.map((student) => {
                  const attention = getStudentAttention(progress, student)
                  const rowSurfaceClass = attention.needsAttention
                    ? 'bg-rose-50/75'
                    : 'bg-[#f8fafc]'

                  return (
                    <TableRow
                      className='group/heatmap-row border-0 hover:bg-transparent'
                      key={student.id}
                    >
                      <TableCell
                        className={cn(
                          'sticky left-0 z-10 rounded-l-xl p-3 font-medium text-[#101828] shadow-[-6px_0_0_white] transition-[background-color] duration-150 group-hover/heatmap-row:bg-[#f2f4f7]',
                          rowSurfaceClass,
                        )}
                      >
                        <StudentDetailButton
                          attention={attention}
                          onSelectStudent={onSelectStudent}
                          studentID={student.id}
                          studentName={student.name}
                        />
                      </TableCell>
                      {progress.topic_ids.map((topicID) => (
                        <TableCell
                          className={cn(
                            'p-2 text-center transition-[background-color] duration-150 group-hover/heatmap-row:bg-[#f2f4f7]',
                            rowSurfaceClass,
                          )}
                          key={`${student.id}-${topicID}`}
                        >
                          <MasteryScore score={student.topics[topicID]} />
                        </TableCell>
                      ))}
                      <TableCell
                        className={cn(
                          'p-2 text-center transition-[background-color] duration-150 group-hover/heatmap-row:bg-[#f2f4f7]',
                          rowSurfaceClass,
                        )}
                      >
                        <StudentAverage attention={attention} />
                      </TableCell>
                      <TableCell
                        className={cn(
                          'rounded-r-xl p-3 text-right transition-[background-color] duration-150 group-hover/heatmap-row:bg-[#f2f4f7]',
                          rowSurfaceClass,
                        )}
                      >
                        <NudgeButton
                          isSending={sendingStudentID === student.id}
                          onNudge={onNudge}
                          studentID={student.id}
                          studentName={student.name}
                        />
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          ) : (
            <div className='py-5'>
              <StatePanel title='No class heatmap yet'>
                Progress appears after students start topics.
              </StatePanel>
            </div>
          )}
        </div>
      </section>
    </AdminSurface>
  )
}

function MasteryLegend() {
  return (
    <div
      aria-label='Mastery score legend'
      className='flex flex-wrap items-center gap-3 text-[11px] text-[#667085]'
    >
      <LegendItem className='bg-rose-400' label='Below 40%' />
      <LegendItem className='bg-amber-400' label='40–59%' />
      <LegendItem className='bg-lime-400' label='60–79%' />
      <LegendItem className='bg-emerald-500' label='80%+' />
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
      <SheetContent className='w-full overflow-y-auto border-l border-[#e6e9ef] bg-[#f8fafc] p-0 shadow-[-16px_0_48px_rgb(16_24_40/0.12)] sm:max-w-[720px]'>
        <SheetHeader className='border-b border-[#e6e9ef] bg-white p-5 sm:p-6'>
          <p className='text-[10px] font-semibold tracking-[0.12em] text-[#2f80ed] uppercase'>
            Learner profile
          </p>
          <SheetTitle className='text-xl'>
            {studentName ?? 'Student detail'}
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
            className='inline-block max-w-28 truncate rounded-md border-0 bg-transparent px-1 py-1 align-middle font-[inherit] tracking-[inherit] text-inherit normal-case transition-[background-color,color] duration-150 hover:bg-[#f2f4f7] hover:text-[#344054] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#2f80ed]'
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
      <span className='inline-flex min-h-9 min-w-16 items-center justify-center rounded-lg border border-dashed border-[#d0d5dd] bg-white/50 px-3 py-1 text-xs font-medium text-[#98a2b3]'>
        --
      </span>
    )
  }

  return (
    <span
      className={cn(
        'inline-flex min-h-9 min-w-16 items-center justify-center rounded-lg border px-3 py-1 text-xs font-semibold shadow-[inset_0_1px_0_rgb(255_255_255/0.45)]',
        getMasteryScoreTone(score),
      )}
    >
      {Math.round(score * 100)}%
    </span>
  )
}

function StudentAverage({ attention }: { attention: StudentAttention }) {
  if (attention.averageMastery === null) {
    return (
      <span
        aria-label='No average mastery data'
        className='inline-flex min-h-9 min-w-16 items-center justify-center rounded-lg border border-dashed border-[#d0d5dd] bg-white/50 px-3 py-1 text-xs font-medium text-[#98a2b3]'
      >
        --
      </span>
    )
  }

  return (
    <span
      aria-label={`${attention.averageMastery}% average mastery`}
      className={cn(
        'inline-flex min-h-9 min-w-16 items-center justify-center rounded-lg border bg-white px-3 py-1 text-xs font-semibold',
        attention.needsAttention
          ? 'border-rose-200 text-rose-700'
          : 'border-[#e6e9ef] text-[#344054]',
      )}
    >
      {attention.averageMastery}%
    </span>
  )
}

function StudentDetailButton({
  attention,
  onSelectStudent,
  studentID,
  studentName,
}: {
  attention: StudentAttention
  onSelectStudent: (studentID: string) => void
  studentID: string
  studentName: string
}) {
  const selectStudent = useCallback(() => {
    onSelectStudent(studentID)
  }, [onSelectStudent, studentID])

  return (
    <button
      className='inline-flex w-full items-center justify-between gap-3 rounded-lg border-0 bg-transparent p-0 text-left font-[inherit] text-[#101828] transition-[color,transform] duration-150 ease-[cubic-bezier(0.23,1,0.32,1)] hover:text-[#175cd3] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#2f80ed] active:scale-[0.99]'
      onClick={selectStudent}
      type='button'
    >
      <span className='inline-flex min-w-0 items-center gap-3'>
        <span
          className={cn(
            'flex size-9 shrink-0 items-center justify-center rounded-lg text-xs font-semibold',
            attention.needsAttention
              ? 'bg-rose-100 text-rose-700'
              : 'bg-[#eaf2ff] text-[#175cd3]',
          )}
        >
          {getInitials(studentName)}
        </span>
        <span className='min-w-0'>
          <span className='block truncate'>{studentName}</span>
          <span className='mt-0.5 flex items-center gap-1.5 text-[11px] font-normal text-[#667085]'>
            <AttentionDot attention={attention} />
            {attention.needsAttention ? 'Needs attention' : 'On track'}
          </span>
        </span>
      </span>
      <ChevronRightIcon
        aria-hidden='true'
        className='size-4 shrink-0 text-[#98a2b3]'
      />
    </button>
  )
}

function AttentionDot({ attention }: { attention: StudentAttention }) {
  return (
    <span
      aria-label={
        attention.averageMastery === null
          ? 'No mastery data'
          : attention.needsAttention
            ? `${attention.averageMastery}% average mastery`
            : 'No attention flag'
      }
      className={cn(
        'size-1.5 shrink-0 rounded-full',
        attention.needsAttention ? 'bg-rose-500' : 'bg-emerald-500',
      )}
      role='img'
    />
  )
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
      className='bg-[#101828] px-3 text-white shadow-[0_1px_2px_rgb(16_24_40/0.12)] hover:bg-[#344054]'
      disabled={isSending}
      onClick={sendNudge}
      size='sm'
      type='button'
    >
      <BellRingIcon data-icon='inline-start' />
      {getNudgeButtonLabel(isSending)}
    </Button>
  )
}

function getNudgeButtonLabel(isSending: boolean): string {
  return isSending ? 'Sending...' : 'Nudge'
}

function getLearnerAttentionNote(attentionCount: number): string {
  if (attentionCount === 0) {
    return 'No learners flagged right now'
  }

  return `${attentionCount} learner${attentionCount === 1 ? '' : 's'} need attention`
}

function getMasteryTone(averageMastery: number): string {
  if (averageMastery >= 80) {
    return 'text-emerald-700 dark:text-emerald-300'
  }

  if (averageMastery >= 60) {
    return 'text-amber-700 dark:text-amber-300'
  }

  return 'text-rose-700 dark:text-rose-300'
}

function getMasteryGrade(averageMastery: number): string {
  return (
    masteryGrades.find((grade) => averageMastery >= grade.minimum)?.label ?? 'F'
  )
}

function getMasteryScoreTone(score: number): string {
  return (
    masteryScoreTones.find((tone) => score >= tone.minimum)?.className ??
    'border-rose-200 bg-rose-100 text-rose-900'
  )
}

type StudentAttention = {
  averageMastery: number | null
  needsAttention: boolean
}

function getStudentAttention(
  progress: ClassProgress,
  student: ClassProgress['students'][number],
): StudentAttention {
  const scores = progress.topic_ids.flatMap((topicID) => {
    const score = student.topics[topicID]

    return typeof score === 'number' ? [score] : []
  })
  const averageMastery: number | null =
    scores.length > 0
      ? Math.round(
          (scores.reduce((sum, score) => sum + score, 0) / scores.length) * 100,
        )
      : null

  return {
    averageMastery,
    needsAttention: averageMastery !== null && averageMastery < 50,
  }
}

const masteryScoreTones = [
  {
    className: 'border-emerald-200 bg-emerald-100/90 text-emerald-800',
    minimum: 0.8,
  },
  { className: 'border-lime-200 bg-lime-100/90 text-lime-800', minimum: 0.6 },
  {
    className: 'border-amber-200 bg-amber-100/90 text-amber-800',
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
