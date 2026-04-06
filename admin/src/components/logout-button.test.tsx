import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { vi } from "vitest";
import { LogoutButton } from "./logout-button";

const { pushMock, refreshMock, logoutMock, errorMock } = vi.hoisted(() => ({
  pushMock: vi.fn(),
  refreshMock: vi.fn(),
  logoutMock: vi.fn(),
  errorMock: vi.fn(),
}));

vi.mock("next/navigation", () => ({
  useRouter: () => ({
    push: pushMock,
    refresh: refreshMock,
  }),
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
    logout: logoutMock,
  };
});

describe("LogoutButton", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("logs out and routes back to login on success", async () => {
    logoutMock.mockResolvedValue(undefined);

    render(<LogoutButton />);

    fireEvent.click(screen.getByRole("button", { name: /log out/i }));

    await waitFor(() => expect(logoutMock).toHaveBeenCalled());
    await waitFor(() => expect(pushMock).toHaveBeenCalledWith("/login"));
    await waitFor(() => expect(refreshMock).toHaveBeenCalled());
  });

  it("surfaces logout failures and leaves routing untouched", async () => {
    logoutMock.mockRejectedValue(new Error("Session revoke failed"));

    render(<LogoutButton />);

    fireEvent.click(screen.getByRole("button", { name: /log out/i }));

    await waitFor(() => expect(errorMock).toHaveBeenCalledWith("Session revoke failed"));
    expect(pushMock).not.toHaveBeenCalled();
    expect(refreshMock).not.toHaveBeenCalled();
  });
});
