import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";
import { ACCESS_TOKEN_COOKIE, parseCookieJSON, USER_COOKIE } from "@/lib/auth-session";
import { getProxyRedirect } from "@/lib/proxy-routing.mjs";

export function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const hasSession = Boolean(request.cookies.get(ACCESS_TOKEN_COOKIE)?.value);
  const user = parseCookieJSON(request.cookies.get(USER_COOKIE)?.value);

  const redirect = getProxyRedirect(pathname, hasSession, user);
  if (redirect) {
    const redirectURL = new URL(redirect.pathname, request.url);
    if (redirect.addNext) {
      redirectURL.searchParams.set("next", pathname);
    }
    return NextResponse.redirect(redirectURL);
  }

  const response = NextResponse.next();
  if (pathname.startsWith("/dashboard") || pathname.startsWith("/students") || pathname.startsWith("/parents")) {
    response.headers.set("Cache-Control", "private, no-store, max-age=0");
  }
  return response;
}

export const config = {
  matcher: ["/dashboard/:path*", "/students/:path*", "/parents/:path*", "/login"],
};
