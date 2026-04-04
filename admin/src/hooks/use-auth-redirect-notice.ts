"use client";

import { useEffect, useEffectEvent, type MutableRefObject } from "react";
import type { AppRouterInstance } from "next/dist/shared/lib/app-router-context.shared-runtime";
import type { ReadonlyURLSearchParams } from "next/navigation";
import { toast } from "sonner";
import { getGoogleAuthErrorMessage } from "@/lib/auth-flow-feedback";

type UseAuthRedirectNoticeParams = {
  pathname: string;
  router: AppRouterInstance;
  searchParams: ReadonlyURLSearchParams;
  lastHandledRef: MutableRefObject<string>;
};

export function useAuthRedirectNotice({
  pathname,
  router,
  searchParams,
  lastHandledRef,
}: UseAuthRedirectNoticeParams) {
  const handleRedirectNotice = useEffectEvent(() => {
    const authProvider = searchParams.get("auth_provider");
    const linkedProvider = searchParams.get("identity_linked");
    const authError = searchParams.get("auth_error");

    if (!authProvider && !linkedProvider && !authError) {
      return;
    }

    const signature = searchParams.toString();
    if (lastHandledRef.current === signature) {
      return;
    }
    lastHandledRef.current = signature;

    if (authError) {
      toast.error(getGoogleAuthErrorMessage(authError) || "Google sign-in failed. Please try again.", {
        id: "auth-redirect",
      });
    } else if (linkedProvider === "google") {
      toast.success("Google is now linked to this admin account.", {
        id: "auth-redirect",
      });
    } else if (authProvider === "google") {
      toast.success("Signed in with Google.", {
        id: "auth-redirect",
      });
    }

    const nextParams = new URLSearchParams(searchParams.toString());
    nextParams.delete("auth_provider");
    nextParams.delete("identity_linked");
    nextParams.delete("auth_error");

    const nextURL = nextParams.toString() ? `${pathname}?${nextParams.toString()}` : pathname;
    router.replace(nextURL, { scroll: false });
  });

  useEffect(() => {
    handleRedirectNotice();
  }, [handleRedirectNotice]);
}
