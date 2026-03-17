import Link from "next/link";
import { Coins, Cpu, MessagesSquare, Orbit } from "lucide-react";
import { PageHero } from "@/components/page-hero";
import { StatCard } from "@/components/stat-card";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
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
    loadError = "Failed to load AI usage.";
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
              {topProvider ? `${topProvider.provider} / ${topProvider.model}` : "No data"}
            </p>
          </div>
          <div className="text-sm text-slate-300">
            {topProvider ? `${formatCompactNumber(topProvider.total_tokens)} tokens handled in this snapshot.` : "Waiting for AI usage data from the admin API."}
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

      <Card className="rounded-[28px] border-white/70 bg-white/85 shadow-[0_18px_60px_rgba(15,23,42,0.05)] dark:border-white/10 dark:bg-slate-950/60 dark:shadow-[0_24px_80px_rgba(2,8,23,0.35)]">
        <CardHeader className="flex flex-row items-center justify-between gap-3">
          <div>
            <CardTitle className="text-xl tracking-tight">Provider breakdown</CardTitle>
            <p className="mt-1 text-sm text-slate-500 dark:text-slate-400">
              Use this table to spot which providers and models are carrying most of the token load.
            </p>
          </div>
          <Link href="/dashboard" className="text-sm font-medium text-sky-700 hover:text-sky-900 dark:text-sky-300 dark:hover:text-sky-200">
            Back to dashboard
          </Link>
        </CardHeader>
        <CardContent className="space-y-4">
          {loadError ? <p className="text-sm text-rose-600">{loadError}</p> : null}
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Provider</TableHead>
                <TableHead>Model</TableHead>
                <TableHead>Messages</TableHead>
                <TableHead>Input</TableHead>
                <TableHead>Output</TableHead>
                <TableHead>Total</TableHead>
                <TableHead>Load</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {(usage?.providers ?? []).map((item) => {
                const share = totalTokens > 0 ? Math.round((item.total_tokens / totalTokens) * 100) : 0;
                return (
                  <TableRow key={`${item.provider}:${item.model}`}>
                    <TableCell className="font-medium capitalize">{item.provider}</TableCell>
                    <TableCell>{item.model}</TableCell>
                    <TableCell>{formatCompactNumber(item.messages)}</TableCell>
                    <TableCell>{formatCompactNumber(item.input_tokens)}</TableCell>
                    <TableCell>{formatCompactNumber(item.output_tokens)}</TableCell>
                    <TableCell>{formatCompactNumber(item.total_tokens)}</TableCell>
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
          {usage && usage.providers.length === 0 ? (
            <p className="text-sm text-slate-500 dark:text-slate-400">No AI usage has been recorded yet.</p>
          ) : null}
        </CardContent>
      </Card>
    </div>
  );
}
