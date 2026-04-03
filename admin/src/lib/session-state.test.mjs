import test from "node:test";
import assert from "node:assert/strict";

import { getClientSessionSnapshot, hasClientSession, hasSessionCookies, syncSessionCookies } from "./session-state.mjs";

test("hasClientSession returns true when token and user are present", () => {
  assert.equal(
    hasClientSession({
      sessionToken: "token-123",
      user: { user_id: "u1", name: "Teacher", email: "teacher@example.com" },
    }),
    true,
  );
});

test("hasClientSession returns false when the session token is missing", () => {
  assert.equal(
    hasClientSession({
      sessionToken: "",
      user: { user_id: "u1", name: "Teacher", email: "teacher@example.com" },
    }),
    false,
  );
});

test("hasClientSession returns false when user is missing", () => {
  assert.equal(
    hasClientSession({
      sessionToken: "token-123",
      user: null,
    }),
    false,
  );
});

test("getClientSessionSnapshot returns the active user for a valid session", () => {
  assert.deepEqual(
    getClientSessionSnapshot({
      sessionToken: "token-123",
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
    sessionToken: "teacher-token",
    user: { user_id: "teacher-1", role: "teacher", name: "Teacher", email: "teacher@example.com" },
  });

  const next = getClientSessionSnapshot({
    sessionToken: "parent-token",
    user: { user_id: "parent-1", role: "parent", name: "Parent", email: "parent@example.com" },
  });

  assert.equal(previous.currentUser.role, "teacher");
  assert.equal(next.currentUser.role, "parent");
});

test("hasSessionCookies returns true when the session cookie is present", () => {
  assert.equal(
    hasSessionCookies("theme=dark; pai_session=token-123"),
    true,
  );
});

test("syncSessionCookies writes the missing session cookie from a valid client session", () => {
  const writes = [];

  assert.equal(
    syncSessionCookies({
      sessionToken: "token-123",
      user: { user_id: "u1", name: "Teacher", email: "teacher@example.com" },
      cookieString: "theme=dark",
      writeCookie(value) {
        writes.push(value);
      },
    }),
    true,
  );
  assert.equal(writes.length, 1);
  assert.match(writes[0], /^pai_session=token-123;/);
});

test("syncSessionCookies skips writes when auth cookies already exist", () => {
  const writes = [];

  assert.equal(
    syncSessionCookies({
      sessionToken: "token-123",
      user: { user_id: "u1", name: "Teacher", email: "teacher@example.com" },
      cookieString: "pai_session=token-123",
      writeCookie(value) {
        writes.push(value);
      },
    }),
    false,
  );
  assert.equal(writes.length, 0);
});
