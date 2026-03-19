import { render, screen } from "@testing-library/react";
import { AdminHighlightPanel } from "@/components/admin-highlight-panel";

describe("AdminHighlightPanel", () => {
  it("renders children inside the shared dark highlight wrapper", () => {
    render(
      <AdminHighlightPanel>
        <p>Highlighted content</p>
      </AdminHighlightPanel>,
    );

    expect(screen.getByText("Highlighted content")).toBeInTheDocument();
  });
});
