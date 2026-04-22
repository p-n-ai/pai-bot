import Link from "next/link";
import type { CSSProperties } from "react";
import { cn } from "@/lib/utils";
import { LandingIcon, type LandingIconName } from "@/components/landing/landing-icon";
import {
  LandingCommandStrip,
  LandingLiveDemo,
  LandingOutcomeRail,
  LandingWorkflowSpotlight,
} from "@/components/landing/landing-live-demo";

const landingButtonClassName =
  "inline-flex shrink-0 items-center justify-center whitespace-nowrap transition-all outline-none focus-visible:ring-3 focus-visible:ring-ring/30";

const landingHeaderPrimaryButtonClassName =
  "h-10 rounded-full bg-foreground px-5 text-sm font-semibold text-background shadow-[0_10px_24px_color-mix(in_oklch,var(--foreground)_14%,transparent)] hover:bg-foreground/88 active:translate-y-px";

const landingPrimaryButtonClassName =
  "h-12 rounded-full bg-primary px-6 text-sm font-semibold text-primary-foreground shadow-[0_14px_30px_color-mix(in_oklch,var(--primary)_22%,transparent)] hover:bg-primary/90 active:translate-y-px";

const heroSignals = [
  { icon: "chatCircleText", label: "Student help", value: "step-by-step chat" },
  { icon: "trendUp", label: "Teacher follow-up", value: "weak topics first" },
  { icon: "usersThree", label: "Family follow-up", value: "clear next action" },
] satisfies Array<{ icon: LandingIconName; label: string; value: string }>;

const dailyLoop = [
  {
    label: "Student",
    title: "A question comes in through chat",
    body: "Students ask inside WhatsApp or Telegram. No extra portal to remember.",
  },
  {
    label: "Tutor",
    title: "One step gets checked",
    body: "The tutor asks for the next move, catches mistakes early, and keeps the student working.",
  },
  {
    label: "Teacher",
    title: "The follow-up list is ready",
    body: "Before class, the teacher sees who needs help and which topic to revisit.",
  },
];

const workflowItems = [
  {
    icon: "chatCircleText",
    label: "Channels",
    title: "Students stay in chat",
    body: "WhatsApp and Telegram become the practice surface. No extra app for students to remember.",
  },
  {
    icon: "bookOpenText",
    label: "Math help",
    title: "Every answer becomes a step",
    body: "The tutor slows the question down, checks the next move, and keeps the student working.",
  },
  {
    icon: "trendUp",
    label: "Signals",
    title: "Weak topics surface early",
    body: "The teacher sees who struggled, which topic slipped, and where the class is starting to drift.",
  },
  {
    icon: "paperPlaneTilt",
    label: "Follow-up",
    title: "Tomorrow's action is ready",
    body: "A short repair list turns chat history into reteach groups, parent notes, or class review.",
  },
] satisfies Array<{ icon: LandingIconName; label: string; title: string; body: string }>;

const teacherOutcomes = [
  {
    value: "who",
    label: "Pull aside the right students",
    body: "Spend attention on the students who got stuck, not only the students who speak up.",
  },
  {
    value: "what",
    label: "Reteach the right topic",
    body: "Use the weakest topic signal to plan the next warm-up, mini lesson, or homework review.",
  },
  {
    value: "when",
    label: "Move on with confidence",
    body: "Coverage and mastery signals make it clearer when the class is ready for the next concept.",
  },
];

const landingThemeStyle = {
  "--background": "#fbf7f1",
  "--foreground": "#171310",
  "--card": "#fffdf9",
  "--card-foreground": "#171310",
  "--primary": "oklch(0.553 0.195 38.402)",
  "--primary-foreground": "#fff7ed",
  "--secondary": "#f1e8db",
  "--secondary-foreground": "#171310",
  "--muted": "#eee6db",
  "--muted-foreground": "#766b5d",
  "--border": "#e4d8c8",
  "--ring": "#a96b48",
  "--destructive": "oklch(0.577 0.245 27.325)",
} as CSSProperties;

export function RootLandingPage({
  primaryHref,
  primaryLabel,
  signedInLabel,
}: {
  primaryHref: string;
  primaryLabel: string;
  signedInLabel: string | null;
}) {
  const primaryActionLabel = signedInLabel ? "Open workspace" : primaryLabel;

  return (
    <main className="relative min-h-[100dvh] overflow-hidden bg-background text-foreground" style={landingThemeStyle}>
      <div className="pointer-events-none absolute inset-0 -z-10 bg-[linear-gradient(90deg,color-mix(in_oklch,var(--foreground)_6%,transparent)_1px,transparent_1px),linear-gradient(180deg,color-mix(in_oklch,var(--foreground)_5%,transparent)_1px,transparent_1px)] bg-[size:88px_88px] opacity-45" />
      <div className="pointer-events-none absolute inset-x-0 top-0 -z-10 h-[42rem] bg-[linear-gradient(180deg,#fffdf9_0%,rgba(251,247,241,0.88)_56%,rgba(251,247,241,0)_100%)]" />

      <div className="mx-auto flex min-h-[100dvh] max-w-[88rem] flex-col px-5 py-0 sm:px-8 lg:px-10">
        <header className="mt-5 flex items-center justify-between gap-6 border-b border-foreground/10 bg-[#fbf7f1]/82 px-0 py-4 backdrop-blur">
          <div className="flex items-center gap-3">
            <div className="flex size-10 items-center justify-center rounded-xl bg-foreground text-background">
              <LandingIcon name="robot" />
            </div>
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-[0.26em] text-foreground">P&amp;AI Bot</p>
              <p className="hidden text-sm font-medium text-muted-foreground sm:block">Chat-first math support for schools</p>
            </div>
          </div>
          <Link
            href={primaryHref}
            className={cn(landingButtonClassName, landingHeaderPrimaryButtonClassName)}
          >
            {primaryActionLabel}
          </Link>
        </header>

        <section className="grid min-h-[calc(100dvh-10rem)] items-center gap-12 py-12 lg:grid-cols-[minmax(0,0.86fr)_minmax(0,1.14fr)] lg:py-14">
          <div className="max-w-3xl">
            <div className="inline-flex items-center gap-2 rounded-full border border-foreground/10 bg-[#fffdf9]/76 px-3 py-1 text-xs font-semibold text-muted-foreground shadow-[0_10px_26px_rgba(23,19,16,0.04)]">
              <LandingIcon name="clipboardText" className="size-4 text-primary" />
              School math support, from chat to follow-up
            </div>
            <h1 className="mt-7 max-w-[13ch] text-[4rem] leading-[0.88] font-semibold tracking-[-0.075em] text-balance sm:text-[5.8rem] lg:text-[6.75rem]">
              See who needs help next.
            </h1>
            <p className="mt-8 max-w-xl text-lg leading-8 text-muted-foreground sm:text-xl">
              P&amp;AI helps students work through math in WhatsApp and Telegram, then shows teachers which students and topics need attention before the class moves on.
            </p>

            <div className="mt-9 flex flex-col gap-3 sm:flex-row">
              <Link href={primaryHref} className={cn(landingButtonClassName, landingPrimaryButtonClassName)}>
                {primaryActionLabel}
                <LandingIcon name="arrowRight" className="ml-2 size-4" />
              </Link>
            </div>

            <div className="mt-12 grid gap-3 sm:grid-cols-3">
              {heroSignals.map((signal) => (
                <div key={signal.label} className="border-t border-foreground/12 pt-4">
                  <LandingIcon name={signal.icon} className="text-primary" />
                  <p className="mt-3 text-xs font-semibold tracking-[0.18em] uppercase text-muted-foreground">{signal.label}</p>
                  <p className="mt-2 text-sm font-semibold leading-5 text-foreground">{signal.value}</p>
                </div>
              ))}
            </div>
          </div>

          <LandingLiveDemo />
        </section>

        <section className="pb-14 lg:pb-18">
          <div className="grid gap-4 py-6 md:grid-cols-[1.12fr_0.88fr_1fr]">
            {dailyLoop.map((item) => (
              <div key={item.label} className="grid grid-cols-[4.5rem_1fr] gap-4 border-t border-foreground/10 pt-5 md:grid-cols-1">
                <p className="text-[11px] font-semibold tracking-[0.2em] uppercase text-primary">{item.label}</p>
                <div>
                  <h2 className="text-2xl font-semibold leading-tight text-foreground">{item.title}</h2>
                  <p className="mt-2 max-w-sm text-sm leading-6 text-muted-foreground">{item.body}</p>
                </div>
              </div>
            ))}
          </div>
        </section>

        <section className="pb-16 lg:pb-24">
          <div className="grid gap-10 lg:grid-cols-[minmax(0,0.72fr)_minmax(0,1.28fr)]">
            <div>
              <p className="text-xs font-semibold tracking-[0.24em] uppercase text-primary">How it works</p>
              <h2 className="mt-4 max-w-lg text-4xl leading-none font-semibold text-balance sm:text-5xl">
                One loop from student question to teacher action.
              </h2>
              <p className="mt-5 max-w-md text-lg leading-8 text-muted-foreground">
                P&amp;AI is not another dashboard students need to open. It starts where they already ask for help, then gives the teacher the useful part.
              </p>
            </div>

            <div className="grid gap-4 sm:grid-cols-2">
              <div className="sm:col-span-2">
                <LandingWorkflowSpotlight />
              </div>
              {workflowItems.map((item) => (
                <article
                  key={item.label}
                  className="rounded-[24px] border border-[#dfd2c2] bg-[#fffdf9]/82 p-5 shadow-[0_14px_36px_rgba(23,19,16,0.05)]"
                >
                  <LandingIcon name={item.icon} className="text-primary" />
                  <p className="mt-5 text-[11px] font-semibold tracking-[0.2em] uppercase text-muted-foreground">{item.label}</p>
                  <h3 className="mt-2 text-2xl font-semibold leading-tight text-foreground">{item.title}</h3>
                  <p className="mt-3 text-sm leading-6 text-muted-foreground">{item.body}</p>
                </article>
              ))}
            </div>
          </div>
        </section>

        <section className="pb-16 lg:pb-24">
          <div className="grid gap-8 lg:grid-cols-[minmax(0,1fr)_minmax(18rem,0.44fr)] lg:items-start">
            <div>
              <p className="text-xs font-semibold tracking-[0.24em] uppercase text-primary">Teacher outcomes</p>
              <h2 className="mt-4 max-w-3xl text-4xl leading-none font-semibold text-balance sm:text-5xl">
                Less guessing after practice.
              </h2>
            </div>
            <p className="text-lg leading-8 text-muted-foreground">
              The product stays useful because every student interaction has a clear destination: better follow-up from the teacher.
            </p>
          </div>

          <div className="mt-8 grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(18rem,0.42fr)]">
            <div className="grid gap-4 md:grid-cols-[1.12fr_0.88fr_1fr]">
              {teacherOutcomes.map((item) => (
                <article key={item.value} className="border-t border-foreground/12 pt-5">
                  <p className="font-mono text-4xl font-semibold text-primary">{item.value}</p>
                  <h3 className="mt-5 text-2xl font-semibold leading-tight text-foreground">{item.label}</h3>
                  <p className="mt-3 text-sm leading-6 text-muted-foreground">{item.body}</p>
                </article>
              ))}
            </div>
            <LandingOutcomeRail />
          </div>
        </section>

        <LandingCommandStrip primaryHref={primaryHref} primaryActionLabel={primaryActionLabel} />

        <footer className="mt-auto border-t border-foreground/10 py-8">
          <div className="flex flex-col gap-5 text-sm text-muted-foreground md:flex-row md:items-center md:justify-between">
            <div className="flex flex-col gap-1">
              <p className="font-medium text-foreground">P&amp;AI Bot</p>
              <p>Math chatbot for WhatsApp, Telegram, and schools.</p>
            </div>
            <Link href={primaryHref} className="inline-flex items-center gap-2 font-semibold text-foreground transition hover:text-primary">
              {primaryActionLabel}
              <LandingIcon name="arrowRight" className="size-4" />
            </Link>
          </div>
        </footer>
      </div>
    </main>
  );
}
