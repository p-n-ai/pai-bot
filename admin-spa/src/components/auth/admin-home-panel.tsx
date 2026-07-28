import { GraduationCapIcon, SchoolIcon, ShieldCheckIcon } from 'lucide-react'

import type { AuthState } from '@/auth-provider'
import type { AuthSession } from '@/lib/auth-types'
import { LoginForm } from '@/components/auth/login-form'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'

export function AdminHomePanel({
  auth,
  nextPath,
  onAuthenticated = noopAuthenticated,
}: {
  auth: AuthState
  nextPath?: string
  onAuthenticated?: (session: AuthSession) => void
}) {
  if (auth.status === 'authenticated') {
    return null
  }

  return (
    <section className='min-h-svh overflow-y-auto bg-[#f7f5ef] px-4 text-slate-text sm:px-6 lg:px-8'>
      <div className='mx-auto grid min-h-svh max-w-[1180px] grid-rows-[56px_minmax(0,1fr)] font-sans sm:grid-rows-[64px_minmax(0,1fr)]'>
        <LoginHeader />
        <AdminEntryPanel
          nextPath={nextPath}
          onAuthenticated={onAuthenticated}
        />
      </div>
    </section>
  )
}

function noopAuthenticated() {}

function LoginHeader() {
  return (
    <header className='flex items-center border-b border-stone-border/80'>
      <a
        className='inline-flex min-h-11 items-center gap-2.5 text-sm font-semibold tracking-tight text-slate-text no-underline outline-none focus-visible:rounded-lg focus-visible:ring-2 focus-visible:ring-[#2f6f5b] focus-visible:ring-offset-2 focus-visible:ring-offset-[#f7f5ef]'
        href='/'
      >
        <span
          aria-hidden='true'
          className='grid size-8 place-items-center rounded-lg bg-[#14251c] text-xs font-bold text-cloud-white'
        >
          P
        </span>
        <span>P&amp;AI Bot</span>
      </a>
    </header>
  )
}

function AdminEntryPanel({
  nextPath,
  onAuthenticated,
}: {
  nextPath?: string
  onAuthenticated: (session: AuthSession) => void
}) {
  return (
    <section
      className='grid min-h-0 items-center py-6 sm:py-8 lg:py-5'
      aria-label='Admin sign in'
      id='sign-in'
    >
      <div className='grid w-full items-center gap-8 lg:grid-cols-[minmax(0,1fr)_minmax(360px,432px)] lg:gap-12'>
        <div className='hidden max-w-[560px] lg:block'>
          <p className='m-0 max-w-[11ch] font-sans text-[64px] leading-[0.94] font-semibold tracking-[-0.025em] text-balance text-slate-text xl:text-[76px]'>
            See who needs help next.
          </p>
          <p className='mt-6 max-w-[42ch] text-base leading-7 text-pretty text-ash-gray'>
            Move from class progress to the right next action without losing the
            learner behind the data.
          </p>
          <div className='mt-8 grid max-w-[520px] gap-3'>
            <EntryPromise
              Icon={GraduationCapIcon}
              title='Guide the next lesson'
              text='See class mastery and student detail in one place.'
            />
            <EntryPromise
              Icon={SchoolIcon}
              title='Run the whole workspace'
              text='Manage classes, people, invites, budgets, and setup.'
            />
            <EntryPromise
              Icon={ShieldCheckIcon}
              title='Keep access accountable'
              text='Backend role checks protect every admin action.'
            />
          </div>
        </div>

        <div className='mx-auto w-full max-w-[432px]'>
          <Card className='gap-5 rounded-[20px] border-[#dfd8cd] bg-cloud-white py-6 shadow-[0_22px_70px_rgba(31,28,24,0.10)] sm:gap-6 sm:py-7'>
            <CardHeader className='gap-2 px-5 sm:px-7'>
              <CardTitle>
                <h1 className='m-0 font-sans text-[32px] leading-[1.08] font-semibold tracking-[-0.018em] text-balance text-slate-text sm:text-[38px]'>
                  Welcome back.
                </h1>
              </CardTitle>
              <CardDescription className='max-w-[36ch] text-sm leading-6 text-pretty text-ash-gray sm:text-[15px]'>
                Sign in to review progress and support the right students.
              </CardDescription>
            </CardHeader>
            <CardContent className='px-5 sm:px-7'>
              <LoginForm
                nextPath={nextPath}
                onAuthenticated={onAuthenticated}
              />
            </CardContent>
          </Card>
        </div>
      </div>
    </section>
  )
}

function EntryPromise({
  Icon,
  title,
  text,
}: {
  Icon: typeof GraduationCapIcon
  title: string
  text: string
}) {
  return (
    <div className='flex items-center gap-3 rounded-xl border border-[#ded6ca] bg-[#fffdf9]/80 p-3 shadow-[0_1px_0_rgba(31,28,24,0.04)]'>
      <span className='grid size-10 shrink-0 place-items-center rounded-[10px] bg-[#14251c] text-cloud-white'>
        <Icon aria-hidden='true' className='size-[18px]' />
      </span>
      <span className='min-w-0'>
        <span className='block text-sm font-semibold text-slate-text'>
          {title}
        </span>
        <span className='mt-0.5 block text-[13px] leading-5 text-ash-gray'>
          {text}
        </span>
      </span>
    </div>
  )
}
