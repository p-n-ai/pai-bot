import type { Metadata } from "next";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import type { AuthUser } from "@/lib/api";
import { ACCESS_TOKEN_COOKIE, parseCookieJSON, USER_COOKIE } from "@/lib/auth-session";
import { getSafeNextPath, hasAdminUIAccess } from "@/lib/rbac.mjs";

export const dynamic = "force-dynamic";
export const metadata: Metadata = {
  title: "P&AI Admin",
  description: "Teacher and parent dashboard for P&AI Bot.",
};

type RootPageProps = {
  searchParams: Promise<{ next?: string }>;
};

export default async function RootPage({ searchParams }: RootPageProps) {
  const cookieStore = await cookies();
  const { next } = await searchParams;
  const accessToken = cookieStore.get(ACCESS_TOKEN_COOKIE)?.value;
  const currentUser = parseCookieJSON<AuthUser>(cookieStore.get(USER_COOKIE)?.value);

  if (accessToken && currentUser && hasAdminUIAccess(currentUser)) {
    redirect(getSafeNextPath(currentUser, next));
  }

  const loginURL = new URL("/login", "http://localhost");
  if (next) {
    loginURL.searchParams.set("next", next);
  }
  redirect(`${loginURL.pathname}${loginURL.search}`);
}
