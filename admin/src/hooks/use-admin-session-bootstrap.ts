"use client";

import { useEffect } from "react";
import type { AuthUser } from "@/lib/api";
import type { SchoolSwitchState } from "@/lib/school-switch-state";
import { initializeAdminSessionStore } from "@/stores/app-store";

export function useAdminSessionBootstrap(currentUser: AuthUser | null, schoolSwitchState: SchoolSwitchState | null) {
  useEffect(() => {
    initializeAdminSessionStore(currentUser, schoolSwitchState);
  }, [currentUser, schoolSwitchState]);
}
