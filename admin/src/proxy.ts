import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";
import { ACCESS_TOKEN_COOKIE, parseCookieJSON, USER_COOKIE } from "@/lib/auth-session";
import { canAccessPath, getDefaultRouteForUser, hasAdminUIAccess } from "@/lib/rbac.mjs";

const protectedPrefixes = ["/dashboard", "/students", "/parents"];

function isProtectedPath(pathname: string): boolean {
  return protectedPrefixes.some((prefix) => pathname === prefix || pathname.startsWith(`${prefix}/`));
}

export function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const hasSession = Boolean(request.cookies.get(ACCESS_TOKEN_COOKIE)?.value);
  const user = parseCookieJSON(request.cookies.get(USER_COOKIE)?.value);

  if (isProtectedPath(pathname) && !hasSession) {
    const loginURL = new URL("/login", request.url);
    loginURL.searchParams.set("next", pathname);
    return NextResponse.redirect(loginURL);
  }

  if (isProtectedPath(pathname) && (!hasAdminUIAccess(user) || !canAccessPath(user, pathname))) {
    const redirectURL = new URL(hasAdminUIAccess(user) ? getDefaultRouteForUser(user) : "/login", request.url);
    if (!hasAdminUIAccess(user)) {
      redirectURL.searchParams.set("next", pathname);
    }
    return NextResponse.redirect(redirectURL);
  }

  if (pathname === "/login" && hasSession && hasAdminUIAccess(user)) {
    return NextResponse.redirect(new URL(getDefaultRouteForUser(user), request.url));
  }

  return NextResponse.next();
}

export const config = {
  matcher: ["/dashboard/:path*", "/students/:path*", "/parents/:path*", "/login"],
};
