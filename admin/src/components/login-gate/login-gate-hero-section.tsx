"use client";

import { motion, useReducedMotion } from "framer-motion";

const heroTitle = "Know where every learner needs help.";
const heroWords = ["Know", "where", "every", "learner", "needs", "help."];
const easeOutQuint: [number, number, number, number] = [0.23, 1, 0.32, 1];

export function LoginGateHeroSection({
  children,
  heroSectionClassName,
}: {
  children: React.ReactNode;
  heroSectionClassName: string;
}) {
  const prefersReducedMotion = useReducedMotion();

  return (
    <section
      aria-labelledby="login-gate-hero-title"
      className={`relative order-2 min-h-80 overflow-hidden px-6 py-12 sm:px-8 lg:order-1 lg:min-h-0 lg:px-12 lg:py-12 ${heroSectionClassName}`}
    >
      <div className="pointer-events-none absolute inset-0 overflow-hidden">{children}</div>
      <div className="relative flex h-full flex-col justify-center">
        <div className="max-w-xl space-y-5">
          <motion.h2
            id="login-gate-hero-title"
            initial={prefersReducedMotion ? false : "hidden"}
            animate="visible"
            aria-label={heroTitle}
            className="max-w-xl text-4xl leading-[0.95] font-semibold tracking-[-0.04em] text-foreground md:text-5xl lg:text-6xl"
          >
            <span className="sr-only">{heroTitle}</span>
            <span aria-hidden="true">
              {heroWords.map((word, index) => (
                <motion.span
                  key={word}
                  variants={{
                    hidden: { opacity: 0, y: "0.6em", filter: "blur(10px)" },
                    visible: {
                      opacity: 1,
                      y: 0,
                      filter: "blur(0px)",
                      transition: {
                        duration: 0.48,
                        delay: 0.08 + index * 0.05,
                        ease: easeOutQuint,
                      },
                    },
                  }}
                  className="inline-block pr-[0.22em] will-change-transform"
                >
                  {word}
                </motion.span>
              ))}
            </span>
          </motion.h2>
          <p className="max-w-3xl text-base leading-8 text-muted-foreground text-pretty md:text-lg">
            See progress, spot gaps, and support the right student sooner.
          </p>
        </div>
      </div>
    </section>
  );
}
