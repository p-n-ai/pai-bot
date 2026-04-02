"use client";

import { useState, type FormEvent } from "react";
import { useRouter } from "next/navigation";
import { FormField } from "@/components/form-field";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { upsertTokenBudgetWindow } from "@/lib/api";

function defaultStartDate() {
  return new Date().toISOString().slice(0, 10);
}

function defaultEndDate() {
  const now = new Date();
  const end = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth() + 1, 0));
  return end.toISOString().slice(0, 10);
}

export function TokenBudgetEditor({
  initialBudgetTokens,
  initialPeriodStart,
  initialPeriodEnd,
}: {
  initialBudgetTokens: number | null;
  initialPeriodStart: string;
  initialPeriodEnd: string;
}) {
  const router = useRouter();
  const [budgetTokens, setBudgetTokens] = useState(
    initialBudgetTokens !== null && Number.isFinite(initialBudgetTokens) ? String(initialBudgetTokens) : "",
  );
  const [periodStart, setPeriodStart] = useState(initialPeriodStart || defaultStartDate());
  const [periodEnd, setPeriodEnd] = useState(initialPeriodEnd || defaultEndDate());
  const [error, setError] = useState("");
  const [isPending, setIsPending] = useState(false);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();

    const parsedBudget = Number.parseInt(budgetTokens, 10);
    if (!Number.isFinite(parsedBudget) || parsedBudget <= 0) {
      setError("Enter a token budget greater than zero.");
      return;
    }

    setError("");
    setIsPending(true);

    try {
      await upsertTokenBudgetWindow({
        budget_tokens: parsedBudget,
        period_start: periodStart,
        period_end: periodEnd,
      });
      router.refresh();
    } catch (submitError) {
      setError(submitError instanceof Error ? submitError.message : "Unable to save the token budget window.");
    } finally {
      setIsPending(false);
    }
  }

  return (
    <form
      className="rounded-3xl border border-slate-200/80 bg-slate-50/80 p-4 dark:border-white/10 dark:bg-white/5"
      onSubmit={handleSubmit}
    >
      <div className="space-y-1">
        <h3 className="text-sm font-semibold text-slate-950 dark:text-slate-50">Configure tenant token budget</h3>
        <p className="text-sm text-slate-600 dark:text-slate-300">
          Save a tenant-wide token allowance window. Reusing the same date range updates the existing limit.
        </p>
      </div>

      <div className="mt-4 grid gap-4 md:grid-cols-3">
        <FormField label="Token limit" htmlFor="token-budget-limit">
          <Input
            id="token-budget-limit"
            name="token-budget-limit"
            type="number"
            min={1}
            step={1}
            value={budgetTokens}
            onChange={(event) => setBudgetTokens(event.target.value)}
            placeholder="250000"
            required
          />
        </FormField>
        <FormField label="Start date" htmlFor="token-budget-start">
          <Input
            id="token-budget-start"
            name="token-budget-start"
            type="date"
            value={periodStart}
            onChange={(event) => setPeriodStart(event.target.value)}
            required
          />
        </FormField>
        <FormField label="End date" htmlFor="token-budget-end">
          <Input
            id="token-budget-end"
            name="token-budget-end"
            type="date"
            value={periodEnd}
            onChange={(event) => setPeriodEnd(event.target.value)}
            required
          />
        </FormField>
      </div>

      {error ? (
        <div className="mt-4 rounded-2xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700 dark:border-rose-400/30 dark:bg-rose-500/10 dark:text-rose-200">
          {error}
        </div>
      ) : null}

      <div className="mt-4 flex justify-end">
        <Button type="submit" disabled={isPending}>
          {isPending ? "Saving budget..." : "Save token budget"}
        </Button>
      </div>
    </form>
  );
}
