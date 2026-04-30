"use client";

import { Check, Copy, Plus, Trash2 } from "lucide-react";
import { FormEvent, useCallback, useMemo, useState, useTransition } from "react";
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
  return `<script src="${process.env.NEXT_PUBLIC_API_URL || ""}/embed/pai-chat.js" data-tenant="${tenantSlug}" data-color="${color}" data-language="${lang}" async></script>`;
}

type ThemeDraft = {
  color: string;
  lang: string;
};

export function EmbedSettingsPanel({ currentUser }: { currentUser: AuthUser }) {
  const queryClient = useQueryClient();
  const [origin, setOrigin] = useState("");
  const [errorMessage, setErrorMessage] = useState("");
  const [copiedSnippet, setCopiedSnippet] = useState("");
  const [isPending, startTransition] = useTransition();

  const { data, isLoading, error } = useQuery({
    queryKey: ["embed", "config"],
    queryFn: getEmbedConfig,
  });

  const color = data?.theme_config.color || defaultColor;
  const lang = data?.theme_config.lang || defaultLang;
  const tenantSlug = currentUser.tenant_slug || currentUser.tenant_id;
  const snippet = useMemo(() => buildSnippet(tenantSlug, color, lang), [tenantSlug, color, lang]);
  const copied = copiedSnippet === snippet;

  const runAction = useCallback((action: () => Promise<void>) => {
    setErrorMessage("");
    startTransition(async () => {
      try {
        await action();
        await queryClient.invalidateQueries({ queryKey: ["embed", "config"] });
      } catch (err) {
        setErrorMessage(err instanceof Error ? err.message : "Embed update failed");
      }
    });
  }, [queryClient]);

  const handleToggle = useCallback((enabled: boolean) => {
    runAction(async () => {
      await updateEmbedConfig({ enabled, theme_config: { color, lang } });
    });
  }, [color, lang, runAction]);

  const handleThemeSubmit = useCallback((theme: ThemeDraft) => {
    const nextTheme = {
      color: theme.color || defaultColor,
      lang: theme.lang.trim() || defaultLang,
    };
    runAction(async () => {
      await updateEmbedConfig({
        enabled: data?.enabled ?? false,
        theme_config: nextTheme,
      });
    });
  }, [data?.enabled, runAction]);

  const handleAddOrigin = useCallback((event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const nextOrigin = origin.trim();
    if (!nextOrigin) return;
    runAction(async () => {
      await addEmbedOrigin(nextOrigin);
      setOrigin("");
    });
  }, [origin, runAction]);

  const handleRemoveOrigin = useCallback((nextOrigin: string) => {
    runAction(async () => {
      await removeEmbedOrigin(nextOrigin);
    });
  }, [runAction]);

  const handleCopy = useCallback(() => {
    startTransition(async () => {
      await navigator.clipboard.writeText(snippet);
      setCopiedSnippet(snippet);
    });
  }, [snippet]);

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
        <WidgetControls
          key={`${color}:${lang}`}
          enabled={data.enabled}
          initialTheme={{ color, lang }}
          isPending={isPending}
          origin={origin}
          allowedOrigins={data.allowed_origins}
          errorMessage={errorMessage}
          handleToggle={handleToggle}
          onThemeSubmit={handleThemeSubmit}
          onOriginChange={setOrigin}
          onAddOrigin={handleAddOrigin}
          onRemoveOrigin={handleRemoveOrigin}
        />
      </AdminSurface>

      <InstallSnippetCard
        copied={copied}
        enabled={data.enabled}
        isPending={isPending}
        snippet={snippet}
        tenantSlug={tenantSlug}
        onCopy={handleCopy}
      />
    </div>
  );
}

function WidgetControls({
  enabled,
  initialTheme,
  isPending,
  origin,
  allowedOrigins,
  errorMessage,
  handleToggle,
  onThemeSubmit,
  onOriginChange,
  onAddOrigin,
  onRemoveOrigin,
}: {
  enabled: boolean;
  initialTheme: ThemeDraft;
  isPending: boolean;
  origin: string;
  allowedOrigins: string[];
  errorMessage: string;
  handleToggle: (enabled: boolean) => void;
  onThemeSubmit: (theme: ThemeDraft) => void;
  onOriginChange: (origin: string) => void;
  onAddOrigin: (event: FormEvent<HTMLFormElement>) => void;
  onRemoveOrigin: (origin: string) => void;
}) {
  const [theme, setTheme] = useState(initialTheme);
  const handleThemeChange = useCallback((field: keyof ThemeDraft, value: string) => {
    setTheme((current) => ({ ...current, [field]: value }));
  }, []);
  const handleThemeSubmit = useCallback((event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    onThemeSubmit(theme);
  }, [onThemeSubmit, theme]);

  return (
    <>
      <AdminSurfaceHeader
        title="Widget controls"
        description="Turn the website chat widget on only after adding the sites allowed to host it."
        action={
          <Switch
            aria-label="Enable embed widget"
            checked={enabled}
            disabled={isPending}
            onCheckedChange={handleToggle}
          />
        }
      />

      <form className="mt-6 grid gap-4 sm:grid-cols-[8rem_minmax(0,1fr)_auto]" onSubmit={handleThemeSubmit}>
        <div className="space-y-2">
          <Label htmlFor="embed-color">Color</Label>
          <Input
            id="embed-color"
            name="color"
            type="color"
            value={theme.color}
            onChange={(event) => handleThemeChange("color", event.target.value)}
            className="h-11 rounded-lg p-1"
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="embed-lang">Language</Label>
          <Input
            id="embed-lang"
            name="lang"
            value={theme.lang}
            maxLength={8}
            onChange={(event) => handleThemeChange("lang", event.target.value)}
            className="h-11 rounded-lg"
          />
        </div>
        <Button type="submit" className="self-end" disabled={isPending}>
          Save
        </Button>
      </form>

      <form className="mt-8 flex flex-col gap-3 sm:flex-row" onSubmit={onAddOrigin}>
        <div className="flex-1 space-y-2">
          <Label htmlFor="embed-origin">Allowed origin</Label>
          <Input
            id="embed-origin"
            value={origin}
            onChange={(event) => onOriginChange(event.target.value)}
            placeholder="https://school.example"
            className="h-11 rounded-lg"
          />
        </div>
        <Button type="submit" className="self-end" disabled={isPending || !origin.trim()}>
          <Plus className="size-4" />
          Add
        </Button>
      </form>

      <AllowedOriginsList origins={allowedOrigins} isPending={isPending} onRemove={onRemoveOrigin} />

      {errorMessage ? <p className="mt-3 text-sm text-destructive">{errorMessage}</p> : null}
    </>
  );
}

function AllowedOriginsList({
  origins,
  isPending,
  onRemove,
}: {
  origins: string[];
  isPending: boolean;
  onRemove: (origin: string) => void;
}) {
  if (!origins.length) {
    return <p className="mt-4 rounded-lg border p-4 text-sm text-muted-foreground">No trusted origins yet.</p>;
  }

  return (
    <div className="mt-4 divide-y rounded-lg border">
      {origins.map((allowedOrigin) => (
        <div key={allowedOrigin} className="flex min-h-12 items-center justify-between gap-3 px-3 py-2">
          <span className="min-w-0 break-all text-sm">{allowedOrigin}</span>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            aria-label={`Remove ${allowedOrigin}`}
            disabled={isPending}
            onClick={() => onRemove(allowedOrigin)}
          >
            <Trash2 className="size-4" />
          </Button>
        </div>
      ))}
    </div>
  );
}

function InstallSnippetCard({
  copied,
  enabled,
  isPending,
  snippet,
  tenantSlug,
  onCopy,
}: {
  copied: boolean;
  enabled: boolean;
  isPending: boolean;
  snippet: string;
  tenantSlug: string;
  onCopy: () => void;
}) {
  return (
    <AdminSurface>
      <AdminSurfaceHeader
        title="Install snippet"
        description="Paste this once on the school site after the origin is trusted."
        action={
          <Button type="button" variant="outline" onClick={onCopy} disabled={isPending}>
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
          <span className={enabled ? "text-emerald-600" : "text-muted-foreground"}>{enabled ? "Enabled" : "Disabled"}</span>
        </div>
        <div className="mt-3 flex items-center justify-between gap-3">
          <span className="font-medium">Tenant</span>
          <span className="min-w-0 break-all text-right text-muted-foreground">{tenantSlug}</span>
        </div>
      </div>
    </AdminSurface>
  );
}
