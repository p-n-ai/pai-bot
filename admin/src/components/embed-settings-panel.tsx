"use client";

import { Check, Copy, Plus, Trash2 } from "lucide-react";
import { FormEvent, useMemo, useState, useTransition } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { AdminSurface, AdminSurfaceHeader } from "@/components/admin-surface";
import { StatePanel } from "@/components/state-panel";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { addEmbedOrigin, getEmbedConfig, removeEmbedOrigin, updateEmbedConfig, type AuthUser } from "@/lib/api";

const defaultColor = "#2563eb";
const defaultLang = "ms";

function buildSnippet(tenantSlug: string, color: string, lang: string) {
  return `<script src="${process.env.NEXT_PUBLIC_API_URL || ""}/embed/pai-chat.js" data-tenant="${tenantSlug}" data-color="${color}" data-lang="${lang}" async></script>`;
}

export function EmbedSettingsPanel({ currentUser }: { currentUser: AuthUser }) {
  const queryClient = useQueryClient();
  const [origin, setOrigin] = useState("");
  const [errorMessage, setErrorMessage] = useState("");
  const [copied, setCopied] = useState(false);
  const [isPending, startTransition] = useTransition();

  const { data, isLoading, error } = useQuery({
    queryKey: ["embed", "config"],
    queryFn: getEmbedConfig,
  });

  const color = data?.theme_config.color || defaultColor;
  const lang = data?.theme_config.lang || defaultLang;
  const tenantSlug = currentUser.tenant_slug || currentUser.tenant_id;
  const snippet = useMemo(() => buildSnippet(tenantSlug, color, lang), [tenantSlug, color, lang]);

  function refreshConfig() {
    queryClient.invalidateQueries({ queryKey: ["embed", "config"] });
  }

  function runAction(action: () => Promise<void>) {
    setErrorMessage("");
    startTransition(async () => {
      try {
        await action();
        refreshConfig();
      } catch (err) {
        setErrorMessage(err instanceof Error ? err.message : "Embed update failed");
      }
    });
  }

  function handleToggle(enabled: boolean) {
    runAction(async () => {
      await updateEmbedConfig({ enabled, theme_config: { color, lang } });
    });
  }

  function handleThemeSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const formData = new FormData(event.currentTarget);
    runAction(async () => {
      await updateEmbedConfig({
        enabled: data?.enabled ?? false,
        theme_config: {
          color: String(formData.get("color") || defaultColor),
          lang: String(formData.get("lang") || defaultLang).trim() || defaultLang,
        },
      });
    });
  }

  function handleAddOrigin(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const nextOrigin = origin.trim();
    if (!nextOrigin) return;
    runAction(async () => {
      await addEmbedOrigin(nextOrigin);
      setOrigin("");
    });
  }

  function handleRemoveOrigin(nextOrigin: string) {
    runAction(async () => {
      await removeEmbedOrigin(nextOrigin);
    });
  }

  function handleCopy() {
    setCopied(false);
    startTransition(async () => {
      await navigator.clipboard.writeText(snippet);
      setCopied(true);
    });
  }

  if (isLoading) {
    return <StatePanel tone="loading" title="Loading embed settings..." description="Checking widget status and allowed origins." />;
  }

  if (error || !data) {
    return (
      <StatePanel
        tone="error"
        title="Could not load embed settings"
        description={error instanceof Error ? error.message : "Check that the admin embed API is available."}
      />
    );
  }

  return (
    <div className="grid gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(20rem,0.85fr)]">
      <AdminSurface>
        <AdminSurfaceHeader
          title="Widget controls"
          description="Turn the website chat widget on only after adding the sites allowed to host it."
          action={
            <Switch
              aria-label="Enable embed widget"
              checked={data.enabled}
              disabled={isPending}
              onCheckedChange={handleToggle}
            />
          }
        />

        <form className="mt-6 grid gap-4 sm:grid-cols-[8rem_minmax(0,1fr)_auto]" onSubmit={handleThemeSubmit}>
          <div className="space-y-2">
            <Label htmlFor="embed-color">Color</Label>
            <Input id="embed-color" name="color" type="color" defaultValue={color} className="h-11 rounded-lg p-1" />
          </div>
          <div className="space-y-2">
            <Label htmlFor="embed-lang">Language</Label>
            <Input id="embed-lang" name="lang" defaultValue={lang} maxLength={8} className="h-11 rounded-lg" />
          </div>
          <Button type="submit" className="self-end" disabled={isPending}>
            Save
          </Button>
        </form>

        <form className="mt-8 flex flex-col gap-3 sm:flex-row" onSubmit={handleAddOrigin}>
          <div className="flex-1 space-y-2">
            <Label htmlFor="embed-origin">Allowed origin</Label>
            <Input
              id="embed-origin"
              value={origin}
              onChange={(event) => setOrigin(event.target.value)}
              placeholder="https://school.example"
              className="h-11 rounded-lg"
            />
          </div>
          <Button type="submit" className="self-end" disabled={isPending || !origin.trim()}>
            <Plus className="size-4" />
            Add
          </Button>
        </form>

        <div className="mt-4 divide-y rounded-lg border">
          {data.allowed_origins.length ? (
            data.allowed_origins.map((allowedOrigin) => (
              <div key={allowedOrigin} className="flex min-h-12 items-center justify-between gap-3 px-3 py-2">
                <span className="min-w-0 break-all text-sm">{allowedOrigin}</span>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  aria-label={`Remove ${allowedOrigin}`}
                  disabled={isPending}
                  onClick={() => handleRemoveOrigin(allowedOrigin)}
                >
                  <Trash2 className="size-4" />
                </Button>
              </div>
            ))
          ) : (
            <p className="p-4 text-sm text-muted-foreground">No trusted origins yet.</p>
          )}
        </div>

        {errorMessage ? <p className="mt-3 text-sm text-destructive">{errorMessage}</p> : null}
      </AdminSurface>

      <AdminSurface>
        <AdminSurfaceHeader
          title="Install snippet"
          description="Paste this once on the school site after the origin is trusted."
          action={
            <Button type="button" variant="outline" onClick={handleCopy} disabled={isPending}>
              {copied ? <Check className="size-4" /> : <Copy className="size-4" />}
              {copied ? "Copied" : "Copy"}
            </Button>
          }
        />
        <pre className="mt-6 overflow-x-auto rounded-lg border bg-muted/50 p-4 text-xs leading-5 text-foreground">
          <code>{snippet}</code>
        </pre>
        <div className="mt-4 rounded-lg border bg-background p-4 text-sm">
          <div className="flex items-center justify-between gap-3">
            <span className="font-medium">Status</span>
            <span className={data.enabled ? "text-emerald-600" : "text-muted-foreground"}>
              {data.enabled ? "Enabled" : "Disabled"}
            </span>
          </div>
          <div className="mt-3 flex items-center justify-between gap-3">
            <span className="font-medium">Tenant</span>
            <span className="min-w-0 break-all text-right text-muted-foreground">{tenantSlug}</span>
          </div>
        </div>
      </AdminSurface>
    </div>
  );
}
