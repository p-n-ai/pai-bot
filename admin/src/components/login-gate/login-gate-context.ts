import { createContext } from "react";
export type LoginGateContextValue = {
  email: string;
  password: string;
  error: string;
  isPending: boolean;
  isGooglePending: boolean;
  setEmail: (value: string) => void;
  setPassword: (value: string) => void;
  submit: (event: React.FormEvent<HTMLFormElement>) => void;
  startGoogleLogin: () => void;
};

export const LoginGateContext = createContext<LoginGateContextValue | null>(null);
