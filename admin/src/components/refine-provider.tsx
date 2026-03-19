"use client";

import { Refine } from "@refinedev/core";
import routerProvider from "@refinedev/nextjs-router/app";
import { getAdminResources } from "@/lib/refine-resources.mjs";

export function RefineProvider({ children }: { children: React.ReactNode }) {
  return (
    <Refine
      routerProvider={routerProvider}
      resources={getAdminResources()}
      options={{
        syncWithLocation: true,
        disableTelemetry: true,
      }}
    >
      {children}
    </Refine>
  );
}
