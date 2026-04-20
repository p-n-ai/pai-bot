import type { Metadata } from "next";
import { WidgetChat } from "@/components/widget/widget-chat";
import { normalizeWidgetConfig } from "@/lib/widget-config";

export const dynamic = "force-dynamic";

export const metadata: Metadata = {
  title: "P&AI Tutor Widget",
  description: "Embeddable P&AI Bot chat widget.",
};

type WidgetPageProps = {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
};

export default async function WidgetPage({ searchParams }: WidgetPageProps) {
  const config = normalizeWidgetConfig(await searchParams, process.env.NEXT_PUBLIC_API_URL ?? "");

  return (
    <main className={config.theme === "dark" ? "dark h-screen" : "h-screen"}>
      <WidgetChat {...config} />
    </main>
  );
}
