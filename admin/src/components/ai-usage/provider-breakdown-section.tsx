import { AdminInsetPanel } from "@/components/admin-inset-panel";
import { AdminSurface, AdminSurfaceHeader } from "@/components/admin-surface";
import { StatePanel } from "@/components/state-panel";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { formatCompactNumber, formatUSD } from "@/lib/ai-usage.mjs";
import type { AIUsageView } from "@/components/ai-usage/types";

function ProviderBreakdownTable({
  view,
}: {
  view: AIUsageView;
}) {
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>Provider</TableHead>
          <TableHead>Model</TableHead>
          <TableHead className="text-right">Messages</TableHead>
          <TableHead className="text-right">Tokens</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {view.providers.map((provider) => (
          <TableRow key={`${provider.provider}-${provider.model}`}>
            <TableCell className="font-medium">{provider.provider}</TableCell>
            <TableCell className="text-muted-foreground">
              {provider.model || "Default model"}
            </TableCell>
            <TableCell className="text-right">{formatCompactNumber(provider.messages)}</TableCell>
            <TableCell className="text-right">{formatCompactNumber(provider.total_tokens)}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

export function AIUsageProviderBreakdownSection({
  view,
}: {
  view: AIUsageView;
}) {
  return (
    <AdminSurface>
      <div className="space-y-5">
        <AdminSurfaceHeader
          title="Provider breakdown"
          description="Provider and model mix for the recorded AI traffic in this workspace."
        />

        {view.providers.length > 0 ? (
          <ProviderBreakdownTable view={view} />
        ) : (
          <StatePanel
            tone="empty"
            title="No provider traffic recorded"
            description="Provider rows will populate after the first successful AI requests for this tenant."
          />
        )}

        <div className="grid gap-4 md:grid-cols-3">
          <AdminInsetPanel title="Monthly cost">
            <p className="text-lg font-semibold text-slate-950 dark:text-slate-50">
              {formatUSD(view.monthlyCost)}
            </p>
          </AdminInsetPanel>
          <AdminInsetPanel title="Budget cap (USD)">
            <p className="text-lg font-semibold text-slate-950 dark:text-slate-50">
              {view.budgetLimit !== null ? formatUSD(view.budgetLimit) : "Not set"}
            </p>
          </AdminInsetPanel>
          <AdminInsetPanel title="Top provider">
            <p className="text-lg font-semibold text-slate-950 dark:text-slate-50">
              {view.topProvider?.provider ?? "None yet"}
            </p>
          </AdminInsetPanel>
        </div>
      </div>
    </AdminSurface>
  );
}
