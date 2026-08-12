import { useCallback, useId, useRef, useState } from 'react'
import type { PandaiIconComponent } from '@/components/ui/pandai-icons'
import type { ChangeEventHandler, FormEvent, RefObject } from 'react'

import type { AuthSession } from '@/lib/auth-types'
import {
  CircleCheckIcon,
  MailCheckIcon,
  MailWarningIcon,
  ShieldCheckIcon,
} from '@/components/ui/pandai-icons'
import { AuthErrorAlert } from '@/components/shared/auth-error-alert'
import { Button } from '@/components/ui/button'
import {
  Field,
  FieldDescription,
  FieldError,
  FieldGroup,
  FieldLabel,
} from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Spinner } from '@/components/ui/spinner'
import { acceptInvite } from '@/lib/auth-client'
import { useInputValue } from '@/hooks/use-input-value'
import { useSubmitStatus } from '@/hooks/use-submit-status'

interface InviteActivationFormProps {
  token: string
  onAuthenticated: (session: AuthSession) => void
}

interface ActivationFieldErrors {
  name: string
  password: string
}

const minimumPasswordLength = 12
const noFieldErrors: ActivationFieldErrors = { name: '', password: '' }

export function InviteActivationForm({
  onAuthenticated,
  token,
}: InviteActivationFormProps) {
  const headingID = useId()
  const setupHeadingID = useId()
  const nameID = useId()
  const nameErrorID = useId()
  const passwordID = useId()
  const passwordDescriptionID = useId()
  const passwordErrorID = useId()
  const nameRef = useRef<HTMLInputElement>(null)
  const passwordRef = useRef<HTMLInputElement>(null)
  const name = useInputValue()
  const password = useInputValue()
  const handleNameValueChange = name.handleChange
  const handlePasswordValueChange = password.handleChange
  const [fieldErrors, setFieldErrors] =
    useState<ActivationFieldErrors>(noFieldErrors)
  const tokenMissing = token.trim() === ''
  const { beginSubmit, error, finishSubmit, isPending, setError } =
    useSubmitStatus()

  const handleNameChange: ChangeEventHandler<HTMLInputElement> = useCallback(
    (event) => {
      handleNameValueChange(event)
      setFieldErrors((current) =>
        current.name ? { ...current, name: '' } : current,
      )
    },
    [handleNameValueChange],
  )

  const handlePasswordChange: ChangeEventHandler<HTMLInputElement> =
    useCallback(
      (event) => {
        handlePasswordValueChange(event)
        setFieldErrors((current) =>
          current.password ? { ...current, password: '' } : current,
        )
      },
      [handlePasswordValueChange],
    )

  const submit = useCallback(
    (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault()

      const nextFieldErrors = readActivationFieldErrors(
        name.value,
        password.value,
      )
      setFieldErrors(nextFieldErrors)

      if (nextFieldErrors.name) {
        nameRef.current?.focus()
        return
      }

      if (nextFieldErrors.password) {
        passwordRef.current?.focus()
        return
      }

      beginSubmit()

      acceptInvite({
        token,
        name: name.value.trim(),
        password: password.value,
      })
        .then(onAuthenticated)
        .catch((caught: unknown) => {
          setError(readInviteActivationError(caught))
        })
        .finally(() => {
          finishSubmit()
        })
    },
    [
      beginSubmit,
      finishSubmit,
      name.value,
      onAuthenticated,
      password.value,
      setError,
      token,
    ],
  )

  return (
    <div className='min-h-svh bg-[var(--surface-secondary-default-hover)] px-4 py-6 text-[var(--text-default-heading)] sm:px-6 sm:py-8 lg:px-8'>
      <section
        aria-labelledby={headingID}
        className='mx-auto grid min-h-[calc(100svh-3rem)] w-full max-w-[1120px] content-center gap-6 min-[1040px]:grid-cols-[minmax(0,1fr)_minmax(22rem,26rem)] min-[1040px]:grid-rows-[auto_auto] min-[1040px]:gap-x-12 min-[1040px]:gap-y-6 sm:min-h-[calc(100svh-4rem)]'
      >
        <InviteActivationHeader headingID={headingID} />

        {tokenMissing ? (
          <MissingInviteLinkState />
        ) : (
          <InviteActivationCard
            error={error}
            fieldErrors={fieldErrors}
            handleNameChange={handleNameChange}
            handlePasswordChange={handlePasswordChange}
            isPending={isPending}
            name={name}
            nameErrorID={nameErrorID}
            nameID={nameID}
            nameRef={nameRef}
            password={password}
            passwordDescriptionID={passwordDescriptionID}
            passwordErrorID={passwordErrorID}
            passwordID={passwordID}
            passwordRef={passwordRef}
            setupHeadingID={setupHeadingID}
            submit={submit}
          />
        )}

        {!tokenMissing && <InviteActivationTrustCues />}
      </section>
    </div>
  )
}

function InviteActivationHeader({ headingID }: { headingID: string }) {
  return (
    <header className='grid gap-3 min-[1040px]:col-start-1 min-[1040px]:row-start-1 min-[1040px]:self-end'>
      <p className='m-0 inline-flex items-center gap-2 text-xs font-bold tracking-[0.14em] text-[var(--text-default-body)] uppercase'>
        <ShieldCheckIcon
          aria-hidden='true'
          className='size-5'
          strokeWidth={1.5}
        />
        Invite activation
      </p>
      <h1
        className='max-w-[16ch] text-[clamp(2rem,3.2vw,2.25rem)] leading-[1.1] font-semibold tracking-[-0.025em] text-balance'
        id={headingID}
      >
        Accept your invite
      </h1>
      <p className='m-0 max-w-[50ch] text-base leading-6 text-pretty text-[var(--text-default-body)]'>
        Create your account once, then use your new password for future
        sign-ins.
      </p>
    </header>
  )
}

function InviteActivationCard({
  error,
  fieldErrors,
  handleNameChange,
  handlePasswordChange,
  isPending,
  name,
  nameErrorID,
  nameID,
  nameRef,
  password,
  passwordDescriptionID,
  passwordErrorID,
  passwordID,
  passwordRef,
  setupHeadingID,
  submit,
}: {
  error: string
  fieldErrors: ActivationFieldErrors
  handleNameChange: ChangeEventHandler<HTMLInputElement>
  handlePasswordChange: ChangeEventHandler<HTMLInputElement>
  isPending: boolean
  name: ReturnType<typeof useInputValue>
  nameErrorID: string
  nameID: string
  nameRef: RefObject<HTMLInputElement | null>
  password: ReturnType<typeof useInputValue>
  passwordDescriptionID: string
  passwordErrorID: string
  passwordID: string
  passwordRef: RefObject<HTMLInputElement | null>
  setupHeadingID: string
  submit: (event: FormEvent<HTMLFormElement>) => void
}) {
  return (
    <section
      aria-labelledby={setupHeadingID}
      className='rounded-[20px] border border-[var(--border-primary-default)] bg-[var(--surface-general-default)] p-5 min-[1040px]:col-start-2 min-[1040px]:row-span-2 min-[1040px]:row-start-1 min-[1040px]:self-center sm:p-7'
    >
      <div className='grid gap-2'>
        <h2
          className='text-2xl leading-tight font-semibold tracking-[-0.025em] text-balance text-[var(--text-default-heading)]'
          id={setupHeadingID}
        >
          Set up your account
        </h2>
        <p className='m-0 text-sm leading-6 text-pretty text-[var(--text-default-body)]'>
          Add your name and create the password you’ll use to sign in.
        </p>
      </div>

      <form
        aria-busy={isPending}
        className='mt-6 flex flex-col gap-5'
        noValidate
        onSubmit={submit}
      >
        <InviteActivationFields
          disabled={isPending}
          errors={fieldErrors}
          handleNameChange={handleNameChange}
          handlePasswordChange={handlePasswordChange}
          name={name}
          nameErrorID={nameErrorID}
          nameID={nameID}
          nameRef={nameRef}
          password={password}
          passwordDescriptionID={passwordDescriptionID}
          passwordErrorID={passwordErrorID}
          passwordID={passwordID}
          passwordRef={passwordRef}
        />

        <AuthErrorAlert message={error} title='Unable to accept invite' />

        <Button
          className='w-full focus-visible:ring-[var(--border-focus-strong)]'
          disabled={isPending}
          size='lg'
          type='submit'
        >
          {isPending && <Spinner aria-hidden='true' />}
          Accept invite
        </Button>
      </form>

      <Button
        asChild
        className='mt-2 px-0 text-[var(--text-tertiary-default)] underline underline-offset-4 focus-visible:ring-[var(--border-focus-strong)]'
        size='lg'
        variant='link'
      >
        <a href='/'>Back to sign in</a>
      </Button>
    </section>
  )
}

function MissingInviteLinkState() {
  const headingID = useId()

  return (
    <section
      aria-labelledby={headingID}
      className='grid gap-6 rounded-[20px] border border-[var(--border-primary-default)] bg-[var(--surface-general-default)] p-5 min-[1040px]:col-start-2 min-[1040px]:row-span-2 min-[1040px]:row-start-1 min-[1040px]:self-center sm:p-7'
    >
      <span className='grid size-11 place-items-center rounded-xl bg-[var(--surface-primary-default-subtle)] text-[var(--icon-tertiary-default)]'>
        <MailWarningIcon
          aria-hidden='true'
          className='size-5'
          strokeWidth={2}
        />
      </span>
      <div className='grid gap-2'>
        <h2
          className='text-2xl leading-tight font-semibold tracking-[-0.025em] text-balance'
          id={headingID}
        >
          This invite link is incomplete
        </h2>
        <p className='m-0 text-sm leading-6 text-pretty text-[var(--text-default-body)]'>
          Open the full invite link from your email. If it still doesn’t work,
          ask your school administrator for a new invite.
        </p>
      </div>
      <Button
        asChild
        className='focus-visible:ring-[var(--border-focus-strong)]'
        size='lg'
      >
        <a href='/'>Back to sign in</a>
      </Button>
    </section>
  )
}

function InviteActivationTrustCues() {
  return (
    <section
      aria-label='What happens next'
      className='grid gap-3 min-[1040px]:col-start-1 min-[1040px]:row-start-2 min-[1040px]:self-start sm:grid-cols-2'
    >
      <TrustCue
        description='This link works once for one school account.'
        Icon={MailCheckIcon}
        title='One-time link'
      />
      <TrustCue
        description='After setup, you’ll continue to the right workspace.'
        Icon={CircleCheckIcon}
        title='Automatic sign-in'
      />
    </section>
  )
}

function TrustCue({
  description,
  Icon,
  title,
}: {
  Icon: PandaiIconComponent
  description: string
  title: string
}) {
  return (
    <article className='grid grid-cols-[auto_1fr] gap-x-3 gap-y-1 rounded-xl bg-[var(--surface-primary-default-subtle)] p-4'>
      <span className='row-span-2 inline-flex pt-0.5 text-[var(--icon-tertiary-default)]'>
        <Icon aria-hidden='true' className='size-5' strokeWidth={2} />
      </span>
      <strong className='leading-5'>{title}</strong>
      <span className='text-sm leading-5 text-[var(--text-default-body)]'>
        {description}
      </span>
    </article>
  )
}

function InviteActivationFields({
  disabled,
  errors,
  handleNameChange,
  handlePasswordChange,
  name,
  nameErrorID,
  nameID,
  nameRef,
  password,
  passwordDescriptionID,
  passwordErrorID,
  passwordID,
  passwordRef,
}: {
  disabled: boolean
  errors: ActivationFieldErrors
  handleNameChange: ChangeEventHandler<HTMLInputElement>
  handlePasswordChange: ChangeEventHandler<HTMLInputElement>
  name: ReturnType<typeof useInputValue>
  nameErrorID: string
  nameID: string
  nameRef: RefObject<HTMLInputElement | null>
  password: ReturnType<typeof useInputValue>
  passwordDescriptionID: string
  passwordErrorID: string
  passwordID: string
  passwordRef: RefObject<HTMLInputElement | null>
}) {
  const passwordDescribedBy = errors.password
    ? `${passwordDescriptionID} ${passwordErrorID}`
    : passwordDescriptionID

  return (
    <FieldGroup>
      <Field data-invalid={Boolean(errors.name)}>
        <FieldLabel htmlFor={nameID}>Full name</FieldLabel>
        <Input
          aria-describedby={errors.name ? nameErrorID : undefined}
          aria-invalid={errors.name ? true : undefined}
          autoComplete='name'
          className='h-11 rounded-[10px] border-[var(--border-control-default)] bg-[var(--surface-general-default)] px-3 text-base text-[var(--text-default-heading)] placeholder:text-[var(--text-default-secondary)] focus-visible:border-[var(--border-focus-strong)] focus-visible:ring-2 focus-visible:ring-[var(--border-focus-strong)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--surface-general-default)] sm:text-sm'
          disabled={disabled}
          id={nameID}
          name='name'
          onChange={handleNameChange}
          placeholder='Amina Rahman'
          ref={nameRef}
          required
          type='text'
          value={name.value}
        />
        <FieldError id={nameErrorID}>{errors.name}</FieldError>
      </Field>

      <Field data-invalid={Boolean(errors.password)}>
        <FieldLabel htmlFor={passwordID}>Password</FieldLabel>
        <Input
          aria-describedby={passwordDescribedBy}
          aria-invalid={errors.password ? true : undefined}
          autoComplete='new-password'
          className='h-11 rounded-[10px] border-[var(--border-control-default)] bg-[var(--surface-general-default)] px-3 text-base text-[var(--text-default-heading)] placeholder:text-[var(--text-default-secondary)] focus-visible:border-[var(--border-focus-strong)] focus-visible:ring-2 focus-visible:ring-[var(--border-focus-strong)] focus-visible:ring-offset-2 focus-visible:ring-offset-[var(--surface-general-default)] sm:text-sm'
          disabled={disabled}
          id={passwordID}
          name='password'
          onChange={handlePasswordChange}
          placeholder='Create a password'
          ref={passwordRef}
          required
          type='password'
          value={password.value}
        />
        <FieldDescription id={passwordDescriptionID}>
          12 characters minimum.
        </FieldDescription>
        <FieldError id={passwordErrorID}>{errors.password}</FieldError>
      </Field>
    </FieldGroup>
  )
}

function readActivationFieldErrors(
  name: string,
  password: string,
): ActivationFieldErrors {
  return {
    name: name.trim() ? '' : 'Enter your full name.',
    password:
      password.length === 0
        ? 'Enter a password.'
        : Array.from(password).length < minimumPasswordLength
          ? 'Use at least 12 characters for your password.'
          : '',
  }
}

function readInviteActivationError(caught: unknown): string {
  const fallback =
    'Unable to accept this invite. Check your connection and try again.'

  if (!(caught instanceof Error)) {
    return fallback
  }

  if (caught.message === 'Failed to fetch') {
    return fallback
  }

  const message = caught.message.toLowerCase()

  if (
    message.includes('invalid invite') ||
    message.includes('invite expired')
  ) {
    return 'This invite link is invalid or has expired. Ask your school administrator for a new invite.'
  }

  return fallback
}
