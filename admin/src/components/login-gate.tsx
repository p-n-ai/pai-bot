"use client";

import { LoginGateProvider } from "@/components/login-gate/login-gate-provider";
import { LoginGateAuthPanel } from "@/components/login-gate/login-gate-auth-panel";
import { DarkLoginGateBackdrop } from "@/components/login-gate/dark-login-gate-backdrop";
import { LightLoginGateBackdrop } from "@/components/login-gate/light-login-gate-backdrop";
import { LoginGateForm } from "@/components/login-gate/login-gate-form";
import { LoginGateHeroSection } from "@/components/login-gate/login-gate-hero-section";
import { LoginGateShell } from "@/components/login-gate/login-gate-shell";

export function LoginGate({ nextPath = null }: { nextPath?: string | null }) {
  return (
    <LoginGateProvider nextPath={nextPath}>
      <LoginGateShell>
        <LoginGateHeroSection heroSectionClassName="bg-[#edf4ff] dark:bg-[#06101d]">
          <div data-testid="login-gate-light-backdrop" className="dark:hidden">
            <LightLoginGateBackdrop />
          </div>
          <div data-testid="login-gate-dark-backdrop" className="hidden dark:block">
            <DarkLoginGateBackdrop />
          </div>
        </LoginGateHeroSection>
        <LoginGateAuthPanel>
          <LoginGateForm />
        </LoginGateAuthPanel>
      </LoginGateShell>
    </LoginGateProvider>
  );
}
