import test from "node:test";
import assert from "node:assert/strict";
import { isPublicEntryRoute } from "./rbac.mjs";

test("isPublicEntryRoute marks only the unauthenticated gate routes as public", () => {
  assert.equal(isPublicEntryRoute("/"), true);
  assert.equal(isPublicEntryRoute("/login"), true);
  assert.equal(isPublicEntryRoute("/dashboard"), false);
  assert.equal(isPublicEntryRoute("/students/student-1"), false);
});
