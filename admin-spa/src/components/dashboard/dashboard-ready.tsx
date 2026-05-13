import { BellRingIcon, ChevronRightIcon } from 'lucide-react'
import { useCallback, useMemo } from 'react'

import type { ClassProgress } from '@/lib/dashboard-types'
import { AdminSurface, AdminSurfaceHeader } from '@/components/shared/admin-surface'
import { StatePanel } from '@/components/shared/state-panel'
import { StatCard } from '@/components/shared/stat-card'
import { StudentDetailPanel } from '@/components/dashboard/student-detail-panel'
import { Button } from '@/components/ui/button'
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
    <div className='mt-8 flex flex-col gap-5'>
      <DashboardStats progress={progress} />
      <DashboardHeatmap
        hasHeatmap={summary.hasHeatmap}
        onNudge={onNudge}
        onSelectStudent={onSelectStudent}
        progress={progress}
        sendingStudentID={sendingStudentID}
      />
      {nudgeMessage ? (
        <p className='text-muted-foreground'>{nudgeMessage}</p>
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
    <section className='grid gap-3 md:grid-cols-2 xl:grid-cols-5'>
      <StatCard
        className='border-amber-200 bg-amber-50/70 shadow-none xl:col-span-2 dark:border-amber-900/40 dark:bg-amber-950/20'
        headerClassName='space-y-2'
        title='Learners'
        value={String(summary.studentCount)}
        note={learnerNote}
        noteClassName={getAttentionTone(summary.attentionCount)}
      />
      <StatCard
        className='shadow-none'
        title='Class grade'
        value={classGrade}
        note={`${summary.averageMastery}% average mastery`}
        noteClassName={getMasteryTone(summary.averageMastery)}
      />
      <StatCard
        className='shadow-none'
        title='Average mastery'
        value={`${summary.averageMastery}%`}
        note={
          <>
            <span className='text-rose-700 dark:text-rose-300'>
              Weakest: {weakestTopicLabel}
            </span>
            <span className='text-muted-foreground'> · </span>
            <span className='text-emerald-700 dark:text-emerald-300'>
              Strongest: {strongestTopicLabel}
            </span>
          </>
        }
        noteClassName='flex flex-wrap gap-1'
      />
      <StatCard
        className='shadow-none'
        title='Coverage'
        value={`${summary.coveragePercent}%`}
        note={`${summary.trackedScores} of ${summary.studentCount * summary.topicCount} score slots filled`}
      />
    </section>
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
    <AdminSurface className='shadow-none' contentClassName='p-4 sm:p-5'>
      <section aria-label='Mastery heatmap'>
        <AdminSurfaceHeader
          description='Students by topic with direct navigation into detail views.'
          title='Mastery heatmap'
        />
        <div className='mt-6'>
          {hasHeatmap ? (
            <Table className='min-w-[760px] border-separate border-spacing-y-2'>
              <TableHeader>
                <TableRow className='border-0 hover:bg-transparent'>
                  <TableHead className='px-3 py-2 text-xs font-semibold tracking-[0.14em] text-muted-foreground uppercase'>
                    Student
                  </TableHead>
                  {progress.topic_ids.map((topicID) => (
                    <TableHead
                      className='px-3 py-2 text-xs font-semibold tracking-[0.14em] text-muted-foreground uppercase'
                      key={topicID}
                    >
                      <TopicHeaderLabel topicID={topicID} />
                    </TableHead>
                  ))}
                  <TableHead className='px-3 py-2 text-xs font-semibold tracking-[0.14em] text-muted-foreground uppercase'>
                    Nudge
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {progress.students.map((student) => {
                  const attention = getStudentAttention(progress, student)
                  const rowSurfaceClass = attention.needsAttention
                    ? 'bg-rose-50/80'
                    : 'bg-muted/40'

                  return (
                    <TableRow
                      className='border-0 hover:bg-transparent'
                      key={student.id}
                    >
                      <TableCell
                        className={cn(
                          'rounded-l-lg p-3 font-medium text-foreground',
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
                          className={cn('p-3', rowSurfaceClass)}
                          key={`${student.id}-${topicID}`}
                        >
                          <MasteryScore score={student.topics[topicID]} />
                        </TableCell>
                      ))}
                      <TableCell
                        className={cn('rounded-r-lg p-3', rowSurfaceClass)}
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
            <StatePanel title='No class heatmap yet'>
              Progress appears after students start topics.
            </StatePanel>
          )}
        </div>
      </section>
    </AdminSurface>
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
  return (
    <Sheet
      open={Boolean(studentID)}
      onOpenChange={(open) => {
        if (!open) {
          onOpenChange()
        }
      }}
    >
      <SheetContent className='w-full overflow-y-auto p-0 sm:max-w-[720px]'>
        <SheetHeader className='border-b border-border p-5'>
          <SheetTitle>{studentName ?? 'Student detail'}</SheetTitle>
          <SheetDescription>
            Progress, streaks, and chat history.
          </SheetDescription>
        </SheetHeader>
        <div className='p-5'>
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
            className='inline-block max-w-28 truncate border-0 bg-transparent p-0 align-middle font-[inherit] tracking-[inherit] text-inherit normal-case focus-visible:rounded-sm focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring/70'
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
      <span className='inline-flex min-h-7 items-center rounded-full px-3 py-1 text-sm font-medium text-muted-foreground'>
        --
      </span>
    )
  }

  return (
    <span
      className={cn(
        'inline-flex min-h-7 items-center rounded-full border px-3 py-1 text-sm font-semibold',
        getMasteryScoreTone(score),
      )}
    >
      {Math.round(score * 100)}%
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
      className='inline-flex w-full items-center justify-between gap-3 border-0 bg-transparent p-0 text-left font-[inherit] text-foreground hover:text-primary focus-visible:rounded-sm focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring/70'
      onClick={selectStudent}
      type='button'
    >
      <span className='inline-flex min-w-0 items-center gap-2'>
        <AttentionDot attention={attention} />
        <span className='truncate'>{studentName}</span>
      </span>
      <ChevronRightIcon aria-hidden='true' className='size-4 shrink-0' />
    </button>
  )
}

function AttentionDot({ attention }: { attention: StudentAttention }) {
  return (
    <span
      aria-label={
        attention.needsAttention
          ? `${attention.averageMastery}% average mastery`
          : 'No attention flag'
      }
      className={cn(
        'size-2.5 shrink-0 rounded-full',
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
    <Button disabled={isSending} onClick={sendNudge} size='sm' type='button'>
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

function getAttentionTone(attentionCount: number): string {
  return attentionCount > 0
    ? 'text-amber-700 dark:text-amber-300'
    : 'text-emerald-700 dark:text-emerald-300'
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
  averageMastery: number
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
  const averageMastery =
    scores.length > 0
      ? Math.round(
          (scores.reduce((sum, score) => sum + score, 0) / scores.length) * 100,
        )
      : 0

  return {
    averageMastery,
    needsAttention: scores.length > 0 && averageMastery < 50,
  }
}

const masteryScoreTones = [
  {
    className: 'border-emerald-200 bg-emerald-100 text-emerald-900',
    minimum: 0.8,
  },
  { className: 'border-lime-200 bg-lime-100 text-lime-900', minimum: 0.6 },
  { className: 'border-amber-200 bg-amber-100 text-amber-900', minimum: 0.4 },
] as const

const masteryGrades = [
  { label: 'A', minimum: 90 },
  { label: 'B', minimum: 80 },
  { label: 'C', minimum: 70 },
  { label: 'D', minimum: 60 },
  { label: 'E', minimum: 50 },
] as const

function noop() {}

function formatTopicLabel(topicID: string): string {
  return topicID
    .split('-')
    .filter(Boolean)
    .map((word) => `${word[0].toUpperCase()}${word.slice(1)}`)
    .join(' ')
}
