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
    <section className='min-h-svh overflow-y-auto bg-background px-4 text-foreground sm:px-6'>
      <div className='mx-auto grid min-h-svh max-w-md grid-rows-[64px_1fr] font-inter'>
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
    <header className='flex items-center border-b border-border/70'>
      <div className='inline-flex items-center gap-2.5 text-sm font-semibold tracking-tight'>
        <span className='grid size-8 place-items-center rounded-lg bg-primary text-xs font-bold text-primary-foreground'>
          P
        </span>
        <span>P&AI Bot</span>
      </div>
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
      className='grid items-center py-10 sm:py-14'
      aria-label='Admin sign in'
      id='sign-in'
    >
      <div className='w-full'>
        <Card className='gap-6 rounded-3xl py-7 shadow-lg'>
          <CardHeader className='gap-2 px-5 text-center sm:px-7'>
            <CardTitle>
              <h1 className='m-0 font-roobert text-4xl leading-tight font-semibold tracking-tight text-balance sm:text-[2.75rem]'>
                Sign in to P&AI Bot
              </h1>
            </CardTitle>
            <CardDescription className='text-sm leading-relaxed text-pretty sm:text-base'>
              Enter your school email and password.
            </CardDescription>
          </CardHeader>
          <CardContent className='px-5 sm:px-7'>
            <LoginForm nextPath={nextPath} onAuthenticated={onAuthenticated} />
          </CardContent>
        </Card>
      </div>
    </section>
  )
}
