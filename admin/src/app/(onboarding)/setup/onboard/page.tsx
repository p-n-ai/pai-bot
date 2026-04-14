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
    <div className="flex flex-1 flex-col gap-4">
      <header className="mx-auto w-full max-w-5xl">
        <div className="max-w-2xl space-y-2">
          <h1 className="text-4xl font-semibold tracking-tight text-foreground">Set up your first class</h1>
          <p className="text-sm leading-6 text-muted-foreground">Choose the syllabus. Name the class. Share it.</p>
        </div>
      </header>
      <div className="mx-auto w-full max-w-5xl">
        <OnboardingWizard initialData={data} loadError={loadError} />
      </div>
    </div>
  );
}
