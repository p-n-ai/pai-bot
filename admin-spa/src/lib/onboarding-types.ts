import { hasStringProps, isRecord } from './type-guards'

export interface OnboardingCurriculum {
  syllabus_id: string
  label: string
}

export interface OnboardingFirstClass {
  id?: string
  name: string
  slug: string
}

export interface OnboardingBotSetup {
  preset: string
}

export interface OnboardingState {
  school_name?: string
  curriculum: OnboardingCurriculum
  first_class: OnboardingFirstClass
  bot_setup: OnboardingBotSetup
  join_link: string
  save_status: string
  configured_at: string
}

export interface OnboardingView {
  tenant_id: string
  tenant_name: string
  onboarding?: OnboardingState | null
}

export interface SubmitOnboardingInput {
  school_name?: string
  curriculum: OnboardingCurriculum
  first_class: OnboardingFirstClass
  bot_setup: OnboardingBotSetup
}

export interface SubmitOnboardingResult {
  class_id: string
  school_name: string
  class_name: string
  join_link: string
  save_status: string
}

export function isOnboardingView(value: unknown): value is OnboardingView {
  if (!isRecord(value)) {
    return false
  }

  return (
    hasStringProps(value, ['tenant_id', 'tenant_name']) &&
    hasOptionalOnboardingState(value.onboarding)
  )
}

export function isSubmitOnboardingResult(
  value: unknown,
): value is SubmitOnboardingResult {
  if (!isRecord(value)) {
    return false
  }

  return hasStringProps(value, [
    'class_id',
    'school_name',
    'class_name',
    'join_link',
    'save_status',
  ])
}

function isOnboardingState(value: unknown): value is OnboardingState {
  if (!isRecord(value)) {
    return false
  }

  return (
    hasOnboardingParts(value) &&
    hasStringProps(value, ['join_link', 'save_status', 'configured_at'])
  )
}

function hasOptionalOnboardingState(value: unknown): boolean {
  return value === undefined || value === null || isOnboardingState(value)
}

function hasOnboardingParts(record: Record<string, unknown>): boolean {
  return (
    isOnboardingCurriculum(record.curriculum) &&
    isOnboardingFirstClass(record.first_class) &&
    isOnboardingBotSetup(record.bot_setup)
  )
}

function isOnboardingCurriculum(value: unknown): value is OnboardingCurriculum {
  if (!isRecord(value)) {
    return false
  }

  return hasStringProps(value, ['syllabus_id', 'label'])
}

function isOnboardingFirstClass(value: unknown): value is OnboardingFirstClass {
  if (!isRecord(value)) {
    return false
  }

  return hasStringProps(value, ['name', 'slug'])
}

function isOnboardingBotSetup(value: unknown): value is OnboardingBotSetup {
  return isRecord(value) && typeof value.preset === 'string'
}
