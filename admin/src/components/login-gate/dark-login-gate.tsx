"use client";

import { DarkLoginGateBackdrop } from "@/components/login-gate/dark-login-gate-backdrop";
import { LoginGateAuthPanel } from "@/components/login-gate/login-gate-auth-panel";
import { LoginGateForm } from "@/components/login-gate/login-gate-form";
import { LoginGateHeroSection } from "@/components/login-gate/login-gate-hero-section";
import { LoginGateShell } from "@/components/login-gate/login-gate-shell";

export function DarkLoginGate() {
  return (
    <LoginGateShell>
      <LoginGateHeroSection heroSectionClassName="bg-[#06101d]">
        <DarkLoginGateBackdrop />
      </LoginGateHeroSection>
      <LoginGateAuthPanel>
        <LoginGateForm />
      </LoginGateAuthPanel>
    </LoginGateShell>
  );
}
