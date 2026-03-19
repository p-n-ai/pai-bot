import test from "node:test";
import assert from "node:assert/strict";

import { getProxyRedirect, isProtectedPath } from "./proxy-routing.mjs";

test("isProtectedPath covers admin app sections behind auth", () => {
  assert.equal(isProtectedPath("/dashboard"), true);
  assert.equal(isProtectedPath("/students/student-1"), true);
  assert.equal(isProtectedPath("/parents/parent-1"), true);
  assert.equal(isProtectedPath("/"), false);
});

test("getProxyRedirect sends anonymous users to login for protected routes", () => {
  assert.deepEqual(getProxyRedirect("/dashboard", false, null), {
    pathname: "/login",
    addNext: true,
  });
});

test("getProxyRedirect restricts parents to their own route", () => {
  assert.deepEqual(
    getProxyRedirect("/dashboard", true, { role: "parent", user_id: "parent-1" }),
    {
      pathname: "/parents/parent-1",
      addNext: false,
    },
  );
});

test("getProxyRedirect returns logged-in users away from /login", () => {
  assert.deepEqual(
    getProxyRedirect("/login", true, { role: "teacher", user_id: "teacher-1" }),
    {
      pathname: "/dashboard",
      addNext: false,
    },
  );
});
