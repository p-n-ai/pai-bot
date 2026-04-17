import { redirect } from "next/navigation";
import { PageHero } from "@/components/page-hero";
import { WhatsAppSetupPanel } from "@/components/whatsapp-setup-panel";
import { getServerAuthSession } from "@/lib/server-api";

export const dynamic = "force-dynamic";

export default async function WhatsAppSetupPage() {
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
        title="WhatsApp setup"
        description="Link your WhatsApp account to enable the bot on WhatsApp. Scan the QR code with your phone to connect."
        surface="plain"
      />
      <WhatsAppSetupPanel />
    </div>
  );
}
