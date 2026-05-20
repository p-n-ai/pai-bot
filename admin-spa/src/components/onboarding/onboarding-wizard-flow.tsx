import { CheckIcon, SparklesIcon } from 'lucide-react'
import { useCallback, useMemo } from 'react'
import type { ChangeEvent, ReactNode } from 'react'

import type { SubmitOnboardingInput } from '@/lib/onboarding-types'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  defaultOnboardingClassName,
  isOnboardingSubmittable,
  onboardingBotPresetOptions,
  onboardingCurriculumOptions,
} from '@/lib/onboarding-form'
import { cn } from '@/lib/utils'

type OnboardingStepID = 'curriculum' | 'class' | 'bot' | 'school'

interface OnboardingStep {
  id: OnboardingStepID
  title: string
}

interface OnboardingWizardStepProps {
  currentPresetTitle: string
  form: SubmitOnboardingInput
  handleFormChange: (form: SubmitOnboardingInput) => void
  tenantName: string
}

const onboardingSteps: ReadonlyArray<OnboardingStep> = [
  { id: 'curriculum', title: 'Curriculum' },
  { id: 'class', title: 'First class' },
  { id: 'bot', title: 'Bot setup' },
  { id: 'school', title: 'School' },
]

const progressWidthClasses = [
  { className: 'w-1/4', value: 25 },
  { className: 'w-1/2', value: 50 },
  { className: 'w-3/4', value: 75 },
  { className: 'w-full', value: 100 },
] as const

const stepComponents = {
  bot: OnboardingBotStep,
  class: OnboardingClassStep,
  curriculum: OnboardingCurriculumStep,
  school: OnboardingSchoolStep,
} satisfies Record<
  OnboardingStepID,
  (props: OnboardingWizardStepProps) => ReactNode
>

export function OnboardingWizardFlow({
  form,
  handleFormChange,
  isPending,
  handleStepIndexChange,
  stepIndex,
  tenantName,
}: {
  form: SubmitOnboardingInput
  handleFormChange: (form: SubmitOnboardingInput) => void
  isPending: boolean
  handleStepIndexChange: (stepIndex: number) => void
  stepIndex: number
  tenantName: string
}) {
  const currentStep = onboardingSteps[stepIndex] ?? onboardingSteps[0]
  const currentPreset = useMemo(
    () => getBotPreset(form.bot_setup.preset),
    [form.bot_setup.preset],
  )
  const canAdvance = canAdvanceStep(currentStep.id, form)
  const canSubmit = isOnboardingSubmittable(form)
  const handleBack = useCallback(() => {
    handleStepIndexChange(Math.max(stepIndex - 1, 0))
  }, [handleStepIndexChange, stepIndex])
  const handleNext = useCallback(() => {
    if (canAdvance) {
      handleStepIndexChange(Math.min(stepIndex + 1, onboardingSteps.length - 1))
    }
  }, [canAdvance, handleStepIndexChange, stepIndex])

  return (
    <>
      <div className='grid gap-4.5'>
        <OnboardingStepper
          handleStepIndexChange={handleStepIndexChange}
          stepIndex={stepIndex}
        />
        <OnboardingStepBody
          currentPresetTitle={currentPreset.title}
          currentStepID={currentStep.id}
          form={form}
          handleFormChange={handleFormChange}
          tenantName={tenantName}
        />
      </div>
      <OnboardingWizardActions
        canAdvance={canAdvance}
        canSubmit={canSubmit}
        handleBack={handleBack}
        handleNext={handleNext}
        isPending={isPending}
        stepIndex={stepIndex}
      />
    </>
  )
}

function OnboardingStepper({
  handleStepIndexChange,
  stepIndex,
}: {
  handleStepIndexChange: (stepIndex: number) => void
  stepIndex: number
}) {
  const progressValue = Math.round(
    ((stepIndex + 1) / onboardingSteps.length) * 100,
  )

  return (
    <div className='grid gap-3'>
      <div
        aria-label='Setup progress'
        aria-valuemax={100}
        aria-valuemin={0}
        aria-valuenow={progressValue}
        className='h-2 overflow-hidden rounded-full bg-muted'
        role='progressbar'
      >
        <span
          className={cn(
            'block h-full rounded-full bg-foreground',
            getProgressWidthClass(progressValue),
          )}
        />
      </div>
      <div
        className='grid grid-cols-[repeat(auto-fit,minmax(130px,1fr))] gap-2 border-b border-border pb-2.5'
        aria-label='Setup steps'
      >
        {onboardingSteps.map((step, index) => (
          <OnboardingStepButton
            handleStepIndexChange={handleStepIndexChange}
            index={index}
            isActive={index === stepIndex}
            key={step.id}
            title={step.title}
          />
        ))}
      </div>
    </div>
  )
}

function OnboardingStepButton({
  handleStepIndexChange,
  index,
  isActive,
  title,
}: {
  handleStepIndexChange: (stepIndex: number) => void
  index: number
  isActive: boolean
  title: string
}) {
  const handleClick = useCallback(() => {
    handleStepIndexChange(index)
  }, [handleStepIndexChange, index])

  return (
    <button
      aria-current={isActive ? 'step' : undefined}
      className={cn(
        'border-b-2 border-transparent bg-transparent px-0 py-2 text-left text-muted-foreground transition-colors hover:border-border hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring',
        isActive && 'border-foreground text-foreground',
      )}
      onClick={handleClick}
      type='button'
    >
      {title}
    </button>
  )
}

function OnboardingStepBody({
  currentPresetTitle,
  currentStepID,
  form,
  handleFormChange,
  tenantName,
}: OnboardingWizardStepProps & {
  currentStepID: OnboardingStepID
}) {
  const StepComponent = stepComponents[currentStepID]

  return (
    <StepComponent
      currentPresetTitle={currentPresetTitle}
      form={form}
      handleFormChange={handleFormChange}
      tenantName={tenantName}
    />
  )
}

function OnboardingCurriculumStep({ form }: OnboardingWizardStepProps) {
  const selectedCurriculum = getCurriculum(form.curriculum.syllabus_id)

  return (
    <OnboardingStepLayout
      description='Choose the syllabus.'
      title='Choose the starting syllabus'
    >
      <div className='flex flex-col gap-2'>
        <Label htmlFor='onboarding-curriculum'>Syllabus</Label>
        <Input
          autoComplete='off'
          id='onboarding-curriculum'
          name='onboarding-curriculum'
          readOnly
          value={selectedCurriculum.label}
        />
      </div>
      <p className='text-muted-foreground'>{selectedCurriculum.note}</p>
    </OnboardingStepLayout>
  )
}

function OnboardingClassStep({
  form,
  handleFormChange,
}: OnboardingWizardStepProps) {
  const handleClassNameChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      handleFormChange({
        ...form,
        first_class: {
          ...form.first_class,
          name: event.target.value,
        },
      })
    },
    [form, handleFormChange],
  )

  return (
    <OnboardingStepLayout
      description='Enter the class name.'
      title='Name the class'
    >
      <div className='flex flex-col gap-2'>
        <Label htmlFor='onboarding-first-class'>First class name</Label>
        <Input
          autoComplete='off'
          id='onboarding-first-class'
          name='first-class-name'
          onChange={handleClassNameChange}
          placeholder={defaultOnboardingClassName}
          required
          value={form.first_class.name}
        />
      </div>
    </OnboardingStepLayout>
  )
}

function OnboardingBotStep({
  form,
  handleFormChange,
}: OnboardingWizardStepProps) {
  return (
    <OnboardingStepLayout
      description='Choose the tutor style.'
      title='Choose how the tutor begins'
    >
      <div className='grid grid-cols-[repeat(auto-fit,minmax(180px,1fr))] gap-3'>
        {onboardingBotPresetOptions.map((option) => (
          <BotPresetButton
            form={form}
            handleFormChange={handleFormChange}
            key={option.id}
            option={option}
          />
        ))}
      </div>
    </OnboardingStepLayout>
  )
}

function BotPresetButton({
  form,
  handleFormChange,
  option,
}: {
  form: SubmitOnboardingInput
  handleFormChange: (form: SubmitOnboardingInput) => void
  option: (typeof onboardingBotPresetOptions)[number]
}) {
  const active = option.id === form.bot_setup.preset
  const handleSelect = useCallback(() => {
    handleFormChange({
      ...form,
      bot_setup: { preset: option.id },
    })
  }, [form, handleFormChange, option.id])

  return (
    <button
      aria-label={option.title}
      aria-pressed={active}
      className={cn(
        'grid min-h-40 gap-4 rounded-lg border border-border bg-card p-3.5 text-left text-foreground transition-colors hover:border-foreground/40 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring',
        active && 'border-primary bg-primary text-primary-foreground',
      )}
      data-active={active}
      onClick={handleSelect}
      type='button'
    >
      <span className='flex items-center gap-2 font-semibold'>
        <BotPresetIcon active={active} />
        {option.title}
      </span>
      <small className='leading-6 opacity-80'>{option.description}</small>
    </button>
  )
}

function BotPresetIcon({ active }: { active: boolean }) {
  return active ? (
    <CheckIcon aria-hidden='true' data-icon='inline-start' />
  ) : (
    <SparklesIcon aria-hidden='true' data-icon='inline-start' />
  )
}

function OnboardingSchoolStep({
  currentPresetTitle,
  form,
  handleFormChange,
  tenantName,
}: OnboardingWizardStepProps) {
  const handleSchoolNameChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      handleFormChange({ ...form, school_name: event.target.value })
    },
    [form, handleFormChange],
  )

  return (
    <OnboardingStepLayout
      description='Check the details.'
      title='Review and save'
    >
      <OnboardingReviewList
        currentPresetTitle={currentPresetTitle}
        form={form}
      />
      <div className='flex flex-col gap-2'>
        <Label htmlFor='onboarding-school-name'>School name (optional)</Label>
        <Input
          autoComplete='off'
          id='onboarding-school-name'
          name='school-name'
          onChange={handleSchoolNameChange}
          placeholder={tenantName || 'Current workspace'}
          value={form.school_name ?? ''}
        />
      </div>
    </OnboardingStepLayout>
  )
}

function OnboardingReviewList({
  currentPresetTitle,
  form,
}: {
  currentPresetTitle: string
  form: SubmitOnboardingInput
}) {
  return (
    <dl className='mt-4 grid gap-2.5'>
      <div className='flex justify-between gap-3'>
        <dt>Curriculum</dt>
        <dd className='m-0'>{form.curriculum.label}</dd>
      </div>
      <div className='flex justify-between gap-3'>
        <dt>First class</dt>
        <dd className='m-0'>
          {form.first_class.name.trim() || 'Add a class name'}
        </dd>
      </div>
      <div className='flex justify-between gap-3'>
        <dt>Tutor style</dt>
        <dd className='m-0'>{currentPresetTitle}</dd>
      </div>
    </dl>
  )
}

function OnboardingStepLayout({
  children,
  description,
  title,
}: {
  children: ReactNode
  description: string
  title: string
}) {
  return (
    <section className='grid gap-5 min-[780px]:grid-cols-[minmax(0,0.42fr)_minmax(0,0.58fr)] min-[780px]:items-start'>
      <div>
        <h2 className='text-2xl leading-tight font-semibold tracking-normal text-foreground'>
          {title}
        </h2>
        <p className='mt-2 max-w-md text-sm leading-6 text-muted-foreground'>
          {description}
        </p>
      </div>
      <div className='grid gap-3.5 rounded-lg border border-border p-4'>
        {children}
      </div>
    </section>
  )
}

function OnboardingWizardActions({
  canAdvance,
  canSubmit,
  handleBack,
  handleNext,
  isPending,
  stepIndex,
}: {
  canAdvance: boolean
  canSubmit: boolean
  handleBack: () => void
  handleNext: () => void
  isPending: boolean
  stepIndex: number
}) {
  return (
    <div className='flex items-center justify-between gap-3 border-t border-border pt-3.5'>
      <Button
        disabled={stepIndex === 0 || isPending}
        onClick={handleBack}
        type='button'
        variant='ghost'
      >
        Back
      </Button>
      <ForwardAction
        canAdvance={canAdvance}
        canSubmit={canSubmit}
        handleNext={handleNext}
        isPending={isPending}
        stepIndex={stepIndex}
      />
    </div>
  )
}

function ForwardAction({
  canAdvance,
  canSubmit,
  handleNext,
  isPending,
  stepIndex,
}: {
  canAdvance: boolean
  canSubmit: boolean
  handleNext: () => void
  isPending: boolean
  stepIndex: number
}) {
  if (stepIndex === onboardingSteps.length - 1) {
    return <SubmitSetupAction canSubmit={canSubmit} isPending={isPending} />
  }

  return (
    <NextSetupAction
      canAdvance={canAdvance}
      handleNext={handleNext}
      isPending={isPending}
    />
  )
}

function SubmitSetupAction({
  canSubmit,
  isPending,
}: {
  canSubmit: boolean
  isPending: boolean
}) {
  const label = isPending ? 'Saving…' : 'Save setup'

  return (
    <Button disabled={!canSubmit || isPending} type='submit'>
      {label}
    </Button>
  )
}

function NextSetupAction({
  canAdvance,
  handleNext,
  isPending,
}: {
  canAdvance: boolean
  handleNext: () => void
  isPending: boolean
}) {
  return (
    <Button
      disabled={!canAdvance || isPending}
      onClick={handleNext}
      type='button'
    >
      Next
    </Button>
  )
}

function getCurriculum(
  syllabusID: string,
): (typeof onboardingCurriculumOptions)[number] {
  return (
    onboardingCurriculumOptions.find(
      (option) => option.syllabus_id === syllabusID,
    ) ?? onboardingCurriculumOptions[0]
  )
}

function getProgressWidthClass(progressValue: number): string {
  return (
    progressWidthClasses.find(({ value }) => value === progressValue)
      ?.className ?? progressWidthClasses[0].className
  )
}

function getBotPreset(
  presetID: string,
): (typeof onboardingBotPresetOptions)[number] {
  return (
    onboardingBotPresetOptions.find((option) => option.id === presetID) ??
    onboardingBotPresetOptions[0]
  )
}

function canAdvanceStep(
  stepID: OnboardingStepID,
  form: SubmitOnboardingInput,
): boolean {
  return stepID === 'class' ? Boolean(form.first_class.name.trim()) : true
}
