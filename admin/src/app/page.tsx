import Link from "next/link";
import { BarChart3, GraduationCap, ShieldCheck, Users } from "lucide-react";

const cards = [
  {
    title: "Teacher Dashboard",
    text: "Scan class mastery, identify weak topics, and queue nudges without waiting for the admin API to land.",
    href: "/dashboard",
    icon: BarChart3,
  },
  {
    title: "Student Detail",
    text: "Inspect one learner's progress, recent conversations, streak, and likely struggle areas.",
    href: "/students/stu_1",
    icon: GraduationCap,
  },
  {
    title: "JWT-ready Surface",
    text: "The API client already expects bearer tokens, so auth can slot in without reworking the UI data layer.",
    href: "/dashboard",
    icon: ShieldCheck,
  },
];

export default function Home() {
  return (
    <main className="min-h-screen overflow-hidden bg-[radial-gradient(circle_at_top_left,_rgba(14,165,233,0.18),transparent_30%),radial-gradient(circle_at_80%_20%,_rgba(249,115,22,0.16),transparent_25%),linear-gradient(180deg,#fffef7_0%,#f6fbff_50%,#ecf7f5_100%)] text-slate-900">
      <div className="mx-auto flex max-w-6xl flex-col gap-10 px-6 py-10 lg:px-10">
        <header className="flex flex-col gap-6 rounded-[32px] border border-white/60 bg-white/70 p-8 shadow-[0_24px_80px_rgba(15,23,42,0.08)] backdrop-blur md:p-10">
          <div className="flex flex-wrap items-center gap-3 text-xs font-semibold uppercase tracking-[0.24em] text-sky-700">
            <span className="rounded-full bg-sky-100 px-3 py-1">P&AI Bot</span>
            <span className="rounded-full bg-amber-100 px-3 py-1">Week 4 / Day 16</span>
          </div>
          <div className="grid gap-6 lg:grid-cols-[1.4fr_0.8fr]">
            <div className="space-y-4">
              <h1 className="max-w-3xl text-4xl font-semibold tracking-tight text-slate-950 md:text-5xl">
                Rebuilt admin panel shell for frontend work while backend admin APIs catch up.
              </h1>
              <p className="max-w-2xl text-base leading-7 text-slate-600 md:text-lg">
                The interface is usable now with fallback data, but it keeps the API contract aligned with the Day 17 backend work.
              </p>
              <div className="flex flex-wrap gap-3">
                <Link
                  href="/dashboard"
                  className="inline-flex items-center justify-center rounded-full bg-slate-950 px-5 py-3 text-sm font-medium text-white transition hover:bg-slate-800"
                >
                  Open dashboard
                </Link>
                <Link
                  href="/students/stu_1"
                  className="inline-flex items-center justify-center rounded-full border border-slate-300 bg-white px-5 py-3 text-sm font-medium text-slate-900 transition hover:border-sky-400 hover:text-sky-700"
                >
                  Open student view
                </Link>
              </div>
            </div>
            <div className="grid gap-3 rounded-[28px] bg-slate-950 p-5 text-slate-50">
              <div className="flex items-center gap-3 rounded-2xl border border-white/10 bg-white/5 p-4">
                <Users className="size-5 text-sky-300" />
                <div>
                  <p className="text-2xl font-semibold">3</p>
                  <p className="text-sm text-slate-300">mock students wired</p>
                </div>
              </div>
              <div className="rounded-2xl border border-white/10 bg-white/5 p-4">
                <p className="text-sm text-slate-300">Current stance</p>
                <p className="mt-2 text-sm leading-6 text-slate-100">
                  Keep the frontend moving with realistic fallback data, but preserve the exact endpoint shapes the Go API needs to implement next.
                </p>
              </div>
            </div>
          </div>
        </header>

        <section className="grid gap-4 md:grid-cols-3">
          {cards.map((card) => {
            const Icon = card.icon;
            return (
              <Link
                key={card.title}
                href={card.href}
                className="group rounded-[28px] border border-slate-200/70 bg-white/80 p-6 shadow-[0_16px_50px_rgba(15,23,42,0.05)] transition hover:-translate-y-1 hover:border-sky-300 hover:shadow-[0_24px_80px_rgba(14,165,233,0.14)]"
              >
                <Icon className="mb-5 size-6 text-sky-600 transition group-hover:text-amber-500" />
                <h2 className="text-xl font-semibold tracking-tight">{card.title}</h2>
                <p className="mt-3 text-sm leading-6 text-slate-600">{card.text}</p>
              </Link>
            );
          })}
        </section>
      </div>
    </main>
  );
}
