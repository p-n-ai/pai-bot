import { useCallback, useEffect, useState } from 'react'
import type { FormEvent } from 'react'

import type {
  OnboardingView,
  SubmitOnboardingInput,
  SubmitOnboardingResult,
} from '@/lib/onboarding-types'
import { AuthErrorAlert } from '@/components/shared/auth-error-alert'
import { OnboardingResultActions } from '@/components/onboarding/onboarding-result-actions'
import { OnboardingWizardFlow } from '@/components/onboarding/onboarding-wizard-flow'
import { LoadState } from '@/components/shared/load-state'
import { StatePanel } from '@/components/shared/state-panel'
import { getOnboarding, submitOnboarding } from '@/lib/admin-api'
import {
  buildInitialOnboardingInput,
  isOnboardingSubmittable,
  prepareOnboardingInput,
} from '@/lib/onboarding-form'
import { useSubmitStatus } from '@/hooks/use-submit-status'

type OnboardingLoadState = 'loading' | 'ready' | 'error'

export function OnboardingPanel() {
  const state = useOnboardingLoad()
  const submitState = useOnboardingSubmit(state.form)

  return <OnboardingContent state={state} submitState={submitState} />
}

function useOnboardingLoad() {
  const [view, setView] = useState<OnboardingView | null>(null)
  const [form, setForm] = useState<SubmitOnboardingInput | null>(null)
  const [loadState, setLoadState] = useState<OnboardingLoadState>('loading')
  const [stepIndex, setStepIndex] = useState(0)

  useEffect(() => {
    let mounted = true

    getOnboarding()
      .then((payload) => {
        if (!mounted) {
          return
        }

        setView(payload)
        setForm(buildInitialOnboardingInput(payload))
        setStepIndex(0)
        setLoadState('ready')
      })
      .catch(() => {
        if (mounted) {
          setLoadState('error')
        }
      })

    return () => {
      mounted = false
    }
  }, [])

  return { form, loadState, setForm, setStepIndex, stepIndex, view }
}

function useOnboardingSubmit(form: SubmitOnboardingInput | null) {
  const { beginSubmit, error, finishSubmit, isPending, setError } =
    useSubmitStatus('')
  const [result, setResult] = useState<SubmitOnboardingResult | null>(null)
  const submit = useCallback(
    (event: FormEvent<HTMLFormElement>) => {
      event.preventDefault()

      if (!isOnboardingSubmittable(form)) {
        setError('Complete the syllabus, class name, and tutor style.')
        return
      }

      beginSubmit()
      submitOnboarding(prepareOnboardingInput(form))
        .then(setResult)
        .catch((cause: unknown) => {
          setError(
            cause instanceof Error
              ? cause.message
              : 'Unable to save the setup. Check your connection and try again.',
          )
        })
        .finally(finishSubmit)
    },
    [beginSubmit, finishSubmit, form, setError],
  )

  return { error, isPending, result, setResult, submit }
}

function OnboardingContent({
  state,
  submitState,
}: {
  state: ReturnType<typeof useOnboardingLoad>
  submitState: ReturnType<typeof useOnboardingSubmit>
}) {
  const { form, loadState, setForm, setStepIndex, stepIndex, view } = state

  if (loadState !== 'ready') {
    return (
      <LoadState
        error='Unable to load setup. Check your connection and try again.'
        errorTitle='Setup unavailable'
        loadingTitle='Loading setup'
        status={loadState}
      />
    )
  }

  return (
    <OnboardingReadyBoundary
      form={form}
      setForm={setForm}
      setStepIndex={setStepIndex}
      stepIndex={stepIndex}
      submitState={submitState}
      view={view}
    />
  )
}

function OnboardingReadyBoundary({
  form,
  setForm,
  setStepIndex,
  stepIndex,
  submitState,
  view,
}: {
  form: SubmitOnboardingInput | null
  setForm: (form: SubmitOnboardingInput) => void
  setStepIndex: (stepIndex: number) => void
  stepIndex: number
  submitState: ReturnType<typeof useOnboardingSubmit>
  view: OnboardingView | null
}) {
  if (!view || !form) {
    return (
      <StatePanel title='Setup is not ready'>
        Reload the page. If the problem continues, contact your platform
        administrator.
      </StatePanel>
    )
  }

  return (
    <OnboardingReady
      error={submitState.error}
      form={form}
      setStepIndex={setStepIndex}
      stepIndex={stepIndex}
      isPending={submitState.isPending}
      onFormChange={setForm}
      handleResultChange={submitState.setResult}
      result={submitState.result}
      submit={submitState.submit}
      view={view}
    />
  )
}

function OnboardingReady({
  error,
  form,
  handleResultChange,
  isPending,
  onFormChange,
  result,
  setStepIndex,
  stepIndex,
  submit,
  view,
}: {
  error: string
  form: SubmitOnboardingInput
  handleResultChange: (result: SubmitOnboardingResult | null) => void
  isPending: boolean
  onFormChange: (form: SubmitOnboardingInput) => void
  result: SubmitOnboardingResult | null
  setStepIndex: (stepIndex: number) => void
  stepIndex: number
  submit: (event: FormEvent<HTMLFormElement>) => void
  view: OnboardingView
}) {
  const handleEditSetup = useCallback(() => {
    handleResultChange(null)
  }, [handleResultChange])
  if (result) {
    return (
      <OnboardingResultActions onEditSetup={handleEditSetup} result={result} />
    )
  }

  return (
    <form className='mt-8 grid max-w-4xl gap-5' onSubmit={submit}>
      <OnboardingWizardFlow
        form={form}
        handleFormChange={onFormChange}
        handleStepIndexChange={setStepIndex}
        isPending={isPending}
        stepIndex={stepIndex}
        tenantName={view.tenant_name}
      />
      <AuthErrorAlert message={error} title='Unable to save setup' />
    </form>
  )
}
