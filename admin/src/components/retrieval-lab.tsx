"use client";

import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";
import { IconChevronDown, IconSearch } from "@tabler/icons-react";
import { Button, buttonVariants } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { clearSession } from "@/lib/api";
import { cn } from "@/lib/utils";

type RetrievalDocument = {
  id: string;
  collection_id?: string;
  kind: string;
  title: string;
  body?: string;
  source_type?: string;
  metadata?: Record<string, string>;
};

type SearchHit = {
  document: RetrievalDocument;
  score: number;
  matched_terms: number;
  high_signal_terms: number;
  excerpt: string;
};

type SearchPayload = {
  query: string;
  limit: number;
  collection_ids?: string[];
  kinds?: string[];
  source_types?: string[];
  metadata?: Record<string, string>;
  include_inactive?: boolean;
};

const EXAMPLE = {
  query: "first step only for linear equations",
  limit: "5",
  repeats: "3",
  collectionIds: "curriculum:matematik-form-1",
  kinds: "topic_card,teaching_note",
  sourceTypes: "curriculum",
  metadata: '{"form":"1"}',
};

function parseCSV(value: string) {
  return value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

function buildPayload(state: {
  query: string;
  limit: string;
  collectionIds: string;
  kinds: string;
  sourceTypes: string;
  metadata: string;
  includeInactive: boolean;
}): SearchPayload {
  const metadata = state.metadata.trim() ? JSON.parse(state.metadata) : undefined;
  const collectionIDs = parseCSV(state.collectionIds);
  const kinds = parseCSV(state.kinds);
  const sourceTypes = parseCSV(state.sourceTypes);

  return {
    query: state.query.trim(),
    limit: Number(state.limit) || 5,
    collection_ids: collectionIDs.length ? collectionIDs : undefined,
    kinds: kinds.length ? kinds : undefined,
    source_types: sourceTypes.length ? sourceTypes : undefined,
    metadata,
    include_inactive: state.includeInactive,
  };
}

function isExpiredSessionMessage(message: string) {
  return message.includes("401 Unauthorized: expired token") || message.includes("401 Unauthorized: missing bearer token");
}

export function RetrievalLab() {
  const router = useRouter();
  const [query, setQuery] = useState("linear equation");
  const [limit, setLimit] = useState("5");
  const [repeats, setRepeats] = useState("3");
  const [collectionIds, setCollectionIds] = useState("");
  const [kinds, setKinds] = useState("");
  const [sourceTypes, setSourceTypes] = useState("");
  const [metadata, setMetadata] = useState("");
  const [includeInactive, setIncludeInactive] = useState(false);
  const [hits, setHits] = useState<SearchHit[]>([]);
  const [durations, setDurations] = useState<number[]>([]);
  const [raw, setRaw] = useState("Run a query to inspect the raw payload and hits.");
  const [status, setStatus] = useState("Idle.");
  const [statusTone, setStatusTone] = useState<"idle" | "ok" | "err">("idle");
  const [isRunning, setIsRunning] = useState(false);
  const [showAdvanced, setShowAdvanced] = useState(false);

  const metrics = useMemo(() => {
    const total = durations.reduce((sum, value) => sum + value, 0);
    const average = durations.length ? total / durations.length : 0;
    const min = durations.length ? Math.min(...durations) : 0;
    const max = durations.length ? Math.max(...durations) : 0;
    const last = durations.length ? durations[durations.length - 1] : 0;

    return {
      last: Math.round(last),
      average: Math.round(average),
      spread: `${Math.round(min)} / ${Math.round(max)}`,
      count: hits.length,
    };
  }, [durations, hits]);

  const showResults = durations.length > 0 || hits.length > 0 || statusTone === "err";
  const normalizedError = useMemo(() => {
    if (statusTone !== "err") {
      return "";
    }
    if (status.includes("404 Not Found: 404 page not found")) {
      return "Retrieval search endpoint unavailable. Restart the local backend and try again.";
    }
    if (status.includes("401 Unauthorized: expired token")) {
      return "Your admin session expired. Sign in again to keep testing retrieval.";
    }
    if (status.includes("401 Unauthorized: missing bearer token")) {
      return "You are signed out. Sign in again to use retrieval search.";
    }
    if (status.startsWith("Bad metadata JSON:")) {
      return status;
    }
    return status || "Retrieval search failed.";
  }, [status, statusTone]);
  async function runSearch() {
    let payload: SearchPayload;
    try {
      payload = buildPayload({
        query,
        limit,
        collectionIds,
        kinds,
        sourceTypes,
        metadata,
        includeInactive,
      });
    } catch (error) {
      setStatus(`Bad metadata JSON: ${(error as Error).message}`);
      setStatusTone("err");
      return;
    }

    if (!payload.query) {
      setStatus("Query required.");
      setStatusTone("err");
      return;
    }

    setIsRunning(true);
    setStatus("Running...");
    setStatusTone("idle");

    const runDurations: number[] = [];
    let lastText = "";
    let lastHits: SearchHit[] = [];

    try {
      const runCount = Math.max(1, Number(repeats) || 1);
      for (let index = 0; index < runCount; index += 1) {
        const startedAt = performance.now();
        const response = await fetch("/api/retrieval-lab/search", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(payload),
        });
        const endedAt = performance.now();
        runDurations.push(endedAt - startedAt);

        lastText = await response.text();
        if (!response.ok) {
          throw new Error(`${response.status} ${response.statusText}: ${lastText}`);
        }
        lastHits = lastText ? (JSON.parse(lastText) as SearchHit[]) : [];
      }

      setDurations(runDurations);
      setHits(lastHits);
      setRaw(JSON.stringify({ payload, durations: runDurations, hits: lastHits }, null, 2));
      setStatus(`Done. ${runDurations.length} run(s).`);
      setStatusTone("ok");
    } catch (error) {
      const message = (error as Error).message;
      if (isExpiredSessionMessage(message)) {
        clearSession();
        router.push("/login?next=%2Fdashboard%2Fretrieval-lab");
        router.refresh();
        return;
      }
      setDurations(runDurations);
      setHits([]);
      setRaw(lastText || String(error));
      setStatus(message);
      setStatusTone("err");
    } finally {
      setIsRunning(false);
    }
  }

  function loadExample() {
    setQuery(EXAMPLE.query);
    setLimit(EXAMPLE.limit);
    setRepeats(EXAMPLE.repeats);
    setCollectionIds(EXAMPLE.collectionIds);
    setKinds(EXAMPLE.kinds);
    setSourceTypes(EXAMPLE.sourceTypes);
    setMetadata(EXAMPLE.metadata);
    setIncludeInactive(false);
    setStatus("Example loaded.");
    setStatusTone("idle");
  }

  return (
    <div className="space-y-10">
      <form
        className="space-y-5"
        onSubmit={(event) => {
          event.preventDefault();
          void runSearch();
        }}
      >
        <Collapsible open={showAdvanced} onOpenChange={setShowAdvanced} className="space-y-4">
          <div className="mx-auto flex max-w-4xl flex-col items-center gap-5 pt-6 text-center">
            <div className="space-y-2">
              <p className="font-serif text-4xl tracking-[-0.04em] text-foreground sm:text-5xl">PaiBot Search</p>
              <p className="max-w-2xl text-sm text-muted-foreground">
                Search retrieval content across collections and sources.
              </p>
            </div>

            <div className="w-full max-w-[756px] space-y-4">
              <div className="flex items-center gap-3 rounded-full border border-border/80 bg-background px-5 shadow-sm">
                <IconSearch className="size-5 text-muted-foreground" />
                <Input
                  value={query}
                  onChange={(event) => setQuery(event.target.value)}
                  className="h-16 border-0 bg-transparent px-0 text-base shadow-none focus-visible:ring-0"
                  placeholder="Search retrieval content..."
                />
              </div>

              <div className="flex items-center justify-center gap-3">
                <Button type="submit" className="min-w-28 rounded-md px-5" disabled={isRunning}>
                  {isRunning ? "Running..." : "Search"}
                </Button>

                <CollapsibleTrigger
                  className={cn(
                    buttonVariants({ variant: "ghost", size: "sm" }),
                    "h-9 min-w-0 gap-2 rounded-md px-3 text-sm text-muted-foreground",
                  )}
                >
                  <span>Additional settings</span>
                  <IconChevronDown className={`size-4 transition-transform ${showAdvanced ? "rotate-180" : ""}`} />
                </CollapsibleTrigger>
              </div>
            </div>
          </div>
          <CollapsibleContent className="mx-auto max-w-[756px] rounded-2xl border border-border/70 bg-muted/20 px-4 py-4">
            <div className="mx-auto max-w-[756px] space-y-4">
              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <label className="text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">Limit</label>
                  <Input value={limit} onChange={(event) => setLimit(event.target.value)} inputMode="numeric" />
                </div>
                <div className="space-y-2">
                  <label className="text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">Repeat runs</label>
                  <Input value={repeats} onChange={(event) => setRepeats(event.target.value)} inputMode="numeric" />
                </div>
              </div>

              <div className="space-y-2">
                <label className="text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">Collection IDs</label>
                <Input value={collectionIds} onChange={(event) => setCollectionIds(event.target.value)} placeholder="curriculum:math-f1,curriculum:math-f2" />
              </div>

              <div className="grid gap-4 md:grid-cols-2">
                <div className="space-y-2">
                  <label className="text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">Kinds</label>
                  <Input value={kinds} onChange={(event) => setKinds(event.target.value)} placeholder="topic_card,teaching_note" />
                </div>
                <div className="space-y-2">
                  <label className="text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">Source types</label>
                  <Input value={sourceTypes} onChange={(event) => setSourceTypes(event.target.value)} placeholder="curriculum,youtube" />
                </div>
              </div>

              <div className="space-y-2">
                <label className="text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">Metadata JSON</label>
                <Textarea value={metadata} onChange={(event) => setMetadata(event.target.value)} className="min-h-24 rounded-2xl font-mono text-xs" placeholder='{"form":"1","topic_id":"F1-02"}' />
              </div>

              <label className="flex items-center gap-3 rounded-2xl border border-border/70 bg-background/60 px-4 py-3 text-sm">
                <Checkbox checked={includeInactive} onCheckedChange={(checked) => setIncludeInactive(Boolean(checked))} />
                Include inactive records
              </label>

              <div className="flex flex-wrap items-center justify-between gap-3 text-xs text-muted-foreground">
                <div className="flex flex-wrap gap-2">
                  <span className="rounded-full border border-border/70 bg-background/60 px-3 py-1.5">
                    Status: {statusTone === "err" ? "Error" : statusTone === "ok" ? "Ready" : "Idle"}
                  </span>
                  <span className="rounded-full border border-border/70 bg-background/60 px-3 py-1.5">
                    Last: {metrics.last} ms
                  </span>
                  <span className="rounded-full border border-border/70 bg-background/60 px-3 py-1.5">
                    Avg: {metrics.average} ms
                  </span>
                </div>
                <Button type="button" variant="outline" className="rounded-md" onClick={loadExample}>
                  Load example
                </Button>
              </div>
            </div>
          </CollapsibleContent>
        </Collapsible>
      </form>

      {showResults ? (
        <section className="mx-auto max-w-4xl space-y-5">
          {statusTone === "err" ? (
            <div className="rounded-2xl border border-red-200 bg-red-50/80 px-5 py-4">
              <p className="text-sm font-medium text-red-900">Search failed</p>
              <p className="mt-1 text-sm leading-6 text-red-800">{normalizedError}</p>
            </div>
          ) : (
            <div className="flex flex-wrap items-center justify-between gap-3 text-sm text-muted-foreground">
              <Tooltip>
                <TooltipTrigger
                  className="cursor-help text-left underline decoration-dotted underline-offset-4"
                  aria-label="Search run summary"
                >
                  {status}
                </TooltipTrigger>
                <TooltipContent>
                  Number of benchmark runs completed for this query. Use repeats to compare latency and ranking stability.
                </TooltipContent>
              </Tooltip>
              <p>
                {metrics.count} hit{metrics.count === 1 ? "" : "s"} in about {metrics.average} ms
              </p>
            </div>
          )}

          {statusTone !== "err" && hits.length === 0 ? (
            <div className="rounded-2xl border border-border/70 bg-muted/20 px-5 py-4 text-sm text-muted-foreground">
              No hits yet.
            </div>
          ) : statusTone !== "err" ? (
            <ol className="space-y-6">
              {hits.map((hit, index) => (
                <li key={hit.document.id} className="space-y-1.5">
                  <p className="text-xs text-muted-foreground">
                    {hit.document.collection_id || "unscoped"} • {hit.document.kind} • score {hit.score.toFixed(2)}
                  </p>
                  <h2 className="text-xl tracking-[-0.02em] text-foreground">{hit.document.title}</h2>
                  <p className="text-sm leading-7 text-muted-foreground">{hit.excerpt}</p>
                  <div className="flex flex-wrap gap-2 text-xs text-muted-foreground">
                    <span className="rounded-full border border-border/70 px-3 py-1">
                      Rank {index + 1}
                    </span>
                    <span className="rounded-full border border-border/70 px-3 py-1">
                      Terms {hit.matched_terms}/{hit.high_signal_terms}
                    </span>
                    <span className="rounded-full border border-border/70 px-3 py-1">
                      {hit.document.id}
                    </span>
                  </div>
                </li>
              ))}
            </ol>
          ) : null}

          <Collapsible className="border-t border-border/70 pt-4">
            <CollapsibleTrigger
              className={cn(
                buttonVariants({ variant: "ghost", size: "sm" }),
                "h-auto rounded-md px-0 py-0 text-sm text-muted-foreground",
              )}
            >
              Raw output
            </CollapsibleTrigger>
            <CollapsibleContent className="pt-3">
              <pre className="overflow-x-auto rounded-2xl border border-border/70 bg-muted/20 p-4 text-xs leading-6 text-foreground">{raw}</pre>
            </CollapsibleContent>
          </Collapsible>
        </section>
      ) : null}
    </div>
  );
}
