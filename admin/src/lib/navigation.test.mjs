import test from "node:test";
import assert from "node:assert/strict";
import { getCurrentSection, getNavigationForUser, isRouteActive } from "./navigation.mjs";

test("isRouteActive matches exact and nested routes", () => {
  assert.equal(isRouteActive("/", "/"), true);
  assert.equal(isRouteActive("/dashboard", "/dashboard"), true);
  assert.equal(isRouteActive("/dashboard/weekly", "/dashboard"), true);
  assert.equal(isRouteActive("/students/123", "/dashboard"), false);
});

test("isRouteActive prefers the most specific dashboard route", () => {
  assert.equal(isRouteActive("/dashboard/ai-usage", "/dashboard"), false);
  assert.equal(isRouteActive("/dashboard/ai-usage", "/dashboard/ai-usage"), true);
  assert.equal(isRouteActive("/dashboard/metrics", "/dashboard"), false);
  assert.equal(isRouteActive("/dashboard/metrics", "/dashboard/metrics"), true);
});

test("getCurrentSection returns student detail metadata for nested student routes", () => {
  assert.deepEqual(getCurrentSection("/students/abc"), {
    eyebrow: "Student detail",
    title: "Learner profile",
    description: "Review mastery, streaks, and recent tutoring conversations.",
  });
});

test("getCurrentSection returns parent detail metadata for nested parent routes", () => {
  assert.deepEqual(getCurrentSection("/parents/parent-1"), {
    eyebrow: "Parent view",
    title: "Child summary",
    description: "Review weekly momentum, topic mastery, and a suggested encouragement for home support.",
  });
});

test("getCurrentSection returns AI usage metadata for dashboard analytics routes", () => {
  assert.deepEqual(getCurrentSection("/dashboard/ai-usage"), {
    eyebrow: "Admin panel",
    title: "AI Usage",
    description: "Review token volume by provider and model across the teacher workspace.",
  });
});

test("getCurrentSection returns metrics metadata for dashboard metrics routes", () => {
  assert.deepEqual(getCurrentSection("/dashboard/metrics"), {
    eyebrow: "Admin panel",
    title: "Metrics",
    description: "Review DAU, retention, nudge response, and token activity across the workspace.",
  });
});

test("getCurrentSection returns neutral overview copy when no pathname is provided", () => {
  assert.deepEqual(getCurrentSection(), {
    eyebrow: "Admin panel",
    title: "Overview",
    description: "Open the teacher workspace and monitor daily activity.",
  });
});

test("getNavigationForUser hides teacher links from parents", () => {
  assert.deepEqual(getNavigationForUser({ role: "parent", user_id: "parent-1" }), [
    {
      title: "Child Summary",
      href: "/parents/parent-1",
      description: "Weekly momentum, mastery, and encouragement for home support.",
    },
  ]);
});

test("getNavigationForUser keeps elevated navigation for teachers", () => {
  assert.deepEqual(
    getNavigationForUser({ role: "teacher", user_id: "teacher-1" }).map((item) => item.href),
    ["/", "/dashboard", "/dashboard/metrics", "/dashboard/ai-usage"],
  );
});
