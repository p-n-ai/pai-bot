"use client";

import dynamic from "next/dynamic";

const Aurora = dynamic(() => import("@/components/Aurora"), { ssr: false });

export function LightLoginGateBackdrop() {
  return (
    <Aurora
      className="absolute inset-0"
      colorStops={["#fffdf9", "#d98b5f", "#a96b48"]}
      amplitude={0.9}
      blend={0.36}
      speed={0.97}
      style={{ opacity: 0.42 }}
    />
  );
}
