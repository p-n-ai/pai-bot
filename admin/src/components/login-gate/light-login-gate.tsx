"use client";

import { LightLoginGateBackdrop } from "@/components/login-gate/light-login-gate-backdrop";
import { LoginGateAuthPanel } from "@/components/login-gate/login-gate-auth-panel";
import { LoginGateForm } from "@/components/login-gate/login-gate-form";
import { LoginGateHeroSection } from "@/components/login-gate/login-gate-hero-section";
import { LoginGateShell } from "@/components/login-gate/login-gate-shell";

export function LightLoginGate() {
  return (
    <LoginGateShell>
      <LoginGateHeroSection heroSectionClassName="bg-[#edf4ff]">
        <LightLoginGateBackdrop />
      </LoginGateHeroSection>
      <LoginGateAuthPanel>
        <LoginGateForm />
      </LoginGateAuthPanel>
    </LoginGateShell>
  );
}
