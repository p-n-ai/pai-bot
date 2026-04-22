"use client";

import {
  ArrowRight,
  BellRinging,
  ChatCircleText,
  CheckCircle,
  ClipboardText,
  ListChecks,
  TrendUp,
} from "@phosphor-icons/react";
import Link from "next/link";
import Image from "next/image";
import { cn } from "@/lib/utils";

const demoStudents = [
  { name: "Alya Sofea", topic: "Functions", score: "47.2%", tone: "bg-primary" },
  { name: "Haris Naufal", topic: "Inequalities", score: "38.6%", tone: "bg-[#6f6256]" },
  { name: "Mei Lin Tan", topic: "Linear graphs", score: "81.4%", tone: "bg-[#9b6a4f]" },
];

const demoSteps = [
  "Student sends a stuck question",
  "Tutor checks the next move",
  "Teacher gets a repair list",
];

const landingButtonClassName =
  "inline-flex shrink-0 items-center justify-center whitespace-nowrap transition-all outline-none focus-visible:ring-3 focus-visible:ring-ring/30";

export function LandingLiveDemo() {
  return (
    <div className="relative min-h-[31rem] lg:min-h-[38rem]">
      <div className="absolute top-0 right-0 w-[82%] overflow-hidden rounded-[2rem] border border-[#dfd2c2] bg-white shadow-[0_24px_74px_rgba(23,19,16,0.12)]">
        <Image
          src="/landing/teacher-abstract-scene.svg"
          alt="Teacher dashboard visual showing follow-up queues and topic score bars."
          width={1600}
          height={1066}
          priority
          className="aspect-[1.08] w-full object-cover"
        />
      </div>

      <div className="absolute bottom-8 left-0 w-[63%] overflow-hidden rounded-[1.75rem] border border-[#dfd2c2] bg-white shadow-[0_22px_64px_rgba(23,19,16,0.14)]">
        <Image
          src="/landing/student-abstract-scene.svg"
          alt="Student chat learning visual with guided math steps."
          width={1600}
          height={1066}
          className="aspect-[1.08] w-full object-cover"
        />
      </div>

      <div className="absolute right-[4%] bottom-[10%] w-[19rem] rounded-[2rem] border border-[#312722] bg-[#171310] p-5 text-[#fbf7f1] shadow-[0_22px_70px_rgba(23,19,16,0.3)]">
        <div className="flex items-start justify-between gap-4">
          <div>
            <p className="text-[11px] font-semibold tracking-[0.22em] uppercase text-[#f3b28f]">Live queue</p>
            <p className="mt-2 text-sm leading-5 text-white/64">Sorted by who needs attention first.</p>
          </div>
          <BellRinging className="size-5 animate-pulse text-[#f3b28f]" weight="duotone" />
        </div>

        <div className="mt-5 space-y-3">
          {demoStudents.map((student, index) => (
            <div
              key={student.name}
              className="grid grid-cols-[1fr_auto] gap-3 rounded-2xl border border-white/10 bg-white/[0.06] p-3 shadow-[inset_0_1px_0_rgba(255,255,255,0.08)]"
              style={{ animationDelay: `${index * 120}ms` }}
            >
              <div>
                <p className="text-sm font-semibold">{student.name}</p>
                <p className="mt-1 text-xs text-white/56">{student.topic}</p>
              </div>
              <div className="text-right">
                <p className="font-mono text-sm font-semibold">{student.score}</p>
                <span className={cn("mt-2 block h-1.5 w-12 rounded-full", student.tone)} />
              </div>
            </div>
          ))}
        </div>
      </div>

      <div className="absolute top-[13%] left-[2%] w-[18rem] rounded-[1.75rem] border border-[#dfd2c2] bg-[#fffdf9]/92 p-4 shadow-[0_18px_48px_rgba(23,19,16,0.1)] backdrop-blur">
        <div className="flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.18em] text-primary">
          <ChatCircleText className="size-4" weight="duotone" />
          Tutor thread
        </div>
        <div className="mt-4 rounded-2xl bg-[#f3eadf] px-4 py-3 text-sm text-[#312722]">
          How do I know which side to divide?
        </div>
        <div className="mt-3 flex items-center gap-2 rounded-2xl border border-[#dfd2c2] bg-white px-4 py-3 text-sm text-muted-foreground">
          <span className="size-2 animate-pulse rounded-full bg-primary" />
          Check the coefficient of x first
        </div>
      </div>

      <div className="absolute right-[17%] top-[4%] hidden rounded-full border border-[#dfd2c2] bg-[#fffdf9]/90 px-4 py-2 text-sm font-semibold text-foreground shadow-[0_14px_36px_rgba(23,19,16,0.08)] backdrop-blur sm:flex sm:items-center sm:gap-2">
        <CheckCircle className="size-4 text-primary" weight="fill" />
        12 of 14 students checked in
      </div>
    </div>
  );
}

export function LandingCommandStrip({
  primaryHref,
  primaryActionLabel,
}: {
  primaryHref: string;
  primaryActionLabel: string;
}) {
  return (
    <section className="pb-16">
      <div className="grid gap-6 rounded-[2rem] border border-[#dfd2c2] bg-[#fffdf9]/82 p-5 shadow-[0_18px_46px_rgba(23,19,16,0.06)] backdrop-blur sm:p-7 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-center">
        <div>
          <div className="flex items-center gap-2 text-sm font-semibold text-primary">
            <ClipboardText className="size-5" weight="duotone" />
            Built for daily classroom follow-up
          </div>
          <div className="mt-5 grid gap-3 md:grid-cols-3">
            {demoSteps.map((step, index) => (
              <div key={step} className="flex items-center gap-3 rounded-2xl border border-[#dfd2c2] bg-white/80 px-4 py-3">
                <span className="font-mono text-xs font-semibold text-primary">0{index + 1}</span>
                <span className="text-sm font-medium text-foreground">{step}</span>
              </div>
            ))}
          </div>
        </div>
        <Link
          href={primaryHref}
          className={cn(
            landingButtonClassName,
            "h-12 rounded-full bg-primary px-6 text-sm font-semibold text-primary-foreground shadow-[0_14px_30px_color-mix(in_oklch,var(--primary)_18%,transparent)] hover:bg-primary/90 active:-translate-y-[1px]",
          )}
        >
          {primaryActionLabel}
          <ArrowRight className="ml-2 size-4" />
        </Link>
      </div>
    </section>
  );
}

export function LandingOutcomeRail() {
  return (
    <div className="relative overflow-hidden rounded-[2rem] border border-[#dfd2c2] bg-[#171310] p-5 text-[#fbf7f1] shadow-[0_18px_46px_rgba(23,19,16,0.16)]">
      <div className="flex items-center justify-between gap-4">
        <div>
          <p className="text-[11px] font-semibold uppercase tracking-[0.22em] text-[#f3b28f]">Tomorrow plan</p>
          <h3 className="mt-3 text-2xl font-semibold leading-tight">Repair first, then move.</h3>
        </div>
        <TrendUp className="size-6 text-[#f3b28f]" weight="duotone" />
      </div>
      <div className="mt-6 space-y-3">
        <div className="rounded-2xl border border-white/10 bg-white/[0.06] p-4">
          <p className="text-sm font-semibold">Warm-up</p>
          <p className="mt-1 text-sm text-white/60">Two function questions before the new topic.</p>
        </div>
        <div className="rounded-2xl border border-white/10 bg-white/[0.06] p-4">
          <p className="text-sm font-semibold">Small group</p>
          <p className="mt-1 text-sm text-white/60">Alya, Haris, and Amir review coefficient steps.</p>
        </div>
        <div className="rounded-2xl border border-white/10 bg-white/[0.06] p-4">
          <p className="text-sm font-semibold">Parent note</p>
          <p className="mt-1 text-sm text-white/60">Send one concrete practice link after class.</p>
        </div>
      </div>
    </div>
  );
}

export function LandingWorkflowSpotlight() {
  return (
    <div className="rounded-[2rem] border border-[#dfd2c2] bg-[#fffdf9]/82 p-5 shadow-[0_18px_46px_rgba(23,19,16,0.06)] sm:p-7">
      <div className="flex items-center gap-2 text-sm font-semibold text-primary">
        <ListChecks className="size-5" weight="duotone" />
        Teacher-ready evidence
      </div>
      <div className="mt-6 space-y-5">
        <div>
          <p className="text-sm font-semibold text-foreground">Functions slipped on step two</p>
          <div className="mt-3 h-2 overflow-hidden rounded-full bg-[#eadfce]">
            <div className="h-full w-[47%] rounded-full bg-primary" />
          </div>
        </div>
        <div className="grid gap-3 sm:grid-cols-2">
          <div className="rounded-2xl border border-[#dfd2c2] bg-white/78 p-4">
            <p className="font-mono text-2xl font-semibold text-primary">47.2%</p>
            <p className="mt-1 text-sm text-muted-foreground">median check score</p>
          </div>
          <div className="rounded-2xl border border-[#dfd2c2] bg-white/78 p-4">
            <p className="font-mono text-2xl font-semibold text-primary">8m</p>
            <p className="mt-1 text-sm text-muted-foreground">until first hint</p>
          </div>
        </div>
      </div>
    </div>
  );
}
