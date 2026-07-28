import { Schema } from 'effect'
import type { Schema as EffectSchema } from 'effect/Schema'

export const OnboardingCurriculumSchema = Schema.Struct({
  syllabus_id: Schema.String,
  label: Schema.String,
})

export interface OnboardingCurriculum extends EffectSchema.Type<
  typeof OnboardingCurriculumSchema
> {}

export const OnboardingFirstClassSchema = Schema.Struct({
  id: Schema.optionalKey(Schema.String),
  name: Schema.String,
  slug: Schema.String,
})

export interface OnboardingFirstClass extends EffectSchema.Type<
  typeof OnboardingFirstClassSchema
> {}

export const OnboardingBotSetupSchema = Schema.Struct({
  preset: Schema.String,
})

export interface OnboardingBotSetup extends EffectSchema.Type<
  typeof OnboardingBotSetupSchema
> {}

export const OnboardingStateSchema = Schema.Struct({
  school_name: Schema.optionalKey(Schema.String),
  curriculum: OnboardingCurriculumSchema,
  first_class: OnboardingFirstClassSchema,
  bot_setup: OnboardingBotSetupSchema,
  join_link: Schema.String,
  save_status: Schema.String,
  configured_at: Schema.String,
})

export interface OnboardingState extends EffectSchema.Type<
  typeof OnboardingStateSchema
> {}

export const OnboardingViewSchema = Schema.Struct({
  tenant_id: Schema.String,
  tenant_name: Schema.String,
  onboarding: Schema.optionalKey(Schema.NullOr(OnboardingStateSchema)),
})

export interface OnboardingView extends EffectSchema.Type<
  typeof OnboardingViewSchema
> {}

export const SubmitOnboardingResultSchema = Schema.Struct({
  class_id: Schema.String,
  school_name: Schema.String,
  class_name: Schema.String,
  join_link: Schema.String,
  save_status: Schema.String,
})

export interface SubmitOnboardingResult extends EffectSchema.Type<
  typeof SubmitOnboardingResultSchema
> {}

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
