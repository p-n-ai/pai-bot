import { buildStudentActivityGrid } from "./student-activity.mjs";

/**
 * @param {Awaited<ReturnType<import("./api").getStudentDetail>> | null | undefined} detail
 * @param {import("./api").StudentConversation[] | null | undefined} conversations
 */
export function buildStudentViewModel(detail, conversations) {
  const progress = Array.isArray(detail?.progress) ? detail.progress : [];
  const safeConversations = Array.isArray(conversations) ? conversations : [];

  return {
    radarData: progress.map((item) => ({
      topic: item.topic_id,
      mastery: Math.round((item.mastery_score || 0) * 100),
    })),
    struggleAreas: progress.filter((item) => (item.mastery_score || 0) < 0.6),
    hasProgress: progress.length > 0,
    hasConversations: safeConversations.length > 0,
    activityGrid: buildStudentActivityGrid(safeConversations),
  };
}
