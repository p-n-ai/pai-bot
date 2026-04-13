import test from "node:test";
import assert from "node:assert/strict";
import { canAccessPath, isPublicEntryRoute } from "./rbac.mjs";

test("isPublicEntryRoute marks unauthenticated entry and invite routes as public", () => {
  assert.equal(isPublicEntryRoute("/"), true);
  assert.equal(isPublicEntryRoute("/login"), true);
  assert.equal(isPublicEntryRoute("/activate"), true);
  assert.equal(isPublicEntryRoute("/join/steady-otter-harbor"), true);
  assert.equal(isPublicEntryRoute("/dashboard"), false);
  assert.equal(isPublicEntryRoute("/students/student-1"), false);
});

test("canAccessPath allows admin settings and export routes but not parent-only users", () => {
  assert.equal(canAccessPath({ role: "admin", user_id: "admin-1" }, "/settings/users"), true);
  assert.equal(canAccessPath({ role: "admin", user_id: "admin-1" }, "/export"), true);
  assert.equal(canAccessPath({ role: "admin", user_id: "admin-1" }, "/setup/onboard"), true);
  assert.equal(canAccessPath({ role: "parent", user_id: "parent-1" }, "/settings/users"), false);
});
