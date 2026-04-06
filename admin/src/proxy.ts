import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";
import { SESSION_COOKIE } from "@/lib/auth-session";

export function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const hasSession = Boolean(request.cookies.get(SESSION_COOKIE)?.value);
  const isProtectedPath =
    pathname === "/dashboard" ||
    pathname.startsWith("/dashboard/") ||
    pathname === "/settings" ||
    pathname.startsWith("/settings/") ||
    pathname === "/export" ||
    pathname.startsWith("/export/") ||
    pathname === "/students" ||
    pathname.startsWith("/students/") ||
    pathname === "/parents" ||
    pathname.startsWith("/parents/");

  if (isProtectedPath && !hasSession) {
    const redirectURL = new URL("/login", request.url);
    redirectURL.searchParams.set("next", pathname);
    return NextResponse.redirect(redirectURL);
  }

  const response = NextResponse.next();
  if (
    pathname.startsWith("/dashboard") ||
    pathname.startsWith("/settings") ||
    pathname.startsWith("/export") ||
    pathname.startsWith("/students") ||
    pathname.startsWith("/parents")
  ) {
    response.headers.set("Cache-Control", "private, no-store, max-age=0");
  }
  return response;
}

export const config = {
  matcher: ["/dashboard/:path*", "/settings/:path*", "/export/:path*", "/students/:path*", "/parents/:path*", "/login"],
};
