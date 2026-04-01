import { getAverageMastery, getTrackedScores } from "./class-progress.mjs";

export function getDashboardSummary(progress) {
  const students = Array.isArray(progress?.students) ? progress.students : [];
  const topicIds = Array.isArray(progress?.topic_ids) ? progress.topic_ids : [];
  const trackedScores = getTrackedScores(progress);
  const totalSlots = students.length * topicIds.length;
  const topicAverages = topicIds
    .map((topicId) => {
      const total = students.reduce((sum, student) => {
        const score = student?.topics?.[topicId];
        return sum + (typeof score === "number" ? score : 0);
      }, 0);

      return {
        topicId,
        score: students.length > 0 ? Math.round((total / students.length) * 100) : 0,
      };
    })
    .sort((left, right) => left.score - right.score);
  const studentAverages = students.map((student) => {
    if (topicIds.length === 0) {
      return 0;
    }

    const total = topicIds.reduce((sum, topicId) => {
      const score = student?.topics?.[topicId];
      return sum + (typeof score === "number" ? score : 0);
    }, 0);

    return Math.round((total / topicIds.length) * 100);
  });

  return {
    studentCount: students.length,
    topicCount: topicIds.length,
    trackedScores,
    averageMastery: getAverageMastery(progress),
    hasHeatmap: students.length > 0 && topicIds.length > 0,
    coveragePercent: totalSlots > 0 ? Math.round((trackedScores / totalSlots) * 100) : 0,
    attentionCount: studentAverages.filter((score) => score < 50).length,
    weakestTopic: topicAverages[0] ?? null,
    strongestTopic: topicAverages.at(-1) ?? null,
  };
}
