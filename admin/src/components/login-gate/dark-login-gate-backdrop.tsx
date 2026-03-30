"use client";

import dynamic from "next/dynamic";

const Aurora = dynamic(() => import("@/components/Aurora"), { ssr: false });

export function DarkLoginGateBackdrop() {
  return (
    <Aurora
      className="absolute inset-0 opacity-100"
      colorStops={["#0ea5e9", "#2563eb", "#06101d"]}
      amplitude={1.15}
      blend={0.42}
      speed={0.9}
    />
  );
}
