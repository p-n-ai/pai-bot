import type { OnboardingView, SubmitOnboardingInput } from './onboarding-types'

export const defaultOnboardingClassName = 'Form 1 Mathematics'

export const onboardingCurriculumOptions = [
  {
    label: 'KSSM Algebra',
    note: 'Demo syllabus for the first onboarding slice.',
    syllabus_id: 'kssm-algebra',
  },
] as const

export const onboardingBotPresetOptions = [
  {
    description:
      'Step-by-step hints, calmer pacing, easier first-time teacher rollout.',
    id: 'guided-practice',
    title: 'Guided practice',
  },
  {
    description:
      'Shorter turns, faster repetition, tighter algebra practice loops.',
    id: 'algebra-drills',
    title: 'Algebra drills',
  },
  {
    description: 'Mixed review mode for recap, catch-up, and light exam prep.',
    id: 'revision-boost',
    title: 'Revision boost',
  },
] as const

const defaultOnboardingInput: SubmitOnboardingInput = {
  school_name: '',
  curriculum: {
    label: onboardingCurriculumOptions[0].label,
    syllabus_id: onboardingCurriculumOptions[0].syllabus_id,
  },
  first_class: {
    name: '',
    slug: '',
  },
  bot_setup: {
    preset: onboardingBotPresetOptions[0].id,
  },
}

export function buildInitialOnboardingInput(
  view: OnboardingView,
): SubmitOnboardingInput {
  return view.onboarding
    ? buildSavedOnboardingInput(view.onboarding)
    : buildDefaultOnboardingInput(view)
}

export function prepareOnboardingInput(
  input: SubmitOnboardingInput,
): SubmitOnboardingInput {
  const className = input.first_class.name.trim()
  const classSlug = normalizeClassSlug(input.first_class.slug || className)

  return {
    school_name: input.school_name?.trim() || undefined,
    curriculum: input.curriculum,
    first_class: {
      name: className,
      slug: classSlug,
    },
    bot_setup: input.bot_setup,
  }
}

export function isOnboardingSubmittable(
  form: SubmitOnboardingInput | null,
): form is SubmitOnboardingInput {
  return Boolean(
    form?.curriculum.syllabus_id.trim() &&
    form.first_class.name.trim() &&
    form.bot_setup.preset.trim(),
  )
}

function buildSavedOnboardingInput(
  saved: NonNullable<OnboardingView['onboarding']>,
): SubmitOnboardingInput {
  return {
    school_name: saved.school_name?.trim() || '',
    curriculum: saved.curriculum,
    first_class: {
      name: saved.first_class.name,
      slug: saved.first_class.slug,
    },
    bot_setup: saved.bot_setup,
  }
}

function buildDefaultOnboardingInput(
  view: OnboardingView,
): SubmitOnboardingInput {
  return {
    ...defaultOnboardingInput,
    school_name: view.tenant_name,
  }
}

function normalizeClassSlug(raw: string): string {
  return raw
    .trim()
    .toLowerCase()
    .replaceAll(/[^a-z0-9]+/g, '-')
    .replaceAll(/^-+|-+$/g, '')
}
