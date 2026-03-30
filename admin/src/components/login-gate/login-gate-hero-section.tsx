"use client";

import Link from "next/link";
import { buttonVariants } from "@/components/ui/button";

export function LoginGateHeroSection({
  children,
  heroSectionClassName,
}: {
  children: React.ReactNode;
  heroSectionClassName: string;
}) {
  return (
    <section className={`relative overflow-hidden px-8 py-10 lg:px-12 lg:py-12 ${heroSectionClassName}`}>
      <div className="pointer-events-none absolute inset-0 overflow-hidden">{children}</div>
      <div className="relative flex h-full flex-col justify-center">
        <div className="max-w-xl space-y-6">
          <div className="space-y-5">
            <h1 className="max-w-xl text-4xl leading-[0.95] font-semibold tracking-[-0.04em] text-slate-950 md:text-5xl lg:text-6xl dark:text-white">
              See who needs help before the exam.
            </h1>
            <p className="max-w-3xl text-base leading-8 text-slate-600 md:text-lg dark:text-slate-300">
              P&amp;AI is a proactive AI learning agent that teaches students through chat. This workspace gives teachers, parents, and school admins visibility into mastery, momentum, and the right moment to intervene.
            </p>
          </div>
          <div className="pt-0">
            <Link
              href="#sign-in-form"
              className={buttonVariants({
                variant: "default",
                size: "lg",
                className:
                  "h-11 rounded-full px-5 text-sm font-semibold shadow-[0_18px_44px_rgba(15,23,42,0.16)] hover:shadow-[0_22px_56px_rgba(15,23,42,0.2)] active:translate-y-px dark:shadow-[0_20px_54px_rgba(2,8,23,0.36)]",
              })}
            >
              Try now
            </Link>
          </div>
        </div>
      </div>
    </section>
  );
}
