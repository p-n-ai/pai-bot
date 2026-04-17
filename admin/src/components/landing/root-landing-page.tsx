import Link from "next/link";
import { Bot } from "lucide-react";
import { cn } from "@/lib/utils";
import { LandingAudienceShowcase } from "@/components/landing/landing-audience-showcase";

const landingButtonClassName =
  "inline-flex shrink-0 items-center justify-center whitespace-nowrap transition-all outline-none focus-visible:ring-3 focus-visible:ring-ring/30";

const landingHeaderPrimaryButtonClassName =
  "h-9 rounded-full bg-primary px-5 text-sm font-semibold text-primary-foreground shadow-[0_10px_24px_color-mix(in_oklch,var(--primary)_16%,transparent)] hover:bg-primary/90 active:translate-y-px";

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
    <main className="relative min-h-[100dvh] overflow-hidden bg-background text-foreground">
      <div className="pointer-events-none absolute inset-x-0 top-0 -z-10 h-[28rem] bg-[linear-gradient(180deg,color-mix(in_oklch,var(--primary)_12%,transparent)_0%,transparent_72%)]" />
      <div className="pointer-events-none absolute left-[8%] top-20 -z-10 size-72 rounded-full bg-primary/8 blur-3xl" />
      <div className="pointer-events-none absolute right-[10%] top-40 -z-10 size-80 rounded-full bg-secondary blur-3xl opacity-45" />

      <div className="mx-auto flex min-h-[100dvh] max-w-7xl flex-col px-6 py-0 sm:px-8 lg:px-10">
        <header className="mt-5 flex items-center justify-between gap-6 rounded-full border border-border/80 bg-background/92 px-5 py-3 shadow-[0_10px_28px_color-mix(in_oklch,var(--foreground)_5%,transparent)] backdrop-blur">
          <div className="flex items-center gap-3">
            <div className="flex size-11 items-center justify-center rounded-2xl bg-primary text-primary-foreground">
              <Bot className="size-5" />
            </div>
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-[0.28em] text-primary">P&amp;AI Bot</p>
              <p className="text-sm font-medium text-muted-foreground">Math chatbot for WhatsApp, Telegram, and schools</p>
            </div>
          </div>
          <Link
            href={primaryHref}
            className={cn(landingButtonClassName, landingHeaderPrimaryButtonClassName)}
          >
            {primaryActionLabel}
          </Link>
        </header>

        <section className="py-14 lg:py-18">
          <div className="mx-auto max-w-5xl text-center">
            <LandingAudienceShowcase
              primaryHref={primaryHref}
              primaryActionLabel={primaryActionLabel}
            />
          </div>
        </section>

        <footer className="mt-auto border-t border-border/80 py-8">
          <div className="flex flex-col gap-5 text-sm text-muted-foreground md:flex-row md:items-center md:justify-between">
            <div className="flex flex-col gap-1">
              <p className="font-medium text-foreground">P&amp;AI Bot</p>
              <p>Math chatbot for WhatsApp, Telegram, and schools.</p>
            </div>
            <Link href={primaryHref} className="transition hover:text-foreground">
              {primaryActionLabel}
            </Link>
          </div>
        </footer>
      </div>
    </main>
  );
}
