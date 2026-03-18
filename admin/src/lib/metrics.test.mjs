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
