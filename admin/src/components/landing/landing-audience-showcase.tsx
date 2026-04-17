"use client";

import Image from "next/image";
import Link from "next/link";
import { useState } from "react";
import { AnimatePresence, motion } from "framer-motion";
import { ArrowRight } from "lucide-react";
import { Card } from "@/components/ui/card";
import { cn } from "@/lib/utils";

const landingButtonClassName =
  "inline-flex shrink-0 items-center justify-center whitespace-nowrap transition-all outline-none focus-visible:ring-3 focus-visible:ring-ring/30";

const landingPrimaryButtonClassName =
  "h-12 rounded-full bg-primary px-6 text-sm font-semibold text-primary-foreground shadow-[0_12px_30px_color-mix(in_oklch,var(--primary)_18%,transparent)] hover:bg-primary/90 active:translate-y-px";

type AudienceKey = "student" | "teacher";
type Tone = "destructive" | "secondary" | "primary";

type ChatTurn = {
  speaker: "Student" | "Bot";
  body: string;
};

type Stat = {
  label: string;
  value: string;
};

type Intervention = {
  name: string;
  topic: string;
  score: string;
  tone: Tone;
};

type TopicScore = {
  name: string;
  value: number;
  tone: Tone;
};

type AudienceFrame = {
  tabLabel: string;
  heroTitle: string;
  heroBody: string;
  imageSrc: string;
  imageAlt: string;
  panelTitle: string;
};

type StudentAudienceView = AudienceFrame & {
  summaryStats: Stat[];
  chatLabel: string;
  chatTurns: ChatTurn[];
};

type TeacherAudienceView = AudienceFrame & {
  summaryStats: Stat[];
  interventions: Intervention[];
  topicScores: TopicScore[];
};

const audienceViews = {
  student: {
    tabLabel: "Students",
    heroTitle: "Learn math in chat.",
    heroBody: "Ask. See the steps. Ask again.",
    imageSrc: "/landing/student-abstract-scene.svg",
    imageAlt: "Abstract chat learning visual with message blocks and step-by-step flow.",
    panelTitle: "Stay with the problem.",
    summaryStats: [
      { label: "Channel", value: "WhatsApp / Telegram" },
      { label: "Style", value: "Step by step" },
      { label: "Flow", value: "Same thread" },
    ],
    chatLabel: "Functions",
    chatTurns: [
      { speaker: "Student", body: "How do I solve 3x + 5 = 20?" },
      { speaker: "Bot", body: "Subtract 5 first. What is 20 - 5?" },
      { speaker: "Student", body: "15" },
      { speaker: "Bot", body: "Good. Now divide by 3. So x = 5." },
    ],
  },
  teacher: {
    tabLabel: "Teachers",
    heroTitle: "See the next follow-up.",
    heroBody: "Weak chats. Weak topics. Clear follow-up.",
    imageSrc: "/landing/teacher-abstract-scene.svg",
    imageAlt: "Abstract school dashboard visual with queues, score bars, and follow-up signals.",
    panelTitle: "Follow-up, in order.",
    summaryStats: [
      { label: "Needs help", value: "3 students" },
      { label: "Weakest topic", value: "Functions" },
      { label: "Coverage", value: "12 / 12 filled" },
    ],
    interventions: [
      { name: "Alya Sofea", topic: "WhatsApp • Functions", score: "30%", tone: "destructive" },
      { name: "Hakim Firdaus", topic: "Telegram • Inequalities", score: "21%", tone: "secondary" },
      { name: "Mei Lin", topic: "WhatsApp • Linear equations", score: "92%", tone: "primary" },
    ],
    topicScores: [
      { name: "Functions", value: 30, tone: "destructive" },
      { name: "Inequalities", value: 21, tone: "secondary" },
      { name: "Linear equations", value: 92, tone: "primary" },
    ],
  },
} satisfies {
  student: StudentAudienceView;
  teacher: TeacherAudienceView;
};

type AudienceView = StudentAudienceView | TeacherAudienceView;

function LandingSurface({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <Card
      className={cn(
        "overflow-hidden rounded-[32px] border border-border/80 bg-card shadow-[0_16px_44px_color-mix(in_oklch,var(--foreground)_6%,transparent)] ring-1 ring-foreground/5 backdrop-blur",
        className,
      )}
    >
      {children}
    </Card>
  );
}

function AudienceSwitch({
  activeAudience,
  onChange,
}: {
  activeAudience: AudienceKey;
  onChange: (value: AudienceKey) => void;
}) {
  return (
    <div className="flex justify-center">
      <div
        className="inline-flex rounded-full border border-foreground/10 bg-foreground/[0.05] p-1 shadow-[0_16px_36px_color-mix(in_oklch,var(--foreground)_6%,transparent)] backdrop-blur"
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
                "relative min-w-[7.25rem] rounded-full px-4 py-2 text-sm font-medium transition focus-visible:ring-2 focus-visible:ring-ring/40 focus-visible:outline-none",
                isActive ? "text-background" : "text-foreground/58 hover:text-foreground",
              )}
            >
              {isActive ? (
                <motion.span
                  layoutId="landing-audience-pill"
                  className="absolute inset-0 rounded-full bg-foreground shadow-[0_8px_22px_color-mix(in_oklch,var(--foreground)_18%,transparent)]"
                  transition={{ type: "spring", stiffness: 320, damping: 30 }}
                />
              ) : null}
              <span className="relative z-10">{audience.tabLabel}</span>
            </button>
          );
        })}
      </div>
    </div>
  );
}

function AudienceImage({ src, alt }: { src: string; alt: string }) {
  return (
    <div className="space-y-4">
      <div className="overflow-hidden rounded-[28px] border border-border/80 bg-background/80">
        <motion.div layout transition={{ type: "spring", stiffness: 220, damping: 26 }}>
          <Image src={src} alt={alt} width={1152} height={768} className="h-auto w-full object-cover" />
        </motion.div>
      </div>
    </div>
  );
}

function SummaryList({ items }: { items: Stat[] }) {
  return (
    <div className="grid gap-3 md:grid-cols-3">
      {items.map((item, index) => (
        <motion.div
          key={item.label}
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.06 * index, duration: 0.24 }}
          className="rounded-[22px] border border-border/80 bg-background/72 px-4 py-4"
        >
          <p className="text-[11px] font-semibold tracking-[0.16em] uppercase text-muted-foreground">{item.label}</p>
          <p className="mt-2 text-lg font-semibold tracking-[-0.03em] text-foreground">{item.value}</p>
        </motion.div>
      ))}
    </div>
  );
}

function LiveThreadBadge() {
  return (
    <div className="inline-flex items-center gap-2 rounded-full border border-border/70 bg-background/70 px-3 py-1 text-xs font-medium text-muted-foreground">
      <motion.span
        aria-hidden="true"
        className="size-2 rounded-full bg-primary"
        animate={{ opacity: [0.35, 1, 0.35], scale: [0.92, 1.18, 0.92] }}
        transition={{ duration: 1.6, repeat: Infinity, ease: "easeInOut" }}
      />
      Live thread
    </div>
  );
}

function TypingDots() {
  return (
    <div className="flex items-center gap-1.5">
      {[0, 1, 2].map((dot) => (
        <motion.span
          key={dot}
          className="size-1.5 rounded-full bg-current/50"
          animate={{ opacity: [0.3, 1, 0.3], y: [0, -1.5, 0] }}
          transition={{ duration: 1.1, delay: dot * 0.12, repeat: Infinity, ease: "easeInOut" }}
        />
      ))}
    </div>
  );
}

function ChatBubble({
  turn,
  index,
  isLast,
}: {
  turn: ChatTurn;
  index: number;
  isLast: boolean;
}) {
  const isStudent = turn.speaker === "Student";

  return (
    <motion.div
      key={`${turn.speaker}-${index}`}
      initial={{ opacity: 0, y: 18, x: isStudent ? 16 : -16, scale: 0.98 }}
      animate={{ opacity: 1, y: 0, x: 0, scale: 1 }}
      transition={{ delay: 0.08 * index, duration: 0.34, ease: [0.22, 1, 0.36, 1] }}
      className={cn(
        "max-w-[88%] rounded-[20px] px-4 py-3 text-sm leading-6 shadow-[0_10px_26px_color-mix(in_oklch,var(--foreground)_4%,transparent)]",
        isStudent
          ? "ml-auto bg-primary text-primary-foreground"
          : "bg-card text-foreground ring-1 ring-border/70",
      )}
    >
      <p className="text-[11px] font-semibold tracking-[0.16em] uppercase opacity-70">{turn.speaker}</p>
      <p className="mt-1">{turn.body}</p>
      {!isStudent && isLast ? <div className="mt-3 text-muted-foreground"><TypingDots /></div> : null}
    </motion.div>
  );
}

function ChatTranscript({ turns, label }: { turns: ChatTurn[]; label: string }) {
  return (
    <div className="relative overflow-hidden rounded-[26px] border border-border/80 bg-background/72 p-4">
      <motion.div
        aria-hidden="true"
        className="pointer-events-none absolute inset-y-0 left-0 w-24 bg-[linear-gradient(90deg,color-mix(in_oklch,var(--primary)_7%,transparent),transparent)]"
        animate={{ x: ["-18%", "132%"] }}
        transition={{ duration: 1.8, ease: "easeInOut", repeat: Infinity, repeatDelay: 2.8 }}
      />

      <div className="relative flex items-center justify-between gap-3">
        <div>
          <p className="text-sm font-semibold text-foreground">Example student chat</p>
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
            isLast={index === turns.length - 1}
          />
        ))}
      </div>
    </div>
  );
}

function InterventionRow({ item }: { item: Intervention }) {
  return (
    <div className="flex items-center justify-between gap-3 rounded-[20px] border border-border/70 bg-background/90 px-4 py-3">
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
    <div className="rounded-[26px] border border-border/80 bg-background/72 p-5">
      <div className="flex items-center justify-between gap-3">
        <p className="text-sm font-semibold text-foreground">Topic scores</p>
        <div className="rounded-full border border-border/70 bg-background/70 px-3 py-1 text-xs font-medium text-muted-foreground">
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
              <motion.div
                initial={{ width: 0 }}
                animate={{ width: `${item.value}%` }}
                transition={{ duration: 0.45, ease: "easeOut" }}
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
    <div className="rounded-[26px] border border-border/80 bg-background/72 p-4">
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
    <motion.section
      key={panelId}
      id={panelId}
      initial={{ opacity: 0, y: 14 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -10 }}
      transition={{ duration: 0.28, ease: [0.22, 1, 0.36, 1] }}
      className="space-y-8"
    >
      <div className="mt-7 flex flex-col gap-3">
        <h1 className="text-5xl leading-[0.9] font-semibold tracking-[-0.07em] text-balance sm:text-6xl lg:text-[5.6rem]">
          {audience.heroTitle}
        </h1>
        <p className="mx-auto max-w-md text-lg leading-8 text-muted-foreground">{audience.heroBody}</p>
      </div>

      <div className="mt-7 flex items-center justify-center">
        <Link href={primaryHref} className={cn(landingButtonClassName, landingPrimaryButtonClassName)}>
          {primaryActionLabel}
        </Link>
      </div>

      <section className="pb-8">
        <LandingSurface className="rounded-[36px] border-border/80 bg-card p-6 shadow-[0_14px_36px_color-mix(in_oklch,var(--foreground)_5%,transparent)]">
          <div className="grid gap-6 lg:grid-cols-[minmax(0,0.98fr)_minmax(0,1.02fr)]">
            <AudienceImage src={audience.imageSrc} alt={audience.imageAlt} />

            <div className="space-y-4 text-left">
              <div className="max-w-xl">
                <h2 className="text-[2.15rem] leading-tight font-semibold tracking-[-0.06em]">{audience.panelTitle}</h2>
              </div>

              {children}
            </div>
          </div>
        </LandingSurface>
      </section>
    </motion.section>
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
      <ChatTranscript turns={audience.chatTurns} label={audience.chatLabel} />
      <SummaryList items={audience.summaryStats} />
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
      <SummaryList items={audience.summaryStats} />
      <TopicScoreList items={audience.topicScores} />
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
    <div className="space-y-8">
      <AudienceSwitch activeAudience={activeAudience} onChange={setActiveAudience} />

      <AnimatePresence mode="wait" initial={false}>
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
      </AnimatePresence>
    </div>
  );
}
