export function formatParentTopicLabel(topicId) {
  return String(topicId || "")
    .split("-")
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

export function getParentMasteryTone(score) {
  if (score >= 0.8) return "bg-emerald-500";
  if (score >= 0.6) return "bg-lime-500";
  if (score >= 0.4) return "bg-amber-400";
  return "bg-rose-400";
}

export function buildParentContextLine(summary) {
  if (!summary?.child || !summary?.parent) {
    return "Pulling weekly activity, mastery, and encouragement from the admin API.";
  }

  const contact = summary.parent.email || summary.parent.name || "Parent";
  return `${summary.child.form} | ${summary.child.channel} | Parent contact ${contact}`;
}
