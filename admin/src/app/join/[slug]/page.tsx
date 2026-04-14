import { notFound } from "next/navigation";
import { getServerJoinClass } from "@/lib/server-api";

type JoinPageProps = {
  params: Promise<{ slug: string }>;
};

export default async function JoinPage({ params }: JoinPageProps) {
  const { slug } = await params;
  let joinClass: Awaited<ReturnType<typeof getServerJoinClass>>;

  try {
    joinClass = await getServerJoinClass(slug);
  } catch {
    notFound();
  }

  return (
    <main className="min-h-screen bg-background px-6 py-16 text-foreground">
      <div className="mx-auto max-w-2xl rounded-2xl border bg-card p-8 shadow-xs">
        <p className="text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">Class join link</p>
        <h1 className="mt-3 text-3xl font-semibold tracking-tight text-foreground">{joinClass.class_name}</h1>
        <p className="mt-4 max-w-xl text-sm leading-6 text-muted-foreground">
          {joinClass.school_name} is ready to use this join route for <span className="font-medium text-foreground">{joinClass.class_slug}</span>.
          Student enrollment and invite completion still land in the next slice.
        </p>
        <dl className="mt-6 space-y-3 text-sm">
          <div className="flex items-center justify-between gap-3">
            <dt className="text-muted-foreground">School</dt>
            <dd>{joinClass.school_name}</dd>
          </div>
          <div className="flex items-center justify-between gap-3">
            <dt className="text-muted-foreground">Curriculum</dt>
            <dd>{joinClass.curriculum_label}</dd>
          </div>
        </dl>
      </div>
    </main>
  );
}
