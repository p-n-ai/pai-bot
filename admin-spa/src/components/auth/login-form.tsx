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
              'Sign-in failed',
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
      className='flex flex-col gap-5'
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

        <AuthErrorAlert message={error} title='Sign-in failed.' />

        <PasswordLoginButton
          isDisabled={isPending || isGooglePending}
          isPending={isPending}
        />
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
          className='text-sm font-medium text-foreground'
          htmlFor={emailID}
        >
          Email
        </FieldLabel>
        <Input
          autoComplete='username'
          className='h-11 rounded-xl px-3 text-base sm:text-sm'
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
          className='text-sm font-medium text-foreground'
          htmlFor={passwordID}
        >
          Password
        </FieldLabel>
        <Input
          autoComplete='current-password'
          className='h-11 rounded-xl px-3 text-base sm:text-sm'
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
      className='h-11 rounded-xl px-4 font-semibold'
      disabled={isDisabled}
      type='submit'
    >
      {isPending ? (
        <Spinner aria-hidden='true' data-icon='inline-start' />
      ) : null}
      Sign in
    </Button>
  )
}

function LoginDivider({ visible }: { visible: boolean }) {
  if (!visible) {
    return null
  }

  return <FieldSeparator>or sign in with email</FieldSeparator>
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
      className='min-h-11 w-full rounded-xl font-semibold'
      disabled={isPending}
      onClick={onClick}
      type='button'
      variant='outline'
    >
      {isRedirecting ? (
        <Spinner aria-hidden='true' data-icon='inline-start' />
      ) : (
        <span
          aria-hidden='true'
          className='inline-grid size-4 place-items-center text-sm leading-none font-extrabold'
          data-icon='inline-start'
        >
          G
        </span>
      )}
      Sign in with Google
    </Button>
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
      className='rounded-xl border border-border bg-muted/50 p-3'
      data-disabled={disabled}
    >
      <div className='mb-3 flex gap-2 text-sm leading-5 text-muted-foreground'>
        <Building2Icon aria-hidden='true' className='mt-0.5 size-4 shrink-0' />
        <p className='m-0'>
          This email belongs to more than one school. Choose where you want to
          sign in.
        </p>
      </div>
      <FieldLabel className='text-xs font-semibold' htmlFor={id}>
        School
      </FieldLabel>
      <Select
        disabled={disabled}
        onValueChange={onChange}
        required
        value={value}
      >
        <SelectTrigger className='mt-1.5 h-11 rounded-xl' id={id}>
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
