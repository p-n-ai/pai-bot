import { redirect } from "next/navigation";

export const dynamic = "force-dynamic";

export default async function MetricsPage() {
  redirect("/dashboard/ai-usage");
}
