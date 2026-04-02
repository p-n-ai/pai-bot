"use client";

import { QueryClientProvider } from "@tanstack/react-query";
import { Refine } from "@refinedev/core";
import routerProvider from "@refinedev/nextjs-router/app";
import { useState } from "react";
import { getQueryClient } from "@/lib/query-client";
import { getAdminResources } from "@/lib/refine-resources.mjs";

export function RefineProvider({ children }: { children: React.ReactNode }) {
  const [queryClient] = useState(() => getQueryClient());

  return (
    <QueryClientProvider client={queryClient}>
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
    </QueryClientProvider>
  );
}
