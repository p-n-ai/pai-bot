import { getAverageMastery, getTrackedScores } from "./class-progress.mjs";

export function getDashboardSummary(progress) {
  const students = Array.isArray(progress?.students) ? progress.students : [];
  const topicIds = Array.isArray(progress?.topic_ids) ? progress.topic_ids : [];

  return {
    studentCount: students.length,
    topicCount: topicIds.length,
    trackedScores: getTrackedScores(progress),
    averageMastery: getAverageMastery(progress),
    hasHeatmap: students.length > 0 && topicIds.length > 0,
  };
}
