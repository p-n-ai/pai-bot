"use client";

import { create } from "zustand";
import type { AuthSession, AuthUser } from "@/lib/api";
import {
  buildSchoolSwitchState,
  type SchoolSwitchState,
  writeSchoolSwitchState,
} from "@/lib/school-switch-state";

type AdminSessionSnapshot = {
  hydrated: boolean;
  isLoggedIn: boolean;
  currentUser: AuthUser | null;
  schoolSwitchState: SchoolSwitchState | null;
  loginFlow: LoginFlowState;
  inviteActivationFlow: InviteActivationFlowState;
};

type LoginDraft = {
  email: string;
  password: string;
};

type LoginPhase =
  | { kind: "editing" }
  | { kind: "submitting" }
  | { kind: "redirecting_google" }
  | { kind: "error"; message: string };

type LoginFlowState = {
  seed: string | null;
  draft: LoginDraft;
  phase: LoginPhase;
};

type InviteActivationDraft = {
  name: string;
  password: string;
};

type InviteActivationPhase =
  | { kind: "editing" }
  | { kind: "submitting" }
  | { kind: "error"; message: string };

type InviteActivationFlowState = {
  seed: string | null;
  draft: InviteActivationDraft;
  phase: InviteActivationPhase;
};

type AdminSessionStore = AdminSessionSnapshot & {
  initializeFromServer: (currentUser: AuthUser | null, schoolSwitchState: SchoolSwitchState | null) => void;
  applySession: (session: AuthSession) => void;
  clearSession: () => void;
  initializeLoginFlow: (seed: string | null, error: string) => void;
  setLoginEmail: (email: string) => void;
  setLoginPassword: (password: string) => void;
  startLoginSubmit: () => void;
  startLoginGoogleRedirect: () => void;
  failLogin: (message: string) => void;
  initializeInviteActivationFlow: (seed: string | null) => void;
  setInviteActivationName: (name: string) => void;
  setInviteActivationPassword: (password: string) => void;
  startInviteActivationSubmit: () => void;
  failInviteActivation: (message: string) => void;
};

function getDefaultSnapshot(): AdminSessionSnapshot {
  return {
    hydrated: false,
    isLoggedIn: false,
    currentUser: null,
    schoolSwitchState: null,
    loginFlow: createLoginFlowState(null, ""),
    inviteActivationFlow: createInviteActivationFlowState(null),
  };
}

function createLoginFlowState(seed: string | null, error: string): LoginFlowState {
  return {
    seed,
    draft: {
      email: "",
      password: "",
    },
    phase: error ? { kind: "error", message: error } : { kind: "editing" },
  };
}

function createInviteActivationFlowState(seed: string | null): InviteActivationFlowState {
  return {
    seed,
    draft: {
      name: "",
      password: "",
    },
    phase: { kind: "editing" },
  };
}

function buildSnapshot(currentUser: AuthUser | null, schoolSwitchState: SchoolSwitchState | null): AdminSessionSnapshot {
  return {
    ...getDefaultSnapshot(),
    hydrated: true,
    currentUser,
    isLoggedIn: Boolean(currentUser?.user_id && currentUser?.email),
    schoolSwitchState,
  };
}

export const useAppStore = create<AdminSessionStore>((set) => ({
  ...getDefaultSnapshot(),
  initializeFromServer: (currentUser, schoolSwitchState) => {
    set(buildSnapshot(currentUser, schoolSwitchState));
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
      }
    }
    set(buildSnapshot(session.user, nextSchoolSwitchState));
  },
  clearSession: () => {
    set(buildSnapshot(null, null));
  },
  initializeLoginFlow: (seed, error) =>
    set((state) => {
      if (state.loginFlow.seed === seed) {
        const currentError = state.loginFlow.phase.kind === "error" ? state.loginFlow.phase.message : "";
        return currentError === error
          ? state
          : {
              loginFlow: {
                ...state.loginFlow,
                phase: error ? { kind: "error", message: error } : { kind: "editing" },
              },
            };
      }
      return {
        loginFlow: createLoginFlowState(seed, error),
      };
    }),
  setLoginEmail: (email) =>
    set((state) => ({
      loginFlow: {
        ...state.loginFlow,
        draft: {
          ...state.loginFlow.draft,
          email,
        },
        phase: { kind: "editing" },
      },
    })),
  setLoginPassword: (password) =>
    set((state) => ({
      loginFlow: {
        ...state.loginFlow,
        draft: {
          ...state.loginFlow.draft,
          password,
        },
        phase: { kind: "editing" },
      },
    })),
  startLoginSubmit: () =>
    set((state) => ({
      loginFlow: {
        ...state.loginFlow,
        phase: { kind: "submitting" },
      },
    })),
  startLoginGoogleRedirect: () =>
    set((state) => ({
      loginFlow: {
        ...state.loginFlow,
        phase: { kind: "redirecting_google" },
      },
    })),
  failLogin: (message) =>
    set((state) => ({
      loginFlow: {
        ...state.loginFlow,
        phase: { kind: "error", message },
      },
    })),
  initializeInviteActivationFlow: (seed) =>
    set((state) => {
      if (state.inviteActivationFlow.seed === seed) {
        return state;
      }
      return {
        inviteActivationFlow: createInviteActivationFlowState(seed),
      };
    }),
  setInviteActivationName: (name) =>
    set((state) => ({
      inviteActivationFlow: {
        ...state.inviteActivationFlow,
        draft: {
          ...state.inviteActivationFlow.draft,
          name,
        },
        phase: { kind: "editing" },
      },
    })),
  setInviteActivationPassword: (password) =>
    set((state) => ({
      inviteActivationFlow: {
        ...state.inviteActivationFlow,
        draft: {
          ...state.inviteActivationFlow.draft,
          password,
        },
        phase: { kind: "editing" },
      },
    })),
  startInviteActivationSubmit: () =>
    set((state) => ({
      inviteActivationFlow: {
        ...state.inviteActivationFlow,
        phase: { kind: "submitting" },
      },
    })),
  failInviteActivation: (message) =>
    set((state) => ({
      inviteActivationFlow: {
        ...state.inviteActivationFlow,
        phase: { kind: "error", message },
      },
    })),
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
