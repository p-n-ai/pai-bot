export type WidgetTheme = "light" | "dark";

export type WidgetConfig = {
  title: string;
  tenant: string;
  theme: WidgetTheme;
  apiOrigin: string;
};

type SearchParamValue = string | string[] | undefined;

const DEFAULT_TITLE = "P&AI Tutor";
const DEFAULT_TENANT = "public";

function firstParam(value: SearchParamValue) {
  return Array.isArray(value) ? value[0] : value;
}

function cleanText(value: SearchParamValue, fallback: string, maxLength: number) {
  const first = firstParam(value);
  if (!first) return fallback;

  const cleaned = first.replace(/[\u0000-\u001f<>]/g, "").trim();
  return cleaned ? cleaned.slice(0, maxLength) : fallback;
}

function normalizeTheme(value: SearchParamValue): WidgetTheme {
  return firstParam(value) === "dark" ? "dark" : "light";
}

export function normalizeHTTPOrigin(value: SearchParamValue) {
  const first = firstParam(value);
  if (!first) return "";

  try {
    const url = new URL(first);
    if (url.protocol !== "http:" && url.protocol !== "https:") {
      return "";
    }
    return url.origin;
  } catch {
    return "";
  }
}

export function normalizeWidgetConfig(
  params: Record<string, SearchParamValue>,
  fallbackApiOrigin = "",
): WidgetConfig {
  return {
    title: cleanText(params.title, DEFAULT_TITLE, 80),
    tenant: cleanText(params.tenant, DEFAULT_TENANT, 80),
    theme: normalizeTheme(params.theme),
    apiOrigin: normalizeHTTPOrigin(params.apiOrigin) || normalizeHTTPOrigin(fallbackApiOrigin),
  };
}

export function buildWidgetWebSocketURL({
  apiOrigin,
  locationOrigin,
}: {
  apiOrigin?: string;
  locationOrigin: string;
}) {
  const origin = normalizeHTTPOrigin(apiOrigin) || normalizeHTTPOrigin(locationOrigin);
  const url = new URL("/ws/chat", origin || "http://localhost");
  url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
  return url.toString();
}
