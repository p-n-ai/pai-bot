import test from "node:test";
import assert from "node:assert/strict";

import { normalizeMetrics } from "./metrics.mjs";

test("normalizeMetrics fills missing arrays and nested summaries with safe defaults", () => {
  const metrics = normalizeMetrics({
    window_days: 14,
    nudge_rate: { nudges_sent: 3 },
  });

  assert.equal(metrics.window_days, 14);
  assert.deepEqual(metrics.daily_active_users, []);
  assert.deepEqual(metrics.retention, []);
  assert.equal(metrics.nudge_rate.nudges_sent, 3);
  assert.equal(metrics.nudge_rate.responses_within_24h, 0);
  assert.equal(metrics.nudge_rate.response_rate, 0);
  assert.deepEqual(metrics.ai_usage.providers, []);
});

test("normalizeMetrics keeps experiment comparison data when present", () => {
  const metrics = normalizeMetrics({
    ab_comparison: {
      experiment_key: "motivation_features",
      window_days: 10,
      metric_name: "retention",
      variant_a: { label: "Control", users: 40, retention_rate: 0.32, nudge_response_rate: 0.18 },
      variant_b: { label: "Motivation", users: 42, retention_rate: 0.41, nudge_response_rate: 0.24 },
      winner: "variant_b",
      delta_retention_rate: 0.09,
    },
  });

  assert.equal(metrics.ab_comparison.experiment_key, "motivation_features");
  assert.equal(metrics.ab_comparison.window_days, 10);
  assert.equal(metrics.ab_comparison.variant_a.label, "Control");
  assert.equal(metrics.ab_comparison.variant_b.users, 42);
  assert.equal(metrics.ab_comparison.delta_retention_rate, 0.09);
  assert.equal(metrics.ab_comparison.delta_challenge_participation_rate, null);
});
