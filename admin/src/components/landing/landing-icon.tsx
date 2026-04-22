"use client";

import {
  ArrowRight,
  BookOpenText,
  ChatCircleText,
  ClipboardText,
  PaperPlaneTilt,
  Robot,
  TrendUp,
  UsersThree,
} from "@phosphor-icons/react";
import { cn } from "@/lib/utils";

const icons = {
  arrowRight: ArrowRight,
  bookOpenText: BookOpenText,
  chatCircleText: ChatCircleText,
  clipboardText: ClipboardText,
  paperPlaneTilt: PaperPlaneTilt,
  robot: Robot,
  trendUp: TrendUp,
  usersThree: UsersThree,
};

export type LandingIconName = keyof typeof icons;

export function LandingIcon({
  name,
  className,
}: {
  name: LandingIconName;
  className?: string;
}) {
  const Icon = icons[name];

  return <Icon className={cn("size-5", className)} weight="duotone" />;
}
