import test from "node:test";
import assert from "node:assert/strict";

import { buildStudentActivityGrid, getActivityLevel } from "./student-activity.mjs";

test("buildStudentActivityGrid buckets conversation timestamps into a fixed date window", () => {
  const grid = buildStudentActivityGrid(
    [
      { timestamp: "2026-03-14T03:00:00Z" },
      { timestamp: "2026-03-14T08:30:00Z" },
      { timestamp: "2026-03-16T12:00:00Z" },
    ],
    { anchorDate: "2026-03-16", windowDays: 3 },
  );

  assert.deepEqual(grid, [
    { date: "2026-03-14", shortLabel: "Mar 14", count: 2, level: 2 },
    { date: "2026-03-15", shortLabel: "Mar 15", count: 0, level: 0 },
    { date: "2026-03-16", shortLabel: "Mar 16", count: 1, level: 1 },
  ]);
});

test("buildStudentActivityGrid falls back to the current day when timestamps are missing", () => {
  const grid = buildStudentActivityGrid([], { anchorDate: "2026-03-19", windowDays: 2 });

  assert.deepEqual(grid.map((item) => item.date), ["2026-03-18", "2026-03-19"]);
  assert.deepEqual(grid.map((item) => item.count), [0, 0]);
});

test("getActivityLevel scales counts into four intensity bands", () => {
  assert.equal(getActivityLevel(0), 0);
  assert.equal(getActivityLevel(1), 1);
  assert.equal(getActivityLevel(2), 2);
  assert.equal(getActivityLevel(4), 3);
  assert.equal(getActivityLevel(6), 4);
});
