import { Schema } from 'effect'

export const OnboardingCurriculumSchema = Schema.Struct({
  syllabus_id: Schema.String,
  label: Schema.String,
})

export type OnboardingCurriculum = typeof OnboardingCurriculumSchema.Type

export const OnboardingFirstClassSchema = Schema.Struct({
  id: Schema.optionalKey(Schema.String),
  name: Schema.String,
  slug: Schema.String,
})

export type OnboardingFirstClass = typeof OnboardingFirstClassSchema.Type

export const OnboardingBotSetupSchema = Schema.Struct({
  preset: Schema.String,
})

export type OnboardingBotSetup = typeof OnboardingBotSetupSchema.Type

export const OnboardingStateSchema = Schema.Struct({
  school_name: Schema.optionalKey(Schema.String),
  curriculum: OnboardingCurriculumSchema,
  first_class: OnboardingFirstClassSchema,
  bot_setup: OnboardingBotSetupSchema,
  join_link: Schema.String,
  save_status: Schema.String,
  configured_at: Schema.String,
})

export type OnboardingState = typeof OnboardingStateSchema.Type

export const OnboardingViewSchema = Schema.Struct({
  tenant_id: Schema.String,
  tenant_name: Schema.String,
  onboarding: Schema.optionalKey(Schema.NullOr(OnboardingStateSchema)),
})

export type OnboardingView = typeof OnboardingViewSchema.Type

export const SubmitOnboardingResultSchema = Schema.Struct({
  class_id: Schema.String,
  school_name: Schema.String,
  class_name: Schema.String,
  join_link: Schema.String,
  save_status: Schema.String,
})

export type SubmitOnboardingResult = typeof SubmitOnboardingResultSchema.Type

export interface SubmitOnboardingInput {
  school_name?: string
  curriculum: OnboardingCurriculum
  first_class: OnboardingFirstClass
  bot_setup: OnboardingBotSetup
}

const matchesOnboardingView = Schema.is(OnboardingViewSchema)
const matchesSubmitOnboardingResult = Schema.is(SubmitOnboardingResultSchema)

/** Returns whether an unknown response satisfies the onboarding view contract. */
export function isOnboardingView(value: unknown): value is OnboardingView {
  return matchesOnboardingView(value)
}

/** Returns whether an unknown response satisfies the onboarding result contract. */
export function isSubmitOnboardingResult(
  value: unknown,
): value is SubmitOnboardingResult {
  return matchesSubmitOnboardingResult(value)
}
