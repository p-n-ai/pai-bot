import { NextResponse } from "next/server";
import { cookies } from "next/headers";
import {
  ACCESS_TOKEN_COOKIE,
  REFRESH_TOKEN_COOKIE,
  USER_COOKIE,
  buildCookieRemoval,
  buildCookieValue,
} from "@/lib/auth-session";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export const dynamic = "force-dynamic";

export async function POST(request: Request) {
  const cookieStore = await cookies();
  const accessToken = cookieStore.get(ACCESS_TOKEN_COOKIE)?.value;
  const refreshToken = cookieStore.get(REFRESH_TOKEN_COOKIE)?.value;
  const body = await request.text();

  let response = await fetch(`${API_BASE}/api/admin/retrieval/search`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
    },
    body,
    cache: "no-store",
  });

  if (response.status === 401 && refreshToken) {
    const refreshResponse = await fetch(`${API_BASE}/api/auth/refresh`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        refresh_token: refreshToken,
      }),
      cache: "no-store",
    });

    if (refreshResponse.ok) {
      const session = (await refreshResponse.json()) as {
        access_token: string;
        refresh_token: string;
        user: unknown;
      };

      response = await fetch(`${API_BASE}/api/admin/retrieval/search`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${session.access_token}`,
        },
        body,
        cache: "no-store",
      });

      const proxied = new NextResponse(await response.text(), {
        status: response.status,
        headers: {
          "Content-Type": response.headers.get("Content-Type") || "application/json",
        },
      });
      proxied.headers.append("Set-Cookie", buildCookieValue(ACCESS_TOKEN_COOKIE, session.access_token, 60 * 60 * 24 * 7));
      proxied.headers.append("Set-Cookie", buildCookieValue(REFRESH_TOKEN_COOKIE, session.refresh_token, 60 * 60 * 24 * 7));
      proxied.headers.append("Set-Cookie", buildCookieValue(USER_COOKIE, JSON.stringify(session.user), 60 * 60 * 24 * 7));
      return proxied;
    }

    const expired = new NextResponse(await refreshResponse.text(), {
      status: refreshResponse.status,
      headers: {
        "Content-Type": refreshResponse.headers.get("Content-Type") || "text/plain; charset=utf-8",
      },
    });
    expired.headers.append("Set-Cookie", buildCookieRemoval(ACCESS_TOKEN_COOKIE));
    expired.headers.append("Set-Cookie", buildCookieRemoval(REFRESH_TOKEN_COOKIE));
    expired.headers.append("Set-Cookie", buildCookieRemoval(USER_COOKIE));
    return expired;
  }

  return new NextResponse(await response.text(), {
    status: response.status,
    headers: {
      "Content-Type": response.headers.get("Content-Type") || "application/json",
    },
  });
}
