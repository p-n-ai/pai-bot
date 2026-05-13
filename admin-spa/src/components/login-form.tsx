import { useCallback, useId, useState } from 'react'
import { Building2Icon, LockKeyholeIcon } from 'lucide-react'
import type { FormEvent } from 'react'

import type { AuthSession, SchoolChoice } from '@/lib/auth-types'
import { AuthErrorAlert } from '@/components/auth-error-alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { readAuthDisplayError } from '@/lib/auth-errors'
import { useInputValue } from '@/hooks/use-input-value'
import { useSubmitStatus } from '@/hooks/use-submit-status'
import { buildGoogleLoginURL, loginWithPassword } from '@/lib/auth-client'
import { getAuthErrorMessage } from '@/lib/auth-feedback'
import { isGoogleLoginEnabled } from '@/lib/login-settings'

interface LoginFormProps {
  authError?: string
  nextPath?: string
  onAuthenticated: (session: AuthSession) => void
}

export function LoginForm({
  authError,
  nextPath,
  onAuthenticated,
}: LoginFormProps) {
  const schoolID = useId()
  const email = useInputValue()
  const password = useInputValue()
  const [tenantID, setTenantID] = useState('')
  const [tenantChoices, setTenantChoices] = useState<Array<SchoolChoice>>([])
  const [isGooglePending, setGooglePending] = useState(false)
  const { beginSubmit, error, finishSubmit, isPending, setError } =
    useSubmitStatus(getAuthErrorMessage(authError))

  const submit = useCallback(
    (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault()
      beginSubmit()

      loginWithPassword({
        email: email.value.trim(),
        password: password.value,
        ...(tenantID ? { tenant_id: tenantID } : {}),
      })
        .then((result) => {
          if (result.kind === 'tenant_required') {
            setTenantChoices(result.tenant_choices)
            setError('Choose a school to continue signing in.')
            return
          }

          onAuthenticated(result.session)
        })
        .catch((caught: unknown) => {
          setError(
            readAuthDisplayError(
              caught,
              'Login failed',
              "We couldn't reach the sign-in service. Check your connection and try again.",
            ),
          )
        })
        .finally(() => {
          finishSubmit()
        })
    },
    [
      beginSubmit,
      email,
      finishSubmit,
      onAuthenticated,
      password,
      setError,
      tenantID,
    ],
  )

  const startGoogleLogin = useCallback(() => {
    if (isPending || isGooglePending) {
      return
    }

    setGooglePending(true)
    window.location.assign(buildGoogleLoginURL(nextPath))
  }, [isGooglePending, isPending, nextPath])

  const showGoogleLogin = isGoogleLoginEnabled()

  return (
    <form
      className='flex flex-col gap-4 font-inter'
      id='sign-in-form'
      onSubmit={submit}
    >
      <GoogleLoginButton
        isPending={isPending || isGooglePending}
        isRedirecting={isGooglePending}
        onClick={startGoogleLogin}
        visible={showGoogleLogin}
      />
      <GoogleLoginDivider visible={showGoogleLogin} />

      <LoginCredentialsFields email={email} password={password} />
      <SchoolSelect
        choices={tenantChoices}
        id={schoolID}
        onChange={setTenantID}
        value={tenantID}
      />

      <AuthErrorAlert message={error} title='Sign-in failed.' />

      <PasswordLoginButton
        isDisabled={isPending || isGooglePending}
        isPending={isPending}
      />
    </form>
  )
}

function LoginCredentialsFields({
  email,
  password,
}: {
  email: ReturnType<typeof useInputValue>
  password: ReturnType<typeof useInputValue>
}) {
  const emailID = useId()
  const passwordID = useId()

  return (
    <>
      <div className='flex flex-col gap-1.5'>
        <Label
          className='text-[12px] font-semibold tracking-[0.03em] text-slate-text'
          htmlFor={emailID}
        >
          Email
        </Label>
        <Input
          autoComplete='email'
          className='h-11 rounded-[10px] border-[#d9d3ca] bg-[#fffdf9] px-3 text-[14px] text-slate-text placeholder:text-ash-gray focus-visible:border-[#2f6f5b] focus-visible:ring-2 focus-visible:ring-[#bedbcf]'
          id={emailID}
          name='email'
          onChange={email.handleChange}
          placeholder='teacher@school.edu'
          required
          type='email'
          value={email.value}
        />
      </div>

      <div className='flex flex-col gap-1.5'>
        <div className='flex items-center justify-between gap-3'>
          <Label
            className='text-[12px] font-semibold tracking-[0.03em] text-slate-text'
            htmlFor={passwordID}
          >
            Password
          </Label>
          <span className='text-[12px] font-medium text-ash-gray'>
            Invite-issued account
          </span>
        </div>
        <Input
          autoComplete='current-password'
          className='h-11 rounded-[10px] border-[#d9d3ca] bg-[#fffdf9] px-3 text-[14px] text-slate-text placeholder:text-ash-gray focus-visible:border-[#2f6f5b] focus-visible:ring-2 focus-visible:ring-[#bedbcf]'
          id={passwordID}
          name='password'
          onChange={password.handleChange}
          placeholder='Enter your password'
          required
          type='password'
          value={password.value}
        />
      </div>
    </>
  )
}

function PasswordLoginButton({
  isDisabled,
  isPending,
}: {
  isDisabled: boolean
  isPending: boolean
}) {
  return (
    <Button
      className='mt-1 h-11 rounded-[10px] bg-[#17211b] px-4 text-[14px] font-semibold text-cloud-white shadow-[0_10px_24px_rgba(23,33,27,0.18)] hover:bg-[#235f72]'
      disabled={isDisabled}
      type='submit'
    >
      {isPending ? 'Signing in...' : 'Sign in'}
    </Button>
  )
}

function GoogleLoginDivider({ visible }: { visible: boolean }) {
  if (!visible) {
    return null
  }

  return (
    <div className='flex items-center gap-3 text-[11px] font-bold tracking-[0.12em] text-ash-gray uppercase'>
      <span className='h-px flex-1 bg-stone-border' aria-hidden='true' />
      <span>or use email</span>
      <span className='h-px flex-1 bg-stone-border' aria-hidden='true' />
    </div>
  )
}

function GoogleLoginButton({
  isRedirecting,
  isPending,
  onClick,
  visible,
}: {
  isRedirecting: boolean
  isPending: boolean
  onClick: () => void
  visible: boolean
}) {
  if (!visible) {
    return null
  }

  return (
    <Button
      className='min-h-11 w-full rounded-[10px] border-[#d9d3ca] bg-cloud-white text-[14px] font-semibold text-slate-text shadow-subtle hover:bg-[#f7f5ef]'
      disabled={isPending}
      onClick={onClick}
      type='button'
      variant='outline'
    >
      <span
        aria-hidden='true'
        className='inline-grid size-4 place-items-center text-sm leading-none font-extrabold text-[#4285f4]'
      >
        G
      </span>
      {isRedirecting ? 'Redirecting to Google...' : 'Continue with Google'}
    </Button>
  )
}

function SchoolSelect({
  choices,
  id,
  onChange,
  value,
}: {
  choices: Array<SchoolChoice>
  id: string
  onChange: (value: string) => void
  value: string
}) {
  if (choices.length === 0) {
    return null
  }

  return (
    <div className='rounded-[12px] border border-[#d9d3ca] bg-[#f7fbf8] p-3'>
      <div className='mb-3 flex gap-2 text-[13px] leading-5 text-[#445c4d]'>
        <Building2Icon aria-hidden='true' className='mt-0.5 size-4 shrink-0' />
        <p className='m-0'>
          This email belongs to more than one school. Choose the workspace for
          this session.
        </p>
      </div>
      <Label className='text-[12px] font-semibold' htmlFor={id}>
        School
      </Label>
      <Select onValueChange={onChange} required value={value}>
        <SelectTrigger className='mt-1.5 h-11 rounded-[10px]' id={id}>
          <SelectValue placeholder='Choose school' />
        </SelectTrigger>
        <SelectContent>
          {choices.map((school) => (
            <SelectItem key={school.tenant_id} value={school.tenant_id}>
              {school.tenant_name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      <p className='mt-2 mb-0 flex items-start gap-2 text-[12px] leading-5 text-ash-gray'>
        <LockKeyholeIcon aria-hidden='true' className='mt-0.5 size-3.5' />
        Your role and tenant access are checked again by the backend.
      </p>
    </div>
  )
}
