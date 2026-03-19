import test from "node:test";
import assert from "node:assert/strict";

import { getClassManagementSummary, getMockClasses } from "./mock-classes.mjs";

test("getMockClasses returns independent copies of the mock class data", () => {
  const first = getMockClasses();
  const second = getMockClasses();

  first[0].name = "Changed";
  first[0].assignedTopics[0].title = "Changed topic";

  assert.notEqual(second[0].name, "Changed");
  assert.notEqual(second[0].assignedTopics[0].title, "Changed topic");
});

test("getClassManagementSummary derives overview metrics from class fixtures", () => {
  const summary = getClassManagementSummary(getMockClasses());

  assert.equal(summary.classCount, 3);
  assert.equal(summary.totalMembers, 55);
  assert.equal(summary.activeStudents, 40);
  assert.equal(summary.averageMastery, 59);
  assert.deepEqual(summary.joinCodes, ["ALG-F1A", "ALG-F2B", "ALG-F3X"]);
});
