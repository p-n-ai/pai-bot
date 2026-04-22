"use client";

import Link from "next/link";
import { useState } from "react";
import { ArrowRight } from "lucide-react";
import { cn } from "@/lib/utils";
import {
  type AudienceFrame,
  type AudienceKey,
  type AudienceView,
  type ChatTurn,
  type Intervention,
  type Stat,
  type StudentAudienceView,
  type TeacherAudienceView,
  type TopicScore,
  audienceViews,
} from "@/components/landing/landing-audience-data";

const landingButtonClassName =
  "inline-flex shrink-0 items-center justify-center whitespace-nowrap transition-all outline-none focus-visible:ring-3 focus-visible:ring-ring/30";

const landingPrimaryButtonClassName =
  "h-11 rounded-full bg-primary px-5 text-sm font-semibold text-primary-foreground shadow-[0_12px_28px_color-mix(in_oklch,var(--primary)_18%,transparent)] hover:bg-primary/90 active:translate-y-px";

function AudienceSwitch({
  activeAudience,
  onChange,
}: {
  activeAudience: AudienceKey;
  onChange: (value: AudienceKey) => void;
}) {
  return (
    <div className="flex">
      <div
        className="inline-flex rounded-full border border-foreground/10 bg-white/62 p-1 shadow-[0_16px_36px_rgba(23,19,16,0.06)] backdrop-blur"
        role="tablist"
        aria-label="Landing audience"
      >
        {(Object.entries(audienceViews) as [AudienceKey, AudienceView][]).map(([key, audience]) => {
          const isActive = key === activeAudience;

          return (
            <button
              key={key}
              type="button"
              role="tab"
              aria-selected={isActive}
              aria-controls={`${key}-panel`}
              onClick={() => onChange(key)}
              className={cn(
                "relative min-w-[7.25rem] rounded-full px-4 py-2 text-sm font-semibold transition focus-visible:ring-2 focus-visible:ring-ring/40 focus-visible:outline-none",
                isActive ? "text-primary-foreground" : "text-foreground/58 hover:text-foreground",
              )}
            >
              {isActive ? (
                <span className="absolute inset-0 rounded-full bg-primary shadow-[0_8px_22px_color-mix(in_oklch,var(--primary)_20%,transparent)]" />
              ) : null}
              <span className="relative z-10">{audience.tabLabel}</span>
            </button>
          );
        })}
      </div>
    </div>
  );
}

function SummaryList({ items }: { items: Stat[] }) {
  return (
    <div className="grid gap-3 sm:grid-cols-3 lg:grid-cols-1">
      {items.map((item) => (
        <div
          key={item.label}
          className="border-t border-foreground/12 pt-4 sm:min-h-[5.5rem] lg:min-h-0"
        >
          <p className="text-[11px] font-semibold tracking-[0.18em] uppercase text-muted-foreground">{item.label}</p>
          <p className="mt-2 text-xl font-semibold text-foreground">{item.value}</p>
        </div>
      ))}
    </div>
  );
}

function LiveThreadBadge() {
  return (
    <div className="inline-flex items-center gap-2 rounded-full border border-foreground/10 bg-white/70 px-3 py-1 text-xs font-medium text-muted-foreground">
      <span aria-hidden="true" className="size-2 rounded-full bg-primary" />
      Live thread
    </div>
  );
}

function ChatBubble({
  turn,
  index,
}: {
  turn: ChatTurn;
  index: number;
}) {
  const isStudent = turn.speaker === "Student";

  return (
    <div
      key={`${turn.speaker}-${index}`}
      className={cn(
        "max-w-[88%] rounded-[20px] px-4 py-3 text-sm leading-6 shadow-[0_10px_26px_color-mix(in_oklch,var(--foreground)_4%,transparent)]",
        isStudent
          ? "ml-auto bg-primary text-primary-foreground"
          : "bg-white/82 text-foreground ring-1 ring-foreground/10",
      )}
    >
      <p className="text-[11px] font-semibold tracking-[0.16em] uppercase opacity-70">{turn.speaker}</p>
      <p className="mt-1">{turn.body}</p>
    </div>
  );
}

function ChatTranscript({ turns, label }: { turns: ChatTurn[]; label: string }) {
  return (
    <div className="relative min-h-full overflow-hidden rounded-[26px] border border-foreground/10 bg-white/64 p-4 shadow-[0_18px_46px_rgba(23,19,16,0.05)]">
      <div className="relative flex items-center justify-between gap-3">
        <div>
          <p className="text-sm font-semibold text-foreground">Example tutoring thread</p>
          <p className="text-sm text-muted-foreground">{label}</p>
        </div>
        <LiveThreadBadge />
      </div>

      <div className="relative mt-4 flex flex-col gap-3">
        {turns.map((turn, index) => (
          <ChatBubble
            key={`${turn.speaker}-${index}`}
            turn={turn}
            index={index}
          />
        ))}
      </div>
    </div>
  );
}

function InterventionRow({ item }: { item: Intervention }) {
  return (
    <div className="flex items-center justify-between gap-3 rounded-[20px] border border-foreground/10 bg-white/76 px-4 py-3">
      <div>
        <p className="font-medium text-foreground">{item.name}</p>
        <p className="text-sm text-muted-foreground">{item.topic}</p>
      </div>
      <div
        className={cn(
          "rounded-full px-3 py-1 text-sm font-semibold",
          item.tone === "destructive" && "bg-destructive/12 text-destructive",
          item.tone === "secondary" && "bg-secondary text-secondary-foreground",
          item.tone === "primary" && "bg-primary/12 text-primary",
        )}
      >
        {item.score}
      </div>
    </div>
  );
}

function TopicScoreList({ items }: { items: TopicScore[] }) {
  return (
    <div className="rounded-[26px] border border-foreground/10 bg-white/64 p-5 shadow-[0_18px_46px_rgba(23,19,16,0.04)]">
      <div className="flex items-center justify-between gap-3">
        <p className="text-sm font-semibold text-foreground">Topic scores</p>
        <div className="rounded-full border border-foreground/10 bg-white/76 px-3 py-1 text-xs font-medium text-muted-foreground">
          12 / 12 filled
        </div>
      </div>

      <div className="mt-4 flex flex-col gap-3">
        {items.map((item) => (
          <div key={item.name} className="flex flex-col gap-2">
            <div className="flex items-center justify-between gap-3 text-sm">
              <span className="font-medium text-foreground">{item.name}</span>
              <span className="text-muted-foreground">{item.value}%</span>
            </div>
            <div className="h-2 rounded-full bg-muted">
              <div
                style={{ width: `${item.value}%` }}
                className={cn(
                  "h-2 rounded-full",
                  item.tone === "destructive" && "bg-destructive",
                  item.tone === "secondary" && "bg-foreground/45",
                  item.tone === "primary" && "bg-primary",
                )}
              />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function TeacherInterventionList({
  items,
  primaryHref,
}: {
  items: Intervention[];
  primaryHref: string;
}) {
  return (
    <div className="rounded-[26px] border border-foreground/10 bg-white/64 p-4 shadow-[0_18px_46px_rgba(23,19,16,0.04)]">
      <div className="flex items-center justify-between gap-3">
        <div>
          <p className="text-sm font-semibold text-foreground">Chats to follow up with</p>
        </div>
        <Link href={primaryHref} className="inline-flex items-center gap-2 text-sm font-semibold text-primary hover:text-primary/80">
          Open workspace
          <ArrowRight className="h-4 w-4" />
        </Link>
      </div>

      <div className="mt-4 flex flex-col gap-3">
        {items.map((item) => (
          <InterventionRow key={item.name} item={item} />
        ))}
      </div>
    </div>
  );
}

function AudiencePanelFrame({
  panelId,
  audience,
  primaryHref,
  primaryActionLabel,
  children,
}: {
  panelId: string;
  audience: AudienceFrame;
  primaryHref: string;
  primaryActionLabel: string;
  children: React.ReactNode;
}) {
  return (
    <section
      key={panelId}
      id={panelId}
      className="grid gap-8 lg:grid-cols-[minmax(0,0.76fr)_minmax(0,1.24fr)] lg:items-stretch"
    >
      <div className="flex min-h-full flex-col justify-between gap-8 rounded-[26px] bg-white/58 p-5 shadow-[inset_0_0_0_1px_rgba(23,19,16,0.05)] sm:p-7">
        <div>
          <p className="text-xs font-semibold tracking-[0.24em] uppercase text-primary">{audience.kicker}</p>
          <h2 className="mt-5 max-w-[12ch] text-4xl leading-none font-semibold text-balance sm:text-5xl lg:text-[3.45rem]">
            {audience.title}
          </h2>
          <p className="mt-6 max-w-md text-lg leading-8 text-muted-foreground">{audience.body}</p>
        </div>

        <div>
          <p className="text-[11px] font-semibold tracking-[0.2em] uppercase text-muted-foreground">{audience.proof}</p>
          <h3 className="mt-4 text-2xl leading-tight font-semibold sm:text-3xl">{audience.panelTitle}</h3>
        </div>

        <Link href={primaryHref} className={cn(landingButtonClassName, landingPrimaryButtonClassName)}>
          {primaryActionLabel}
          <ArrowRight className="ml-2 size-4" />
        </Link>
      </div>

      <div className="grid gap-4 lg:grid-cols-[minmax(12rem,0.58fr)_minmax(0,1fr)]">
        {children}
      </div>
    </section>
  );
}

function StudentAudiencePanel({
  audience,
  primaryHref,
  primaryActionLabel,
}: {
  audience: StudentAudienceView;
  primaryHref: string;
  primaryActionLabel: string;
}) {
  return (
    <AudiencePanelFrame
      panelId="student-panel"
      audience={audience}
      primaryHref={primaryHref}
      primaryActionLabel={primaryActionLabel}
    >
      <SummaryList items={audience.summaryStats} />
      <ChatTranscript turns={audience.chatTurns} label={audience.chatLabel} />
    </AudiencePanelFrame>
  );
}

function TeacherAudiencePanel({
  audience,
  primaryHref,
  primaryActionLabel,
}: {
  audience: TeacherAudienceView;
  primaryHref: string;
  primaryActionLabel: string;
}) {
  return (
    <AudiencePanelFrame
      panelId="teacher-panel"
      audience={audience}
      primaryHref={primaryHref}
      primaryActionLabel={primaryActionLabel}
    >
      <TeacherInterventionList items={audience.interventions} primaryHref={primaryHref} />
      <div className="space-y-4">
        <SummaryList items={audience.summaryStats} />
        <TopicScoreList items={audience.topicScores} />
      </div>
    </AudiencePanelFrame>
  );
}

export function LandingAudienceShowcase({
  primaryHref,
  primaryActionLabel,
}: {
  primaryHref: string;
  primaryActionLabel: string;
}) {
  const [activeAudience, setActiveAudience] = useState<AudienceKey>("student");

  return (
    <div>
      <div className="grid gap-7 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-end">
        <div className="max-w-3xl">
          <p className="text-xs font-semibold tracking-[0.24em] uppercase text-primary">Classroom loop</p>
          <h2 className="mt-4 text-4xl leading-none font-semibold text-balance sm:text-5xl">
            Practice in chat. Follow up in class.
          </h2>
          <p className="mt-5 max-w-2xl text-lg leading-8 text-muted-foreground">
            One question becomes a guided thread for the student, then a short repair list for the teacher.
          </p>
        </div>
        <div className="flex lg:justify-end">
          <AudienceSwitch activeAudience={activeAudience} onChange={setActiveAudience} />
        </div>
      </div>

      <div className="mt-10">
        {activeAudience === "student" ? (
          <StudentAudiencePanel
            key="student"
            audience={audienceViews.student}
            primaryHref={primaryHref}
            primaryActionLabel={primaryActionLabel}
          />
        ) : (
          <TeacherAudiencePanel
            key="teacher"
            audience={audienceViews.teacher}
            primaryHref={primaryHref}
            primaryActionLabel={primaryActionLabel}
          />
        )}
      </div>
    </div>
  );
}
