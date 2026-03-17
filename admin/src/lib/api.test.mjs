import test from "node:test";
import assert from "node:assert/strict";

import { getDefaultRouteForUser } from "./default-route.mjs";

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
