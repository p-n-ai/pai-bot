"use client";

import dynamic from "next/dynamic";

const Aurora = dynamic(() => import("@/components/Aurora"), { ssr: false });

export function LightLoginGateBackdrop() {
  return (
    <Aurora
      className="absolute inset-0"
      colorStops={["#43aedf", "#057aff", "#ffffff"]}
      amplitude={0.96}
      blend={0.39}
      speed={0.97}
      style={{ opacity: 0.44 }}
    />
  );
}
