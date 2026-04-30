import { redirect } from "next/navigation";
import { EmbedSettingsPanel } from "@/components/embed-settings-panel";
import { PageHero } from "@/components/page-hero";
import { getServerAuthSession } from "@/lib/server-api";

export const dynamic = "force-dynamic";

export default async function EmbedSettingsPage() {
  const session = await getServerAuthSession();
  const currentUser = session?.user ?? null;

  if (
    !currentUser ||
    (currentUser.role !== "admin" && currentUser.role !== "platform_admin")
  ) {
    redirect("/dashboard");
  }

  return (
    <div className="space-y-6">
      <PageHero
        eyebrow="Integration"
        title="Embed settings"
        description="Enable the website chat widget, trust the school website origin, and copy the script snippet."
        surface="plain"
      />
      <EmbedSettingsPanel currentUser={currentUser} />
    </div>
  );
}
