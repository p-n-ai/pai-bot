import { getAverageMastery, getTrackedScores } from "./class-progress.mjs";

export function getDashboardSummary(progress) {
  const students = Array.isArray(progress?.students) ? progress.students : [];
  const topicIds = Array.isArray(progress?.topic_ids) ? progress.topic_ids : [];
  const trackedScores = getTrackedScores(progress);
  const totalSlots = students.length * topicIds.length;
  const topicAverages = topicIds
    .map((topicId) => {
      const scores = students.flatMap((student) => {
        const score = student?.topics?.[topicId];
        return typeof score === "number" ? [score] : [];
      });

      return {
        topicId,
        score: scores.length > 0 ? Math.round((scores.reduce((sum, score) => sum + score, 0) / scores.length) * 100) : null,
      };
    })
    .filter((topic) => typeof topic.score === "number")
    .sort((left, right) => left.score - right.score);
  const studentAverages = students.map((student) => {
    const scores = topicIds.flatMap((topicId) => {
      const score = student?.topics?.[topicId];
      return typeof score === "number" ? [score] : [];
    });

    if (scores.length === 0) {
      return null;
    }

    const total = scores.reduce((sum, score) => sum + score, 0);
    return Math.round((total / scores.length) * 100);
  });

  return {
    studentCount: students.length,
    topicCount: topicIds.length,
    trackedScores,
    averageMastery: getAverageMastery(progress),
    hasHeatmap: students.length > 0 && topicIds.length > 0,
    coveragePercent: totalSlots > 0 ? Math.round((trackedScores / totalSlots) * 100) : 0,
    attentionCount: studentAverages.filter((score) => typeof score === "number" && score < 50).length,
    weakestTopic: topicAverages[0] ?? null,
    strongestTopic: topicAverages.at(-1) ?? null,
  };
}
