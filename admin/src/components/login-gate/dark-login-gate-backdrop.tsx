"use client";

import dynamic from "next/dynamic";

const Aurora = dynamic(() => import("@/components/Aurora"), { ssr: false });

export function DarkLoginGateBackdrop() {
  return (
    <Aurora
      className="absolute inset-0 opacity-100"
      colorStops={["#fff7ed", "#a96b48", "#171310"]}
      amplitude={1.05}
      blend={0.4}
      speed={0.9}
    />
  );
}
