import Link from "next/link";
import { Coins, Cpu, MessagesSquare, Orbit } from "lucide-react";
import { AdminSurface, AdminSurfaceHeader } from "@/components/admin-surface";
import { PageHero } from "@/components/page-hero";
import { StatCard } from "@/components/stat-card";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import type { AIUsageSummary } from "@/lib/api";
import { formatCompactNumber, getTopProvider } from "@/lib/ai-usage.mjs";
import { getServerAIUsage } from "@/lib/server-api";

function providerTone(provider: string) {
  switch (provider) {
    case "openai":
      return "bg-sky-500";
    case "anthropic":
      return "bg-amber-400";
    case "google":
      return "bg-emerald-500";
    case "openrouter":
      return "bg-fuchsia-500";
    default:
      return "bg-slate-500";
  }
}

export const dynamic = "force-dynamic";

export default async function AIUsagePage() {
  let usage: AIUsageSummary | null = null;
  let loadError = "";

  try {
    usage = await getServerAIUsage();
  } catch {
    loadError = "AI usage isn't available right now.";
  }

  const topProvider = getTopProvider(usage);
  const totalTokens = (usage?.total_input_tokens ?? 0) + (usage?.total_output_tokens ?? 0);

  return (
    <div className="space-y-6">
      <PageHero
        eyebrow="AI operations"
        title="Provider usage at a glance"
        description="Track message volume, token load, and the models currently carrying the teacher workspace."
        aside={
          <div className="grid gap-3 rounded-[24px] bg-slate-950 p-4 text-white dark:bg-slate-900/90">
          <div>
            <p className="text-xs uppercase tracking-[0.22em] text-slate-400">Top provider</p>
            <p className="mt-2 text-3xl font-semibold">
              {topProvider ? `${topProvider.provider} / ${topProvider.model}` : "Usage snapshot pending"}
            </p>
          </div>
          <div className="text-sm text-slate-300">
            {topProvider ? `${formatCompactNumber(topProvider.total_tokens)} tokens handled in this snapshot.` : "Usage details will appear once model activity has been recorded."}
          </div>
        </div>
        }
      />

      <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
        <StatCard icon={MessagesSquare} title="AI messages" value={formatCompactNumber(usage?.total_messages ?? 0)} note="Messages with a recorded model" />
        <StatCard icon={Coins} title="Total tokens" value={formatCompactNumber(totalTokens)} note="Prompt plus completion tokens" />
        <StatCard icon={Cpu} title="Input tokens" value={formatCompactNumber(usage?.total_input_tokens ?? 0)} note="Prompt-side token volume" />
        <StatCard icon={Orbit} title="Providers" value={String(usage?.providers.length ?? 0)} note="Distinct provider/model rows returned" />
      </section>

      <AdminSurface>
        <AdminSurfaceHeader
          title="Provider breakdown"
          description="Use this table to spot which providers and models are carrying most of the token load."
          action={
            <Link href="/dashboard" className="text-sm font-medium text-sky-700 hover:text-sky-900 dark:text-sky-300 dark:hover:text-sky-200">
              Back to dashboard
            </Link>
          }
        />
        <div className="mt-6 space-y-4">
          {loadError ? <p className="text-sm text-slate-500 dark:text-slate-400">{loadError}</p> : null}
          {usage && usage.providers.length > 0 ? (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="text-slate-600 dark:text-slate-300">Provider</TableHead>
                <TableHead className="text-slate-600 dark:text-slate-300">Model</TableHead>
                <TableHead className="text-slate-600 dark:text-slate-300">Messages</TableHead>
                <TableHead className="text-slate-600 dark:text-slate-300">Input</TableHead>
                <TableHead className="text-slate-600 dark:text-slate-300">Output</TableHead>
                <TableHead className="text-slate-600 dark:text-slate-300">Total</TableHead>
                <TableHead className="text-slate-600 dark:text-slate-300">Load</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(usage?.providers ?? []).map((item) => {
                const share = totalTokens > 0 ? Math.round((item.total_tokens / totalTokens) * 100) : 0;
                return (
                  <TableRow key={`${item.provider}:${item.model}`}>
                    <TableCell className="font-medium capitalize text-slate-600 dark:text-slate-300">{item.provider}</TableCell>
                    <TableCell className="text-slate-600 dark:text-slate-300">{item.model}</TableCell>
                    <TableCell className="text-slate-600 dark:text-slate-300">{formatCompactNumber(item.messages)}</TableCell>
                    <TableCell className="text-slate-600 dark:text-slate-300">{formatCompactNumber(item.input_tokens)}</TableCell>
                    <TableCell className="text-slate-600 dark:text-slate-300">{formatCompactNumber(item.output_tokens)}</TableCell>
                    <TableCell className="text-slate-600 dark:text-slate-300">{formatCompactNumber(item.total_tokens)}</TableCell>
                    <TableCell>
                      <div className="min-w-[150px] space-y-2">
                        <div className="h-2 overflow-hidden rounded-full bg-slate-200 dark:bg-slate-800">
                          <div className={`h-full rounded-full ${providerTone(item.provider)}`} style={{ width: `${share}%` }} />
                        </div>
                        <span className="text-xs uppercase tracking-[0.18em] text-slate-500 dark:text-slate-400">{share}%</span>
                      </div>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
          ) : (
            <p className="text-sm text-slate-500 dark:text-slate-400">
              {loadError ? "Please check back after the next sync." : "No AI usage has been recorded yet."}
            </p>
          )}
        </div>
      </AdminSurface>
    </div>
  );
}
