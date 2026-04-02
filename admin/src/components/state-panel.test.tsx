import { render, screen } from "@testing-library/react";
import { StatePanel } from "@/components/state-panel";

describe("StatePanel", () => {
  it("renders title and description", () => {
    render(<StatePanel tone="empty" title="No data yet" description="Waiting for the first sync." />);

    expect(screen.getByText("No data yet")).toBeInTheDocument();
    expect(screen.getByText("Waiting for the first sync.")).toBeInTheDocument();
  });

  it("applies tone-specific styling", () => {
    render(<StatePanel tone="error" title="Error" description="Could not load." />);

    expect(screen.getByRole("alert")).toBeInTheDocument();
  });
});
