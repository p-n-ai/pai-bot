import test from "node:test";
import assert from "node:assert/strict";

import { hasClientSession } from "./session-state.mjs";

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
