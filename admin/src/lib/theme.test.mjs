import assert from "node:assert/strict";

import { getPreferredTheme, isTheme, toggleTheme } from "./theme.mjs";

assert.equal(isTheme("light"), true);
assert.equal(isTheme("dark"), true);
assert.equal(isTheme("system"), true);

assert.equal(isTheme("sepia"), false);
assert.equal(isTheme(""), false);
assert.equal(isTheme(undefined), false);

assert.equal(getPreferredTheme("dark", "light"), "dark");
assert.equal(getPreferredTheme("light", "dark"), "light");

assert.equal(getPreferredTheme("system", "dark"), "dark");
assert.equal(getPreferredTheme("system", "light"), "light");
assert.equal(getPreferredTheme("invalid", "dark"), "dark");

assert.equal(toggleTheme("light"), "dark");
assert.equal(toggleTheme("dark"), "light");
