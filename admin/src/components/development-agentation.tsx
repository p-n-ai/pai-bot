"use client";

import { usePathname } from "next/navigation";
import { Agentation } from "agentation";

const agentationEndpoint = process.env.NEXT_PUBLIC_AGENTATION_ENDPOINT;
const showAgentation = process.env.NODE_ENV === "development" && Boolean(agentationEndpoint);

export function DevelopmentAgentation() {
  usePathname();

  if (!showAgentation) {
    return null;
  }

  return <Agentation endpoint={agentationEndpoint} />;
}
