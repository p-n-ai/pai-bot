"use client";

import { useEffect } from "react";
import { useAppStore } from "@/stores/app-store";

export function useLoginFlowBootstrap(seed: string, error: string) {
  const initializeLoginFlow = useAppStore((state) => state.initializeLoginFlow);

  useEffect(() => {
    initializeLoginFlow(seed, error);
  }, [error, initializeLoginFlow, seed]);
}

export function useInviteActivationFlowBootstrap(seed: string) {
  const initializeInviteActivationFlow = useAppStore((state) => state.initializeInviteActivationFlow);

  useEffect(() => {
    initializeInviteActivationFlow(seed);
  }, [initializeInviteActivationFlow, seed]);
}
