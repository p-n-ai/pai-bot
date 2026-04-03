import { createContext } from "react";
import type { TenantChoice } from "@/lib/api";

export type LoginGateContextValue = {
  email: string;
  password: string;
  tenantID: string;
  tenantChoices: TenantChoice[];
  error: string;
  isPending: boolean;
  isGooglePending: boolean;
  setEmail: (value: string) => void;
  setPassword: (value: string) => void;
  setTenantID: (value: string) => void;
  submit: (event: React.FormEvent<HTMLFormElement>) => void;
  startGoogleLogin: () => void;
};

export const LoginGateContext = createContext<LoginGateContextValue | null>(null);
