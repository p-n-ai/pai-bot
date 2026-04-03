"use client";

import { create } from "zustand";
import type { AuthSession, AuthUser } from "@/lib/api";
import { readSchoolSwitchState, type SchoolSwitchState } from "@/lib/school-switch-state";

type AdminSessionSnapshot = {
  hydrated: boolean;
  isLoggedIn: boolean;
  currentUser: AuthUser | null;
  schoolSwitchState: SchoolSwitchState | null;
  pendingTenantID: string | null;
  isSwitchingTenant: boolean;
};

type AdminSessionStore = AdminSessionSnapshot & {
  initializeFromServer: (currentUser: AuthUser | null, schoolSwitchState: SchoolSwitchState | null) => void;
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

function buildSnapshot(currentUser: AuthUser | null, schoolSwitchState: SchoolSwitchState | null): AdminSessionSnapshot {
  return {
    hydrated: true,
    currentUser,
    isLoggedIn: Boolean(currentUser?.user_id && currentUser?.email),
    schoolSwitchState,
    pendingTenantID: null,
    isSwitchingTenant: false,
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
    set(buildSnapshot(session.user, typeof window === "undefined" ? null : readSchoolSwitchState()));
  },
  clearSession: () => {
    set(buildSnapshot(null, typeof window === "undefined" ? null : readSchoolSwitchState()));
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

export function initializeAdminSessionStore(currentUser: AuthUser | null, schoolSwitchState: SchoolSwitchState | null) {
  useAppStore.getState().initializeFromServer(currentUser, schoolSwitchState);
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
