import { IconAlertTriangle, IconFileSearch, IconLoader2 } from "@tabler/icons-react";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty";
import { cn } from "@/lib/utils";

const toneClasses = {
  loading: {
    icon: IconLoader2,
    iconClassName: "animate-spin text-primary",
  },
  empty: {
    icon: IconFileSearch,
    iconClassName: "text-muted-foreground",
  },
  error: {
    icon: IconAlertTriangle,
    iconClassName: "text-destructive",
  },
} as const;

export function StatePanel({
  tone,
  title,
  description,
  className,
}: {
  tone: keyof typeof toneClasses;
  title: string;
  description: string;
  className?: string;
}) {
  const config = toneClasses[tone];
  const Icon = config.icon;

  if (tone === "error") {
    return (
      <Alert variant="destructive" className={cn("rounded-xl", className)}>
        <Icon className={config.iconClassName} />
        <AlertTitle>{title}</AlertTitle>
        <AlertDescription>{description}</AlertDescription>
      </Alert>
    );
  }

  return (
    <Empty className={cn("min-h-0 items-start rounded-xl border bg-card p-6 text-left", className)}>
      <EmptyHeader className="items-start text-left">
        <EmptyMedia variant="icon">
          <Icon className={config.iconClassName} />
        </EmptyMedia>
        <EmptyTitle>{title}</EmptyTitle>
        <EmptyDescription>{description}</EmptyDescription>
      </EmptyHeader>
    </Empty>
  );
}
