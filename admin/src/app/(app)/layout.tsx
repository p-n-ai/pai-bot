import { cookies } from "next/headers";
import { APP_SIDEBAR_COOKIE_NAME } from "@/components/app-sidebar";
import { AppShell } from "@/components/app-shell";
import { getServerAuthSession } from "@/lib/server-api";
import { buildSchoolSwitchState } from "@/lib/school-switch-state";

export default async function ShellLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  const cookieStore = await cookies();
  const session = await getServerAuthSession();
  const defaultSidebarOpen = cookieStore.get(APP_SIDEBAR_COOKIE_NAME)?.value !== "false";
  const initialCurrentUser = session?.user ?? null;
  const initialSchoolSwitchState =
    session ? buildSchoolSwitchState(session.user.email, session.user.tenant_id, session.tenant_choices ?? []) : null;

  return (
    <AppShell
      defaultSidebarOpen={defaultSidebarOpen}
      initialCurrentUser={initialCurrentUser}
      initialSchoolSwitchState={initialSchoolSwitchState}
    >
      {children}
    </AppShell>
  );
}
