export type AudienceKey = "student" | "teacher";
export type Tone = "destructive" | "secondary" | "primary";

export type ChatTurn = {
  speaker: "Student" | "Bot";
  body: string;
};

export type Stat = {
  label: string;
  value: string;
};

export type Intervention = {
  name: string;
  topic: string;
  score: string;
  tone: Tone;
};

export type TopicScore = {
  name: string;
  value: number;
  tone: Tone;
};

export type AudienceFrame = {
  tabLabel: string;
  kicker: string;
  title: string;
  body: string;
  imageSrc: string;
  imageAlt: string;
  panelTitle: string;
  proof: string;
};

export type StudentAudienceView = AudienceFrame & {
  summaryStats: Stat[];
  chatLabel: string;
  chatTurns: ChatTurn[];
};

export type TeacherAudienceView = AudienceFrame & {
  summaryStats: Stat[];
  interventions: Intervention[];
  topicScores: TopicScore[];
};

export type AudienceView = StudentAudienceView | TeacherAudienceView;

export const audienceViews = {
  student: {
    tabLabel: "Students",
    kicker: "Student practice",
    title: "Unstuck in chat.",
    body: "Ask, try one step, continue in the same thread.",
    imageSrc: "/landing/student-abstract-scene.svg",
    imageAlt: "Abstract chat learning visual with message blocks and step-by-step flow.",
    panelTitle: "Hint. Check. Continue.",
    proof: "Question to next step.",
    summaryStats: [
      { label: "Channel", value: "WhatsApp / Telegram" },
      { label: "Style", value: "Step by step" },
      { label: "Flow", value: "Same thread" },
    ],
    chatLabel: "Functions",
    chatTurns: [
      { speaker: "Student", body: "How do I solve 3x + 5 = 20?" },
      { speaker: "Bot", body: "Subtract 5 first. What is 20 - 5?" },
      { speaker: "Student", body: "15" },
      { speaker: "Bot", body: "Good. Now divide by 3. So x = 5." },
    ],
  },
  teacher: {
    tabLabel: "Teachers",
    kicker: "Teacher follow-up",
    title: "Fix the right topic.",
    body: "See who needs help, what slipped, and what to revisit.",
    imageSrc: "/landing/teacher-abstract-scene.svg",
    imageAlt: "Abstract school dashboard visual with queues, score bars, and follow-up signals.",
    panelTitle: "Daily follow-up list.",
    proof: "Student. Topic. Priority.",
    summaryStats: [
      { label: "Needs help", value: "3 students" },
      { label: "Weakest topic", value: "Functions" },
      { label: "Coverage", value: "12 / 12 filled" },
    ],
    interventions: [
      { name: "Alya Sofea", topic: "WhatsApp • Functions", score: "30%", tone: "destructive" },
      { name: "Hakim Firdaus", topic: "Telegram • Inequalities", score: "21%", tone: "secondary" },
      { name: "Mei Lin", topic: "WhatsApp • Linear equations", score: "92%", tone: "primary" },
    ],
    topicScores: [
      { name: "Functions", value: 30, tone: "destructive" },
      { name: "Inequalities", value: 21, tone: "secondary" },
      { name: "Linear equations", value: 92, tone: "primary" },
    ],
  },
} satisfies {
  student: StudentAudienceView;
  teacher: TeacherAudienceView;
};
