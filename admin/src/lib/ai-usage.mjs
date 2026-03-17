function isRecord(value) {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

export function normalizeAIUsage(payload) {
  const source = isRecord(payload) ? payload : {};
  const providers = Array.isArray(source.providers)
    ? source.providers
        .filter(isRecord)
        .map((provider) => ({
          provider: typeof provider.provider === "string" ? provider.provider : "unknown",
          model: typeof provider.model === "string" ? provider.model : "",
          messages: typeof provider.messages === "number" ? provider.messages : 0,
          input_tokens: typeof provider.input_tokens === "number" ? provider.input_tokens : 0,
          output_tokens: typeof provider.output_tokens === "number" ? provider.output_tokens : 0,
          total_tokens:
            typeof provider.total_tokens === "number"
              ? provider.total_tokens
              : (typeof provider.input_tokens === "number" ? provider.input_tokens : 0) +
                (typeof provider.output_tokens === "number" ? provider.output_tokens : 0),
        }))
    : [];

  return {
    total_messages: typeof source.total_messages === "number" ? source.total_messages : 0,
    total_input_tokens: typeof source.total_input_tokens === "number" ? source.total_input_tokens : 0,
    total_output_tokens: typeof source.total_output_tokens === "number" ? source.total_output_tokens : 0,
    providers,
  };
}

export function getTopProvider(usage) {
  const providers = Array.isArray(usage?.providers) ? usage.providers : [];
  if (providers.length === 0) {
    return null;
  }

  return providers.reduce((best, current) => {
    if (!best || current.total_tokens > best.total_tokens) {
      return current;
    }
    return best;
  }, null);
}

export function formatCompactNumber(value) {
  return new Intl.NumberFormat("en", { notation: "compact", maximumFractionDigits: 1 }).format(value || 0);
}
