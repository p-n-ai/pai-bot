import type { Metadata } from "next";
import { RootLandingPage } from "@/components/landing/root-landing-page";
import { getServerAuthSession, getServerPostAuthPath } from "@/lib/server-api";
import { hasAdminUIAccess } from "@/lib/rbac.mjs";

export const dynamic = "force-dynamic";
export const metadata: Metadata = {
  title: "P&AI Bot | WhatsApp and Telegram learning support",
  description: "Students chat with the bot in WhatsApp and Telegram while teachers see weak topics and follow-up needs.",
};

type RootPageProps = {
  searchParams: Promise<{ next?: string }>;
};

export default async function RootPage({ searchParams }: RootPageProps) {
  const { next } = await searchParams;
  const session = await getServerAuthSession();
  const currentUser = session?.user ?? null;
  const loginURL = new URL("/login", "http://localhost");

  if (next) {
    loginURL.searchParams.set("next", next);
  }

  const loginHref = `${loginURL.pathname}${loginURL.search}`;
  const hasWorkspaceAccess = Boolean(currentUser && hasAdminUIAccess(currentUser));
  const primaryHref = hasWorkspaceAccess
    ? await getServerPostAuthPath(currentUser!, next ?? null)
    : loginHref;

  return (
    <RootLandingPage
      primaryHref={primaryHref}
      primaryLabel={hasWorkspaceAccess ? "Open workspace" : "Sign in"}
      signedInLabel={hasWorkspaceAccess ? "Signed in" : null}
    />
  );
}
