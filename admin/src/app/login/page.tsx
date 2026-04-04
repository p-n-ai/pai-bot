import type { Metadata } from "next";
import { redirect } from "next/navigation";
import { LoginGate } from "@/components/login-gate";
import { getServerAuthSession } from "@/lib/server-api";
import { getSafeNextPath, hasAdminUIAccess } from "@/lib/rbac.mjs";

export const dynamic = "force-dynamic";
export const metadata: Metadata = {
  title: "Sign In | P&AI Admin",
  description: "Sign in to manage your P&AI Bot conversations and settings.",
};

type LoginPageProps = {
  searchParams: Promise<{ next?: string; auth_error?: string }>;
};

export default async function LoginPage({ searchParams }: LoginPageProps) {
  const { next, auth_error: authError } = await searchParams;
  const session = await getServerAuthSession();
  const currentUser = session?.user ?? null;

  if (currentUser && hasAdminUIAccess(currentUser)) {
    redirect(getSafeNextPath(currentUser, next));
  }

  return <LoginGate nextPath={next ?? null} authError={authError ?? null} />;
}
