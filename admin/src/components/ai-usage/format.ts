const dayFormatter = new Intl.DateTimeFormat("en-US", {
  month: "short",
  day: "numeric",
  timeZone: "UTC",
});

export function formatAIUsageDateLabel(value: string) {
  if (!value) {
    return "Not set";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return dayFormatter.format(date);
}

export function formatBudgetWindowLabel(start: string, end: string) {
  if (!start && !end) {
    return "No active token window";
  }

  if (!start) {
    return `Ends ${formatAIUsageDateLabel(end)}`;
  }

  if (!end) {
    return `Started ${formatAIUsageDateLabel(start)}`;
  }

  return `${formatAIUsageDateLabel(start)} to ${formatAIUsageDateLabel(end)}`;
}
