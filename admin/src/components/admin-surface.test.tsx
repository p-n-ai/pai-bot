import { render, screen } from "@testing-library/react";
import { AdminSurface, AdminSurfaceHeader } from "@/components/admin-surface";

describe("AdminSurface", () => {
  it("renders children inside the shared surface wrapper", () => {
    render(
      <AdminSurface>
        <p>Shared admin content</p>
      </AdminSurface>,
    );

    expect(screen.getByText("Shared admin content")).toBeInTheDocument();
  });

  it("renders a consistent header title and description", () => {
    render(<AdminSurfaceHeader title="Mastery heatmap" description="Students by topic." />);

    expect(screen.getByText("Mastery heatmap")).toBeInTheDocument();
    expect(screen.getByText("Students by topic.")).toBeInTheDocument();
  });
});
