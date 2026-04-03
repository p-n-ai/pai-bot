"use client";

import { create } from "zustand";
import type { AuthSession, AuthUser } from "@/lib/api";
import {
  buildSchoolSwitchState,
  clearSchoolSwitchState,
  readSchoolSwitchState,
  type SchoolSwitchState,
  writeSchoolSwitchState,
} from "@/lib/school-switch-state";

type AdminSessionSnapshot = {
  hydrated: boolean;
  isLoggedIn: boolean;
  currentUser: AuthUser | null;
  schoolSwitchState: SchoolSwitchState | null;
};

type AdminSessionStore = AdminSessionSnapshot & {
  initializeFromServer: (currentUser: AuthUser | null, schoolSwitchState: SchoolSwitchState | null) => void;
  applySession: (session: AuthSession) => void;
  clearSession: () => void;
};

function getDefaultSnapshot(): AdminSessionSnapshot {
  return {
    hydrated: false,
    isLoggedIn: false,
    currentUser: null,
    schoolSwitchState: null,
  };
}

function buildSnapshot(currentUser: AuthUser | null, schoolSwitchState: SchoolSwitchState | null): AdminSessionSnapshot {
  return {
    hydrated: true,
    currentUser,
    isLoggedIn: Boolean(currentUser?.user_id && currentUser?.email),
    schoolSwitchState,
  };
}

export const useAppStore = create<AdminSessionStore>((set) => ({
  ...getDefaultSnapshot(),
  initializeFromServer: (currentUser, schoolSwitchState) => {
    const nextSchoolSwitchState =
      schoolSwitchState ?? (typeof window === "undefined" ? null : readSchoolSwitchState());
    set(buildSnapshot(currentUser, nextSchoolSwitchState));
  },
  applySession: (session) => {
    const nextSchoolSwitchState = buildSchoolSwitchState(
      session.user.email,
      session.user.tenant_id,
      session.tenant_choices ?? [],
    );
    if (typeof window !== "undefined") {
      if (nextSchoolSwitchState) {
        writeSchoolSwitchState(nextSchoolSwitchState);
      } else {
        clearSchoolSwitchState();
      }
    }
    set(buildSnapshot(session.user, nextSchoolSwitchState));
  },
  clearSession: () => {
    set(buildSnapshot(null, typeof window === "undefined" ? null : readSchoolSwitchState()));
  },
}));

export function initializeAdminSessionStore(currentUser: AuthUser | null, schoolSwitchState: SchoolSwitchState | null) {
  useAppStore.getState().initializeFromServer(currentUser, schoolSwitchState);
}

export function applyAdminSessionToStore(session: AuthSession) {
  useAppStore.getState().applySession(session);
}

export function clearAdminSessionStore() {
  useAppStore.getState().clearSession();
}
