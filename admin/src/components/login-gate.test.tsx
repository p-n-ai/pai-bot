import { render, screen } from "@testing-library/react";
import { vi } from "vitest";
import { ThemeProvider } from "@/components/theme-provider";
import { LoginGate } from "@/components/login-gate";

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
    expect(screen.getByTestId("login-gate-light-backdrop")).toBeInTheDocument();
    expect(screen.getByTestId("login-gate-dark-backdrop")).toBeInTheDocument();
  });
});
