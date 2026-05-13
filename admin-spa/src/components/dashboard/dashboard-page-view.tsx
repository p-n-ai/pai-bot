import type { DashboardProgressResult } from '@/lib/dashboard-progress'
import { AdminPageSection } from '@/components/shared/admin-page-section'
import { AdminSurface } from '@/components/shared/admin-surface'
import { DashboardReady } from '@/components/dashboard/dashboard-ready'
import { StatePanel } from '@/components/shared/state-panel'
import { Card, CardHeader } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'

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
  nudgeMessage: string
  onCloseStudent: () => void
  onNudge: (studentID: string, studentName: string) => void
  onSelectStudent: (studentID: string) => void
  selectedStudentID: string | undefined
  sendingStudentID: string
  state: DashboardState
}

export function DashboardPageView({
  nudgeMessage,
  onCloseStudent,
  onNudge,
  onSelectStudent,
  selectedStudentID,
  sendingStudentID,
  state,
}: DashboardPageViewProps) {
  return (
    <AdminPageSection
      description='Track who needs support today across the class.'
      eyebrow='Dashboard'
      hideEyebrow
      title='Dashboard'
    >
      <DashboardContent
        nudgeMessage={nudgeMessage}
        onCloseStudent={onCloseStudent}
        onNudge={onNudge}
        onSelectStudent={onSelectStudent}
        selectedStudentID={selectedStudentID}
        sendingStudentID={sendingStudentID}
        state={state}
      />
    </AdminPageSection>
  )
}

function DashboardContent({
  nudgeMessage,
  onCloseStudent,
  onNudge,
  onSelectStudent,
  selectedStudentID,
  sendingStudentID,
  state,
}: DashboardPageViewProps) {
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
      onSelectStudent={onSelectStudent}
      progress={state.result.progress}
      selectedStudentID={selectedStudentID}
      sendingStudentID={sendingStudentID}
    />
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
              {Array.from({ length: 5 }).map((_, index) => (
                <Skeleton className='h-4' key={index} />
              ))}
            </div>
            <div className='flex flex-col gap-2'>
              {Array.from({ length: 4 }).map((_, rowIndex) => (
                <div
                  className='grid grid-cols-[1.4fr_repeat(3,1fr)_0.8fr] gap-2 rounded-lg bg-muted/40 p-3'
                  key={rowIndex}
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
