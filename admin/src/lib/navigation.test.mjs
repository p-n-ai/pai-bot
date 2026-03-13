import test from "node:test";
import assert from "node:assert/strict";
import { getCurrentSection, isRouteActive } from "./navigation.mjs";

test("isRouteActive matches exact and nested routes", () => {
  assert.equal(isRouteActive("/", "/"), true);
  assert.equal(isRouteActive("/dashboard", "/dashboard"), true);
  assert.equal(isRouteActive("/dashboard/weekly", "/dashboard"), true);
  assert.equal(isRouteActive("/students/123", "/dashboard"), false);
});

test("getCurrentSection returns student detail metadata for nested student routes", () => {
  assert.deepEqual(getCurrentSection("/students/abc"), {
    eyebrow: "Student detail",
    title: "Learner profile",
    description: "Review mastery, streaks, and recent tutoring conversations.",
  });
});
