import { generateSlug } from "random-word-slugs";
import type { OnboardingView, SubmitOnboardingInput } from "@/lib/api";

export const onboardingCurriculumOptions = [
  {
    syllabus_id: "kssm-algebra",
    label: "KSSM Algebra",
    note: "Demo syllabus for the first onboarding slice.",
  },
] as const;

export const onboardingBotPresetOptions = [
  {
    id: "guided-practice",
    title: "Guided practice",
    description: "Step-by-step hints, calmer pacing, easier first-time teacher rollout.",
  },
  {
    id: "algebra-drills",
    title: "Algebra drills",
    description: "Shorter turns, faster repetition, tighter algebra practice loops.",
  },
  {
    id: "revision-boost",
    title: "Revision boost",
    description: "Mixed review mode for recap, catch-up, and light exam prep.",
  },
] as const;

export function generateDefaultClassName(): string {
  return generateSlug(3, {
    format: "kebab",
    partsOfSpeech: ["noun", "noun", "noun"],
  });
}

export function normalizeClassSlug(raw: string): string {
  return raw
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

export function buildInitialOnboardingInput(view: OnboardingView | null): SubmitOnboardingInput {
  const saved = view?.onboarding ?? null;
  const defaultCurriculum = onboardingCurriculumOptions[0];
  const defaultClassName = generateDefaultClassName();
  const className = saved?.first_class?.name?.trim() || defaultClassName;
  const classSlug = saved?.first_class?.slug?.trim() || normalizeClassSlug(className);
  const defaultPreset = onboardingBotPresetOptions[0]?.id ?? "guided-practice";

  return {
    school_name: saved?.school_name?.trim() || view?.tenant_name?.trim() || "",
    curriculum: {
      syllabus_id: saved?.curriculum?.syllabus_id?.trim() || defaultCurriculum.syllabus_id,
      label: saved?.curriculum?.label?.trim() || defaultCurriculum.label,
    },
    first_class: {
      name: className,
      slug: classSlug,
    },
    bot_setup: {
      preset: saved?.bot_setup?.preset?.trim() || defaultPreset,
    },
  };
}
