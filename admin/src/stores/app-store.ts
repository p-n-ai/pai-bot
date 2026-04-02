"use client";

import { create } from "zustand";
import type { AuthSession, AuthUser } from "@/lib/api";
import { readSchoolSwitchState, type SchoolSwitchState } from "@/lib/school-switch-state";
import { getClientSessionSnapshot, syncSessionCookies } from "@/lib/session-state.mjs";
import { hasStoredSession, readStoredAccessToken, readStoredUser } from "@/lib/client-session";

type AdminSessionSnapshot = {
  hydrated: boolean;
  isLoggedIn: boolean;
  currentUser: AuthUser | null;
  schoolSwitchState: SchoolSwitchState | null;
  pendingTenantID: string | null;
  isSwitchingTenant: boolean;
};

type AdminSessionStore = AdminSessionSnapshot & {
  syncFromStorage: () => void;
  applySession: (session: AuthSession) => void;
  clearSession: () => void;
  setSchoolSwitchState: (state: SchoolSwitchState | null) => void;
  startTenantSwitch: (tenantID: string, state: SchoolSwitchState | null) => void;
  finishTenantSwitch: () => void;
};

function getDefaultSnapshot(): AdminSessionSnapshot {
  return {
    hydrated: false,
    isLoggedIn: false,
    currentUser: null,
    schoolSwitchState: null,
    pendingTenantID: null,
    isSwitchingTenant: false,
  };
}

function readSnapshotFromStorage(): AdminSessionSnapshot {
  if (typeof window === "undefined") {
    return getDefaultSnapshot();
  }

  const snapshot = getClientSessionSnapshot({
    accessToken: readStoredAccessToken(),
    user: readStoredUser(),
  });

  syncSessionCookies({
    accessToken: readStoredAccessToken(),
    user: snapshot.currentUser,
    cookieString: document.cookie,
    writeCookie(value: string) {
      document.cookie = value;
    },
  });

  return {
    hydrated: true,
    currentUser: snapshot.currentUser,
    isLoggedIn: snapshot.isLoggedIn && hasStoredSession(),
    schoolSwitchState: readSchoolSwitchState(),
    pendingTenantID: null,
    isSwitchingTenant: false,
  };
}

export const useAppStore = create<AdminSessionStore>((set) => ({
  ...readSnapshotFromStorage(),
  syncFromStorage: () => {
    set(readSnapshotFromStorage());
  },
  applySession: (session) => {
    set({
      hydrated: true,
      isLoggedIn: true,
      currentUser: session.user,
      schoolSwitchState: readSchoolSwitchState(),
      pendingTenantID: null,
      isSwitchingTenant: false,
    });
  },
  clearSession: () => {
    set({
      hydrated: true,
      isLoggedIn: false,
      currentUser: null,
      schoolSwitchState: readSchoolSwitchState(),
      pendingTenantID: null,
      isSwitchingTenant: false,
    });
  },
  setSchoolSwitchState: (state) => {
    set({ schoolSwitchState: state });
  },
  startTenantSwitch: (tenantID, state) => {
    set({
      schoolSwitchState: state,
      pendingTenantID: tenantID,
      isSwitchingTenant: true,
    });
  },
  finishTenantSwitch: () => {
    set({
      pendingTenantID: null,
      isSwitchingTenant: false,
    });
  },
}));

export function syncAdminSessionStoreFromStorage() {
  useAppStore.getState().syncFromStorage();
}

export function applyAdminSessionToStore(session: AuthSession) {
  useAppStore.getState().applySession(session);
}

export function clearAdminSessionStore() {
  useAppStore.getState().clearSession();
}

export function setAdminSchoolSwitchState(state: SchoolSwitchState | null) {
  useAppStore.getState().setSchoolSwitchState(state);
}

export function startAdminTenantSwitch(tenantID: string, state: SchoolSwitchState | null) {
  useAppStore.getState().startTenantSwitch(tenantID, state);
}

export function finishAdminTenantSwitch() {
  useAppStore.getState().finishTenantSwitch();
}
