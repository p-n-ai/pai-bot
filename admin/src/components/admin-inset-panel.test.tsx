import { render, screen } from "@testing-library/react";
import { AdminInsetPanel } from "@/components/admin-inset-panel";

describe("AdminInsetPanel", () => {
  it("renders heading and content inside the shared inset wrapper", () => {
    render(
      <AdminInsetPanel title="Focus">
        <p>Review learner progress.</p>
      </AdminInsetPanel>,
    );

    expect(screen.getByText("Focus")).toBeInTheDocument();
    expect(screen.getByText("Review learner progress.")).toBeInTheDocument();
  });
});
