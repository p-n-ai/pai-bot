/**
 * @param {import("./api").ParentSummary | null | undefined} summary
 */
export function getParentViewModel(summary) {
  const masteryRows = Array.isArray(summary?.mastery) ? summary.mastery : [];

  return {
    masteryRows,
    hasMastery: masteryRows.length > 0,
    encouragementHeadline: summary?.encouragement?.headline ?? "A suggested encouragement will appear here soon.",
    encouragementText:
      summary?.encouragement?.text ??
      "Once the weekly summary is ready, you'll see a short message you can send or say at home.",
  };
}
