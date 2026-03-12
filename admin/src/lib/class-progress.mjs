function isRecord(value) {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

export function normalizeClassProgress(payload) {
  const source = isRecord(payload) ? payload : {};
  const students = Array.isArray(source.students)
    ? source.students.map((student) => {
        const safeStudent = isRecord(student) ? student : {};
        return {
          id: typeof safeStudent.id === "string" ? safeStudent.id : "",
          name: typeof safeStudent.name === "string" ? safeStudent.name : "Unknown student",
          topics: isRecord(safeStudent.topics) ? safeStudent.topics : {},
        };
      })
    : [];

  const topicIds = Array.isArray(source.topic_ids)
    ? source.topic_ids.filter((topicId) => typeof topicId === "string")
    : [];

  return {
    students,
    topic_ids: topicIds,
  };
}

export function getAverageMastery(progress) {
  const students = Array.isArray(progress?.students) ? progress.students : [];
  const topicIds = Array.isArray(progress?.topic_ids) ? progress.topic_ids : [];
  const totalSlots = students.length * topicIds.length;
  if (totalSlots === 0) {
    return 0;
  }

  const totalScore = students.reduce((sum, student) => {
    const topics = isRecord(student?.topics) ? student.topics : {};
    return (
      sum +
      topicIds.reduce((topicSum, topicId) => {
        const rawScore = topics[topicId];
        return topicSum + (typeof rawScore === "number" ? rawScore : 0);
      }, 0)
    );
  }, 0);

  return Math.round((totalScore / totalSlots) * 100);
}

export function getTrackedScores(progress) {
  const students = Array.isArray(progress?.students) ? progress.students : [];
  return students.reduce((sum, student) => {
    const topics = isRecord(student?.topics) ? student.topics : {};
    return sum + Object.keys(topics).length;
  }, 0);
}
