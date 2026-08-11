import { useCallback, useEffect, useId, useState } from 'react'
import { Building2Icon } from 'lucide-react'
import type { FormEvent } from 'react'

import type { AuthSession, SchoolChoice } from '@/lib/auth-types'
import { AuthErrorAlert } from '@/components/shared/auth-error-alert'
import { Button } from '@/components/ui/button'
import {
  Field,
  FieldGroup,
  FieldLabel,
  FieldSeparator,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Spinner } from '@/components/ui/spinner'
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
import {
  buildGoogleLoginURL,
  loginWithPassword,
  readAuthCapabilities,
} from '@/lib/auth-client'
import { getAuthErrorMessage } from '@/lib/auth-feedback'
import { cn } from '@/lib/utils'

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
  const [showGoogleLogin, setShowGoogleLogin] = useState(false)
  const { beginSubmit, error, finishSubmit, isPending, setError } =
    useSubmitStatus(getAuthErrorMessage(authError))

  useEffect(() => {
    let active = true

    readAuthCapabilities()
      .then((capabilities) => {
        if (active) {
          setShowGoogleLogin(capabilities.google_login)
        }
      })
      .catch(() => {
        if (active) {
          setShowGoogleLogin(false)
        }
      })

    return () => {
      active = false
    }
  }, [])

  const submit = useCallback(
    (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault()

      if (tenantChoices.length > 0 && !tenantID) {
        setError('Choose a school to continue signing in.')
        return
      }

      beginSubmit()

      loginWithPassword({
        email: email.value.trim(),
        password: password.value,
        ...(tenantID ? { tenant_id: tenantID } : {}),
      })
        .then((result) => {
          if (result.kind === 'tenant_required') {
            setTenantChoices(result.tenant_choices)
            setError('')
            return
          }

          onAuthenticated(result.session)
        })
        .catch((caught: unknown) => {
          setError(
            readAuthDisplayError(
              caught,
              'Unable to sign in',
              'Unable to reach the sign-in service. Check your connection and try again.',
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
      tenantChoices.length,
    ],
  )

  const startGoogleLogin = useCallback(() => {
    if (isPending || isGooglePending) {
      return
    }

    setGooglePending(true)
    window.location.assign(buildGoogleLoginURL(nextPath))
  }, [isGooglePending, isPending, nextPath])

  return (
    <form
      aria-busy={isPending || isGooglePending}
      className='flex flex-col gap-5 font-sans'
      id='sign-in-form'
      onSubmit={submit}
    >
      <FieldGroup>
        <GoogleLoginButton
          isPending={isPending || isGooglePending}
          isRedirecting={isGooglePending}
          onClick={startGoogleLogin}
          visible={showGoogleLogin}
        />
        <LoginDivider visible={showGoogleLogin} />

        <LoginCredentialsFields
          disabled={isPending || isGooglePending}
          email={email}
          password={password}
        />
        <SchoolSelect
          choices={tenantChoices}
          disabled={isPending || isGooglePending}
          id={schoolID}
          onChange={setTenantID}
          value={tenantID}
        />

        <AuthErrorAlert
          message={error}
          title={
            tenantChoices.length > 0 && !tenantID
              ? 'Choose a school'
              : 'Unable to sign in'
          }
        />

        <div className='grid gap-2'>
          <PasswordLoginButton
            isDisabled={isPending || isGooglePending}
            isPending={isPending}
          />
          <p className='m-0 text-center text-xs leading-5 text-[var(--text-default-body)]'>
            Need help signing in? Contact your platform administrator.
          </p>
        </div>
      </FieldGroup>
    </form>
  )
}

function LoginCredentialsFields({
  disabled,
  email,
  password,
}: {
  disabled: boolean
  email: ReturnType<typeof useInputValue>
  password: ReturnType<typeof useInputValue>
}) {
  const emailID = useId()
  const passwordID = useId()

  return (
    <>
      <Field data-disabled={disabled}>
        <FieldLabel
          className='text-[13px] font-semibold tracking-[0.01em] text-[var(--text-default-heading)]'
          htmlFor={emailID}
        >
          Email
        </FieldLabel>
        <Input
          autoComplete='username'
          className='h-11 rounded-[10px] border-[var(--border-control-default)] bg-[var(--surface-general-default)] px-3 text-base text-[var(--text-default-heading)] placeholder:text-[var(--text-default-secondary)] focus-visible:border-[var(--border-primary-focus)] focus-visible:ring-2 focus-visible:ring-[var(--border-primary-focus)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--surface-general-default)] sm:text-sm'
          disabled={disabled}
          id={emailID}
          name='email'
          onChange={email.handleChange}
          placeholder='name@school.edu'
          required
          spellCheck={false}
          type='email'
          value={email.value}
        />
      </Field>

      <Field data-disabled={disabled}>
        <FieldLabel
          className='text-[13px] font-semibold tracking-[0.01em] text-[var(--text-default-heading)]'
          htmlFor={passwordID}
        >
          Password
        </FieldLabel>
        <Input
          autoComplete='current-password'
          className='h-11 rounded-[10px] border-[var(--border-control-default)] bg-[var(--surface-general-default)] px-3 text-base text-[var(--text-default-heading)] placeholder:text-[var(--text-default-secondary)] focus-visible:border-[var(--border-primary-focus)] focus-visible:ring-2 focus-visible:ring-[var(--border-primary-focus)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--surface-general-default)] sm:text-sm'
          disabled={disabled}
          id={passwordID}
          name='password'
          onChange={password.handleChange}
          placeholder='Enter your password'
          required
          type='password'
          value={password.value}
        />
      </Field>
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
      className={cn(
        'h-11 rounded-[10px] border border-[var(--border-primary-focus)] bg-[var(--surface-tertiary-default)] px-4 text-sm font-semibold text-[var(--text-primary-on-color)] shadow-none hover:border-[var(--border-secondary-focus)] hover:bg-[var(--surface-secondary-default)] hover:text-[var(--text-tertiary-default)] focus-visible:ring-2 focus-visible:ring-[var(--border-primary-focus)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--surface-general-default)] active:border-[var(--border-primary-focus)] active:bg-[var(--surface-primary-default)] active:text-[var(--text-default-heading)] disabled:opacity-100',
        isPending
          ? 'disabled:border-[var(--border-primary-focus)] disabled:bg-[var(--surface-tertiary-default)] disabled:text-[var(--text-primary-on-color)]'
          : 'disabled:border-[var(--border-disabled-disabled)] disabled:bg-[var(--surface-disabled-primary)] disabled:text-[var(--text-disabled-default)]',
      )}
      disabled={isDisabled}
      type='submit'
    >
      {isPending ? (
        <Spinner aria-hidden='true' data-icon='inline-start' />
      ) : null}
      {isPending ? 'Signing in…' : 'Sign in'}
    </Button>
  )
}

function LoginDivider({ visible }: { visible: boolean }) {
  if (!visible) {
    return null
  }

  return (
    <FieldSeparator className='text-xs text-[var(--text-default-body)]'>
      or sign in with email
    </FieldSeparator>
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
      className='min-h-11 w-full rounded-[10px] border-[var(--border-control-default)] bg-[var(--surface-general-default)] text-sm font-semibold text-[var(--text-tertiary-default)] shadow-none hover:bg-[var(--surface-secondary-default-subtle)] hover:text-[var(--text-tertiary-default)] focus-visible:ring-2 focus-visible:ring-[var(--border-primary-focus)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--surface-general-default)]'
      disabled={isPending}
      onClick={onClick}
      type='button'
      variant='outline'
    >
      {isRedirecting ? (
        <Spinner aria-hidden='true' data-icon='inline-start' />
      ) : (
        <GoogleMark />
      )}
      {isRedirecting ? 'Opening Google…' : 'Sign in with Google'}
    </Button>
  )
}

function GoogleMark() {
  return (
    <svg
      aria-hidden='true'
      data-icon='inline-start'
      data-testid='google-mark'
      viewBox='0 0 18 18'
    >
      <path
        d='M17.64 9.205c0-.638-.057-1.252-.164-1.841H9v3.482h4.844a4.14 4.14 0 0 1-1.797 2.715v2.259h2.909c1.702-1.567 2.684-3.874 2.684-6.615Z'
        fill='#4285F4'
      />
      <path
        d='M9 18c2.43 0 4.467-.806 5.956-2.18l-2.909-2.259c-.806.54-1.835.859-3.047.859-2.344 0-4.328-1.585-5.037-3.714H.956v2.333A9 9 0 0 0 9 18Z'
        fill='#34A853'
      />
      <path
        d='M3.963 10.706A5.41 5.41 0 0 1 3.682 9c0-.592.102-1.168.281-1.706V4.961H.956A9 9 0 0 0 0 9c0 1.45.347 2.824.956 4.039l3.007-2.333Z'
        fill='#FBBC05'
      />
      <path
        d='M9 3.58c1.321 0 2.507.454 3.44 1.345l2.581-2.581C13.463.892 11.426 0 9 0A9 9 0 0 0 .956 4.961l3.007 2.333C4.672 5.165 6.656 3.58 9 3.58Z'
        fill='#EA4335'
      />
    </svg>
  )
}

function SchoolSelect({
  choices,
  disabled,
  id,
  onChange,
  value,
}: {
  choices: Array<SchoolChoice>
  disabled: boolean
  id: string
  onChange: (value: string) => void
  value: string
}) {
  if (choices.length === 0) {
    return null
  }

  return (
    <Field
      className='rounded-xl border border-[var(--border-primary-default)] bg-[var(--surface-primary-default-subtle)] p-3'
      data-disabled={disabled}
    >
      <div
        className='mb-3 flex gap-2 text-sm leading-5 text-[var(--text-default-body)]'
        role='status'
      >
        <Building2Icon aria-hidden='true' className='mt-0.5 size-4 shrink-0' />
        <p className='m-0'>
          This email belongs to more than one school. Choose where you want to
          sign in.
        </p>
      </div>
      <FieldLabel
        className='text-xs font-semibold text-[var(--text-default-heading)]'
        htmlFor={id}
      >
        School
      </FieldLabel>
      <Select
        disabled={disabled}
        onValueChange={onChange}
        required
        value={value}
      >
        <SelectTrigger
          className='mt-1.5 h-11 rounded-[10px] border-[var(--border-control-default)] bg-[var(--surface-general-default)] focus-visible:border-[var(--border-primary-focus)] focus-visible:ring-2 focus-visible:ring-[var(--border-primary-focus)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--surface-primary-default-subtle)]'
          id={id}
        >
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
    </Field>
  )
}
