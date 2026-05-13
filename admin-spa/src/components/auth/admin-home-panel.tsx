import {
  ArrowRightIcon,
  GraduationCapIcon,
  SchoolIcon,
  ShieldCheckIcon,
} from 'lucide-react'

import type { AuthState } from '@/auth-provider'
import type { AuthSession } from '@/lib/auth-types'
import { LoginForm } from '@/components/auth/login-form'

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
    <section className='h-svh overflow-hidden bg-[#f7f5ef] px-4 text-slate-text sm:px-6 lg:px-8'>
      <div className='mx-auto grid h-svh max-w-[1180px] grid-rows-[64px_minmax(0,1fr)] font-inter'>
        <LoginNav />
        <AdminEntryPanel
          nextPath={nextPath}
          onAuthenticated={onAuthenticated}
        />
      </div>
    </section>
  )
}

function noopAuthenticated() {}

function LoginNav() {
  return (
    <nav
      className='flex min-h-0 items-center justify-between gap-6 border-b border-stone-border/80 font-inter'
      aria-label='Admin entry'
    >
      <a
        className='inline-flex items-center gap-2 text-[15px] font-semibold tracking-[0.003em] text-slate-text no-underline'
        href='/'
      >
        <span className='grid size-8 place-items-center rounded-[7px] bg-slate-text text-[13px] font-bold text-cloud-white'>
          P
        </span>
        <span>P&AI Bot</span>
      </a>
    </nav>
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
      className='grid min-h-0 items-center py-5'
      aria-label='Admin sign in'
      id='sign-in'
    >
      <div className='grid w-full items-center gap-7 lg:grid-cols-[minmax(0,1fr)_minmax(360px,432px)] lg:gap-12'>
        <div className='hidden max-w-[560px] lg:block'>
          <p className='m-0 max-w-[10ch] font-roobert text-[64px] leading-[0.92] font-semibold tracking-[-0.025em] text-slate-text xl:text-[76px]'>
            Start with the right class.
          </p>
          <div className='mt-8 grid max-w-[520px] gap-3'>
            <EntryPromise
              Icon={GraduationCapIcon}
              text='Teachers see class mastery and student detail first.'
            />
            <EntryPromise
              Icon={SchoolIcon}
              text='Admins manage classes, users, invites, budgets, exports, and setup.'
            />
            <EntryPromise
              Icon={ShieldCheckIcon}
              text='One session cookie; backend RBAC still gates every admin API.'
            />
          </div>
        </div>

        <div className='mx-auto w-full max-w-[432px]'>
          <div className='mb-5 text-center lg:text-left'>
            <h1 className='m-0 font-roobert text-[32px] leading-[1.08] font-semibold tracking-[-0.018em] text-slate-text sm:text-[38px] lg:text-[34px]'>
              Welcome back.
            </h1>
            <p className='mx-auto mt-3 mb-0 max-w-[34ch] text-[14px] leading-[1.56] tracking-[0.003em] text-ash-gray sm:text-[15px] lg:mx-0'>
              Review classes and decide who needs help next.
            </p>
          </div>

          <div className='grid gap-4 rounded-[18px] border border-[#dfd8cd] bg-cloud-white p-5 shadow-[0_22px_70px_rgba(31,28,24,0.10)] sm:p-6'>
            <LoginForm nextPath={nextPath} onAuthenticated={onAuthenticated} />
          </div>
        </div>
      </div>
    </section>
  )
}

function EntryPromise({
  Icon,
  text,
}: {
  Icon: typeof GraduationCapIcon
  text: string
}) {
  return (
    <div className='flex items-center gap-3 rounded-[12px] border border-[#ded6ca] bg-[#fffdf9]/75 p-3 text-[14px] leading-5 text-[#39342f] shadow-[0_1px_0_rgba(31,28,24,0.04)]'>
      <span className='grid size-9 shrink-0 place-items-center rounded-[9px] bg-[#14251c] text-cloud-white'>
        <Icon aria-hidden='true' className='size-4' />
      </span>
      <span>{text}</span>
      <ArrowRightIcon
        aria-hidden='true'
        className='ml-auto size-4 shrink-0 text-ash-gray'
      />
    </div>
  )
}
