import Link from "next/link";
import { BarChart3, Coins, Users } from "lucide-react";
import { getServerClassProgress } from "@/lib/server-api";

const cards = [
  {
    title: "Teacher Dashboard",
    text: "Scan class mastery, identify weak topics, and move quickly on learners who need attention.",
    href: "/dashboard",
    icon: BarChart3,
  },
  {
    title: "AI Usage",
    text: "Inspect token volume by provider and model before budget tracking lands in the admin panel.",
    href: "/dashboard/ai-usage",
    icon: Coins,
  },
];

export const dynamic = "force-dynamic";

export default async function Home() {
  let studentCount = "...";

  try {
    const data = await getServerClassProgress("all-students");
    studentCount = String(data.students.length);
  } catch {
    studentCount = "...";
  }

  return (
    <div className="flex flex-col gap-8">
      <section className="flex flex-col gap-6 rounded-[32px] border border-white/60 bg-white/70 p-8 shadow-[0_24px_80px_rgba(15,23,42,0.08)] backdrop-blur md:p-10 dark:border-white/10 dark:bg-slate-950/55 dark:shadow-[0_24px_80px_rgba(2,8,23,0.45)]">
          <div className="flex flex-wrap items-center gap-3 text-xs font-semibold uppercase tracking-[0.24em] text-sky-700 dark:text-sky-300">
            <span className="rounded-full bg-sky-100 px-3 py-1 dark:bg-sky-400/15 dark:text-sky-100">P&AI Bot</span>
            <span className="rounded-full bg-amber-100 px-3 py-1 dark:bg-amber-300/15 dark:text-amber-100">Admin Panel</span>
          </div>
          <div className="grid gap-6 lg:grid-cols-[1.4fr_0.8fr]">
            <div className="space-y-4">
              <h1 className="max-w-3xl text-4xl font-semibold tracking-tight text-slate-950 md:text-5xl dark:text-white">
                Track class momentum, student mastery, and recent tutoring activity in one place.
              </h1>
              <p className="max-w-2xl text-base leading-7 text-slate-600 md:text-lg dark:text-slate-300">
                Use the dashboard to scan class-level performance, then drill into a learner profile to review progress and conversations.
              </p>
              <div className="flex flex-wrap gap-3">
                <Link
                  href="/dashboard"
                  className="inline-flex items-center justify-center rounded-full bg-slate-950 px-5 py-3 text-sm font-medium text-white transition hover:bg-slate-800 dark:bg-sky-400 dark:text-slate-950 dark:hover:bg-sky-300"
                >
                  Open dashboard
                </Link>
                <Link
                  href="/login"
                  className="inline-flex items-center justify-center rounded-full border border-slate-300/80 bg-white/70 px-5 py-3 text-sm font-medium text-slate-900 transition hover:border-sky-300 hover:text-sky-700 dark:border-white/15 dark:bg-slate-950/40 dark:text-slate-100 dark:hover:border-sky-400 dark:hover:text-sky-200"
                >
                  Sign in
                </Link>
              </div>
            </div>
            <div className="grid gap-3 rounded-[28px] bg-slate-950 p-5 text-slate-50 dark:bg-slate-900/90">
              <div className="flex items-center gap-3 rounded-2xl border border-white/10 bg-white/5 p-4 dark:bg-white/[0.03]">
                <Users className="size-5 text-sky-300 dark:text-sky-200" />
                  <div>
                  <p className="text-2xl font-semibold">{studentCount}</p>
                  <p className="text-sm text-slate-300">students currently returned by the admin API</p>
                </div>
              </div>
              <div className="rounded-2xl border border-white/10 bg-white/5 p-4 dark:bg-white/[0.03]">
                <p className="text-sm text-slate-300">What you can inspect</p>
                <p className="mt-2 text-sm leading-6 text-slate-100">
                  Teacher-facing summaries, per-topic mastery, and recent tutoring exchanges from the Go backend.
                </p>
              </div>
            </div>
          </div>
      </section>

      <section className="grid gap-4 md:grid-cols-2">
        {cards.map((card) => {
          const Icon = card.icon;
          return (
            <Link
              key={card.title}
              href={card.href}
              className="group rounded-[28px] border border-slate-200/70 bg-white/80 p-6 shadow-[0_16px_50px_rgba(15,23,42,0.05)] transition hover:-translate-y-1 hover:border-sky-300 hover:shadow-[0_24px_80px_rgba(14,165,233,0.14)] dark:border-white/10 dark:bg-slate-950/55 dark:shadow-[0_20px_60px_rgba(2,8,23,0.35)] dark:hover:border-sky-400 dark:hover:shadow-[0_24px_80px_rgba(56,189,248,0.16)]"
            >
              <Icon className="mb-5 size-6 text-sky-600 transition group-hover:text-amber-500 dark:text-sky-300 dark:group-hover:text-amber-300" />
              <h2 className="text-xl font-semibold tracking-tight dark:text-white">{card.title}</h2>
              <p className="mt-3 text-sm leading-6 text-slate-600 dark:text-slate-300">{card.text}</p>
            </Link>
          );
        })}
      </section>
    </div>
  );
}
