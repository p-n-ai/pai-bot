import { redirect } from "next/navigation";
import { PageHero } from "@/components/page-hero";
import { UserManagementPanel } from "@/components/user-management-panel";
import { getServerAuthSession, getServerUserManagement } from "@/lib/server-api";

export const dynamic = "force-dynamic";

export default async function UserManagementPage() {
  const session = await getServerAuthSession();
  const currentUser = session?.user ?? null;

  if (!currentUser || (currentUser.role !== "admin" && currentUser.role !== "platform_admin")) {
    redirect("/dashboard");
  }

  let data = null;
  let loadError = "";

  try {
    data = await getServerUserManagement();
  } catch {
    loadError = "User management data isn't available right now.";
  }

  return (
    <div className="space-y-6">
      <PageHero
        eyebrow="Administration"
        title="User and invite management"
        description="Review active access, search the current workspace, and issue new teacher, parent, or admin invites through the shared activation flow."
        surface="plain"
      />
      <UserManagementPanel data={data} loadError={loadError} />
    </div>
  );
}
