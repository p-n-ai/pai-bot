import { render, screen } from "@testing-library/react";
import { useHydrated } from "@/hooks/use-hydrated";

function HydrationStateProbe() {
  const isHydrated = useHydrated();

  return <span>{isHydrated ? "hydrated" : "pending"}</span>;
}

describe("useHydrated", () => {
  it("reports hydration complete after mount", async () => {
    render(<HydrationStateProbe />);

    expect(await screen.findByText("hydrated")).toBeInTheDocument();
  });
});
