import { redirect } from "next/navigation";
import { OnboardingWizard } from "@/components/onboarding-wizard";
import { PageHero } from "@/components/page-hero";
import { getServerAuthSession, getServerOnboarding } from "@/lib/server-api";

export const dynamic = "force-dynamic";

export default async function OnboardingPage() {
  const session = await getServerAuthSession();
  const currentUser = session?.user ?? null;

  if (!currentUser || (currentUser.role !== "admin" && currentUser.role !== "platform_admin")) {
    redirect("/dashboard");
  }

  let data = null;
  let loadError = "";

  try {
    data = await getServerOnboarding();
  } catch {
    loadError = "Onboarding data is not available right now.";
  }

  return (
    <div className="flex flex-1 flex-col gap-8">
      <PageHero
        eyebrow="Administration"
        title="School onboarding"
        description="Multi-step setup for curriculum, first class, and a minimal bot preset. Teacher invites stay outside this first delivery slice."
        surface="plain"
        className="mx-auto w-full max-w-5xl"
      />
      <div className="mx-auto w-full max-w-5xl">
        <OnboardingWizard initialData={data} loadError={loadError} />
      </div>
    </div>
  );
}
