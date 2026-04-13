import { redirect } from "next/navigation";
import { ExportPanel } from "@/components/export-panel";
import { PageHero } from "@/components/page-hero";
import { getServerAuthSession } from "@/lib/server-api";

export const dynamic = "force-dynamic";

export default async function ExportPage() {
  const session = await getServerAuthSession();
  const currentUser = session?.user ?? null;

  if (!currentUser || (currentUser.role !== "admin" && currentUser.role !== "platform_admin")) {
    redirect("/dashboard");
  }

  return (
    <div className="space-y-6">
      <PageHero
        eyebrow="Administration"
        title="Data export"
        description="Download tenant-scoped student, conversation, and progress datasets for reporting, migration, and audit workflows."
        surface="plain"
      />
      <ExportPanel />
    </div>
  );
}
