import test from "node:test";
import assert from "node:assert/strict";

import { getClientSessionSnapshot, hasClientSession, hasSessionCookies, syncSessionCookies } from "./session-state.mjs";

test("hasClientSession returns true when token and user are present", () => {
  assert.equal(
    hasClientSession({
      accessToken: "token-123",
      user: { user_id: "u1", name: "Teacher", email: "teacher@example.com" },
    }),
    true,
  );
});

test("hasClientSession returns false when access token is missing", () => {
  assert.equal(
    hasClientSession({
      accessToken: "",
      user: { user_id: "u1", name: "Teacher", email: "teacher@example.com" },
    }),
    false,
  );
});

test("hasClientSession returns false when user is missing", () => {
  assert.equal(
    hasClientSession({
      accessToken: "token-123",
      user: null,
    }),
    false,
  );
});

test("getClientSessionSnapshot returns the active user for a valid session", () => {
  assert.deepEqual(
    getClientSessionSnapshot({
      accessToken: "token-123",
      user: { user_id: "u1", role: "teacher", name: "Teacher", email: "teacher@example.com" },
    }),
    {
      isLoggedIn: true,
      currentUser: { user_id: "u1", role: "teacher", name: "Teacher", email: "teacher@example.com" },
    },
  );
});

test("getClientSessionSnapshot replaces the old role when a different user signs in", () => {
  const previous = getClientSessionSnapshot({
    accessToken: "teacher-token",
    user: { user_id: "teacher-1", role: "teacher", name: "Teacher", email: "teacher@example.com" },
  });

  const next = getClientSessionSnapshot({
    accessToken: "parent-token",
    user: { user_id: "parent-1", role: "parent", name: "Parent", email: "parent@example.com" },
  });

  assert.equal(previous.currentUser.role, "teacher");
  assert.equal(next.currentUser.role, "parent");
});

test("hasSessionCookies returns true when both auth cookies are present", () => {
  assert.equal(
    hasSessionCookies("theme=dark; pai_admin_access=token-123; pai_admin_user=%7B%22user_id%22%3A%22u1%22%7D"),
    true,
  );
});

test("syncSessionCookies writes missing auth cookies from a valid client session", () => {
  const writes = [];

  assert.equal(
    syncSessionCookies({
      accessToken: "token-123",
      user: { user_id: "u1", name: "Teacher", email: "teacher@example.com" },
      cookieString: "theme=dark",
      writeCookie(value) {
        writes.push(value);
      },
    }),
    true,
  );
  assert.equal(writes.length, 2);
  assert.match(writes[0], /^pai_admin_access=token-123;/);
  assert.match(writes[1], /^pai_admin_user=%7B/);
});

test("syncSessionCookies skips writes when auth cookies already exist", () => {
  const writes = [];

  assert.equal(
    syncSessionCookies({
      accessToken: "token-123",
      user: { user_id: "u1", name: "Teacher", email: "teacher@example.com" },
      cookieString: "pai_admin_access=token-123; pai_admin_user=%7B%22user_id%22%3A%22u1%22%7D",
      writeCookie(value) {
        writes.push(value);
      },
    }),
    false,
  );
  assert.equal(writes.length, 0);
});
