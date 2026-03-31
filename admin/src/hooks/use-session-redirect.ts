"use client";

import { useEffect, useEffectEvent } from "react";
import type { AppRouterInstance } from "next/dist/shared/lib/app-router-context.shared-runtime";
import type { AuthUser } from "@/lib/api";
import { getSafeNextPath } from "@/lib/rbac.mjs";

type UseSessionRedirectParams = {
  enabled: boolean;
  router: AppRouterInstance;
  user: AuthUser | null;
};

export function useSessionRedirect({ enabled, router, user }: UseSessionRedirectParams) {
  const redirect = useEffectEvent(() => {
    if (!enabled || !user) {
      return;
    }

    router.replace(getSafeNextPath(user, null));
  });

  useEffect(() => {
    redirect();
  }, [redirect]);
}
