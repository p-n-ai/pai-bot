import { render, screen } from "@testing-library/react";
import { ExportPanel } from "@/components/export-panel";

describe("ExportPanel", () => {
  it("renders the export actions with download links", () => {
    render(<ExportPanel />);

    expect(screen.getByRole("link", { name: /Students CSV/i })).toHaveAttribute("href", "/api/admin/export/students");
    expect(screen.getByRole("link", { name: /Conversations JSON/i })).toHaveAttribute("href", "/api/admin/export/conversations");
    expect(screen.getByRole("link", { name: /Progress CSV/i })).toHaveAttribute("href", "/api/admin/export/progress");
  });
});
