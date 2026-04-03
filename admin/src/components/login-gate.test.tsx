import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { vi } from "vitest";
import { ThemeProvider } from "@/components/theme-provider";
import { LoginGate } from "@/components/login-gate";
import { ThemeToggle } from "@/components/theme-toggle";

vi.mock("next/navigation", () => ({
  useRouter: () => ({
    push: vi.fn(),
  }),
}));

vi.mock("@/components/login-gate/dark-login-gate-backdrop", () => ({
  DarkLoginGateBackdrop: () => <div>dark backdrop</div>,
}));

vi.mock("@/components/login-gate/light-login-gate-backdrop", () => ({
  LightLoginGateBackdrop: () => <div>light backdrop</div>,
}));

describe("LoginGate", () => {
  it("renders a stable login shell with one form and both themed backdrops", () => {
    const { container } = render(
      <ThemeProvider>
        <LoginGate />
      </ThemeProvider>,
    );

    expect(container.querySelectorAll("form")).toHaveLength(1);
    expect(screen.getByRole("button", { name: "Continue with Google" })).toBeInTheDocument();
    expect(screen.getByTestId("login-gate-light-backdrop")).toBeInTheDocument();
    expect(screen.getByTestId("login-gate-dark-backdrop")).toBeInTheDocument();
  });

  it("flips active backdrop state when the theme changes", async () => {
    render(
      <ThemeProvider>
        <ThemeToggle />
        <LoginGate />
      </ThemeProvider>,
    );

    const lightBackdrop = screen.getByTestId("login-gate-light-backdrop");
    const darkBackdrop = screen.getByTestId("login-gate-dark-backdrop");
    const toggle = await screen.findByRole("button", { name: "Switch to dark theme" });

    expect(lightBackdrop).toHaveAttribute("aria-hidden", "false");
    expect(darkBackdrop).toHaveAttribute("aria-hidden", "true");

    fireEvent.click(toggle);

    await waitFor(() => {
      expect(screen.getByTestId("login-gate-light-backdrop")).toHaveAttribute("aria-hidden", "true");
      expect(screen.getByTestId("login-gate-dark-backdrop")).toHaveAttribute("aria-hidden", "false");
    });
  });

  it("surfaces Google auth callback errors in the form panel", () => {
    render(
      <ThemeProvider>
        <LoginGate authError="link_required" />
      </ThemeProvider>,
    );

    expect(screen.getByText("We couldn't sign you in yet.")).toBeInTheDocument();
    expect(screen.getByText(/sign in with email once, then link Google/i)).toBeInTheDocument();
  });
});
