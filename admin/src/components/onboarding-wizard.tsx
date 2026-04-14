"use client";

import { useTransition } from "react";
import type { ReactNode } from "react";
import { IconArrowLeft, IconArrowRight, IconCheck, IconSparkles } from "@tabler/icons-react";
import { AdminSurface } from "@/components/admin-surface";
import { OnboardingSuccessSection } from "@/components/onboarding-success-section";
import { StatePanel } from "@/components/state-panel";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Field, FieldDescription, FieldGroup, FieldLabel } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Progress, ProgressLabel } from "@/components/ui/progress";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { submitOnboarding, type OnboardingView, type SubmitOnboardingInput } from "@/lib/api";
import { normalizeClassSlug, onboardingBotPresetOptions, onboardingCurriculumOptions } from "@/lib/onboarding";
import { cn } from "@/lib/utils";
import { useOnboardingWizardStore } from "@/stores/onboarding-wizard-store";

type OnboardingStep = {
  id: "curriculum" | "class" | "bot" | "school";
  title: string;
  description: string;
};

const steps: readonly OnboardingStep[] = [
  { id: "curriculum", title: "Curriculum", description: "Choose the starting syllabus for this class." },
  { id: "class", title: "First class", description: "Name the class students will join first." },
  { id: "bot", title: "Bot setup", description: "Choose how the tutor should respond at the start." },
  { id: "school", title: "School", description: "Review the setup and add the school name if you want it shown." },
];

export function OnboardingWizard({ initialData, loadError = "" }: {
  initialData: OnboardingView | null;
  loadError?: string;
}) {
  if (initialData) {
    const snapshot = useOnboardingWizardStore.getState();
    if (snapshot.tenantId !== initialData.tenant_id) {
      snapshot.initializeFromView(initialData);
    }
  }

  const tenantId = useOnboardingWizardStore((state) => state.tenantId);
  const form = useOnboardingWizardStore((state) => state.form);
  const stepIndex = useOnboardingWizardStore((state) => state.stepIndex);
  const submitError = useOnboardingWizardStore((state) => state.submitError);
  const copyFeedback = useOnboardingWizardStore((state) => state.copyFeedback);
  const result = useOnboardingWizardStore((state) => state.result);
  const setStepIndex = useOnboardingWizardStore((state) => state.setStepIndex);
  const setSchoolName = useOnboardingWizardStore((state) => state.setSchoolName);
  const setClassName = useOnboardingWizardStore((state) => state.setClassName);
  const setBotPreset = useOnboardingWizardStore((state) => state.setBotPreset);
  const setSubmitError = useOnboardingWizardStore((state) => state.setSubmitError);
  const setCopyFeedback = useOnboardingWizardStore((state) => state.setCopyFeedback);
  const setResult = useOnboardingWizardStore((state) => state.setResult);
  const syncNormalizedClassSlug = useOnboardingWizardStore((state) => state.syncNormalizedClassSlug);
  const clearTransientFeedback = useOnboardingWizardStore((state) => state.clearTransientFeedback);
  const [isPending, startTransition] = useTransition();

  if (!initialData) {
    return (
      <StatePanel
        tone={loadError ? "error" : "empty"}
        title={loadError ? "Onboarding unavailable" : "Onboarding not ready"}
        description={loadError || "The onboarding API is not returning tenant setup data yet."}
      />
    );
  }

  if (tenantId !== initialData.tenant_id) {
    return (
      <StatePanel
        tone="loading"
        title="Loading onboarding"
        description="Syncing the current workspace setup into the onboarding flow."
      />
    );
  }

  const currentStep = steps[stepIndex];
  const savedState = initialData.onboarding ?? null;
  const className = form.first_class.name.trim();
  const canAdvance =
    currentStep.id === "class" ? Boolean(className) : currentStep.id === "bot" ? Boolean(form.bot_setup.preset.trim()) : true;
  const canSubmit = Boolean(form.curriculum.syllabus_id.trim() && className && form.bot_setup.preset.trim());
  const currentPreset = onboardingBotPresetOptions.find((option) => option.id === form.bot_setup.preset) ?? onboardingBotPresetOptions[0];
  const resultSchoolName = result?.school_name?.trim() || form.school_name?.trim() || initialData.tenant_name || "Current workspace";

  async function handleCopyJoinLink() {
    if (!result?.join_link) {
      return;
    }

    try {
      await navigator.clipboard.writeText(result.join_link);
      setCopyFeedback("Copied");
      window.setTimeout(() => {
        if (useOnboardingWizardStore.getState().copyFeedback === "Copied") {
          useOnboardingWizardStore.getState().setCopyFeedback("");
        }
      }, 2000);
    } catch {
      setSubmitError("Could not copy the join link.");
    }
  }

  function handleNext() {
    if (canAdvance) {
      setStepIndex(Math.min(stepIndex + 1, steps.length - 1));
    }
  }

  function handleBack() {
    setStepIndex(Math.max(stepIndex - 1, 0));
  }

  function handleSubmit() {
    if (!canSubmit) {
      setSubmitError("Curriculum, first class, and bot setup are required.");
      return;
    }

    clearTransientFeedback();

    startTransition(async () => {
      try {
        const trimmedClassName = form.first_class.name.trim();
        const classSlug = normalizeClassSlug(form.first_class.slug || trimmedClassName);
        syncNormalizedClassSlug();
        const payload = await submitOnboarding({
          school_name: form.school_name?.trim() || undefined,
          curriculum: form.curriculum,
          first_class: {
            name: trimmedClassName,
            slug: classSlug,
          },
          bot_setup: form.bot_setup,
        });
        setResult(payload);
      } catch (error) {
        setSubmitError(error instanceof Error ? error.message : "Onboarding save failed.");
      }
    });
  }

  if (result) {
    return (
      <OnboardingSuccessSection
        copyFeedback={copyFeedback}
        onCopyJoinLink={() => void handleCopyJoinLink()}
        onEditSetup={() => setResult(null)}
        result={result}
        resultSchoolName={resultSchoolName}
      />
    );
  }

  return (
    <div className="flex flex-col gap-4">
      {savedState ? (
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <span className="h-1.5 w-1.5 rounded-full bg-foreground/40" aria-hidden="true" />
          <span>
            <span className="font-medium text-foreground">{savedState.first_class.name}</span> is already set up. Save again to update it.
          </span>
        </div>
      ) : null}

      <AdminSurface>
        <div className="flex min-h-[35rem] flex-col gap-6">
          <OnboardingStepper currentStepIndex={stepIndex} steps={steps} onStepSelect={setStepIndex} />

          <OnboardingStepBody
            currentPresetTitle={currentPreset?.title ?? form.bot_setup.preset}
            currentStepId={currentStep.id}
            form={form}
            tenantName={initialData.tenant_name}
            onBotPresetChange={setBotPreset}
            onClassNameChange={setClassName}
            onSchoolNameChange={setSchoolName}
          />

          {submitError ? (
            <Alert variant="destructive">
              <AlertTitle>Save failed</AlertTitle>
              <AlertDescription>{submitError}</AlertDescription>
            </Alert>
          ) : null}

          <div className="flex flex-wrap items-center justify-between gap-3">
            <Button type="button" variant="ghost" size="sm" onClick={handleBack} disabled={stepIndex === 0 || isPending}>
              <IconArrowLeft data-icon="inline-start" />
              Back
            </Button>
            {stepIndex < steps.length - 1 ? (
              <Button type="button" onClick={handleNext} disabled={!canAdvance || isPending}>
                Next
                <IconArrowRight data-icon="inline-end" />
              </Button>
            ) : (
              <Button type="button" onClick={handleSubmit} disabled={!canSubmit || isPending}>
                {isPending ? "Saving..." : "Save setup"}
              </Button>
            )}
          </div>
        </div>
      </AdminSurface>
    </div>
  );
}

function OnboardingStepBody({
  currentPresetTitle,
  currentStepId,
  form,
  tenantName,
  onBotPresetChange,
  onClassNameChange,
  onSchoolNameChange,
}: {
  currentPresetTitle: string;
  currentStepId: OnboardingStep["id"];
  form: ReturnType<typeof useOnboardingWizardStore.getState>["form"];
  tenantName: string;
  onBotPresetChange: (preset: string) => void;
  onClassNameChange: (className: string) => void;
  onSchoolNameChange: (schoolName: string) => void;
}) {
  switch (currentStepId) {
    case "curriculum":
      return <OnboardingCurriculumStep syllabusID={form.curriculum.syllabus_id} />;
    case "class":
      return <OnboardingClassStep className={form.first_class.name} onClassNameChange={onClassNameChange} />;
    case "bot":
      return <OnboardingBotStep selectedPreset={form.bot_setup.preset} onPresetSelect={onBotPresetChange} />;
    case "school":
      return (
        <OnboardingSchoolStep
          form={form}
          currentPresetTitle={currentPresetTitle}
          tenantName={tenantName}
          onSchoolNameChange={onSchoolNameChange}
        />
      );
    default:
      return null;
  }
}

function OnboardingStepper({
  currentStepIndex,
  steps,
  onStepSelect,
}: {
  currentStepIndex: number;
  steps: readonly OnboardingStep[];
  onStepSelect: (stepIndex: number) => void;
}) {
  const completionRatio = ((currentStepIndex + 1) / steps.length) * 100;

  return (
    <div className="flex flex-col gap-3">
      <p className="text-sm font-medium text-foreground">
        Step {currentStepIndex + 1} of {steps.length}
      </p>
      <Progress value={completionRatio}>
        <ProgressLabel className="sr-only">Setup progress</ProgressLabel>
      </Progress>
      <div className="grid gap-x-4 gap-y-2 border-b border-border/70 pb-2 sm:grid-cols-2 xl:grid-cols-4">
        {steps.map((step, index) => (
          <button
            key={step.id}
            type="button"
            onClick={() => onStepSelect(index)}
            className={cn(
              "flex items-center gap-2 border-b-2 pb-2 text-left transition-colors",
              index === currentStepIndex
                ? "border-foreground text-foreground"
                : "border-transparent text-muted-foreground hover:border-border hover:text-foreground",
            )}
          >
            <span className="text-[11px] font-medium tabular-nums opacity-70">{index + 1}</span>
            <p className="text-sm font-medium">{step.title}</p>
          </button>
        ))}
      </div>
    </div>
  );
}

function OnboardingCurriculumStep({ syllabusID }: { syllabusID: string }) {
  return (
    <OnboardingStepLayout
      title="Choose the starting syllabus"
      description="Choose the syllabus this class will start with."
      supportTitle="What happens next"
      supportContent="This syllabus is attached to the class you create in the next step."
    >
      <FieldGroup>
        <Field>
          <FieldLabel htmlFor="onboarding-curriculum">Syllabus</FieldLabel>
          <Select value={syllabusID} disabled>
            <SelectTrigger id="onboarding-curriculum">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {onboardingCurriculumOptions.map((option) => (
                <SelectItem key={option.syllabus_id} value={option.syllabus_id}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
          <FieldDescription>This is the starting syllabus for your first class.</FieldDescription>
        </Field>
      </FieldGroup>
    </OnboardingStepLayout>
  );
}

function OnboardingClassStep({
  className,
  onClassNameChange,
}: {
  className: string;
  onClassNameChange: (className: string) => void;
}) {
  return (
    <OnboardingStepLayout
      title="Name the class"
      description="Give students and teachers a clear class name."
      supportTitle="Student join link preview"
      supportContent={`/join/${normalizeClassSlug(className)}`}
    >
      <FieldGroup>
        <Field>
          <FieldLabel htmlFor="onboarding-first-class">First class name</FieldLabel>
          <Input
            id="onboarding-first-class"
            value={className}
            onChange={(event) => onClassNameChange(event.target.value)}
            placeholder="Pilot Class A"
          />
          <FieldDescription>Keep it simple and recognizable, for example Form 1 Algebra or Pilot Class A.</FieldDescription>
        </Field>
      </FieldGroup>
    </OnboardingStepLayout>
  );
}

function OnboardingBotStep({
  selectedPreset,
  onPresetSelect,
}: {
  selectedPreset: string;
  onPresetSelect: (preset: string) => void;
}) {
  return (
    <OnboardingStepLayout
      title="Choose how the tutor begins"
      description="Choose how the tutor should respond when the class begins."
      supportTitle="How to choose"
      supportContent="Pick the tone that best fits your first rollout. You can refine it later."
    >
      <div className="grid w-full grid-cols-1 items-stretch gap-4 sm:grid-cols-2">
        {onboardingBotPresetOptions.map((option) => {
          const active = option.id === selectedPreset;
          return (
            <button
              key={option.id}
              type="button"
              onClick={() => onPresetSelect(option.id)}
              aria-pressed={active}
              className={cn(
                "flex h-full min-h-40 w-full flex-col items-stretch justify-between rounded-xl border px-4 py-4 text-left transition-colors",
                active
                  ? "border-foreground bg-foreground text-background"
                  : "border-border bg-card text-card-foreground hover:border-foreground/40",
              )}
            >
              <div className="flex w-full items-start justify-between gap-3">
                <p className="text-sm font-medium">{option.title}</p>
                {active ? <IconCheck className="size-4" /> : <IconSparkles className="size-4 opacity-70" />}
              </div>
              <p className={cn("mt-3 text-xs leading-5", active ? "text-background/80" : "text-muted-foreground")}>{option.description}</p>
            </button>
          );
        })}
      </div>
    </OnboardingStepLayout>
  );
}

function OnboardingSchoolStep({
  form,
  currentPresetTitle,
  tenantName,
  onSchoolNameChange,
}: {
  form: SubmitOnboardingInput;
  currentPresetTitle: string;
  tenantName: string;
  onSchoolNameChange: (schoolName: string) => void;
}) {
  return (
    <OnboardingStepLayout
      title="Review and save"
      description="Check the setup before you save."
      supportTitle="What you are about to save"
      supportContent={
        <dl className="space-y-2 text-sm text-muted-foreground">
          <div className="flex items-center justify-between gap-3">
            <dt>Curriculum</dt>
            <dd>{form.curriculum.label}</dd>
          </div>
          <div className="flex items-center justify-between gap-3">
            <dt>First class</dt>
            <dd>{form.first_class.name.trim() || "Add a class name"}</dd>
          </div>
          <div className="flex items-center justify-between gap-3">
            <dt>Tutor style</dt>
            <dd>{currentPresetTitle}</dd>
          </div>
        </dl>
      }
    >
      <FieldGroup>
        <Field>
          <FieldLabel htmlFor="onboarding-school-name">School name (optional)</FieldLabel>
          <Input
            id="onboarding-school-name"
            value={form.school_name ?? ""}
            onChange={(event) => onSchoolNameChange(event.target.value)}
            placeholder={tenantName || "Current workspace"}
          />
          <FieldDescription>This helps teachers recognize the workspace. Leave it blank if you want to finish setup first.</FieldDescription>
        </Field>
      </FieldGroup>
    </OnboardingStepLayout>
  );
}

function OnboardingStepLayout({
  title,
  description,
  supportTitle,
  supportContent,
  children,
}: {
  title: string;
  description: string;
  supportTitle: string;
  supportContent: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="grid gap-4">
      <section className="space-y-2">
        <h2 className="text-2xl font-semibold tracking-tight text-foreground">{title}</h2>
        <p className="max-w-2xl text-sm leading-6 text-muted-foreground">{description}</p>
      </section>
      <section className="grid gap-5 md:grid-cols-[minmax(0,503px)_minmax(0,483px)] md:items-start md:justify-between">
        <div className="rounded-2xl border bg-card p-5 md:p-6">{children}</div>
        <div className="px-1 py-2 md:px-2">
          <p className="text-[11px] font-medium uppercase tracking-[0.08em] text-muted-foreground">{supportTitle}</p>
          <div className="mt-2 text-sm leading-6 text-muted-foreground">{supportContent}</div>
        </div>
      </section>
    </div>
  );
}
