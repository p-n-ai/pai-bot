type JoinPageProps = {
  params: Promise<{ slug: string }>;
};

export default async function JoinPage({ params }: JoinPageProps) {
  const { slug } = await params;

  return (
    <main className="min-h-screen bg-background px-6 py-16 text-foreground">
      <div className="mx-auto max-w-2xl rounded-2xl border bg-card p-8 shadow-xs">
        <p className="text-xs font-medium uppercase tracking-[0.12em] text-muted-foreground">Class join link</p>
        <h1 className="mt-3 text-3xl font-semibold tracking-tight text-foreground">{slug}</h1>
        <p className="mt-4 max-w-xl text-sm leading-6 text-muted-foreground">
          This onboarding slice now creates a stable join link for the first class. Student join and teacher invite completion still
          land in the next slice.
        </p>
      </div>
    </main>
  );
}
