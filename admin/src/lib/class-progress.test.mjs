import assert from "node:assert/strict";

import { getAverageMastery, normalizeClassProgress } from "./class-progress.mjs";
import { getMockClassProgress } from "./mock-classes.mjs";

const normalized = normalizeClassProgress({
  students: [{ id: "student-1", name: "Alya", topics: { linear_equations: 0.8 } }],
  topic_ids: null,
});

assert.deepEqual(normalized.topic_ids, []);
assert.equal(normalized.students.length, 1);
assert.equal(getAverageMastery(normalized), 0);

const filled = normalizeClassProgress({
  students: [{ id: "student-1", name: "Alya", topics: { linear_equations: 0.8, algebraic_expressions: 0.6 } }],
  topic_ids: ["linear_equations", "algebraic_expressions"],
});

assert.equal(getAverageMastery(filled), 70);

const mockProgress = getMockClassProgress("all-students");

assert.ok(mockProgress.students.length > 0);
assert.ok(mockProgress.topic_ids.length > 0);
assert.equal(typeof mockProgress.students[0]?.topics[mockProgress.topic_ids[0]], "number");
