import { useCallback, useId } from 'react'
import { CircleCheckIcon, MailCheckIcon, ShieldCheckIcon } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import type { FormEvent } from 'react'

import type { AuthSession } from '@/lib/auth-types'
import { AuthErrorAlert } from '@/components/shared/auth-error-alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { acceptInvite } from '@/lib/auth-client'
import { readAuthDisplayError } from '@/lib/auth-errors'
import { useInputValue } from '@/hooks/use-input-value'
import { useSubmitStatus } from '@/hooks/use-submit-status'

interface InviteActivationFormProps {
  token: string
  onAuthenticated: (session: AuthSession) => void
}

export function InviteActivationForm({
  onAuthenticated,
  token,
}: InviteActivationFormProps) {
  const nameID = useId()
  const passwordID = useId()
  const passwordDescriptionID = useId()
  const name = useInputValue()
  const password = useInputValue()
  const tokenMissing = isMissingToken(token)
  const { beginSubmit, error, finishSubmit, isPending, setError } =
    useSubmitStatus(readInitialActivationError(tokenMissing))
  const submit = useInviteActivationSubmit({
    beginSubmit,
    finishSubmit,
    name: name.value,
    onAuthenticated,
    password: password.value,
    setError,
    token,
    tokenMissing,
  })

  return (
    <main
      className='grid min-h-screen items-center gap-8 px-6 py-10 min-[960px]:grid-cols-[minmax(0,1.1fr)_minmax(22rem,0.9fr)] min-[960px]:p-10'
      aria-label='Invite activation'
    >
      <section className='grid gap-6'>
        <InviteActivationHeader />
        <InviteActivationTrustCues />
      </section>

      <section
        className='grid gap-5 rounded-lg border border-white/75 bg-white/80 p-7 shadow-[0_28px_90px_rgb(15_23_42_/_12%)]'
        aria-label='Set up invite access'
      >
        <div className='grid gap-2'>
          <h2 className='text-2xl leading-tight font-semibold text-[#17211b]'>
            Set up access
          </h2>
          <p className='m-0 leading-6 text-muted-foreground'>
            Set your name and password.
          </p>
        </div>
        <form className='mt-6 flex flex-col gap-4' onSubmit={submit}>
          <InviteActivationFields
            name={name}
            nameID={nameID}
            password={password}
            passwordDescriptionID={passwordDescriptionID}
            passwordID={passwordID}
          />

          <AuthErrorAlert message={error} title='Activation failed.' />

          <ActivationSubmitButton
            isPending={isPending}
            tokenMissing={tokenMissing}
          />
        </form>

        <a
          className='text-muted-foreground underline underline-offset-4 hover:text-foreground'
          href='/'
        >
          Back to sign in
        </a>
      </section>
    </main>
  )
}

function InviteActivationHeader() {
  return (
    <header className='mt-6 flex flex-col gap-5'>
      <p className='mb-2 text-xs font-bold tracking-[0.14em] text-muted-foreground uppercase'>
        <ShieldCheckIcon data-icon='inline-start' />
        Invite Activation
      </p>
      <h1>Accept your invite</h1>
      <p className='mt-4 max-w-140 text-lg leading-7 text-muted-foreground'>
        Set up access once. Sign in normally after.
      </p>
    </header>
  )
}

function InviteActivationTrustCues() {
  return (
    <section
      aria-label='Invite activation details'
      className='grid grid-cols-[repeat(auto-fit,minmax(180px,1fr))] gap-3'
    >
      <TrustCue
        description='This link activates one workspace account.'
        Icon={MailCheckIcon}
        title='One-time activation'
      />
      <TrustCue
        description='Go straight to the right admin view.'
        Icon={CircleCheckIcon}
        title='Immediate sign-in'
      />
    </section>
  )
}

function TrustCue({
  description,
  Icon,
  title,
}: {
  Icon: LucideIcon
  description: string
  title: string
}) {
  return (
    <article className='grid gap-2 rounded-lg border border-border bg-card p-3.5'>
      <span className='inline-flex w-6 text-[#2f6f5b]'>
        <Icon aria-hidden='true' />
      </span>
      <strong>{title}</strong>
      <span className='text-sm leading-6 text-muted-foreground'>
        {description}
      </span>
    </article>
  )
}

function isMissingToken(token: string): boolean {
  return token.trim() === ''
}

function readInitialActivationError(tokenMissing: boolean): string {
  return tokenMissing
    ? 'Invite token missing. Open the full invite link from the email and try again.'
    : ''
}

function ActivationSubmitButton({
  isPending,
  tokenMissing,
}: {
  isPending: boolean
  tokenMissing: boolean
}) {
  return (
    <Button disabled={isPending || tokenMissing} type='submit'>
      {isPending ? 'Activating...' : 'Accept invite'}
    </Button>
  )
}

function useInviteActivationSubmit({
  beginSubmit,
  finishSubmit,
  name,
  onAuthenticated,
  password,
  setError,
  token,
  tokenMissing,
}: {
  beginSubmit: () => void
  finishSubmit: () => void
  name: string
  onAuthenticated: (session: AuthSession) => void
  password: string
  setError: (message: string) => void
  token: string
  tokenMissing: boolean
}) {
  return useCallback(
    (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault()

      if (tokenMissing) {
        setError(
          'Invite token missing. Open the full invite link from the email and try again.',
        )
        return
      }

      beginSubmit()

      acceptInvite({
        token,
        name: name.trim(),
        password,
      })
        .then(onAuthenticated)
        .catch((caught: unknown) => {
          setError(
            readAuthDisplayError(
              caught,
              'Invite activation failed',
              "We couldn't reach the activation service. Check your connection and try again.",
            ),
          )
        })
        .finally(() => {
          finishSubmit()
        })
    },
    [
      beginSubmit,
      finishSubmit,
      name,
      onAuthenticated,
      password,
      setError,
      token,
      tokenMissing,
    ],
  )
}

function InviteActivationFields({
  name,
  nameID,
  password,
  passwordDescriptionID,
  passwordID,
}: {
  name: ReturnType<typeof useInputValue>
  nameID: string
  password: ReturnType<typeof useInputValue>
  passwordDescriptionID: string
  passwordID: string
}) {
  return (
    <>
      <div className='flex flex-col gap-2'>
        <Label htmlFor={nameID}>Full name</Label>
        <Input
          autoComplete='name'
          id={nameID}
          name='name'
          onChange={name.handleChange}
          placeholder='Parent One'
          required
          type='text'
          value={name.value}
        />
      </div>

      <div className='flex flex-col gap-2'>
        <Label htmlFor={passwordID}>Password</Label>
        <p
          className='m-0 text-sm text-muted-foreground'
          id={passwordDescriptionID}
        >
          Use a strong password for future sign-ins.
        </p>
        <Input
          aria-describedby={passwordDescriptionID}
          autoComplete='new-password'
          id={passwordID}
          name='password'
          onChange={password.handleChange}
          placeholder='Create a strong password'
          required
          type='password'
          value={password.value}
        />
      </div>
    </>
  )
}
