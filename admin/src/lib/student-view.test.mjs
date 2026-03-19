import test from "node:test";
import assert from "node:assert/strict";

import { buildStudentViewModel } from "./student-view.mjs";

test("buildStudentViewModel derives student detail sections from progress and conversations", () => {
  const model = buildStudentViewModel(
    {
      progress: [
        { topic_id: "linear-equations", mastery_score: 0.8 },
        { topic_id: "fractions", mastery_score: 0.45 },
      ],
    },
    [{ timestamp: "2026-03-19T10:00:00Z", role: "student", text: "hi" }],
  );

  assert.deepEqual(model.radarData, [
    { topic: "linear-equations", mastery: 80 },
    { topic: "fractions", mastery: 45 },
  ]);
  assert.equal(model.struggleAreas.length, 1);
  assert.equal(model.struggleAreas[0].topic_id, "fractions");
  assert.equal(model.hasProgress, true);
  assert.equal(model.hasConversations, true);
  assert.equal(model.activityGrid.at(-1)?.count, 1);
});

test("buildStudentViewModel handles an empty student record", () => {
  const model = buildStudentViewModel(null, []);

  assert.deepEqual(model.radarData, []);
  assert.deepEqual(model.struggleAreas, []);
  assert.equal(model.hasProgress, false);
  assert.equal(model.hasConversations, false);
  assert.equal(model.activityGrid.length, 14);
});
