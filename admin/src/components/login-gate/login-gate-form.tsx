"use client";

import { useState } from "react";
import { IconAlertCircle, IconEye, IconEyeOff } from "@tabler/icons-react";
import { LoginGateGoogleButton } from "@/components/login-gate/login-gate-google-button";
import { useLoginGate } from "@/components/login-gate/use-login-gate";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Field, FieldGroup, FieldLabel, FieldSeparator } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from "@/components/ui/input-group";
import { Spinner } from "@/components/ui/spinner";

export function LoginGateForm() {
  const { email, password, error, isPending, isGooglePending, setEmail, setPassword, submit } = useLoginGate();
  const [showPassword, setShowPassword] = useState(false);
  const showGoogleLogin = process.env.NEXT_PUBLIC_PAI_AUTH_GOOGLE_LOGIN_ENABLED === "true";
  const isBusy = isPending || isGooglePending;

  const errorMessage =
    error === "Failed to fetch"
      ? "We couldn't reach the sign-in service. Check your connection and try again."
      : error;

  return (
    <form
      id="sign-in-form"
      className="space-y-6 transition-[opacity,transform] duration-200 ease-out"
      aria-busy={isBusy}
      aria-describedby={error ? "sign-in-error" : undefined}
      onSubmit={submit}
    >
      {showGoogleLogin ? (
        <FieldGroup className="gap-5">
          <LoginGateGoogleButton />
          <FieldSeparator>or use email</FieldSeparator>
        </FieldGroup>
      ) : null}

      <FieldGroup className="gap-5">
        <Field data-disabled={isBusy}>
          <FieldLabel htmlFor="email">Email</FieldLabel>
          <Input
            id="email"
            name="email"
            type="email"
            value={email}
            onChange={(event) => setEmail(event.target.value)}
            placeholder="teacher@school.edu"
            autoComplete="username"
            spellCheck={false}
            className="h-12 rounded-2xl border-border bg-background text-foreground placeholder:text-muted-foreground transition-all duration-150 ease-out"
            disabled={isBusy}
            required
          />
        </Field>

        <Field data-disabled={isBusy}>
          <FieldLabel htmlFor="password">Password</FieldLabel>
          <InputGroup className="h-12 rounded-2xl border border-border bg-background">
            <InputGroupInput
              id="password"
              name="password"
              type={showPassword ? "text" : "password"}
              value={password}
              onChange={(event) => setPassword(event.target.value)}
              autoComplete="current-password"
              className="text-foreground placeholder:text-muted-foreground"
              disabled={isBusy}
              required
            />
            <InputGroupAddon align="inline-end">
              <InputGroupButton
                size="icon-sm"
                className="size-10"
                aria-label={showPassword ? "Hide password" : "Show password"}
                aria-pressed={showPassword}
                onClick={() => setShowPassword((isVisible) => !isVisible)}
                disabled={isBusy}
              >
                {showPassword ? <IconEyeOff /> : <IconEye />}
              </InputGroupButton>
            </InputGroupAddon>
          </InputGroup>
        </Field>
      </FieldGroup>

      {error ? (
        <Alert
          id="sign-in-error"
          variant="destructive"
          aria-live="assertive"
          className="animate-in fade-in-0 slide-in-from-top-1 gap-y-1 rounded-2xl px-4 py-3 shadow-none duration-200"
        >
          <IconAlertCircle />
          <AlertTitle>Sign-in failed.</AlertTitle>
          <AlertDescription className="leading-6">
            {errorMessage}
          </AlertDescription>
        </Alert>
      ) : null}

      <Button
        type="submit"
        size="lg"
        className="h-12 w-full rounded-full transition-all duration-150 ease-out active:translate-y-px"
        disabled={isBusy}
      >
        {isPending ? (
          <>
            <Spinner aria-hidden="true" />
            Sign in
          </>
        ) : (
          "Sign in"
        )}
      </Button>
    </form>
  );
}
