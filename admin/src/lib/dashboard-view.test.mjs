import test from "node:test";
import assert from "node:assert/strict";

import { getDashboardSummary } from "./dashboard-view.mjs";

test("getDashboardSummary returns dashboard counts and heatmap readiness", () => {
  const summary = getDashboardSummary({
    students: [
      { id: "student-1", topics: { algebra: 0.8, equations: 0.6 } },
      { id: "student-2", topics: { algebra: 0.4 } },
    ],
    topic_ids: ["algebra", "equations"],
  });

  assert.deepEqual(summary, {
    studentCount: 2,
    topicCount: 2,
    trackedScores: 3,
    averageMastery: 60,
    hasHeatmap: true,
    coveragePercent: 75,
    attentionCount: 1,
    weakestTopic: {
      topicId: "algebra",
      score: 60,
    },
    strongestTopic: {
      topicId: "equations",
      score: 60,
    },
  });
});

test("getDashboardSummary handles an empty class snapshot", () => {
  assert.deepEqual(getDashboardSummary({ students: [], topic_ids: [] }), {
    studentCount: 0,
    topicCount: 0,
    trackedScores: 0,
    averageMastery: 0,
    hasHeatmap: false,
    coveragePercent: 0,
    attentionCount: 0,
    weakestTopic: null,
    strongestTopic: null,
  });
});
