"use client";

import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useRef } from "react";
import { useAuthRedirectNotice } from "@/hooks/use-auth-redirect-notice";

export function AuthRedirectNotice() {
  const pathname = usePathname();
  const router = useRouter();
  const searchParams = useSearchParams();
  const lastHandled = useRef("");

  useAuthRedirectNotice({
    pathname,
    router,
    searchParams,
    lastHandledRef: lastHandled,
  });

  return null;
}
