import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { vi } from "vitest";
import DashboardPage from "./page";

const { useQueryMock, sendStudentNudgeMock, useAppStoreMock } = vi.hoisted(() => ({
  useQueryMock: vi.fn(),
  sendStudentNudgeMock: vi.fn(),
  useAppStoreMock: vi.fn(),
}));

vi.mock("@tanstack/react-query", () => ({
  useQuery: useQueryMock,
}));

vi.mock("framer-motion", () => ({
  motion: {
    header: ({ children, ...props }: React.HTMLAttributes<HTMLElement>) => <header {...props}>{children}</header>,
    section: ({ children, ...props }: React.HTMLAttributes<HTMLElement>) => <section {...props}>{children}</section>,
    div: ({ children, ...props }: React.HTMLAttributes<HTMLDivElement>) => <div {...props}>{children}</div>,
  },
  useReducedMotion: () => true,
}));

vi.mock("@/lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/api")>();
  return {
    ...actual,
    sendStudentNudge: sendStudentNudgeMock,
  };
});

vi.mock("@/stores/app-store", () => ({
  useAppStore: useAppStoreMock,
}));

vi.mock("@/components/animated-number", () => ({
  AnimatedNumber: ({ value, formatter }: { value: number; formatter?: (value: number) => string }) => (
    <span>{formatter ? formatter(value) : value}</span>
  ),
}));

vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  TooltipTrigger: ({ children }: { children: React.ReactNode }) => <>{children}</>,
  TooltipContent: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

type QueryState = {
  data?: unknown;
  isPending: boolean;
  isError: boolean;
  error?: unknown;
};

function setQueryStates(dashboardState: QueryState, previewState: QueryState) {
  useQueryMock.mockReset();
  useQueryMock.mockImplementation(({ queryKey }: { queryKey: unknown[] }) => {
    const state = queryKey[0] === "dashboard-progress-preview" ? previewState : dashboardState;
    return {
      data: state.data,
      isPending: state.isPending,
      isError: state.isError,
      error: state.error ?? null,
    };
  });
}

describe("DashboardPage", () => {
  beforeEach(() => {
    sendStudentNudgeMock.mockReset();
    useAppStoreMock.mockImplementation((selector: (state: { currentUser: { tenant_id: string } }) => unknown) =>
      selector({ currentUser: { tenant_id: "tenant-1" } }),
    );
  });

  it("shows the loading state while the live dashboard query is in flight", () => {
    setQueryStates({ isPending: true, isError: false }, { isPending: false, isError: false });

    render(<DashboardPage />);

    expect(screen.getByText("Preparing the latest class snapshot")).toBeInTheDocument();
    expect(screen.queryByText("Waiting for class data")).not.toBeInTheDocument();
  });

  it("shows the preview fallback error state when live data fails and only preview metadata is available", () => {
    setQueryStates(
      { isPending: false, isError: true, error: new Error("Admin API offline") },
      {
        isPending: false,
        isError: false,
        data: {
          source: "preview",
          issue: "Preview unavailable",
          progress: { students: [], topic_ids: [] },
        },
      },
    );

    render(<DashboardPage />);

    expect(screen.getByText("Live class data is unavailable")).toBeInTheDocument();
    expect(screen.getByText(/Admin API offline Showing preview data until the admin API is reachable again\./i)).toBeInTheDocument();
  });

  it("renders live heatmap data and confirms a successful nudge action", async () => {
    sendStudentNudgeMock.mockResolvedValue(undefined);
    setQueryStates(
      {
        isPending: false,
        isError: false,
        data: {
          source: "live",
          progress: {
            topic_ids: ["linear-equations"],
            students: [
              {
                id: "student-1",
                name: "Alya",
                topics: { "linear-equations": 0.83 },
              },
            ],
          },
        },
      },
      { isPending: false, isError: false },
    );

    render(<DashboardPage />);

    expect(screen.getByText("Alya")).toBeInTheDocument();
    expect(screen.getAllByText("83%")).toHaveLength(2);

    fireEvent.click(screen.getByRole("button", { name: /nudge/i }));

    await waitFor(() => expect(sendStudentNudgeMock).toHaveBeenCalledWith("student-1"));
    await waitFor(() => expect(screen.getByText("Nudge sent to Alya on Telegram.")).toBeInTheDocument());
  });
});
