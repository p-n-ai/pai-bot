import { render, screen } from "@testing-library/react";
import { vi } from "vitest";
import ParentPage from "./page";

const { getServerParentSummaryMock } = vi.hoisted(() => ({
  getServerParentSummaryMock: vi.fn(),
}));

vi.mock("@/lib/server-api", () => ({
  getServerParentSummary: getServerParentSummaryMock,
}));

describe("ParentPage", () => {
  it("shows the load error without falling through to the empty mastery state", async () => {
    getServerParentSummaryMock.mockRejectedValue(new Error("Parent API offline"));

    render(await ParentPage({ params: Promise.resolve({ id: "parent-1" }) }));

    expect(screen.getByText("Parent summary unavailable")).toBeInTheDocument();
    expect(screen.getByText("The parent summary isn't available right now.")).toBeInTheDocument();
    expect(screen.queryByText("No mastery data yet")).not.toBeInTheDocument();
  });
});
