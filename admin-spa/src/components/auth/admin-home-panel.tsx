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
    <section className='min-h-svh overflow-y-auto bg-[var(--surface-secondary-default-hover)] px-4 text-[var(--text-default-heading)] sm:px-6 lg:px-8'>
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
    <header className='flex items-center border-b border-[var(--border-default)]'>
      <a
        className='inline-flex min-h-11 items-center gap-2.5 text-sm font-semibold tracking-tight text-[var(--text-default-heading)] no-underline outline-none focus-visible:rounded-lg focus-visible:ring-2 focus-visible:ring-[var(--border-primary-focus)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--surface-secondary-default-hover)]'
        href='/'
      >
        <span
          aria-hidden='true'
          className='grid size-8 place-items-center rounded-lg border border-[var(--border-primary-default)] bg-[var(--surface-tertiary-default)] text-xs font-bold text-[var(--text-primary-on-color)]'
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
      className='grid min-h-0 items-start py-10 min-[1040px]:items-center min-[1040px]:py-5 sm:py-14'
      aria-label='Admin sign in'
      id='sign-in'
    >
      <div className='grid w-full items-center gap-8 min-[1040px]:grid-cols-[minmax(0,1fr)_minmax(360px,432px)] min-[1040px]:gap-8 xl:gap-12'>
        <div className='hidden max-w-[560px] min-[1040px]:block'>
          <p className='m-0 max-w-[13ch] font-sans text-[64px] leading-[0.94] font-semibold tracking-[-0.025em] text-balance text-[var(--text-default-heading)] xl:text-[76px]'>
            See who needs help next.
          </p>
          <p className='mt-6 max-w-[42ch] text-base leading-7 text-pretty text-[var(--text-default-body)]'>
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
              title='Manage school operations'
              text='Keep classes, people, invites, budgets, and setup in one place.'
            />
            <EntryPromise
              Icon={ShieldCheckIcon}
              title='Protect admin access'
              text='Role-based permissions protect every admin action.'
            />
          </div>
        </div>

        <div className='mx-auto w-full max-w-[432px]'>
          <Card className='gap-5 rounded-[20px] bg-[var(--surface-general-default)] py-6 shadow-none ring-[var(--border-default)] sm:gap-6 sm:py-7'>
            <CardHeader className='gap-2 px-5 sm:px-7'>
              <CardTitle>
                <h1 className='m-0 font-sans !text-[32px] !leading-[1.08] font-semibold tracking-[-0.018em] text-balance text-[var(--text-default-heading)] sm:!text-[38px]'>
                  Welcome back.
                </h1>
              </CardTitle>
              <CardDescription className='max-w-[36ch] text-sm leading-6 text-pretty text-[var(--text-default-body)] sm:text-[15px]'>
                Sign in to see who needs support and choose the next action.
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
    <div className='flex items-center gap-3 rounded-xl border border-[var(--border-primary-default)] bg-[var(--surface-secondary-default-hover)] p-3 shadow-none'>
      <span className='grid size-10 shrink-0 place-items-center rounded-[10px] border border-[var(--border-primary-focus)] bg-[var(--surface-primary-default)] text-[var(--icon-tertiary-default)]'>
        <Icon aria-hidden='true' className='size-[18px]' />
      </span>
      <span className='min-w-0'>
        <span className='block text-sm font-semibold text-[var(--text-default-heading)]'>
          {title}
        </span>
        <span className='mt-0.5 block text-[13px] leading-5 text-[var(--text-default-body)]'>
          {text}
        </span>
      </span>
    </div>
  )
}
