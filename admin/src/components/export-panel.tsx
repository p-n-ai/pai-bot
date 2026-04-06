import { IconDatabaseExport, IconFileSpreadsheet, IconMessages } from "@tabler/icons-react";
import { buildAPIPath } from "@/lib/api";

const exportsList = [
  {
    title: "Students CSV",
    description: "Download student roster details with form, channel, tracked topics, and average mastery.",
    href: buildAPIPath("/api/admin/export/students"),
    icon: IconFileSpreadsheet,
  },
  {
    title: "Conversations JSON",
    description: "Download structured conversation transcripts with message metadata for audit and review workflows.",
    href: buildAPIPath("/api/admin/export/conversations"),
    icon: IconMessages,
  },
  {
    title: "Progress CSV",
    description: "Download per-topic mastery records with scheduling fields for spreadsheet or warehouse import.",
    href: buildAPIPath("/api/admin/export/progress"),
    icon: IconDatabaseExport,
  },
];

export function ExportPanel() {
  return (
    <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
      {exportsList.map((item) => (
        <a
          key={item.href}
          href={item.href}
          download
          className="group block rounded-[28px] border border-slate-200/70 bg-white/80 p-6 shadow-[0_16px_50px_rgba(15,23,42,0.05)] transition hover:-translate-y-1 hover:border-sky-300 hover:shadow-[0_24px_80px_rgba(14,165,233,0.14)] dark:border-white/10 dark:bg-slate-950/55 dark:shadow-[0_20px_60px_rgba(2,8,23,0.35)] dark:hover:border-sky-400 dark:hover:shadow-[0_24px_80px_rgba(56,189,248,0.16)]"
        >
          <item.icon className="mb-5 size-6 text-sky-600 transition group-hover:text-amber-500 dark:text-sky-300 dark:group-hover:text-amber-300" />
          <h2 className="text-xl font-semibold tracking-tight dark:text-white">{item.title}</h2>
          <p className="mt-3 text-sm leading-6 text-slate-600 dark:text-slate-300">{item.description}</p>
        </a>
      ))}
    </section>
  );
}
