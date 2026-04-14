const mockClasses = [
  {
    id: "class-f1-algebra-a",
    name: "Form 1 Algebra A",
    syllabus: "KSSM Form 1",
    subject: "Mathematics",
    joinCode: "ALG-F1A",
    cadence: "Mon, Wed, Fri",
    summary: "Early algebra cohort preparing for linear equations and algebraic expressions.",
    memberCount: 24,
    activeStudents: 18,
    averageMastery: 0.67,
    assignedTopics: [
      { id: "algebraic-expressions", title: "Algebraic Expressions", status: "Live", progress: 0.76 },
      { id: "linear-equations", title: "Linear Equations", status: "Starting next", progress: 0.42 },
    ],
    members: [
      { id: "student-001", name: "Alya", status: "Active", channel: "telegram", mastery: 0.82 },
      { id: "student-002", name: "Hakim", status: "Needs nudge", channel: "telegram", mastery: 0.48 },
      { id: "student-003", name: "Sofia", status: "Active", channel: "telegram", mastery: 0.71 },
      { id: "student-004", name: "Rizqi", status: "Review due", channel: "telegram", mastery: 0.59 },
    ],
  },
  {
    id: "class-f2-algebra-b",
    name: "Form 2 Algebra B",
    syllabus: "KSSM Form 2",
    subject: "Mathematics",
    joinCode: "ALG-F2B",
    cadence: "Tue, Thu",
    summary: "Mixed-ability class focused on algebraic formulae and substitution practice.",
    memberCount: 19,
    activeStudents: 13,
    averageMastery: 0.58,
    assignedTopics: [
      { id: "formulae", title: "Formulae and Substitution", status: "Live", progress: 0.61 },
      { id: "expansion", title: "Algebraic Expansion", status: "Planned", progress: 0.18 },
    ],
    members: [
      { id: "student-101", name: "Irfan", status: "Active", channel: "telegram", mastery: 0.66 },
      { id: "student-102", name: "Amina", status: "Needs nudge", channel: "telegram", mastery: 0.44 },
      { id: "student-103", name: "Devi", status: "Active", channel: "telegram", mastery: 0.63 },
    ],
  },
  {
    id: "class-f3-bridging",
    name: "Form 3 Bridging Group",
    syllabus: "KSSM Form 3",
    subject: "Mathematics",
    joinCode: "ALG-F3X",
    cadence: "Sat bootcamp",
    summary: "Short-cycle intervention class for learners who need reinforcement before assessment week.",
    memberCount: 12,
    activeStudents: 9,
    averageMastery: 0.51,
    assignedTopics: [
      { id: "simultaneous-equations", title: "Simultaneous Equations", status: "Live", progress: 0.53 },
      { id: "inequalities", title: "Linear Inequalities", status: "Needs setup", progress: 0 },
    ],
    members: [
      { id: "student-201", name: "Nadia", status: "Review due", channel: "telegram", mastery: 0.38 },
      { id: "student-202", name: "Faris", status: "Active", channel: "telegram", mastery: 0.57 },
    ],
  },
];

function clampScore(value) {
  return Math.max(0, Math.min(1, Math.round(value * 100) / 100));
}

export function getMockClasses() {
  return mockClasses.map((item) => ({
    ...item,
    assignedTopics: item.assignedTopics.map((topic) => ({ ...topic })),
    members: item.members.map((member) => ({ ...member })),
  }));
}

export function getMockClassProgress(classId = "all-students") {
  const classes = getMockClasses();
  const selectedClasses = classId === "all-students" ? classes : classes.filter((item) => item.id === classId);
  const topicIds = [...new Set(selectedClasses.flatMap((item) => item.assignedTopics.map((topic) => topic.id)))];

  return {
    topic_ids: topicIds,
    students: selectedClasses.flatMap((item, classIndex) =>
      item.members.map((member, memberIndex) => ({
        id: member.id,
        name: member.name,
        topics: Object.fromEntries(
          topicIds.map((topicId, topicIndex) => {
            const assignedTopic = item.assignedTopics.find((topic) => topic.id === topicId);
            const topicProgress = assignedTopic?.progress ?? member.mastery;
            const offset = ((classIndex + memberIndex + topicIndex) % 3 - 1) * 0.05;
            const score = clampScore(member.mastery * 0.65 + topicProgress * 0.35 + offset);
            return [topicId, score];
          }),
        ),
      })),
    ),
  };
}

export function getClassManagementSummary(classes) {
  const safeClasses = Array.isArray(classes) ? classes : [];
  const classCount = safeClasses.length;
  const totalMembers = safeClasses.reduce((sum, item) => sum + (item.memberCount || 0), 0);
  const activeStudents = safeClasses.reduce((sum, item) => sum + (item.activeStudents || 0), 0);
  const joinCodes = safeClasses.map((item) => item.joinCode).filter(Boolean);
  const averageMastery =
    classCount > 0
      ? Math.round((safeClasses.reduce((sum, item) => sum + (item.averageMastery || 0), 0) / classCount) * 100)
      : 0;

  return {
    classCount,
    totalMembers,
    activeStudents,
    averageMastery,
    joinCodes,
  };
}

export function getMockAssignedTopics() {
  return [
    { id: "algebraic-expressions", title: "Algebraic Expressions", status: "Live", progress: 0.76 },
    { id: "linear-equations", title: "Linear Equations", status: "Starting next", progress: 0.42 },
    { id: "fractions", title: "Fractions", status: "Upcoming", progress: 0 },
  ];
}
