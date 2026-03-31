"use client";

import { use } from "react";
import { LoginGateContext } from "@/components/login-gate/login-gate-context";

export function useLoginGate() {
  const context = use(LoginGateContext);

  if (!context) {
    throw new Error("useLoginGate must be used within LoginGateProvider");
  }

  return context;
}
