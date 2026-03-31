import { render, screen } from "@testing-library/react";
import { AdminLinkCard } from "@/components/admin-link-card";

describe("AdminLinkCard", () => {
  it("renders a linked card title and description", () => {
    render(<AdminLinkCard href="/dashboard" title="Dashboard" description="Open the main workspace." />);

    expect(screen.getByRole("link", { name: /dashboard/i })).toBeInTheDocument();
    expect(screen.getByText("Open the main workspace.")).toBeInTheDocument();
  });
});
