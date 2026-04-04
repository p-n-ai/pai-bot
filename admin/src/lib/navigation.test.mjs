import test from "node:test";
import assert from "node:assert/strict";
import { getBreadcrumbs, getCurrentSection, getNavigationForUser, isRouteActive } from "./navigation.mjs";

test("isRouteActive matches exact and nested routes", () => {
  assert.equal(isRouteActive("/", "/"), true);
  assert.equal(isRouteActive("/dashboard", "/dashboard"), true);
  assert.equal(isRouteActive("/dashboard/weekly", "/dashboard"), true);
  assert.equal(isRouteActive("/students/123", "/dashboard"), false);
});

test("isRouteActive prefers the most specific dashboard route", () => {
  assert.equal(isRouteActive("/dashboard/ai-usage", "/dashboard"), false);
  assert.equal(isRouteActive("/dashboard/ai-usage", "/dashboard/ai-usage"), true);
  assert.equal(isRouteActive("/dashboard/metrics", "/dashboard"), true);
  assert.equal(isRouteActive("/dashboard/classes", "/dashboard"), false);
  assert.equal(isRouteActive("/dashboard/classes", "/dashboard/classes"), true);
});

test("getCurrentSection returns student detail metadata for nested student routes", () => {
  assert.deepEqual(getCurrentSection("/students/abc"), {
    eyebrow: "Learner detail",
    title: "Learner profile",
    description: "Review mastery, activity, and conversation history before the next intervention.",
  });
});

test("getCurrentSection returns parent detail metadata for nested parent routes", () => {
  assert.deepEqual(getCurrentSection("/parents/parent-1"), {
    eyebrow: "Parent support",
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

test("getCurrentSection returns retrieval lab metadata for both canonical and typo routes", () => {
  assert.deepEqual(getCurrentSection("/dashboard/retrieval-lab"), {
    eyebrow: "Retrieval",
    title: "BM25 query lab",
    description: "Try search queries, filters, and repeat runs against the shared retrieval service.",
  });
  assert.deepEqual(getCurrentSection("/dashboard/retreival-lab"), {
    eyebrow: "Retrieval",
    title: "BM25 query lab",
    description: "Try search queries, filters, and repeat runs against the shared retrieval service.",
  });
});

test("getCurrentSection falls back to dashboard metadata for the legacy metrics route", () => {
  assert.deepEqual(getCurrentSection("/dashboard/metrics"), {
    eyebrow: "Admin panel",
    title: "Dashboard",
    description: "Mastery heatmap, nudges, and learner drill-down.",
  });
});

test("getCurrentSection returns class management metadata for dashboard classes routes", () => {
  assert.deepEqual(getCurrentSection("/dashboard/classes"), {
    eyebrow: "Teaching operations",
    title: "Class management",
    description: "Review the planned class setup, join code, roster, and topic assignment layout.",
  });
});

test("getCurrentSection returns neutral overview copy when no pathname is provided", () => {
  assert.deepEqual(getCurrentSection(), {
    eyebrow: "Admin panel",
    title: "Dashboard",
    description: "Review mastery, nudges, and learner drill-down.",
  });
});

test("getNavigationForUser hides teacher links from parents", () => {
  assert.deepEqual(getNavigationForUser({ role: "parent", user_id: "parent-1" }), [
    {
      title: "Child Summary",
      href: "/parents/parent-1",
      description: "Weekly momentum, mastery, and encouragement for home support.",
      group: "Workspace",
    },
  ]);
});

test("getNavigationForUser keeps elevated navigation for teachers", () => {
  assert.deepEqual(
    getNavigationForUser({ role: "teacher", user_id: "teacher-1" }).map((item) => item.href),
    ["/dashboard", "/dashboard/classes", "/dashboard/ai-usage"],
  );
});

test("getBreadcrumbs returns learner detail hierarchy", () => {
  assert.deepEqual(getBreadcrumbs("/students/student-1"), [
    { label: "Dashboard", href: "/dashboard" },
    { label: "Learner profile", href: "/students/student-1" },
  ]);
});

test("getBreadcrumbs returns parent hierarchy for a parent user", () => {
  assert.deepEqual(getBreadcrumbs("/parents/parent-1", { role: "parent", user_id: "parent-1" }), [
    { label: "Child summary", href: "/parents/parent-1" },
  ]);
});

test("getBreadcrumbs returns class management hierarchy", () => {
  assert.deepEqual(getBreadcrumbs("/dashboard/classes"), [
    { label: "Dashboard", href: "/dashboard" },
    { label: "Classes", href: "/dashboard/classes" },
  ]);
});
