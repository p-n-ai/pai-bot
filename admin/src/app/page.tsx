import Link from "next/link";
import { cookies } from "next/headers";
import { BarChart3, Coins, Users } from "lucide-react";
import { AdminHighlightPanel } from "@/components/admin-highlight-panel";
import { AdminLinkCard } from "@/components/admin-link-card";
import { AdminSurface } from "@/components/admin-surface";
import { parseCookieJSON, USER_COOKIE } from "@/lib/auth-session";
import type { AuthUser } from "@/lib/api";
import { hasAdminUIAccess } from "@/lib/rbac.mjs";
import { getServerClassProgress } from "@/lib/server-api";

const elevatedCards = [
  {
    title: "Teacher Dashboard",
    text: "Scan class mastery, identify weak topics, and move quickly on learners who need attention.",
    href: "/dashboard",
    icon: BarChart3,
  },
  {
    title: "AI Usage",
    text: "Inspect token volume by provider and model inside the admin panel.",
    href: "/dashboard/ai-usage",
    icon: Coins,
  },
];

export const dynamic = "force-dynamic";

export default async function Home() {
  const cookieStore = await cookies();
  const currentUser = parseCookieJSON<AuthUser>(cookieStore.get(USER_COOKIE)?.value);
  const hasWorkspaceAccess = hasAdminUIAccess(currentUser);
  const isParentView = hasWorkspaceAccess && currentUser?.role === "parent";
  const cards = isParentView
    ? [
        {
          title: "Child Summary",
          text: "Open the weekly parent summary for mastery updates, activity snapshots, and encouragement guidance.",
          href: `/parents/${currentUser.user_id}`,
          icon: Users,
        },
      ]
    : elevatedCards;
  let studentCount = "Not available yet";

  try {
    const data = await getServerClassProgress("all-students");
    studentCount = String(data.students.length);
  } catch {
    studentCount = "Not available yet";
  }

  return (
    <div className="flex flex-col gap-8">
      <AdminSurface className="rounded-[32px] border-white/60 bg-white/70 md:p-2 dark:bg-slate-950/55" contentClassName="p-8 md:p-10">
        <section className="flex flex-col gap-6">
          <div className="flex flex-wrap items-center gap-3 text-xs font-semibold uppercase tracking-[0.24em] text-sky-700 dark:text-sky-300">
            <span className="rounded-full bg-sky-100 px-3 py-1 dark:bg-sky-400/15 dark:text-sky-100">P&AI Bot</span>
            <span className="rounded-full bg-amber-100 px-3 py-1 dark:bg-amber-300/15 dark:text-amber-100">Admin Panel</span>
          </div>
          <div className="grid gap-6 lg:grid-cols-[1.4fr_0.8fr]">
            <div className="space-y-4">
               <h1 className="max-w-3xl text-4xl font-semibold tracking-tight text-slate-950 md:text-5xl dark:text-white">
                 {isParentView
                   ? "See your child's weekly learning momentum and where encouragement will help most."
                   : "Track class momentum, student mastery, and recent tutoring activity in one place."}
               </h1>
               <p className="max-w-2xl text-base leading-7 text-slate-600 md:text-lg dark:text-slate-300">
                 {isParentView
                   ? "Use the parent summary to review weekly activity, topic mastery, and the next conversation to have at home."
                   : "Use the dashboard to scan class-level performance, then drill into a learner profile to review progress and conversations."}
               </p>
               <div className="flex flex-wrap gap-3">
                 <Link
                   href={isParentView ? `/parents/${currentUser.user_id}` : "/dashboard"}
                   className="inline-flex items-center justify-center rounded-full bg-slate-950 px-5 py-3 text-sm font-medium text-white transition hover:bg-slate-800 dark:bg-sky-400 dark:text-slate-950 dark:hover:bg-sky-300"
                 >
                   {isParentView ? "Open child summary" : "Open dashboard"}
                 </Link>
                {!hasWorkspaceAccess ? (
                  <Link
                    href="/login"
                    className="inline-flex items-center justify-center rounded-full border border-slate-300/80 bg-white/70 px-5 py-3 text-sm font-medium text-slate-900 transition hover:border-sky-300 hover:text-sky-700 dark:border-white/15 dark:bg-slate-950/40 dark:text-slate-100 dark:hover:border-sky-400 dark:hover:text-sky-200"
                  >
                    Sign in
                  </Link>
                ) : null}
              </div>
            </div>
            <AdminHighlightPanel className="rounded-[28px] p-5 text-slate-50">
              <div className="flex items-center gap-3 rounded-2xl border border-white/10 bg-white/5 p-4 dark:bg-white/[0.03]">
                <Users className="size-5 text-sky-300 dark:text-sky-200" />
                  <div>
                 <p className="text-2xl font-semibold">{studentCount}</p>
                  <p className="text-sm text-slate-300">
                    {isParentView ? "student records currently returned by the admin API" : "students currently returned by the admin API"}
                  </p>
                </div>
              </div>
              <div className="rounded-2xl border border-white/10 bg-white/5 p-4 dark:bg-white/[0.03]">
                <p className="text-sm text-slate-300">What you can inspect</p>
                <p className="mt-2 text-sm leading-6 text-slate-100">
                  {isParentView
                    ? "Parent-facing weekly summaries, per-topic mastery, and encouragement guidance from the Go backend."
                    : "Teacher-facing summaries, per-topic mastery, and recent tutoring exchanges from the Go backend."}
                </p>
              </div>
            </AdminHighlightPanel>
          </div>
        </section>
      </AdminSurface>

      <section className="grid gap-4 md:grid-cols-2">
        {cards.map((card) => {
          return (
            <AdminLinkCard
              key={card.title}
              href={card.href}
              title={card.title}
              description={card.text}
              icon={card.icon}
            />
          );
        })}
      </section>
    </div>
  );
}
