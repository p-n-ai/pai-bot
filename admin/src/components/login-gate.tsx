"use client";

import { useTheme } from "@/components/theme-provider";
import { DarkLoginGate } from "@/components/login-gate/dark-login-gate";
import { LightLoginGate } from "@/components/login-gate/light-login-gate";
import { LoginGateProvider } from "@/components/login-gate/login-gate-provider";

export function LoginGate({ nextPath = null }: { nextPath?: string | null }) {
  const { theme, mounted } = useTheme();
  const isDark = mounted && theme === "dark";

  return (
    <LoginGateProvider nextPath={nextPath}>
      {isDark ? <DarkLoginGate /> : <LightLoginGate />}
    </LoginGateProvider>
  );
}
