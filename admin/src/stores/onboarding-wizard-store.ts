"use client";

import { create } from "zustand";
import type { OnboardingView, SubmitOnboardingInput, SubmitOnboardingResult } from "@/lib/api";
import { buildInitialOnboardingInput, normalizeClassSlug } from "@/lib/onboarding";

type OnboardingWizardSnapshot = {
  tenantId: string | null;
  form: SubmitOnboardingInput;
  stepIndex: number;
  submitError: string;
  copyFeedback: string;
  result: SubmitOnboardingResult | null;
};

type OnboardingWizardStore = OnboardingWizardSnapshot & {
  initializeFromView: (view: OnboardingView) => void;
  setStepIndex: (stepIndex: number) => void;
  setSchoolName: (schoolName: string) => void;
  setClassName: (className: string) => void;
  setBotPreset: (preset: string) => void;
  setSubmitError: (submitError: string) => void;
  setCopyFeedback: (copyFeedback: string) => void;
  setResult: (result: SubmitOnboardingResult | null) => void;
  syncNormalizedClassSlug: () => void;
  clearTransientFeedback: () => void;
};

function getDefaultForm(): SubmitOnboardingInput {
  return buildInitialOnboardingInput(null);
}

function getDefaultSnapshot(): OnboardingWizardSnapshot {
  return {
    tenantId: null,
    form: getDefaultForm(),
    stepIndex: 0,
    submitError: "",
    copyFeedback: "",
    result: null,
  };
}

export const useOnboardingWizardStore = create<OnboardingWizardStore>((set) => ({
  ...getDefaultSnapshot(),
  initializeFromView: (view) => {
    set({
      tenantId: view.tenant_id,
      form: buildInitialOnboardingInput(view),
      stepIndex: 0,
      submitError: "",
      copyFeedback: "",
      result: null,
    });
  },
  setStepIndex: (stepIndex) => set({ stepIndex }),
  setSchoolName: (schoolName) =>
    set((state) => ({
      form: {
        ...state.form,
        school_name: schoolName,
      },
    })),
  setClassName: (className) =>
    set((state) => ({
      form: {
        ...state.form,
        first_class: {
          name: className,
          slug: normalizeClassSlug(className),
        },
      },
    })),
  setBotPreset: (preset) =>
    set((state) => ({
      form: {
        ...state.form,
        bot_setup: { preset },
      },
    })),
  setSubmitError: (submitError) => set({ submitError }),
  setCopyFeedback: (copyFeedback) => set({ copyFeedback }),
  setResult: (result) => set({ result }),
  syncNormalizedClassSlug: () =>
    set((state) => {
      const className = state.form.first_class.name.trim();
      const classSlug = normalizeClassSlug(state.form.first_class.slug || className);
      return {
        form: {
          ...state.form,
          first_class: {
            name: className,
            slug: classSlug,
          },
        },
      };
    }),
  clearTransientFeedback: () => set({ submitError: "", copyFeedback: "" }),
}));
