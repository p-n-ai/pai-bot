import assert from "node:assert/strict";
import test from "node:test";
import { formatAdminDateTime } from "./dates.mjs";

test("formatAdminDateTime returns stable UTC output", () => {
  assert.equal(formatAdminDateTime("2026-03-17T10:15:00Z"), "17 Mar 2026, 10:15 UTC");
});

test("formatAdminDateTime returns empty string for invalid input", () => {
  assert.equal(formatAdminDateTime("not-a-date"), "");
  assert.equal(formatAdminDateTime(""), "");
});
