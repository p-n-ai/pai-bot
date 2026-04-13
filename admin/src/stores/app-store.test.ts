import { beforeEach, describe, expect, it } from "vitest";
import { clearAdminSessionStore, useAppStore } from "@/stores/app-store";

describe("app-store auth flows", () => {
  beforeEach(() => {
    clearAdminSessionStore();
  });

  it("keeps login draft for the same seed while clearing stale errors", () => {
    const store = useAppStore.getState();

    store.initializeLoginFlow("login-seed", "");
    store.setLoginEmail("admin@example.com");
    store.setLoginPassword("demo-password");
    store.failLogin("Bad credentials");
    useAppStore.getState().initializeLoginFlow("login-seed", "");

    const nextState = useAppStore.getState().loginFlow;

    expect(nextState.seed).toBe("login-seed");
    expect(nextState.draft).toEqual({
      email: "admin@example.com",
      password: "demo-password",
    });
    expect(nextState.phase).toEqual({ kind: "editing" });
  });

  it("resets login draft when the bootstrap seed changes", () => {
    const store = useAppStore.getState();

    store.initializeLoginFlow("seed-a", "");
    store.setLoginEmail("admin@example.com");
    store.setLoginPassword("demo-password");
    useAppStore.getState().initializeLoginFlow("seed-b", "Google link required");

    const nextState = useAppStore.getState().loginFlow;

    expect(nextState.seed).toBe("seed-b");
    expect(nextState.draft).toEqual({
      email: "",
      password: "",
    });
    expect(nextState.phase).toEqual({ kind: "error", message: "Google link required" });
  });

  it("resets invite activation draft when the token seed changes", () => {
    const store = useAppStore.getState();

    store.initializeInviteActivationFlow("token-a");
    store.setInviteActivationName("Thoriq");
    store.setInviteActivationPassword("very-secret");
    useAppStore.getState().initializeInviteActivationFlow("token-b");

    const nextState = useAppStore.getState().inviteActivationFlow;

    expect(nextState.seed).toBe("token-b");
    expect(nextState.draft).toEqual({
      name: "",
      password: "",
    });
    expect(nextState.phase).toEqual({ kind: "editing" });
  });
});
