import type { Metadata } from "next";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { LoginGate } from "@/components/login-gate";
import type { AuthUser } from "@/lib/api";
import {
  ACCESS_TOKEN_COOKIE,
  parseCookieJSON,
  USER_COOKIE,
} from "@/lib/auth-session";
import { getSafeNextPath, hasAdminUIAccess } from "@/lib/rbac.mjs";

export const dynamic = "force-dynamic";
export const metadata: Metadata = {
  title: "Sign In | P&AI Admin",
  description: "Sign in to manage your P&AI Bot conversations and settings.",
};

type LoginPageProps = {
  searchParams: Promise<{ next?: string }>;
};

export default async function LoginPage({ searchParams }: LoginPageProps) {
  const cookieStore = await cookies();
  const { next } = await searchParams;
  const accessToken = cookieStore.get(ACCESS_TOKEN_COOKIE)?.value;
  const currentUser = parseCookieJSON<AuthUser>(
    cookieStore.get(USER_COOKIE)?.value,
  );

  if (accessToken && currentUser && hasAdminUIAccess(currentUser)) {
    redirect(getSafeNextPath(currentUser, next));
  }

  return <LoginGate nextPath={next ?? null} />;
}
