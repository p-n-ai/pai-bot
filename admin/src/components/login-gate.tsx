"use client";

import { motion, useReducedMotion } from "framer-motion";
import { LoginGateProvider } from "@/components/login-gate/login-gate-provider";
import { LoginGateAuthPanel } from "@/components/login-gate/login-gate-auth-panel";
import { DarkLoginGateBackdrop } from "@/components/login-gate/dark-login-gate-backdrop";
import { LightLoginGateBackdrop } from "@/components/login-gate/light-login-gate-backdrop";
import { LoginGateForm } from "@/components/login-gate/login-gate-form";
import { LoginGateHeroSection } from "@/components/login-gate/login-gate-hero-section";
import { LoginGateShell } from "@/components/login-gate/login-gate-shell";
import { useTheme } from "@/components/theme-provider";

const backdropEase = [0.22, 1, 0.36, 1] as const;

export function LoginGate({ nextPath = null }: { nextPath?: string | null }) {
  const { theme, mounted } = useTheme();
  const prefersReducedMotion = useReducedMotion();
  const isDark = mounted && theme === "dark";

  return (
    <LoginGateProvider nextPath={nextPath}>
      <LoginGateShell>
        <LoginGateHeroSection heroSectionClassName="bg-[#edf4ff] dark:bg-[#06101d]">
          <motion.div
            data-testid="login-gate-light-backdrop"
            initial={false}
            animate={
              prefersReducedMotion
                ? { opacity: isDark ? 0 : 1 }
                : {
                    opacity: isDark ? 0 : 1,
                    scale: isDark ? 1.04 : 1,
                    filter: isDark ? "blur(18px)" : "blur(0px)",
                  }
            }
            transition={{ duration: prefersReducedMotion ? 0.16 : 0.52, ease: backdropEase }}
            className="absolute inset-0"
            aria-hidden={isDark}
          >
            <LightLoginGateBackdrop />
          </motion.div>
          <motion.div
            data-testid="login-gate-dark-backdrop"
            initial={false}
            animate={
              prefersReducedMotion
                ? { opacity: isDark ? 1 : 0 }
                : {
                    opacity: isDark ? 1 : 0,
                    scale: isDark ? 1 : 0.97,
                    filter: isDark ? "blur(0px)" : "blur(14px)",
                  }
            }
            transition={{ duration: prefersReducedMotion ? 0.16 : 0.46, ease: backdropEase }}
            className="absolute inset-0"
            aria-hidden={!isDark}
          >
            <DarkLoginGateBackdrop />
          </motion.div>
        </LoginGateHeroSection>
        <LoginGateAuthPanel>
          <LoginGateForm />
        </LoginGateAuthPanel>
      </LoginGateShell>
    </LoginGateProvider>
  );
}
