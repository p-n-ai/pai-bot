import { redirect } from "next/navigation";
import { OnboardingWizard } from "@/components/onboarding-wizard";
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
    <div className="flex flex-1 flex-col gap-3">
      <header className="mx-auto w-full max-w-5xl">
        <div className="max-w-2xl">
          <h1 className="text-4xl font-semibold tracking-tight text-foreground">Set up your first class</h1>
        </div>
      </header>
      <div className="mx-auto w-full max-w-5xl">
        <OnboardingWizard initialData={data} loadError={loadError} />
      </div>
    </div>
  );
}
