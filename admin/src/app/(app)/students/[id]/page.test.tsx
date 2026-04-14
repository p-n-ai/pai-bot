import { render, screen } from "@testing-library/react";
import { vi } from "vitest";
import StudentPage from "./page";

const { useAsyncResourceMock } = vi.hoisted(() => ({
  useAsyncResourceMock: vi.fn(),
}));

vi.mock("next/navigation", () => ({
  useParams: () => ({ id: "student-1" }),
}));

vi.mock("@/hooks/use-async-resource", () => ({
  useAsyncResource: useAsyncResourceMock,
}));

describe("StudentPage", () => {
  it("shows a loading state instead of empty-data placeholders while the detail request is in flight", () => {
    useAsyncResourceMock.mockReturnValue({
      data: null,
      loading: true,
      error: "",
      setData: vi.fn(),
      setError: vi.fn(),
    });

    render(<StudentPage />);

    expect(screen.getByText("Loading student detail")).toBeInTheDocument();
    expect(screen.queryByText("No mastery radar yet")).not.toBeInTheDocument();
    expect(screen.queryByText("No active struggle areas")).not.toBeInTheDocument();
    expect(screen.queryByText("No tutoring messages yet")).not.toBeInTheDocument();
  });

  it("shows an error state instead of empty-data placeholders when the detail request fails", () => {
    useAsyncResourceMock.mockReturnValue({
      data: null,
      loading: false,
      error: "Student API offline",
      setData: vi.fn(),
      setError: vi.fn(),
    });

    render(<StudentPage />);

    expect(screen.getByText("Student detail unavailable")).toBeInTheDocument();
    expect(screen.getByText("Student API offline")).toBeInTheDocument();
    expect(screen.queryByText("No mastery radar yet")).not.toBeInTheDocument();
    expect(screen.queryByText("No topic progress yet")).not.toBeInTheDocument();
    expect(screen.queryByText("No tutoring messages yet")).not.toBeInTheDocument();
  });
});
