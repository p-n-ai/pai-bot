import { AIUsageBudgetSection } from "@/components/ai-usage/budget-section";
import { AIUsageDailyTrendSection } from "@/components/ai-usage/daily-trend-section";
import { AIUsageOverviewSection } from "@/components/ai-usage/overview-section";
import { AIUsageProviderBreakdownSection } from "@/components/ai-usage/provider-breakdown-section";
import type { AIUsageView } from "@/components/ai-usage/types";
import { StatePanel } from "@/components/state-panel";
import { ServerAPIError, getServerAIUsage, getServerAuthSession } from "@/lib/server-api";
import { getAIUsageBudgetViewModel } from "@/lib/ai-usage.mjs";

export const dynamic = "force-dynamic";

type AIUsagePageResult =
  | { ok: true; view: AIUsageView }
  | { ok: false; description: string };

async function loadAIUsagePageResult(): Promise<AIUsagePageResult> {
  try {
    const usage = await getServerAIUsage();
    return { ok: true, view: getAIUsageBudgetViewModel(usage) as AIUsageView };
  } catch (error) {
    const description =
      error instanceof ServerAPIError
        ? `The admin API returned ${error.status} while loading AI usage.`
        : error instanceof Error
          ? error.message
          : "The admin AI usage page could not be loaded.";

    return { ok: false, description };
  }
}

export default async function AIUsagePage() {
  const [session, result] = await Promise.all([getServerAuthSession(), loadAIUsagePageResult()]);
  const currentUser = session?.user ?? null;
  const canManageBudget = currentUser?.role === "admin";

  if (!result.ok) {
    return (
      <StatePanel
        tone="error"
        title="Unable to load AI usage"
        description={result.description}
      />
    );
  }

  return (
    <div className="space-y-6">
      <AIUsageOverviewSection view={result.view} />

      <div className="grid gap-6 xl:grid-cols-[1.25fr_0.95fr]">
        <AIUsageBudgetSection view={result.view} canManageBudget={canManageBudget} />
        <AIUsageDailyTrendSection view={result.view} />
      </div>

      <AIUsageProviderBreakdownSection view={result.view} />
    </div>
  );
}
