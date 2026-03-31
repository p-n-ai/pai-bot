import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { ThemeProvider } from "@/components/theme-provider";
import { ThemeToggle } from "@/components/theme-toggle";
import { THEME_STORAGE_KEY } from "@/lib/theme.mjs";

describe("ThemeToggle", () => {
  beforeEach(() => {
    window.localStorage.clear();
    document.documentElement.className = "";
    document.documentElement.style.colorScheme = "";
  });

  it("renders an accessible toggle button immediately", () => {
    render(
      <ThemeProvider>
        <ThemeToggle />
      </ThemeProvider>,
    );

    expect(screen.getByRole("button", { name: "Switch to dark theme" })).toBeInTheDocument();
  });

  it("renders the current theme action once mounted", async () => {
    render(
      <ThemeProvider>
        <ThemeToggle />
      </ThemeProvider>,
    );

    expect(await screen.findByRole("button", { name: "Switch to dark theme" })).toBeInTheDocument();
  });

  it("renders the stored dark theme action after mount", async () => {
    window.localStorage.setItem(THEME_STORAGE_KEY, "dark");

    render(
      <ThemeProvider>
        <ThemeToggle />
      </ThemeProvider>,
    );

    expect(await screen.findByRole("button", { name: "Switch to light theme" })).toBeInTheDocument();
  });

  it("toggles theme state and persists it to localStorage", async () => {
    render(
      <ThemeProvider>
        <ThemeToggle />
      </ThemeProvider>,
    );

    const button = await screen.findByRole("button", { name: "Switch to dark theme" });
    fireEvent.click(button);

    await waitFor(() => {
      expect(window.localStorage.getItem(THEME_STORAGE_KEY)).toBe("dark");
      expect(document.documentElement).toHaveClass("dark");
      expect(document.documentElement.style.colorScheme).toBe("dark");
      expect(screen.getByRole("button", { name: "Switch to light theme" })).toBeInTheDocument();
    });
  });
});
