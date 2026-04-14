import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { vi } from "vitest";
import { LinkedIdentitiesCard } from "./linked-identities-card";

const { useQueryMock, startGoogleLinkMock, errorMock, assignMock } = vi.hoisted(() => ({
  useQueryMock: vi.fn(),
  startGoogleLinkMock: vi.fn(),
  errorMock: vi.fn(),
  assignMock: vi.fn(),
}));

vi.mock("@tanstack/react-query", () => ({
  useQuery: useQueryMock,
}));

vi.mock("sonner", () => ({
  toast: {
    error: errorMock,
  },
}));

vi.mock("@/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api")>();
  return {
    ...actual,
    startGoogleLink: startGoogleLinkMock,
  };
});

describe("LinkedIdentitiesCard", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    Object.defineProperty(window, "location", {
      configurable: true,
      value: {
        assign: assignMock,
      },
    });
  });

  it("returns null when linking is disabled", () => {
    useQueryMock.mockReturnValue({
      data: [],
      isLoading: false,
      isError: false,
    });

    const { container } = render(<LinkedIdentitiesCard enabled={false} nextPath="/dashboard" />);

    expect(container).toBeEmptyDOMElement();
  });

  it("renders a linked Google identity when one exists", () => {
    useQueryMock.mockReturnValue({
      data: [
        {
          provider: "google",
          email: "teacher@example.com",
          last_used_at: "2026-04-06T12:00:00Z",
        },
      ],
      isLoading: false,
      isError: false,
    });

    render(<LinkedIdentitiesCard nextPath="/dashboard" />);

    expect(screen.getByText("Google linked")).toBeInTheDocument();
    expect(screen.getByText("teacher@example.com")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /change link/i })).toBeInTheDocument();
  });

  it("starts Google linking and redirects the browser on success", async () => {
    useQueryMock.mockReturnValue({
      data: [],
      isLoading: false,
      isError: false,
    });
    startGoogleLinkMock.mockResolvedValue("https://accounts.google.com/o/oauth2/auth");

    render(<LinkedIdentitiesCard nextPath="/dashboard/settings" />);

    fireEvent.click(screen.getByRole("button", { name: /link google/i }));

    await waitFor(() => expect(startGoogleLinkMock).toHaveBeenCalledWith("/dashboard/settings"));
    await waitFor(() => expect(assignMock).toHaveBeenCalledWith("https://accounts.google.com/o/oauth2/auth"));
  });

  it("shows a toast when Google linking cannot start", async () => {
    useQueryMock.mockReturnValue({
      data: [],
      isLoading: false,
      isError: false,
    });
    startGoogleLinkMock.mockRejectedValue(new Error("Google OAuth unavailable"));

    render(<LinkedIdentitiesCard nextPath="/dashboard/settings" />);

    fireEvent.click(screen.getByRole("button", { name: /link google/i }));

    await waitFor(() => expect(errorMock).toHaveBeenCalledWith("Google OAuth unavailable"));
    expect(assignMock).not.toHaveBeenCalled();
  });

  it("shows the fetch error state when linked providers cannot be loaded", () => {
    useQueryMock.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
    });

    render(<LinkedIdentitiesCard nextPath="/dashboard" />);

    expect(screen.getByText("We couldn't load linked providers right now.")).toBeInTheDocument();
  });
});
