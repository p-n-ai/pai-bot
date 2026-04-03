import { PageHero } from "@/components/page-hero";
import { RetrievalLab } from "@/components/retrieval-lab";

export const dynamic = "force-dynamic";

export default function RetrievalLabPage() {
  return (
    <div className="space-y-6">
      <PageHero
        eyebrow="Retrieval"
        title="BM25 query lab"
        description="Try different retrieval queries, filters, and repeat runs from inside the admin app. This route uses your current session and proxies to the backend retrieval search endpoint."
        surface="plain"
      />
      <RetrievalLab />
    </div>
  );
}
