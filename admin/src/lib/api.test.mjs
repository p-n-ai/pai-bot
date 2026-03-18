import test from "node:test";
import assert from "node:assert/strict";

import { getDefaultRouteForUser } from "./default-route.mjs";
import { canAccessPath, getSafeNextPath, hasAdminUIAccess } from "./rbac.mjs";

test("getDefaultRouteForUser sends parents to their child summary route", () => {
  assert.equal(
    getDefaultRouteForUser({ role: "parent", user_id: "parent-1" }),
    "/parents/parent-1",
  );
});

test("getDefaultRouteForUser sends teacher and admin roles to the dashboard", () => {
  assert.equal(
    getDefaultRouteForUser({ role: "teacher", user_id: "teacher-1" }),
    "/dashboard",
  );
  assert.equal(
    getDefaultRouteForUser({ role: "admin", user_id: "admin-1" }),
    "/dashboard",
  );
});

test("getDefaultRouteForUser rejects student access to the admin UI", () => {
  assert.equal(
    getDefaultRouteForUser({ role: "student", user_id: "student-1" }),
    "/login",
  );
  assert.equal(
    hasAdminUIAccess({ role: "student", user_id: "student-1" }),
    false,
  );
});

test("canAccessPath limits parent access to their own summary route", () => {
  const parentUser = { role: "parent", user_id: "parent-1" };
  assert.equal(canAccessPath(parentUser, "/parents/parent-1"), true);
  assert.equal(canAccessPath(parentUser, "/dashboard"), false);
  assert.equal(canAccessPath(parentUser, "/parents/parent-2"), false);
});

test("getSafeNextPath falls back when the requested route is not allowed", () => {
  assert.equal(
    getSafeNextPath({ role: "parent", user_id: "parent-1" }, "/dashboard"),
    "/parents/parent-1",
  );
  assert.equal(
    getSafeNextPath({ role: "teacher", user_id: "teacher-1" }, "/dashboard/ai-usage"),
    "/dashboard/ai-usage",
  );
});
