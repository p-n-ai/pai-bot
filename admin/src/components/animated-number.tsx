"use client";

import { motion, useMotionValue, useMotionValueEvent, useReducedMotion, useSpring } from "framer-motion";
import { useState } from "react";
import { cn } from "@/lib/utils";

type AnimatedNumberProps = {
  value: number;
  className?: string;
  delay?: number;
  duration?: number;
  formatter?: (value: number) => string;
};

export function AnimatedNumber({
  value,
  className,
  delay = 0,
  duration = 0.72,
  formatter = String,
}: AnimatedNumberProps) {
  const prefersReducedMotion = useReducedMotion();
  const roundedValue = Math.round(value);
  const motionValue = useMotionValue(prefersReducedMotion ? roundedValue : 0);
  const springValue = useSpring(motionValue, prefersReducedMotion ? { duration: 0 } : { damping: 26, stiffness: 180, mass: 0.7 });
  const [displayValue, setDisplayValue] = useState(prefersReducedMotion ? roundedValue : 0);
  useMotionValueEvent(springValue, "change", (latest) => {
    setDisplayValue(Math.round(latest));
  });

  if (motionValue.get() !== roundedValue) {
    if (prefersReducedMotion) {
      motionValue.jump(roundedValue);
    } else {
      motionValue.set(roundedValue);
    }
  }

  return (
    <motion.span
      className={cn("tabular-nums", className)}
      initial={prefersReducedMotion ? false : { opacity: 0.55, filter: "blur(8px)" }}
      animate={prefersReducedMotion ? { opacity: 1 } : { opacity: 1, filter: "blur(0px)" }}
      transition={{ delay, duration, ease: [0.22, 1, 0.36, 1] }}
    >
      {formatter(displayValue)}
    </motion.span>
  );
}
