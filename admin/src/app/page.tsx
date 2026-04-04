import type { Metadata } from "next";
import { redirect } from "next/navigation";
import { getServerAuthSession } from "@/lib/server-api";
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
  const { next } = await searchParams;
  const session = await getServerAuthSession();
  const currentUser = session?.user ?? null;

  if (currentUser && hasAdminUIAccess(currentUser)) {
    redirect(getSafeNextPath(currentUser, next));
  }

  const loginURL = new URL("/login", "http://localhost");
  if (next) {
    loginURL.searchParams.set("next", next);
  }
  redirect(`${loginURL.pathname}${loginURL.search}`);
}
